import { DashboardControls, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
import * as React from "react";
import Grants from "./Grants";
import Integrations from "./Integrations";
import Inventory from "./Inventory";
import Queue from "./Queue";
import Requests from "./Requests";
import Signals from "./Signals";
import Workflow from "./Workflow";

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
        <Meta title="Access" />

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
          <Queue periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={620}>
          <Requests periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={620}>
          <Workflow periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={620}>
          <Integrations periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={420}>
          <Grants periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={280}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
