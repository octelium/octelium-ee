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
  BookKey,
  Folder,
  Gauge,
  HeartPulse,
  Layers,
  ShieldCheck,
  Telescope,
} from "lucide-react";
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
  const [customizing, setCustomizing] = React.useState(false);

  const widgets: DashboardWidget[] = [
    {
      id: "signals",
      title: "Signals",
      description:
        "Certificates, directories, secret stores and device managers",
      icon: Gauge,
      eager: true,
      render: () => <Signals periodMinutes={periodMinutes} />,
    },
    {
      id: "health",
      title: "Integration health",
      description: "Health of the Enterprise integrations",
      icon: HeartPulse,
      eager: true,
      locked: true,
      render: () => <Health periodMinutes={periodMinutes} />,
    },
    {
      id: "certificates",
      title: "Certificates",
      description: "Issuance, expiry and the Certificates to renew first",
      icon: ShieldCheck,
      height: 820,
      render: () => <Certificates periodMinutes={periodMinutes} />,
    },
    {
      id: "directories",
      title: "Identity directories",
      description: "Directory providers, synchronization and linked identities",
      icon: Folder,
      height: 760,
      render: () => <Directories periodMinutes={periodMinutes} />,
    },
    {
      id: "platform",
      title: "Key management & device posture",
      description: "Secret stores, key re-wrapping and device managers",
      icon: BookKey,
      height: 720,
      render: () => <Platform periodMinutes={periodMinutes} />,
    },
    {
      id: "telemetry",
      title: "Telemetry & DNS",
      description: "Collector exporters and DNS providers",
      icon: Telescope,
      height: 420,
      render: () => <Telemetry periodMinutes={periodMinutes} />,
    },
    {
      id: "inventory",
      title: "Enterprise inventory",
      description: "Counts of the Enterprise resources",
      icon: Layers,
      height: 320,
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
        <Meta title="Enterprise" />

        <DashboardControls
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          onCustomize={() => setCustomizing(true)}
        />

        <DashboardLayout
          page="enterprise"
          widgets={widgets}
          customizing={customizing}
          onCustomizingChange={setCustomizing}
        />
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
