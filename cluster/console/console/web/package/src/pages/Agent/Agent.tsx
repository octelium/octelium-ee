import type { AgentInfo } from "@/apis/consoleagent/protocol";
import {
  StartWorkspaceRequest,
  StopWorkspaceRequest,
  Workspace,
  Workspace_Status_State,
} from "@/apis/cordiumv1/cordiumv1";
import {
  Agent as AgentEnv,
  Agent_State,
  Agent_Workspace,
  Agent_Workspace_Type,
} from "@/apis/enterprisev1/enterprisev1";
import {
  DeleteOptions,
  GetOptions,
  ObjectReference,
} from "@/apis/metav1/metav1";
import Meta from "@/components/Meta";
import { getClientAgent, getClientCordium } from "@/utils/client";
import { Badge, Button, Loader, Menu } from "@mantine/core";
import type { RpcError } from "@protobuf-ts/runtime-rpc";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion, MotionConfig } from "framer-motion";
import {
  Check,
  ChevronDown,
  ChevronsUpDown,
  CircleAlert,
  Container,
  ExternalLink,
  FileText,
  Fingerprint,
  Plus,
  Power,
  RefreshCw,
  ShieldCheck,
  Sparkles,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import Chat from "./Chat";
import { AgentClient } from "./client";
import {
  AgentMark,
  ConfirmModal,
  Shimmer,
  StatusDot,
  type DotTone,
} from "./ui";
import {
  getCordiumURL,
  getWorkspaceCordiumURL,
  getWorkspaceDisplayName,
  isWorkspaceStarting,
  isWorkspaceStopping,
  readStorage,
  workspaceStateLabel,
  writeStorage,
} from "./utils";

const devAgentURL = import.meta.env.VITE_CONSOLE_AGENT_URL as
  string | undefined;

const agentKey = ["agent", "environment"];
const workspaceStorageKey = "octelium-console-agent-workspace";
const agentStartupHintMs = 90000;
const ease = [0.22, 1, 0.36, 1] as const;

const getErrorMessage = (err: unknown): string =>
  (err as RpcError | Error | undefined)?.message ?? String(err);

const workspaceTone = (ws?: Workspace): DotTone => {
  if (ws?.status?.failure) return "red";
  if (ws?.status?.state === Workspace_Status_State.RUNNING) return "green";
  if (isWorkspaceStarting(ws) || isWorkspaceStopping(ws)) return "blue";
  return "gray";
};

const Shell = (props: { children: React.ReactNode }) => (
  <div className="h-[calc(100dvh-92px)] min-h-[480px]">
    <div className="relative flex h-full overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      {props.children}
    </div>
  </div>
);

const StatePanel = (props: {
  title: string;
  icon?: React.ReactNode;
  tone?: "default" | "error";
  loading?: boolean;
  header?: React.ReactNode;
  banner?: React.ReactNode;
  children?: React.ReactNode;
}) => (
  <div className="flex min-w-0 flex-1 flex-col">
    <Meta title="Agent" />
    <div className="flex h-12 shrink-0 items-center gap-2.5 px-3">
      <AgentMark />
      <span className="flex-1 text-body font-semibold text-slate-800">
        Agent
      </span>
      {props.header}
    </div>
    {props.banner && <div className="px-3">{props.banner}</div>}
    <div className="flex min-h-0 flex-1 overflow-y-auto px-4 py-8">
      <motion.div
        key={props.title}
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease }}
        className="m-auto flex w-full max-w-md flex-col items-center text-center"
      >
        {props.icon !== undefined ? (
          <span
            className={twMerge(
              "flex h-11 w-11 items-center justify-center rounded-2xl shadow-sm",
              props.tone === "error"
                ? "bg-red-50 text-red-600 ring-1 ring-red-200"
                : "bg-slate-900 text-white",
            )}
          >
            {props.icon}
          </span>
        ) : (
          <AgentMark size="lg" live={props.loading} />
        )}
        <h2 className="mt-4 text-base font-bold text-slate-900">
          {props.title}
        </h2>
        <div className="mt-1.5 w-full text-xs leading-5 text-slate-500">
          {props.children}
        </div>
      </motion.div>
    </div>
  </div>
);

