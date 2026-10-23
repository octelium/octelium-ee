import {
  StartWorkspaceRequest,
  Workspace,
  Workspace_Status_State,
} from "@/apis/cordiumv1/cordiumv1";
import { Agent_State } from "@/apis/enterprisev1/enterprisev1";
import { GetOptions, ObjectReference } from "@/apis/metav1/metav1";
import {
  agentKey,
  readStorage,
  workspaceKey,
  workspaceStorageKey,
} from "@/pages/Agent/environment";
import { getClientAgent, getClientCordium } from "@/utils/client";
import type { RpcError } from "@protobuf-ts/runtime-rpc";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import * as React from "react";

export const startingStates = new Set([
  Workspace_Status_State.INIT_REQUEST,
  Workspace_Status_State.INITIALIZING,
  Workspace_Status_State.PULLING_IMAGE,
  Workspace_Status_State.BUILDING_IMAGE,
  Workspace_Status_State.STARTING_RUNTIME,
  Workspace_Status_State.PREPARING,
]);

export const stoppingStates = new Set([
  Workspace_Status_State.STOPPING_REQUEST,
  Workspace_Status_State.STOPPING,
]);

export const workspaceStateLabel = (state?: Workspace_Status_State) => {
  switch (state) {
    case Workspace_Status_State.INIT_REQUEST:
      return "Requested";
    case Workspace_Status_State.INITIALIZING:
      return "Initializing";
    case Workspace_Status_State.PULLING_IMAGE:
      return "Pulling the image";
    case Workspace_Status_State.BUILDING_IMAGE:
      return "Building the image";
    case Workspace_Status_State.STARTING_RUNTIME:
      return "Starting the runtime";
    case Workspace_Status_State.PREPARING:
      return "Preparing";
    case Workspace_Status_State.RUNNING:
      return "Running";
    case Workspace_Status_State.STOPPING_REQUEST:
    case Workspace_Status_State.STOPPING:
      return "Stopping";
    case Workspace_Status_State.STOPPED:
      return "Stopped";
    default:
      return "Unknown";
  }
};

export const workspaceName = (ws?: Workspace) =>
  ws?.metadata?.displayName || ws?.metadata?.name || "Workspace";

export const errorMessage = (err: unknown) =>
  (err as RpcError | Error | undefined)?.message ?? String(err);

export const useAgentWorkspace = () => {
  const queryClient = useQueryClient();

  const agentQuery = useQuery({
    queryKey: agentKey,
    queryFn: async () => (await getClientAgent().getAgent({})).response,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const agent = agentQuery.data;
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

  const workspaceQuery = useQuery({
    queryKey: workspaceKey(uid),
    queryFn: async () =>
      (await getClientCordium().getWorkspace(GetOptions.create({ uid })))
        .response,
    enabled: !!initial,
    initialData: initial,
    refetchOnWindowFocus: false,
    refetchInterval: (query) => {
      const state = query.state.data?.status?.state;
      if (state === Workspace_Status_State.RUNNING) return 15000;
      if (state === Workspace_Status_State.STOPPED) return false;
      return 2000;
    },
  });

  const workspace = initial ? (workspaceQuery.data ?? initial) : undefined;
  const state = workspace?.status?.state;

  const start = useMutation({
    mutationFn: async (ws: Workspace) =>
      getClientCordium().startWorkspace(
        StartWorkspaceRequest.create({
          workspaceRef: ObjectReference.create({
            uid: ws.metadata?.uid,
            name: ws.metadata?.name,
          }),
        }),
      ),
    onSuccess: (_data, ws) =>
      queryClient.setQueryData<Workspace>(
        workspaceKey(ws.metadata?.uid ?? ""),
        (current) =>
          current?.status?.state === Workspace_Status_State.STOPPED
            ? {
                ...current,
                status: {
                  ...current.status,
                  state: Workspace_Status_State.INIT_REQUEST,
                },
              }
            : current,
      ),
    onSettled: (_data, _err, ws) =>
      queryClient.invalidateQueries({
        queryKey: workspaceKey(ws.metadata?.uid ?? ""),
      }),
  });

  const initialize = useMutation({
    mutationFn: async () =>
      (await getClientAgent().initializeAgent({})).response,
    onSuccess: (data) => queryClient.setQueryData(agentKey, data),
  });

  const { mutate } = start;
  const startWorkspace = React.useCallback(() => {
    if (workspace) mutate(workspace);
  }, [workspace, mutate]);

  const attempted = !start.isIdle && start.variables?.metadata?.uid === uid;
  const shouldStart =
    !!workspace &&
    state === Workspace_Status_State.STOPPED &&
    !workspace.status?.failure &&
    !attempted;

  React.useEffect(() => {
    if (shouldStart) startWorkspace();
  }, [shouldStart, startWorkspace]);

  const startError =
    start.error && (start.error as RpcError).code !== "ALREADY_EXISTS"
      ? start.error
      : undefined;

  return {
    agentQuery,
    agent,
    workspace,
    state,
    starting:
      (state !== undefined && startingStates.has(state)) ||
      (state === Workspace_Status_State.STOPPED &&
        (start.isPending || shouldStart)),
    stopping: state !== undefined && stoppingStates.has(state),
    running: state === Workspace_Status_State.RUNNING,
    startWorkspace,
    startPending: start.isPending,
    startError,
    initialize,
  };
};
