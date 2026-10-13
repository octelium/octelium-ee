import { DashboardControls, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
import * as React from "react";
import Authentication from "./Authentication";
import DataPlane from "./DataPlane";
import Identities from "./Identities";
import Inventory from "./Inventory";
import Services from "./Services";
import Sessions from "./Sessions";
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
        <Meta title="Core" />

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
          <Sessions periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={880}>
          <Authentication periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={720}>
          <Identities periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={720}>
          <Services periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={420}>
          <DataPlane periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={360}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
