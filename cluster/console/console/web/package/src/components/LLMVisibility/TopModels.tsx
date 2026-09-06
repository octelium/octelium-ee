import {
  Filter,
  ListTopModelRequest,
  Metric,
  ModelField,
} from "@/apis/visibilityv1/llm/vllmv1";
import { Select } from "@mantine/core";
import { getClientVisibilityLLM } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { QueryState } from "./Primitives";
import StatBars from "./StatBars";
import { METRIC_ACCESSORS, metricAccessor, num } from "./utils";

const FIELD_OPTIONS = [
  { value: String(ModelField.EFFECTIVE), label: "Effective" },
  { value: String(ModelField.REQUESTED), label: "Requested" },
  { value: String(ModelField.REPORTED), label: "Reported" },
];

const TopModels = (props: {
  filter: Filter;
  enabled?: boolean;
  onSelect?: (model: string, field: ModelField) => void;
}) => {
  const [field, setField] = React.useState<ModelField>(ModelField.EFFECTIVE);
  const [metric, setMetric] = React.useState<Metric>(Metric.REQUESTS);
  const accessor = metricAccessor(metric);
  const enabled = props.enabled ?? true;

  const qry = useQuery({
    queryKey: [
      "visibility",
      "llm",
      "topModel",
      { filter: props.filter, field, metric },
    ],
    enabled,
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().listTopModel(
        ListTopModelRequest.create({
          filter: props.filter,
          field,
          limit: 12,
          orderBy: metric,
          includeQuantiles: accessor.kind === "duration",
        }),
      );
      return response;
    },
  });

  const items = React.useMemo(
    () =>
      (qry.data?.items ?? []).map((item) => ({
        key: item.model,
        stats: item.stats,
        requested: num(item.requestedCount),
        effective: num(item.effectiveCount),
        reported: num(item.reportedCount),
      })),
    [qry.data],
  );

  return (
    <div className="flex w-full flex-col gap-3">
      <div className="flex flex-wrap items-end gap-2">
        <Select
          size="xs"
          label="Model field"
          className="min-w-[150px]"
          allowDeselect={false}
          value={String(field)}
          data={FIELD_OPTIONS}
          onChange={(value) => value && setField(Number(value))}
        />
        <Select
          size="xs"
          label="Rank by"
          className="min-w-[190px]"
          allowDeselect={false}
          value={String(metric)}
          data={METRIC_ACCESSORS.map((item) => ({
            value: String(item.metric),
            label: item.label,
          }))}
          onChange={(value) => value && setMetric(Number(value))}
        />
      </div>
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
              ? (item) => props.onSelect?.(item.key, field)
              : undefined
          }
          renderLabel={(item) => {
            const row = items.find((entry) => entry.key === item.key);
            return (
              <span className="flex min-w-0 items-center gap-2">
                <span className="truncate">{item.key || "Not set"}</span>
                {row && row.requested !== row.effective && (
                  <span className="shrink-0 rounded bg-amber-50 px-1.5 py-px text-[0.58rem] font-bold uppercase tracking-[0.04em] text-amber-700">
                    rewritten
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

export default TopModels;
