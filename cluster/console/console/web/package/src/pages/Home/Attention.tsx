import {
  GetRequestSummaryResponse,
  GetReviewSummaryResponse,
} from "@/apis/visibilityv1/access/vaccessv1";
import {
  GetAuthenticatorSummaryResponse,
  GetDeviceSummaryResponse,
  GetSessionSummaryResponse,
} from "@/apis/visibilityv1/core/vcorev1";
import {
  GetCertificateIssuerSummaryResponse,
  GetCertificateSummaryResponse,
  GetDirectoryProviderSummaryResponse,
  GetSecretStoreSummaryResponse,
} from "@/apis/visibilityv1/enterprise/venterprisev1";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  ArrowUpRight,
  BookKey,
  CalendarClock,
  CalendarX2,
  CheckCircle2,
  ClipboardCheck,
  Crown,
  Folder,
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
import { twMerge } from "tailwind-merge";
import { Panel } from "./components";
import { useComponentSummary, useResourceSummary } from "./queries";
import { compact, exact } from "./utils";

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

  const requests = useResourceSummary<GetRequestSummaryResponse>({
    api: "access",
    kind: "Request",
    method: "getRequestSummary",
    priority: QUERY_PRIORITY.high,
  });
  const reviews = useResourceSummary<GetReviewSummaryResponse>({
    api: "access",
    kind: "Review",
    method: "getReviewSummary",
    priority: QUERY_PRIORITY.high,
  });
  const sessions = useResourceSummary<GetSessionSummaryResponse>({
    api: "core",
    kind: "Session",
    method: "getSessionSummary",
    priority: QUERY_PRIORITY.high,
  });
  const devices = useResourceSummary<GetDeviceSummaryResponse>({
    api: "core",
    kind: "Device",
    method: "getDeviceSummary",
    priority: QUERY_PRIORITY.high,
  });
  const authenticators = useResourceSummary<GetAuthenticatorSummaryResponse>({
    api: "core",
    kind: "Authenticator",
    method: "getAuthenticatorSummary",
    priority: QUERY_PRIORITY.normal,
  });
  const certificates = useResourceSummary<GetCertificateSummaryResponse>({
    api: "enterprise",
    kind: "Certificate",
    method: "getCertificateSummary",
    priority: QUERY_PRIORITY.normal,
  });
  const issuers = useResourceSummary<GetCertificateIssuerSummaryResponse>({
    api: "enterprise",
    kind: "CertificateIssuer",
    method: "getCertificateIssuerSummary",
    priority: QUERY_PRIORITY.low,
  });
  const directories = useResourceSummary<GetDirectoryProviderSummaryResponse>({
    api: "enterprise",
    kind: "DirectoryProvider",
    method: "getDirectoryProviderSummary",
    priority: QUERY_PRIORITY.low,
  });
  const secretStores = useResourceSummary<GetSecretStoreSummaryResponse>({
    api: "enterprise",
    kind: "SecretStore",
    method: "getSecretStoreSummary",
    priority: QUERY_PRIORITY.low,
  });
  const componentLogs = useComponentSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.high,
  );

  const candidates: Item[] = [
    {
      id: "requestsPending",
      count: n(requests.data?.totalPending),
      label: "Access requests awaiting a decision",
      scope: "Access",
      severity: "warning",
      to: "/access/requests?state=PENDING",
      icon: Inbox,
    },
    {
      id: "requestsDeadline",
      count: n(requests.data?.totalDeadlinePassed),
      label: "Access requests past their deadline",
      scope: "Access",
      severity: "critical",
      to: "/access/requests?state=PENDING",
      icon: CalendarClock,
    },
    {
      id: "reviewsPending",
      count: n(reviews.data?.totalPending),
      label: "Reviews waiting on a reviewer",
      scope: "Access",
      severity: "warning",
      to: "/access/reviews?isDecided=false",
      icon: ClipboardCheck,
    },
    {
      id: "sessionsPending",
      count: n(sessions.data?.totalPending),
      label: "Sessions pending approval",
      scope: "Core",
      severity: "warning",
      to: "/core/sessions?state=PENDING",
      icon: Terminal,
    },
    {
      id: "devicesPending",
      count: n(devices.data?.totalPending),
      label: "Devices pending approval",
      scope: "Core",
      severity: "warning",
      to: "/core/devices?state=PENDING",
      icon: LaptopMinimal,
    },
    {
      id: "authenticatorsPending",
      count: n(authenticators.data?.totalPending),
      label: "Authenticators pending registration",
      scope: "Core",
      severity: "info",
      to: "/core/authenticators?state=PENDING",
      icon: LockKeyhole,
    },
    {
      id: "certsExpired",
      count: n(certificates.data?.totalExpired),
      label: "Certificates already expired",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificates?isExpired=true",
      icon: CalendarX2,
    },
    {
      id: "certsExpiring",
      count: n(certificates.data?.totalExpiringSoon),
      label: "Certificates expiring within 30 days",
      scope: "Enterprise",
      severity: "warning",
      to: "/enterprise/certificates?isExpiringSoon=true",
      icon: ShieldCheck,
    },
    {
      id: "certsFailed",
      count: n(certificates.data?.totalIssuanceFailed),
      label: "Certificates that failed issuance",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificates?issuanceState=FAILED",
      icon: TriangleAlert,
    },
    {
      id: "issuersNotReady",
      count: n(issuers.data?.totalNotReady),
      label: "Certificate issuers not ready",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/certificateissuers?state=NOT_READY",
      icon: Crown,
    },
    {
      id: "directorySync",
      count: n(directories.data?.totalSynchronizationFailed),
      label: "Directory providers failing to sync",
      scope: "Enterprise",
      severity: "critical",
      to: "/enterprise/directoryproviders?synchronizationState=FAILED",
      icon: Folder,
    },
    {
      id: "secretStoreSync",
      count: n(secretStores.data?.totalSynchronizationFailed),
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
      to: "/visibility/componentlogs?level=ERROR",
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

  const queries = [
    requests,
    reviews,
    sessions,
    devices,
    authenticators,
    certificates,
    issuers,
    directories,
    secretStores,
    componentLogs,
  ];
  const isLoading = queries.some((query) => query.isLoading);
  const failed = queries.filter((query) => query.isError).length;

  const critical = items.filter((item) => item.severity === "critical").length;

  return (
    <Panel
      icon={critical > 0 ? ShieldAlert : AlertTriangle}
      title="Needs attention"
      description={
        items.length === 0
          ? "Everything across Core, Access and Enterprise looks healthy"
          : `${items.length} signal${items.length === 1 ? "" : "s"} across Core, Access and Enterprise${critical > 0 ? ` · ${critical} critical` : ""}`
      }
    >
      {isLoading && items.length === 0 ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {[0, 1, 2, 3, 4, 5].map((index) => (
            <div
              key={index}
              className="h-[104px] animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {items.length === 0 ? (
            <AllClear />
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
              {items.map((item) => (
                <Card key={item.id} item={item} />
              ))}
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
