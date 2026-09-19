import { StatTile } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { useAccessCreated, useAccessTotals } from "./queries";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.critical);

  const requests = totals.data?.access?.request;
  const reviews = totals.data?.access?.review;
  const fresh = created.data?.access?.request;

  const pending = n(requests?.totalPending);
  const decided = n(fresh?.totalApproved) + n(fresh?.totalRejected);
  const urgent =
    n(requests?.totalUrgencyHigh) +
    n(requests?.totalUrgencyVeryHigh) +
    n(requests?.totalUrgencyHighest);

  return (
    <section
      aria-label="Access signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Awaiting review"
        value={pending}
        badge={
          n(reviews?.totalPending) > 0
            ? `${compact(n(reviews?.totalPending))} reviews`
            : undefined
        }
        footer={`${compact(n(requests?.totalNumber))} Requests all time`}
        bar={{
          value: pending,
          total: n(requests?.totalNumber),
          label: "still undecided",
        }}
        color={pending > 0 ? STATUS_COLORS.warning : STATUS_COLORS.good}
        to="/access/requests?state=PENDING"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Past deadline"
        value={n(requests?.totalDeadlinePassed)}
        footer={`${compact(n(requests?.totalWithDeadline))} Requests carry a deadline`}
        bar={{
          value: n(requests?.totalDeadlinePassed),
          total: n(requests?.totalWithDeadline),
          label: "already overdue",
        }}
        color={STATUS_COLORS.critical}
        to="/access/requests?isDeadlinePassed=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="High urgency"
        value={urgent}
        footer={`${compact(n(requests?.totalUrgencyHighest))} highest · ${compact(n(requests?.totalUrgencyVeryHigh))} very high`}
        bar={{
          value: urgent,
          total: n(requests?.totalNumber),
          label: "of all Requests",
        }}
        color={STATUS_COLORS.serious}
        to="/access/requests?urgency=HIGHEST"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label={`Raised · ${rangeLabel}`}
        value={n(fresh?.totalNumber)}
        trend={{
          cur: n(fresh?.totalNumber),
          prev: n(fresh?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        footer={`${compact(n(fresh?.totalUser))} requesters · ${compact(n(fresh?.totalSubjectUser))} subjects`}
        bar={{
          value: decided,
          total: n(fresh?.totalNumber),
          label: "already decided",
        }}
        color={seriesColor(0)}
        to="/access/requests"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label={`Approved · ${rangeLabel}`}
        value={n(fresh?.totalApproved)}
        badge={
          decided > 0 ? `${pct(n(fresh?.totalApproved), decided)}%` : undefined
        }
        trend={{
          cur: n(fresh?.totalApproved),
          prev: n(fresh?.previous?.totalApproved),
          upIsGood: true,
          rangeLabel,
        }}
        footer={`${compact(n(fresh?.totalRejected))} rejected in the same window`}
        bar={{
          value: n(fresh?.totalApproved),
          total: decided,
          label: "of the decisions",
        }}
        color={STATUS_COLORS.good}
        to="/access/requests?state=APPROVED"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label="Active grants"
        value={n(requests?.totalActive)}
        footer={`${compact(n(requests?.totalService))} Services · ${compact(n(requests?.totalCatalog))} Catalogs granted`}
        bar={{
          value: n(requests?.totalActive),
          total: n(requests?.totalApproved),
          label: "of the approvals are live",
        }}
        color={seriesColor(2)}
        to="/access/requests?isActive=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />
    </section>
  );
};

export default Signals;
