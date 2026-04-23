import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogSummaryRequest,
  ListAuthenticationLogTopIdentityProviderRequest,
  ListAuthenticationLogTopUserRequest,
} from "@/apis/visibilityv1/visibilityv1";
import {
  getClientVisibilityAuthenticationLog,
  refetchIntervalChart,
} from "@/utils/client";
import { Button, Menu } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  Activity,
  ChevronDown,
  Fingerprint,
  KeyRound,
  Minus,
  RefreshCw,
  Repeat2,
  ShieldUser,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import LineChart from "../Charts/LineChart";
import TopList from "../TopList";

interface PeriodOption {
  label: string;
  minutes: number;
}

const PRIMARY_PERIODS: PeriodOption[] = [
  { label: "30m", minutes: 30 },
  { label: "1h", minutes: 60 },
  { label: "3h", minutes: 180 },
  { label: "6h", minutes: 360 },
  { label: "12h", minutes: 720 },
  { label: "24h", minutes: 1440 },
];

const EXTENDED_PERIODS: PeriodOption[] = [
  { label: "5m", minutes: 5 },
  { label: "10m", minutes: 10 },
  { label: "15m", minutes: 15 },
  { label: "2d", minutes: 2880 },
  { label: "3d", minutes: 4320 },
  { label: "7d", minutes: 10080 },
  { label: "14d", minutes: 20160 },
];

const ALL_PERIODS = [...PRIMARY_PERIODS, ...EXTENDED_PERIODS];

const createDuration = (val: number, unit: string): Duration => {
  const typePayload = match(unit)
    .with("millisecond", () => ({
      oneofKind: "milliseconds" as const,
      milliseconds: val,
    }))
    .with("second", () => ({ oneofKind: "seconds" as const, seconds: val }))
    .with("minute", () => ({ oneofKind: "minutes" as const, minutes: val }))
    .with("hour", () => ({ oneofKind: "hours" as const, hours: val }))
    .with("day", () => ({ oneofKind: "days" as const, days: val }))
    .with("week", () => ({ oneofKind: "weeks" as const, weeks: val }))
    .with("month", () => ({ oneofKind: "months" as const, months: val }))
    .otherwise(() => ({ oneofKind: "seconds" as const, seconds: val }));
  return Duration.create({ type: typePayload as any });
};

const getAutoInterval = (periodMinutes: number): Duration => {
  if (periodMinutes <= 15) return createDuration(30, "second");
  if (periodMinutes <= 60) return createDuration(1, "minute");
  if (periodMinutes <= 180) return createDuration(5, "minute");
  if (periodMinutes <= 360) return createDuration(10, "minute");
  if (periodMinutes <= 720) return createDuration(15, "minute");
  if (periodMinutes <= 1440) return createDuration(30, "minute");
  if (periodMinutes <= 4320) return createDuration(1, "hour");
  if (periodMinutes <= 10080) return createDuration(3, "hour");
  return createDuration(6, "hour");
};

const buildTimestamps = (periodMinutes: number) => {
  const now = dayjs();
  const curFrom = now.subtract(periodMinutes, "minute").valueOf();
  const curTo = now.valueOf();
  const prevFrom = now.subtract(periodMinutes * 2, "minute").valueOf();
  const prevTo = curFrom;
  return { curFrom, curTo, prevFrom, prevTo };
};

const toTs = (ms: number) => Timestamp.fromDate(new Date(ms));

const n = (v: unknown) => Number(v ?? 0);

const deltaPct = (cur: number, prev: number) =>
  prev === 0 ? 0 : Math.round(((cur - prev) / prev) * 100);

const pct = (value: number, total: number) =>
  total === 0 ? 0 : Math.round((value / total) * 100);

const refKey = (ref?: ObjectReference) => ref?.uid ?? ref?.name ?? null;

