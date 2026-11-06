import {
  Breakdown,
  Meter,
  MiniStat,
  MiniStatGrid,
  Panel,
} from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import {
  CHART_INK,
  seriesColor,
  STATUS_COLORS,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  AlertTriangle,
  CircleX,
  LaptopMinimalCheck,
  Link2,
  MonitorSmartphone,
  RefreshCcw,
  ShieldAlert,
  ShieldQuestion,
  Swords,
  UserX,
} from "lucide-react";
import { useEnterpriseTotals } from "./queries";

const MANAGERS = "/enterprise/devicemanagers";
const DEVICES = "/core/devices";

const DeviceManagers = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  useChartColorScheme();

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const managers = totals.data?.enterprise?.deviceManager;
  const unbound =
    n(managers?.totalAmbiguous) +
    n(managers?.totalConflicts) +
    n(managers?.totalOwnerMismatches);

  return (
    <Panel
      icon={LaptopMinimalCheck}
      title="Device posture"
      description="The external managers that vouch for the Devices, and how their inventories link to the Cluster"
      to={MANAGERS}
      toLabel="Device Managers"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Device managers"
            value={n(managers?.totalNumber)}
            icon={LaptopMinimalCheck}
            to={MANAGERS}
          />
          <MiniStat
            label="Healthy"
            value={n(managers?.totalOK)}
            tone="positive"
            icon={LaptopMinimalCheck}
            to={`${MANAGERS}?state=OK`}
          />
          <MiniStat
            label="Degraded"
            value={n(managers?.totalDegraded)}
            tone={n(managers?.totalDegraded) > 0 ? "warning" : "default"}
            icon={AlertTriangle}
            to={`${MANAGERS}?state=DEGRADED`}
          />
          <MiniStat
            label="Errored"
            value={n(managers?.totalError)}
            tone={n(managers?.totalError) > 0 ? "critical" : "default"}
            icon={CircleX}
            to={`${MANAGERS}?state=ERROR`}
          />
          <MiniStat
            label="Managed devices"
            value={n(managers?.totalManagedDevices)}
            icon={MonitorSmartphone}
          />
          <MiniStat
            label="Linked Devices"
            value={n(managers?.totalLinkedDevices)}
            icon={Link2}
            to={`${DEVICES}?bindingState=ACCEPTED`}
          />
        </MiniStatGrid>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Managers by vendor"
            note={`${compact(n(managers?.totalNumber))} integrated`}
          >
            <CompositionBar
              segments={[
                { label: "CrowdStrike", value: n(managers?.totalCrowdStrike) },
                { label: "SentinelOne", value: n(managers?.totalSentinelOne) },
                { label: "Intune", value: n(managers?.totalMicrosoftIntune) },
                { label: "Jamf Pro", value: n(managers?.totalJamfPro) },
                { label: "1Password", value: n(managers?.totalOnePassword) },
                { label: "Fleet", value: n(managers?.totalFleetDM) },
                { label: "Huntress", value: n(managers?.totalHuntress) },
                { label: "Iru", value: n(managers?.totalIru) },
              ]}
              total={n(managers?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="Collection health"
            note={`${compact(n(managers?.totalOK))} healthy`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Healthy",
                  value: n(managers?.totalOK),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Loading",
                  value: n(managers?.totalLoading),
                  color: seriesColor(0),
                },
                {
                  label: "Degraded",
                  value: n(managers?.totalDegraded),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Error",
                  value: n(managers?.totalError),
                  color: STATUS_COLORS.critical,
                },
                {
                  label: "Polling off",
                  value: n(managers?.totalPollingDisabled),
                  color: CHART_INK.muted,
                },
              ]}
              total={n(managers?.totalNumber)}
            />
          </Breakdown>
        </div>

        <Breakdown
          title="Managed devices linked to the Cluster"
          note={`${compact(n(managers?.totalManagedDevices))} managed · ${compact(unbound)} not bound · ${compact(n(managers?.totalSuspended))} suspended`}
        >
          <Meter
            value={n(managers?.totalLinkedDevices)}
            total={n(managers?.totalManagedDevices)}
            label="linked to a Device"
            color={STATUS_COLORS.good}
          />
          <MiniStatGrid>
            <MiniStat
              label="Ambiguous"
              value={n(managers?.totalAmbiguous)}
              tone={n(managers?.totalAmbiguous) > 0 ? "warning" : "default"}
              icon={ShieldQuestion}
              to={`${DEVICES}?bindingState=AMBIGUOUS`}
            />
            <MiniStat
              label="Conflicts"
              value={n(managers?.totalConflicts)}
              tone={n(managers?.totalConflicts) > 0 ? "warning" : "default"}
              icon={Swords}
              to={`${DEVICES}?bindingState=CONFLICT`}
            />
            <MiniStat
              label="Owner mismatches"
              value={n(managers?.totalOwnerMismatches)}
              tone={
                n(managers?.totalOwnerMismatches) > 0 ? "critical" : "default"
              }
              icon={UserX}
              to={`${DEVICES}?bindingState=OWNER_MISMATCH`}
            />
            <MiniStat
              label="Suspended"
              value={n(managers?.totalSuspended)}
              tone={n(managers?.totalSuspended) > 0 ? "critical" : "default"}
              icon={ShieldAlert}
              to={`${DEVICES}?hasInvalidBinding=true`}
            />
            <MiniStat
              label="Failed updates"
              value={n(managers?.totalFailedUpdates)}
              tone={
                n(managers?.totalFailedUpdates) > 0 ? "critical" : "default"
              }
              icon={RefreshCcw}
              to={`${MANAGERS}?hasLinkingIssues=true`}
            />
          </MiniStatGrid>
        </Breakdown>
      </div>
    </Panel>
  );
};

export default DeviceManagers;
