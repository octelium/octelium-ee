import { DashboardControls } from "@/components/Dashboard/components";
import {
  DashboardLayout,
  DashboardWidget,
} from "@/components/Dashboard/layout";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { motion } from "framer-motion";
import {
  Activity as ActivityIcon,
  ClipboardCheck,
  Gauge,
  Hourglass,
  Layers,
  Plug,
  ShieldAlert,
} from "lucide-react";
import * as React from "react";
import Activity from "./Activity";
import Attention from "./Attention";
import Governance from "./Governance";
import Integrations from "./Integrations";
import Inventory from "./Inventory";
import { LiveAccess, Pending } from "./Queues";
import Signals from "./Signals";

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [customizing, setCustomizing] = React.useState(false);

  const widgets: DashboardWidget[] = [
    {
      id: "signals",
      title: "Signals",
      description: "Pending requests, decisions, live access and deliveries",
      icon: Gauge,
      eager: true,
      render: () => <Signals periodMinutes={periodMinutes} />,
    },
    {
      id: "attention",
      title: "Needs attention",
      description: "Conditions of the access workflow to act on",
      icon: ShieldAlert,
      eager: true,
      locked: true,
      render: () => <Attention periodMinutes={periodMinutes} />,
    },
    {
      id: "queues",
      title: "Waiting for a decision & live access",
      description: "The pending requests and the access that is granted now",
      icon: Hourglass,
      eager: true,
      render: () => (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <Pending periodMinutes={periodMinutes} />
          <LiveAccess periodMinutes={periodMinutes} />
        </div>
      ),
    },
    {
      id: "activity",
      title: "Request activity",
      description: "Requests, decisions, time to decision and top requesters",
      icon: ActivityIcon,
      height: 760,
      render: () => <Activity periodMinutes={periodMinutes} />,
    },
    {
      id: "governance",
      title: "Reviews & policies",
      description: "Reviews, rule effects, reviewers and matching Policies",
      icon: ClipboardCheck,
      height: 560,
      render: () => <Governance periodMinutes={periodMinutes} />,
    },
    {
      id: "integrations",
      title: "Integrations",
      description: "Integration health and failing deliveries",
      icon: Plug,
      height: 420,
      render: () => <Integrations periodMinutes={periodMinutes} />,
    },
    {
      id: "inventory",
      title: "Access inventory",
      description: "Counts of the access resources",
      icon: Layers,
      height: 280,
      render: () => <Inventory periodMinutes={periodMinutes} />,
    },
  ];

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
          onCustomize={() => setCustomizing(true)}
        />

        <DashboardLayout
          page="access"
          widgets={widgets}
          customizing={customizing}
          onCustomizingChange={setCustomizing}
        />
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
