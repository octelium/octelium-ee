import "@xterm/xterm/css/xterm.css";
import { ListenTerminalResponse, Workspace } from "@/apis/cordiumv1/cordiumv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { getClientCordiumWorkspace } from "@/utils/client";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import type { RpcError } from "@protobuf-ts/runtime-rpc";
import type { ConnectInput, ConnectPlan, Facts } from "./plan";
import { toolLabels } from "./plan";
import {
  buildSetupScript,
  parseSetupLine,
  setupCommand,
  setupSteps,
  type SetupEvent,
} from "./setup";
import { terminalTheme } from "./theme";

export type StepStatus = "pending" | "running" | "done" | "failed" | "skipped";

export interface SessionStep {
  id: string;
  label: string;
  status: StepStatus;
  detail?: string;
}

export type SessionPhase =
  "starting" | "preparing" | "ready" | "ended" | "error";

export interface SessionState {
  phase: SessionPhase;
  steps: SessionStep[];
  log: string[];
  facts: Facts;
  terminalId?: string;
  error?: string;
  warning?: string;
  issue?: string;
  selection: boolean;
  kept: boolean;
  cols: number;
  rows: number;
}

const CHUNK_SIZE = 4096;
const LOG_LINES = 200;
const SETUP_TIMEOUT_MS = 6 * 60_000;
const ISSUE_WINDOW_MS = 20_000;
const OUTPUT_CHARS = 6000;

const ESC = String.fromCharCode(27);
const BEL = String.fromCharCode(7);
const ansiPattern = new RegExp(
  [
    `${ESC}\\[[0-?]*[ -/]*[@-~]`,
    `${ESC}\\][^${BEL}${ESC}]*(?:${BEL}|${ESC}\\\\)`,
    `${ESC}[PX^_][^${ESC}]*${ESC}\\\\`,
    `${ESC}[@-Z\\\\-_]`,
    `[${String.fromCharCode(0)}-${String.fromCharCode(8)}${String.fromCharCode(11)}${String.fromCharCode(12)}${String.fromCharCode(14)}-${String.fromCharCode(31)}${String.fromCharCode(127)}]`,
  ].join("|"),
  "g",
);

export const stripAnsi = (value: string) => value.replace(ansiPattern, "");

