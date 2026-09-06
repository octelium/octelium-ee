import * as CoreP from "@/apis/corev1/corev1";
import {
  Dimension,
  EntryScope,
  GetSummaryRequest,
  Metric,
} from "@/apis/visibilityv1/llm/vllmv1";
import { getClientVisibilityLLM } from "@/utils/client";
import { Badge } from "@mantine/core";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Bot,
  Boxes,
  DatabaseZap,
  Gauge,
  Radar,
  ScrollText,
  ShieldAlert,
  Signpost,
  Users,
  Wrench,
} from "lucide-react";
import * as React from "react";
import AccessLogs from "./AccessLogs";
import { BreakdownTabs } from "./Breakdown";
import FilterBar from "./FilterBar";
import { LazySection, Panel, QueryState } from "./Primitives";
import StatsOverview from "./StatsOverview";
import TopModels from "./TopModels";
import TopPrincipals from "./TopPrincipals";
import TopTools from "./TopTools";
import Trend from "./Trend";
import {
  applyDrilldowns,
  buildFilter,
  dimensionLabel,
  Drilldown,
  drilldownID,
  intervalForMinutes,
  LLMRange,
  LLMScope,
  ModelFieldDimension,
  num,
  rangeFromMinutes,
} from "./utils";

const CARDINALITY_DIMENSIONS = [
  Dimension.MODEL,
  Dimension.USER,
  Dimension.SERVICE,
  Dimension.TOOL,
];

