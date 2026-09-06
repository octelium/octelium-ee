import { Duration } from "@/apis/metav1/metav1";
import {
  Dimension,
  Filter,
  GetDataPointRequest,
  Metric,
} from "@/apis/visibilityv1/llm/vllmv1";
import SeriesChart, { Series } from "@/components/Charts/SeriesChart";
import { getClientVisibilityLLM } from "@/utils/client";
import { Select, SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { QueryState } from "./Primitives";
import {
  dimensionLabel,
  formatMetricValue,
  METRIC_ACCESSORS,
  metricAccessor,
  prettyKey,
} from "./utils";

const GROUP_DIMENSIONS = [
  Dimension.DIMENSION_UNSET,
  Dimension.MODEL,
  Dimension.PROTOCOL,
  Dimension.OPERATION,
  Dimension.ROUTE,
  Dimension.STATUS,
  Dimension.SOURCE,
  Dimension.GUARDRAIL_RESULT,
  Dimension.SEMANTIC_CACHE_RESULT,
  Dimension.FINISH_REASON,
  Dimension.USER,
  Dimension.SERVICE,
];

const Trend = (props: {
  filter: Filter;
  interval: Duration;
  enabled?: boolean;
}) => {
  const [metric, setMetric] = React.useState<Metric>(Metric.REQUESTS);
  const [groupBy, setGroupBy] = React.useState<Dimension>(
    Dimension.DIMENSION_UNSET,
  );
  const [variant, setVariant] = React.useState<"line" | "bar">("line");

  const accessor = metricAccessor(metric);
  const enabled = props.enabled ?? true;

  const qry = useQuery({
    queryKey: [
      "visibility",
      "llm",
      "dataPoint",
      { filter: props.filter, interval: props.interval, groupBy, metric },
    ],
    enabled,
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().getDataPoint(
        GetDataPointRequest.create({
          filter: props.filter,
          interval: props.interval,
          groupBy,
          limit: 6,
          orderBy: metric,
          includeQuantiles: accessor.kind === "duration",
        }),
      );
      return response;
    },
  });

  const series: Series[] = React.useMemo(() => {
    const items = qry.data?.series ?? [];
    const ret: Series[] = items.map((item) => ({
      name:
        groupBy === Dimension.DIMENSION_UNSET
          ? accessor.label
          : (item.ref?.name ?? prettyKey(item.key)),
      points: item.datapoints
        .filter((point) => point.timestamp)
        .map((point) => ({
          ts: point.timestamp!,
          value: accessor.get(point.stats),
        })),
    }));

    if (qry.data?.other) {
      ret.push({
        name: "Other",
        points: qry.data.other.datapoints
          .filter((point) => point.timestamp)
          .map((point) => ({
            ts: point.timestamp!,
            value: accessor.get(point.stats),
          })),
      });
    }

    return ret;
  }, [accessor, groupBy, qry.data]);

  return (
    <div className="flex w-full flex-col gap-3">
      <div className="flex flex-wrap items-end gap-2">
        <Select
          size="xs"
          label="Metric"
          className="min-w-[190px]"
          allowDeselect={false}
          value={String(metric)}
          data={METRIC_ACCESSORS.map((item) => ({
            value: String(item.metric),
            label: item.label,
          }))}
          onChange={(value) => value && setMetric(Number(value))}
        />
        <Select
          size="xs"
          label="Group by"
          className="min-w-[170px]"
          allowDeselect={false}
          value={String(groupBy)}
          data={GROUP_DIMENSIONS.map((dimension) => ({
            value: String(dimension),
            label:
              dimension === Dimension.DIMENSION_UNSET
                ? "None"
                : dimensionLabel(dimension),
          }))}
          onChange={(value) => value && setGroupBy(Number(value))}
        />
        <div className="flex-1" />
        <SegmentedControl
          size="xs"
          value={variant}
          onChange={(value) => setVariant(value as "line" | "bar")}
          data={[
            { value: "line", label: "Line" },
            { value: "bar", label: "Bar" },
          ]}
        />
      </div>

      <QueryState
        isLoading={enabled && qry.isPending}
        isError={qry.isError}
        isEmpty={enabled && qry.isSuccess && series.length === 0}
        minHeight={280}
      >
        <SeriesChart
          series={series}
          height={280}
          variant={variant}
          stacked={variant === "bar" && groupBy !== Dimension.DIMENSION_UNSET}
          valueFormatter={(value) =>
            formatMetricValue(accessor.kind, value)
          }
        />
      </QueryState>
    </div>
  );
};

export default Trend;
