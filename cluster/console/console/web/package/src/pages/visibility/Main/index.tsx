import AccessLogHealthWidget from "@/components/AccessLogViewer/AccessLogWidget";
import { AccessLogStatusFilter } from "@/components/AccessLogViewer/utils";
import AuditLogHealthWidget from "@/components/AuditLogViewer/AuditLogWidget";
import AuthenticationLogHealthWidget from "@/components/AuthenticationLogViewer/AuthenticationLogWidget";
import ClusterHealth from "@/components/ClusterHealth";
import ComponentLogHealthWidget from "@/components/ComponentLogViewer/ComponentLogWidget";
import ResourceInventory from "@/components/ResourceInventory";
import {
  ALL_PERIODS,
  DEFAULT_PERIOD_MINUTES,
  isKnownPeriod,
  periodLabel,
} from "@/utils/visibility";
import { Select, SegmentedControl } from "@mantine/core";
import { AnimatePresence, motion } from "framer-motion";
import { BarChart3, ChartNoAxesCombined, Layers } from "lucide-react";
import * as React from "react";
import { useSearchParams } from "react-router-dom";
import MetricsPage from "../Metrics";

type TabValue = "logs" | "resources" | "metrics";

const TABS: { value: TabValue; label: string; icon: typeof BarChart3 }[] = [
  { value: "logs", label: "Activity & Logs", icon: BarChart3 },
  { value: "resources", label: "Resources", icon: Layers },
  { value: "metrics", label: "Metrics", icon: ChartNoAxesCombined },
];

const isTab = (value: string | null): value is TabValue =>
  TABS.some((tab) => tab.value === value);

const isStatus = (value: string | null): value is AccessLogStatusFilter =>
  value === "all" || value === "allowed" || value === "denied";

const Widget = (props: { children: React.ReactNode }) => (
  <div className="rounded-xl border border-slate-200 bg-white px-4 py-3.5">
    {props.children}
  </div>
);

export default () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const tabParam = searchParams.get("tab");
  const tab: TabValue = isTab(tabParam) ? tabParam : "logs";

  const rangeParam = Number(searchParams.get("range"));
  const periodMinutes = isKnownPeriod(rangeParam)
    ? rangeParam
    : DEFAULT_PERIOD_MINUTES;

  const statusParam = searchParams.get("status");
  const status: AccessLogStatusFilter = isStatus(statusParam)
    ? statusParam
    : "all";

  const patchParams = (patch: Record<string, string | null>) => {
    const next = new URLSearchParams(searchParams);
    Object.entries(patch).forEach(([key, value]) => {
      if (value === null) next.delete(key);
      else next.set(key, value);
    });
    setSearchParams(next, { replace: true, preventScrollReset: true });
  };

  const setPeriodMinutes = (value: number) =>
    patchParams({
      range: value === DEFAULT_PERIOD_MINUTES ? null : String(value),
    });

  const setStatus = (value: AccessLogStatusFilter) =>
    patchParams({ status: value === "all" ? null : value });

  return (
    <div className="flex w-full flex-col gap-4 py-4">
      <header className="rounded-xl border border-slate-200 bg-white px-4 py-3.5 shadow-card">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-lg font-semibold tracking-tight text-slate-950">
              Visibility
            </h1>
            <p className="mt-0.5 text-xs leading-5 text-slate-500">
              Cluster health, security activity, inventory, and runtime metrics.
            </p>
          </div>

          {tab === "logs" && (
            <Select
              size="xs"
              aria-label="Time range"
              allowDeselect={false}
              checkIconPosition="right"
              value={String(periodMinutes)}
              onChange={(value) => value && setPeriodMinutes(Number(value))}
              data={ALL_PERIODS.map((option) => ({
                value: String(option.minutes),
                label: `Last ${option.label}`,
              }))}
              className="w-36"
            />
          )}
        </div>

        <div className="mt-3 overflow-x-auto border-t border-slate-100 pt-3">
          <SegmentedControl
            value={tab}
            onChange={(value) =>
              patchParams({ tab: value === "logs" ? null : value })
            }
            data={TABS.map((item) => {
              const Icon = item.icon;
              return {
                value: item.value,
                label: (
                  <span className="flex items-center justify-center gap-2 px-1.5 py-0.5">
                    <Icon size={14} strokeWidth={2.5} />
                    <span>{item.label}</span>
                  </span>
                ),
              };
            })}
          />
        </div>
      </header>

      <AnimatePresence mode="wait">
        {tab === "logs" ? (
          <motion.div
            key="logs"
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className="flex flex-col gap-4"
          >
            <ClusterHealth periodMinutes={periodMinutes} />

            <p className="text-micro font-normal text-slate-500">
              All panels below share the same window — last{" "}
              {periodLabel(periodMinutes)}. Change it from the range selector
              above.
            </p>

            <Widget>
              <AccessLogHealthWidget
                periodMinutes={periodMinutes}
                onPeriodChange={setPeriodMinutes}
                hideRangeControl
                status={status}
                onStatusChange={setStatus}
              />
            </Widget>
            <Widget>
              <AuthenticationLogHealthWidget
                periodMinutes={periodMinutes}
                onPeriodChange={setPeriodMinutes}
                hideRangeControl
              />
            </Widget>
            <Widget>
              <AuditLogHealthWidget
                periodMinutes={periodMinutes}
                onPeriodChange={setPeriodMinutes}
                hideRangeControl
              />
            </Widget>
            <Widget>
              <ComponentLogHealthWidget
                periodMinutes={periodMinutes}
                onPeriodChange={setPeriodMinutes}
                hideRangeControl
              />
            </Widget>
          </motion.div>
        ) : tab === "resources" ? (
          <motion.div
            key="resources"
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 6 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
          >
            <ResourceInventory />
          </motion.div>
        ) : (
          <MetricsPage key="metrics" />
        )}
      </AnimatePresence>
    </div>
  );
};
