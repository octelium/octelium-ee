import {
  DeviceManager,
  DeviceManager_Status_State,
} from "@/apis/enterprisev1/enterprisev1";
import { GetDeviceManagerSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import {
  ResourceListLabel,
  ResourceListLabelWrap,
} from "@/components/ResourceList";
import type { OverviewTone } from "@/components/ResourceLayout/overview";
import {
  SummaryItemCount,
  SummaryItemCountWrap,
  SummaryNoItems,
} from "@/components/Summary";
import TimeAgo from "@/components/TimeAgo";
import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  CircleX,
  Link2,
  Loader2,
  LucideIcon,
  MonitorSmartphone,
  RefreshCcw,
  ShieldAlert,
  ShieldQuestion,
  Swords,
  UserX,
} from "lucide-react";
import { match } from "ts-pattern";
import {
  getProvider,
  ManagerKind,
  ManagerMark,
  managerIcon,
  PROVIDERS,
} from "./providers";

const DEVICES = "/core/devices";

export const getType = (item: DeviceManager): string =>
  getProvider(item.spec?.type?.oneofKind)?.label ?? "Not configured";

export const getStateMeta = (state?: DeviceManager_Status_State) =>
  match(state)
    .with(DeviceManager_Status_State.OK, () => ({
      label: "Healthy",
      tone: "success" as OverviewTone,
    }))
    .with(DeviceManager_Status_State.LOADING, () => ({
      label: "Loading",
      tone: "info" as OverviewTone,
    }))
    .with(DeviceManager_Status_State.DEGRADED, () => ({
      label: "Degraded",
      tone: "warning" as OverviewTone,
    }))
    .with(DeviceManager_Status_State.ERROR, () => ({
      label: "Error",
      tone: "danger" as OverviewTone,
    }))
    .with(DeviceManager_Status_State.DISABLED, () => ({
      label: "Polling disabled",
      tone: "neutral" as OverviewTone,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "neutral" as OverviewTone }));

export const devicesURL = (
  filters: Record<string, string> = {},
  manager?: string,
) => {
  const params = new URLSearchParams(filters);
  if (manager) params.set("deviceManagerRef.name", manager);
  const search = params.toString();
  return search ? `${DEVICES}?${search}` : DEVICES;
};

export interface LinkingIssue {
  key: string;
  label: string;
  hint: string;
  count: number;
  tone: OverviewTone;
  icon: LucideIcon;
  filters?: Record<string, string>;
}

export const getLinkingIssues = (item: DeviceManager): LinkingIssue[] => {
  const linking = item.status?.linking;
  return [
    {
      key: "ambiguous",
      label: "Ambiguous",
      hint: "More than one inventory entry matches the Device.",
      count: linking?.ambiguous ?? 0,
      tone: "warning",
      icon: ShieldQuestion,
      filters: { bindingState: "AMBIGUOUS" },
    },
    {
      key: "conflicts",
      label: "Conflicts",
      hint: "The inventory entry is already bound to another Device.",
      count: linking?.conflicts ?? 0,
      tone: "warning",
      icon: Swords,
      filters: { bindingState: "CONFLICT" },
    },
    {
      key: "suspended",
      label: "Suspended",
      hint: "The Binding is not valid, so its posture is not used.",
      count: linking?.suspended ?? 0,
      tone: "danger",
      icon: ShieldAlert,
      filters: { hasInvalidBinding: "true" },
    },
    {
      key: "ownerMismatches",
      label: "Owner mismatches",
      hint: "The provider reports another owner for the entry.",
      count: linking?.ownerMismatches ?? 0,
      tone: "danger",
      icon: UserX,
      filters: { bindingState: "OWNER_MISMATCH" },
    },
    {
      key: "failedUpdates",
      label: "Failed updates",
      hint: "The Devices could not be updated during the last sweep.",
      count: linking?.failedUpdates ?? 0,
      tone: "danger",
      icon: RefreshCcw,
    },
  ];
};

export const getManagerPresentation = (item: DeviceManager) => {
  const kind = item.spec?.type?.oneofKind;
  const managed = item.status?.collection?.managedDevices ?? 0;
  const linked = item.status?.linking?.linkedDevices ?? 0;
  const issues = getLinkingIssues(item);

  return {
    kind,
    provider: getProvider(kind),
    type: getType(item),
    state: getStateMeta(item.status?.state),
    collection: item.status?.collection,
    linking: item.status?.linking,
    isPollingDisabled: !!item.spec?.polling?.isDisabled,
    managed,
    linked,
    coverage: managed > 0 ? Math.min(100, (linked / managed) * 100) : undefined,
    issues,
    issueCount: issues.reduce((total, issue) => total + issue.count, 0),
  };
};

