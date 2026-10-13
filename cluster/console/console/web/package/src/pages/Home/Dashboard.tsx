import { DashboardControls, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
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

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);

  return (
    <AutoRefreshContext.Provider value={autoRefresh}>
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="flex w-full flex-col gap-4 py-4"
      >
        <Meta title="Overview" />

        <DashboardControls
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
        />

        <Deferred eager>
          <Signals periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred eager>
          <Attention periodMinutes={periodMinutes} />
        </Deferred>

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