const bootSteps: { label: string; states: Workspace_Status_State[] }[] = [
  {
    label: "Starting the Workspace",
    states: [
      Workspace_Status_State.STOPPED,
      Workspace_Status_State.INIT_REQUEST,
      Workspace_Status_State.INITIALIZING,
    ],
  },
  {
    label: "Preparing the image",
    states: [
      Workspace_Status_State.PULLING_IMAGE,
      Workspace_Status_State.BUILDING_IMAGE,
    ],
  },
  {
    label: "Starting the runtime",
    states: [
      Workspace_Status_State.STARTING_RUNTIME,
      Workspace_Status_State.PREPARING,
    ],
  },
  {
    label: "Starting the agent",
    states: [Workspace_Status_State.RUNNING],
  },
];

const BootSteps = (props: { state: Workspace_Status_State }) => {
  const current = Math.max(
    0,
    bootSteps.findIndex((step) => step.states.includes(props.state)),
  );

  return (
    <ol className="mx-auto mt-5 w-full max-w-[280px] space-y-0 text-left">
      {bootSteps.map((step, idx) => {
        const done = idx < current;
        const active = idx === current;
        return (
          <li key={step.label} className="relative flex gap-3 pb-4 last:pb-0">
            {idx < bootSteps.length - 1 && (
              <span
                className={twMerge(
                  "absolute left-[9.5px] top-6 h-[calc(100%-20px)] w-px transition-colors duration-500",
                  done ? "bg-emerald-300" : "bg-slate-200",
                )}
              />
            )}
            <span
              className={twMerge(
                "relative z-[1] flex h-5 w-5 shrink-0 items-center justify-center rounded-full transition-colors duration-300",
                done
                  ? "bg-emerald-500 text-white"
                  : active
                    ? "bg-white ring-1 ring-slate-300"
                    : "bg-white ring-1 ring-slate-200",
              )}
            >
              {done ? (
                <Check size={11} strokeWidth={3} />
              ) : active ? (
                <Loader size={10} color="gray" />
              ) : (
                <span className="h-1.5 w-1.5 rounded-full bg-slate-300" />
              )}
            </span>
            <span className="min-w-0 pt-px">
              <span
                className={twMerge(
                  "block text-xs font-semibold",
                  done
                    ? "text-slate-700"
                    : active
                      ? "text-slate-900"
                      : "text-slate-400",
                )}
              >
                {active ? <Shimmer>{step.label}</Shimmer> : step.label}
              </span>
              {active && idx < bootSteps.length - 1 && (
                <span className="block text-micro text-slate-500">
                  {workspaceStateLabel(props.state)}
                </span>
              )}
            </span>
          </li>
        );
      })}
    </ol>
  );
};

const AgentFooter = (props: { info: AgentInfo }) => (
  <div className="flex items-center gap-2.5 px-2 py-1.5">
    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 shadow-card">
      <Container size={15} />
    </span>
    <span className="min-w-0 flex-1">
      <span className="block truncate text-xs font-semibold text-slate-800">
        {props.info.workspace?.name ?? "Local agent"}
      </span>
      <span className="block truncate text-micro text-slate-500">
        {props.info.name} {props.info.version}
      </span>
    </span>
  </div>
);

const AgentReady = (props: {
  url: string;
  info?: AgentInfo;
  footer?: React.ReactNode;
  banner?: React.ReactNode;
  header?: React.ReactNode;
}) => {
  const client = React.useMemo(() => new AgentClient(props.url), [props.url]);
  const infoQuery = useQuery({
    queryKey: ["agent", client.baseUrl, "info"],
    queryFn: () => client.info(),
    initialData: props.info,
  });

  if (!infoQuery.data) {
    return (
      <StatePanel title="Connecting to the agent" loading header={props.header}>
        <p>Reaching the agent inside your Workspace…</p>
      </StatePanel>
    );
  }

  return (
    <Chat
      client={client}
      info={infoQuery.data}
      footer={props.footer ?? <AgentFooter info={infoQuery.data} />}
      banner={props.banner}
    />
  );
};

