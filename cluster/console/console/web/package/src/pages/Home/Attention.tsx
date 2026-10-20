import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  ArrowUpRight,
  BookKey,
  CalendarClock,
  CalendarX2,
  CheckCircle2,
  Crown,
  Folder,
  Hourglass,
  Inbox,
  LaptopMinimal,
  LockKeyhole,
  LucideIcon,
  ScrollText,
  ShieldAlert,
  ShieldCheck,
  Terminal,
  TriangleAlert,
} from "lucide-react";
import { Link } from "react-router-dom";
import { twMerge } from "@/lib/utils";
import { Panel } from "@/components/Dashboard/components";
import { usePageConditions } from "@/components/PageHeader/context";
import { StatusBadge, SubsystemGrid } from "@/components/Dashboard/health";
import { GetClusterHealthResponse_Status } from "@/apis/visibilityv1/visibilityv1";
import {
  useClusterSummary,
  useComponentSummary,
  useClusterHealth,
} from "@/components/Dashboard/queries";
import { compact, exact } from "@/components/Dashboard/utils";

type Severity = "critical" | "warning" | "info";

type Item = {
  id: string;
  count: number;
  label: string;
  scope: string;
  severity: Severity;
  to: string;
  icon: LucideIcon;
};

const SEVERITY_RANK: Record<Severity, number> = {
  critical: 0,
  warning: 1,
  info: 2,
};

const SEVERITY_STYLE: Record<Severity, string> = {
  critical: "border-red-200 bg-red-50/60 hover:border-red-300",
  warning: "border-amber-200 bg-amber-50/60 hover:border-amber-300",
  info: "border-blue-200 bg-blue-50/60 hover:border-blue-300",
};

const SEVERITY_TEXT: Record<Severity, string> = {
  critical: "text-red-700",
  warning: "text-amber-700",
  info: "text-blue-700",
};

