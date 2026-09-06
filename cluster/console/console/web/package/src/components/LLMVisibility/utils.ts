import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { Duration, ObjectReference } from "@/apis/metav1/metav1";
import * as CoreP from "@/apis/corev1/corev1";
import {
  Dimension,
  EntryScope,
  Filter,
  HTTPStatusClass,
  Metric,
  ModelField,
  Stats,
} from "@/apis/visibilityv1/llm/vllmv1";

export type LLMScope = {
  userRef?: ObjectReference;
  sessionRef?: ObjectReference;
  deviceRef?: ObjectReference;
  serviceRef?: ObjectReference;
  namespaceRef?: ObjectReference;
  regionRef?: ObjectReference;
  policyRef?: ObjectReference;
};

export type LLMRange = {
  from?: Timestamp;
  to?: Timestamp;
};

export const RANGE_PRESETS = [
  { label: "1h", minutes: 60 },
  { label: "6h", minutes: 360 },
  { label: "24h", minutes: 1440 },
  { label: "7d", minutes: 10080 },
  { label: "30d", minutes: 43200 },
];

export const rangeFromMinutes = (minutes: number): LLMRange => ({
  from: Timestamp.fromDate(new Date(Date.now() - minutes * 60000)),
  to: undefined,
});

export const intervalForMinutes = (minutes: number): Duration => {
  if (minutes <= 60) return { type: { oneofKind: "minutes", minutes: 2 } };
  if (minutes <= 360) return { type: { oneofKind: "minutes", minutes: 10 } };
  if (minutes <= 1440) return { type: { oneofKind: "minutes", minutes: 30 } };
  if (minutes <= 10080) return { type: { oneofKind: "hours", hours: 3 } };
  return { type: { oneofKind: "hours", hours: 12 } };
};

export const buildFilter = (arg: {
  scope?: LLMScope;
  range?: LLMRange;
  entryScope?: EntryScope;
  extra?: Partial<Filter>;
}): Filter =>
  Filter.create({
    ...arg.scope,
    from: arg.range?.from,
    to: arg.range?.to,
    entryScope: arg.entryScope ?? EntryScope.TERMINAL,
    ...arg.extra,
  });

export const num = (value: bigint | number | undefined): number =>
  value === undefined ? 0 : Number(value);

export const requests = (stats?: Stats) => stats?.requests;

export const ratio = (part: number, total: number): number =>
  total <= 0 ? 0 : (part / total) * 100;

export const formatPercent = (value: number, digits = 1): string =>
  `${value.toFixed(digits)}%`;

export const formatMs = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) return "—";
  if (value < 1000) return `${Math.round(value)} ms`;
  if (value < 60000) return `${(value / 1000).toFixed(2)} s`;
  return `${(value / 60000).toFixed(1)} min`;
};

