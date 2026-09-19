import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { STATUS_COLORS, useChartColorScheme } from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Bot,
  Fingerprint,
  LaptopMinimal,
  LockOpen,
  User,
  Users,
  UserX,
} from "lucide-react";
import { useCoreCreated, useCoreTotals } from "./queries";

const Identities = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCoreCreated(periodMinutes, QUERY_PRIORITY.high);

  const users = totals.data?.core?.user;
  const groups = totals.data?.core?.group;
  const devices = totals.data?.core?.device;
  const credentials = totals.data?.core?.credential;
  const providers = totals.data?.core?.identityProvider;

  const newUsers = created.data?.core?.user;
  const newDevices = created.data?.core?.device;
  const newCredentials = created.data?.core?.credential;

  return (
    <Panel
      icon={User}
      title="Identities & devices"
      description="Who and what can reach the Cluster, and how they prove it"
      to="/core/users"
      toLabel="All Users"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Users"
            value={n(users?.totalNumber)}
            icon={User}
            to="/core/users"
          />
          <MiniStat
            label="Workloads"
            value={n(users?.totalWorkload)}
            icon={Bot}
            to="/core/users?type=WORKLOAD"
          />
          <MiniStat
            label="Deactivated"
            value={n(users?.totalDisabled)}
            tone={n(users?.totalDisabled) > 0 ? "warning" : "default"}
            icon={UserX}
            to="/core/users?isDisabled=true"
          />
          <MiniStat
            label="Groups"
            value={n(groups?.totalNumber)}
            icon={Users}
            to="/core/groups"
          />
          <MiniStat
            label="Credentials"
            value={n(credentials?.totalNumber)}
            icon={LockOpen}
            to="/core/credentials"
          />
          <MiniStat
            label="Providers"
            value={n(providers?.totalNumber)}
            icon={Fingerprint}
            to="/core/identityproviders"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 xl:grid-cols-4">
          <Breakdown
            title="User types"
            note={`${compact(n(users?.totalNumber))} total`}
          >
            <CompositionBar
              segments={[
                { label: "Humans", value: n(users?.totalHuman) },
                { label: "Workloads", value: n(users?.totalWorkload) },
              ]}
              total={n(users?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Devices by platform"
            note={`${compact(n(devices?.totalNumber))} enrolled`}
          >
            <CompositionBar
              segments={[
                { label: "Linux", value: n(devices?.totalLinux) },
                { label: "Windows", value: n(devices?.totalWindows) },
                { label: "macOS", value: n(devices?.totalMac) },
                { label: "Android", value: n(devices?.totalAndroid) },
                { label: "iOS", value: n(devices?.totalIOS) },
              ]}
              total={n(devices?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Credentials"
            note={`${compact(n(credentials?.totalUser))} users`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Auth token",
                  value: n(credentials?.totalAuthenticationToken),
                },
                {
                  label: "Access token",
                  value: n(credentials?.totalAccessToken),
                },
                { label: "OAuth2", value: n(credentials?.totalOAuth2) },
              ]}
              total={n(credentials?.totalNumber)}
            />
            {n(credentials?.totalDisabled) > 0 && (
              <span className="text-micro font-semibold text-amber-700">
                {compact(n(credentials?.totalDisabled))} disabled
              </span>
            )}
          </Breakdown>

          <Breakdown
            title="Identity providers"
            note={
              n(providers?.totalDisabled) > 0
                ? `${compact(n(providers?.totalDisabled))} disabled`
                : undefined
            }
          >
            <CompositionBar
              segments={[
                { label: "OIDC", value: n(providers?.totalOIDC) },
                { label: "SAML", value: n(providers?.totalSAML) },
                { label: "GitHub", value: n(providers?.totalGithub) },
                {
                  label: "ID token",
                  value: n(providers?.totalOIDCIdentityToken),
                },
              ]}
              total={n(providers?.totalNumber)}
            />
          </Breakdown>
        </div>

        <Breakdown
          title="Device enrolment"
          note={`${compact(n(devices?.totalUser))} users with a Device`}
        >
          <CompositionBar
            segments={[
              {
                label: "Approved",
                value: n(devices?.totalActive),
                color: STATUS_COLORS.good,
              },
              {
                label: "Pending",
                value: n(devices?.totalPending),
                color: STATUS_COLORS.warning,
              },
              {
                label: "Rejected",
                value: n(devices?.totalRejected),
                color: STATUS_COLORS.critical,
              },
            ]}
            total={n(devices?.totalNumber)}
          />
        </Breakdown>

        <DeltaStatGrid>
          <DeltaStat
            label={`New Users · ${rangeLabel}`}
            value={n(newUsers?.totalNumber)}
            prev={n(newUsers?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={User}
            to="/core/users"
          />
          <DeltaStat
            label="New humans"
            value={n(newUsers?.totalHuman)}
            prev={n(newUsers?.previous?.totalHuman)}
            rangeLabel={rangeLabel}
            icon={User}
            to="/core/users?type=HUMAN"
          />
          <DeltaStat
            label="New workloads"
            value={n(newUsers?.totalWorkload)}
            prev={n(newUsers?.previous?.totalWorkload)}
            rangeLabel={rangeLabel}
            icon={Bot}
            to="/core/users?type=WORKLOAD"
          />
          <DeltaStat
            label="New Devices"
            value={n(newDevices?.totalNumber)}
            prev={n(newDevices?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={LaptopMinimal}
            to="/core/devices"
          />
          <DeltaStat
            label="New Credentials"
            value={n(newCredentials?.totalNumber)}
            prev={n(newCredentials?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={LockOpen}
            to="/core/credentials"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Identities;
