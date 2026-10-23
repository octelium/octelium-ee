import { Service } from "@/apis/corev1/corev1";
import { Workspace } from "@/apis/cordiumv1/cordiumv1";
import { Agent_State } from "@/apis/enterprisev1/enterprisev1";
import {
  registerAgentSection,
  useAgentAsk,
} from "@/components/AgentDock/context";
import {
  KeepAliveStore,
  useKeepAliveStore,
} from "@/components/ResourceLayout/keepAlive";
import { twMerge } from "@/lib/utils";
import { CopyButton, StatusDot, type DotTone } from "@/pages/Agent/ui";
import { getDomain } from "@/utils";
import { getServiceHostname } from "@/utils/octelium";
import { ActionIcon, Button, Loader, Tooltip } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import {
  Check,
  ChevronDown,
  CircleAlert,
  Container,
  ExternalLink,
  FileText,
  Info,
  Minus,
  Power,
  RotateCcw,
  Sparkles,
  SquareTerminal,
  TriangleAlert,
  X,
} from "lucide-react";
import * as React from "react";
import { terminalContext } from "./agent";
import {
  buildConnectPlan,
  mainCommand,
  toolLabels,
  type ConnectPlan,
} from "./plan";
import {
  ConnectSession,
  planSteps,
  type SessionState,
  type SessionStep,
} from "./session";
import {
  errorMessage,
  useAgentWorkspace,
  workspaceName,
  workspaceStateLabel,
} from "./workspace";

const ease = [0.22, 1, 0.36, 1] as const;

const emptyLog: string[] = [];

const getCordiumURL = (path: string) => `https://cordium.${getDomain()}${path}`;

const workspaceURL = (ws: Workspace, page = "") =>
  getCordiumURL(
    `/workspaces/${encodeURIComponent(ws.metadata?.name ?? "")}${page}`,
  );

const useStore = () => {
  const scoped = useKeepAliveStore();
  const [local] = React.useState(() => new KeepAliveStore());
  React.useEffect(() => {
    if (scoped) return;
    local.retain();
    return () => local.scheduleDispose();
  }, [scoped, local]);
  return scoped ?? local;
};

const useSession = (
  store: KeepAliveStore,
  key: string | undefined,
  create: () => ConnectSession,
) => {
  const session = key ? store.ensure(key, create) : undefined;
  const previous = React.useRef<string | undefined>(undefined);
  React.useEffect(() => {
    const last = previous.current;
    previous.current = key;
    if (last && last !== key) store.release(last);
  }, [store, key]);
  return session;
};

const noSession = () => () => {};

const useSessionState = (session?: ConnectSession) =>
  React.useSyncExternalStore(session?.subscribe ?? noSession, () =>
    session?.getSnapshot(),
  );

const toolStepLabel = (step: SessionStep) => {
  if (!(step.id in toolLabels)) return step.label;
  switch (step.status) {
    case "running":
      return `Installing ${step.label}`;
    case "done":
      return step.detail === "Installed"
        ? `Installed ${step.label}`
        : `${step.label} is available`;
    case "failed":
      return `${step.label} could not be installed`;
    case "skipped":
      return `${step.label} was skipped`;
    default:
      return `Check ${step.label}`;
  }
};

const StepIcon = (props: { status: SessionStep["status"] }) => (
  <span
    className={twMerge(
      "flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full transition-colors duration-300",
      props.status === "done" && "bg-terminal-success text-terminal",
      props.status === "failed" && "bg-terminal-error text-terminal",
      props.status === "running" && "ring-1 ring-terminal-line",
      (props.status === "pending" || props.status === "skipped") &&
        "ring-1 ring-terminal-line text-terminal-muted",
    )}
  >
    {props.status === "done" ? (
      <Check size={11} strokeWidth={3} />
    ) : props.status === "failed" ? (
      <X size={11} strokeWidth={3} />
    ) : props.status === "running" ? (
      <Loader size={10} color="var(--color-terminal-accent)" />
    ) : props.status === "skipped" ? (
      <Minus size={10} strokeWidth={3} />
    ) : (
      <span className="h-1 w-1 rounded-full bg-terminal-muted" />
    )}
  </span>
);

