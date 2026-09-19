import { Request_Spec_Urgency } from "@/apis/accessv1/accessv1";
import {
  GetCatalogSummaryResponse,
  GetPolicySummaryResponse,
  GetRequestSummaryResponse,
  GetReviewSummaryResponse,
} from "@/apis/visibilityv1/access/vaccessv1";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { getUrgencyColor } from "@/pages/access/Request/utils";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  CalendarClock,
  CircleCheck,
  CircleX,
  ClipboardCheck,
  Clock3,
  Inbox,
  Layers,
  Shield,
  Timer,
  UserCheck,
} from "lucide-react";
import { MiniStat, MiniStatGrid, Panel } from "./components";
import { useResourceRangeSummary, useResourceSummary } from "./queries";
import { compact } from "./utils";

const STATE_COLORS = {
  approved: "var(--color-emerald-600)",
  rejected: "var(--color-red-600)",
  revoked: "var(--color-red-600)",
  expired: "var(--color-slate-500)",
  cancelled: "var(--color-slate-500)",
  pending: "var(--color-amber-600)",
} as const;

const Governance = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const requests = useResourceSummary<GetRequestSummaryResponse>({
    api: "access",
    kind: "Request",
    method: "getRequestSummary",
    priority: QUERY_PRIORITY.high,
  });
  const newRequests = useResourceRangeSummary<GetRequestSummaryResponse>({
    api: "access",
    kind: "Request",
    method: "getRequestSummary",
    periodMinutes,
    priority: QUERY_PRIORITY.normal,
  });
  const reviews = useResourceSummary<GetReviewSummaryResponse>({
    api: "access",
    kind: "Review",
    method: "getReviewSummary",
    priority: QUERY_PRIORITY.high,
  });
  const policies = useResourceSummary<GetPolicySummaryResponse>({
    api: "access",
    kind: "Policy",
    method: "getPolicySummary",
    priority: QUERY_PRIORITY.low,
  });
  const catalogs = useResourceSummary<GetCatalogSummaryResponse>({
    api: "access",
    kind: "Catalog",
    method: "getCatalogSummary",
    priority: QUERY_PRIORITY.low,
  });

  const data = requests.data;
  const states = [
    {
      label: "Approved",
      value: n(data?.totalApproved),
      color: STATE_COLORS.approved,
    },
    {
      label: "Pending",
      value: n(data?.totalPending),
      color: STATE_COLORS.pending,
    },
    {
      label: "Rejected",
      value: n(data?.totalRejected),
      color: STATE_COLORS.rejected,
    },
    {
      label: "Revoked",
      value: n(data?.totalRevoked),
      color: STATE_COLORS.revoked,
    },
    {
      label: "Expired",
      value: n(data?.totalExpired),
      color: STATE_COLORS.expired,
    },
    {
      label: "Cancelled",
      value: n(data?.totalCancelled),
      color: STATE_COLORS.cancelled,
    },
  ];

  const urgencies = [
    {
      label: "Highest",
      value: n(data?.totalUrgencyHighest),
      color: getUrgencyColor(Request_Spec_Urgency.HIGHEST),
    },
    {
      label: "Very high",
      value: n(data?.totalUrgencyVeryHigh),
      color: getUrgencyColor(Request_Spec_Urgency.VERY_HIGH),
    },
    {
      label: "High",
      value: n(data?.totalUrgencyHigh),
      color: getUrgencyColor(Request_Spec_Urgency.HIGH),
    },
    {
      label: "Normal",
      value: n(data?.totalUrgencyNormal),
      color: getUrgencyColor(Request_Spec_Urgency.NORMAL),
    },
    {
      label: "Low",
      value: n(data?.totalUrgencyLow),
      color: getUrgencyColor(Request_Spec_Urgency.LOW),
    },
  ];

  return (
    <Panel
      icon={UserCheck}
      title="Access governance"
      description={`${compact(n(newRequests.data?.totalNumber))} requests raised in the last ${rangeLabel}`}
      to="/access"
      toLabel="Access API"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Pending"
            value={n(data?.totalPending)}
            tone={n(data?.totalPending) > 0 ? "warning" : "default"}
            icon={Clock3}
            to="/access/requests?state=PENDING"
          />
          <MiniStat
            label="Active grants"
            value={n(data?.totalActive)}
            tone="positive"
            icon={Timer}
            to="/access/requests?isActive=true"
          />
          <MiniStat
            label="Reviews open"
            value={n(reviews.data?.totalPending)}
            tone={n(reviews.data?.totalPending) > 0 ? "warning" : "default"}
            icon={ClipboardCheck}
            to="/access/reviews?isDecided=false"
          />
          <MiniStat
            label="Past deadline"
            value={n(data?.totalDeadlinePassed)}
            tone={n(data?.totalDeadlinePassed) > 0 ? "critical" : "default"}
            icon={CalendarClock}
          />
          <MiniStat
            label="Policies"
            value={n(policies.data?.totalNumber)}
            icon={Shield}
            to="/access/policies"
          />
          <MiniStat
            label="Catalogs"
            value={n(catalogs.data?.totalNumber)}
            icon={Layers}
            to="/access/catalogs"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                Request outcomes
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(data?.totalNumber))} all time
              </span>
            </div>
            <CompositionBar segments={states} total={n(data?.totalNumber)} />
          </div>

          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                Urgency mix
              </span>
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {compact(n(data?.totalSubjectUser))} subjects
              </span>
            </div>
            <CompositionBar segments={urgencies} total={n(data?.totalNumber)} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
          <MiniStat
            label={`New · ${rangeLabel}`}
            value={n(newRequests.data?.totalNumber)}
            icon={Inbox}
            to="/access/requests"
          />
          <MiniStat
            label={`Approved · ${rangeLabel}`}
            value={n(newRequests.data?.totalApproved)}
            tone="positive"
            icon={CircleCheck}
            to="/access/requests?state=APPROVED"
          />
          <MiniStat
            label={`Rejected · ${rangeLabel}`}
            value={n(newRequests.data?.totalRejected)}
            tone={
              n(newRequests.data?.totalRejected) > 0 ? "critical" : "default"
            }
            icon={CircleX}
            to="/access/requests?state=REJECTED"
          />
          <MiniStat
            label="Review steps"
            value={n(policies.data?.totalReviewStep)}
            icon={ClipboardCheck}
          />
          <MiniStat
            label="Reviewers"
            value={n(policies.data?.totalReviewer)}
            icon={UserCheck}
          />
          <MiniStat
            label="Auto-approve rules"
            value={n(policies.data?.totalRuleAutoApprove)}
            icon={CircleCheck}
            to="/access/policies?effect=AUTO_APPROVE"
          />
        </div>
      </div>
    </Panel>
  );
};

export default Governance;