const issuePatterns = [
  /permission denied/i,
  /connection refused/i,
  /connection timed out/i,
  /connection reset by peer/i,
  /could not resolve hostname/i,
  /name or service not known/i,
  /no route to host/i,
  /host key verification failed/i,
  /\bFATAL:/,
  /psql: error/i,
  /^ERROR \d+ \(/,
  /access denied/i,
  /unable to connect to the server/i,
  /could(?: not|n't) connect to (?:the )?server/i,
  /\bforbidden\b/i,
  /command not found/i,
  /NOAUTH/,
];

const sleep = (ms: number, signal: AbortSignal) =>
  new Promise<void>((resolve, reject) => {
    if (signal.aborted) {
      reject(new DOMException("Aborted", "AbortError"));
      return;
    }
    const timer = window.setTimeout(resolve, ms);
    signal.addEventListener(
      "abort",
      () => {
        window.clearTimeout(timer);
        reject(new DOMException("Aborted", "AbortError"));
      },
      { once: true },
    );
  });

const isAbort = (err: unknown) =>
  (err as Error | undefined)?.name === "AbortError" ||
  (err as RpcError | undefined)?.code === "CANCELLED";

const errorMessage = (err: unknown) =>
  (err as RpcError | Error | undefined)?.message || String(err);

const randomNonce = () =>
  Array.from(crypto.getRandomValues(new Uint8Array(6)), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("");

interface Stream {
  id: string;
  output: boolean;
  lastOutputAt: number;
}

export const planSteps = (
  plan: ConnectPlan,
  terminal: StepStatus = "pending",
): SessionStep[] => [
  { id: "terminal", label: "Open a terminal", status: terminal },
  ...setupSteps(plan).map((step) => ({
    ...step,
    status: "pending" as const,
  })),
  {
    id: "connect",
    label: plan.tool
      ? plan.commands({}).some((input) => input.run)
        ? `Connect with ${plan.label}`
        : `Type the ${plan.label} command`
      : "Open the shell",
    status: "pending",
  },
];

export class ConnectSession {
  readonly plan: ConnectPlan;
  readonly workspace: Workspace;
  private readonly domain: string;
  private readonly element: HTMLDivElement;
  private readonly encoder = new TextEncoder();
  private readonly listeners = new Set<() => void>();
  private term?: Terminal;
  private fit?: FitAddon;
  private container?: HTMLElement;
  private observer?: ResizeObserver;
  private resizeTimer = 0;
  private sentSize = "";
  private controller = new AbortController();
  private generation = 0;
  private started = false;
  private disposed = false;
  private main?: Stream;
  private setup?: Stream;
  private skipSetup?: () => void;
  private queue: Uint8Array[] = [];
  private flushing = false;
  private decoder = new TextDecoder();
  private watchUntil = 0;
  private watchLine = "";
  private state: SessionState;

  constructor(opts: {
    plan: ConnectPlan;
    workspace: Workspace;
    domain: string;
  }) {
    this.plan = opts.plan;
    this.workspace = opts.workspace;
    this.domain = opts.domain;
    this.element = document.createElement("div");
    this.element.className = "h-full w-full";
    this.state = this.initialState();
  }

  private initialState(): SessionState {
    return {
      phase: "starting",
      steps: planSteps(this.plan, "running"),
      log: [],
      facts: {},
      selection: false,
      kept: false,
      cols: 0,
      rows: 0,
    };
  }

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  };

  getSnapshot = () => this.state;

  private update(patch: Partial<SessionState>) {
    this.state = { ...this.state, ...patch };
    for (const listener of this.listeners) listener();
  }

  private setStep(id: string, patch: Partial<SessionStep>, label?: string) {
    const steps = this.state.steps.some((step) => step.id === id)
      ? this.state.steps.map((step) =>
          step.id === id ? { ...step, ...patch } : step,
        )
      : (() => {
          const ret = [...this.state.steps];
          const at = Math.max(
            0,
            ret.findIndex((step) => step.status === "running"),
          );
          ret.splice(at, 0, {
            id,
            label: label ?? id,
            status: "pending",
            ...patch,
          });
          return ret;
        })();
    this.update({ steps });
  }

  private get client() {
    return getClientCordiumWorkspace(this.workspace.status?.regionRef);
  }

  private get workspaceRef() {
    return ObjectReference.create({
      uid: this.workspace.metadata?.uid,
      name: this.workspace.metadata?.name,
    });
  }

  attach(container: HTMLElement) {
    if (this.disposed) return;
    this.container = container;
    container.appendChild(this.element);
    if (!this.term) {
      this.term = new Terminal({
        convertEol: false,
        cursorBlink: true,
        disableStdin: true,
        fontFamily:
          "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace",
        fontSize: 13,
        fontWeight: 450,
        fontWeightBold: 700,
        lineHeight: 1.25,
        scrollback: 5000,
        allowProposedApi: true,
        theme: terminalTheme,
      });
      this.fit = new FitAddon();
      this.term.loadAddon(this.fit);
      this.term.open(this.element);
      this.term.onData((data) => this.send(data));
      this.term.onSelectionChange(() => {
        const selection = !!this.term?.hasSelection();
        if (selection !== this.state.selection) this.update({ selection });
      });
    }
    this.observer?.disconnect();
    this.observer = new ResizeObserver(() => this.scheduleFit());
    this.observer.observe(container);
    this.fitNow();
    this.term.refresh(0, this.term.rows - 1);
  }

  detach(container: HTMLElement) {
    if (this.container !== container) return;
    this.observer?.disconnect();
    this.observer = undefined;
    window.clearTimeout(this.resizeTimer);
    this.element.remove();
    this.container = undefined;
  }

  private scheduleFit() {
    window.clearTimeout(this.resizeTimer);
    this.resizeTimer = window.setTimeout(() => this.fitNow(), 60);
  }

  private fitNow() {
    const container = this.container;
    if (!this.term || !this.fit || !container) return;
    if (container.clientWidth === 0 || container.clientHeight === 0) return;
    this.fit.fit();
    const { cols, rows } = this.term;
    if (cols !== this.state.cols || rows !== this.state.rows) {
      this.update({ cols, rows });
    }
    this.syncSize();
  }

  private syncSize() {
    const id = this.main?.id;
    if (!id || !this.term) return;
    const { cols, rows } = this.term;
    const key = `${id}:${cols}x${rows}`;
    if (key === this.sentSize) return;
    this.sentSize = key;
    this.client.setTerminalWindowSize({ id, cols, rows }).then(
      () => undefined,
      () => undefined,
    );
  }

  start() {
    if (this.started || this.disposed) return;
    this.started = true;
    window.addEventListener("pagehide", this.onPageHide);
    void this.connect(this.generation, this.controller.signal);
  }

  restart() {
    if (this.disposed) return;
    this.stop(!this.state.kept);
    this.generation += 1;
    this.controller = new AbortController();
    this.term?.reset();
    if (this.term) this.term.options.disableStdin = true;
    this.watchUntil = 0;
    this.decoder = new TextDecoder();
    this.state = {
      ...this.initialState(),
      cols: this.state.cols,
      rows: this.state.rows,
    };
    this.update({});
    this.started = true;
    void this.connect(this.generation, this.controller.signal);
  }

  skip() {
    this.skipSetup?.();
  }

  keep() {
    if (!this.state.kept) this.update({ kept: true });
  }

  focus() {
    this.term?.focus();
  }

  type(text: string, run = false) {
    if (this.state.phase !== "ready") return;
    this.send(run ? `${text}\r` : text);
    this.focus();
  }

  selectionText() {
    return this.term?.getSelection() ?? "";
  }

  output(maxChars = OUTPUT_CHARS) {
    const term = this.term;
    if (!term) return "";
    const buffer = term.buffer.active;
    const end = buffer.baseY + buffer.cursorY;
    const lines: string[] = [];
    let size = 0;
    for (let idx = end; idx >= 0 && size < maxChars; idx -= 1) {
      const line = buffer.getLine(idx)?.translateToString(true) ?? "";
      lines.unshift(line);
      size += line.length + 1;
    }
    const text = lines.join("\n").replace(/\n+$/, "").replace(/^\n+/, "");
    return text.length > maxChars ? text.slice(text.length - maxChars) : text;
  }

  dismissIssue() {
    if (this.state.issue) this.update({ issue: undefined });
  }

  dispose() {
    if (this.disposed) return;
    this.disposed = true;
    window.removeEventListener("pagehide", this.onPageHide);
    this.stop(!this.state.kept);
    this.observer?.disconnect();
    window.clearTimeout(this.resizeTimer);
    this.term?.dispose();
    this.term = undefined;
    this.element.remove();
    this.listeners.clear();
  }

  private onPageHide = () => {
    const ids = [
      ...(this.state.kept ? [] : [this.main?.id]),
      this.setup?.id,
    ].filter((id): id is string => !!id);
    for (const id of ids) {
      this.client
        .removeTerminal(
          { id },
          { fetchInit: { credentials: "include", keepalive: true } },
        )
        .then(
          () => undefined,
          () => undefined,
        );
    }
  };

  private stop(removeMain: boolean) {
    this.controller.abort();
    this.skipSetup = undefined;
    const ids = [removeMain ? this.main?.id : undefined, this.setup?.id].filter(
      (id): id is string => !!id,
    );
    this.main = undefined;
    this.setup = undefined;
    this.queue = [];
    this.sentSize = "";
    for (const id of ids) {
      this.client.removeTerminal({ id }).then(
        () => undefined,
        () => undefined,
      );
    }
  }

  private async createTerminal(cols: number, rows: number) {
    const { response } = await this.client.createTerminal({
      workspaceRef: this.workspaceRef,
      cols,
      rows,
    });
    return response.id;
  }

  private async connect(generation: number, signal: AbortSignal) {
    const current = () => generation === this.generation && !this.disposed;
    try {
      const cols = this.term?.cols || 120;
      const rows = this.term?.rows || 32;
      const created = await Promise.allSettled([
        this.createTerminal(cols, rows),
        this.createTerminal(200, 50),
      ]);
      const failure = created.find((itm) => itm.status === "rejected");
      if (failure || !current()) {
        for (const itm of created) {
          if (itm.status !== "fulfilled") continue;
          this.client.removeTerminal({ id: itm.value }).then(
            () => undefined,
            () => undefined,
          );
        }
        if (failure) throw failure.reason;
        return;
      }
      const [mainID, setupID] = created.map((itm) =>
        itm.status === "fulfilled" ? itm.value : "",
      );
      this.main = { id: mainID, output: false, lastOutputAt: 0 };
      this.setup = { id: setupID, output: false, lastOutputAt: 0 };
      this.setStep("terminal", { status: "done" });
      this.update({ phase: "preparing", terminalId: mainID });
      this.syncSize();

      const mainClosed = this.listenMain(this.main, signal).then(
        (closed) => {
          if (closed && current()) {
            if (this.term) this.term.options.disableStdin = true;
            this.update({ phase: "ended" });
          }
        },
        (err) => {
          if (current() && !isAbort(err)) {
            this.update({ phase: "error", error: errorMessage(err) });
          }
        },
      );

      const { facts, skipped } = await this.runSetup(this.setup, signal);
      if (!current()) return;
      this.client.removeTerminal({ id: setupID }).then(
        () => undefined,
        () => undefined,
      );
      this.setup = undefined;
      if (this.state.phase !== "preparing") return;

      const tool = this.plan.tool;
      const toolStep = this.state.steps.find((step) => step.id === tool);
      const failed = toolStep?.status === "failed";
      const inputs = this.plan
        .commands(facts)
        .map((input): ConnectInput =>
          failed ? { ...input, run: false } : input,
        );

      if (failed && tool) {
        this.update({
          warning: `${toolLabels[tool]} could not be installed in the Workspace (${toolStep?.detail ?? "unknown error"}). The command is typed in the terminal so that you can run it once ${toolLabels[tool]} is available.`,
        });
      }

      this.setStep("connect", { status: "running" });
      await this.waitIdle(this.main, 300, 8000, signal);
      if (!skipped) {
        let sentAt = 0;
        for (const [idx, input] of inputs.entries()) {
          if (idx > 0) {
            await this.waitIdle(this.main, 200, 3000, signal, sentAt);
          }
          sentAt = Date.now();
          this.send(input.text + (input.run ? "\r" : ""));
          if (input.cursor) this.send(`${ESC}[D`.repeat(input.cursor));
          if (input.run && idx === inputs.length - 1) {
            this.watchUntil = Date.now() + ISSUE_WINDOW_MS;
            this.watchLine = "";
          }
        }
      }
      this.setStep("connect", { status: skipped ? "skipped" : "done" });
      if (this.term) this.term.options.disableStdin = false;
      this.update({ phase: "ready" });
      this.focus();
      await mainClosed;
    } catch (err) {
      if (!current() || isAbort(err)) return;
      this.update({ phase: "error", error: errorMessage(err) });
    }
  }

  private async waitIdle(
    stream: Stream,
    idleMs: number,
    maxMs: number,
    signal: AbortSignal,
    since = 0,
  ) {
    const deadline = Date.now() + maxMs;
    while (Date.now() < deadline) {
      if (
        stream.output &&
        stream.lastOutputAt > since &&
        Date.now() - stream.lastOutputAt >= idleMs
      ) {
        return;
      }
      await sleep(50, signal);
    }
  }

  private async listen(
    stream: Stream,
    signal: AbortSignal,
    onMessage: (msg: ListenTerminalResponse, first: boolean) => void,
  ): Promise<boolean> {
    let failures = 0;
    while (!signal.aborted) {
      let first = true;
      try {
        const call = this.client.listenTerminal(
          { id: stream.id },
          { abort: signal },
        );
        for await (const msg of call.responses) {
          if (signal.aborted) break;
          failures = 0;
          if (msg.type.oneofKind === "close") return true;
          if (msg.type.oneofKind === "stdout") {
            stream.output = true;
            stream.lastOutputAt = Date.now();
          }
          onMessage(msg, first);
          first = false;
        }
      } catch (err) {
        if (signal.aborted || isAbort(err)) throw err;
        if (/not found|does not exist/i.test(errorMessage(err))) return true;
        failures += 1;
        if (failures > 5) throw err;
      }
      await sleep(Math.min(1000 * 2 ** failures, 8000), signal);
    }
    return false;
  }

  private listenMain(stream: Stream, signal: AbortSignal) {
    return this.listen(stream, signal, (msg) => {
      const term = this.term;
      if (msg.type.oneofKind === "windowSize") {
        term?.reset();
        return;
      }
      if (msg.type.oneofKind !== "stdout") return;
      const data = msg.type.stdout.data;
      term?.write(data);
      if (this.watchUntil && Date.now() < this.watchUntil) {
        this.watch(this.decoder.decode(data, { stream: true }));
      } else if (this.watchUntil) {
        this.watchUntil = 0;
      }
    });
  }

  private watch(text: string) {
    const lines = stripAnsi(`${this.watchLine}${text}`).split(/\r?\n/);
    this.watchLine = (lines.pop() ?? "").slice(-512);
    for (const raw of lines) {
      const line = (raw.split("\r").at(-1) ?? "").trim();
      if (line && issuePatterns.some((pattern) => pattern.test(line))) {
        this.watchUntil = 0;
        this.update({ issue: line.slice(0, 300) });
        return;
      }
    }
  }

  private async runSetup(
    stream: Stream,
    signal: AbortSignal,
  ): Promise<{ facts: Facts; skipped: boolean }> {
    const nonce = randomNonce();
    const script = buildSetupScript(this.plan, this.domain, nonce);
    const facts: Facts = {};
    const decoder = new TextDecoder();
    let pending = "";
    let started = false;
    let done = false;
    let log: string[] = [];

    const apply = (event: SetupEvent) => {
      switch (event.type) {
        case "start":
          started = true;
          return;
        case "done":
          done = true;
          return;
        case "fact":
          facts[event.key] = event.value;
          this.update({ facts: { ...facts } });
          return;
        case "step": {
          const step = this.state.steps.find((itm) => itm.id === event.id);
          const tool = event.id in toolLabels ? event.id : undefined;
          if (
            event.status === "ok" &&
            tool &&
            event.detail &&
            event.detail !== toolLabels[tool as keyof typeof toolLabels] &&
            !facts[tool]
          ) {
            facts[tool] = event.detail;
          }
          this.setStep(
            event.id,
            event.status === "run"
              ? { status: "running", detail: undefined }
              : event.status === "ok"
                ? {
                    status: "done",
                    detail: tool
                      ? step?.status === "running"
                        ? "Installed"
                        : "Available"
                      : undefined,
                  }
                : event.status === "skip"
                  ? { status: "skipped" }
                  : { status: "failed", detail: event.detail },
          );
          return;
        }
      }
    };

    const consume = (text: string) => {
      const lines = `${pending}${text}`.split("\n");
      pending = lines.pop() ?? "";
      let changed = false;
      for (const raw of lines) {
        const line = stripAnsi(raw.split("\r").filter(Boolean).at(-1) ?? "");
        const event = parseSetupLine(line, nonce);
        if (event) {
          apply(event);
        } else if (started && line.trim()) {
          log = [...log, line].slice(-LOG_LINES);
          changed = true;
        }
      }
      if (changed) this.update({ log });
    };

    const setupSignal = AbortSignal.any([
      signal,
      AbortSignal.timeout(SETUP_TIMEOUT_MS),
    ]);
    const skipController = new AbortController();
    this.skipSetup = () => skipController.abort();
    const listenSignal = AbortSignal.any([setupSignal, skipController.signal]);

    const listening = this.listen(stream, listenSignal, (msg) => {
      if (msg.type.oneofKind === "windowSize") {
        pending = "";
        log = [];
        return;
      }
      if (msg.type.oneofKind === "stdout") {
        consume(decoder.decode(msg.type.stdout.data, { stream: true }));
      }
    });
    listening.catch(() => undefined);

    try {
      this.setStep("dns", { status: "running" });
      await this.waitIdle(stream, 250, 6000, listenSignal);
      await this.writeAll(stream.id, setupCommand(script, nonce), listenSignal);
      await listening;
    } catch (err) {
      if (signal.aborted) throw err;
      const skipped = skipController.signal.aborted;
      for (const step of this.state.steps) {
        if (
          step.id !== "connect" &&
          (step.status === "running" || step.status === "pending")
        ) {
          this.setStep(
            step.id,
            skipped
              ? { status: "skipped", detail: undefined }
              : {
                  status: "failed",
                  detail: setupSignal.aborted ? "timed out" : errorMessage(err),
                },
          );
        }
      }
      return { facts, skipped };
    } finally {
      this.skipSetup = undefined;
    }

    if (!done) {
      for (const step of this.state.steps) {
        if (
          step.id !== "connect" &&
          (step.status === "running" || step.status === "pending")
        ) {
          this.setStep(step.id, {
            status: "failed",
            detail: "the setup ended unexpectedly",
          });
        }
      }
    }
    return { facts, skipped: false };
  }

  private async writeAll(id: string, text: string, signal: AbortSignal) {
    const data = this.encoder.encode(text);
    for (let offset = 0; offset < data.length; offset += CHUNK_SIZE) {
      await this.client.writeTerminalData(
        { id, data: data.subarray(offset, offset + CHUNK_SIZE) },
        { abort: signal },
      );
    }
  }

  private send(text: string) {
    if (!this.main || !text) return;
    this.queue.push(this.encoder.encode(text));
    void this.flush();
  }

  private async flush() {
    if (this.flushing) return;
    this.flushing = true;
    const signal = this.controller.signal;
    try {
      while (this.queue.length > 0 && this.main && !signal.aborted) {
        const id = this.main.id;
        const parts: Uint8Array[] = [];
        let size = 0;
        while (this.queue.length > 0 && size < CHUNK_SIZE) {
          const next = this.queue[0];
          if (size + next.length <= CHUNK_SIZE) {
            parts.push(next);
            size += next.length;
            this.queue.shift();
          } else {
            const room = CHUNK_SIZE - size;
            parts.push(next.subarray(0, room));
            this.queue[0] = next.subarray(room);
            size += room;
          }
        }
        const data = new Uint8Array(size);
        let offset = 0;
        for (const part of parts) {
          data.set(part, offset);
          offset += part.length;
        }
        try {
          await this.client.writeTerminalData({ id, data }, { abort: signal });
        } catch (err) {
          if (signal.aborted || isAbort(err)) return;
        }
      }
    } finally {
      this.flushing = false;
    }
  }
}
