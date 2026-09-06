import * as CoreP from "@/apis/corev1/corev1";
import {
  Filter,
  ListAccessLogRequest,
  ListAccessLogRequest_OrderBy_Mode,
  ListAccessLogRequest_OrderBy_Type,
} from "@/apis/visibilityv1/llm/vllmv1";
import { Duration } from "@/apis/metav1/metav1";
import LogEditor from "@/components/AccessLogViewer/Editor";
import Paginator from "@/components/AccessLogViewer/Paginator";
import TimeAgo from "@/components/TimeAgo";
import { getClientVisibilityLLM } from "@/utils/client";
import { Badge, Select, SegmentedControl, Tooltip } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowDownWideNarrow,
  ArrowUpWideNarrow,
  Ban,
  ChevronDown,
  DatabaseZap,
  ShieldAlert,
  Wrench,
  Zap,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { QueryState } from "./Primitives";
import { formatMs, formatTokens, num, prettyKey } from "./utils";

const ORDER_OPTIONS = [
  { value: String(ListAccessLogRequest_OrderBy_Type.CREATED_AT), label: "Time" },
  {
    value: String(ListAccessLogRequest_OrderBy_Type.TOTAL_TOKENS),
    label: "Total tokens",
  },
  {
    value: String(ListAccessLogRequest_OrderBy_Type.OUTPUT_TOKENS),
    label: "Output tokens",
  },
  { value: String(ListAccessLogRequest_OrderBy_Type.LATENCY), label: "Latency" },
  {
    value: String(ListAccessLogRequest_OrderBy_Type.TIME_TO_FIRST_TOKEN),
    label: "TTFT",
  },
  {
    value: String(ListAccessLogRequest_OrderBy_Type.TOOL_CALLS),
    label: "Tool calls",
  },
];

const getLLM = (item: CoreP.AccessLog) => {
  const type = item.entry?.info?.type;
  return type?.oneofKind === "llm" ? type.llm : undefined;
};

const durationMs = (duration?: Duration): number => {
  const type = duration?.type;
  switch (type?.oneofKind) {
    case "milliseconds":
      return type.milliseconds;
    case "seconds":
      return type.seconds * 1000;
    case "minutes":
      return type.minutes * 60000;
    default:
      return 0;
  }
};

const entryLatencyMs = (item: CoreP.AccessLog): number => {
  const started = item.entry?.common?.startedAt;
  const ended = item.entry?.common?.endedAt;
  if (!started || !ended) return 0;
  return (
    new Date(Number(ended.seconds) * 1000 + ended.nanos / 1e6).getTime() -
    new Date(Number(started.seconds) * 1000 + started.nanos / 1e6).getTime()
  );
};

const Chip = (props: {
  children: React.ReactNode;
  tone?: "default" | "danger" | "warning" | "positive" | "accent";
  icon?: React.ElementType<{ size?: number; strokeWidth?: number }>;
  title?: string;
}) => {
  const Icon = props.icon;
  const tone = props.tone ?? "default";
  return (
    <span
      title={props.title}
      className={twMerge(
        "inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[0.62rem] font-bold",
        tone === "default" && "border-slate-200 bg-slate-50 text-slate-600",
        tone === "danger" && "border-red-200 bg-red-50 text-red-700",
        tone === "warning" && "border-amber-200 bg-amber-50 text-amber-700",
        tone === "positive" && "border-emerald-200 bg-emerald-50 text-emerald-700",
        tone === "accent" && "border-indigo-200 bg-indigo-50 text-indigo-700",
      )}
    >
      {Icon && <Icon size={10} strokeWidth={2.6} />}
      {props.children}
    </span>
  );
};

