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
  BookKey,
  Clock3,
  KeyRound,
  LaptopMinimal,
  RefreshCcw,
  ShieldQuestion,
  TriangleAlert,
} from "lucide-react";
import { useEnterpriseCreated, useEnterpriseTotals } from "./queries";

const Platform = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useEnterpriseCreated(periodMinutes, QUERY_PRIORITY.high);

  const stores = totals.data?.enterprise?.secretStore;
  const secrets = totals.data?.enterprise?.secret;
  const managers = totals.data?.enterprise?.deviceManager;
  const freshSecrets = created.data?.enterprise?.secret;

  return (
    <Panel
      icon={BookKey}
      title="Key management & device posture"
      description="Where the Cluster's secrets are wrapped and which external managers vouch for the Devices"
      to="/enterprise/secretstores"
      toLabel="Secret Stores"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Secret stores"
            value={n(stores?.totalNumber)}
            icon={BookKey}
            to="/enterprise/secretstores"
          />
          <MiniStat
            label="Stores healthy"
            value={n(stores?.totalOK)}
            tone="positive"
            icon={BookKey}
            to="/enterprise/secretstores?state=OK"
          />
          <MiniStat
            label="Sync failed"
            value={n(stores?.totalSynchronizationFailed)}
            tone={
              n(stores?.totalSynchronizationFailed) > 0 ? "critical" : "default"
            }
            icon={TriangleAlert}
            to="/enterprise/secretstores?synchronizationState=FAILED"
          />
          <MiniStat
            label="Secrets"
            value={n(secrets?.totalNumber)}
            icon={KeyRound}
            to="/enterprise/secrets"
          />
          <MiniStat
            label="Device managers"
            value={n(managers?.totalNumber)}
            icon={LaptopMinimal}
          />
          <MiniStat
            label="Waiting approval"
            value={n(managers?.totalWaitingApproval)}
            tone={n(managers?.totalWaitingApproval) > 0 ? "warning" : "default"}
            icon={Clock3}
            to="/core/devices?state=PENDING"
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Secret stores by backend"
            note={`${compact(n(stores?.totalNumber))} configured`}
          >
            <CompositionBar
              segments={[
                {
                  label: "HashiCorp Vault",
                  value: n(stores?.totalHashicorpVault),
                },
                {
                  label: "Azure Key Vault",
                  value: n(stores?.totalAzureKeyVault),
                },
                { label: "AWS KMS", value: n(stores?.totalAWSKMS) },
                { label: "GCP KMS", value: n(stores?.totalGCPKMS) },
                { label: "Kubernetes", value: n(stores?.totalKubernetes) },
              ]}
              total={n(stores?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Key re-wrapping"
            note={`${compact(n(stores?.totalLoading))} loading`}
          >
            <CompositionBar
              segments={[
                {
                  label: "In sync",
                  value: n(stores?.totalSynchronizationSuccess),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Running",
                  value: n(stores?.totalSynchronizing),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Failed",
                  value: n(stores?.totalSynchronizationFailed),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={n(stores?.totalNumber)}
            />
          </Breakdown>
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Device managers by vendor"
            note={`${compact(n(managers?.totalNumber))} integrated`}
          >
            <CompositionBar
              segments={[
                { label: "CrowdStrike", value: n(managers?.totalCrowdStrike) },
                { label: "SentinelOne", value: n(managers?.totalSentinelOne) },
                { label: "Intune", value: n(managers?.totalMicrosoftIntune) },
                { label: "Jamf Pro", value: n(managers?.totalJamfPro) },
                { label: "1Password", value: n(managers?.totalOnePassword) },
                { label: "FleetDM", value: n(managers?.totalFleetDM) },
                { label: "Huntress", value: n(managers?.totalHuntress) },
                { label: "Iru", value: n(managers?.totalIru) },
              ]}
              total={n(managers?.totalNumber)}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span>
                Healthy
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(managers?.totalOK))}
                </span>
              </span>
              <span>
                Degraded
                <span className="ml-1 font-semibold tabular-nums text-amber-700">
                  {compact(n(managers?.totalDegraded))}
                </span>
              </span>
              <span>
                Errored
                <span className="ml-1 font-semibold tabular-nums text-red-700">
                  {compact(n(managers?.totalError))}
                </span>
              </span>
              <span>
                Polling off
                <span className="ml-1 font-semibold tabular-nums text-slate-700">
                  {compact(n(managers?.totalPollingDisabled))}
                </span>
              </span>
            </div>
          </Breakdown>

          <Breakdown
            title="Managed devices linked to the Cluster"
            note={`${compact(n(managers?.totalManagedDevices))} managed`}
          >
            <Meter
              value={n(managers?.totalLinkedDevices)}
              total={n(managers?.totalManagedDevices)}
              label="linked to a Device"
              color={STATUS_COLORS.good}
            />
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-micro font-normal text-slate-500">
              <span className="inline-flex items-center gap-1">
                <ShieldQuestion size={11} strokeWidth={2.3} />
                Ambiguous
                <span className="font-semibold tabular-nums text-amber-700">
                  {compact(n(managers?.totalAmbiguous))}
                </span>
              </span>
              <span className="inline-flex items-center gap-1">
                <RefreshCcw size={11} strokeWidth={2.3} />
                Failed updates
                <span className="font-semibold tabular-nums text-red-700">
                  {compact(n(managers?.totalFailedUpdates))}
                </span>
              </span>
            </div>
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`New Secrets · ${rangeLabel}`}
            value={n(freshSecrets?.totalNumber)}
            prev={n(freshSecrets?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={KeyRound}
            to="/enterprise/secrets"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Platform;