const StepList = (props: { steps: SessionStep[] }) => (
  <ol className="flex flex-col gap-2" aria-label="Connection steps">
    {props.steps.map((step) => (
      <li
        key={step.id}
        data-step={step.id}
        data-status={step.status}
        className="flex min-w-0 items-center gap-2.5"
      >
        <StepIcon status={step.status} />
        <span
          className={twMerge(
            "min-w-0 flex-1 truncate text-xs font-semibold",
            step.status === "pending" || step.status === "skipped"
              ? "text-terminal-muted"
              : step.status === "failed"
                ? "text-terminal-error"
                : "text-terminal-text",
          )}
        >
          {toolStepLabel(step)}
        </span>
        {step.detail && step.status !== "done" && (
          <span
            className="max-w-[45%] shrink-0 truncate text-micro text-terminal-muted"
            title={step.detail}
          >
            {step.detail}
          </span>
        )}
      </li>
    ))}
  </ol>
);

const CanvasCard = (props: {
  title: string;
  subtitle?: React.ReactNode;
  tone?: "default" | "error";
  icon?: React.ReactNode;
  children?: React.ReactNode;
  footer?: React.ReactNode;
}) => (
  <motion.div
    initial={{ opacity: 0, y: 6 }}
    animate={{ opacity: 1, y: 0 }}
    exit={{ opacity: 0, y: 4 }}
    transition={{ duration: 0.25, ease }}
    className="absolute inset-0 z-10 flex items-center justify-center overflow-y-auto bg-terminal/80 p-4 backdrop-blur-[2px]"
  >
    <div
      role="status"
      className="w-full max-w-sm rounded-xl border border-terminal-line bg-terminal-panel p-4 shadow-overlay"
    >
      <div className="flex items-start gap-3">
        <span
          className={twMerge(
            "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-terminal-line bg-terminal-hover",
            props.tone === "error"
              ? "text-terminal-error"
              : "text-terminal-bright",
          )}
        >
          {props.icon ?? <SquareTerminal size={16} strokeWidth={2.2} />}
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-sm font-bold text-terminal-bright">
            {props.title}
          </p>
          {props.subtitle && (
            <p className="mt-0.5 text-xs leading-5 text-terminal-muted">
              {props.subtitle}
            </p>
          )}
        </div>
      </div>
      {props.children && <div className="mt-4">{props.children}</div>}
      {props.footer && (
        <div className="mt-4 flex flex-wrap items-center justify-end gap-2">
          {props.footer}
        </div>
      )}
    </div>
  </motion.div>
);

const canvasButton =
  "inline-flex h-7 cursor-pointer items-center gap-1.5 rounded-md px-2.5 text-xs font-semibold outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-terminal-accent disabled:cursor-default disabled:opacity-50";

const CanvasButton = (props: {
  children: React.ReactNode;
  onClick?: () => void;
  href?: string;
  primary?: boolean;
  disabled?: boolean;
}) => {
  const className = twMerge(
    canvasButton,
    props.primary
      ? "bg-terminal-bright text-terminal hover:bg-terminal-text"
      : "text-terminal-text hover:bg-terminal-hover",
  );
  return props.href ? (
    <a
      className={className}
      href={props.href}
      target="_blank"
      rel="noreferrer noopener"
    >
      {props.children}
    </a>
  ) : (
    <button
      type="button"
      className={className}
      disabled={props.disabled}
      onClick={props.onClick}
    >
      {props.children}
    </button>
  );
};

const ProgressCard = (props: {
  service: Service;
  plan: ConnectPlan;
  workspace?: Workspace;
  steps: SessionStep[];
  log?: string[];
  onSkip?: () => void;
}) => {
  const [details, setDetails] = React.useState(false);
  const log = props.log ?? emptyLog;
  const logRef = React.useRef<HTMLPreElement>(null);

  React.useEffect(() => {
    const element = logRef.current;
    if (element) element.scrollTop = element.scrollHeight;
  }, [log, details]);

  return (
    <CanvasCard
      title={`Connecting to ${getServiceHostname(props.service)}`}
      subtitle={
        <>
          {props.plan.tool ? (
            <span className="font-mono text-terminal-text">
              {props.plan.label}
            </span>
          ) : (
            "A shell"
          )}{" "}
          in your agent Workspace{" "}
          <span className="font-semibold text-terminal-text">
            {workspaceName(props.workspace)}
          </span>
        </>
      }
      footer={
        (log.length > 0 || props.onSkip) && (
          <>
            {log.length > 0 && (
              <CanvasButton onClick={() => setDetails((value) => !value)}>
                <ChevronDown
                  size={13}
                  className={twMerge(
                    "transition-transform duration-150",
                    details && "rotate-180",
                  )}
                />
                {details ? "Hide details" : "Show details"}
              </CanvasButton>
            )}
            <span className="flex-1" />
            {props.onSkip && (
              <CanvasButton onClick={props.onSkip}>
                Skip and open the shell
              </CanvasButton>
            )}
          </>
        )
      }
    >
      <StepList steps={props.steps} />
      {details && log.length > 0 && (
        <pre
          ref={logRef}
          aria-label="Setup output"
          className="mt-3 max-h-40 overflow-auto whitespace-pre-wrap break-all rounded-lg border border-terminal-line bg-terminal p-2 font-mono text-micro leading-4 text-terminal-muted"
        >
          {log.join("\n")}
        </pre>
      )}
    </CanvasCard>
  );
};

