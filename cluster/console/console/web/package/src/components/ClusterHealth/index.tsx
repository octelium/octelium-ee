import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  GetAccessLogDataPointRequest,
  GetAccessLogSummaryRequest,
  GetAuditLogDataPointRequest,
  GetAuditLogSummaryRequest,
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogSummaryRequest,
  GetComponentLogDataPointRequest,
  GetComponentLogSummaryRequest,
} from "@/apis/visibilityv1/visibilityv1";
import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import LineChart from "@/components/Charts/LineChart";
import { seriesColor, STATUS_COLORS } from "@/utils/charts/palette";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
  getClientVisibilityComponentLog,
  getClientVisibilityCore,
  refetchIntervalChart,
} from "@/utils/client";
import {
  buildTimestamps,
  deltaPct,
  getAutoInterval,
  n,
  NO_REFS,
  pct,
  periodLabel,
  toTs,
  visibilityKeys,
} from "@/utils/visibility";
import { useQuery } from "@tanstack/react-query";
import { Minus, TrendingDown, TrendingUp } from "lucide-react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

type Point = { ts: Timestamp; value: number };

const SUMMARY_REFETCH = 60_000;

const compact = (value: number) =>
  new Intl.NumberFormat(undefined, {
    notation: value >= 10_000 ? "compact" : "standard",
    maximumFractionDigits: 1,
  }).format(value);

const Delta = (props: {
  cur: number;
  prev: number;
  upIsGood: boolean;
  rangeLabel: string;
}) => {
  const change = deltaPct(props.cur, props.prev);

  if (props.prev === 0 && props.cur === 0) {
    return (
      <span className="inline-flex items-center gap-1 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={2.5} />
        No change
      </span>
    );
  }

  if (change === 0) {
    return (
      <span className="inline-flex items-center gap-1 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={2.5} />
        Flat vs previous {props.rangeLabel}
      </span>
    );
  }

  const up = change > 0;
  const good = up === props.upIsGood;
  const Icon = up ? TrendingUp : TrendingDown;

  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-1 text-micro font-semibold",
        good ? "text-emerald-700" : "text-red-700",
      )}
    >
      <Icon size={10} strokeWidth={2.5} />
      {up ? "+" : ""}
      {change}% vs previous {props.rangeLabel}
    </span>
  );
};

const Tile = (props: {
  label: string;
  value: number;
  suffix?: string;
  cur: number;
  prev: number;
  upIsGood: boolean;
  rangeLabel: string;
  points?: Point[];
  color: string;
  to: string;
  isLoading?: boolean;
}) => (
  <div className="relative flex flex-col gap-2 rounded-xl border border-slate-200 bg-white px-4 py-3.5 transition-shadow duration-150 hover:shadow-raised">
    <div className="flex items-baseline justify-between gap-2">
      <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
        {props.label}
      </span>
      {props.suffix && (
        <span className="text-micro font-normal tabular-nums text-slate-500">
          {props.suffix}
        </span>
      )}
    </div>

    {props.isLoading ? (
      <div className="h-7 w-20 animate-pulse rounded bg-slate-100" />
    ) : (
      <span className="text-2xl font-semibold leading-none text-slate-900">
        {compact(props.value)}
      </span>
    )}

    <Delta
      cur={props.cur}
      prev={props.prev}
      upIsGood={props.upIsGood}
      rangeLabel={props.rangeLabel}
    />

    <LineChart sparkline points={props.points} color={props.color} height={28} />

    <Link
      to={props.to}
      className="absolute inset-0 rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
      aria-label={`${props.label} — open logs`}
    />
  </div>
);

const toPoints = (
  datapoints?: { timestamp?: Timestamp; count: number | bigint }[],
): Point[] =>
  (datapoints ?? [])
    .filter((x) => !!x.timestamp)
    .map((x) => ({ ts: x.timestamp!, value: Number(x.count) }));