export const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`;
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return value.toLocaleString();
};

export const DIMENSION_LABELS: Record<number, string> = {
  [Dimension.MODEL]: "Model",
  [Dimension.MODEL_REQUESTED]: "Requested model",
  [Dimension.MODEL_REPORTED]: "Reported model",
  [Dimension.MODEL_SOURCE]: "Model source",
  [Dimension.MODEL_PLUGIN]: "Model plugin",
  [Dimension.PROTOCOL]: "Protocol",
  [Dimension.OPERATION]: "Operation",
  [Dimension.ROUTE]: "Route",
  [Dimension.SOURCE]: "Response source",
  [Dimension.USAGE_STATE]: "Usage state",
  [Dimension.ESTIMATE_QUALITY]: "Estimate quality",
  [Dimension.GUARDRAIL_RESULT]: "Guardrail result",
  [Dimension.GUARDRAIL_PLUGIN]: "Guardrail plugin",
  [Dimension.GUARDRAIL_LEG]: "Guardrail leg",
  [Dimension.FINISH_REASON]: "Finish reason",
  [Dimension.FINISH_REASON_RAW]: "Raw finish reason",
  [Dimension.REASONING_EFFORT]: "Reasoning effort",
  [Dimension.TOOL]: "Offered tool",
  [Dimension.CALLED_TOOL]: "Called tool",
  [Dimension.REMOVED_TOOL]: "Removed tool",
  [Dimension.STATUS]: "Status",
  [Dimension.DENY_REASON]: "Deny reason",
  [Dimension.HTTP_STATUS_CODE]: "HTTP status",
  [Dimension.HTTP_STATUS_CLASS]: "HTTP status class",
  [Dimension.IS_STREAM]: "Streamed",
  [Dimension.IS_UPSTREAM_INVOKED]: "Upstream invoked",
  [Dimension.USER_AGENT]: "User agent",
  [Dimension.USER]: "User",
  [Dimension.SESSION]: "Session",
  [Dimension.DEVICE]: "Device",
  [Dimension.SERVICE]: "Service",
  [Dimension.NAMESPACE]: "Namespace",
  [Dimension.REGION]: "Region",
  [Dimension.POLICY]: "Policy",
  [Dimension.HTTP_PATH]: "HTTP path",
  [Dimension.SEMANTIC_CACHE_RESULT]: "Cache result",
  [Dimension.SEMANTIC_CACHE_PLUGIN]: "Cache plugin",
  [Dimension.SEMANTIC_ROUTER_RESULT]: "Router result",
  [Dimension.SEMANTIC_ROUTER_ROUTE]: "Router route",
  [Dimension.SEMANTIC_ROUTER_PLUGIN]: "Router plugin",
  [Dimension.SEMANTIC_ROUTER_MODEL]: "Routed model",
  [Dimension.TOKEN_RATE_LIMIT_RESULT]: "Token quota result",
  [Dimension.TOKEN_RATE_LIMIT_PLUGIN]: "Token quota plugin",
  [Dimension.TOKEN_RATE_LIMIT_SCOPE]: "Token quota scope",
  [Dimension.HAS_IMAGE_INPUT]: "Image input",
  [Dimension.HAS_AUDIO_INPUT]: "Audio input",
};

export const dimensionLabel = (dimension: Dimension): string =>
  DIMENSION_LABELS[dimension] ?? Dimension[dimension] ?? "Dimension";

export const METRIC_LABELS: Record<number, string> = {
  [Metric.REQUESTS]: "Requests",
  [Metric.ALLOWED_REQUESTS]: "Allowed requests",
  [Metric.DENIED_REQUESTS]: "Denied requests",
  [Metric.FAILED_REQUESTS]: "Failed requests",
  [Metric.STREAMED_REQUESTS]: "Streamed requests",
  [Metric.CACHED_REQUESTS]: "Cache-served requests",
  [Metric.GUARDRAIL_DENIED_REQUESTS]: "Guardrail denials",
  [Metric.TOOL_CALL_REQUESTS]: "Requests with tool calls",
  [Metric.INPUT_TOKENS]: "Input tokens",
  [Metric.OUTPUT_TOKENS]: "Output tokens",
  [Metric.TOTAL_TOKENS]: "Total tokens",
  [Metric.CACHE_READ_INPUT_TOKENS]: "Cache-read input tokens",
  [Metric.CACHE_WRITE_INPUT_TOKENS]: "Cache-write input tokens",
  [Metric.REASONING_OUTPUT_TOKENS]: "Reasoning tokens",
  [Metric.ESTIMATED_INPUT_TOKENS]: "Estimated input tokens",
  [Metric.LATENCY_AVG]: "Average latency",
  [Metric.LATENCY_P95]: "P95 latency",
  [Metric.LATENCY_MAX]: "Max latency",
  [Metric.TIME_TO_FIRST_TOKEN_AVG]: "Average TTFT",
  [Metric.TIME_TO_FIRST_TOKEN_P95]: "P95 TTFT",
  [Metric.TOOL_CALLS]: "Tool calls",
  [Metric.STREAM_EVENTS]: "Stream events",
  [Metric.UPSTREAM_INVOKED_REQUESTS]: "Upstream-invoked requests",
  [Metric.DISCARDED_TOKENS]: "Discarded tokens",
  [Metric.TOKEN_RATE_LIMIT_DENIED_REQUESTS]: "Token quota denials",
  [Metric.TOOLS_OFFERED]: "Tools offered",
  [Metric.REQUEST_BODY_BYTES]: "Request bytes",
  [Metric.RESPONSE_BODY_BYTES]: "Response bytes",
};

export const metricLabel = (metric: Metric): string =>
  METRIC_LABELS[metric] ?? Metric[metric] ?? "Metric";

export type MetricAccessor = {
  metric: Metric;
  label: string;
  kind: "count" | "tokens" | "duration" | "bytes";
  get: (stats?: Stats) => number;
};

export const METRIC_ACCESSORS: MetricAccessor[] = [
  {
    metric: Metric.REQUESTS,
    label: "Requests",
    kind: "count",
    get: (s) => num(s?.requests?.total),
  },
  {
    metric: Metric.DENIED_REQUESTS,
    label: "Denied requests",
    kind: "count",
    get: (s) => num(s?.requests?.denied),
  },
  {
    metric: Metric.FAILED_REQUESTS,
    label: "Failed requests",
    kind: "count",
    get: (s) => num(s?.requests?.failed),
  },
  {
    metric: Metric.TOTAL_TOKENS,
    label: "Total tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.total),
  },
  {
    metric: Metric.INPUT_TOKENS,
    label: "Input tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.input),
  },
  {
    metric: Metric.OUTPUT_TOKENS,
    label: "Output tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.output),
  },
  {
    metric: Metric.REASONING_OUTPUT_TOKENS,
    label: "Reasoning tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.reasoningOutput),
  },
  {
    metric: Metric.CACHE_READ_INPUT_TOKENS,
    label: "Cache-read tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.cacheReadInput),
  },
  {
    metric: Metric.DISCARDED_TOKENS,
    label: "Discarded tokens",
    kind: "tokens",
    get: (s) => num(s?.tokens?.discarded),
  },
  {
    metric: Metric.LATENCY_P95,
    label: "P95 latency",
    kind: "duration",
    get: (s) => s?.latency?.p95Ms ?? 0,
  },
  {
    metric: Metric.LATENCY_AVG,
    label: "Average latency",
    kind: "duration",
    get: (s) => s?.latency?.avgMs ?? 0,
  },
  {
    metric: Metric.TIME_TO_FIRST_TOKEN_P95,
    label: "P95 TTFT",
    kind: "duration",
    get: (s) => s?.timeToFirstToken?.p95Ms ?? 0,
  },
  {
    metric: Metric.GUARDRAIL_DENIED_REQUESTS,
    label: "Guardrail denials",
    kind: "count",
    get: (s) => num(s?.requests?.guardrailDenied),
  },
  {
    metric: Metric.CACHED_REQUESTS,
    label: "Cache-served requests",
    kind: "count",
    get: (s) => num(s?.requests?.cacheExactHit) + num(s?.requests?.cacheSemanticHit),
  },
  {
    metric: Metric.TOOL_CALLS,
    label: "Tool calls",
    kind: "count",
    get: (s) => num(s?.toolCalls),
  },
];

export const metricAccessor = (metric: Metric): MetricAccessor =>
  METRIC_ACCESSORS.find((item) => item.metric === metric) ??
  METRIC_ACCESSORS[0];

export const formatMetricValue = (
  kind: MetricAccessor["kind"],
  value: number,
): string => {
  switch (kind) {
    case "duration":
      return formatMs(value);
    case "tokens":
      return formatTokens(value);
    case "bytes":
      return formatTokens(value);
    default:
      return value.toLocaleString();
  }
};

const ENUM_LABEL_OVERRIDES: Record<string, string> = {
  SOURCE_UNSET: "Not set",
  STATE_UNSET: "Not set",
  RESULT_UNSET: "Not set",
  FINISH_REASON_UNSET: "Not set",
  ESTIMATE_QUALITY_UNSET: "Not set",
  SCOPE_UNSET: "Not set",
  LEG_UNSET: "Not set",
  ROUTE_UNSET: "Not set",
  TYPE_UNSET: "Not set",
  OPERATION_UNSET: "Not set",
  PROTOCOL_UNSET: "Not set",
  MODE_UNSET: "Not set",
  true: "Yes",
  false: "No",
};

export const prettyKey = (key: string): string => {
  if (!key) return "—";
  const override = ENUM_LABEL_OVERRIDES[key];
  if (override) return override;
  if (!/^[A-Z0-9_]+$/.test(key)) return key;
  return key
    .toLowerCase()
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
};

export const cacheHitRate = (stats?: Stats): number => {
  const hits =
    num(stats?.requests?.cacheExactHit) +
    num(stats?.requests?.cacheSemanticHit);
  const considered = hits + num(stats?.requests?.cacheMiss);
  return ratio(hits, considered);
};

export const isEmptyStats = (stats?: Stats): boolean =>
  num(stats?.requests?.total) === 0;

export type Drilldown = {
  dimension: Dimension;
  key: string;
};

const enumValue = (enumObj: any, key: string): number | undefined => {
  const value = enumObj?.[key];
  return typeof value === "number" ? value : undefined;
};

export const DRILLDOWN_DIMENSIONS = new Set<Dimension>([
  Dimension.MODEL,
  Dimension.MODEL_REQUESTED,
  Dimension.MODEL_REPORTED,
  Dimension.MODEL_SOURCE,
  Dimension.MODEL_PLUGIN,
  Dimension.PROTOCOL,
  Dimension.OPERATION,
  Dimension.ROUTE,
  Dimension.SOURCE,
  Dimension.USAGE_STATE,
  Dimension.ESTIMATE_QUALITY,
  Dimension.GUARDRAIL_RESULT,
  Dimension.GUARDRAIL_PLUGIN,
  Dimension.GUARDRAIL_LEG,
  Dimension.FINISH_REASON,
  Dimension.FINISH_REASON_RAW,
  Dimension.TOOL,
  Dimension.CALLED_TOOL,
  Dimension.REMOVED_TOOL,
  Dimension.STATUS,
  Dimension.HTTP_STATUS_CODE,
  Dimension.HTTP_STATUS_CLASS,
  Dimension.USER_AGENT,
  Dimension.HTTP_PATH,
  Dimension.SEMANTIC_CACHE_RESULT,
  Dimension.SEMANTIC_CACHE_PLUGIN,
  Dimension.SEMANTIC_ROUTER_RESULT,
  Dimension.SEMANTIC_ROUTER_ROUTE,
  Dimension.SEMANTIC_ROUTER_PLUGIN,
  Dimension.SEMANTIC_ROUTER_MODEL,
  Dimension.TOKEN_RATE_LIMIT_RESULT,
  Dimension.TOKEN_RATE_LIMIT_PLUGIN,
  Dimension.TOKEN_RATE_LIMIT_SCOPE,
  Dimension.IS_STREAM,
  Dimension.IS_UPSTREAM_INVOKED,
  Dimension.HAS_IMAGE_INPUT,
  Dimension.HAS_AUDIO_INPUT,
]);

const pushString = (filter: Filter, field: keyof Filter, value: string) => {
  const list = filter[field] as unknown as string[];
  if (!list.includes(value)) list.push(value);
};

const pushEnum = (
  filter: Filter,
  field: keyof Filter,
  enumObj: any,
  key: string,
) => {
  const value = enumValue(enumObj, key);
  if (value === undefined) return;
  const list = filter[field] as unknown as number[];
  if (!list.includes(value)) list.push(value);
};

export const applyDrilldowns = (
  filter: Filter,
  drilldowns: Drilldown[],
): Filter => {
  drilldowns.forEach(({ dimension, key }) => {
    switch (dimension) {
      case Dimension.MODEL:
        filter.modelField = ModelField.EFFECTIVE;
        pushString(filter, "models", key);
        break;
      case Dimension.MODEL_REQUESTED:
        filter.modelField = ModelField.REQUESTED;
        pushString(filter, "models", key);
        break;
      case Dimension.MODEL_REPORTED:
        filter.modelField = ModelField.REPORTED;
        pushString(filter, "models", key);
        break;
      case Dimension.MODEL_SOURCE:
        pushEnum(
          filter,
          "modelSources",
          CoreP.AccessLog_Entry_Info_LLM_Model_Source,
          key,
        );
        break;
      case Dimension.MODEL_PLUGIN:
        pushString(filter, "modelPlugins", key);
        break;
      case Dimension.PROTOCOL:
        pushEnum(
          filter,
          "protocols",
          CoreP.Service_Spec_Config_LLM_Protocol,
          key,
        );
        break;
      case Dimension.OPERATION:
        pushEnum(
          filter,
          "operations",
          CoreP.Service_Spec_Config_LLM_Operation,
          key,
        );
        break;
      case Dimension.ROUTE:
        pushEnum(filter, "routes", CoreP.RequestContext_Request_LLM_Route, key);
        break;
      case Dimension.SOURCE:
        pushEnum(filter, "sources", CoreP.AccessLog_Entry_Info_LLM_Source, key);
        break;
      case Dimension.USAGE_STATE:
        pushEnum(
          filter,
          "usageStates",
          CoreP.AccessLog_Entry_Info_LLM_Usage_State,
          key,
        );
        break;
      case Dimension.ESTIMATE_QUALITY:
        pushEnum(
          filter,
          "estimateQualities",
          CoreP.RequestContext_Request_LLM_EstimateQuality,
          key,
        );
        break;
      case Dimension.GUARDRAIL_RESULT:
        pushEnum(
          filter,
          "guardrailResults",
          CoreP.AccessLog_Entry_Info_LLM_Guardrail_Result,
          key,
        );
        break;
      case Dimension.GUARDRAIL_LEG:
        pushEnum(
          filter,
          "guardrailLegs",
          CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Leg,
          key,
        );
        break;
      case Dimension.GUARDRAIL_PLUGIN:
        pushString(filter, "guardrailPlugins", key);
        break;
      case Dimension.FINISH_REASON:
        pushEnum(
          filter,
          "finishReasons",
          CoreP.AccessLog_Entry_Info_LLM_FinishReason,
          key,
        );
        break;
      case Dimension.FINISH_REASON_RAW:
        pushString(filter, "rawFinishReasons", key);
        break;
      case Dimension.TOOL:
        pushString(filter, "tools", key);
        break;
      case Dimension.CALLED_TOOL:
        pushString(filter, "calledTools", key);
        break;
      case Dimension.REMOVED_TOOL:
        pushString(filter, "removedTools", key);
        break;
      case Dimension.STATUS:
        filter.status = enumValue(CoreP.AccessLog_Entry_Common_Status, key) ?? 0;
        break;
      case Dimension.HTTP_STATUS_CODE: {
        const code = Number(key);
        if (Number.isFinite(code) && !filter.httpStatusCodes.includes(code)) {
          filter.httpStatusCodes.push(code);
        }
        break;
      }
      case Dimension.HTTP_STATUS_CLASS:
        pushEnum(filter, "httpStatusClasses", HTTPStatusClass, key);
        break;
      case Dimension.USER_AGENT:
        pushString(filter, "userAgents", key);
        break;
      case Dimension.HTTP_PATH:
        pushString(filter, "httpPaths", key);
        break;
      case Dimension.SEMANTIC_CACHE_RESULT:
        pushEnum(
          filter,
          "semanticCacheResults",
          CoreP.AccessLog_Entry_Info_LLM_SemanticCache_Result,
          key,
        );
        break;
      case Dimension.SEMANTIC_CACHE_PLUGIN:
        pushString(filter, "semanticCachePlugins", key);
        break;
      case Dimension.SEMANTIC_ROUTER_RESULT:
        pushEnum(
          filter,
          "semanticRouterResults",
          CoreP.AccessLog_Entry_Info_LLM_SemanticRouter_Result,
          key,
        );
        break;
      case Dimension.SEMANTIC_ROUTER_ROUTE:
        pushString(filter, "semanticRouterRoutes", key);
        break;
      case Dimension.SEMANTIC_ROUTER_PLUGIN:
        pushString(filter, "semanticRouterPlugins", key);
        break;
      case Dimension.SEMANTIC_ROUTER_MODEL:
        filter.modelField = ModelField.EFFECTIVE;
        pushString(filter, "models", key);
        break;
      case Dimension.TOKEN_RATE_LIMIT_RESULT:
        pushEnum(
          filter,
          "tokenRateLimitResults",
          CoreP.AccessLog_Entry_Info_LLM_TokenRateLimit_Result,
          key,
        );
        break;
      case Dimension.TOKEN_RATE_LIMIT_PLUGIN:
        pushString(filter, "tokenRateLimitPlugins", key);
        break;
      case Dimension.TOKEN_RATE_LIMIT_SCOPE:
        pushEnum(
          filter,
          "tokenRateLimitScopes",
          CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope,
          key,
        );
        break;
      case Dimension.IS_STREAM:
        filter.stream = key === "true";
        break;
      case Dimension.IS_UPSTREAM_INVOKED:
        filter.isUpstreamInvoked = key === "true";
        break;
      case Dimension.HAS_IMAGE_INPUT:
        filter.hasImageInput = key === "true";
        break;
      case Dimension.HAS_AUDIO_INPUT:
        filter.hasAudioInput = key === "true";
        break;
      default:
        break;
    }
  });

  return filter;
};

export const drilldownLabel = (drilldown: Drilldown): string =>
  `${dimensionLabel(drilldown.dimension)}: ${prettyKey(drilldown.key)}`;

export const drilldownID = (drilldown: Drilldown): string =>
  `${drilldown.dimension}:${drilldown.key}`;

export const ModelFieldDimension: Record<number, Dimension> = {
  [ModelField.EFFECTIVE]: Dimension.MODEL,
  [ModelField.REQUESTED]: Dimension.MODEL_REQUESTED,
  [ModelField.REPORTED]: Dimension.MODEL_REPORTED,
};