const TrendBadge = ({ cur, prev }: { cur: number; prev: number }) => {
  const d = deltaPct(cur, prev);
  if (d === 0 || prev === 0)
    return (
      <span className="inline-flex items-center gap-0.5 text-[0.65rem] font-bold text-slate-400">
        <Minus size={10} strokeWidth={3} /> —
      </span>
    );
  const up = d > 0;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-[0.65rem] font-bold",
        up ? "text-emerald-600" : "text-red-500",
      )}
    >
      {up ? (
        <TrendingUp size={10} strokeWidth={2.5} />
      ) : (
        <TrendingDown size={10} strokeWidth={2.5} />
      )}
      {up ? "+" : ""}
      {d}%
    </span>
  );
};

const StatCard = ({
  label,
  value,
  prevValue,
  icon: Icon,
  color,
}: {
  label: string;
  value: number;
  prevValue: number;
  icon: React.FC<any>;
  color: "slate" | "blue" | "violet" | "amber" | "teal";
}) => {
  const palette = {
    slate: {
      bg: "bg-slate-50",
      border: "border-slate-200",
      icon: "text-slate-500",
      value: "text-slate-700",
    },
    blue: {
      bg: "bg-blue-50",
      border: "border-blue-100",
      icon: "text-blue-600",
      value: "text-blue-700",
    },
    violet: {
      bg: "bg-violet-50",
      border: "border-violet-100",
      icon: "text-violet-600",
      value: "text-violet-700",
    },
    amber: {
      bg: "bg-amber-50",
      border: "border-amber-100",
      icon: "text-amber-600",
      value: "text-amber-700",
    },
    teal: {
      bg: "bg-teal-50",
      border: "border-teal-100",
      icon: "text-teal-600",
      value: "text-teal-700",
    },
  }[color];

  return (
    <div
      className={twMerge(
        "flex flex-col gap-2.5 p-4 rounded-xl border",
        palette.bg,
        palette.border,
      )}
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Icon size={13} className={palette.icon} strokeWidth={2.5} />
          <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <TrendBadge cur={value} prev={prevValue} />
      </div>
      <span
        className={twMerge("text-2xl font-bold tabular-nums", palette.value)}
      >
        {value.toLocaleString()}
      </span>
      <span className="text-[0.63rem] font-semibold text-slate-400">
        prev: {prevValue.toLocaleString()}
      </span>
    </div>
  );
};

const MiniStat = ({
  label,
  value,
  total,
}: {
  label: string;
  value: number;
  total: number;
}) => (
  <div className="flex flex-col gap-1 px-3 py-2.5 rounded-lg border border-slate-200 bg-white">
    <span className="text-[0.6rem] font-bold uppercase tracking-[0.07em] text-slate-400">
      {label}
    </span>
    <div className="flex items-baseline gap-1.5">
      <span className="text-[0.9rem] font-bold text-slate-700 tabular-nums">
        {value.toLocaleString()}
      </span>
      {total > 0 && (
        <span className="text-[0.63rem] font-semibold text-slate-400">
          {pct(value, total)}%
        </span>
      )}
    </div>
    {/**
    
     {total > 0 && (
      <div className="h-0.5 w-full bg-slate-100 rounded-full overflow-hidden mt-0.5">
        <div
          className="h-full bg-slate-400 rounded-full transition-[width] duration-500"
          style={{ width: `${pct(value, total)}%` }}
        />
      </div>
    )}
    **/}
  </div>
);

