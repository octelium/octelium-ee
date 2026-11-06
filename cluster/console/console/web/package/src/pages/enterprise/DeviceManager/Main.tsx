import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import {
  durationSeconds,
  formatDuration,
  HeroStat,
  Meter,
  MutedValue,
  OverviewHero,
  OverviewNotice,
  OverviewTone,
  ResourceActionButton,
  toDate,
  TONE_TEXT,
  TonePill,
} from "@/components/ResourceLayout/overview";
import { ResourceListLabel } from "@/components/ResourceList";
import TimeAgo, { useNow } from "@/components/TimeAgo";
import { twMerge } from "@/lib/utils";
import { ResourceInfoMainItem, ResourceMainInfo } from "@/pages/utils/types";
import { getClientEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import {
  Activity,
  AlertTriangle,
  Ban,
  Filter,
  History,
  LaptopMinimal,
  Link2,
  ListOrdered,
  Loader2,
  LucideIcon,
  MonitorSmartphone,
  Plug,
  RefreshCcw,
} from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { devicesURL, getManagerPresentation } from "./List";
import { ManagerMark } from "./providers";

const DEFAULT_STALE_AFTER = 3600;

type Item = ResourceInfoMainItem;

export default (_props: { item: EnterpriseP.DeviceManager }) => <></>;

const secret = (ref?: EnterpriseP.DeviceManager_Spec_SecretRef) =>
  ref?.type.oneofKind === "fromSecret" && ref.type.fromSecret ? (
    <ResourceListLabel
      itemRef={ObjectReference.create({
        apiVersion: "enterprise/v1",
        kind: "Secret",
        name: ref.type.fromSecret,
      })}
    />
  ) : (
    <MutedValue>Not set</MutedValue>
  );

const copied = (value?: string, fallback = "Not set") =>
  value ? <CopyText value={value} /> : <MutedValue>{fallback}</MutedValue>;

const code = (value: string) => (
  <code className="block rounded-md border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-xs leading-5 text-slate-700 [overflow-wrap:anywhere]">
    {value}
  </code>
);

const chips = (values: string[]) => (
  <div className="flex flex-wrap gap-1">
    {values.map((value) => (
      <Label key={value}>{value}</Label>
    ))}
  </div>
);

const toggle = (enabled: boolean, on = "Enabled", off = "Disabled") => (
  <TonePill tone={enabled ? "success" : "neutral"}>
    {enabled ? on : off}
  </TonePill>
);

const regionLabel = (
  region: EnterpriseP.DeviceManager_Spec_CrowdStrike_Region,
) =>
  EnterpriseP.DeviceManager_Spec_CrowdStrike_Region[region]
    .toLowerCase()
    .replace(/_/g, "-");

const cloudLabel = (
  cloud: EnterpriseP.DeviceManager_Spec_MicrosoftIntune_Cloud,
) => {
  const Cloud = EnterpriseP.DeviceManager_Spec_MicrosoftIntune_Cloud;
  if (cloud === Cloud.US_GOV) return "US Government";
  if (cloud === Cloud.CHINA) return "China (21Vianet)";
  return "Public";
};

const connectionItems = (item: EnterpriseP.DeviceManager): Item[] => {
  const type = item.spec?.type;
  const group = "Connection";

  switch (type?.oneofKind) {
    case "crowdStrike":
      return [
        {
          label: "Falcon cloud",
          group,
          value: type.crowdStrike.region ? (
            <span className="font-semibold">
              {regionLabel(type.crowdStrike.region)}
            </span>
          ) : (
            <MutedValue>Discovered automatically</MutedValue>
          ),
        },
        ...(type.crowdStrike.memberCID
          ? [
              {
                label: "Member CID",
                group,
                value: copied(type.crowdStrike.memberCID),
              },
            ]
          : []),
        { label: "Client ID", group, value: copied(type.crowdStrike.clientID) },
        {
          label: "Client secret",
          group,
          value: secret(type.crowdStrike.clientSecret),
        },
        {
          label: "Zero Trust Assessment",
          group,
          value: toggle(
            !type.crowdStrike.disableZeroTrustAssessment,
            "Collected",
            "Not collected",
          ),
        },
      ];
    case "sentinelOne":
      return [
        {
          label: "Management URL",
          group,
          span: "full",
          value: copied(type.sentinelOne.managementURL),
        },
        { label: "API token", group, value: secret(type.sentinelOne.apiToken) },
      ];
    case "microsoftIntune":
      return [
        {
          label: "Cloud",
          group,
          value: (
            <span className="font-semibold">
              {cloudLabel(type.microsoftIntune.cloud)}
            </span>
          ),
        },
        {
          label: "Tenant ID",
          group,
          value: copied(type.microsoftIntune.tenantID),
        },
        {
          label: "Client ID",
          group,
          value: copied(type.microsoftIntune.clientID),
        },
        {
          label: "Client secret",
          group,
          value: secret(type.microsoftIntune.clientSecret),
        },
      ];
    case "jamf":
      return [
        {
          label: "Server URL",
          group,
          span: "full",
          value: copied(type.jamf.baseURL),
        },
        { label: "Client ID", group, value: copied(type.jamf.clientID) },
        {
          label: "Client secret",
          group,
          value: secret(type.jamf.clientSecret),
        },
      ];
    case "onePassword":
      return [
        {
          label: "API URL",
          group,
          span: "full",
          value: copied(type.onePassword.baseURL, "Default public API"),
        },
        { label: "API token", group, value: secret(type.onePassword.apiToken) },
      ];
    case "fleetDM":
      return [
        {
          label: "Server URL",
          group,
          span: "full",
          value: copied(type.fleetDM.baseURL),
        },
        { label: "API token", group, value: secret(type.fleetDM.apiToken) },
      ];
    case "huntress":
      return [
        {
          label: "API URL",
          group,
          span: "full",
          value: copied(type.huntress.baseURL, "Default public API"),
        },
        { label: "API key", group, value: copied(type.huntress.apiKey) },
        { label: "API secret", group, value: secret(type.huntress.apiSecret) },
      ];
    case "iru":
      return [
        {
          label: "API URL",
          group,
          span: "full",
          value: copied(type.iru.baseURL),
        },
        { label: "API token", group, value: secret(type.iru.apiToken) },
      ];
    default:
      return [];
  }
};

const scopeItems = (item: EnterpriseP.DeviceManager): Item[] => {
  const type = item.spec?.type;
  const group = "Inventory scope";
  const items: Item[] = [];
  const full = (label: string, value: React.ReactNode) =>
    items.push({ label, group, span: "full", value });

  if (type?.oneofKind === "crowdStrike" && type.crowdStrike.hostFilter) {
    full("Host filter", code(type.crowdStrike.hostFilter));
  }
  if (type?.oneofKind === "sentinelOne") {
    const { siteIDs, accountIDs, agentFilters } = type.sentinelOne;
    const filters = Object.entries(agentFilters);
    if (siteIDs.length) full("Sites", chips(siteIDs));
    if (accountIDs.length) full("Accounts", chips(accountIDs));
    if (filters.length) {
      full(
        "Agent filters",
        chips(filters.map(([key, value]) => `${key}: ${value}`)),
      );
    }
  }
  if (type?.oneofKind === "microsoftIntune") {
    if (type.microsoftIntune.filter) {
      full("Filter", code(type.microsoftIntune.filter));
    }
    items.push(
      {
        label: "Grace period",
        group,
        value: toggle(
          type.microsoftIntune.isGracePeriodCompliant,
          "Treated as compliant",
          "Reported as a grace period",
        ),
      },
      {
        label: "Configuration Manager",
        group,
        value: toggle(
          type.microsoftIntune.isConfigManagerCompliant,
          "Treated as compliant",
          "Evaluated by Intune",
        ),
      },
    );
  }
  if (type?.oneofKind === "jamf") {
    if (type.jamf.filter) full("Filter", code(type.jamf.filter));
    full(
      "Compliant groups",
      type.jamf.compliantGroups.length ? (
        chips(type.jamf.compliantGroups)
      ) : (
        <MutedValue>Compliance is not reported</MutedValue>
      ),
    );
  }
  if (type?.oneofKind === "fleetDM") {
    items.push({
      label: "Team",
      group,
      value:
        type.fleetDM.teamID === undefined ? (
          <MutedValue>Every team</MutedValue>
        ) : type.fleetDM.teamID === 0 ? (
          <span className="font-semibold">Hosts without a team</span>
        ) : (
          <span className="font-semibold tabular-nums">
            {type.fleetDM.teamID}
          </span>
        ),
    });
  }
  if (type?.oneofKind === "huntress") {
    full(
      "Organizations",
      type.huntress.organizationIDs.length ? (
        chips(type.huntress.organizationIDs.map(String))
      ) : (
        <MutedValue>Every organization of the account</MutedValue>
      ),
    );
  }

  return items.length
    ? items
    : [
        {
          label: "Scope",
          group,
          span: "full",
          value: <MutedValue>The whole provider inventory</MutedValue>,
        },
      ];
};

const conditionSummary = (condition?: CoreP.Condition) => {
  const type = condition?.type;
  switch (type?.oneofKind) {
    case "match":
      return code(type.match);
    case "not":
      return code(`!(${type.not})`);
    case "all":
      return `All of ${type.all.of.length} conditions`;
    case "any":
      return `Any of ${type.any.of.length} conditions`;
    case "none":
      return `None of ${type.none.of.length} conditions`;
    case "opa":
      return "Evaluated by an OPA policy";
    default:
      return "Every Device";
  }
};

const useSelectionOrder = (name?: string) => {
  const query = useQuery({
    queryKey: ["enterprise", "clusterconfig"],
    queryFn: async () => getClientEnterprise().getClusterConfig({}),
  });
  const order = query.data?.response.spec?.deviceManagers;
  const index = name && order ? order.indexOf(name) : -1;

  return {
    isKnown: !!order,
    total: order?.length ?? 0,
    index,
    isExcluded: !!order && order.length > 0 && index < 0,
  };
};

const SelectionOrder = (props: { name?: string }) => {
  const order = useSelectionOrder(props.name);
  if (!order.isKnown) return <MutedValue>Unavailable</MutedValue>;
  if (order.isExcluded) return <TonePill tone="warning">Not listed</TonePill>;
  if (order.total === 0) return <span className="font-semibold">By name</span>;
  return (
    <span className="font-semibold tabular-nums">
      {order.index + 1} of {order.total}
    </span>
  );
};

const LinkingStat = (props: {
  label: string;
  hint: string;
  count: number;
  tone: OverviewTone;
  icon: LucideIcon;
  to?: string;
}) => {
  const Icon = props.icon;
  const content = (
    <>
      <span className="flex min-w-0 items-center gap-1.5 text-xs font-medium text-slate-500">
        <Icon size={12} strokeWidth={2.3} className="shrink-0" />
        <span className="truncate">{props.label}</span>
      </span>
      <span
        className={twMerge(
          "text-base font-bold tabular-nums leading-6",
          props.count > 0 ? TONE_TEXT[props.tone] : "text-slate-500",
        )}
      >
        {props.count.toLocaleString()}
      </span>
    </>
  );
  const style =
    "flex min-w-0 flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2";

  return props.to && props.count > 0 ? (
    <Link
      to={props.to}
      title={props.hint}
      className={twMerge(
        style,
        "outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50 focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {content}
    </Link>
  ) : (
    <div title={props.hint} className={style}>
      {content}
    </div>
  );
};

const LinkingStats = (props: { item: EnterpriseP.DeviceManager }) => {
  const name = props.item.metadata?.name;
  const p = getManagerPresentation(props.item);

  return (
    <div className="grid grid-cols-2 gap-2 @2xl:grid-cols-3">
      <LinkingStat
        label="Linked"
        hint="The Devices that are bound to an inventory entry."
        count={p.linked}
        tone="success"
        icon={Link2}
        to={devicesURL({ bindingState: "ACCEPTED" }, name)}
      />
      {p.issues.map((issue) => (
        <LinkingStat
          key={issue.key}
          label={issue.label}
          hint={issue.hint}
          count={issue.count}
          tone={issue.tone}
          icon={issue.icon}
          to={issue.filters ? devicesURL(issue.filters, name) : undefined}
        />
      ))}
    </div>
  );
};

const ViewDevicesButton = (props: { item: EnterpriseP.DeviceManager }) => {
  const navigate = useNavigate();
  return (
    <ResourceActionButton
      icon={LaptopMinimal}
      onClick={() => navigate(devicesURL({}, props.item.metadata?.name))}
    >
      View Devices
    </ResourceActionButton>
  );
};

const ManagerHero = (props: { item: EnterpriseP.DeviceManager }) => {
  const { item } = props;
  const p = getManagerPresentation(item);
  const now = useNow();
  const order = useSelectionOrder(item.metadata?.name);
  const State = EnterpriseP.DeviceManager_Status_State;
  const state = item.status?.state;
  const polling = item.spec?.polling;

  const lastSuccess = toDate(p.collection?.lastSuccessAt);
  const staleAfter =
    durationSeconds(polling?.staleAfter) || DEFAULT_STALE_AFTER;
  const isStale =
    !!lastSuccess && now - lastSuccess.getTime() > staleAfter * 1000;

  const notices = [
    state === State.ERROR && (
      <OverviewNotice
        key="error"
        tone="danger"
        icon={AlertTriangle}
        title="The last collection failed."
      >
        {p.collection?.lastError ||
          "The provider could not be reached or the DeviceManager could not be set up."}
      </OverviewNotice>
    ),
    state === State.DEGRADED && (
      <OverviewNotice
        key="degraded"
        tone="warning"
        icon={AlertTriangle}
        title="The last collection is incomplete."
      >
        {p.collection?.lastError ||
          "Some of the information could not be collected."}
      </OverviewNotice>
    ),
    p.isPollingDisabled && (
      <OverviewNotice
        key="disabled"
        tone="neutral"
        icon={Ban}
        title="Polling is disabled."
      >
        The inventory is not collected until polling is enabled again.
      </OverviewNotice>
    ),
    isStale && (
      <OverviewNotice
        key="stale"
        tone="warning"
        icon={History}
        title="The inventory is stale."
      >
        It was last collected <TimeAgo rfc3339={p.collection?.lastSuccessAt} />{" "}
        and is no longer used to link Devices or refresh their posture.
      </OverviewNotice>
    ),
    order.isExcluded && (
      <OverviewNotice
        key="excluded"
        tone="warning"
        icon={ListOrdered}
        title="This manager is not selected for any Device."
      >
        The enterprise{" "}
        <Link
          to="/enterprise/clusterconfig"
          className="font-semibold underline"
        >
          ClusterConfig
        </Link>{" "}
        lists other managers, and only the listed ones bind Devices.
      </OverviewNotice>
    ),
  ].filter(Boolean);

  return (
    <OverviewHero
      tone={isStale && p.state.tone === "success" ? "warning" : p.state.tone}
      icon={
        state === State.LOADING ? (
          <Loader2 size={18} className="motion-safe:animate-spin" />
        ) : (
          <ManagerMark kind={p.kind} size={24} />
        )
      }
      title={p.type}
      detail={
        <>
          {p.state.label}
          {p.collection?.lastSuccessAt && (
            <>
              {" · collected "}
              <TimeAgo rfc3339={p.collection.lastSuccessAt} />
            </>
          )}
          {!p.isPollingDisabled &&
            ` · every ${formatDuration(polling?.interval, "5m")}`}
        </>
      }
      meter={
        p.coverage !== undefined ? (
          <Meter
            value={p.coverage}
            label="Inventory linked to Devices"
            caption={`${p.linked.toLocaleString()} of ${p.managed.toLocaleString()} managed devices are linked to a Device`}
          />
        ) : undefined
      }
      aside={
        p.linking?.lastSweepAt ? (
          <HeroStat
            label="Linking"
            tone={p.issueCount > 0 ? "warning" : "success"}
            value={
              p.issueCount > 0
                ? `${p.issueCount.toLocaleString()} ${p.issueCount === 1 ? "issue" : "issues"}`
                : "No issues"
            }
          />
        ) : undefined
      }
    >
      {notices.length > 0 ? notices : undefined}
    </OverviewHero>
  );
};

export const MainInfo = (props: {
  item: EnterpriseP.DeviceManager;
}): ResourceMainInfo => {
  const { item } = props;
  const p = getManagerPresentation(item);
  const polling = item.spec?.polling;
  const linking = item.spec?.linking;

  return {
    header: <ManagerHero item={item} />,
    actions: <ViewDevicesButton item={item} />,
    groupOrder: [
      { title: "Connection", icon: Plug },
      { title: "Inventory scope", icon: Filter },
      { title: "Collection", icon: RefreshCcw },
      {
        title: "Device linking",
        icon: Link2,
        description:
          "How the Cluster's Devices are bound to the entries of the inventory.",
      },
    ],
    items: [
      {
        label: "State",
        primary: true,
        icon: Activity,
        value: <TonePill tone={p.state.tone}>{p.state.label}</TonePill>,
      },
      {
        label: "Managed devices",
        primary: true,
        icon: MonitorSmartphone,
        value: p.collection?.lastSuccessAt ? (
          <span className="font-semibold tabular-nums">
            {p.managed.toLocaleString()}
          </span>
        ) : (
          <MutedValue>Not collected yet</MutedValue>
        ),
      },
      {
        label: "Linked Devices",
        primary: true,
        icon: Link2,
        value: p.linking?.lastSweepAt ? (
          <span className="font-semibold tabular-nums">
            {p.linked.toLocaleString()}
          </span>
        ) : (
          <MutedValue>Not linked yet</MutedValue>
        ),
        hint:
          p.coverage !== undefined
            ? `${Math.round(p.coverage)}% of the inventory`
            : undefined,
      },
      {
        label: "Selection order",
        primary: true,
        icon: ListOrdered,
        value: <SelectionOrder name={item.metadata?.name} />,
        hint: "Set by the enterprise ClusterConfig",
      },

      ...connectionItems(item),
      ...scopeItems(item),

      {
        label: "Interval",
        group: "Collection",
        value: formatDuration(polling?.interval, "5m (default)"),
      },
      {
        label: "Timeout",
        group: "Collection",
        value: formatDuration(polling?.timeout, "2m (default)"),
      },
      {
        label: "Stale after",
        group: "Collection",
        value: formatDuration(polling?.staleAfter, "1h (default)"),
      },
      {
        label: "Maximum observation age",
        group: "Collection",
        value: formatDuration(polling?.maxObservationAge, "7d (default)"),
      },
      {
        label: "Last attempt",
        group: "Collection",
        value: p.collection?.lastAttemptAt ? (
          <TimeAgo rfc3339={p.collection.lastAttemptAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },
      {
        label: "Last success",
        group: "Collection",
        value: p.collection?.lastSuccessAt ? (
          <TimeAgo rfc3339={p.collection.lastSuccessAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },

      {
        label: "Devices",
        group: "Device linking",
        layout: "stacked",
        value: <LinkingStats item={item} />,
      },
      {
        label: "Last sweep",
        group: "Device linking",
        value: p.linking?.lastSweepAt ? (
          <TimeAgo rfc3339={p.linking.lastSweepAt} />
        ) : (
          <MutedValue>Never</MutedValue>
        ),
      },
      {
        label: "Applies to",
        group: "Device linking",
        value: conditionSummary(item.spec?.condition),
      },
      {
        label: "Owner match",
        group: "Device linking",
        value: p.provider?.reportsOwners ? (
          toggle(!!linking?.requireOwnerMatch, "Required", "Not required")
        ) : (
          <MutedValue>The provider reports no owners</MutedValue>
        ),
      },
      {
        label: "Verification",
        group: "Device linking",
        value: p.provider?.publishesProbes ? (
          linking?.verificationInterval ? (
            <>Every {formatDuration(linking.verificationInterval)}</>
          ) : (
            <MutedValue>Not periodic</MutedValue>
          )
        ) : (
          <MutedValue>The provider publishes no Probes</MutedValue>
        ),
      },
    ],
  };
};
