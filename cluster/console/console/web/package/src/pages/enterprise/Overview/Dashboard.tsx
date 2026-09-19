import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  BookKey,
  Crown,
  FlaskConical,
  Folder,
  Globe2,
  KeyRound,
  Settings2,
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

const SHORTCUTS = [
  { label: "Certificates", to: "/enterprise/certificates", icon: ShieldCheck },
  {
    label: "Certificate Issuers",
    to: "/enterprise/certificateissuers",
    icon: Crown,
  },
  {
    label: "Directory Providers",
    to: "/enterprise/directoryproviders",
    icon: Folder,
  },
  { label: "Secret Stores", to: "/enterprise/secretstores", icon: BookKey },
  { label: "Secrets", to: "/enterprise/secrets", icon: KeyRound },
  {
    label: "Collector Exporters",
    to: "/enterprise/collectorexporters",
    icon: Telescope,
  },
  { label: "DNS Providers", to: "/enterprise/dnsproviders", icon: Globe2 },
  {
    label: "Policy Tester",
    to: "/enterprise/policytester",
    icon: FlaskConical,
  },
  { label: "Cluster Config", to: "/enterprise/clusterconfig", icon: Settings2 },
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
        <Meta title="Enterprise" />

        <DashboardHeader
          title="Enterprise"
          description={`Certificates, identity directories, key management and telemetry integrations · deltas cover the last ${rangeLabel}`}
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Jump to an enterprise resource"
        />

        <Signals periodMinutes={periodMinutes} />

        <Health periodMinutes={periodMinutes} />

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
