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
  ChartSpline,
  Library,
  ScrollText,
  ShieldUser,
  Trophy,
  Waves,
} from "lucide-react";
import * as React from "react";
import { Navigate, useSearchParams } from "react-router-dom";
import AccessLeaders from "./AccessLeaders";
import Breakdown from "./Breakdown";
import ChangeLeaders from "./ChangeLeaders";
import Components from "./Components";
import IdentityLeaders from "./IdentityLeaders";
import Streams from "./Streams";

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const [customizing, setCustomizing] = React.useState(false);
  const [searchParams] = useSearchParams();

  if (searchParams.get("tab") === "metrics") {
    return <Navigate to="/visibility/metrics" replace />;
  }

  const widgets: DashboardWidget[] = [
    {
      id: "streams",
      title: "Log streams",
      description:
        "Volume of the access, authentication, audit and component logs",
      icon: Waves,
      eager: true,
      render: () => <Streams periodMinutes={periodMinutes} />,
    },
    {
      id: "breakdown",
      title: "Breakdown explorer",
      description: "Split a log stream by any dimension",
      icon: ChartSpline,
      eager: true,
      render: () => <Breakdown periodMinutes={periodMinutes} />,
    },
    {
      id: "components",
      title: "Components",
      description:
        "Which Cluster components log the most and complain the most",
      icon: ScrollText,
      height: 620,
      render: () => <Components periodMinutes={periodMinutes} />,
    },
    {
      id: "access-leaders",
      title: "Access leaders",
      description: "Top Users, Services, Policies, Sessions and deny reasons",
      icon: Trophy,
      height: 900,
      render: () => <AccessLeaders periodMinutes={periodMinutes} />,
    },
    {
      id: "identity-leaders",
      title: "Identity leaders",
      description: "Top Users, identity providers and Credentials by sign-ins",
      icon: ShieldUser,
      height: 520,
      render: () => <IdentityLeaders periodMinutes={periodMinutes} />,
    },
    {
      id: "change-leaders",
      title: "Configuration leaders",
      description: "Who changed the configuration, and from which Sessions",
      icon: Library,
      height: 720,
      render: () => <ChangeLeaders periodMinutes={periodMinutes} />,
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
        <Meta title="Visibility" />

        <DashboardControls
          periodMinutes={periodMinutes}
          onPeriodChange={setPeriodMinutes}
          autoRefresh={autoRefresh}
          onAutoRefreshChange={setAutoRefresh}
          onCustomize={() => setCustomizing(true)}
        />

        <DashboardLayout
          page="visibility"
          widgets={widgets}
          customizing={customizing}
          onCustomizingChange={setCustomizing}
        />
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
