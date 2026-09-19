import { StatTile, toPoints } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  useAuthDataPoint,
  useAuthSummary,
} from "@/components/Dashboard/queries";
import {
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { AUTH_BY_ASSURANCE, useCoreCreated, useCoreTotals } from "./queries";

const Signals = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCoreCreated(periodMinutes, QUERY_PRIORITY.critical);
  const auth = useAuthSummary(periodMinutes, QUERY_PRIORITY.high);
  const authPoints = useAuthDataPoint(
    periodMinutes,
    QUERY_PRIORITY.high,
    AUTH_BY_ASSURANCE,
  );

  const sessions = totals.data?.core?.session;
  const users = totals.data?.core?.user;
  const devices = totals.data?.core?.device;
  const services = totals.data?.core?.service;
  const authenticators = totals.data?.core?.authenticator;

  const newSessions = created.data?.core?.session;
  const newUsers = created.data?.core?.user;
  const newDevices = created.data?.core?.device;

  const pending =
    n(sessions?.totalPending) +
    n(devices?.totalPending) +
    n(authenticators?.totalPending);
  const enrollable =
    n(sessions?.totalNumber) +
    n(devices?.totalNumber) +
    n(authenticators?.totalNumber);

  const logins = n(auth.data?.totalNumber);
  const strongLogins = n(auth.data?.totalAAL2) + n(auth.data?.totalAAL3);

  return (
    <section
      aria-label="Core signals"
      className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6"
    >
      <StatTile
        label="Connected"
        value={n(sessions?.totalConnected)}
        badge={`${compact(n(sessions?.totalNumber))} sessions`}
        footer={`${compact(n(sessions?.totalUser))} users · ${compact(n(sessions?.totalDevice))} devices`}
        bar={{
          value: n(sessions?.totalConnected),
          total: n(sessions?.totalNumber),
          label: "connected right now",
        }}
        color={seriesColor(2)}
        to="/core/sessions?isConnected=true"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label={`New sessions · ${rangeLabel}`}
        value={n(newSessions?.totalNumber)}
        trend={{
          cur: n(newSessions?.totalNumber),
          prev: n(newSessions?.previous?.totalNumber),
          upIsGood: true,
          rangeLabel,
        }}
        footer={`${compact(n(newSessions?.totalClient))} client · ${compact(n(newSessions?.totalClientless))} clientless`}
        bar={{
          value: n(newSessions?.totalClient),
          total: n(newSessions?.totalNumber),
          label: "client-based",
        }}
        color={seriesColor(0)}
        to="/core/sessions"
        isLoading={created.isLoading}
        isError={created.isError}
        hasData={created.data !== undefined}
      />

      <StatTile
        label="Logins"
        value={logins}
        badge={
          logins > 0
            ? `${Math.round((strongLogins / logins) * 100)}% MFA`
            : undefined
        }
        footer={`${compact(n(auth.data?.totalUser))} users · ${compact(n(auth.data?.totalReauthentication))} re-auths`}
        trend={{
          cur: logins,
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
        label="Users"
        value={n(users?.totalNumber)}
        badge={
          n(newUsers?.totalNumber) > 0
            ? `+${compact(n(newUsers?.totalNumber))}`
            : undefined
        }
        footer={`${compact(n(users?.totalWorkload))} workloads · ${compact(n(users?.totalDisabled))} deactivated`}
        bar={{
          value: n(users?.totalHuman),
          total: n(users?.totalNumber),
          label: "humans",
        }}
        color={seriesColor(3)}
        to="/core/users"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Devices"
        value={n(devices?.totalNumber)}
        badge={
          n(newDevices?.totalNumber) > 0
            ? `+${compact(n(newDevices?.totalNumber))}`
            : undefined
        }
        footer={`${compact(n(devices?.totalUser))} users enrolled · ${compact(n(services?.totalNumber))} Services reachable`}
        bar={{
          value: n(devices?.totalActive),
          total: n(devices?.totalNumber),
          label: "approved",
        }}
        color={seriesColor(6)}
        to="/core/devices"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />

      <StatTile
        label="Awaiting approval"
        value={pending}
        footer={`${compact(n(sessions?.totalPending))} sessions · ${compact(n(devices?.totalPending))} devices · ${compact(n(authenticators?.totalPending))} authenticators`}
        bar={{
          value: pending,
          total: enrollable,
          label: "of all enrolments",
        }}
        color={pending > 0 ? STATUS_COLORS.warning : STATUS_COLORS.good}
        to="/core/devices?state=PENDING"
        isLoading={totals.isLoading}
        isError={totals.isError}
        hasData={totals.data !== undefined}
      />
    </section>
  );
};

export default Signals;
