import {
  Breakdown,
  DeltaStat,
  DeltaStatGrid,
  Meter,
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
  Folder,
  PowerOff,
  RefreshCcw,
  TriangleAlert,
  UserRound,
  UsersRound,
} from "lucide-react";
import { useEnterpriseCreated, useEnterpriseTotals } from "./queries";

const Directories = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useEnterpriseCreated(periodMinutes, QUERY_PRIORITY.high);

  const providers = totals.data?.enterprise?.directoryProvider;
  const users = totals.data?.enterprise?.directoryProviderUser;
  const groups = totals.data?.enterprise?.directoryProviderGroup;
  const freshUsers = created.data?.enterprise?.directoryProviderUser;
  const freshGroups = created.data?.enterprise?.directoryProviderGroup;

  return (
    <Panel
      icon={Folder}
      title="Identity directories"
      description={`External identity sources and the Users and Groups they keep in sync · deltas cover the last ${rangeLabel}`}
      to="/enterprise/directoryproviders"
      toLabel="Directory Providers"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Providers"
            value={n(providers?.totalNumber)}
            icon={Folder}
            to="/enterprise/directoryproviders"
          />
          <MiniStat
            label="Synchronizing"
            value={n(providers?.totalSynchronizing)}
            icon={RefreshCcw}
            to="/enterprise/directoryproviders?synchronizationState=SYNCING"
          />
          <MiniStat
            label="Sync failed"
            value={n(providers?.totalSynchronizationFailed)}
            tone={
              n(providers?.totalSynchronizationFailed) > 0
                ? "critical"
                : "default"
            }
            icon={TriangleAlert}
            to="/enterprise/directoryproviders?synchronizationState=FAILED"
          />
          <MiniStat
            label="Disabled"
            value={n(providers?.totalDisabled)}
            tone={n(providers?.totalDisabled) > 0 ? "warning" : "default"}
            icon={PowerOff}
            to="/enterprise/directoryproviders?isDisabled=true"
          />
          <MiniStat
            label="Directory users"
            value={n(users?.totalNumber)}
            icon={UserRound}
          />
          <MiniStat
            label="Directory groups"
            value={n(groups?.totalNumber)}
            icon={UsersRound}
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Providers by type"
            note={`${compact(n(providers?.totalNumber))} configured`}
          >
            <CompositionBar
              segments={[
                { label: "SCIM", value: n(providers?.totalSCIM) },
                {
                  label: "Google Workspace",
                  value: n(providers?.totalGoogleWorkspace),
                },
                { label: "Keycloak", value: n(providers?.totalKeycloak) },
              ]}
              total={n(providers?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Synchronization"
            note={`${compact(n(providers?.totalSynchronizationSuccess))} healthy`}
          >
            <CompositionBar
              segments={[
                {
                  label: "In sync",
                  value: n(providers?.totalSynchronizationSuccess),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Running",
                  value: n(providers?.totalSynchronizing),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Failed",
                  value: n(providers?.totalSynchronizationFailed),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={n(providers?.totalNumber)}
            />
          </Breakdown>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Directory users linked to Cluster Users"
            note={`${compact(n(users?.totalDirectoryProvider))} providers`}
          >
            <Meter
              value={n(users?.totalUser)}
              total={n(users?.totalNumber)}
              label="linked"
              color={STATUS_COLORS.good}
            />
          </Breakdown>

          <Breakdown
            title="Directory groups linked to Cluster Groups"
            note={`${compact(n(groups?.totalDirectoryProvider))} providers`}
          >
            <Meter
              value={n(groups?.totalGroup)}
              total={n(groups?.totalNumber)}
              label="linked"
              color={STATUS_COLORS.good}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`New directory users · ${rangeLabel}`}
            value={n(freshUsers?.totalNumber)}
            prev={n(freshUsers?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={UserRound}
          />
          <DeltaStat
            label="Newly linked users"
            value={n(freshUsers?.totalUser)}
            prev={n(freshUsers?.previous?.totalUser)}
            rangeLabel={rangeLabel}
            icon={UserRound}
            to="/core/users"
          />
          <DeltaStat
            label="New directory groups"
            value={n(freshGroups?.totalNumber)}
            prev={n(freshGroups?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={UsersRound}
          />
          <DeltaStat
            label="Newly linked groups"
            value={n(freshGroups?.totalGroup)}
            prev={n(freshGroups?.previous?.totalGroup)}
            rangeLabel={rangeLabel}
            icon={UsersRound}
            to="/core/groups"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Directories;