const Row = (props: { item: CoreP.AccessLog }) => {
  const [opened, setOpened] = React.useState(false);
  const llm = getLLM(props.item);
  const common = props.item.entry?.common;
  const denied = common?.status === CoreP.AccessLog_Entry_Common_Status.DENIED;
  const usage = llm?.usage;
  const latency = entryLatencyMs(props.item);
  const ttft = durationMs(llm?.timeToFirstToken);
  const guardrailDenied = (llm?.guardrails ?? []).some(
    (item) =>
      item.result === CoreP.AccessLog_Entry_Info_LLM_Guardrail_Result.DENIED,
  );
  const cacheHit =
    llm?.semanticCache?.result ===
      CoreP.AccessLog_Entry_Info_LLM_SemanticCache_Result.EXACT_HIT ||
    llm?.semanticCache?.result ===
      CoreP.AccessLog_Entry_Info_LLM_SemanticCache_Result.SEMANTIC_HIT;
  const statusCode = llm?.http?.response?.code ?? 0;

  return (
    <div
      className={twMerge(
        "overflow-hidden rounded-xl border bg-white transition-[border-color,box-shadow] duration-400",
        opened
          ? "border-slate-300 shadow-[0_4px_14px_rgba(15,23,42,0.07)]"
          : "border-slate-200",
      )}
    >
      <button
        type="button"
        onClick={() => setOpened((value) => !value)}
        aria-expanded={opened}
        className="flex w-full cursor-pointer items-center gap-3 px-3 py-2.5 text-left outline-none transition-colors duration-300 hover:bg-slate-50/70"
      >
        <span
          aria-hidden="true"
          className={twMerge(
            "h-8 w-1 shrink-0 rounded-full",
            denied ? "bg-red-500" : statusCode >= 500 ? "bg-amber-500" : "bg-emerald-500",
          )}
        />
        <span className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex min-w-0 flex-wrap items-center gap-1.5">
            <span className="truncate text-[0.76rem] font-bold text-slate-800">
              {llm?.model?.effective || llm?.model?.requested || "—"}
            </span>
            <Chip tone="accent">
              {prettyKey(
                CoreP.Service_Spec_Config_LLM_Protocol[llm?.protocol ?? 0] ?? "",
              )}
            </Chip>
            <Chip>
              {prettyKey(
                CoreP.Service_Spec_Config_LLM_Operation[llm?.operation ?? 0] ??
                  "",
              )}
            </Chip>
            {llm?.stream && <Chip icon={Zap}>Stream</Chip>}
            {cacheHit && (
              <Chip tone="positive" icon={DatabaseZap}>
                Cache
              </Chip>
            )}
            {guardrailDenied && (
              <Chip tone="danger" icon={ShieldAlert}>
                Guardrail
              </Chip>
            )}
            {denied && (
              <Chip tone="danger" icon={Ban}>
                Denied
              </Chip>
            )}
            {num(llm?.tools?.callCount) > 0 && (
              <Chip icon={Wrench}>
                {num(llm?.tools?.callCount)} call
                {num(llm?.tools?.callCount) === 1 ? "" : "s"}
              </Chip>
            )}
          </span>
          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-[0.65rem] font-semibold text-slate-400">
            <span className="truncate">
              {common?.userRef?.name ?? "anonymous"}
            </span>
            <span className="truncate">{common?.serviceRef?.name}</span>
            <span className="truncate font-mono">
              {llm?.http?.request?.path}
            </span>
            {props.item.metadata?.createdAt && (
              <TimeAgo rfc3339={props.item.metadata.createdAt} />
            )}
          </span>
        </span>

        <span className="hidden shrink-0 items-center gap-4 sm:flex">
          <Tooltip label="Input / output tokens" withArrow>
            <span className="text-right">
              <span className="block text-[0.58rem] font-bold uppercase tracking-[0.06em] text-slate-400">
                Tokens
              </span>
              <span className="block text-[0.72rem] font-bold tabular-nums text-slate-700">
                {formatTokens(num(usage?.inputTokens))} /{" "}
                {formatTokens(num(usage?.outputTokens))}
              </span>
            </span>
          </Tooltip>
          <Tooltip label="Latency · time to first token" withArrow>
            <span className="text-right">
              <span className="block text-[0.58rem] font-bold uppercase tracking-[0.06em] text-slate-400">
                Latency
              </span>
              <span className="block text-[0.72rem] font-bold tabular-nums text-slate-700">
                {formatMs(latency)}
                {ttft > 0 ? ` · ${formatMs(ttft)}` : ""}
              </span>
            </span>
          </Tooltip>
        </span>

        <ChevronDown
          size={14}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-slate-400 transition-transform duration-400",
            opened && "rotate-180",
          )}
        />
      </button>

      <AnimatePresence initial={false}>
        {opened && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden border-t border-slate-100 bg-slate-50/50"
          >
            <div className="grid gap-3 px-3.5 py-3 sm:grid-cols-2 lg:grid-cols-3">
              <Detail label="Requested model" value={llm?.model?.requested} />
              <Detail label="Effective model" value={llm?.model?.effective} />
              <Detail label="Reported model" value={llm?.model?.reported} />
              <Detail
                label="Model source"
                value={prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_Model_Source[
                    llm?.model?.source ?? 0
                  ] ?? "",
                )}
              />
              <Detail
                label="Route"
                value={prettyKey(
                  CoreP.RequestContext_Request_LLM_Route[llm?.route ?? 0] ?? "",
                )}
              />
              <Detail
                label="Response source"
                value={prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_Source[llm?.source ?? 0] ?? "",
                )}
              />
              <Detail
                label="Finish reason"
                value={`${prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_FinishReason[
                    llm?.finishReason ?? 0
                  ] ?? "",
                )}${llm?.rawFinishReason ? ` (${llm.rawFinishReason})` : ""}`}
              />
              <Detail
                label="Usage state"
                value={prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_Usage_State[
                    usage?.state ?? 0
                  ] ?? "",
                )}
              />
              <Detail
                label="Estimated input"
                value={formatTokens(num(llm?.estimatedInputTokens))}
              />
              <Detail
                label="Cache read / write"
                value={`${formatTokens(num(usage?.cacheReadInputTokens))} / ${formatTokens(num(usage?.cacheWriteInputTokens))}`}
              />
              <Detail
                label="Reasoning tokens"
                value={formatTokens(num(usage?.reasoningOutputTokens))}
              />
              <Detail
                label="Stream events"
                value={num(llm?.eventCount).toLocaleString()}
              />
              <Detail
                label="Tools offered / called / removed"
                value={`${num(llm?.tools?.count)} / ${num(llm?.tools?.callCount)} / ${num(llm?.tools?.removedCount)}`}
              />
              <Detail
                label="Token quota"
                value={prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_TokenRateLimit_Result[
                    llm?.tokenRateLimit?.result ?? 0
                  ] ?? "",
                )}
              />
              <Detail
                label="Semantic cache"
                value={prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_SemanticCache_Result[
                    llm?.semanticCache?.result ?? 0
                  ] ?? "",
                )}
              />
              <Detail
                label="Semantic router"
                value={`${prettyKey(
                  CoreP.AccessLog_Entry_Info_LLM_SemanticRouter_Result[
                    llm?.semanticRouter?.result ?? 0
                  ] ?? "",
                )}${llm?.semanticRouter?.route ? ` → ${llm.semanticRouter.route}` : ""}`}
              />
              <Detail label="HTTP status" value={String(statusCode || "—")} />
              <Detail label="Response ID" value={llm?.responseID} />
            </div>

            {(llm?.guardrails?.length ?? 0) > 0 && (
              <div className="flex flex-wrap gap-1.5 border-t border-slate-100 px-3.5 py-2.5">
                {llm!.guardrails.map((guardrail, index) => (
                  <Chip
                    key={`${guardrail.plugin}-${index}`}
                    tone={
                      guardrail.result ===
                      CoreP.AccessLog_Entry_Info_LLM_Guardrail_Result.DENIED
                        ? "danger"
                        : guardrail.result ===
                            CoreP.AccessLog_Entry_Info_LLM_Guardrail_Result
                              .ERROR
                          ? "warning"
                          : "default"
                    }
                    icon={ShieldAlert}
                  >
                    {guardrail.plugin || "guardrail"} ·{" "}
                    {prettyKey(
                      CoreP.AccessLog_Entry_Info_LLM_Guardrail_Result[
                        guardrail.result
                      ] ?? "",
                    )}{" "}
                    ·{" "}
                    {prettyKey(
                      CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Leg[
                        guardrail.leg
                      ] ?? "",
                    )}
                  </Chip>
                ))}
              </div>
            )}

            <div className="flex items-center justify-end gap-2 border-t border-slate-100 px-3.5 py-2.5">
              <LogEditor item={props.item} />
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

