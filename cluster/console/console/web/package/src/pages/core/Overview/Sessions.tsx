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
  CircleCheck,
  CircleX,
  Clock3,
  Globe2,
  Laptop,
  Monitor,
  Terminal,
  Users,
  Wifi,
} from "lucide-react";
import { useCoreCreated, useCoreTotals } from "./queries";

const Sessions = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCoreCreated(periodMinutes, QUERY_PRIORITY.critical);

  const data = totals.data?.core?.session;
  const fresh = created.data?.core?.session;
  const total = n(data?.totalNumber);

  return (
    <Panel
      icon={Terminal}
      title="Sessions"
      description={`Who is connected right now and how the Cluster was joined over the last ${rangeLabel}`}
      to="/core/sessions"
      toLabel="All Sessions"
    >
      <div className="flex flex-col gap-5">
        <MiniStatGrid>
          <MiniStat
            label="Connected"
            value={n(data?.totalConnected)}
            tone="positive"
            icon={Wifi}
            to="/core/sessions?isConnected=true"
          />
          <MiniStat
            label="Active"
            value={n(data?.totalActive)}
            icon={CircleCheck}
            to="/core/sessions?state=ACTIVE"
          />
          <MiniStat
            label="Pending"
            value={n(data?.totalPending)}
            tone={n(data?.totalPending) > 0 ? "warning" : "default"}
            icon={Clock3}
            to="/core/sessions?state=PENDING"
          />
          <MiniStat
            label="Rejected"
            value={n(data?.totalRejected)}
            tone={n(data?.totalRejected) > 0 ? "critical" : "default"}
            icon={CircleX}
            to="/core/sessions?state=REJECTED"
          />
          <MiniStat
            label="Users"
            value={n(data?.totalUser)}
            icon={Users}
            to="/core/users"
          />
          <MiniStat
            label="Devices"
            value={n(data?.totalDevice)}
            icon={Laptop}
            to="/core/devices"
          />
        </MiniStatGrid>

        <Meter
          value={n(data?.totalConnected)}
          total={total}
          label="of all Sessions are connected"
          color={STATUS_COLORS.good}
        />

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="How the Cluster is joined"
            note={`${compact(total)} Sessions`}
          >
            <CompositionBar
              segments={[
                { label: "Client", value: n(data?.totalClient) },
                { label: "Browser", value: n(data?.totalClientlessBrowser) },
                { label: "SDK", value: n(data?.totalClientlessSDK) },
                { label: "OAuth2", value: n(data?.totalClientlessOAuth2) },
              ]}
              total={total}
            />
          </Breakdown>

          <Breakdown
            title="Lifecycle"
            note={`${compact(n(data?.totalActive))} active`}
          >
            <CompositionBar
              segments={[
                {
                  label: "Active",
                  value: n(data?.totalActive),
                  color: STATUS_COLORS.good,
                },
                {
                  label: "Pending",
                  value: n(data?.totalPending),
                  color: STATUS_COLORS.warning,
                },
                {
                  label: "Rejected",
                  value: n(data?.totalRejected),
                  color: STATUS_COLORS.critical,
                },
              ]}
              total={total}
            />
          </Breakdown>
        </div>

        <DeltaStatGrid>
          <DeltaStat
            label={`Created · ${rangeLabel}`}
            value={n(fresh?.totalNumber)}
            prev={n(fresh?.previous?.totalNumber)}
            rangeLabel={rangeLabel}
            icon={Terminal}
            to="/core/sessions"
          />
          <DeltaStat
            label="New client"
            value={n(fresh?.totalClient)}
            prev={n(fresh?.previous?.totalClient)}
            rangeLabel={rangeLabel}
            icon={Terminal}
            to="/core/sessions?type=CLIENT"
          />
          <DeltaStat
            label="New clientless"
            value={n(fresh?.totalClientless)}
            prev={n(fresh?.previous?.totalClientless)}
            rangeLabel={rangeLabel}
            icon={Globe2}
            to="/core/sessions?type=CLIENTLESS"
          />
          <DeltaStat
            label="New browser"
            value={n(fresh?.totalClientlessBrowser)}
            prev={n(fresh?.previous?.totalClientlessBrowser)}
            rangeLabel={rangeLabel}
            icon={Monitor}
            to="/core/sessions?isBrowser=true"
          />
          <DeltaStat
            label="New pending"
            value={n(fresh?.totalPending)}
            prev={n(fresh?.previous?.totalPending)}
            rangeLabel={rangeLabel}
            upIsGood={false}
            icon={Clock3}
            to="/core/sessions?state=PENDING"
          />
        </DeltaStatGrid>
      </div>
    </Panel>
  );
};

export default Sessions;
