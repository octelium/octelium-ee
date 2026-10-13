import { DashboardControls, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
import * as React from "react";
import Certificates from "./Certificates";
import Directories from "./Directories";
import Health from "./Health";
import Inventory from "./Inventory";
import Platform from "./Platform";
import Signals from "./Signals";
import Telemetry from "./Telemetry";

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
        <Meta title="Enterprise" />

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
          <Health periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={820}>
          <Certificates periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={760}>
          <Directories periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={720}>
          <Platform periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={420}>
          <Telemetry periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={320}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
