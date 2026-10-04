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
import { Badge, Button, Loader, Menu, Select } from "@mantine/core";
import type { RpcError } from "@protobuf-ts/runtime-rpc";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  Bot,
  CircleAlert,
  ExternalLink,
  FileText,
  Plus,
  Power,
  RefreshCw,
  Settings2,
  Sparkles,
  Trash2,
} from "lucide-react";
import * as React from "react";
import { toast } from "sonner";
import Chat from "./Chat";
import { AgentClient } from "./client";
import {
  getCordiumURL,
  getWorkspaceCordiumURL,
  getWorkspaceDisplayName,
  isWorkspaceStarting,
  isWorkspaceStopping,
  workspaceStateLabel,
} from "./utils";

const devAgentURL = import.meta.env.VITE_CONSOLE_AGENT_URL as
  string | undefined;

const agentKey = ["agent", "environment"];
const workspaceStorageKey = "octelium-console-agent-workspace";
const agentStartupHintMs = 90000;

const getErrorMessage = (err: unknown): string =>
  (err as RpcError | Error | undefined)?.message ?? String(err);

const readStoredWorkspace = (): string | undefined => {
  try {
    return localStorage.getItem(workspaceStorageKey) ?? undefined;
  } catch {
    return undefined;
  }
};

const storeWorkspace = (uid: string) => {
  try {
    localStorage.setItem(workspaceStorageKey, uid);
  } catch {
    return;
  }
};

const Card = (props: {
  icon?: React.ReactNode;
  title: string;
  children?: React.ReactNode;
  tone?: "default" | "error";
}) => (
  <motion.div
    initial={{ opacity: 0, y: 8 }}
    animate={{ opacity: 1, y: 0 }}
    transition={{ duration: 0.25, ease: "easeOut" }}
    className="mx-auto mt-10 w-full max-w-lg rounded-2xl border border-slate-200 bg-white px-6 py-6 text-center shadow-card"
  >
    <span
      className={
        props.tone === "error"
          ? "mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-red-50 text-red-600"
          : "mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-900 text-white"
      }
    >
      {props.icon ?? <Bot size={23} />}
    </span>
    <p className="mt-3 text-sm font-bold text-slate-800">{props.title}</p>
    <div className="mt-1.5 text-xs leading-5 text-slate-500">
      {props.children}
    </div>
  </motion.div>
);

const Header = (props: { children?: React.ReactNode }) => (
  <div className="flex flex-wrap items-center justify-between gap-3">
    <div className="flex items-center gap-2.5">
      <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-slate-900 text-white">
        <Sparkles size={17} />
      </span>
      <div>
        <h1 className="text-base font-bold text-slate-900">Agent</h1>
        <p className="text-micro text-slate-500">
          An AI agent running inside your own Cordium Workspace
        </p>
      </div>
    </div>
    {props.children}
  </div>
);

const AgentReady = (props: { url: string; info?: AgentInfo }) => {
  const client = React.useMemo(() => new AgentClient(props.url), [props.url]);
  const infoQuery = useQuery({
    queryKey: ["agent", client.baseUrl, "info"],
    queryFn: () => client.info(),
    initialData: props.info,
  });

  if (!infoQuery.data) {
    return (
      <Card title="Connecting to the agent">
        <Loader size="sm" color="gray" />
      </Card>
    );
  }

  return <Chat client={client} info={infoQuery.data} />;
};

