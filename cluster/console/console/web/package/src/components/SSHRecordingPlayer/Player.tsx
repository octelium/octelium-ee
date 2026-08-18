import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { SSHSession } from "@/apis/visibilityv1/visibilityv1";
import { isDev } from "@/utils";
import { getClientVisibilityAccessLog } from "@/utils/client";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import { Gauge, Play, RotateCcw, Square } from "lucide-react";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { twMerge } from "tailwind-merge";

const MAX_FRAME_DELAY_MS = 2000;
const UI_UPDATE_INTERVAL_MS = 50;

interface SSHFrame {
  content: Uint8Array<ArrayBufferLike>;
  timestamp: number;
}

interface APIResponse {
  frames: SSHFrame[];
  hasMore: boolean;
  nextPage?: number;
  total?: number;
}

interface XTermSSHReplayProps {
  sshSession: SSHSession;
  initialPage?: number;
}

const generateMockPage = (page: number, fromMs?: number): APIResponse => {
  const encoder = new TextEncoder();
  const baseTime = 1000000000000;

  const allFrames: { text: string; delay: number }[] = [
    { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 0 },
    { text: "l", delay: 150 },
    { text: "s -la\r\n", delay: 200 },
    { text: "total 48\r\n", delay: 50 },
    {
      text: "drwxr-xr-x  5 user user 4096 Dec 17 10:30 \x1b[34m.\x1b[0m\r\n",
      delay: 20,
    },
    {
      text: "drwxr-xr-x  3 root root 4096 Dec 10 08:15 \x1b[34m..\x1b[0m\r\n",
      delay: 20,
    },
    {
      text: "-rw-r--r--  1 user user 3526 Dec 10 08:15 .bashrc\r\n",
      delay: 20,
    },
    {
      text: "drwxr-xr-x  3 user user 4096 Dec 15 14:20 \x1b[34mDocuments\x1b[0m\r\n",
      delay: 20,
    },
    { text: "\x1b[32muser@server:~$\x1b[0m ", delay: 100 },
    { text: "cd Documents\r\n", delay: 2000 },
    { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 50 },
    { text: "cat readme.txt\r\n", delay: 1500 },
    { text: "=================================\r\n", delay: 30 },
    { text: "Welcome to the SSH Replay Demo!\r\n", delay: 30 },
    { text: "=================================\r\n", delay: 30 },
    { text: "\x1b[32muser@server:~/Documents$\x1b[0m ", delay: 200 },
    { text: "exit\r\n", delay: 500 },
    { text: "\x1b[33mConnection to server closed.\x1b[0m\r\n", delay: 50 },
  ];

  let currentTime = baseTime;
  const timestampedFrames: SSHFrame[] = allFrames.map((item) => {
    currentTime += item.delay;
    return { content: encoder.encode(item.text), timestamp: currentTime };
  });

  const PAGE_SIZE = 6;
  const filtered =
    fromMs != null
      ? timestampedFrames.filter((f) => f.timestamp >= fromMs)
      : timestampedFrames;

  const start = (page - 1) * PAGE_SIZE;
  const end = start + PAGE_SIZE;
  const pageFrames = filtered.slice(start, end);
  const hasMore = end < filtered.length;

  return {
    frames: pageFrames,
    hasMore,
    nextPage: hasMore ? page + 1 : undefined,
    total: filtered.length,
  };
};

type PlaybackStatus =
  | { type: "idle" }
  | { type: "loading"; page: number }
  | { type: "playing"; current: number; total: number }
  | { type: "done" }
  | { type: "stopped" }
  | { type: "error"; message: string };

const statusLabel = (s: PlaybackStatus): string =>
  s.type === "idle"
    ? "Ready"
    : s.type === "loading"
      ? `Loading page ${s.page}…`
      : s.type === "playing"
        ? `Frame ${s.current} / ${s.total}`
    : s.type === "done"
      ? "Playback complete"
      : s.type === "stopped"
        ? "Stopped — press play to restart"
        : `Error: ${s.message}`;

const statusColor = (s: PlaybackStatus): string =>
  s.type === "error"
    ? "text-red-400"
    : s.type === "done"
      ? "text-emerald-400"
      : s.type === "stopped"
        ? "text-amber-400"
        : "text-slate-400";

const formatTime = (ms: number): string => {
  const totalSec = Math.floor(ms / 1000);
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
};