export const LabelComponent = (props: { item: DeviceManager }) => {
  const p = getManagerPresentation(props.item);
  const state = props.item.status?.state;

  return (
    <ResourceListLabelWrap>
      <ResourceListLabel label="Provider">
        <ManagerMark kind={p.kind} />
        {p.type}
      </ResourceListLabel>
      <ResourceListLabel label="State">
        {state === DeviceManager_Status_State.LOADING ? (
          <Loader2
            size={11}
            className="motion-safe:animate-spin text-blue-500"
          />
        ) : state === DeviceManager_Status_State.OK ? (
          <CheckCircle2 size={11} className="text-emerald-500" />
        ) : state === DeviceManager_Status_State.DEGRADED ? (
          <AlertTriangle size={11} className="text-amber-500" />
        ) : state === DeviceManager_Status_State.ERROR ? (
          <CircleX size={11} className="text-red-500" />
        ) : null}
        {p.state.label}
      </ResourceListLabel>
      {!!p.collection?.lastSuccessAt && (
        <ResourceListLabel label="Managed">
          {p.managed.toLocaleString()}
        </ResourceListLabel>
      )}
      {!!p.linking?.lastSweepAt && (
        <ResourceListLabel label="Linked">
          {p.linked.toLocaleString()}
        </ResourceListLabel>
      )}
      {p.issueCount > 0 && (
        <ResourceListLabel label="Linking issues">
          <AlertTriangle size={11} className="text-amber-500" />
          {p.issueCount.toLocaleString()}
        </ResourceListLabel>
      )}
      {!!p.collection?.lastSuccessAt && (
        <ResourceListLabel label="Last collected">
          <TimeAgo rfc3339={p.collection.lastSuccessAt} />
        </ResourceListLabel>
      )}
    </ResourceListLabelWrap>
  );
};

const count = (resp: GetDeviceManagerSummaryResponse, kind: ManagerKind) =>
  resp[getProvider(kind)!.summaryField] as number;

const DoSummary = ({ resp }: { resp: GetDeviceManagerSummaryResponse }) => (
  <SummaryItemCountWrap>
    <SummaryItemCount count={resp.totalNumber} to="/enterprise/devicemanagers">
      Total
    </SummaryItemCount>
    {PROVIDERS.map((provider) => (
      <SummaryItemCount
        key={provider.kind}
        count={count(resp, provider.kind)}
        to={`/enterprise/devicemanagers?type=${provider.statusType}`}
        icon={managerIcon(provider.kind)}
      >
        {provider.label}
      </SummaryItemCount>
    ))}
    <SummaryItemCount
      count={resp.totalOK}
      to="/enterprise/devicemanagers?state=OK"
      icon={CheckCircle2}
    >
      Healthy
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalLoading}
      to="/enterprise/devicemanagers?state=LOADING"
      icon={Loader2}
    >
      Loading
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalDegraded}
      to="/enterprise/devicemanagers?state=DEGRADED"
      icon={AlertTriangle}
    >
      Degraded
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalError}
      to="/enterprise/devicemanagers?state=ERROR"
      icon={CircleX}
    >
      Error
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalPollingDisabled}
      to="/enterprise/devicemanagers?isPollingDisabled=true"
      icon={Ban}
    >
      Polling disabled
    </SummaryItemCount>
    <SummaryItemCount count={resp.totalManagedDevices} icon={MonitorSmartphone}>
      Managed devices
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalLinkedDevices}
      to={devicesURL({ bindingState: "ACCEPTED" })}
      icon={Link2}
    >
      Linked devices
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalAmbiguous}
      to={devicesURL({ bindingState: "AMBIGUOUS" })}
      icon={ShieldQuestion}
    >
      Ambiguous
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalConflicts}
      to={devicesURL({ bindingState: "CONFLICT" })}
      icon={Swords}
    >
      Conflicts
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalSuspended}
      to={devicesURL({ hasInvalidBinding: "true" })}
      icon={ShieldAlert}
    >
      Suspended
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalOwnerMismatches}
      to={devicesURL({ bindingState: "OWNER_MISMATCH" })}
      icon={UserX}
    >
      Owner mismatches
    </SummaryItemCount>
    <SummaryItemCount
      count={resp.totalFailedUpdates}
      to="/enterprise/devicemanagers?hasLinkingIssues=true"
      icon={RefreshCcw}
    >
      Failed updates
    </SummaryItemCount>
  </SummaryItemCountWrap>
);

export const Summary = ({ showNoItems }: { showNoItems?: boolean }) => {
  const query = useQuery({
    queryKey: ["visibility", "enterprise", "summary", "DeviceManager"],
    queryFn: async () =>
      (await getClientVisibilityEnterprise().getDeviceManagerSummary({}))
        .response,
  });
  if (!query.data) return null;
  return query.data.totalNumber > 0 ? (
    <DoSummary resp={query.data} />
  ) : showNoItems ? (
    <SummaryNoItems />
  ) : null;
};