const Detail = (props: { label: string; value?: string }) => (
  <div className="min-w-0">
    <p className="truncate text-[0.6rem] font-bold uppercase tracking-[0.06em] text-slate-400">
      {props.label}
    </p>
    <p className="mt-0.5 truncate text-[0.72rem] font-bold text-slate-700">
      {props.value || "—"}
    </p>
  </div>
);

const AccessLogs = (props: {
  filter: Filter;
  enabled?: boolean;
  itemsPerPage?: number;
}) => {
  const [page, setPage] = React.useState(0);
  const [orderType, setOrderType] = React.useState(
    ListAccessLogRequest_OrderBy_Type.CREATED_AT,
  );
  const [mode, setMode] = React.useState(
    ListAccessLogRequest_OrderBy_Mode.DESC,
  );
  const enabled = props.enabled ?? true;
  const itemsPerPage = props.itemsPerPage ?? 25;

  React.useEffect(() => {
    setPage(0);
  }, [props.filter, orderType, mode]);

  const qry = useQuery({
    queryKey: [
      "visibility",
      "llm",
      "accessLog",
      { filter: props.filter, page, orderType, mode, itemsPerPage },
    ],
    enabled,
    queryFn: async () => {
      const { response } = await getClientVisibilityLLM().listAccessLog(
        ListAccessLogRequest.create({
          filter: props.filter,
          page,
          itemsPerPage,
          orderBy: { type: orderType, mode },
        }),
      );
      return response;
    },
  });

  return (
    <div className="flex w-full flex-col gap-3">
      <div className="flex flex-wrap items-end gap-2">
        <Select
          size="xs"
          label="Sort by"
          className="min-w-[160px]"
          allowDeselect={false}
          value={String(orderType)}
          data={ORDER_OPTIONS}
          onChange={(value) => value && setOrderType(Number(value))}
        />
        <SegmentedControl
          size="xs"
          value={String(mode)}
          onChange={(value) => setMode(Number(value))}
          data={[
            {
              value: String(ListAccessLogRequest_OrderBy_Mode.DESC),
              label: (
                <span className="flex items-center gap-1 px-1">
                  <ArrowDownWideNarrow size={12} strokeWidth={2.5} />
                  Desc
                </span>
              ),
            },
            {
              value: String(ListAccessLogRequest_OrderBy_Mode.ASC),
              label: (
                <span className="flex items-center gap-1 px-1">
                  <ArrowUpWideNarrow size={12} strokeWidth={2.5} />
                  Asc
                </span>
              ),
            },
          ]}
        />
        <div className="flex-1" />
        {qry.data?.listResponseMeta && (
          <Badge size="sm" variant="light" color="gray">
            {num(qry.data.listResponseMeta.totalCount).toLocaleString()} entries
          </Badge>
        )}
      </div>

      <QueryState
        isLoading={enabled && qry.isPending}
        isError={qry.isError}
        isEmpty={enabled && qry.isSuccess && (qry.data?.items.length ?? 0) === 0}
        minHeight={200}
      >
        <div className="flex flex-col gap-1.5">
          {qry.data?.items.map((item) => (
            <Row key={item.metadata?.id} item={item} />
          ))}
        </div>
        <div className="mt-4">
          <Paginator
            listResponseMeta={qry.data?.listResponseMeta}
            onPageChange={(value) => setPage(value - 1)}
          />
        </div>
      </QueryState>
    </div>
  );
};

export default AccessLogs;
