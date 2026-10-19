import { Panel } from "@/components/Dashboard/components";
import { compact, exact } from "@/components/Dashboard/utils";
import { twMerge } from "@/lib/utils";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlarmClock,
  ArrowUpRight,
  CalendarClock,
  CheckCircle2,
  Hourglass,
  LucideIcon,
  Plug,
  PowerOff,
  SearchX,
  Send,
  ShieldAlert,
  TriangleAlert,
} from "lucide-react";
import { Link } from "react-router-dom";
import { useAccessCreated, useAccessTotals } from "./queries";

type Severity = "critical" | "warning" | "info";

type Item = {
  id: string;
  count: number;
  label: string;
  hint: string;
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
        "group relative flex min-w-0 flex-col gap-1.5 rounded-xl border px-3.5 py-3 outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
        SEVERITY_STYLE[item.severity],
      )}
    >
      <div className="flex items-center gap-2">
        <Icon
          size={14}
          strokeWidth={2.3}
          className={twMerge("shrink-0", SEVERITY_TEXT[item.severity])}
        />
        <span
          className={twMerge(
            "text-xl font-bold leading-none tabular-nums",
            SEVERITY_TEXT[item.severity],
          )}
          title={exact(item.count)}
        >
          {compact(item.count)}
        </span>
      </div>
      <span className="text-xs font-semibold leading-4 text-slate-700">
        {item.label}
      </span>
      <span className="text-micro font-normal leading-4 text-slate-500">
        {item.hint}
      </span>
      <ArrowUpRight
        size={12}
        strokeWidth={2.5}
        aria-hidden="true"
        className="absolute right-3 top-3 text-slate-400 opacity-0 transition-opacity duration-150 group-hover:opacity-100"
      />
    </Link>
  );
};

const Attention = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.critical);

  const requests = totals.data?.access?.request;
  const policies = totals.data?.access?.policy;
  const integrations = totals.data?.access?.integration;
  const deliveries = totals.data?.access?.integrationBinding;
  const fresh = created.data?.access?.request;

  const candidates: Item[] = [
    {
      id: "deadline",
      count: n(requests?.totalPendingDeadlinePassed),
      label: "Requests past their deadline",
      hint: "Still pending although the requester needed an answer earlier.",
      severity: "critical",
      to: "/access/requests?isDeadlinePassed=true&state=PENDING",
      icon: CalendarClock,
    },
    {
      id: "integrationsError",
      count: n(integrations?.totalError),
      label: "Integrations failing",
      hint: "Requests are not presented there until they recover.",
      severity: "critical",
      to: "/access/integrations?state=ERROR",
      icon: Plug,
    },
    {
      id: "deliveries",
      count: n(deliveries?.totalFailing),
      label: "Deliveries failing",
      hint: "Messages or issues that could not be created or updated.",
      severity: "critical",
      to: "/access/integrations",
      icon: Send,
    },
    {
      id: "stale",
      count: n(requests?.totalPendingStale),
      label: "Waiting for more than a day",
      hint: "Reviewers may be unaware: check the review surfaces.",
      severity: "warning",
      to: "/access/requests?isPendingStale=true",
      icon: Hourglass,
    },
    {
      id: "unmatched",
      count: n(fresh?.totalUnmatched),
      label: `Rejected with no matching rule · ${rangeLabel}`,
      hint: "A gap in your Policies: nobody could even review them.",
      severity: "warning",
      to: "/access/requests?isUnmatched=true",
      icon: SearchX,
    },
    {
      id: "integrationsDegraded",
      count: n(integrations?.totalDegraded),
      label: "Integrations degraded",
      hint: "The last operation failed after earlier successes.",
      severity: "warning",
      to: "/access/integrations?state=DEGRADED",
      icon: TriangleAlert,
    },
    {
      id: "expiring",
      count: n(requests?.totalExpiringSoon),
      label: "Access ending within the hour",
      hint: "People may need to request again soon.",
      severity: "info",
      to: "/access/requests?isExpiringSoon=true",
      icon: AlarmClock,
    },
    {
      id: "policiesDisabled",
      count: n(policies?.totalDisabled),
      label: "Policies disabled",
      hint: "Their rules are skipped when requests are evaluated.",
      severity: "info",
      to: "/access/policies?isDisabled=true",
      icon: PowerOff,
    },
  ];

  const items = candidates
    .filter((item) => item.count > 0)
    .sort(
      (a, b) =>
        SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity] ||
        b.count - a.count,
    );
  const critical = items.filter((item) => item.severity === "critical").length;
  const isLoading = totals.isLoading || created.isLoading;
  const incomplete =
    !totals.data ||
    !created.data ||
    totals.data.unavailables.length > 0 ||
    created.data.unavailables.length > 0;

  return (
    <Panel
      icon={critical > 0 ? ShieldAlert : TriangleAlert}
      title="Needs attention"
      description={
        isLoading
          ? "Checking requests, integrations and policies"
          : items.length === 0 && incomplete
            ? "Some signals are unavailable; the assessment is incomplete"
            : items.length === 0
              ? "No overdue request, failing integration or policy gap"
              : `${items.length} signal${items.length === 1 ? "" : "s"}${critical > 0 ? ` · ${critical} critical` : ""}`
      }
    >
      {isLoading && items.length === 0 ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {[0, 1, 2, 3].map((index) => (
            <div
              key={index}
              className="h-[104px] motion-safe:animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : items.length === 0 && incomplete ? (
        <div
          role="status"
          className="rounded-xl border border-amber-200 bg-amber-50/60 px-4 py-4 text-body text-amber-700"
        >
          Incomplete assessment
        </div>
      ) : items.length === 0 ? (
        <div className="flex min-h-20 items-center justify-center gap-3 rounded-xl border border-dashed border-emerald-200 bg-emerald-50/50 px-6 py-5 text-center">
          <CheckCircle2 size={17} strokeWidth={2.2} className="text-emerald-600" />
          <span className="text-body font-semibold text-slate-700">
            The access workflow is healthy
          </span>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {items.map((item) => (
            <Card key={item.id} item={item} />
          ))}
        </div>
      )}
      {!isLoading && incomplete && (
        <p role="alert" className="mt-3 text-micro font-semibold text-amber-700">
          Some summaries could not be loaded, so this list may be incomplete.
        </p>
      )}
    </Panel>
  );
};

export default Attention;