const StatusPill = (props: {
  tone: DotTone;
  label: string;
  pulse?: boolean;
}) => (
  <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600">
    <StatusDot tone={props.tone} pulse={props.pulse} />
    {props.label}
  </span>
);

const Strip = (props: {
  tone: "info" | "warning" | "error";
  children: React.ReactNode;
  action?: React.ReactNode;
  onDismiss?: () => void;
}) => {
  const Icon =
    props.tone === "info"
      ? Info
      : props.tone === "warning"
        ? TriangleAlert
        : CircleAlert;
  return (
    <div
      role={props.tone === "info" ? "note" : "alert"}
      className={twMerge(
        "flex items-start gap-2 border-b px-4 py-2 text-xs leading-5",
        props.tone === "info" && "border-slate-200 bg-slate-50 text-slate-600",
        props.tone === "warning" &&
          "border-amber-200 bg-amber-50 text-amber-800",
        props.tone === "error" && "border-red-200 bg-red-50 text-red-700",
      )}
    >
      <Icon size={14} className="mt-[3px] shrink-0" />
      <div className="min-w-0 flex-1 break-words">{props.children}</div>
      {props.action}
      {props.onDismiss && (
        <ActionIcon
          size="sm"
          variant="subtle"
          color="gray"
          aria-label="Dismiss"
          onClick={props.onDismiss}
        >
          <X size={13} />
        </ActionIcon>
      )}
    </div>
  );
};

const statusOf = (
  state: SessionState | undefined,
  env: ReturnType<typeof useAgentWorkspace>,
): { tone: DotTone; label: string; pulse?: boolean } => {
  if (state) {
    switch (state.phase) {
      case "ready":
        return state.issue
          ? { tone: "amber", label: "Needs attention" }
          : { tone: "green", label: "Connected" };
      case "ended":
        return { tone: "gray", label: "Ended" };
      case "error":
        return { tone: "red", label: "Failed" };
      default:
        return { tone: "blue", label: "Connecting", pulse: true };
    }
  }
  if (env.agentQuery.isError || env.startError) {
    return { tone: "red", label: "Unavailable" };
  }
  if (env.starting || env.agentQuery.isPending || env.running) {
    return { tone: "blue", label: "Starting", pulse: true };
  }
  if (env.stopping) return { tone: "blue", label: "Stopping", pulse: true };
  return { tone: "gray", label: "Not connected" };
};

