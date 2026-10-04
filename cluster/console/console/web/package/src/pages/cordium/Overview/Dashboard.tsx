import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  AlertTriangle,
  CirclePlay,
  Container,
  GitBranch,
  Hammer,
  Layers,
  LayoutTemplate,
  UsersRound,
} from "lucide-react";
import * as React from "react";
import Attention from "./Attention";
import Capacity from "./Capacity";
import Collaboration from "./Collaboration";
import Inventory from "./Inventory";
import Signals from "./Signals";
import Templates from "./Templates";
import Workspaces from "./Workspaces";

const SHORTCUTS = [
  { label: "Workspaces", to: "/cordium/workspaces", icon: Container },
  { label: "Templates", to: "/cordium/templates", icon: LayoutTemplate },
  { label: "Spaces", to: "/cordium/spaces", icon: Layers },
  { label: "Memberships", to: "/cordium/memberships", icon: UsersRound },
  { label: "Git Providers", to: "/cordium/gitproviders", icon: GitBranch },
  {
    label: "Running",
    to: "/cordium/workspaces?state=RUNNING",
    icon: CirclePlay,
  },
  {
    label: "Failed",
    to: "/cordium/workspaces?isFailed=true",
    icon: AlertTriangle,
  },
  {
    label: "Building Templates",
    to: "/cordium/templates?isBuilding=true",
    icon: Hammer,
  },
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
        <Meta title="Cordium" />

        <DashboardHeader
          title="Cordium"
          description={`Workspaces, the Templates they are built from and the Spaces that own them · deltas cover the last ${rangeLabel}`}
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Jump to a Cordium resource"
        />

        <Signals periodMinutes={periodMinutes} />

        <Attention periodMinutes={periodMinutes} />

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
