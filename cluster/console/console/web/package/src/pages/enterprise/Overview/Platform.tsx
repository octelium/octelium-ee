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
  BookKey,
  KeyRound,
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
  const freshSecrets = created.data?.enterprise?.secret;

  return (
    <Panel
      icon={BookKey}
      title="Key management"
      description="Where the Cluster's secrets are wrapped and how their keys are re-wrapped"
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
