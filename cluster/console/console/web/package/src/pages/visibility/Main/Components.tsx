import { ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import {
  STATUS_COLORS,
  seriesColor,
  useChartColorScheme,
} from "@/utils/charts/palette";
import { n, periodLabel } from "@/utils/visibility";
import { SegmentedControl } from "@mantine/core";
import { ScrollText } from "lucide-react";
import * as React from "react";
import Leaderboard, { LeaderboardItem } from "./Leaderboard";
import { COMPONENT_LIMIT, useComponentLeaderboard } from "./queries";

type LevelFilter = "all" | "warn" | "error";
type SortMode = "volume" | "severity";

const LEVELS: Record<LevelFilter, ComponentLog_Entry_Level | undefined> = {
  all: undefined,
  warn: ComponentLog_Entry_Level.WARN,
  error: ComponentLog_Entry_Level.ERROR,
};

const Chip = (props: {
  label: string;
  value: number;
  tone: "warn" | "error";
}) => {
  if (props.value === 0) return null;

  return (
    <span
      className={
        props.tone === "error"
          ? "hidden shrink-0 items-center gap-1 rounded-full border border-red-200 bg-red-50 px-1.5 py-px text-micro font-semibold tabular-nums text-red-700 sm:inline-flex"
          : "hidden shrink-0 items-center gap-1 rounded-full border border-amber-200 bg-amber-50 px-1.5 py-px text-micro font-semibold tabular-nums text-amber-700 sm:inline-flex"
      }
    >
      {compact(props.value)} {props.label}
    </span>
  );
};

const Components = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);
  useChartColorScheme();

  const [level, setLevel] = React.useState<LevelFilter>("all");
  const [sort, setSort] = React.useState<SortMode>("volume");

  const query = useComponentLeaderboard(periodMinutes, LEVELS[level]);

  const raw = query.data?.items ?? [];
  const severityOf = (item: (typeof raw)[number]) =>
    n(item.countError) + n(item.countPanic) + n(item.countFatal);

  const items: LeaderboardItem[] = [...raw]
    .sort((a, b) =>
      sort === "severity"
        ? severityOf(b) - severityOf(a) || n(b.count) - n(a.count)
        : n(b.count) - n(a.count),
    )
    .map((item) => {
      const errors = severityOf(item);
      const type = item.component?.type ?? "unknown";
      const namespace = item.component?.namespace ?? "";
      const params = new URLSearchParams();
      if (type) params.set("component.type", type);
      if (namespace) params.set("component.namespace", namespace);
      if (level !== "all") params.set("level", level.toUpperCase());

      return {
        id: `${namespace}/${type}`,
        name: type,
        sub: namespace || undefined,
        count: n(item.count),
        to: `/visibility/componentlogs?${params.toString()}`,
        accent: errors > 0 ? STATUS_COLORS.critical : seriesColor(2),
        trailing: (
          <>
            <Chip label="warn" value={n(item.countWarn)} tone="warn" />
            <Chip label="err" value={errors} tone="error" />
          </>
        ),
      };
    });

  return (
    <Panel
      icon={ScrollText}
      title="Components"
      description={`Which Cluster components are talking, and which ones are complaining, over the last ${rangeLabel}`}
      to="/visibility/componentlogs"
      toLabel="Component logs"
      actions={
        <>
          <SegmentedControl
            size="xs"
            value={sort}
            onChange={(value) => setSort(value as SortMode)}
            data={[
              { value: "volume", label: "By volume" },
              { value: "severity", label: "By severity" },
            ]}
          />
          <SegmentedControl
            size="xs"
            value={level}
            onChange={(value) => setLevel(value as LevelFilter)}
            data={[
              { value: "all", label: "All" },
              { value: "warn", label: "Warn" },
              { value: "error", label: "Error" },
            ]}
          />
        </>
      }
    >
      <Leaderboard
        title={`Top ${COMPONENT_LIMIT} components`}
        icon={ScrollText}
        items={items}
        totalCount={n(query.data?.totalCount)}
        totalOther={n(query.data?.totalOther)}
        to="/visibility/componentlogs"
        isLoading={query.isLoading}
        unit="entries"
        emptyLabel={`No component emitted a ${level === "all" ? "" : `${level} `}log entry in the last ${rangeLabel}.`}
      />
    </Panel>
  );
};

export default Components;
