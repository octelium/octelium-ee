import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetRequestDataPointRequest_GroupBy } from "@/apis/visibilityv1/access/vaccessv1";
import { StatTile, toPoints } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { formatSeconds } from "../common";
import {
  useAccessCreated,
  useAccessTotals,
  useRequestDataPoint,
} from "./queries";

export const waitedFor = (since?: Timestamp) =>
  since
    ? formatSeconds((Date.now() - Timestamp.toDate(since).getTime()) / 1000)
    : undefined;

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.critical);
  const points = useRequestDataPoint(
    periodMinutes,
    GetRequestDataPointRequest_GroupBy.GROUP_BY_UNSET,
    QUERY_PRIORITY.high,
  );

  const requests = totals.data?.access?.request;
  const fresh = created.data?.access?.request;
  const integrations = totals.data?.access?.integration;
  const deliveries = totals.data?.access?.integrationBinding;

  const pending = n(requests?.totalPending);
  const stale = n(requests?.totalPendingStale);
  const oldest = waitedFor(requests?.oldestPendingAt);

  const approved = n(fresh?.totalGranted);
  const rejected = n(fresh?.totalRejected);
  const decided = approved + rejected;
  const prevDecided =
    n(fresh?.previous?.totalGranted) + n(fresh?.previous?.totalRejected);

  const decision = fresh?.decisionTime;
  const prevDecision = fresh?.previous?.decisionTime;

  const integrationsTotal = n(integrations?.totalNumber);
  const integrationsProblems =
    n(integrations?.totalDegraded) + n(integrations?.totalError);
  const failingDeliveries = n(deliveries?.totalFailing);

  return (
    <section
      aria-label="Access signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Awaiting a decision"
        value={pending}
        badge={oldest && pending > 0 ? `oldest ${oldest}` : undefined}
        footer={
          pending === 0
            ? "Every request has been decided"
            : `${compact(stale)} waiting > 24h · ${compact(n(requests?.totalPendingDeadlinePassed))} past deadline`
        }
        bar={{
          value: stale,
          total: pending,
          label: "waiting more than a day",
        }}
        color={stale > 0 ? STATUS_COLORS.serious : STATUS_COLORS.warning}
        to="/access/requests?state=PENDING"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Live access"
        value={n(requests?.totalActive)}
        badge={
          n(requests?.totalExpiringSoon) > 0
            ? `${compact(n(requests?.totalExpiringSoon))} ending < 1h`
            : undefined
        }
        footer={`${compact(n(requests?.totalService))} Services · ${compact(n(requests?.totalCatalog))} Catalogs requested so far`}
        bar={{
          value: n(requests?.totalActive),
          total: n(requests?.totalApproved),
          label: "of the approvals are still live",
        }}
        color={seriesColor(2)}
        to="/access/requests?isActive=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label={`Requests · ${rangeLabel}`}
        value={n(fresh?.totalNumber)}
        trend={{
          cur: n(fresh?.totalNumber),
          prev: n(fresh?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(points.data?.datapoints)}
        footer={`${compact(n(fresh?.totalUser))} requesters · ${compact(n(fresh?.totalOnBehalf))} on behalf of others`}
        color={seriesColor(0)}
        to="/access/requests"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label={`Approval rate · ${rangeLabel}`}
        value={pct(approved, decided)}
        valueLabel={decided > 0 ? `${pct(approved, decided)}%` : "—"}
        badge={
          n(fresh?.totalAutoApproved) > 0 && approved > 0
            ? `${pct(n(fresh?.totalAutoApproved), approved)}% auto`
            : undefined
        }
        trend={
          decided > 0 && prevDecided > 0
            ? {
                cur: pct(approved, decided),
                prev: pct(n(fresh?.previous?.totalGranted), prevDecided),
                upIsGood: true,
                rangeLabel,
              }
            : undefined
        }
        footer={`${compact(approved)} granted · ${compact(rejected)} rejected · ${compact(n(fresh?.totalUnmatched))} without a rule`}
        bar={{ value: approved, total: decided, label: "of the decisions" }}
        color={STATUS_COLORS.good}
        to="/access/requests?state=REJECTED"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label={`Time to decision · ${rangeLabel}`}
        value={decision?.p50Seconds ?? 0}
        valueLabel={decision ? formatSeconds(decision.p50Seconds) : "—"}
        badge={decision ? "median" : undefined}
        trend={
          decision && prevDecision
            ? {
                cur: Math.round(decision.p50Seconds),
                prev: Math.round(prevDecision.p50Seconds),
                upIsGood: false,
                rangeLabel,
              }
            : undefined
        }
        footer={
          decision
            ? `p90 ${formatSeconds(decision.p90Seconds)} · ${compact(n(decision.count))} reviewed requests`
            : "No reviewed request was decided in this window"
        }
        bar={{
          value: n(fresh?.totalReviewed),
          total: n(fresh?.totalNumber),
          label: "of the requests needed a review",
        }}
        color={seriesColor(4)}
        to="/access/requests?effect=REVIEW"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label="Integrations"
        value={n(integrations?.totalReady)}
        valueLabel={
          integrationsTotal > 0
            ? `${compact(n(integrations?.totalReady))}/${compact(integrationsTotal)}`
            : "0"
        }
        badge={
          failingDeliveries > 0
            ? `${compact(failingDeliveries)} failing deliveries`
            : undefined
        }
        footer={
          integrationsTotal === 0
            ? "Connect Slack, Jira or a webhook to review outside the portal"
            : integrationsProblems > 0
              ? `${compact(integrationsProblems)} with problems · ${compact(n(integrations?.totalDisabled))} disabled`
              : `All healthy · ${compact(n(integrations?.totalDisabled))} disabled`
        }
        bar={{
          value: n(integrations?.totalReady),
          total: integrationsTotal,
          label: "ready",
        }}
        color={
          integrationsProblems > 0 || failingDeliveries > 0
            ? STATUS_COLORS.critical
            : STATUS_COLORS.good
        }
        to="/access/integrations"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />
    </section>
  );
};

export default Signals;
