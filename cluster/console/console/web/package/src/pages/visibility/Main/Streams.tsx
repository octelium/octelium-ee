import { StatTile, toPoints } from "@/components/Dashboard/components";
import {
  useAccessDataPoint,
  useAccessSummary,
  useAuditDataPoint,
  useAuditSummary,
  useAuthDataPoint,
  useAuthSummary,
  useComponentDataPoint,
  useComponentSummary,
} from "@/components/Dashboard/queries";
import { compact } from "@/components/Dashboard/utils";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";

const Streams = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const access = useAccessSummary(periodMinutes, QUERY_PRIORITY.critical);
  const accessPoints = useAccessDataPoint(periodMinutes, QUERY_PRIORITY.high);
  const auth = useAuthSummary(periodMinutes, QUERY_PRIORITY.critical);
  const authPoints = useAuthDataPoint(periodMinutes, QUERY_PRIORITY.high);
  const audit = useAuditSummary(periodMinutes, QUERY_PRIORITY.high);
  const auditPoints = useAuditDataPoint(periodMinutes, QUERY_PRIORITY.normal);
  const component = useComponentSummary(periodMinutes, QUERY_PRIORITY.high);
  const componentPoints = useComponentDataPoint(
    periodMinutes,
    "all",
    QUERY_PRIORITY.normal,
  );

  const errors =
    n(component.data?.totalError) +
    n(component.data?.totalPanic) +
    n(component.data?.totalFatal);

  return (
    <section
      aria-label="Log streams"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4"
    >
      <StatTile
        label="Access log"
        value={n(access.data?.totalNumber)}
        badge={`${compact(n(access.data?.totalDenied))} denied`}
        footer={`${compact(n(access.data?.totalSession))} sessions · ${compact(n(access.data?.totalService))} Services touched`}
        trend={{
          cur: n(access.data?.totalNumber),
          prev: n(access.data?.previous?.totalNumber),
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
        label="Authentication log"
        value={n(auth.data?.totalNumber)}
        badge={`${compact(n(auth.data?.totalReauthentication))} re-auth`}
        footer={`${compact(n(auth.data?.totalUser))} users · ${compact(n(auth.data?.totalIdentityProvider))} providers used`}
        trend={{
          cur: n(auth.data?.totalNumber),
          prev: n(auth.data?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(authPoints.data?.datapoints)}
        color={seriesColor(1)}
        to="/visibility/authenticationlogs"
        isLoading={auth.isLoading}
        isError={auth.isError}
        hasData={auth.data !== undefined}
      />

      <StatTile
        label="Audit log"
        value={n(audit.data?.totalNumber)}
        badge={`${compact(n(audit.data?.totalDelete))} deletes`}
        footer={`${compact(n(audit.data?.totalUser))} operators · ${compact(n(audit.data?.totalResource))} resources changed`}
        trend={{
          cur: n(audit.data?.totalNumber),
          prev: n(audit.data?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(auditPoints.data?.datapoints)}
        color={seriesColor(5)}
        to="/visibility/auditlogs"
        isLoading={audit.isLoading}
        isError={audit.isError}
        hasData={audit.data !== undefined}
      />

      <StatTile
        label="Component log"
        value={n(component.data?.totalNumber)}
        badge={errors > 0 ? `${compact(errors)} errors` : undefined}
        footer={`${compact(n(component.data?.totalComponent))} components · ${compact(n(component.data?.totalWarn))} warnings`}
        trend={{
          cur: n(component.data?.totalNumber),
          prev: n(component.data?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        points={toPoints(componentPoints.data?.datapoints)}
        color={errors > 0 ? STATUS_COLORS.critical : seriesColor(2)}
        to="/visibility/componentlogs"
        isLoading={component.isLoading}
        isError={component.isError}
        hasData={component.data !== undefined}
      />
    </section>
  );
};

export default Streams;
