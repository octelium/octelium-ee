import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import {
  GetAuthenticationLogDataPointRequest,
  GetAuthenticationLogSummaryRequest,
  ListAuthenticationLogTopCredentialRequest,
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
  ArrowUpRight,
  ChevronDown,
  Fingerprint,
  KeyRound,
  Minus,
  Repeat2,
  ShieldUser,
  TrendingDown,
  TrendingUp,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import ChartPanel from "../Charts/ChartPanel";
import { LogWidgetHeader } from "../LogWidget";
import TopList from "../TopList";
import {
  ALL_PERIODS,
  buildTimestamps,
  deltaPct,
  EXTENDED_PERIODS,
  getAutoInterval,
  n,
  pct,
  periodLabel,
  PRIMARY_PERIODS,
  refKey,
  toTs,
  visibilityKeys,
} from "@/utils/visibility";
import PeriodSelector from "../LogWidget/PeriodSelector";

const TrendBadge = ({ cur, prev }: { cur: number; prev: number }) => {
  const d = deltaPct(cur, prev);
  if (d === 0 || prev === 0)
    return (
      <span className="inline-flex items-center gap-0.5 text-micro font-normal text-slate-500">
        <Minus size={10} strokeWidth={3} /> —
      </span>
    );
  const up = d > 0;
  return (
    <span
      className={twMerge(
        "inline-flex items-center gap-0.5 text-micro font-semibold",
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
  to,
}: {
  label: string;
  value: number;
  prevValue: number;
  icon: React.FC<any>;
  color: "slate" | "blue" | "violet" | "amber" | "teal";
  to?: string;
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

  const content = (
    <>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Icon size={13} className={palette.icon} strokeWidth={2.5} />
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {label}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <TrendBadge cur={value} prev={prevValue} />
          {to && (
            <ArrowUpRight
              size={11}
              className="text-slate-300 transition-colors duration-200 group-hover:text-slate-500"
            />
          )}
        </div>
      </div>
      <span
        className={twMerge("text-2xl font-bold tabular-nums", palette.value)}
      >
        {value.toLocaleString()}
      </span>
      <span className="text-micro font-normal text-slate-500">
        Previous period: {prevValue.toLocaleString()}
      </span>
    </>
  );

  const className = twMerge(
    "group flex min-h-[112px] flex-col gap-2.5 rounded-xl border p-3",
    palette.bg,
    palette.border,
    to &&
      "outline-none transition-[border-color,box-shadow] duration-200 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-blue-500/30",
  );

  return to ? (
    <Link to={to} className={className}>
      {content}
    </Link>
  ) : (
    <div className={className}>{content}</div>
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
    <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {label}
    </span>
    <div className="flex items-baseline gap-1.5">
      <span className="text-sm font-bold text-slate-700 tabular-nums">
        {value.toLocaleString()}
      </span>
      {total > 0 && (
        <span className="text-micro font-normal text-slate-500">
          {pct(value, total)}%
        </span>
      )}
    </div>
  </div>
);

interface AuthenticationLogHealthWidgetProps {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  identityProviderRef?: ObjectReference;
  credentialRef?: ObjectReference;
  authenticatorRef?: ObjectReference;
  periodMinutes?: number;
  onPeriodChange?: (value: number) => void;
  hideRangeControl?: boolean;
}

const getAuthenticationLogPath = (
  props: AuthenticationLogHealthWidgetProps,
) => {
  const params = new URLSearchParams();
  const refs = [
    ["userRef", props.userRef],
    ["sessionRef", props.sessionRef],
    ["deviceRef", props.deviceRef],
    ["identityProviderRef", props.identityProviderRef],
    ["credentialRef", props.credentialRef],
    ["authenticatorRef", props.authenticatorRef],
  ] as const;

  refs.forEach(([key, ref]) => {
    if (ref?.name) params.set(`${key}.name`, ref.name);
    else if (ref?.uid) params.set(`${key}.uid`, ref.uid);
  });

  const query = params.toString();
  return `/visibility/authenticationlogs${query ? `?${query}` : ""}`;
};

const AuthenticationLogHealthWidget = (
  props: AuthenticationLogHealthWidgetProps,
) => {
  const [localPeriodMinutes, setLocalPeriodMinutes] = useState(60);
  const periodMinutes = props.periodMinutes ?? localPeriodMinutes;
  const setPeriodMinutes = props.onPeriodChange ?? setLocalPeriodMinutes;
  const { curFrom, curTo, prevFrom, prevTo } = buildTimestamps(periodMinutes);
  const autoInterval = getAutoInterval(periodMinutes);
  const rangeLabel = periodLabel(periodMinutes);

  const refKeys = {
    userRef: refKey(props.userRef),
    sessionRef: refKey(props.sessionRef),
    deviceRef: refKey(props.deviceRef),
    identityProviderRef: refKey(props.identityProviderRef),
    credentialRef: refKey(props.credentialRef),
    authenticatorRef: refKey(props.authenticatorRef),
  };

  const showTopUsers = !props.userRef && !props.sessionRef && !props.deviceRef;
  const showTopIdentityProviders = !props.identityProviderRef;
  const showTopCredentials = !props.credentialRef;

  const curSummary = useQuery({
    queryKey: visibilityKeys.authSummary("current", periodMinutes, refKeys),
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
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const prevSummary = useQuery({
    queryKey: visibilityKeys.authSummary("previous", periodMinutes, refKeys),
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
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
          }),
        );
      return response;
    },
    refetchInterval: 60_000,
  });

  const dataPoint = useQuery({
    queryKey: visibilityKeys.authDataPoint(periodMinutes, refKeys),
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
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topUsers = useQuery({
    queryKey: ["authLogTopUser", periodMinutes, refKeys],
    enabled: showTopUsers,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopUser(
          ListAuthenticationLogTopUserRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            identityProviderRef: props.identityProviderRef,
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topIdentityProviders = useQuery({
    queryKey: ["authLogTopIdentityProvider", periodMinutes, refKeys],
    enabled: showTopIdentityProviders,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopIdentityProvider(
          ListAuthenticationLogTopIdentityProviderRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
            credentialRef: props.credentialRef,
            authenticatorRef: props.authenticatorRef,
          }),
        );
      return response;
    },
    refetchInterval: refetchIntervalChart,
  });

  const topCredentials = useQuery({
    queryKey: ["authLogTopCredential", periodMinutes, refKeys],
    enabled: showTopCredentials,
    queryFn: async () => {
      const { response } =
        await getClientVisibilityAuthenticationLog().listAuthenticationLogTopCredential(
          ListAuthenticationLogTopCredentialRequest.create({
            from: toTs(curFrom),
            to: toTs(curTo),
            userRef: props.userRef,
            sessionRef: props.sessionRef,
            deviceRef: props.deviceRef,
            identityProviderRef: props.identityProviderRef,
            authenticatorRef: props.authenticatorRef,
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
    topIdentityProviders.isLoading ||
    topCredentials.isLoading;
  const hasError = [
    curSummary,
    prevSummary,
    dataPoint,
    topUsers,
    topIdentityProviders,
    topCredentials,
  ].some((query) => query.isError);

  const updatedAt = Math.max(curSummary.dataUpdatedAt, dataPoint.dataUpdatedAt);

  const refetchAll = () => {
    curSummary.refetch();
    prevSummary.refetch();
    dataPoint.refetch();
    if (showTopUsers) topUsers.refetch();
    if (showTopIdentityProviders) topIdentityProviders.refetch();
    if (showTopCredentials) topCredentials.refetch();
  };

  return (
    <div className="flex w-full flex-col gap-4">
      <LogWidgetHeader
        icon={ShieldUser}
        title="Authentication activity"
        description={`Compared with the previous ${rangeLabel}`}
        isLoading={isAnyLoading}
        isError={hasError}
        updatedAt={updatedAt}
        onRefresh={refetchAll}
      >
        {!props.hideRangeControl && (
          <PeriodSelector value={periodMinutes} onChange={setPeriodMinutes} />
        )}
      </LogWidgetHeader>

      {hasError && (
        <div
          role="alert"
          className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-semibold text-amber-800"
        >
          Some authentication-log data could not be loaded. Showing the
          available results; try refreshing to retry.
        </div>
      )}

      {isSummaryLoading ? (
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-28 animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : cur && prev ? (
        <>
          <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              label="Total"
              value={n(cur.totalNumber)}
              prevValue={n(prev.totalNumber)}
              icon={Activity}
              color="slate"
              to={getAuthenticationLogPath(props)}
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
              <div className="flex flex-col gap-2 rounded-xl border border-slate-200 bg-white p-3">
                <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
                  Assurance level breakdown
                </span>
                <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
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
                          "h-full transition-[width] duration-200",
                          color,
                        )}
                        style={{ width: `${pctVal}%` }}
                      />
                    ))}
                  </div>
                  <div className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1">
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
                        className="flex items-center gap-1 text-micro font-normal text-slate-500"
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

              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
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

              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
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
                  <span className="text-xs font-semibold text-amber-700">
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
              <span className="text-body font-normal text-slate-500">
                No authentication events in this period
              </span>
            </div>
          )}
        </>
      ) : (
        <div className="flex items-center justify-center py-8">
          <span className="text-body font-normal text-slate-500">
            No data available
          </span>
        </div>
      )}

      <ChartPanel
        caption={`Activity — last ${rangeLabel}`}
        points={(dataPoint.data?.datapoints ?? []).map((x) => ({
          ts: x.timestamp!,
          value: x.count,
        }))}
      />

      {((showTopUsers && topUsers.data && topUsers.data?.items.length > 0) ||
        (showTopIdentityProviders &&
          topIdentityProviders.data &&
          topIdentityProviders.data?.items.length > 0) ||
        (showTopCredentials &&
          topCredentials.data &&
          topCredentials.data?.items.length > 0)) && (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {showTopUsers && topUsers.data && topUsers.data?.items.length > 0 && (
            <TopList
              title="Top Users"
              to={getAuthenticationLogPath(props)}
              items={topUsers.data.items.map((x) => ({
                resource: x.user!,
                count: x.count,
              }))}
            />
          )}
          {showTopIdentityProviders &&
            topIdentityProviders.data &&
            topIdentityProviders.data?.items.length > 0 && (
              <TopList
                title="Top Identity Providers"
                to={getAuthenticationLogPath(props)}
                items={topIdentityProviders.data.items.map((x) => ({
                  resource: x.identityProvider!,
                  count: x.count,
                }))}
              />
            )}
          {showTopCredentials &&
            topCredentials.data &&
            topCredentials.data?.items.length > 0 && (
              <TopList
                title="Top Credentials"
                to={getAuthenticationLogPath(props)}
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