const LLMVisibility = (props: {
  scope?: LLMScope;
  defaultMinutes?: number;
  hideDimensions?: Dimension[];
  principalKinds?: ("user" | "session" | "service" | "policy")[];
}) => {
  const queryClient = useQueryClient();
  const defaultMinutes = props.defaultMinutes ?? 1440;
  const [minutes, setMinutes] = React.useState<number | undefined>(
    defaultMinutes,
  );
  const [range, setRange] = React.useState<LLMRange>(
    rangeFromMinutes(defaultMinutes),
  );
  const [status, setStatus] = React.useState<CoreP.AccessLog_Entry_Common_Status>(
    CoreP.AccessLog_Entry_Common_Status.STATUS_UNSET,
  );
  const [drilldowns, setDrilldowns] = React.useState<Drilldown[]>([]);

  const hidden = React.useMemo(
    () => new Set(props.hideDimensions ?? []),
    [props.hideDimensions],
  );
  const visible = React.useCallback(
    (dimensions: Dimension[]) =>
      dimensions.filter((dimension) => !hidden.has(dimension)),
    [hidden],
  );

  const filter = React.useMemo(
    () =>
      applyDrilldowns(
        buildFilter({
          scope: props.scope,
          range,
          entryScope: EntryScope.TERMINAL,
          extra: { status },
        }),
        drilldowns,
      ),
    [drilldowns, props.scope, range, status],
  );

  const interval = React.useMemo(
    () => intervalForMinutes(minutes ?? defaultMinutes),
    [defaultMinutes, minutes],
  );

  const summaryQry = useQuery({
    queryKey: ["visibility", "llm", "summary", { filter }],
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().getSummary(
        GetSummaryRequest.create({
          filter,
          includeQuantiles: true,
          cardinalities: visible(CARDINALITY_DIMENSIONS),
        }),
      );
      return response;
    },
  });

  const secondaryEnabled = summaryQry.isFetched;

  const addDrilldown = (drilldown: Drilldown) =>
    setDrilldowns((current) =>
      current.some((item) => drilldownID(item) === drilldownID(drilldown))
        ? current
        : [...current, drilldown],
    );

  const removeDrilldown = (drilldown: Drilldown) =>
    setDrilldowns((current) =>
      current.filter((item) => drilldownID(item) !== drilldownID(drilldown)),
    );

  return (
    <div className="flex w-full flex-col gap-4">
      <FilterBar
        minutes={minutes}
        onMinutesChange={(value) => {
          setMinutes(value);
          setRange(rangeFromMinutes(value));
        }}
        range={range}
        onRangeChange={(value) => {
          setMinutes(undefined);
          setRange(value);
        }}
        status={status}
        onStatusChange={setStatus}
        drilldowns={drilldowns}
        onRemoveDrilldown={removeDrilldown}
        onClearDrilldowns={() => setDrilldowns([])}
        onRefresh={() =>
          queryClient.invalidateQueries({
            queryKey: ["visibility", "llm"],
          })
        }
      />

      <Panel
        title="Inference overview"
        description="Every request that the Cluster's LLM Services served in the selected range"
        icon={Activity}
        action={
          (summaryQry.data?.cardinalities.length ?? 0) > 0 ? (
            <div className="flex flex-wrap items-center gap-1.5">
              {summaryQry.data!.cardinalities.map((item) => (
                <Badge
                  key={item.dimension}
                  size="sm"
                  variant="light"
                  color="gray"
                >
                  {num(item.count).toLocaleString()}{" "}
                  {dimensionLabel(item.dimension).toLowerCase()}
                  {num(item.count) === 1 ? "" : "s"}
                </Badge>
              ))}
            </div>
          ) : undefined
        }
      >
        <QueryState
          isLoading={summaryQry.isPending}
          isError={summaryQry.isError}
          isEmpty={
            summaryQry.isSuccess &&
            num(summaryQry.data?.stats?.requests?.total) === 0
          }
          minHeight={190}
        >
          <StatsOverview stats={summaryQry.data?.stats} />
        </QueryState>
      </Panel>

      <Panel
        title="Traffic and tokens"
        description="Pick a metric and optionally split it by any dimension"
        icon={Radar}
      >
        <Trend filter={filter} interval={interval} enabled={secondaryEnabled} />
      </Panel>

      <Panel
        title="Models"
        description="The models the Cluster actually served, and the ones it was asked for"
        icon={Bot}
      >
        <TopModels
          filter={filter}
          enabled={secondaryEnabled}
          onSelect={(model, field) =>
            addDrilldown({
              dimension: ModelFieldDimension[field] ?? Dimension.MODEL,
              key: model,
            })
          }
        />
      </Panel>

      <div className="grid grid-cols-1 gap-4 2xl:grid-cols-2">
        <LazySection
          title="Guardrails and denials"
          description="What the inspection plugins and the Policies blocked"
          icon={ShieldAlert}
        >
          {(opened) => (
            <BreakdownTabs
              filter={filter}
              enabled={opened}
              showMetricSelect
              defaultMetric={Metric.REQUESTS}
              dimensions={visible([
                Dimension.GUARDRAIL_RESULT,
                Dimension.GUARDRAIL_PLUGIN,
                Dimension.GUARDRAIL_LEG,
                Dimension.DENY_REASON,
                Dimension.POLICY,
              ])}
              onDrilldown={addDrilldown}
            />
          )}
        </LazySection>

        <LazySection
          title="Cost controls"
          description="Semantic cache, semantic router and token quotas"
          icon={DatabaseZap}
        >
          {(opened) => (
            <BreakdownTabs
              filter={filter}
              enabled={opened}
              showMetricSelect
              defaultMetric={Metric.TOTAL_TOKENS}
              dimensions={visible([
                Dimension.SEMANTIC_CACHE_RESULT,
                Dimension.SEMANTIC_ROUTER_RESULT,
                Dimension.SEMANTIC_ROUTER_ROUTE,
                Dimension.TOKEN_RATE_LIMIT_RESULT,
                Dimension.TOKEN_RATE_LIMIT_SCOPE,
                Dimension.SOURCE,
              ])}
              onDrilldown={addDrilldown}
            />
          )}
        </LazySection>

        <LazySection
          title="Protocols and routes"
          description="How the traffic reaches the gateway"
          icon={Signpost}
        >
          {(opened) => (
            <BreakdownTabs
              filter={filter}
              enabled={opened}
              showMetricSelect
              dimensions={visible([
                Dimension.PROTOCOL,
                Dimension.OPERATION,
                Dimension.ROUTE,
                Dimension.HTTP_PATH,
                Dimension.IS_STREAM,
                Dimension.MODEL_SOURCE,
              ])}
              onDrilldown={addDrilldown}
            />
          )}
        </LazySection>

        <LazySection
          title="Reliability"
          description="How the requests ended, and what the upstream returned"
          icon={Gauge}
        >
          {(opened) => (
            <BreakdownTabs
              filter={filter}
              enabled={opened}
              showMetricSelect
              dimensions={visible([
                Dimension.FINISH_REASON,
                Dimension.FINISH_REASON_RAW,
                Dimension.HTTP_STATUS_CLASS,
                Dimension.HTTP_STATUS_CODE,
                Dimension.USAGE_STATE,
                Dimension.ESTIMATE_QUALITY,
              ])}
              onDrilldown={addDrilldown}
            />
          )}
        </LazySection>

        <LazySection
          title="Tools"
          description="The tools the callers offered, called and had removed"
          icon={Wrench}
        >
          {(opened) => (
            <TopTools
              filter={filter}
              enabled={opened}
              onSelect={(tool, scope) =>
                addDrilldown({
                  dimension:
                    scope === 2
                      ? Dimension.CALLED_TOOL
                      : scope === 3
                        ? Dimension.REMOVED_TOOL
                        : Dimension.TOOL,
                  key: tool,
                })
              }
            />
          )}
        </LazySection>

        <LazySection
          title="Clients"
          description="Which agents and integrations drive the traffic"
          icon={Boxes}
        >
          {(opened) => (
            <BreakdownTabs
              filter={filter}
              enabled={opened}
              showMetricSelect
              dimensions={visible([
                Dimension.USER_AGENT,
                Dimension.DEVICE,
                Dimension.NAMESPACE,
                Dimension.REGION,
                Dimension.HAS_IMAGE_INPUT,
                Dimension.HAS_AUDIO_INPUT,
              ])}
              onDrilldown={addDrilldown}
            />
          )}
        </LazySection>
      </div>

      {(props.principalKinds?.length ?? 0) > 0 && (
        <LazySection
          title="Top consumers"
          description="Who and what consumed the most inference"
          icon={Users}
        >
          {(opened) => (
            <TopPrincipals
              filter={filter}
              enabled={opened}
              kinds={props.principalKinds!}
            />
          )}
        </LazySection>
      )}

      <LazySection
        title="Requests"
        description="Individual inference requests matching the current filters"
        icon={ScrollText}
      >
        {(opened) => <AccessLogs filter={filter} enabled={opened} />}
      </LazySection>
    </div>
  );
};

export default LLMVisibility;
