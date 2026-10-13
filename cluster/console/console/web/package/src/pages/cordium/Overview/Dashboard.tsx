import { DashboardControls, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
import * as React from "react";
import Attention from "./Attention";
import Capacity from "./Capacity";
import Collaboration from "./Collaboration";
import Inventory from "./Inventory";
import Signals from "./Signals";
import Templates from "./Templates";
import Workspaces from "./Workspaces";

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
        <Meta title="Cordium" />

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

        <Deferred height={620}>
          <Workspaces periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={300}>
          <Capacity periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={560}>
          <Templates periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={560}>
          <Collaboration periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={280}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
