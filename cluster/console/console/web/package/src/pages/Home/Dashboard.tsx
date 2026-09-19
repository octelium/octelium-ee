import PeriodSelector from "@/components/LogWidget/PeriodSelector";
import Meta from "@/components/Meta";
import { getDomain } from "@/utils";
import { periodLabel } from "@/utils/visibility";
import { ActionIcon, Switch, Tooltip } from "@mantine/core";
import { useIsFetching, useQueryClient } from "@tanstack/react-query";
import { motion } from "framer-motion";
import {
  Building2,
  ChartNoAxesCombined,
  Cpu,
  Eye,
  RefreshCw,
  Settings2,
  UserCheck,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import Activity from "./Activity";
import Attention from "./Attention";
import ClusterCard from "./ClusterCard";
import { Deferred } from "./components";
import Governance from "./Governance";
import Health from "./Health";
import Identity from "./Identity";
import Inventory from "./Inventory";
import Operations from "./Operations";
import Runtime from "./Runtime";
import Signals from "./Signals";
import { AutoRefreshContext, useDashboardRange } from "./utils";

const SHORTCUTS = [
  { label: "Core", to: "/core", icon: Cpu },
  { label: "Access", to: "/access", icon: UserCheck },
  { label: "Enterprise", to: "/enterprise", icon: Building2 },
  { label: "Visibility", to: "/visibility", icon: Eye },
  { label: "Metrics", to: "/visibility/metrics", icon: ChartNoAxesCombined },
  { label: "Cluster", to: "/clusterman", icon: Settings2 },
];

const Dashboard = () => {
  const { periodMinutes, setPeriodMinutes } = useDashboardRange();
  const [autoRefresh, setAutoRefresh] = React.useState(true);
  const queryClient = useQueryClient();
  const fetching = useIsFetching();
  const rangeLabel = periodLabel(periodMinutes);

  return (
    <AutoRefreshContext.Provider value={autoRefresh}>
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="flex w-full flex-col gap-4 py-4"
      >
        <Meta title="Overview" />

        <header className="rounded-xl border border-slate-200 bg-white px-4 py-4 shadow-card sm:px-5">
          <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
            <div className="min-w-0">
              <h1 className="text-xl font-bold tracking-[-0.02em] text-slate-950">
                Cluster overview
              </h1>
              <p className="mt-1 text-xs font-normal leading-5 text-slate-500">
                Security, identity and platform posture for{" "}
                <span className="font-semibold text-slate-700">
                  {getDomain()}
                </span>{" "}
                · every panel below covers the last {rangeLabel}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2.5">
              {fetching > 0 && (
                <span className="text-micro font-normal tabular-nums text-slate-500">
                  Updating {fetching}…
                </span>
              )}

              <PeriodSelector
                value={periodMinutes}
                onChange={setPeriodMinutes}
              />

              <Switch
                size="xs"
                label="Auto"
                checked={autoRefresh}
                onChange={(event) =>
                  setAutoRefresh(event.currentTarget.checked)
                }
              />

              <Tooltip label="Refresh everything" withArrow>
                <ActionIcon
                  type="button"
                  variant="default"
                  size="sm"
                  aria-label="Refresh the dashboard"
                  onClick={() => queryClient.invalidateQueries()}
                >
                  <RefreshCw
                    size={12}
                    strokeWidth={2.5}
                    className={fetching > 0 ? "animate-spin" : ""}
                  />
                </ActionIcon>
              </Tooltip>
            </div>
          </div>

          <nav
            aria-label="Jump to an API"
            className="mt-3.5 flex flex-wrap gap-1.5 border-t border-slate-100 pt-3.5"
          >
            {SHORTCUTS.map((shortcut) => {
              const Icon = shortcut.icon;
              return (
                <Link
                  key={shortcut.to}
                  to={shortcut.to}
                  className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-slate-50/70 px-2.5 py-1.5 text-micro font-semibold text-slate-600 outline-none transition-colors duration-150 hover:border-slate-300 hover:bg-white hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
                >
                  <Icon size={12} strokeWidth={2.3} />
                  {shortcut.label}
                </Link>
              );
            })}
          </nav>
        </header>

        <Signals periodMinutes={periodMinutes} />

        <Attention periodMinutes={periodMinutes} />

        <Deferred height={420}>
          <Health periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={760}>
          <Activity periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={680}>
          <Identity periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={460}>
          <Governance periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={400}>
          <Inventory periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={760}>
          <Runtime periodMinutes={periodMinutes} />
        </Deferred>

        <Deferred height={420}>
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <Operations periodMinutes={periodMinutes} />
            <ClusterCard />
          </div>
        </Deferred>
      </motion.div>
    </AutoRefreshContext.Provider>
  );
};

export default Dashboard;