const Card = (props: { item: Item }) => {
  const { item } = props;
  const Icon = item.icon;

  return (
    <Link
      to={item.to}
      className={twMerge(
        "group relative flex min-w-0 flex-col gap-2 rounded-xl border px-3.5 py-3 outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
        SEVERITY_STYLE[item.severity],
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <Icon
          size={14}
          strokeWidth={2.3}
          className={twMerge("shrink-0", SEVERITY_TEXT[item.severity])}
        />
        <span className="rounded-full border border-slate-200 bg-white px-1.5 py-px text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
          {item.scope}
        </span>
      </div>

      <span
        className={twMerge(
          "text-2xl font-bold leading-none tabular-nums",
          SEVERITY_TEXT[item.severity],
        )}
        title={exact(item.count)}
      >
        {compact(item.count)}
      </span>

      <span className="text-micro font-semibold leading-4 text-slate-600">
        {item.label}
      </span>

      <ArrowUpRight
        size={12}
        strokeWidth={2.5}
        aria-hidden="true"
        className="absolute right-3 bottom-3 text-slate-400 opacity-0 transition-opacity duration-150 group-hover:opacity-100"
      />
    </Link>
  );
};

const AllClear = () => (
  <div className="flex min-h-24 flex-col items-center justify-center gap-2 rounded-xl border border-dashed border-emerald-200 bg-emerald-50/50 px-6 py-6 text-center">
    <span className="flex h-9 w-9 items-center justify-center rounded-xl border border-emerald-200 bg-white text-emerald-600">
      <CheckCircle2 size={17} strokeWidth={2.2} />
    </span>
    <span className="text-body font-semibold text-slate-800">
      Nothing needs your attention
    </span>
    <span className="text-micro font-normal text-slate-500">
      No pending approvals, failing integrations, or expiring certificates.
    </span>
  </div>
);

const Attention = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const cluster = useClusterSummary(
    "total",
    periodMinutes,
    QUERY_PRIORITY.critical,
  );
  const componentLogs = useComponentSummary(periodMinutes, QUERY_PRIORITY.high);
  const health = useClusterHealth(periodMinutes, QUERY_PRIORITY.high);
  const unhealthySubsystems = (health.data?.subsystems ?? []).filter(
    (item) =>
      item.status === GetClusterHealthResponse_Status.CRITICAL ||
      item.status === GetClusterHealthResponse_Status.DEGRADED,
  );
  const unknownSubsystems = (health.data?.subsystems ?? []).filter(
    (item) =>
      item.status !== GetClusterHealthResponse_Status.OK &&
      item.status !== GetClusterHealthResponse_Status.CRITICAL &&
      item.status !== GetClusterHealthResponse_Status.DEGRADED,
  );
  const healthNeedsAttention =
    health.data?.status === GetClusterHealthResponse_Status.CRITICAL ||
    health.data?.status === GetClusterHealthResponse_Status.DEGRADED ||
    unhealthySubsystems.length > 0;

  const requests = cluster.data?.access?.request;
  const sessions = cluster.data?.core?.session;
  const devices = cluster.data?.core?.device;
  const authenticators = cluster.data?.core?.authenticator;
  const certificates = cluster.data?.enterprise?.certificate;
  const issuers = cluster.data?.enterprise?.certificateIssuer;
  const directories = cluster.data?.enterprise?.directoryProvider;
  const secretStores = cluster.data?.enterprise?.secretStore;

  const candidates: Item[] = [
    {
      id: "requestsPending",
      count: n(requests?.totalPending),
      label: "Access requests awaiting a decision",
      scope: "Access",
      severity: "warning",
      to: "/access/requests?state=PENDING",
      icon: Inbox,
    },
    {
      id: "requestsDeadline",
      count: n(requests?.totalPendingDeadlinePassed),
      label: "Access requests past their deadline",
      scope: "Access",
      severity: "critical",
      to: "/access/requests?isDeadlinePassed=true&state=PENDING",
      icon: CalendarClock,
    },
    {
      id: "requestsStale",
      count: n(requests?.totalPendingStale),
      label: "Access requests waiting for more than a day",
      scope: "Access",
      severity: "warning",
      to: "/access/requests?isPendingStale=true",
      icon: Hourglass,
    },
    {
      id: "sessionsPending",
      count: n(sessions?.totalPending),
      label: "Sessions pending approval",
      scope: "Core",
      severity: "warning",
      to: "/core/sessions?state=PENDING",
      icon: Terminal,
    },
    {
      id: "devicesPending",
      count: n(devices?.totalPending),
      label: "Devices pending approval",
      scope: "Core",
      severity: "warning",
      to: "/core/devices?state=PENDING",
      icon: LaptopMinimal,
    },
    {
      id: "authenticatorsPending",
      count: n(authenticators?.totalPending),
      label: "Authenticators pending registration",
      scope: "Core",
      severity: "info",
      to: "/core/authenticators?state=PENDING",
      icon: LockKeyhole,
    },
    {
      id: "certsExpired",
      count: n(certificates?.totalExpired),
      label: "Certificates already expired",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificates?isExpired=true",
      icon: CalendarX2,
    },
    {
      id: "certsExpiring",
      count: n(certificates?.totalExpiringSoon),
      label: "Certificates expiring within 30 days",
      scope: "Enterprise",
      severity: "warning",
      to: "/enterprise/certificates?isExpiringSoon=true",
      icon: ShieldCheck,
    },
    {
      id: "certsFailed",
      count: n(certificates?.totalIssuanceFailed),
      label: "Certificates that failed issuance",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificates?issuanceState=FAILED",
      icon: TriangleAlert,
    },
    {
      id: "issuersNotReady",
      count: n(issuers?.totalNotReady),
      label: "Certificate issuers not ready",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificateissuers?state=NOT_READY",
      icon: Crown,
    },
    {
      id: "directorySync",
      count: n(directories?.totalSynchronizationFailed),
      label: "Directory providers failing to sync",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/directoryproviders?synchronizationState=FAILED",
      icon: Folder,
    },
    {
      id: "secretStoreSync",
      count: n(secretStores?.totalSynchronizationFailed),
      label: "Secret stores failing to sync",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/secretstores?synchronizationState=FAILED",
      icon: BookKey,
    },
    {
      id: "componentErrors",
      count:
        n(componentLogs.data?.totalError) +
        n(componentLogs.data?.totalPanic) +
        n(componentLogs.data?.totalFatal),
      label: `Component errors in the last ${rangeLabel}`,
      scope: "Logs",
      severity: "critical",
      to: "/visibility/componentlogs?levels=ERROR,PANIC,FATAL",
      icon: ScrollText,
    },
  ];

  const items = candidates
    .filter((item) => item.count > 0)
    .sort(
      (a, b) =>
        SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity] ||
        b.count - a.count,
    );

  usePageConditions(
    items.map((item) => ({ label: item.label, count: item.count, to: item.to })),
  );

  const isLoading =
    cluster.isLoading || componentLogs.isLoading || health.isLoading;
  const failed =
    (cluster.data?.unavailables.length ?? 0) +
    (cluster.isError ? 1 : 0) +
    (componentLogs.isError ? 1 : 0) +
    (health.isError ? 1 : 0) +
    unknownSubsystems.length;
  const incomplete =
    failed > 0 || !health.data || !cluster.data || !componentLogs.data ||
    (health.data.status !== GetClusterHealthResponse_Status.OK &&
      !healthNeedsAttention);

  const critical =
    items.filter((item) => item.severity === "critical").length +
    unhealthySubsystems.filter(
      (item) => item.status === GetClusterHealthResponse_Status.CRITICAL,
    ).length;
  const signalCount = items.length + unhealthySubsystems.length;

  return (
    <Panel
      icon={critical > 0 ? ShieldAlert : AlertTriangle}
      title="Needs attention"
      description={
        isLoading
          ? "Checking signals across Core, Access and Enterprise"
          : incomplete
            ? "Some signals are unavailable; the assessment is incomplete"
            : items.length === 0 && !healthNeedsAttention
              ? "Everything across Core, Access and Enterprise looks healthy"
              : items.length === 0
                ? "Cluster health has signals that need attention"
                : `${signalCount} signal${signalCount === 1 ? "" : "s"} across Core, Access and Enterprise${critical > 0 ? ` · ${critical} critical` : ""}`
      }
    >
      {isLoading && items.length === 0 ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {[0, 1, 2, 3, 4, 5].map((index) => (
            <div
              key={index}
              className="h-[104px] motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {items.length === 0 && !healthNeedsAttention && !incomplete ? (
            <AllClear />
          ) : items.length > 0 ? (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
              {items.map((item) => (
                <Card key={item.id} item={item} />
              ))}
            </div>
          ) : null}

          {unhealthySubsystems.length > 0 ? (
            <SubsystemGrid subsystems={unhealthySubsystems} />
          ) : healthNeedsAttention ? (
            <div className="flex items-center gap-2 text-body text-slate-700">
              <StatusBadge status={health.data!.status} />
              Cluster health needs attention. Review the health signals below.
            </div>
          ) : null}

          {incomplete && (
            <div
              role="status"
              className="rounded-xl border border-amber-200 bg-amber-50/60 px-4 py-4 text-body text-amber-700"
            >
              Incomplete assessment
            </div>
          )}

          {failed > 0 && (
            <p role="alert" className="text-micro font-semibold text-amber-700">
              {failed} signal source{failed === 1 ? "" : "s"} could not be
              loaded, so this list may be incomplete.
            </p>
          )}
        </div>
      )}
    </Panel>
  );
};

export default Attention;
