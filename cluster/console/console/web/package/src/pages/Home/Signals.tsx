import { GetRequestSummaryResponse } from "@/apis/visibilityv1/access/vaccessv1";
import { GetSessionSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { StatTile, toPoints } from "./components";
import {
  useAccessDataPoint,
  useAccessSummary,
  useAuthDataPoint,
  useAuthSummary,
  useComponentDataPoint,
  useComponentSummary,
  useResourceRangeSummary,
  useResourceSummary,
} from "./queries";
import { compact } from "./utils";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const accessCur = useAccessSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.critical,
  );
  const accessPrev = useAccessSummary(
    periodMinutes,
    "previous",
    QUERY_PRIORITY.high,
  );
  const accessPoints = useAccessDataPoint(
    periodMinutes,
    "all",
    QUERY_PRIORITY.critical,
  );
  const deniedPoints = useAccessDataPoint(
    periodMinutes,
    "denied",
    QUERY_PRIORITY.high,
  );

  const authCur = useAuthSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.critical,
  );
  const authPrev = useAuthSummary(
    periodMinutes,
    "previous",
    QUERY_PRIORITY.high,
  );
  const authPoints = useAuthDataPoint(periodMinutes, QUERY_PRIORITY.high);

  const sessions = useResourceSummary<GetSessionSummaryResponse>({
    api: "core",
    kind: "Session",
    method: "getSessionSummary",
    priority: QUERY_PRIORITY.critical,
  });
  const newSessions = useResourceRangeSummary<GetSessionSummaryResponse>({
    api: "core",
    kind: "Session",
    method: "getSessionSummary",
    periodMinutes,
    priority: QUERY_PRIORITY.high,
  });

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

  const componentCur = useComponentSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.high,
  );
  const componentPrev = useComponentSummary(
    periodMinutes,
    "previous",
    QUERY_PRIORITY.normal,
  );
  const componentErrorPoints = useComponentDataPoint(
    periodMinutes,
    "error",
    QUERY_PRIORITY.normal,
  );

  const totalRequests = n(accessCur.data?.totalNumber);
  const totalRequestsPrev = n(accessPrev.data?.totalNumber);
  const denied = n(accessCur.data?.totalDenied);
  const deniedPrev = n(accessPrev.data?.totalDenied);
  const logins = n(authCur.data?.totalNumber);
  const loginsPrev = n(authPrev.data?.totalNumber);
  const connected = n(sessions.data?.totalConnected);
  const pendingRequests = n(requests.data?.totalPending);
  const createdRequests = n(newRequests.data?.totalNumber);

  const errorsOf = (summary?: {
    totalError?: unknown;
    totalPanic?: unknown;
    totalFatal?: unknown;
  }) =>
    n(summary?.totalError) + n(summary?.totalPanic) + n(summary?.totalFatal);
  const errors = errorsOf(componentCur.data);
  const errorsPrev = errorsOf(componentPrev.data);

  return (
    <section
      aria-label="Cluster signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Requests"
        value={totalRequests}
        footer={`${compact(n(accessCur.data?.totalUser))} users · ${compact(n(accessCur.data?.totalService))} services`}
        trend={{
          cur: totalRequests,
          prev: totalRequestsPrev,
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(accessPoints.data?.datapoints)}
        color={seriesColor(0)}
        to="/visibility/accesslogs"
        isLoading={accessCur.isLoading}
        isError={accessCur.isError}
        hasData={accessCur.data !== undefined}
      />

      <StatTile
        label="Denied"
        value={denied}
        badge={`${pct(denied, totalRequests)}%`}
        footer={`${compact(n(accessCur.data?.totalAllowed))} allowed in the last ${rangeLabel}`}
        trend={{ cur: denied, prev: deniedPrev, upIsGood: false, rangeLabel }}
        points={toPoints(deniedPoints.data?.datapoints)}
        color={STATUS_COLORS.critical}
        to="/visibility/accesslogs?status=DENIED"
        isLoading={accessCur.isLoading}
        isError={accessCur.isError}
        hasData={accessCur.data !== undefined}
      />

      <StatTile
        label="Logins"
        value={logins}
        footer={`${compact(n(authCur.data?.totalUser))} users · ${compact(n(authCur.data?.totalReauthentication))} re-auths`}
        trend={{ cur: logins, prev: loginsPrev, upIsGood: true, rangeLabel }}
        points={toPoints(authPoints.data?.datapoints)}
        color={seriesColor(1)}
        to="/visibility/authenticationlogs"
        isLoading={authCur.isLoading}
        isError={authCur.isError}
        hasData={authCur.data !== undefined}
      />

      <StatTile
        label="Connected"
        value={connected}
        badge={`${compact(n(sessions.data?.totalNumber))} total`}
        footer={
          newSessions.data
            ? `${compact(n(newSessions.data.totalNumber))} new sessions in the last ${rangeLabel}`
            : "Sessions currently connected"
        }
        bar={{
          value: connected,
          total: n(sessions.data?.totalNumber),
          label: "connected",
        }}
        color={seriesColor(2)}
        to="/core/sessions?isConnected=true"
        isLoading={sessions.isLoading}
        isError={sessions.isError}
        hasData={sessions.data !== undefined}
      />

      <StatTile
        label="Access requests"
        value={createdRequests}
        badge={pendingRequests > 0 ? `${pendingRequests} pending` : undefined}
        footer={`${compact(n(requests.data?.totalActive))} active grants · ${compact(n(requests.data?.totalNumber))} all time`}
        bar={{
          value: pendingRequests,
          total: n(requests.data?.totalNumber),
          label: "awaiting review",
        }}
        color={seriesColor(4)}
        to="/access/requests"
        isLoading={newRequests.isLoading}
        isError={newRequests.isError}
        hasData={newRequests.data !== undefined}
      />

      <StatTile
        label="Errors"
        value={errors}
        footer={`${compact(n(componentCur.data?.totalWarn))} warnings in the last ${rangeLabel}`}
        trend={{ cur: errors, prev: errorsPrev, upIsGood: false, rangeLabel }}
        points={toPoints(componentErrorPoints.data?.datapoints)}
        color={STATUS_COLORS.critical}
        to="/visibility/componentlogs"
        isLoading={componentCur.isLoading}
        isError={componentCur.isError}
        hasData={componentCur.data !== undefined}
      />
    </section>
  );
};

export default Signals;
