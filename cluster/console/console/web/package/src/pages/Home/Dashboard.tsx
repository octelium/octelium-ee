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
  Gauge,
  HeartPulse,
  Layers,
  Library,
  ServerCog,
  ShieldAlert,
  ShieldUser,
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

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [customizing, setCustomizing] = React.useState(false);

  const widgets: DashboardWidget[] = [
    {
      id: "signals",
      title: "Signals",
      description: "Requests, errors, connected Sessions and access requests",
      icon: Gauge,
      eager: true,
      render: () => <Signals periodMinutes={periodMinutes} />,
    },
    {
      id: "attention",
      title: "Needs attention",
      description: "Conditions across Core, Access and Enterprise to act on",
      icon: ShieldAlert,
      eager: true,
      locked: true,
      render: () => <Attention periodMinutes={periodMinutes} />,
    },
    {
      id: "health",
      title: "Cluster health",
      description: "Health of the Cluster subsystems, components and Regions",
      icon: HeartPulse,
      height: 420,
      render: () => <Health periodMinutes={periodMinutes} />,
    },
    {
      id: "activity",
      title: "Access activity",
      description: "Authorization traffic, decisions and the top identities",
      icon: ActivityIcon,
      height: 760,
      render: () => <Activity periodMinutes={periodMinutes} />,
    },
    {
      id: "identity",
      title: "Identity & authentication",
      description: "Sign-ins, assurance levels and identity providers",
      icon: ShieldUser,
      height: 680,
      render: () => <Identity periodMinutes={periodMinutes} />,
    },
    {
      id: "governance",
      title: "Access governance",
      description: "Access requests, their deadlines and urgency",
      icon: UserCheck,
      height: 460,
      render: () => <Governance periodMinutes={periodMinutes} />,
    },
    {
      id: "inventory",
      title: "Cluster inventory",
      description: "Resource counts across every API",
      icon: Layers,
      height: 400,
      render: () => <Inventory periodMinutes={periodMinutes} />,
    },
    {
      id: "runtime",
      title: "Runtime & platform",
      description: "Request rate, latency, CPU and component logs",
      icon: ServerCog,
      height: 760,
      render: () => <Runtime periodMinutes={periodMinutes} />,
    },
    {
      id: "operations",
      title: "Configuration changes",
      description: "Recent configuration changes and the Cluster's version",
      icon: Library,
      height: 420,
      render: () => (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
          <Operations periodMinutes={periodMinutes} />
          <ClusterCard />
        </div>
      ),
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
        <Meta title="Overview" />

        <DashboardControls
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          onCustomize={() => setCustomizing(true)}
        />

        <DashboardLayout
          page="home"
          widgets={widgets}
          customizing={customizing}
          onCustomizingChange={setCustomizing}
        />
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