const WorkspaceActions = (props: {
  agent: AgentEnv;
  current: Agent_Workspace;
  onSelect: (uid: string) => void;
  onRestart: () => void;
  onStop: () => void;
}) => {
  const queryClient = useQueryClient();
  const { agent, current } = props;
  const ws = current.workspace!;
  const isPrimary = current.type === Agent_Workspace_Type.PRIMARY;

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

  const removeWorkspace = useMutation({
    mutationFn: async () =>
      getClientCordium().deleteWorkspace(
        DeleteOptions.create({ uid: ws.metadata?.uid }),
      ),
    onSuccess: () => refresh(),
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  const removeSpace = useMutation({
    mutationFn: async () =>
      getClientCordium().deleteSpace(
        DeleteOptions.create({ uid: agent.spaceRef?.uid }),
      ),
    onSuccess: () => refresh(),
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  const running = ws.status?.state === Workspace_Status_State.RUNNING;

  return (
    <div className="flex flex-wrap items-center gap-2">
      {agent.workspaces.length > 1 && (
        <Select
          size="xs"
          className="w-56"
          aria-label="Workspace"
          value={ws.metadata?.uid ?? null}
          allowDeselect={false}
          data={agent.workspaces.map((itm) => ({
            value: itm.workspace?.metadata?.uid ?? "",
            label: `${getWorkspaceDisplayName(itm.workspace)}${itm.type === Agent_Workspace_Type.PRIMARY ? " (primary)" : ` · ${itm.workspace?.metadata?.name}`}`,
          }))}
          onChange={(value) => value && props.onSelect(value)}
        />
      )}
      <Badge
        variant="light"
        color={running ? "green" : isWorkspaceStarting(ws) ? "blue" : "gray"}
      >
        {workspaceStateLabel(
          ws.status?.state ?? Workspace_Status_State.UNKNOWN,
        )}
      </Badge>
      <Menu position="bottom-end" withinPortal>
        <Menu.Target>
          <Button
            size="xs"
            variant="default"
            leftSection={<Settings2 size={13} />}
            loading={
              create.isPending ||
              removeWorkspace.isPending ||
              removeSpace.isPending
            }
          >
            Workspace
          </Button>
        </Menu.Target>
        <Menu.Dropdown>
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
            onClick={() => create.mutate()}
          >
            New fresh Workspace
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            color="red"
            leftSection={<Trash2 size={13} />}
            onClick={() => {
              if (
                window.confirm(
                  `Delete the Workspace "${getWorkspaceDisplayName(ws)}"? Its conversations and files are deleted as well.${isPrimary ? " A new primary Workspace is created the next time you open the agent." : ""}`,
                )
              ) {
                removeWorkspace.mutate();
              }
            }}
          >
            Delete Workspace
          </Menu.Item>
          <Menu.Item
            color="red"
            leftSection={<Trash2 size={13} />}
            onClick={() => {
              if (
                window.confirm(
                  "Delete the whole agent environment (i.e. its Cordium Space and all of its Workspaces, conversations and files)?",
                )
              ) {
                removeSpace.mutate();
              }
            }}
          >
            Delete agent environment
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
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

  const actions = (
    <WorkspaceActions
      agent={props.agent}
      current={{ ...props.current, workspace: ws }}
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

  const failure = ws.status?.failure;
  const waitingTooLong = !!url && slowURL === url;

  let body: React.ReactNode;
  if (url && infoQuery.data) {
    body = <AgentReady url={url} info={infoQuery.data} />;
  } else if (running) {
    body = (
      <Card title="Waiting for the agent to start">
        <Loader size="sm" color="gray" className="mx-auto my-2" />
        <p>
          The Workspace is running and the agent is starting. The first start
          can take a few minutes since its dependencies are being installed.
        </p>
        {waitingTooLong && (
          <p className="mt-2 text-amber-700">
            The agent is taking longer than expected.{" "}
            <a
              className="font-semibold text-blue-600"
              href={getWorkspaceCordiumURL(ws, "logs")}
              target="_blank"
              rel="noreferrer noopener"
            >
              Check the Workspace logs
            </a>
            {infoQuery.error ? ` (${getErrorMessage(infoQuery.error)})` : ""}
          </p>
        )}
      </Card>
    );
  } else if (
    isWorkspaceStarting(ws) ||
    (stopped && (start.isPending || restarting))
  ) {
    body = (
      <Card title="Starting your agent Workspace">
        <Loader size="sm" color="gray" className="mx-auto my-2" />
        <p>{workspaceStateLabel(state)}…</p>
      </Card>
    );
  } else if (isWorkspaceStopping(ws)) {
    body = (
      <Card title="Stopping your agent Workspace">
        <Loader size="sm" color="gray" className="mx-auto my-2" />
      </Card>
    );
  } else {
    body = (
      <Card
        title={
          failure ? "The Workspace failed" : "Your agent Workspace is stopped"
        }
        tone={failure ? "error" : "default"}
        icon={failure ? <CircleAlert size={23} /> : undefined}
      >
        {failure?.message && (
          <p className="mb-2 text-red-700">{failure.message}</p>
        )}
        <p>
          Your conversations and files are kept while the Workspace is stopped.
        </p>
        <div className="mt-3 flex justify-center gap-2">
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
              component="a"
              href={getWorkspaceCordiumURL(ws, "logs")}
              target="_blank"
              rel="noreferrer noopener"
            >
              View logs
            </Button>
          )}
        </div>
      </Card>
    );
  }

  return (
    <div className="flex h-[calc(100vh-112px)] min-h-[560px] flex-col gap-3">
      <Header>{actions}</Header>
      {props.agent.state === Agent_State.OUTDATED && <OutdatedBanner />}
      {body}
    </div>
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
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
      <span>
        Your agent environment does not match the Cluster's agent configuration.
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

const AgentEnvironment = () => {
  const queryClient = useQueryClient();
  const [selected, setSelected] = React.useState(readStoredWorkspace);

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
    storeWorkspace(uid);
    setSelected(uid);
  };

  if (agentQuery.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Header />
        <Card title="Loading your agent">
          <Loader size="sm" color="gray" className="mx-auto my-2" />
        </Card>
      </div>
    );
  }

  if (agentQuery.isError) {
    return (
      <div className="flex flex-col gap-3">
        <Header />
        <Card
          title="The agent is not available"
          tone="error"
          icon={<CircleAlert size={23} />}
        >
          <p>{getErrorMessage(agentQuery.error)}</p>
          <Button
            className="mt-3"
            size="xs"
            variant="default"
            leftSection={<RefreshCw size={13} />}
            onClick={() => void agentQuery.refetch()}
          >
            Retry
          </Button>
        </Card>
      </div>
    );
  }

  const agent = agentQuery.data;
  const current =
    agent.workspaces.find((itm) => itm.workspace?.metadata?.uid === selected) ??
    agent.workspaces[0];

  if (agent.state === Agent_State.NOT_INITIALIZED || !current?.workspace) {
    const isNew = agent.state === Agent_State.NOT_INITIALIZED;
    return (
      <div className="flex flex-col gap-3">
        <Header />
        <Card
          title={isNew ? "Set up your agent" : "Repair your agent environment"}
        >
          <p>
            The agent runs inside a dedicated Workspace in your personal{" "}
            <a
              className="font-semibold text-blue-600"
              href={getCordiumURL()}
              target="_blank"
              rel="noreferrer noopener"
            >
              Cordium
            </a>{" "}
            Space. It acts on your behalf with your own Octelium identity and
            permissions. You can also access its Workspaces directly from
            Cordium.
          </p>
          <Button
            className="mt-4"
            size="sm"
            color="dark"
            leftSection={<Sparkles size={14} />}
            loading={initialize.isPending}
            onClick={() => initialize.mutate()}
          >
            {isNew ? "Set up my agent" : "Repair"}
          </Button>
        </Card>
      </div>
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
  <>
    <Meta title="Agent" />
    {devAgentURL ? (
      <div className="flex h-[calc(100vh-112px)] min-h-[560px] flex-col gap-3">
        <Header />
        <AgentReady url={devAgentURL} />
      </div>
    ) : (
      <AgentEnvironment />
    )}
  </>
);

export default Agent;