const WorkspaceMenu = (props: {
  agent: AgentEnv;
  current: Agent_Workspace;
  compact?: boolean;
  onSelect: (uid: string) => void;
  onRestart: () => void;
  onStop: () => void;
}) => {
  const queryClient = useQueryClient();
  const { agent, current } = props;
  const ws = current.workspace!;
  const isPrimary = current.type === Agent_Workspace_Type.PRIMARY;
  const [confirm, setConfirm] = React.useState<"workspace" | "space">();

  const refresh = () => queryClient.invalidateQueries({ queryKey: agentKey });

  const create = useMutation({
    mutationFn: async () =>
      (
        await getClientAgent().createAgentWorkspace({
          displayName: "Fresh Workspace",
          isEphemeral: false,
        })
      ).response,
    onSuccess: async (res) => {
      await refresh();
      if (res.workspace?.metadata?.uid) {
        props.onSelect(res.workspace.metadata.uid);
      }
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  const running = ws.status?.state === Workspace_Status_State.RUNNING;
  const tone = workspaceTone(ws);
  const stateLabel = workspaceStateLabel(
    ws.status?.state ?? Workspace_Status_State.UNKNOWN,
  );

  return (
    <>
      <Menu
        position={props.compact ? "bottom-end" : "top-start"}
        width={props.compact ? 272 : "target"}
        withinPortal
      >
        <Menu.Target>
          {props.compact ? (
            <button
              type="button"
              className="flex h-8 min-w-0 cursor-pointer items-center gap-2 rounded-lg border border-slate-200 bg-white px-2.5 text-slate-700 shadow-card transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised"
            >
              {create.isPending ? (
                <Loader size={10} color="gray" />
              ) : (
                <StatusDot tone={tone} pulse={tone === "blue"} />
              )}
              <span className="max-w-[160px] truncate text-xs font-semibold">
                {getWorkspaceDisplayName(ws)}
              </span>
              <ChevronDown size={13} className="shrink-0 text-slate-400" />
            </button>
          ) : (
            <button
              type="button"
              className="group flex w-full cursor-pointer items-center gap-2.5 rounded-lg px-2 py-1.5 text-left outline-none transition-colors duration-150 hover:bg-slate-200/50 focus-visible:ring-2 focus-visible:ring-slate-400"
            >
              <span className="relative flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600 shadow-card">
                {create.isPending ? (
                  <Loader size={12} color="gray" />
                ) : (
                  <Container size={15} />
                )}
                <span className="absolute -bottom-0.5 -right-0.5 flex rounded-full bg-slate-50 p-[2px]">
                  <StatusDot tone={tone} pulse={tone === "blue"} />
                </span>
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-xs font-semibold text-slate-800">
                  {getWorkspaceDisplayName(ws)}
                </span>
                <span className="block truncate text-micro text-slate-500">
                  {stateLabel} · {ws.metadata?.name}
                </span>
              </span>
              <ChevronsUpDown size={14} className="shrink-0 text-slate-400" />
            </button>
          )}
        </Menu.Target>
        <Menu.Dropdown>
          {agent.workspaces.length > 1 && (
            <>
              <Menu.Label>Workspaces</Menu.Label>
              {agent.workspaces.map((itm) => {
                const uid = itm.workspace?.metadata?.uid ?? "";
                const isCurrent = uid === ws.metadata?.uid;
                return (
                  <Menu.Item
                    key={uid}
                    leftSection={
                      <StatusDot tone={workspaceTone(itm.workspace)} />
                    }
                    rightSection={
                      isCurrent ? (
                        <Check size={13} strokeWidth={2.75} />
                      ) : itm.type === Agent_Workspace_Type.PRIMARY ? (
                        <Badge size="xs" variant="light" color="gray">
                          primary
                        </Badge>
                      ) : undefined
                    }
                    onClick={() => !isCurrent && props.onSelect(uid)}
                  >
                    <span className="block truncate">
                      {getWorkspaceDisplayName(itm.workspace)}
                    </span>
                  </Menu.Item>
                );
              })}
              <Menu.Divider />
            </>
          )}
          <Menu.Label>
            {getWorkspaceDisplayName(ws)} · {ws.metadata?.name}
          </Menu.Label>
          <Menu.Item
            leftSection={<ExternalLink size={13} />}
            component="a"
            href={getWorkspaceCordiumURL(ws)}
            target="_blank"
            rel="noreferrer noopener"
          >
            Open in Cordium
          </Menu.Item>
          <Menu.Item
            leftSection={<FileText size={13} />}
            component="a"
            href={getWorkspaceCordiumURL(ws, "logs")}
            target="_blank"
            rel="noreferrer noopener"
          >
            View logs
          </Menu.Item>
          <Menu.Item
            leftSection={<RefreshCw size={13} />}
            disabled={!running}
            onClick={props.onRestart}
          >
            Restart
          </Menu.Item>
          <Menu.Item
            leftSection={<Power size={13} />}
            disabled={!running && !isWorkspaceStarting(ws)}
            onClick={props.onStop}
          >
            Stop
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            leftSection={<Plus size={13} />}
            disabled={create.isPending}
            onClick={() => create.mutate()}
          >
            New fresh Workspace
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            color="red"
            leftSection={<Trash2 size={13} />}
            onClick={() => setConfirm("workspace")}
          >
            Delete Workspace
          </Menu.Item>
          <Menu.Item
            color="red"
            leftSection={<Trash2 size={13} />}
            onClick={() => setConfirm("space")}
          >
            Delete agent environment
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>

      <ConfirmModal
        opened={confirm === "workspace"}
        onClose={() => setConfirm(undefined)}
        title="Delete the Workspace"
        confirmLabel="Delete Workspace"
        onConfirm={async () => {
          await getClientCordium().deleteWorkspace(
            DeleteOptions.create({ uid: ws.metadata?.uid }),
          );
          await refresh();
        }}
      >
        <span className="font-semibold text-slate-800">
          {getWorkspaceDisplayName(ws)}
        </span>{" "}
        is permanently deleted together with its conversations and files.
        {isPrimary &&
          " A new primary Workspace is created the next time you open the agent."}
      </ConfirmModal>

      <ConfirmModal
        opened={confirm === "space"}
        onClose={() => setConfirm(undefined)}
        title="Delete the agent environment"
        confirmLabel="Delete environment"
        onConfirm={async () => {
          await getClientCordium().deleteSpace(
            DeleteOptions.create({ uid: agent.spaceRef?.uid }),
          );
          await refresh();
        }}
      >
        Your agent's Cordium Space is permanently deleted together with all of
        its Workspaces, conversations and files.
      </ConfirmModal>
    </>
  );
};

const OutdatedBanner = () => {
  const queryClient = useQueryClient();
  const initialize = useMutation({
    mutationFn: async () =>
      (await getClientAgent().initializeAgent({})).response,
    onSuccess: (data) => {
      queryClient.setQueryData(agentKey, data);
      toast.success(
        "The agent environment is updated. Restart the Workspace to use the new configuration",
      );
    },
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  return (
    <div className="mt-1 flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 py-1.5 pl-3 pr-1.5 text-xs text-amber-800">
      <TriangleAlert size={13} className="shrink-0" />
      <span className="min-w-0 flex-1">
        Your agent environment does not match the Cluster's agent configuration
      </span>
      <Button
        size="compact-xs"
        color="dark"
        loading={initialize.isPending}
        onClick={() => initialize.mutate()}
      >
        Update
      </Button>
    </div>
  );
};

const WorkspaceRunner = (props: {
  agent: AgentEnv;
  current: Agent_Workspace;
  onSelect: (uid: string) => void;
}) => {
  const queryClient = useQueryClient();
  const initial = props.current.workspace!;
  const uid = initial.metadata?.uid ?? "";
  const [userStopped, setUserStopped] = React.useState(false);
  const [restarting, setRestarting] = React.useState(false);
  const [slowURL, setSlowURL] = React.useState<string>();
  const startedRef = React.useRef(false);

  const wsQuery = useQuery({
    queryKey: ["agent", "workspace", uid],
    queryFn: async () =>
      (await getClientCordium().getWorkspace(GetOptions.create({ uid })))
        .response,
    initialData: initial,
    refetchInterval: (query) => {
      const state = query.state.data?.status?.state;
      if (state === Workspace_Status_State.RUNNING) return 30000;
      if (state === Workspace_Status_State.STOPPED) return false;
      return 2000;
    },
  });

  const ws: Workspace = wsQuery.data ?? initial;
  const state = ws.status?.state ?? Workspace_Status_State.UNKNOWN;
  const stopped = state === Workspace_Status_State.STOPPED;
  const running = state === Workspace_Status_State.RUNNING;
  const hostname = ws.status?.hostname;

  const start = useMutation({
    mutationFn: async () =>
      getClientCordium().startWorkspace(
        StartWorkspaceRequest.create({
          workspaceRef: ObjectReference.create({
            uid,
            name: ws.metadata?.name,
          }),
        }),
      ),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: ["agent", "workspace", uid] }),
    onError: (err) => {
      if ((err as RpcError).code !== "ALREADY_EXISTS") {
        toast.error(getErrorMessage(err));
      }
    },
  });

  const stop = useMutation({
    mutationFn: async () =>
      getClientCordium().stopWorkspace(
        StopWorkspaceRequest.create({
          workspaceRef: ObjectReference.create({
            uid,
            name: ws.metadata?.name,
          }),
        }),
      ),
    onSettled: () =>
      queryClient.invalidateQueries({ queryKey: ["agent", "workspace", uid] }),
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  React.useEffect(() => {
    if (!stopped || start.isPending) return;
    if (restarting) {
      setRestarting(false);
      start.mutate();
      return;
    }
    if (!startedRef.current && !userStopped) {
      startedRef.current = true;
      start.mutate();
    }
  }, [stopped, restarting, userStopped, start]);

  const url = running && hostname ? `https://${hostname}` : undefined;

  const infoQuery = useQuery({
    queryKey: ["agent", url, "info"],
    queryFn: () => new AgentClient(url!).info(AbortSignal.timeout(8000)),
    enabled: !!url,
    retry: false,
    refetchInterval: (query) => (query.state.data ? false : 2500),
  });
  const infoReady = !!infoQuery.data;

  React.useEffect(() => {
    if (!url || infoReady) return;
    const timer = window.setTimeout(() => setSlowURL(url), agentStartupHintMs);
    return () => window.clearTimeout(timer);
  }, [url, infoReady]);

  const menu = (compact: boolean) => (
    <WorkspaceMenu
      agent={props.agent}
      current={{ ...props.current, workspace: ws }}
      compact={compact}
      onSelect={props.onSelect}
      onStop={() => {
        setUserStopped(true);
        stop.mutate();
      }}
      onRestart={() => {
        setRestarting(true);
        stop.mutate();
      }}
    />
  );

  const banner =
    props.agent.state === Agent_State.OUTDATED ? <OutdatedBanner /> : undefined;
  const failure = ws.status?.failure;
  const waitingTooLong = !!url && slowURL === url;

  if (url && infoQuery.data) {
    return (
      <AgentReady
        url={url}
        info={infoQuery.data}
        footer={menu(false)}
        header={menu(true)}
        banner={banner}
      />
    );
  }

  if (running) {
    return (
      <StatePanel
        title="Starting the agent"
        loading
        header={menu(true)}
        banner={banner}
      >
        <p>
          The Workspace is running and the agent is starting. The first start
          can take a few minutes while its dependencies are installed.
        </p>
        <BootSteps state={state} />
        {waitingTooLong && (
          <p className="mt-5 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-amber-800">
            The agent is taking longer than expected.{" "}
            <a
              className="font-semibold underline underline-offset-2"
              href={getWorkspaceCordiumURL(ws, "logs")}
              target="_blank"
              rel="noreferrer noopener"
            >
              Check the Workspace logs
            </a>
            {infoQuery.error ? ` (${getErrorMessage(infoQuery.error)})` : ""}
          </p>
        )}
      </StatePanel>
    );
  }

  if (isWorkspaceStarting(ws) || (stopped && (start.isPending || restarting))) {
    return (
      <StatePanel
        title="Starting your agent Workspace"
        loading
        header={menu(true)}
        banner={banner}
      >
        <p>Your conversations and files are kept between restarts.</p>
        <BootSteps state={state} />
      </StatePanel>
    );
  }

  if (isWorkspaceStopping(ws)) {
    return (
      <StatePanel
        title="Stopping your agent Workspace"
        loading
        header={menu(true)}
        banner={banner}
      >
        <p>Your conversations and files are kept while it is stopped.</p>
      </StatePanel>
    );
  }

  return (
    <StatePanel
      title={
        failure ? "The Workspace failed" : "Your agent Workspace is stopped"
      }
      tone={failure ? "error" : "default"}
      icon={failure ? <CircleAlert size={21} /> : <Power size={19} />}
      header={menu(true)}
      banner={banner}
    >
      {failure?.message && (
        <p className="mb-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-left font-mono text-micro text-red-700">
          {failure.message}
        </p>
      )}
      <p>
        Your conversations and files are kept while the Workspace is stopped.
      </p>
      <div className="mt-4 flex justify-center gap-2">
        <Button
          size="xs"
          color="dark"
          leftSection={<Power size={13} />}
          loading={start.isPending}
          onClick={() => {
            setUserStopped(false);
            start.mutate();
          }}
        >
          Resume
        </Button>
        {failure && (
          <Button
            size="xs"
            variant="default"
            leftSection={<FileText size={13} />}
            component="a"
            href={getWorkspaceCordiumURL(ws, "logs")}
            target="_blank"
            rel="noreferrer noopener"
          >
            View logs
          </Button>
        )}
      </div>
    </StatePanel>
  );
};

const setupPoints = [
  {
    icon: Container,
    text: "Runs inside a dedicated Workspace in your personal Cordium Space",
  },
  {
    icon: Fingerprint,
    text: "Acts on your behalf with your own Octelium identity and permissions",
  },
  {
    icon: ShieldCheck,
    text: "Asks for your approval before changing anything in the Cluster",
  },
];

const AgentEnvironment = () => {
  const queryClient = useQueryClient();
  const [selected, setSelected] = React.useState(() =>
    readStorage(workspaceStorageKey),
  );

  const agentQuery = useQuery({
    queryKey: agentKey,
    queryFn: async () => (await getClientAgent().getAgent({})).response,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const initialize = useMutation({
    mutationFn: async () =>
      (await getClientAgent().initializeAgent({})).response,
    onSuccess: (data) => queryClient.setQueryData(agentKey, data),
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  const select = (uid: string) => {
    writeStorage(workspaceStorageKey, uid);
    setSelected(uid);
  };

  if (agentQuery.isPending) {
    return (
      <StatePanel title="Loading your agent" loading>
        <p>Looking up your agent environment…</p>
      </StatePanel>
    );
  }

  if (agentQuery.isError) {
    return (
      <StatePanel
        title="The agent is not available"
        tone="error"
        icon={<CircleAlert size={21} />}
      >
        <p>{getErrorMessage(agentQuery.error)}</p>
        <Button
          className="mt-4"
          size="xs"
          variant="default"
          leftSection={<RefreshCw size={13} />}
          onClick={() => void agentQuery.refetch()}
        >
          Retry
        </Button>
      </StatePanel>
    );
  }

  const agent = agentQuery.data;
  const current =
    agent.workspaces.find((itm) => itm.workspace?.metadata?.uid === selected) ??
    agent.workspaces[0];

  if (agent.state === Agent_State.NOT_INITIALIZED || !current?.workspace) {
    const isNew = agent.state === Agent_State.NOT_INITIALIZED;
    return (
      <StatePanel
        title={isNew ? "Set up your agent" : "Repair your agent environment"}
        icon={<Sparkles size={19} />}
      >
        <p>
          An AI agent that inspects, troubleshoots and manages the Cluster for
          you.{" "}
          {isNew
            ? "It is set up once and kept in"
            : "Its Workspace is missing from"}{" "}
          your{" "}
          <a
            className="font-semibold text-blue-600 hover:text-blue-700"
            href={getCordiumURL()}
            target="_blank"
            rel="noreferrer noopener"
          >
            Cordium
          </a>{" "}
          Space.
        </p>
        <ul className="mt-5 space-y-2 text-left">
          {setupPoints.map((point) => (
            <li
              key={point.text}
              className="flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50/60 px-3 py-2.5"
            >
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-600">
                <point.icon size={14} />
              </span>
              <span className="text-xs leading-5 text-slate-700">
                {point.text}
              </span>
            </li>
          ))}
        </ul>
        <Button
          className="mt-5"
          size="sm"
          color="dark"
          leftSection={<Sparkles size={14} />}
          loading={initialize.isPending}
          onClick={() => initialize.mutate()}
        >
          {isNew ? "Set up my agent" : "Repair"}
        </Button>
      </StatePanel>
    );
  }

  return (
    <WorkspaceRunner
      key={current.workspace.metadata?.uid}
      agent={agent}
      current={current}
      onSelect={select}
    />
  );
};

const Agent = () => (
  <MotionConfig reducedMotion="user">
    <Shell>
      {devAgentURL ? <AgentReady url={devAgentURL} /> : <AgentEnvironment />}
    </Shell>
  </MotionConfig>
);

export default Agent;
