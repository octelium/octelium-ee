import { AccessLog_Entry_Common_Status } from "@/apis/corev1/corev1";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, pct, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { seriesPoints, StatTile, toPoints } from "./components";
import {
  useAccessDataPoint,
  useAccessSummary,
  useAuthDataPoint,
  useAuthSummary,
  useClusterSummary,
  useComponentDataPoint,
  useComponentSummary,
} from "./queries";
import { compact } from "./utils";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const access = useAccessSummary(periodMinutes, QUERY_PRIORITY.critical);
  const accessPoints = useAccessDataPoint(
    periodMinutes,
    QUERY_PRIORITY.critical,
  );
  const auth = useAuthSummary(periodMinutes, QUERY_PRIORITY.critical);
  const authPoints = useAuthDataPoint(periodMinutes, QUERY_PRIORITY.high);

  const cluster = useClusterSummary(
    "total",
    periodMinutes,
    QUERY_PRIORITY.critical,
  );
  const clusterRange = useClusterSummary(
    "range",
    periodMinutes,
    QUERY_PRIORITY.high,
  );

  const component = useComponentSummary(periodMinutes, QUERY_PRIORITY.high);
  const componentErrorPoints = useComponentDataPoint(
    periodMinutes,
    "error",
    QUERY_PRIORITY.normal,
  );

  const totalRequests = n(access.data?.totalNumber);
  const totalRequestsPrev = n(access.data?.previous?.totalNumber);
  const denied = n(access.data?.totalDenied);
  const deniedPrev = n(access.data?.previous?.totalDenied);
  const logins = n(auth.data?.totalNumber);
  const loginsPrev = n(auth.data?.previous?.totalNumber);

  const sessions = cluster.data?.core?.session;
  const requests = cluster.data?.access?.request;
  const connected = n(sessions?.totalConnected);
  const pendingRequests = n(requests?.totalPending);

  const newSessions = n(clusterRange.data?.core?.session?.totalNumber);
  const createdRequests = n(clusterRange.data?.access?.request?.totalNumber);

  const errorsOf = (summary?: {
    totalError?: unknown;
    totalPanic?: unknown;
    totalFatal?: unknown;
  }) =>
    n(summary?.totalError) + n(summary?.totalPanic) + n(summary?.totalFatal);
  const errors = errorsOf(component.data);
  const errorsPrev = errorsOf(component.data?.previous);

  return (
    <section
      aria-label="Cluster signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Requests"
        value={totalRequests}
        footer={`${compact(n(access.data?.totalUser))} users · ${compact(n(access.data?.totalService))} services`}
        trend={{
          cur: totalRequests,
          prev: totalRequestsPrev,
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(accessPoints.data?.datapoints)}
        color={seriesColor(0)}
        to="/visibility/accesslogs"
        isLoading={access.isLoading}
        isError={access.isError}
        hasData={access.data !== undefined}
      />

      <StatTile
        label="Denied"
        value={denied}
        badge={`${pct(denied, totalRequests)}%`}
        footer={`${compact(n(access.data?.totalAllowed))} allowed in the last ${rangeLabel}`}
        trend={{ cur: denied, prev: deniedPrev, upIsGood: false, rangeLabel }}
        points={seriesPoints(
          accessPoints.data?.series,
          AccessLog_Entry_Common_Status[AccessLog_Entry_Common_Status.DENIED],
        )}
        color={STATUS_COLORS.critical}
        to="/visibility/accesslogs?status=DENIED"
        isLoading={access.isLoading}
        isError={access.isError}
        hasData={access.data !== undefined}
      />

      <StatTile
        label="Logins"
        value={logins}
        footer={`${compact(n(auth.data?.totalUser))} users · ${compact(n(auth.data?.totalReauthentication))} re-auths`}
        trend={{ cur: logins, prev: loginsPrev, upIsGood: true, rangeLabel }}
        points={toPoints(authPoints.data?.datapoints)}
        color={seriesColor(1)}
        to="/visibility/authenticationlogs"
        isLoading={auth.isLoading}
        isError={auth.isError}
        hasData={auth.data !== undefined}
      />

      <StatTile
        label="Connected"
        value={connected}
        badge={`${compact(n(sessions?.totalNumber))} total`}
        footer={
          clusterRange.data
            ? `${compact(newSessions)} new sessions in the last ${rangeLabel}`
            : "Sessions currently connected"
        }
        bar={{
          value: connected,
          total: n(sessions?.totalNumber),
          label: "connected",
        }}
        color={seriesColor(2)}
        to="/core/sessions?isConnected=true"
        isLoading={cluster.isLoading}
        isError={cluster.isError}
        hasData={cluster.data !== undefined}
      />

      <StatTile
        label="Access requests"
        value={createdRequests}
        badge={pendingRequests > 0 ? `${pendingRequests} pending` : undefined}
        footer={`${compact(n(requests?.totalActive))} active grants · ${compact(n(requests?.totalNumber))} all time`}
        bar={{
          value: pendingRequests,
          total: n(requests?.totalNumber),
          label: "awaiting review",
        }}
        color={seriesColor(4)}
        to="/access/requests"
        isLoading={clusterRange.isLoading}
        isError={clusterRange.isError}
        hasData={clusterRange.data !== undefined}
      />

      <StatTile
        label="Errors"
        value={errors}
        footer={`${compact(n(component.data?.totalWarn))} warnings across ${compact(n(component.data?.totalComponent))} components`}
        trend={{ cur: errors, prev: errorsPrev, upIsGood: false, rangeLabel }}
        points={toPoints(componentErrorPoints.data?.datapoints)}
        color={STATUS_COLORS.critical}
        to="/visibility/componentlogs?level=ERROR"
        isLoading={component.isLoading}
        isError={component.isError}
        hasData={component.data !== undefined}
      />
    </section>
  );
};

export default Signals;
