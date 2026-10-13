import type { AgentInfo } from "@/apis/consoleagent/protocol";
import { Workspace, Workspace_Status_State } from "@/apis/cordiumv1/cordiumv1";
import { Agent_State } from "@/apis/enterprisev1/enterprisev1";
import { GetOptions } from "@/apis/metav1/metav1";
import { AgentClient } from "@/pages/Agent/client";
import {
  agentKey,
  devAgentURL,
  infoKey,
  readStorage,
  workspaceKey,
  workspaceStorageKey,
} from "@/pages/Agent/environment";
import { getClientAgent, getClientCordium } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";

export type AgentStatus =
  "loading" | "unavailable" | "setup" | "stopped" | "starting" | "ready";

export interface AgentStatusInfo {
  status: AgentStatus;
  url?: string;
  info?: AgentInfo;
}

const transitionalStates = new Set([
  Workspace_Status_State.INIT_REQUEST,
  Workspace_Status_State.INITIALIZING,
  Workspace_Status_State.PULLING_IMAGE,
  Workspace_Status_State.BUILDING_IMAGE,
  Workspace_Status_State.STARTING_RUNTIME,
  Workspace_Status_State.PREPARING,
  Workspace_Status_State.STOPPING_REQUEST,
  Workspace_Status_State.STOPPING,
]);

const isTransitional = (ws?: Workspace) =>
  ws?.status?.state !== undefined && transitionalStates.has(ws.status.state);

const isStopping = (ws?: Workspace) =>
  ws?.status?.state === Workspace_Status_State.STOPPING_REQUEST ||
  ws?.status?.state === Workspace_Status_State.STOPPING;

export const useAgentStatus = (): AgentStatusInfo => {
  const envQuery = useQuery({
    queryKey: agentKey,
    queryFn: async () => (await getClientAgent().getAgent({})).response,
    enabled: !devAgentURL,
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 5 * 60_000,
  });

  const agent = envQuery.data;
  const selected = readStorage(workspaceStorageKey);
  const initial =
    agent && agent.state !== Agent_State.NOT_INITIALIZED
      ? (
          agent.workspaces.find(
            (itm) => itm.workspace?.metadata?.uid === selected,
          ) ?? agent.workspaces[0]
        )?.workspace
      : undefined;
  const uid = initial?.metadata?.uid ?? "";

  const wsQuery = useQuery({
    queryKey: workspaceKey(uid),
    queryFn: async () =>
      (await getClientCordium().getWorkspace(GetOptions.create({ uid })))
        .response,
    enabled: !!initial,
    initialData: initial,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      const ws = query.state.data;
      if (isTransitional(ws)) return 5000;
      if (ws?.status?.state === Workspace_Status_State.RUNNING) {
        return 120_000;
      }
      return false;
    },
  });

  const ws = initial ? (wsQuery.data ?? initial) : undefined;
  const url =
    devAgentURL ??
    (ws?.status?.state === Workspace_Status_State.RUNNING && ws.status.hostname
      ? `https://${ws.status.hostname}`
      : undefined);

  const infoQuery = useQuery({
    queryKey: infoKey(url),
    queryFn: () => new AgentClient(url!).info(AbortSignal.timeout(8000)),
    enabled: !!url,
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 5 * 60_000,
    refetchInterval: (query) =>
      query.state.data || devAgentURL ? false : 15_000,
  });

  if (devAgentURL) {
    if (infoQuery.data) {
      return { status: "ready", url: devAgentURL, info: infoQuery.data };
    }
    return { status: infoQuery.isError ? "unavailable" : "loading" };
  }

  if (envQuery.isPending) return { status: "loading" };
  if (envQuery.isError) return { status: "unavailable" };
  if (!ws) return { status: "setup" };
  if (url && infoQuery.data) {
    return { status: "ready", url, info: infoQuery.data };
  }
  if (url || (isTransitional(ws) && !isStopping(ws))) {
    return { status: "starting" };
  }
  return { status: "stopped" };
};