const ClusterHealth = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const interval = getAutoInterval(periodMinutes);
  const rangeLabel = periodLabel(periodMinutes);

  const accessCur = useQuery({
    queryKey: visibilityKeys.accessSummary(
      "current",
      periodMinutes,
      "all",
      NO_REFS,
    ),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const accessPrev = useQuery({
    queryKey: visibilityKeys.accessSummary(
      "previous",
      periodMinutes,
      "all",
      NO_REFS,
    ),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogSummary(
          GetAccessLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const accessPoints = useQuery({
    queryKey: visibilityKeys.accessDataPoint(periodMinutes, "all", NO_REFS),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAccessLog().getAccessLogDataPoint(
          GetAccessLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const authCur = useQuery({
    queryKey: visibilityKeys.authSummary("current", periodMinutes, NO_REFS),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const authPrev = useQuery({
    queryKey: visibilityKeys.authSummary("previous", periodMinutes, NO_REFS),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const authPoints = useQuery({
    queryKey: visibilityKeys.authDataPoint(periodMinutes, NO_REFS),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
          GetAuthenticationLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const auditCur = useQuery({
    queryKey: visibilityKeys.auditSummary("current", periodMinutes, {
      userRef: null,
      sessionRef: null,
      deviceRef: null,
      resourceRef: null,
    }),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const auditPrev = useQuery({
    queryKey: visibilityKeys.auditSummary("previous", periodMinutes, {
      userRef: null,
      sessionRef: null,
      deviceRef: null,
      resourceRef: null,
    }),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogSummary(
          GetAuditLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const auditPoints = useQuery({
    queryKey: visibilityKeys.auditDataPoint(periodMinutes, {
      userRef: null,
      sessionRef: null,
      deviceRef: null,
      resourceRef: null,
    }),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuditLog().getAuditLogDataPoint(
          GetAuditLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const componentCur = useQuery({
    queryKey: visibilityKeys.componentSummary("current", periodMinutes),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const componentPrev = useQuery({
    queryKey: visibilityKeys.componentSummary("previous", periodMinutes),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogSummary(
          GetComponentLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
          }),
        );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const sessionSummary = useQuery({
    queryKey: ["visibility", "core", "summary", "Session"],
    queryFn: async () => {
      const { response } = await getClientVisibilityCore().getSessionSummary(
        {},
      );
      return response;
    },
    refetchInterval: SUMMARY_REFETCH,
  });

  const componentErrorPoints = useQuery({
    queryKey: visibilityKeys.componentErrorDataPoint(periodMinutes),
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().getComponentLogDataPoint(
          GetComponentLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval,
            level: ComponentLog_Entry_Level.ERROR,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const requests = n(accessCur.data?.totalNumber);
  const requestsPrev = n(accessPrev.data?.totalNumber);
  const denied = n(accessCur.data?.totalDenied);
  const deniedPrev = n(accessPrev.data?.totalDenied);
  const connectedSessions = n(sessionSummary.data?.totalConnected);
  const auths = n(authCur.data?.totalNumber);
  const authsPrev = n(authPrev.data?.totalNumber);
  const changes = n(auditCur.data?.totalNumber);
  const changesPrev = n(auditPrev.data?.totalNumber);

  const errorsOf = (s?: {
    totalError?: unknown;
    totalPanic?: unknown;
    totalFatal?: unknown;
  }) => n(s?.totalError) + n(s?.totalPanic) + n(s?.totalFatal);
  const errors = errorsOf(componentCur.data);
  const errorsPrev = errorsOf(componentPrev.data);

  const allPoints = toPoints(accessPoints.data?.datapoints);

  return (
    <section
      aria-label="Cluster health"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <Tile
        label="Requests"
        value={requests}
        cur={requests}
        prev={requestsPrev}
        upIsGood
        rangeLabel={rangeLabel}
        points={allPoints}
        color={seriesColor(0)}
        to="/visibility/accesslogs"
        isLoading={accessCur.isLoading}
      />
      <Tile
        label="Denied"
        value={denied}
        suffix={`${pct(denied, requests)}%`}
        cur={denied}
        prev={deniedPrev}
        upIsGood={false}
        rangeLabel={rangeLabel}
        points={allPoints}
        color={STATUS_COLORS.critical}
        to="/visibility/accesslogs?status=DENIED"
        isLoading={accessCur.isLoading}
      />
      <Tile
        label="Connected sessions"
        value={connectedSessions}
        cur={connectedSessions}
        prev={connectedSessions}
        upIsGood
        rangeLabel={rangeLabel}
        color={seriesColor(2)}
        to="/core/sessions?isConnected=true"
        isLoading={sessionSummary.isLoading}
      />
      <Tile
        label="Authentications"
        value={auths}
        cur={auths}
        prev={authsPrev}
        upIsGood
        rangeLabel={rangeLabel}
        points={toPoints(authPoints.data?.datapoints)}
        color={seriesColor(1)}
        to="/visibility/authenticationlogs"
        isLoading={authCur.isLoading}
      />
      <Tile
        label="Audit logs"
        value={changes}
        cur={changes}
        prev={changesPrev}
        upIsGood
        rangeLabel={rangeLabel}
        points={toPoints(auditPoints.data?.datapoints)}
        color={seriesColor(5)}
        to="/visibility/auditlogs"
        isLoading={auditCur.isLoading}
      />
      <Tile
        label="Component errors"
        value={errors}
        cur={errors}
        prev={errorsPrev}
        upIsGood={false}
        rangeLabel={rangeLabel}
        points={toPoints(componentErrorPoints.data?.datapoints)}
        color={STATUS_COLORS.critical}
        to="/visibility/componentlogs"
        isLoading={componentCur.isLoading}
      />
    </section>
  );
};

export default ClusterHealth;
