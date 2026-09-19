import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  Boxes,
  DoorClosed,
  Fingerprint,
  Globe,
  KeyRound,
  LaptopMinimal,
  LockKeyhole,
  LockOpen,
  PanelTop,
  Settings2,
  Shield,
  Terminal,
  User,
  Users,
} from "lucide-react";
import * as React from "react";
import Authentication from "./Authentication";
import DataPlane from "./DataPlane";
import Identities from "./Identities";
import Inventory from "./Inventory";
import Services from "./Services";
import Sessions from "./Sessions";
import Signals from "./Signals";

const SHORTCUTS = [
  { label: "Users", to: "/core/users", icon: User },
  { label: "Sessions", to: "/core/sessions", icon: Terminal },
  { label: "Devices", to: "/core/devices", icon: LaptopMinimal },
  { label: "Services", to: "/core/services", icon: PanelTop },
  { label: "Namespaces", to: "/core/namespaces", icon: Boxes },
  { label: "Policies", to: "/core/policies", icon: Shield },
  { label: "Groups", to: "/core/groups", icon: Users },
  { label: "Credentials", to: "/core/credentials", icon: LockOpen },
  {
    label: "Identity Providers",
    to: "/core/identityproviders",
    icon: Fingerprint,
  },
  { label: "Authenticators", to: "/core/authenticators", icon: LockKeyhole },
  { label: "Secrets", to: "/core/secrets", icon: KeyRound },
  { label: "Gateways", to: "/core/gateways", icon: DoorClosed },
  { label: "Regions", to: "/core/regions", icon: Globe },
  { label: "Cluster Config", to: "/core/clusterconfig", icon: Settings2 },
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
        <Meta title="Core" />

        <DashboardHeader
          title="Core"
          description={`Identities, Sessions, Devices and Services · counters and trends cover the last ${rangeLabel}`}
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Jump to a core resource"
        />

        <Signals periodMinutes={periodMinutes} />

        <Sessions periodMinutes={periodMinutes} />

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
