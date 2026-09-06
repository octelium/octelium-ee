import {
  Filter,
  ListTopToolRequest,
  Metric,
  ToolScope,
} from "@/apis/visibilityv1/llm/vllmv1";
import { getClientVisibilityLLM } from "@/utils/client";
import { SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { QueryState } from "./Primitives";
import StatBars from "./StatBars";
import { metricAccessor, num } from "./utils";

const SCOPE_OPTIONS = [
  { value: String(ToolScope.OFFERED), label: "Offered" },
  { value: String(ToolScope.CALLED), label: "Called" },
  { value: String(ToolScope.REMOVED), label: "Removed" },
];

const TopTools = (props: {
  filter: Filter;
  enabled?: boolean;
  onSelect?: (tool: string, scope: ToolScope) => void;
}) => {
  const [scope, setScope] = React.useState<ToolScope>(ToolScope.OFFERED);
  const enabled = props.enabled ?? true;
  const accessor = metricAccessor(Metric.REQUESTS);

  const qry = useQuery({
    queryKey: ["visibility", "llm", "topTool", { filter: props.filter, scope }],
    enabled,
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().listTopTool(
        ListTopToolRequest.create({
          filter: props.filter,
          scope,
          limit: 12,
          orderBy: Metric.REQUESTS,
        }),
      );
      return response;
    },
  });

  const items = React.useMemo(
    () =>
      (qry.data?.items ?? []).map((item) => ({
        key: item.tool,
        stats: item.stats,
        offered: num(item.offeredCount),
        called: num(item.calledCount),
        removed: num(item.removedCount),
      })),
    [qry.data],
  );

  return (
    <div className="flex w-full flex-col gap-3">
      <SegmentedControl
        size="xs"
        className="w-fit"
        value={String(scope)}
        onChange={(value) => setScope(Number(value))}
        data={SCOPE_OPTIONS}
      />
      <QueryState
        isLoading={enabled && qry.isPending}
        isError={qry.isError}
        isEmpty={enabled && qry.isSuccess && items.length === 0}
        minHeight={160}
      >
        <StatBars
          items={items}
          other={qry.data?.other}
          totalCount={qry.data?.totalCount}
          accessor={accessor}
          colored
          onSelect={
            props.onSelect
              ? (item) => props.onSelect?.(item.key, scope)
              : undefined
          }
          renderLabel={(item) => {
            const row = items.find((entry) => entry.key === item.key);
            return (
              <span className="flex min-w-0 items-center gap-2">
                <span className="truncate font-mono">{item.key}</span>
                {row && (
                  <span className="shrink-0 text-[0.6rem] font-semibold text-slate-400">
                    {row.offered.toLocaleString()} offered ·{" "}
                    {row.called.toLocaleString()} called ·{" "}
                    {row.removed.toLocaleString()} removed
                  </span>
                )}
              </span>
            );
          }}
        />
      </QueryState>
    </div>
  );
};

export default TopTools;