const PeriodSelector = ({
  value,
  onChange,
}: {
  value: number;
  onChange: (v: number) => void;
}) => {
  const isExtended = EXTENDED_PERIODS.some((p) => p.minutes === value);
  const extendedLabel = isExtended
    ? ALL_PERIODS.find((p) => p.minutes === value)?.label
    : undefined;

  return (
    <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
      {PRIMARY_PERIODS.map((opt) => {
        const active = opt.minutes === value;
        return (
          <Button
            key={opt.minutes}
            onClick={() => onChange(opt.minutes)}
            styles={{
              root: {
                height: "26px",
                fontSize: "0.7rem",
                fontWeight: 700,
                fontFamily: "Ubuntu, sans-serif",
                padding: "0 10px",
                backgroundColor: active ? "#0f172a" : "#ffffff",
                color: active ? "#ffffff" : "#64748b",
                border: "none",
                borderRadius: 0,
                transition: "background-color 150ms, color 150ms",
                "&:hover": {
                  backgroundColor: active ? "#1e293b" : "#f8fafc",
                  color: active ? "#ffffff" : "#0f172a",
                },
              },
            }}
          >
            {opt.label}
          </Button>
        );
      })}

      <Menu position="bottom-end" offset={4} withArrow={false}>
        <Menu.Target>
          <Button
            styles={{
              root: {
                height: "26px",
                fontSize: "0.7rem",
                fontWeight: 700,
                fontFamily: "Ubuntu, sans-serif",
                padding: "0 8px",
                backgroundColor: isExtended ? "#0f172a" : "#ffffff",
                color: isExtended ? "#ffffff" : "#64748b",
                border: "none",
                borderLeft: "1px solid #e2e8f0",
                borderRadius: 0,
                transition: "background-color 150ms, color 150ms",
                "&:hover": {
                  backgroundColor: isExtended ? "#1e293b" : "#f8fafc",
                  color: isExtended ? "#ffffff" : "#0f172a",
                },
              },
            }}
          >
            <span className="flex items-center gap-1">
              {extendedLabel ?? "More"}
              <ChevronDown size={10} strokeWidth={2.5} />
            </span>
          </Button>
        </Menu.Target>
        <Menu.Dropdown>
          <div className="flex flex-col py-1 min-w-[100px]">
            {EXTENDED_PERIODS.map((opt) => (
              <button
                key={opt.minutes}
                onClick={() => onChange(opt.minutes)}
                className={twMerge(
                  "flex items-center px-3 h-8 text-[0.75rem] font-bold cursor-pointer transition-colors duration-100 text-left",
                  opt.minutes === value
                    ? "bg-slate-900 text-white"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900",
                )}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </Menu.Dropdown>
      </Menu>
    </Button.Group>
  );
};

interface AuthenticationLogHealthWidgetProps {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
}

const AuthenticationLogHealthWidget = (
  props: AuthenticationLogHealthWidgetProps,
) => {
  const [periodMinutes, setPeriodMinutes] = useState(60);
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const periodLabel =
    ALL_PERIODS.find((o) => o.minutes === periodMinutes)?.label ?? "";

  const refKeys = {
    userRef: refKey(props.userRef),
    sessionRef: refKey(props.sessionRef),
    deviceRef: refKey(props.deviceRef),
    identityProviderRef: refKey(props.identityProviderRef),
  };

  const curSummary = useQuery({
    queryKey: ["authLogSummary", "current", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const prevSummary = useQuery({
    queryKey: ["authLogSummary", "previous", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogSummary(
          GetAuthenticationLogSummaryRequest.create({
            from: toTs(prevFrom),
            to: toTs(prevTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const dataPoint = useQuery({
    queryKey: ["authLogDataPoint", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().getAuthenticationLogDataPoint(
          GetAuthenticationLogDataPointRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            interval: autoInterval,
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topUsers = useQuery({
    queryKey: ["authLogTopUser", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopUser(
          ListAuthenticationLogTopUserRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),

            identityProviderRef: props.identityProviderRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topIdentityProviders = useQuery({
    queryKey: ["authLogTopIdentityProvider", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopIdentityProvider(
          ListAuthenticationLogTopIdentityProviderRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topCredentials = useQuery({
    queryKey: ["authLogTopCredential", periodMinutes, refKeys],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopCredential(
          ListAuthenticationLogTopIdentityProviderRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const isSummaryLoading = curSummary.isLoading || prevSummary.isLoading;
  const cur = curSummary.data;
  const prev = prevSummary.data;

  const isAnyLoading =
    isSummaryLoading ||
    dataPoint.isLoading ||
    topUsers.isLoading ||
    topIdentityProviders.isLoading;

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    topUsers.refetch();
    topIdentityProviders.refetch();
    topCredentials.refetch();
  };

  return (
    <div className="w-full flex flex-col gap-5">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
          <span className="text-[0.68rem] font-semibold text-slate-400">
            vs previous {periodLabel}
          </span>
        </div>
        <button
          onClick={refetchAll}
          disabled={isAnyLoading}
          className="flex items-center justify-center w-7 h-7 rounded-md text-slate-400 border border-slate-200 bg-white hover:text-slate-700 hover:border-slate-300 hover:bg-slate-50 transition-colors duration-150 cursor-pointer shadow-[0_1px_2px_rgba(15,23,42,0.05)] disabled:opacity-50"
        >
          <RefreshCw
            size={12}
            strokeWidth={2.5}
            className={isAnyLoading ? "animate-spin" : ""}
          />
        </button>
      </div>

      {isSummaryLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-[120px] rounded-xl border border-slate-200 bg-slate-50 animate-pulse"
            />
          ))}
        </div>
      ) : cur && prev ? (
        <>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <StatCard
              label="Total"
              value={n(cur.totalNumber)}
              prevValue={n(prev.totalNumber)}
              icon={Activity}
              color="slate"
            />
            <StatCard
              label="Identity Provider"
              value={n(cur.totalIdentityProvider)}
              prevValue={n(prev.totalIdentityProvider)}
              icon={ShieldUser}
              color="blue"
            />
            <StatCard
              label="Authenticator"
              value={n(cur.totalAuthenticator)}
              prevValue={n(prev.totalAuthenticator)}
              icon={Fingerprint}
              color="violet"
            />
            <StatCard
              label="Credential"
              value={n(cur.totalCredential)}
              prevValue={n(prev.totalCredential)}
              icon={KeyRound}
              color="amber"
            />
          </div>

          {n(cur.totalNumber) > 0 && (
            <>
              <div className="flex flex-col gap-2 p-4 rounded-xl border border-slate-200 bg-white">
                <span className="text-[0.68rem] font-bold uppercase tracking-[0.06em] text-slate-400">
                  Assurance level breakdown
                </span>
                <div className="flex items-center gap-2">
                  <div className="flex-1 h-2 rounded-full bg-slate-100 overflow-hidden flex">
                    {[
                      {
                        pctVal: pct(n(cur.totalAAL1), n(cur.totalNumber)),
                        color: "bg-blue-400",
                      },
                      {
                        pctVal: pct(n(cur.totalAAL2), n(cur.totalNumber)),
                        color: "bg-violet-500",
                      },
                      {
                        pctVal: pct(n(cur.totalAAL3), n(cur.totalNumber)),
                        color: "bg-teal-500",
                      },
                    ].map(({ pctVal, color }, i) => (
                      <div
                        key={i}
                        className={twMerge(
                          "h-full transition-[width] duration-500",
                          color,
                        )}
                        style={{ width: `${pctVal}%` }}
                      />
                    ))}
                  </div>
                  <div className="flex items-center gap-3 shrink-0">
                    {[
                      {
                        label: "AAL1",
                        value: n(cur.totalAAL1),
                        color: "bg-blue-400",
                      },
                      {
                        label: "AAL2",
                        value: n(cur.totalAAL2),
                        color: "bg-violet-500",
                      },
                      {
                        label: "AAL3",
                        value: n(cur.totalAAL3),
                        color: "bg-teal-500",
                      },
                    ].map(({ label, value, color }) => (
                      <span
                        key={label}
                        className="flex items-center gap-1 text-[0.65rem] font-bold text-slate-500"
                      >
                        <span
                          className={twMerge(
                            "w-2 h-2 rounded-full shrink-0",
                            color,
                          )}
                        />
                        {label}: {value.toLocaleString()}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <MiniStat
                  label="Passkey"
                  value={n(cur.totalAuthenticatorPasskey)}
                  total={n(cur.totalAuthenticator)}
                />
                <MiniStat
                  label="MFA"
                  value={n(cur.totalAuthenticatorMFA)}
                  total={n(cur.totalAuthenticator)}
                />
                <MiniStat
                  label="FIDO"
                  value={n(cur.totalAuthenticatorFIDO)}
                  total={n(cur.totalAuthenticator)}
                />
                <MiniStat
                  label="TOTP"
                  value={n(cur.totalAuthenticatorTOTP)}
                  total={n(cur.totalAuthenticator)}
                />
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
                <MiniStat
                  label="Sessions"
                  value={n(cur.totalSession)}
                  total={n(cur.totalNumber)}
                />
                <MiniStat
                  label="Users"
                  value={n(cur.totalUser)}
                  total={n(cur.totalNumber)}
                />
                <MiniStat
                  label="Refresh tokens"
                  value={n(cur.totalRefreshToken)}
                  total={n(cur.totalNumber)}
                />
              </div>

              {n(cur.totalReauthentication) > 0 && (
                <div className="flex items-center gap-2 px-3 py-2.5 rounded-lg border border-amber-200 bg-amber-50">
                  <Repeat2
                    size={13}
                    className="text-amber-600 shrink-0"
                    strokeWidth={2.5}
                  />
                  <span className="text-[0.72rem] font-semibold text-amber-700">
                    <span className="font-bold">
                      {n(cur.totalReauthentication).toLocaleString()}
                    </span>{" "}
                    re-authentication
                    {n(cur.totalReauthentication) !== 1 ? "s" : ""} in this
                    period
                  </span>
                </div>
              )}
            </>
          )}

          {n(cur.totalNumber) === 0 && (
            <div className="flex items-center justify-center py-8">
              <span className="text-[0.75rem] font-semibold text-slate-400">
                No authentication events in this period
              </span>
            </div>
          )}
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-[0.75rem] font-semibold text-slate-400">
            No data available
          </span>
        </div>
      )}

      {dataPoint.data?.datapoints && dataPoint.data.datapoints.length > 0 && (
        <div className="rounded-xl border border-slate-200 bg-white p-4">
          <span className="text-[0.62rem] font-bold uppercase tracking-[0.07em] text-slate-400 block mb-3">
            Activity — last {periodLabel}
          </span>
          <LineChart
            points={dataPoint.data.datapoints.map((x) => ({
              ts: x.timestamp!,
              value: x.count,
            }))}
          />
        </div>
      )}

      {((topUsers.data && topUsers.data?.items.length > 0) ||
        (topIdentityProviders.data &&
          topIdentityProviders.data?.items.length > 0) ||
        (topCredentials.data && topCredentials.data?.items.length > 0)) && (
        <div className="grid grid-cols-2 gap-4">
          {topUsers.data && topUsers.data?.items.length > 0 && (
            <TopList
              title="Top Users"
              to="/visibility/authenticationlogs"
              items={topUsers.data.items.map((x) => ({
                resource: x.user!,
                count: x.count,
              }))}
            />
          )}
          {topIdentityProviders.data &&
            topIdentityProviders.data?.items.length > 0 && (
              <TopList
                title="Top Identity Providers"
                to="/visibility/authenticationlogs"
                items={topIdentityProviders.data.items.map((x) => ({
                  resource: x.identityProvider!,
                  count: x.count,
                }))}
              />
            )}
          {topCredentials.data && topCredentials.data?.items.length > 0 && (
            <TopList
              title="Top Credentials"
              to="/visibility/authenticationlogs"
              items={topCredentials.data.items.map((x) => ({
                resource: x.credential!,
                count: x.count,
              }))}
            />
          )}
        </div>
      )}
    </div>
  );
};

export default AuthenticationLogHealthWidget;
