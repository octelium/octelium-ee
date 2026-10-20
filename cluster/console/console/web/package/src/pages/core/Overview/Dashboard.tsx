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
  Gauge,
  Globe,
  Layers,
  PanelTop,
  ShieldUser,
  Terminal,
  User,
} from "lucide-react";
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
  const [customizing, setCustomizing] = React.useState(false);

  const widgets: DashboardWidget[] = [
    {
      id: "signals",
      title: "Signals",
      description: "Connected Sessions, logins, pending approvals and Devices",
      icon: Gauge,
      eager: true,
      render: () => <Signals periodMinutes={periodMinutes} />,
    },
    {
      id: "sessions",
      title: "Sessions",
      description: "Connected, active and pending Sessions and how they join",
      icon: Terminal,
      eager: true,
      render: () => <Sessions periodMinutes={periodMinutes} />,
    },
    {
      id: "authentication",
      title: "Authentication & assurance",
      description: "Logins, re-authentications, authenticators and assurance",
      icon: ShieldUser,
      height: 880,
      render: () => <Authentication periodMinutes={periodMinutes} />,
    },
    {
      id: "identities",
      title: "Identities & devices",
      description: "Users, Groups, Devices and Credentials",
      icon: User,
      height: 720,
      render: () => <Identities periodMinutes={periodMinutes} />,
    },
    {
      id: "services",
      title: "Services & exposure",
      description: "Services by mode, public exposure and authorization rules",
      icon: PanelTop,
      height: 720,
      render: () => <Services periodMinutes={periodMinutes} />,
    },
    {
      id: "dataplane",
      title: "Data plane",
      description: "Regions, Gateways and Secrets",
      icon: Globe,
      height: 420,
      render: () => <DataPlane periodMinutes={periodMinutes} />,
    },
    {
      id: "inventory",
      title: "Core inventory",
      description: "Counts of the Core resources",
      icon: Layers,
      height: 360,
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
        <Meta title="Core" />

        <DashboardControls
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          onCustomize={() => setCustomizing(true)}
        />

        <DashboardLayout
          page="core"
          widgets={widgets}
          customizing={customizing}
          onCustomizingChange={setCustomizing}
        />
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
