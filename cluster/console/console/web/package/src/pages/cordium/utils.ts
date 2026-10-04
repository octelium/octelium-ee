import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ResourceStatusTone } from "@/pages/utils/types";
import { match } from "ts-pattern";

export type StateMeta = {
  label: string;
  tone: ResourceStatusTone;
  className: string;
};

const refOf = (
  itemRef: ObjectReference | undefined,
  apiVersion: string,
  kind: string,
): ObjectReference | undefined =>
  itemRef?.name || itemRef?.uid
    ? ObjectReference.create({
        ...itemRef,
        apiVersion: itemRef.apiVersion || apiVersion,
        kind: itemRef.kind || kind,
      })
    : undefined;

export const userRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "core/v1", "User");

export const sessionRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "core/v1", "Session");

export const regionRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "core/v1", "Region");

export const spaceRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "cordium/v1", "Space");

export const templateRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "cordium/v1", "Template");

export const gitProviderRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "cordium/v1", "GitProvider");

export const workspaceRefOf = (itemRef?: ObjectReference) =>
  refOf(itemRef, "cordium/v1", "Workspace");

export const coreRegionRefOf = (name: string) =>
  ObjectReference.create({ apiVersion: "core/v1", kind: "Region", name });

const STARTING_STATES = [
  CordiumP.Workspace_Status_State.INIT_REQUEST,
  CordiumP.Workspace_Status_State.INITIALIZING,
  CordiumP.Workspace_Status_State.PULLING_IMAGE,
  CordiumP.Workspace_Status_State.BUILDING_IMAGE,
  CordiumP.Workspace_Status_State.STARTING_RUNTIME,
  CordiumP.Workspace_Status_State.PREPARING,
];

export const isWorkspaceStarting = (state?: CordiumP.Workspace_Status_State) =>
  state !== undefined && STARTING_STATES.includes(state);

export const isWorkspaceFailed = (item: CordiumP.Workspace): boolean =>
  !!item.status?.run?.failure;