const EnvironmentCanvas = (props: {
  env: ReturnType<typeof useAgentWorkspace>;
  service: Service;
  plan: ConnectPlan;
}) => {
  const { env } = props;
  const ws = env.workspace;
  const failure = ws?.status?.failure;

  if (env.agentQuery.isPending) {
    return (
      <CanvasCard
        title="Looking up your agent Workspace"
        icon={<Loader size={14} color="var(--color-terminal-accent)" />}
        subtitle="Terminals run in the Cordium Workspace of your Octelium agent."
      />
    );
  }

  if (env.agentQuery.isError) {
    return (
      <CanvasCard
        title="Terminals are not available"
        tone="error"
        icon={<CircleAlert size={17} />}
        subtitle={errorMessage(env.agentQuery.error)}
        footer={
          <CanvasButton onClick={() => void env.agentQuery.refetch()}>
            <RotateCcw size={13} />
            Retry
          </CanvasButton>
        }
      />
    );
  }

  if (!ws) {
    const isNew = env.agent?.state === Agent_State.NOT_INITIALIZED;
    return (
      <CanvasCard
        title={
          isNew
            ? "Set up your agent environment"
            : "Repair your agent environment"
        }
        icon={<Sparkles size={16} />}
        subtitle={
          isNew
            ? "Terminals run in the Cordium Workspace of your Octelium agent, with your own identity and permissions. It is set up once and kept in your personal Cordium Space."
            : "The Workspace of your Octelium agent is missing from your Cordium Space."
        }
        footer={
          <>
            {env.initialize.error && (
              <span className="mr-auto text-micro text-terminal-error">
                {errorMessage(env.initialize.error)}
              </span>
            )}
            <CanvasButton
              primary
              disabled={env.initialize.isPending}
              onClick={() => env.initialize.mutate()}
            >
              {env.initialize.isPending ? (
                <Loader size={10} color="var(--color-terminal)" />
              ) : (
                <Sparkles size={13} />
              )}
              {isNew ? "Set up" : "Repair"}
            </CanvasButton>
          </>
        }
      />
    );
  }

  if (env.starting || env.running) {
    return (
      <ProgressCard
        service={props.service}
        plan={props.plan}
        workspace={ws}
        steps={[
          {
            id: "workspace",
            label: "Start the agent Workspace",
            status: env.running ? "done" : "running",
            detail: workspaceStateLabel(env.state),
          },
          ...planSteps(props.plan),
        ]}
      />
    );
  }

  if (env.stopping) {
    return (
      <CanvasCard
        title="Your agent Workspace is stopping"
        icon={<Loader size={14} color="var(--color-terminal-accent)" />}
        subtitle="You can start it again once it is stopped."
      />
    );
  }

  return (
    <CanvasCard
      title={
        failure
          ? "Your agent Workspace failed"
          : env.startError
            ? "Your agent Workspace could not be started"
            : "Your agent Workspace is stopped"
      }
      tone={failure || env.startError ? "error" : "default"}
      icon={
        failure || env.startError ? (
          <CircleAlert size={17} />
        ) : (
          <Power size={16} />
        )
      }
      subtitle={
        env.startError
          ? errorMessage(env.startError)
          : failure?.message ||
            "Start it to open a terminal and connect to this Service."
      }
      footer={
        <>
          <CanvasButton href={workspaceURL(ws, "/logs")}>
            <FileText size={13} />
            View logs
          </CanvasButton>
          <CanvasButton
            primary
            disabled={env.startPending}
            onClick={env.startWorkspace}
          >
            {env.startPending ? (
              <Loader size={10} color="var(--color-terminal)" />
            ) : (
              <Power size={13} />
            )}
            Start
          </CanvasButton>
        </>
      }
    />
  );
};

const Snippets = (props: {
  plan: ConnectPlan;
  disabled: boolean;
  onType: (text: string) => void;
}) =>
  props.plan.snippets.length > 0 ? (
    <div
      role="toolbar"
      aria-label="Quick commands"
      className="flex items-center gap-1.5 overflow-x-auto border-b border-slate-200 bg-white px-4 py-1.5 [scrollbar-width:none]"
    >
      <span className="mr-1 shrink-0 text-micro font-semibold uppercase tracking-[0.06em] text-slate-400">
        Quick
      </span>
      {props.plan.snippets.map((snippet) => (
        <Tooltip
          key={snippet.text}
          label={`Type ${snippet.text}`}
          withArrow
          openDelay={400}
        >
          <button
            type="button"
            disabled={props.disabled}
            onClick={() => props.onType(snippet.text)}
            className="shrink-0 cursor-pointer whitespace-nowrap rounded-md border border-slate-200 bg-white px-2 py-0.5 font-mono text-micro text-slate-600 shadow-card outline-none transition-colors duration-150 hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-default disabled:opacity-50"
          >
            {snippet.label}
          </button>
        </Tooltip>
      ))}
    </div>
  ) : null;

