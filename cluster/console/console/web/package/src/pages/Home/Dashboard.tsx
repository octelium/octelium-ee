import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { getDomain } from "@/utils";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  Building2,
  ChartNoAxesCombined,
  Cpu,
  Eye,
  Settings2,
  UserCheck,
} from "lucide-react";
import * as React from "react";
import Activity from "./Activity";
import Attention from "./Attention";
import ClusterCard from "./ClusterCard";
import Governance from "./Governance";
import Health from "./Health";
import Identity from "./Identity";
import Inventory from "./Inventory";
import Operations from "./Operations";
import Runtime from "./Runtime";
import Signals from "./Signals";

const SHORTCUTS = [
  { label: "Core", to: "/core", icon: Cpu },
  { label: "Access", to: "/access", icon: UserCheck },
  { label: "Enterprise", to: "/enterprise", icon: Building2 },
  { label: "Visibility", to: "/visibility", icon: Eye },
  { label: "Metrics", to: "/visibility/metrics", icon: ChartNoAxesCombined },
  { label: "Cluster", to: "/clusterman", icon: Settings2 },
];

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const rangeLabel = periodLabel(periodMinutes);

  return (
    <AutoRefreshContext.Provider value={autoRefresh}>
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="flex w-full flex-col gap-4 py-4"
      >
        <Meta title="Overview" />

        <DashboardHeader
          title="Cluster overview"
          description={
            <>
              Security, identity and platform posture for{" "}
              <span className="font-semibold text-slate-700">
                {getDomain()}
              </span>{" "}
              · every panel below covers the last {rangeLabel}
            </>
          }
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Jump to an API"
        />

        <Signals periodMinutes={periodMinutes} />

        <Attention periodMinutes={periodMinutes} />

        <Deferred height={420}>
          <Health periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={760}>
          <Activity periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={680}>
          <Identity periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={460}>
          <Governance periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={400}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={760}>
          <Runtime periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={420}>
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <Operations periodMinutes={periodMinutes} />
            <ClusterCard />
          </div>
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