export const getWorkspaceStateMeta = (
  state?: CordiumP.Workspace_Status_State,
): StateMeta =>
  match(state)
    .with(CordiumP.Workspace_Status_State.INIT_REQUEST, () => ({
      label: "Start requested",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.INITIALIZING, () => ({
      label: "Initializing",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.PULLING_IMAGE, () => ({
      label: "Pulling image",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.BUILDING_IMAGE, () => ({
      label: "Building image",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.STARTING_RUNTIME, () => ({
      label: "Starting runtime",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.PREPARING, () => ({
      label: "Preparing",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Workspace_Status_State.RUNNING, () => ({
      label: "Running",
      tone: "success" as const,
      className: "text-emerald-600",
    }))
    .with(CordiumP.Workspace_Status_State.STOPPING_REQUEST, () => ({
      label: "Stop requested",
      tone: "warning" as const,
      className: "text-amber-600",
    }))
    .with(CordiumP.Workspace_Status_State.STOPPING, () => ({
      label: "Stopping",
      tone: "warning" as const,
      className: "text-amber-600",
    }))
    .with(CordiumP.Workspace_Status_State.STOPPED, () => ({
      label: "Stopped",
      tone: "neutral" as const,
      className: "text-slate-500",
    }))
    .otherwise(() => ({
      label: "Unknown",
      tone: "neutral" as const,
      className: "text-slate-500",
    }));

export const getWorkspaceMeta = (item: CordiumP.Workspace): StateMeta =>
  isWorkspaceFailed(item) &&
  item.status?.state === CordiumP.Workspace_Status_State.STOPPED
    ? { label: "Failed", tone: "danger", className: "text-red-600" }
    : getWorkspaceStateMeta(item.status?.state);

export const getFailureLabel = (
  failure?: CordiumP.Workspace_Status_Failure,
): string => {
  const type = failure?.type;
  return match(type?.oneofKind)
    .with("imageBuild", () => "Image build")
    .with("imagePull", () => "Image pull")
    .with("repoClone", () => "Repository clone")
    .with("buildTimeoutExceeded", () => "Build timeout")
    .with("task", () =>
      type?.oneofKind === "task"
        ? `Task ${type.task.name} exited with ${type.task.exitCode}`
        : "Task",
    )
    .with("startupUnknown", () => "Startup")
    .with("startupTimeoutExceeded", () => "Startup timeout")
    .with("loadStorage", () => "Storage load")
    .with("saveStorage", () => "Storage save")
    .with("stoppageTimeoutExceeded", () => "Stop timeout")
    .with("runContainer", () => "Container run")
    .with("healthCheck", () => "Health check")
    .with("additionalRepoClone", () =>
      type?.oneofKind === "additionalRepoClone"
        ? `Repository ${type.additionalRepoClone.name} clone`
        : "Repository clone",
    )
    .with("networkPolicy", () => "Network policy")
    .with("volume", () =>
      type?.oneofKind === "volume" ? `Volume ${type.volume.name}` : "Volume",
    )
    .otherwise(() => "Unknown");
};

export const getStoppingReasonLabel = (
  reason?: CordiumP.Workspace_Status_StoppingReason,
): string =>
  match(reason)
    .with(CordiumP.Workspace_Status_StoppingReason.API, () => "Stopped via API")
    .with(CordiumP.Workspace_Status_StoppingReason.ERROR, () => "Error")
    .with(CordiumP.Workspace_Status_StoppingReason.CLUSTER, () => "Cluster")
    .otherwise(() => "Unset");

export const getSharedPortModeLabel = (
  mode?: CordiumP.Workspace_Status_SharedPort_Mode,
): string =>
  match(mode)
    .with(CordiumP.Workspace_Status_SharedPort_Mode.MEMBERS, () => "Members")
    .with(CordiumP.Workspace_Status_SharedPort_Mode.ALL, () => "Everyone")
    .otherwise(() => "Unset");

export const getSpaceTypeLabel = (type?: CordiumP.Space_Status_Type): string =>
  match(type)
    .with(CordiumP.Space_Status_Type.USER, () => "Personal")
    .with(CordiumP.Space_Status_Type.ORGANIZATION, () => "Organization")
    .otherwise(() => "Unset");

export const getBuildStateMeta = (
  state?: CordiumP.Template_Status_BuildInfo_Build_State,
): StateMeta =>
  match(state)
    .with(CordiumP.Template_Status_BuildInfo_Build_State.RUNNING, () => ({
      label: "Building",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Template_Status_BuildInfo_Build_State.READY, () => ({
      label: "Ready",
      tone: "success" as const,
      className: "text-emerald-600",
    }))
    .with(CordiumP.Template_Status_BuildInfo_Build_State.FAILED, () => ({
      label: "Failed",
      tone: "danger" as const,
      className: "text-red-600",
    }))
    .otherwise(() => ({
      label: "Unknown",
      tone: "neutral" as const,
      className: "text-slate-500",
    }));

export const getLatestBuild = (
  item: CordiumP.Template,
): CordiumP.Template_Status_BuildInfo_Build | undefined =>
  item.status?.buildInfo?.builds.at(0);

export const getTemplateMeta = (item: CordiumP.Template): StateMeta => {
  const latest = getLatestBuild(item);
  if (!latest) {
    return {
      label: "Never built",
      tone: "neutral",
      className: "text-slate-500",
    };
  }

  if (latest.isCanceled) {
    return { label: "Canceled", tone: "warning", className: "text-amber-600" };
  }

  return getBuildStateMeta(latest.state);
};

export const getRoleMeta = (role?: CordiumP.Membership_Spec_Role): StateMeta =>
  match(role)
    .with(CordiumP.Membership_Spec_Role.OWNER, () => ({
      label: "Owner",
      tone: "info" as const,
      className: "text-blue-600",
    }))
    .with(CordiumP.Membership_Spec_Role.ADMIN, () => ({
      label: "Admin",
      tone: "warning" as const,
      className: "text-amber-600",
    }))
    .with(CordiumP.Membership_Spec_Role.USER, () => ({
      label: "User",
      tone: "neutral" as const,
      className: "text-slate-600",
    }))
    .otherwise(() => ({
      label: "Unknown",
      tone: "neutral" as const,
      className: "text-slate-500",
    }));

export const getGitProviderTypeLabel = (item: CordiumP.GitProvider): string =>
  match(item.spec?.type.oneofKind)
    .with("github", () => "GitHub")
    .with("gitlab", () => "GitLab")
    .with("oauth2", () => "OAuth2")
    .otherwise(() => "Unset");

export const getGitProviderConfig = (
  item: CordiumP.GitProvider,
):
  | {
      clientID: string;
      scopes: string[];
      clientSecret?: string;
      authURL?: string;
      tokenURL?: string;
    }
  | undefined => {
  const type = item.spec?.type;
  const secretOf = (arg?: {
    type:
      | { oneofKind: "fromSecret"; fromSecret: string }
      | { oneofKind: undefined };
  }) =>
    arg?.type.oneofKind === "fromSecret" ? arg.type.fromSecret : undefined;

  switch (type?.oneofKind) {
    case "github":
      return {
        clientID: type.github.clientID,
        scopes: type.github.scopes,
        clientSecret: secretOf(type.github.clientSecret),
      };
    case "gitlab":
      return {
        clientID: type.gitlab.clientID,
        scopes: type.gitlab.scopes,
        clientSecret: secretOf(type.gitlab.clientSecret),
      };
    case "oauth2":
      return {
        clientID: type.oauth2.clientID,
        scopes: type.oauth2.scopes,
        clientSecret: secretOf(type.oauth2.clientSecret),
        authURL: type.oauth2.authURL,
        tokenURL: type.oauth2.tokenURL,
      };
    default:
      return undefined;
  }
};

export const getUserSecretTypeLabel = (
  type?: CordiumP.UserSecret_Spec_Type,
): string =>
  match(type)
    .with(CordiumP.UserSecret_Spec_Type.SSH_KEY, () => "SSH key")
    .otherwise(() => "Default");

export const getImageSource = (
  image?: CordiumP.Workspace_Spec_Image,
): { label: string; value?: string } | undefined => {
  const type = image?.type;
  switch (type?.oneofKind) {
    case "registry":
      return { label: "Registry", value: type.registry.url };
    case "git":
      return { label: "Git", value: type.git.url };
    case "dockerfile":
      return {
        label: "Dockerfile",
        value:
          type.dockerfile.type.oneofKind === "url"
            ? type.dockerfile.type.url
            : undefined,
      };
    case "repository":
      return {
        label:
          type.repository.type.oneofKind === "devcontainer"
            ? "Repository devcontainer"
            : "Repository Dockerfile",
      };
    default:
      return undefined;
  }
};

export const formatMillicores = (millicores: number): string =>
  millicores >= 1000 || millicores === 0
    ? `${Math.round((millicores / 1000) * 100) / 100} vCPU`
    : `${millicores}m vCPU`;

export const formatMegabytes = (megabytes: number): string =>
  megabytes >= 1024
    ? `${Math.round((megabytes / 1024) * 10) / 10} GiB`
    : `${megabytes} MiB`;

export const getLimitItems = (
  limit?: CordiumP.Workspace_Spec_Limit,
): { label: string; value: string }[] => [
  ...(limit?.cpu?.millicores
    ? [{ label: "CPU", value: formatMillicores(limit.cpu.millicores) }]
    : []),
  ...(limit?.memory?.megabytes
    ? [{ label: "Memory", value: formatMegabytes(limit.memory.megabytes) }]
    : []),
  ...(limit?.storage?.megabytes
    ? [{ label: "Storage", value: formatMegabytes(limit.storage.megabytes) }]
    : []),
];
