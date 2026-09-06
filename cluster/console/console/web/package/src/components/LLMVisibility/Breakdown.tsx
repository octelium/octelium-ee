import {
  Dimension,
  Filter,
  ListTopDimensionRequest,
  Metric,
} from "@/apis/visibilityv1/llm/vllmv1";
import { getClientVisibilityLLM } from "@/utils/client";
import { SegmentedControl, Select } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { QueryState } from "./Primitives";
import StatBars from "./StatBars";
import {
  dimensionLabel,
  Drilldown,
  DRILLDOWN_DIMENSIONS,
  METRIC_ACCESSORS,
  metricAccessor,
} from "./utils";

const Breakdown = (props: {
  filter: Filter;
  dimension: Dimension;
  enabled?: boolean;
  limit?: number;
  defaultMetric?: Metric;
  showMetricSelect?: boolean;
  colored?: boolean;
  onDrilldown?: (drilldown: Drilldown) => void;
}) => {
  const [metric, setMetric] = React.useState<Metric>(
    props.defaultMetric ?? Metric.REQUESTS,
  );
  const accessor = metricAccessor(metric);
  const enabled = props.enabled ?? true;
  const limit = props.limit ?? 10;

  const qry = useQuery({
    queryKey: [
      "visibility",
      "llm",
      "topDimension",
      { filter: props.filter, dimension: props.dimension, metric, limit },
    ],
    enabled,
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().listTopDimension(
        ListTopDimensionRequest.create({
          filter: props.filter,
          dimension: props.dimension,
          limit,
          orderBy: metric,
          includeQuantiles: accessor.kind === "duration",
        }),
      );
      return response;
    },
  });

  const canDrilldown =
    props.onDrilldown && DRILLDOWN_DIMENSIONS.has(props.dimension);

  return (
    <div className="flex w-full flex-col gap-3">
      {props.showMetricSelect && (
        <Select
          size="xs"
          className="max-w-[220px]"
          label="Rank by"
          allowDeselect={false}
          value={String(metric)}
          data={METRIC_ACCESSORS.map((item) => ({
            value: String(item.metric),
            label: item.label,
          }))}
          onChange={(value) => value && setMetric(Number(value))}
        />
      )}
      <QueryState
        isLoading={enabled && qry.isPending}
        isError={qry.isError}
        isEmpty={enabled && qry.isSuccess && (qry.data?.items.length ?? 0) === 0}
        minHeight={140}
      >
        <StatBars
          items={qry.data?.items ?? []}
          other={qry.data?.other}
          totalCount={qry.data?.totalCount}
          accessor={accessor}
          colored={props.colored}
          onSelect={
            canDrilldown
              ? (item) =>
                  props.onDrilldown?.({
                    dimension: props.dimension,
                    key: item.key,
                  })
              : undefined
          }
        />
      </QueryState>
    </div>
  );
};

export default Breakdown;

export const BreakdownTabs = (props: {
  filter: Filter;
  dimensions: Dimension[];
  enabled?: boolean;
  limit?: number;
  defaultMetric?: Metric;
  showMetricSelect?: boolean;
  onDrilldown?: (drilldown: Drilldown) => void;
}) => {
  const [dimension, setDimension] = React.useState<Dimension>(
    props.dimensions[0] ?? Dimension.DIMENSION_UNSET,
  );

  if (props.dimensions.length === 0) return null;

  return (
    <div className="flex w-full flex-col gap-3">
      <SegmentedControl
        size="xs"
        className="w-fit max-w-full overflow-x-auto"
        value={String(dimension)}
        onChange={(value) => setDimension(Number(value))}
        data={props.dimensions.map((item) => ({
          value: String(item),
          label: dimensionLabel(item),
        }))}
      />
      <Breakdown
        filter={props.filter}
        dimension={dimension}
        enabled={props.enabled}
        limit={props.limit}
        defaultMetric={props.defaultMetric}
        showMetricSelect={props.showMetricSelect}
        colored
        onDrilldown={props.onDrilldown}
      />
    </div>
  );
};
