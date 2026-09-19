import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  CalendarClock,
  ClipboardCheck,
  Inbox,
  Layers,
  Shield,
  Timer,
} from "lucide-react";
import * as React from "react";
import Grants from "./Grants";
import Inventory from "./Inventory";
import Queue from "./Queue";
import Requests from "./Requests";
import Signals from "./Signals";
import Workflow from "./Workflow";

const SHORTCUTS = [
  { label: "Requests", to: "/access/requests", icon: Inbox },
  { label: "Reviews", to: "/access/reviews", icon: ClipboardCheck },
  { label: "Policies", to: "/access/policies", icon: Shield },
  { label: "Catalogs", to: "/access/catalogs", icon: Layers },
  { label: "Pending", to: "/access/requests?state=PENDING", icon: Timer },
  {
    label: "Past deadline",
    to: "/access/requests?isDeadlinePassed=true",
    icon: CalendarClock,
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
        <Meta title="Access" />

        <DashboardHeader
          title="Access"
          description={`Just-in-time access Requests, reviews and the Policies behind them · deltas cover the last ${rangeLabel}`}
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Jump to an access resource"
        />

        <Signals periodMinutes={periodMinutes} />

        <Queue periodMinutes={periodMinutes} />

        <Deferred height={620}>
          <Requests periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={620}>
          <Workflow periodMinutes={periodMinutes} />
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