const ServiceConnect = (props: { service: Service }) => {
  const { service } = props;
  const domain = getDomain();
  const plan = React.useMemo(
    () => buildConnectPlan(service, domain),
    [service, domain],
  );
  const env = useAgentWorkspace();
  const store = useStore();
  const ws = env.workspace;
  const { available, ask } = useAgentAsk();
  const terminalRef = React.useRef<HTMLDivElement>(null);
  const sectionRef = React.useRef<HTMLDivElement>(null);
  const name = service.metadata?.name ?? "";

  const key =
    env.running && ws?.metadata?.uid
      ? `connect:${ws.metadata.uid}:${plan.tool ?? "shell"}:${mainCommand(plan) ?? ""}`
      : undefined;
  const session = useSession(
    store,
    key,
    () => new ConnectSession({ plan, workspace: ws!, domain }),
  );
  const state = useSessionState(session);

  React.useEffect(() => {
    const element = terminalRef.current;
    if (!session || !element) return;
    session.attach(element);
    session.start();
    return () => session.detach(element);
  }, [session]);

  const pinned = React.useCallback(
    (selection?: boolean) =>
      session
        ? terminalContext({
            service,
            plan: session.plan,
            facts: session.getSnapshot().facts,
            workspace: session.workspace,
            terminalId: session.getSnapshot().terminalId,
            output: session.output(),
            selection: selection ? session.selectionText() : undefined,
            issue: session.getSnapshot().issue,
          })
        : {
            label: `Service ${name}`,
            resources: [
              {
                apiVersion: "core/v1",
                kind: "Service",
                name,
                uid: service.metadata?.uid,
              },
            ],
          },
    [session, service, name],
  );

  React.useEffect(() => {
    const element = sectionRef.current;
    if (!element) return;
    return registerAgentSection(element, () => pinned());
  }, [pinned]);

  const askAgent = (selection?: boolean, text?: string) =>
    ask({ pinned: pinned(selection), fresh: true, text });

  const status = statusOf(state, env);
  const command = mainCommand(plan, state?.facts);
  const ready = state?.phase === "ready";
  const preparing = state?.phase === "starting" || state?.phase === "preparing";
  const cordiumURL =
    ws && state?.terminalId
      ? `${workspaceURL(ws, "/terminals")}?terminal=${encodeURIComponent(state.terminalId)}`
      : undefined;

  return (
    <div className="@container w-full">
      <div
        ref={sectionRef}
        data-agent-section
        className="flex h-[calc(100dvh-11.125rem)] min-h-[26rem] w-full flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card"
      >
        <header className="flex flex-col gap-3 border-b border-slate-200 bg-slate-50/60 px-4 py-3 @xl:flex-row @xl:items-center">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
              <SquareTerminal size={16} strokeWidth={2.2} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <h2 className="min-w-0 truncate text-sm font-bold text-slate-800">
                  {plan.tool ? `Connect with ${plan.label}` : "Terminal"}
                </h2>
                <StatusPill {...status} />
              </div>
              <div className="mt-0.5 flex min-w-0 items-center gap-0.5">
                <span
                  className="min-w-0 truncate font-mono text-micro text-slate-500"
                  title={command ?? plan.address}
                  data-testid="connect-command"
                >
                  {command ?? plan.address}
                </span>
                <CopyButton
                  value={() => command ?? plan.address}
                  label={command ? "Copy the command" : "Copy the address"}
                />
              </div>
            </div>
          </div>
          <div className="flex shrink-0 items-center justify-end gap-1.5">
            {available && (
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<Sparkles size={13} />}
                onClick={() => askAgent()}
              >
                Ask agent
              </Button>
            )}
            <Tooltip label="Start a new session" withArrow openDelay={300}>
              <ActionIcon
                variant="default"
                size={26}
                aria-label="Start a new session"
                disabled={!session || preparing}
                onClick={() => session?.restart()}
              >
                <RotateCcw size={13} />
              </ActionIcon>
            </Tooltip>
            <Tooltip
              label="Continue in Cordium. The terminal stays open after you close this drawer"
              withArrow
              openDelay={300}
              multiline
              w={240}
            >
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<ExternalLink size={13} />}
                component="a"
                href={cordiumURL ?? "#"}
                target="_blank"
                rel="noreferrer noopener"
                data-disabled={!cordiumURL || undefined}
                onClick={(event: React.MouseEvent) => {
                  if (!cordiumURL) {
                    event.preventDefault();
                    return;
                  }
                  session?.keep();
                }}
              >
                Open in Cordium
              </Button>
            </Tooltip>
          </div>
        </header>

        {service.spec?.isDisabled && (
          <Strip tone="warning">
            This Service is disabled, so its connections are refused until it is
            enabled again.
          </Strip>
        )}
        {state?.issue && (
          <Strip
            tone="error"
            onDismiss={() => session?.dismissIssue()}
            action={
              available && (
                <Button
                  size="compact-xs"
                  variant="white"
                  color="dark"
                  leftSection={<Sparkles size={11} strokeWidth={2.5} />}
                  onClick={() =>
                    askAgent(
                      false,
                      "Why did this connection fail and how can I fix it?",
                    )
                  }
                >
                  Ask agent
                </Button>
              )
            }
          >
            The connection seems to have failed:{" "}
            <code className="break-all font-mono font-semibold">
              {state.issue}
            </code>
          </Strip>
        )}
        {state?.warning && <Strip tone="warning">{state.warning}</Strip>}
        {(plan.hint || plan.inferred) && (
          <Strip tone={plan.hint?.tone ?? "info"}>
            {plan.hint?.text}
            {plan.inferred && (
              <span className={twMerge(plan.hint && "text-slate-500")}>
                {plan.hint && " "}
                {plan.inferred}
              </span>
            )}
          </Strip>
        )}
        <Snippets
          plan={plan}
          disabled={!ready}
          onType={(text) => session?.type(text)}
        />

        <div className="relative min-h-0 flex-1 bg-terminal">
          <div
            ref={terminalRef}
            data-testid="connect-terminal"
            className="absolute inset-0 py-2 pl-3 pr-1"
          />
          {ready && state.selection && available && (
            <button
              type="button"
              onClick={() => askAgent(true)}
              className="absolute right-4 top-2 z-[5] flex cursor-pointer items-center gap-1.5 rounded-md border border-terminal-line bg-terminal-panel px-2 py-1 text-micro font-semibold text-terminal-text shadow-raised outline-none transition-colors duration-150 hover:bg-terminal-hover focus-visible:ring-2 focus-visible:ring-terminal-accent"
            >
              <Sparkles size={11} />
              Ask about the selection
            </button>
          )}
          <AnimatePresence>
            {!session && (
              <EnvironmentCanvas
                key="environment"
                env={env}
                service={service}
                plan={plan}
              />
            )}
            {session && state && preparing && (
              <ProgressCard
                key="progress"
                service={service}
                plan={plan}
                workspace={ws}
                steps={[
                  {
                    id: "workspace",
                    label: "Start the agent Workspace",
                    status: "done",
                  },
                  ...state.steps,
                ]}
                log={state.log}
                onSkip={
                  state.phase === "preparing" ? () => session.skip() : undefined
                }
              />
            )}
            {session && state?.phase === "error" && (
              <CanvasCard
                key="error"
                title="The terminal is not available"
                tone="error"
                icon={<CircleAlert size={17} />}
                subtitle={state.error}
                footer={
                  <CanvasButton primary onClick={() => session.restart()}>
                    <RotateCcw size={13} />
                    Try again
                  </CanvasButton>
                }
              />
            )}
          </AnimatePresence>
          {session && state?.phase === "ended" && (
            <div className="absolute inset-x-0 bottom-4 z-10 flex justify-center px-4">
              <div
                role="status"
                className="flex items-center gap-3 rounded-lg border border-terminal-line bg-terminal-panel py-1.5 pl-3 pr-1.5 text-xs text-terminal-text shadow-overlay"
              >
                The session has ended
                <CanvasButton primary onClick={() => session.restart()}>
                  <RotateCcw size={13} />
                  New session
                </CanvasButton>
              </div>
            </div>
          )}
        </div>

        <div className="flex min-w-0 items-center gap-3 border-t border-terminal-line bg-terminal-panel px-3 py-1.5 text-micro text-terminal-muted">
          <span className="flex min-w-0 items-center gap-1.5">
            <Container size={12} className="shrink-0" />
            <span className="truncate" title={ws?.metadata?.name}>
              {ws ? workspaceName(ws) : "Agent Workspace"}
            </span>
          </span>
          {state?.terminalId && (
            <span className="hidden truncate font-mono @md:inline">
              {state.terminalId}
            </span>
          )}
          <span className="flex-1" />
          {state?.kept && (
            <span className="flex shrink-0 items-center gap-1 text-terminal-text">
              <ExternalLink size={11} />
              Kept open in Cordium
            </span>
          )}
          {state && state.cols > 0 && (
            <span className="shrink-0 font-mono tabular-nums">
              {state.cols}×{state.rows}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ServiceConnect;
