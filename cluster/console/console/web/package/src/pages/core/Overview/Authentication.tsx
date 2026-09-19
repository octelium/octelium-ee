import SeriesChart from "@/components/Charts/SeriesChart";
import {
  Breakdown,
  MiniStat,
  MiniStatGrid,
  Panel,
  seriesPoints,
} from "@/components/Dashboard/components";
import {
  useAuthDataPoint,
  useAuthSummary,
  useAuthTop,
} from "@/components/Dashboard/queries";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TopList from "@/components/TopList";
import {
  STATUS_COLORS,
  seriesColor,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Clock3,
  KeyRound,
  LockKeyhole,
  RefreshCcw,
  ShieldUser,
  Usb,
} from "lucide-react";
import { AUTH_BY_ASSURANCE, useCoreTotals } from "./queries";

const Authentication = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const summary = useAuthSummary(periodMinutes, QUERY_PRIORITY.high);
  const dataPoints = useAuthDataPoint(
    periodMinutes,
    QUERY_PRIORITY.high,
    AUTH_BY_ASSURANCE,
  );
  const topUsers = useAuthTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const topProviders = useAuthTop("identityProvider", periodMinutes);

  const data = summary.data;
  const registered = totals.data?.core?.authenticator;
  const logins = n(data?.totalNumber);

  const userItems = (topUsers.data?.items ?? [])
    .map((item: any) => ({ resource: item.user, count: item.count }))
    .filter((item) => !!item.resource);
  const providerItems = (topProviders.data?.items ?? [])
    .map((item: any) => ({
      resource: item.identityProvider,
      count: item.count,
    }))
    .filter((item) => !!item.resource);

  return (
    <Panel
      icon={ShieldUser}
      title="Authentication & assurance"
      description={`Registered authenticators and the factors that were actually used over the last ${rangeLabel}`}
      to="/core/authenticators"
      toLabel="Authenticators"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Logins"
            value={logins}
            icon={ShieldUser}
            to="/visibility/authenticationlogs"
          />
          <MiniStat
            label="Re-auths"
            value={n(data?.totalReauthentication)}
            icon={RefreshCcw}
          />
          <MiniStat
            label="AAL3 logins"
            value={n(data?.totalAAL3)}
            tone="positive"
            icon={Usb}
          />
          <MiniStat
            label="Registered"
            value={n(registered?.totalNumber)}
            icon={LockKeyhole}
            to="/core/authenticators"
          />
          <MiniStat
            label="Passkeys"
            value={n(registered?.totalFIDOIsPasskey)}
            icon={KeyRound}
          />
          <MiniStat
            label="Pending"
            value={n(registered?.totalPending)}
            tone={n(registered?.totalPending) > 0 ? "warning" : "default"}
            icon={Clock3}
            to="/core/authenticators?state=PENDING"
          />
        </MiniStatGrid>

        <SeriesChart
          height={280}
          stacked
          colors={[STATUS_COLORS.warning, seriesColor(0), STATUS_COLORS.good]}
          series={[
            {
              name: "AAL1",
              points: seriesPoints(dataPoints.data?.series, "AAL1"),
            },
            {
              name: "AAL2",
              points: seriesPoints(dataPoints.data?.series, "AAL2"),
            },
            {
              name: "AAL3",
              points: seriesPoints(dataPoints.data?.series, "AAL3"),
            },
          ]}
          emptyLabel={`No authentications in the last ${rangeLabel}`}
        />

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Authenticators registered"
            note={`${compact(n(registered?.totalUser))} users`}
          >
            <CompositionBar
              segments={[
                { label: "FIDO", value: n(registered?.totalFIDO) },
                { label: "TPM", value: n(registered?.totalTPM) },
                { label: "TOTP", value: n(registered?.totalTOTP) },
              ]}
              total={n(registered?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Platform
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(registered?.totalFIDOPlatform))}
                </span>
              </span>
              <span>
                Roaming
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(registered?.totalFIDORoaming))}
                </span>
              </span>
              <span>
                Hardware-backed
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(registered?.totalFIDOIsHardware))}
                </span>
              </span>
              <span>
                Devices
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(registered?.totalDevice))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title={`Factors used · ${rangeLabel}`}
            note={`${compact(n(data?.totalAuthenticatorMFA))} MFA logins`}
          >
            <CompositionBar
              segments={[
                { label: "FIDO", value: n(data?.totalAuthenticatorFIDO) },
                { label: "TPM", value: n(data?.totalAuthenticatorTPM) },
                { label: "TOTP", value: n(data?.totalAuthenticatorTOTP) },
              ]}
              total={
                n(data?.totalAuthenticatorFIDO) +
                n(data?.totalAuthenticatorTPM) +
                n(data?.totalAuthenticatorTOTP)
              }
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Passkeys
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(data?.totalAuthenticatorPasskey))}
                </span>
              </span>
              <span>
                Refresh tokens
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(data?.totalRefreshToken))}
                </span>
              </span>
              <span>
                Credentials
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(data?.totalCredential))}
                </span>
              </span>
              <span>
                Providers
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(data?.totalIdentityProvider))}
                </span>
              </span>
            </div>
          </Breakdown>
        </div>

        <Breakdown
          title="Assurance levels reached"
          note={`${compact(logins)} logins`}
        >
          <CompositionBar
            segments={[
              {
                label: "AAL1",
                value: n(data?.totalAAL1),
                color: STATUS_COLORS.warning,
              },
              {
                label: "AAL2",
                value: n(data?.totalAAL2),
                color: seriesColor(0),
              },
              {
                label: "AAL3",
                value: n(data?.totalAAL3),
                color: STATUS_COLORS.good,
              },
            ]}
            total={logins}
          />
        </Breakdown>

        {(userItems.length > 0 || providerItems.length > 0) && (
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
            {userItems.length > 0 && (
              <TopList
                title="Top Users"
                to="/visibility/authenticationlogs"
                items={userItems}
              />
            )}
            {providerItems.length > 0 && (
              <TopList
                title="Top Identity Providers"
                to="/core/identityproviders"
                items={providerItems}
              />
            )}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Authentication;
