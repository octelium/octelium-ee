import { DashboardHeader, Deferred } from "@/components/Dashboard/components";
import {
  AutoRefreshContext,
  useDashboardRange,
} from "@/components/Dashboard/utils";
import Meta from "@/components/Meta";
import { getDomain } from "@/utils";
import { periodLabel } from "@/utils/visibility";
import { motion } from "framer-motion";
import {
  Activity,
  Bot,
  ChartNoAxesCombined,
  Library,
  ScrollText,
  ShieldUser,
  TerminalSquare,
} from "lucide-react";
import * as React from "react";
import { Navigate, useSearchParams } from "react-router-dom";
import AccessLeaders from "./AccessLeaders";
import Breakdown from "./Breakdown";
import ChangeLeaders from "./ChangeLeaders";
import Components from "./Components";
import IdentityLeaders from "./IdentityLeaders";
import Streams from "./Streams";

const SHORTCUTS = [
  { label: "Access logs", to: "/visibility/accesslogs", icon: Activity },
  {
    label: "Authentication logs",
    to: "/visibility/authenticationlogs",
    icon: ShieldUser,
  },
  { label: "Audit logs", to: "/visibility/auditlogs", icon: Library },
  {
    label: "Component logs",
    to: "/visibility/componentlogs",
    icon: ScrollText,
  },
  {
    label: "Metrics",
    to: "/visibility/metrics",
    icon: ChartNoAxesCombined,
  },
  { label: "LLM", to: "/visibility/llm", icon: Bot },
  { label: "SSH sessions", to: "/visibility/ssh", icon: TerminalSquare },
];

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const rangeLabel = periodLabel(periodMinutes);
  const [searchParams] = useSearchParams();

  if (searchParams.get("tab") === "metrics") {
    return <Navigate to="/visibility/metrics" replace />;
  }

  return (
    <AutoRefreshContext.Provider value={autoRefresh}>
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="flex w-full flex-col gap-4 py-4"
      >
        <Meta title="Visibility" />

        <DashboardHeader
          title="Visibility"
          description={
            <>
              Explore every log stream of{" "}
              <span className="font-semibold text-slate-700">
                {getDomain()}
              </span>{" "}
              · everything below covers the last {rangeLabel}
            </>
          }
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          shortcuts={SHORTCUTS}
          shortcutsLabel="Open a log explorer"
        />

        <Streams periodMinutes={periodMinutes} />

        <Breakdown periodMinutes={periodMinutes} />

        <Deferred height={620}>
          <Components periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={900}>
          <AccessLeaders periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={520}>
          <IdentityLeaders periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={720}>
          <ChangeLeaders periodMinutes={periodMinutes} />
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