export const XTermSSHReplay: React.FC<XTermSSHReplayProps> = ({
  initialPage = 1,
  sshSession,
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal | null>(null);
  const fitAddon = useRef<FitAddon | null>(null);
  const abortController = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);
  const isScrubbingRef = useRef(false);
  const lastUiUpdateRef = useRef(0);
  const playbackSpeedRef = useRef(1);

  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);
  const [playbackStatus, setPlaybackStatus] = useState<PlaybackStatus>({
    type: "idle",
  });

  // Progress scrubbing state — all in milliseconds relative to session start
  const [durationMs, setDurationMs] = useState(0);
  const [positionMs, setPositionMs] = useState(0);
  const sessionStartMs = useRef<number>(0);

  useEffect(() => {
    if (!terminalRef.current) return;
    mountedRef.current = true;

    terminal.current = new Terminal({
      convertEol: false,
      fontFamily:
        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace",
      fontWeight: 450,
      fontWeightBold: 700,
      cursorBlink: false,
      scrollback: 5000,
      fontSize: 14,
      lineHeight: 1.4,
      theme: {
        background: "#0f172a",
        foreground: "#e2e8f0",
        cursor: "#f8fafc",
        selectionBackground: "#334155",
        black: "#475569",
        red: "#fb7185",
        green: "#34d399",
        yellow: "#fbbf24",
        blue: "#60a5fa",
        magenta: "#c084fc",
        cyan: "#22d3ee",
        white: "#cbd5e1",
        brightBlack: "#64748b",
        brightRed: "#fda4af",
        brightGreen: "#6ee7b7",
        brightYellow: "#fde68a",
        brightBlue: "#93c5fd",
        brightMagenta: "#d8b4fe",
        brightCyan: "#67e8f9",
        brightWhite: "#f8fafc",
      },
    });

    fitAddon.current = new FitAddon();
    terminal.current.loadAddon(fitAddon.current);
    terminal.current.open(terminalRef.current);
    fitAddon.current.fit();

    const handleResize = () => fitAddon.current?.fit();
    window.addEventListener("resize", handleResize);

    const resizeObserver =
      typeof ResizeObserver === "undefined"
        ? undefined
        : new ResizeObserver(handleResize);
    resizeObserver?.observe(terminalRef.current);

    return () => {
      mountedRef.current = false;
      abortController.current?.abort();
      abortController.current = null;
      resizeObserver?.disconnect();
      window.removeEventListener("resize", handleResize);
      terminal.current?.dispose();
      terminal.current = null;
    };
  }, []);

  const fetchPage = useCallback(
    async (
      page: number,
      signal: AbortSignal,
      fromMs?: number,
    ): Promise<APIResponse> => {
      if (isDev()) {
        await new Promise((resolve) => setTimeout(resolve, 150));
        if (signal.aborted) throw new DOMException("Aborted", "AbortError");
        return generateMockPage(page, fromMs);
      }

      const { response } =
        await getClientVisibilityAccessLog().listSSHSessionRecording({
          sessionID: sshSession.id,
          page: page - 1,
          from:
            fromMs != null ? Timestamp.fromDate(new Date(fromMs)) : undefined,
        });

      return {
        frames: response.items.flatMap((x) =>
          x.timestamp
            ? [
                {
                  content: x.data,
                  timestamp: Timestamp.toDate(x.timestamp).getTime(),
                },
              ]
            : [],
        ),
        hasMore: response.listResponseMeta?.hasMore ?? false,
        nextPage: response.listResponseMeta?.hasMore ? page + 1 : undefined,
        total: response.listResponseMeta?.totalCount,
      };
    },
    [sshSession.id],
  );

  const runPlayback = useCallback(
    async (fromMs?: number) => {
      if (!terminal.current || !mountedRef.current) return;

      abortController.current?.abort();
      const controller = new AbortController();
      abortController.current = controller;
      const signal = controller.signal;

      const isCurrentRun = () =>
        mountedRef.current && abortController.current === controller;

      setIsPlaying(true);
      terminal.current.clear();

      try {
        let currentPage = initialPage;
        let frameCount = 0;
        let totalFrames: number | undefined;
        let firstTimestamp: number | undefined;
        let previousTimestamp: number | undefined;
        let lastTimestamp: number | undefined;

        while (true) {
          if (signal.aborted || !isCurrentRun()) break;
          setPlaybackStatus({ type: "loading", page: currentPage });

          const response = await fetchPage(currentPage, signal, fromMs);
          totalFrames = response.total ?? totalFrames;

          for (const frame of [...response.frames].sort(
            (a, b) => a.timestamp - b.timestamp,
          )) {
            if (signal.aborted || !isCurrentRun()) break;

            if (firstTimestamp == null) {
              firstTimestamp = frame.timestamp;
              if (fromMs == null) {
                sessionStartMs.current = firstTimestamp;
              }
            }

            const rawDelay =
              previousTimestamp == null
                ? 0
                : Math.max(0, frame.timestamp - previousTimestamp);
            const delay = Math.min(
              rawDelay / playbackSpeedRef.current,
              MAX_FRAME_DELAY_MS,
            );

            if (delay > 0) {
              await new Promise<void>((resolve, reject) => {
                const onAbort = () => {
                  clearTimeout(id);
                  reject(new DOMException("Aborted", "AbortError"));
                };
                const id = setTimeout(() => {
                  signal.removeEventListener("abort", onAbort);
                  resolve();
                }, delay);
                signal.addEventListener("abort", onAbort, { once: true });
              });
            }

            if (!terminal.current || !isCurrentRun()) break;
            terminal.current.write(frame.content);
            previousTimestamp = frame.timestamp;
            lastTimestamp = frame.timestamp;
            frameCount += 1;

            const elapsed = frame.timestamp - sessionStartMs.current;
            const now = performance.now();
            const shouldUpdate =
              now - lastUiUpdateRef.current >= UI_UPDATE_INTERVAL_MS;
            if (fromMs == null && firstTimestamp != null && shouldUpdate) {
              setDurationMs(frame.timestamp - firstTimestamp);
            }
            if (!isScrubbingRef.current && shouldUpdate) {
              setPositionMs(elapsed);
            }
            if (shouldUpdate) lastUiUpdateRef.current = now;

            if (shouldUpdate) {
              setPlaybackStatus({
                type: "playing",
                current: frameCount,
                total: totalFrames ?? frameCount,
              });
            }
          }

          if (!response.hasMore || !response.nextPage) break;
          currentPage = response.nextPage;
        }

        if (!isCurrentRun()) return;
        if (signal.aborted) {
          setPlaybackStatus({ type: "stopped" });
          return;
        }

        if (frameCount === 0 || firstTimestamp == null || lastTimestamp == null) {
          setPlaybackStatus({ type: "done" });
          return;
        }

        if (!isCurrentRun()) return;
        setPlaybackStatus(
          signal.aborted ? { type: "stopped" } : { type: "done" },
        );
        if (!signal.aborted) {
          if (fromMs == null) {
            setDurationMs(lastTimestamp - firstTimestamp);
          }
          setPositionMs(
            fromMs == null ? lastTimestamp - firstTimestamp : durationMs,
          );
        }
      } catch (err: unknown) {
        if (!isCurrentRun()) return;
        if (err instanceof DOMException && err.name === "AbortError") {
          setPlaybackStatus({ type: "stopped" });
        } else {
          const message = err instanceof Error ? err.message : "Unknown error";
          setPlaybackStatus({ type: "error", message });
          console.error("Playback error:", err);
        }
      } finally {
        if (abortController.current === controller) {
          setIsPlaying(false);
          abortController.current = null;
        }
      }
    },
    [initialPage, fetchPage, durationMs],
  );

  const handlePlayPause = useCallback(() => {
    if (isPlaying) {
      abortController.current?.abort();
    } else {
      runPlayback();
    }
  }, [isPlaying, runPlayback]);

  const resetTerminal = useCallback(() => {
    const controller = abortController.current;
    abortController.current = null;
    controller?.abort();
    setIsPlaying(false);
    terminal.current?.clear();
    setPlaybackStatus({ type: "idle" });
    setPositionMs(0);
    setDurationMs(0);
    sessionStartMs.current = 0;
  }, []);

  const handleScrubStart = useCallback(() => {
    isScrubbingRef.current = true;
    if (isPlaying) {
      abortController.current?.abort();
    }
  }, [isPlaying]);

  const inputRef = useRef<HTMLInputElement>(null);

  const handleScrubEnd = useCallback(() => {
    if (!isScrubbingRef.current) return;
    isScrubbingRef.current = false;
    const targetMs = Number(inputRef.current?.value ?? 0);
    setPositionMs(targetMs);
    const absoluteMs = sessionStartMs.current + targetMs;
    runPlayback(absoluteMs);
  }, [runPlayback]);

  const handleScrubMove = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (isScrubbingRef.current) {
        setPositionMs(Number(e.target.value));
      }
    },
    [],
  );

  const progressPct =
    durationMs > 0
      ? Math.max(0, Math.min(100, Math.round((positionMs / durationMs) * 100)))
      : 0;

  return (
    <div className="w-full flex flex-col rounded-xl overflow-hidden border border-slate-700 shadow-[0_4px_24px_rgba(1,4,9,0.4)]">
      <div
        ref={terminalRef}
        className="h-[min(60vh,640px)] min-h-[320px] max-h-[720px]"
        style={{ background: "#0f172a" }}
      />

      <div className="bg-[#111827] border-t border-slate-700 px-4 pt-2.5 pb-3 flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <span className="text-[0.65rem] font-semibold text-slate-500 w-10 shrink-0 text-right tabular-nums">
            {formatTime(positionMs)}
          </span>

          <div className="relative flex-1 flex items-center">
            <div className="absolute w-full h-1 bg-slate-700 rounded-full pointer-events-none">
              <div
                className="h-full bg-emerald-500 rounded-full transition-[width] duration-100"
                style={{ width: `${progressPct}%` }}
              />
            </div>
            <input
              ref={inputRef}
              type="range"
              min={0}
              max={durationMs || 100}
              step={100}
              value={positionMs}
              disabled={durationMs === 0}
              aria-label="Replay position"
              aria-valuetext={`${formatTime(positionMs)} of ${formatTime(durationMs)}`}
              onPointerDown={handleScrubStart}
              onChange={handleScrubMove}
              onPointerUp={handleScrubEnd}
              onPointerCancel={handleScrubEnd}
              onKeyUp={handleScrubEnd}
              onBlur={handleScrubEnd}
              className="relative w-full h-2 appearance-none bg-transparent cursor-pointer accent-emerald-500 disabled:cursor-default disabled:opacity-40"
              style={{
                WebkitAppearance: "none",
              }}
            />
          </div>

          <span className="text-[0.65rem] font-semibold text-slate-500 w-10 shrink-0 tabular-nums">
            {formatTime(durationMs)}
          </span>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            type="button"
            onClick={handlePlayPause}
            className={twMerge(
              "flex items-center justify-center w-8 h-8 rounded-md cursor-pointer transition-colors duration-150",
              isPlaying
                ? "text-red-400 hover:text-red-300 hover:bg-red-400/10"
                : "text-emerald-400 hover:text-emerald-300 hover:bg-emerald-400/10",
            )}
            title={isPlaying ? "Stop" : "Play"}
            aria-label={isPlaying ? "Stop replay" : "Play replay"}
          >
            {isPlaying ? (
              <Square size={14} strokeWidth={2.5} />
            ) : (
              <Play size={14} strokeWidth={2.5} />
            )}
          </button>

          <button
            type="button"
            onClick={resetTerminal}
            className="flex items-center justify-center w-8 h-8 rounded-md cursor-pointer text-slate-400 hover:text-slate-200 hover:bg-slate-400/10 transition-colors duration-150"
            title="Reset"
            aria-label="Reset replay"
          >
            <RotateCcw size={13} strokeWidth={2.5} />
          </button>

          <div className="flex items-center gap-1.5 rounded-md border border-slate-700 px-1.5 py-1 text-slate-400">
            <Gauge size={12} strokeWidth={2.25} aria-hidden="true" />
            <span className="sr-only">Playback speed</span>
            {[0.5, 1, 2].map((speed) => (
              <button
                type="button"
                key={speed}
                onClick={() => {
                  playbackSpeedRef.current = speed;
                  setPlaybackSpeed(speed);
                }}
                aria-pressed={playbackSpeed === speed}
                className={twMerge(
                  "rounded px-1.5 py-0.5 text-[0.62rem] font-bold transition-colors duration-150",
                  playbackSpeed === speed
                    ? "bg-emerald-500 text-slate-950"
                    : "text-slate-400 hover:bg-slate-700 hover:text-slate-100",
                )}
              >
                {speed}x
              </button>
            ))}
          </div>

          <span className="hidden text-[0.62rem] font-semibold text-slate-500 sm:inline">
            Long pauses are capped at 2s
          </span>

          <div className="flex-1" />

          <span
            role="status"
            aria-live="polite"
            className={twMerge(
              "text-right text-[0.72rem] font-semibold",
              statusColor(playbackStatus),
            )}
          >
            {statusLabel(playbackStatus)}
          </span>
        </div>
      </div>
    </div>
  );
};
