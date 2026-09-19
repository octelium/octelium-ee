import SeriesChart from "@/components/Charts/SeriesChart";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import TopList from "@/components/TopList";
import { seriesColor, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Fingerprint,
  KeyRound,
  RefreshCcw,
  ShieldUser,
  Terminal,
  User,
} from "lucide-react";
import { MiniStat, MiniStatGrid, Panel, toPoints } from "./components";
import { useAuthDataPoint, useAuthSummary, useAuthTop } from "./queries";

const Identity = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const summary = useAuthSummary(
    periodMinutes,
    "current",
    QUERY_PRIORITY.critical,
  );
  const dataPoints = useAuthDataPoint(periodMinutes, QUERY_PRIORITY.high);
  const topUsers = useAuthTop("user", periodMinutes, QUERY_PRIORITY.normal);
  const topProviders = useAuthTop("identityProvider", periodMinutes);

  const data = summary.data;
  const assurance = [
    { label: "AAL1", value: n(data?.totalAAL1) },
    { label: "AAL2", value: n(data?.totalAAL2) },
    { label: "AAL3", value: n(data?.totalAAL3) },
  ];
  const assuranceTotal = assurance.reduce((sum, item) => sum + item.value, 0);

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
      title="Identity & authentication"
      description={`Logins, re-authentications and assurance levels in the last ${rangeLabel}`}
      to="/visibility/authenticationlogs"
      toLabel="Authentication logs"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Logins"
            value={n(data?.totalNumber)}
            icon={ShieldUser}
            to="/visibility/authenticationlogs"
          />
          <MiniStat
            label="Users"
            value={n(data?.totalUser)}
            icon={User}
            to="/core/users"
          />
          <MiniStat
            label="Sessions"
            value={n(data?.totalSession)}
            icon={Terminal}
            to="/core/sessions"
          />
          <MiniStat
            label="Re-auths"
            value={n(data?.totalReauthentication)}
            icon={RefreshCcw}
          />
          <MiniStat
            label="Credentials"
            value={n(data?.totalCredential)}
            icon={KeyRound}
            to="/core/credentials"
          />
          <MiniStat
            label="Providers"
            value={n(data?.totalIdentityProvider)}
            icon={Fingerprint}
            to="/core/identityproviders"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Assurance levels
            </span>
            <CompositionBar segments={assurance} total={assuranceTotal} />
          </div>

          <div className="flex flex-col gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
              Authenticators used
            </span>
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
                MFA
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {n(data?.totalAuthenticatorMFA).toLocaleString()}
                </span>
              </span>
              <span>
                Passkeys
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {n(data?.totalAuthenticatorPasskey).toLocaleString()}
                </span>
              </span>
              <span>
                Refresh tokens
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {n(data?.totalRefreshToken).toLocaleString()}
                </span>
              </span>
            </div>
          </div>
        </div>

        <SeriesChart
          height={260}
          colors={[seriesColor(1)]}
          series={[
            {
              name: "Authentications",
              points: toPoints(dataPoints.data?.datapoints),
            },
          ]}
          emptyLabel={`No authentications in the last ${rangeLabel}`}
        />

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
                to="/visibility/authenticationlogs"
                items={providerItems}
              />
            )}
          </div>
        )}
      </div>
    </Panel>
  );
};

export default Identity;
