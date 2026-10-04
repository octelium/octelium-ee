import type {
  ApprovalBlock,
  ApprovalDecision,
  ArtifactBlock,
  Block,
  ChartBlock,
  ChartSpec,
  ErrorBlock,
  NoticeBlock,
  ResourcesBlock,
  TableBlock,
  TableColumn,
  ThinkingBlock,
  ToolBlock,
} from "@/apis/consoleagent/protocol";
import Markdown from "@/components/LLMPlayground/Markdown";
import {
  CHART_FONT,
  CHART_INK,
  CHART_TOOLTIP,
  seriesColors,
} from "@/utils/charts/palette";
import { Badge, Button, Loader, Tooltip } from "@mantine/core";
import ReactEChartsCore from "echarts-for-react";
import { BarChart, LineChart, PieChart, ScatterChart } from "echarts/charts";
import {
  AriaComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { AnimatePresence, motion } from "framer-motion";
import {
  Ban,
  Box,
  Brain,
  ChevronDown,
  CircleAlert,
  CircleCheck,
  CircleSlash,
  CircleX,
  Database,
  Download,
  FileText,
  Info,
  ShieldAlert,
  ShieldQuestion,
  Table as TableIcon,
  Terminal,
  TriangleAlert,
  Wrench,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { AgentClient } from "./client";
import {
  formatBytes,
  formatValue,
  getResourceRoute,
  getResourceURIRoute,
  riskColor,
  saveBlob,
  toCSV,
  toJSON,
  urlTransform,
} from "./utils";

echarts.use([
  AriaComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  BarChart,
  LineChart,
  PieChart,
  ScatterChart,
  CanvasRenderer,
]);

export interface BlockContext {
  client: AgentClient;
  runId?: string;
  streaming: boolean;
  onDecide?: (
    runId: string,
    approvalId: string,
    decision: ApprovalDecision,
  ) => Promise<void>;
}

const Collapse = (props: { opened: boolean; children: React.ReactNode }) => (
  <AnimatePresence initial={false}>
    {props.opened && (
      <motion.div
        initial={{ height: 0, opacity: 0 }}
        animate={{ height: "auto", opacity: 1 }}
        exit={{ height: 0, opacity: 0 }}
        transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
        className="overflow-hidden"
      >
        {props.children}
      </motion.div>
    )}
  </AnimatePresence>
);

const Code = (props: { children: string; className?: string }) => (
  <pre
    className={twMerge(
      "max-h-[320px] overflow-auto rounded-lg border border-slate-200 bg-slate-50 px-3 py-2",
      props.className,
    )}
  >
    <code className="whitespace-pre font-mono text-xs leading-5 text-slate-700">
      {props.children}
    </code>
  </pre>
);

const Section = (props: { label: string; children: React.ReactNode }) => (
  <div className="mt-2.5 first:mt-0">
    <p className="mb-1 text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
      {props.label}
    </p>
    {props.children}
  </div>
);

const AgentLink = (props: { href?: string; children?: React.ReactNode }) => {
  const route = props.href ? getResourceURIRoute(props.href) : undefined;
  const className =
    "font-semibold text-blue-600 underline decoration-blue-300 underline-offset-2 hover:text-blue-700";

  if (route) {
    return (
      <Link to={route} className={className}>
        {props.children}
      </Link>
    );
  }

  if (props.href?.startsWith("octelium://")) {
    return (
      <span className="font-semibold text-slate-800">{props.children}</span>
    );
  }

  return (
    <a
      href={props.href}
      target="_blank"
      rel="noreferrer noopener"
      className={className}
    >
      {props.children}
    </a>
  );
};

export const AgentMarkdown = (props: { children: string }) => (
  <Markdown urlTransform={urlTransform} components={{ a: AgentLink }}>
    {props.children}
  </Markdown>
);

const ThinkingView = (props: { block: ThinkingBlock; streaming: boolean }) => {
  const [opened, setOpened] = React.useState(false);
  const { block } = props;

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-violet-200 bg-violet-50/60">
      <button
        type="button"
        onClick={() => setOpened((value) => !value)}
        aria-expanded={opened}
        className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left outline-none transition-colors duration-200 hover:bg-violet-100/50"
      >
        <Brain
          size={13}
          strokeWidth={2.4}
          className="shrink-0 text-violet-600"
        />
        <span className="flex-1 text-xs font-semibold uppercase tracking-[0.06em] text-violet-700">
          {props.streaming ? "Thinking…" : "Reasoning"}
        </span>
        <ChevronDown
          size={13}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-violet-500 transition-transform duration-200",
            opened && "rotate-180",
          )}
        />
      </button>
      <Collapse opened={opened}>
        <div className="max-h-[320px] overflow-y-auto border-t border-violet-200 px-3 py-2.5">
          <p className="whitespace-pre-wrap text-body leading-6 text-violet-900/80">
            {block.redacted ? "The reasoning is redacted." : block.text}
          </p>
        </div>
      </Collapse>
    </div>
  );
};

const ToolStatusIcon = (props: { block: ToolBlock }) => {
  switch (props.block.status) {
    case "pending":
    case "running":
      return <Loader size={13} color="gray" />;
    case "awaiting_approval":
      return <ShieldQuestion size={14} className="text-amber-600" />;
    case "completed":
      return <CircleCheck size={14} className="text-emerald-600" />;
    case "failed":
      return <CircleX size={14} className="text-red-600" />;
    case "rejected":
      return <Ban size={14} className="text-red-600" />;
    case "cancelled":
      return <CircleSlash size={14} className="text-slate-500" />;
  }
};

const toolStatusLabel: Record<ToolBlock["status"], string> = {
  pending: "Preparing",
  running: "Running",
  awaiting_approval: "Awaiting approval",
  completed: "Done",
  failed: "Failed",
  rejected: "Rejected",
  cancelled: "Cancelled",
};

const ToolKindIcon = (props: { block: ToolBlock }) => {
  switch (props.block.details?.kind) {
    case "octelium_api":
      return <Database size={13} className="shrink-0 text-slate-500" />;
    case "command":
      return <Terminal size={13} className="shrink-0 text-slate-500" />;
    case "file":
      return <FileText size={13} className="shrink-0 text-slate-500" />;
    default:
      return <Wrench size={13} className="shrink-0 text-slate-500" />;
  }
};

const ResourceChip = (props: {
  resource: { apiVersion: string; kind: string; name?: string };
}) => {
  const route = getResourceRoute(props.resource);
  const label = (
    <>
      <span className="text-slate-500">{props.resource.kind}</span>
      <span className="font-semibold">{props.resource.name}</span>
    </>
  );

  return route ? (
    <Link
      to={route}
      className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-micro text-slate-800 transition-colors duration-200 hover:border-slate-300 hover:bg-slate-50"
    >
      {label}
    </Link>
  ) : (
    <span className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-micro text-slate-800">
      {label}
    </span>
  );
};

const ToolDetailsView = (props: { block: ToolBlock }) => {
  const { block } = props;
  const details = block.details;

  if (details?.kind === "octelium_api") {
    return (
      <>
        <Section label="Method">
          <div className="flex flex-wrap items-center gap-2">
            <code className="font-mono text-xs font-semibold text-slate-800">
              {details.method}
            </code>
            <Badge size="xs" variant="light" color={riskColor(details.risk)}>
              {details.risk}
            </Badge>
          </div>
        </Section>
        {details.request !== undefined && (
          <Section label="Request">
            <Code>{toJSON(details.request)}</Code>
          </Section>
        )}
        {details.error ? (
          <Section label="Error">
            <p className="text-xs font-semibold text-red-700">
              {details.error.code ? `${details.error.code}: ` : ""}
              {details.error.message}
            </p>
          </Section>
        ) : details.response !== undefined ? (
          <Section
            label={
              details.responseTruncated ? "Response (truncated)" : "Response"
            }
          >
            <Code>{toJSON(details.response)}</Code>
          </Section>
        ) : null}
        {details.resultPath && (
          <Section label="Saved result">
            <code className="font-mono text-xs text-slate-700">
              {details.resultPath}
            </code>
          </Section>
        )}
        {details.resources && details.resources.length > 0 && (
          <Section label="Resources">
            <div className="flex flex-wrap gap-1">
              {details.resources.slice(0, 50).map((ref) => (
                <ResourceChip
                  key={`${ref.apiVersion}/${ref.kind}/${ref.name}`}
                  resource={ref}
                />
              ))}
            </div>
          </Section>
        )}
      </>
    );
  }

  return (
    <>
      {details?.kind === "command" ? (
        <Section label="Command">
          <Code>{`$ ${details.command}`}</Code>
        </Section>
      ) : details?.kind === "file" ? (
        <Section label={details.operation}>
          <code className="font-mono text-xs text-slate-700">
            {details.path}
          </code>
        </Section>
      ) : (
        block.input !== undefined && (
          <Section label="Input">
            <Code>{toJSON(block.input)}</Code>
          </Section>
        )
      )}
      {details?.kind === "file" && details.diff && (
        <Section label="Diff">
          <Code>{details.diff}</Code>
        </Section>
      )}
      {block.output && (
        <Section
          label={
            details?.kind === "command" && details.exitCode !== undefined
              ? `Output (exit code ${details.exitCode})`
              : block.outputTruncated
                ? "Output (truncated)"
                : "Output"
          }
        >
          <Code>{block.output}</Code>
        </Section>
      )}
    </>
  );
};

const ToolView = (props: { block: ToolBlock }) => {
  const [opened, setOpened] = React.useState(false);
  const { block } = props;

  return (
    <div className="my-1.5 overflow-hidden rounded-xl border border-slate-200 bg-slate-50/60">
      <button
        type="button"
        onClick={() => setOpened((value) => !value)}
        aria-expanded={opened}
        className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left outline-none transition-colors duration-200 hover:bg-slate-100/70"
      >
        <ToolKindIcon block={block} />
        <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
          {block.title || block.name}
        </span>
        <span className="flex shrink-0 items-center gap-1.5 text-micro font-semibold text-slate-500">
          <ToolStatusIcon block={block} />
          {toolStatusLabel[block.status]}
        </span>
        <ChevronDown
          size={13}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-slate-500 transition-transform duration-200",
            opened && "rotate-180",
          )}
        />
      </button>
      <Collapse opened={opened}>
        <div className="border-t border-slate-200 bg-white px-3 py-2.5">
          <ToolDetailsView block={block} />
        </div>
      </Collapse>
    </div>
  );
};

const ApprovalView = (props: { block: ApprovalBlock; ctx: BlockContext }) => {
  const { block, ctx } = props;
  const [pending, setPending] = React.useState<ApprovalDecision | undefined>();

  const decide = async (decision: ApprovalDecision) => {
    if (!ctx.runId || !ctx.onDecide) return;
    setPending(decision);
    try {
      await ctx.onDecide(ctx.runId, block.approvalId, decision);
    } finally {
      setPending(undefined);
    }
  };

  const isPending = block.status === "pending";

  return (
    <div
      className={twMerge(
        "my-2 overflow-hidden rounded-xl border",
        isPending
          ? "border-amber-300 bg-amber-50/70"
          : "border-slate-200 bg-slate-50/60",
      )}
    >
      <div className="flex items-start gap-2.5 px-3 py-2.5">
        <ShieldAlert
          size={16}
          className={twMerge(
            "mt-0.5 shrink-0",
            isPending ? "text-amber-600" : "text-slate-500",
          )}
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-body font-semibold text-slate-800">
              {block.title}
            </p>
            <Badge size="xs" variant="light" color={riskColor(block.risk)}>
              {block.risk}
            </Badge>
            {!isPending && (
              <Badge
                size="xs"
                variant="light"
                color={block.status === "approved" ? "green" : "gray"}
              >
                {block.status}
              </Badge>
            )}
          </div>
          {block.description && (
            <p className="mt-1 text-xs leading-5 text-slate-600">
              {block.description}
            </p>
          )}
          {isPending && block.preview?.method && (
            <Section label="Method">
              <code className="font-mono text-xs font-semibold text-slate-800">
                {block.preview.method}
              </code>
            </Section>
          )}
          {isPending && block.preview?.request !== undefined && (
            <Section label="Request">
              <Code className="bg-white">{toJSON(block.preview.request)}</Code>
            </Section>
          )}
          {isPending && block.preview?.command && (
            <Section label="Command">
              <Code className="bg-white">{`$ ${block.preview.command}`}</Code>
            </Section>
          )}
          {block.reason && (
            <p className="mt-1.5 text-xs text-slate-600">
              Reason: {block.reason}
            </p>
          )}
          {isPending && (
            <div className="mt-3 flex flex-wrap gap-2">
              <Button
                size="xs"
                color="dark"
                loading={pending === "approve"}
                disabled={!!pending || !ctx.runId}
                onClick={() => void decide("approve")}
              >
                Approve
              </Button>
              <Button
                size="xs"
                variant="default"
                loading={pending === "reject"}
                disabled={!!pending || !ctx.runId}
                onClick={() => void decide("reject")}
              >
                Reject
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

const formatCell = (value: unknown, column: TableColumn): React.ReactNode => {
  if (value === null || value === undefined || value === "") {
    return <span className="text-slate-400">—</span>;
  }

  switch (column.type) {
    case "number":
      return typeof value === "number"
        ? `${value.toLocaleString(undefined, { maximumFractionDigits: 4 })}${column.unit ? ` ${column.unit}` : ""}`
        : formatValue(value);
    case "boolean":
      return value ? "Yes" : "No";
    case "datetime": {
      const date = new Date(String(value));
      return Number.isNaN(date.getTime())
        ? formatValue(value)
        : date.toLocaleString();
    }
    case "json":
      return (
        <code className="font-mono text-micro text-slate-600">
          {formatValue(value).slice(0, 200)}
        </code>
      );
    default:
      return formatValue(value);
  }
};

const tablePageSize = 50;

const TableView = (props: { block: TableBlock }) => {
  const { block } = props;
  const [limit, setLimit] = React.useState(tablePageSize);
  const rows = block.rows.slice(0, limit);

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-slate-200 bg-white">
      <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
        <TableIcon size={13} className="shrink-0 text-slate-500" />
        <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
          {block.title ?? "Table"}
        </span>
        <span className="text-micro font-semibold text-slate-500">
          {block.totalRows.toLocaleString()} rows
        </span>
        <Tooltip label="Download as CSV" withArrow>
          <Button
            size="compact-xs"
            variant="subtle"
            color="gray"
            leftSection={<Download size={12} />}
            onClick={() =>
              saveBlob(
                new Blob([toCSV(block.columns, block.rows)], {
                  type: "text/csv",
                }),
                `${(block.title ?? "table").replace(/[^\w.-]+/g, "_")}.csv`,
              )
            }
          >
            CSV
          </Button>
        </Tooltip>
      </div>
      <div className="max-h-[420px] overflow-auto">
        <table className="w-full border-collapse text-xs">
          <thead className="sticky top-0 bg-slate-50 text-slate-600">
            <tr>
              {block.columns.map((column) => (
                <th
                  key={column.key}
                  className="whitespace-nowrap border-b border-slate-200 px-3 py-2 text-left font-bold"
                >
                  {column.label ?? column.key}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, idx) => (
              <tr key={idx} className="hover:bg-slate-50/70">
                {block.columns.map((column) => (
                  <td
                    key={column.key}
                    className={twMerge(
                      "max-w-[320px] truncate border-b border-slate-100 px-3 py-1.5 align-top text-slate-700",
                      column.type === "number" && "text-right tabular-nums",
                    )}
                  >
                    {formatCell(row[column.key], column)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {(block.caption || block.rows.length > limit || block.truncated) && (
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-100 px-3 py-1.5 text-micro text-slate-500">
          <span>
            {block.caption}
            {block.truncated &&
              ` Showing ${block.rows.length.toLocaleString()} of ${block.totalRows.toLocaleString()} rows.`}
          </span>
          {block.rows.length > limit && (
            <Button
              size="compact-xs"
              variant="subtle"
              color="gray"
              onClick={() => setLimit((value) => value + tablePageSize * 4)}
            >
              Show more
            </Button>
          )}
        </div>
      )}
    </div>
  );
};

const toNumber = (value: unknown): number | null => {
  if (value === null || value === undefined || value === "") return null;
  const ret = typeof value === "number" ? value : Number(value);
  return Number.isFinite(ret) ? ret : null;
};

export const buildChartOption = (spec: ChartSpec) => {
  const textStyle = {
    color: CHART_INK.muted,
    fontSize: 10,
    fontWeight: 600,
    fontFamily: CHART_FONT,
  };
  const tooltip = {
    confine: true,
    backgroundColor: CHART_TOOLTIP.backgroundColor,
    borderColor: CHART_TOOLTIP.borderColor,
    borderWidth: 1,
    textStyle: {
      color: CHART_INK.onDark,
      fontSize: 12,
      fontFamily: CHART_FONT,
    },
    extraCssText: CHART_TOOLTIP.extraCssText,
  };
  const legend = {
    show: spec.series.length > 1 || spec.chartType === "pie",
    type: "scroll",
    top: 0,
    icon: "roundRect",
    itemWidth: 9,
    itemHeight: 9,
    textStyle,
  };

  if (spec.chartType === "pie") {
    const series = spec.series[0];
    return {
      color: seriesColors(),
      tooltip: { ...tooltip, trigger: "item" },
      legend,
      series: [
        {
          type: "pie",
          radius: ["42%", "70%"],
          top: 24,
          label: { ...textStyle, formatter: "{b}: {d}%" },
          data: spec.rows.map((row) => ({
            name: formatValue(row[spec.x.key]),
            value: toNumber(row[series?.key ?? ""]) ?? 0,
          })),
        },
      ],
    };
  }

  const xType =
    spec.x.type === "time"
      ? "time"
      : spec.x.type === "number"
        ? "value"
        : "category";

  const xValue = (value: unknown) =>
    xType === "time"
      ? new Date(String(value)).getTime()
      : xType === "value"
        ? toNumber(value)
        : formatValue(value);

  return {
    color: seriesColors(),
    aria: { enabled: true, decal: { show: false } },
    grid: {
      top: legend.show ? 34 : 16,
      right: 14,
      bottom: 24,
      left: 8,
      containLabel: true,
    },
    legend,
    tooltip: {
      ...tooltip,
      trigger: spec.chartType === "scatter" ? "item" : "axis",
    },
    xAxis: {
      type: xType,
      name: spec.x.label,
      nameLocation: "middle",
      nameGap: 28,
      nameTextStyle: textStyle,
      data:
        xType === "category"
          ? spec.rows.map((row) => formatValue(row[spec.x.key]))
          : undefined,
      axisLabel: { ...textStyle, hideOverlap: true },
      axisLine: { lineStyle: { color: CHART_INK.grid } },
      axisTick: { show: false },
    },
    yAxis: {
      type: "value",
      name: spec.y?.label ?? spec.y?.unit,
      nameTextStyle: textStyle,
      axisLabel: textStyle,
      splitLine: { lineStyle: { color: CHART_INK.grid, type: "dashed" } },
    },
    series: spec.series.map((series) => ({
      name: series.label ?? series.key,
      type: spec.chartType === "area" ? "line" : spec.chartType,
      stack: spec.stacked ? "total" : undefined,
      smooth: spec.chartType === "line" || spec.chartType === "area",
      showSymbol: spec.chartType === "scatter" || spec.rows.length < 40,
      areaStyle: spec.chartType === "area" ? { opacity: 0.18 } : undefined,
      barMaxWidth: 28,
      data:
        xType === "category"
          ? spec.rows.map((row) => toNumber(row[series.key]))
          : spec.rows.map((row) => [
              xValue(row[spec.x.key]),
              toNumber(row[series.key]),
            ]),
    })),
  };
};

const ChartView = (props: { block: ChartBlock }) => {
  const { chart } = props.block;
  const option = React.useMemo(() => buildChartOption(chart), [chart]);

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-slate-200 bg-white px-3 py-2.5">
      {chart.title && (
        <p className="text-xs font-semibold text-slate-700">{chart.title}</p>
      )}
      {chart.description && (
        <p className="mt-0.5 text-micro text-slate-500">{chart.description}</p>
      )}
      <ReactEChartsCore
        echarts={echarts}
        option={option}
        notMerge
        style={{ height: 280, width: "100%" }}
      />
    </div>
  );
};

const ResourcesView = (props: { block: ResourcesBlock }) => {
  const { block } = props;
  const [opened, setOpened] = React.useState<string | undefined>();

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-slate-200 bg-white">
      <div className="flex items-center gap-2 border-b border-slate-100 px-3 py-2">
        <Box size={13} className="shrink-0 text-slate-500" />
        <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
          {block.title ?? "Resources"}
        </span>
        <span className="text-micro font-semibold text-slate-500">
          {block.resources.length}
        </span>
      </div>
      <ul className="max-h-[420px] divide-y divide-slate-100 overflow-y-auto">
        {block.resources.map((item) => {
          const route = getResourceRoute(item.ref);
          const isOpened = opened === item.uri;
          return (
            <li key={item.uri} className="px-3 py-1.5">
              <div className="flex items-center gap-2">
                <Badge size="xs" variant="light" color="gray">
                  {item.ref.kind}
                </Badge>
                {route ? (
                  <Link
                    to={route}
                    className="min-w-0 flex-1 truncate text-xs font-semibold text-blue-600 hover:text-blue-700"
                  >
                    {item.ref.name}
                  </Link>
                ) : (
                  <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-800">
                    {item.ref.name}
                  </span>
                )}
                <span className="hidden text-micro text-slate-400 sm:inline">
                  {item.ref.apiVersion}
                </span>
                {item.snapshot && (
                  <button
                    type="button"
                    aria-label="Show details"
                    aria-expanded={isOpened}
                    onClick={() => setOpened(isOpened ? undefined : item.uri)}
                    className="flex h-5 w-5 cursor-pointer items-center justify-center rounded text-slate-500 hover:bg-slate-100"
                  >
                    <ChevronDown
                      size={12}
                      className={twMerge(
                        "transition-transform duration-200",
                        isOpened && "rotate-180",
                      )}
                    />
                  </button>
                )}
              </div>
              <Collapse opened={isOpened}>
                <Code className="mt-1.5">{toJSON(item.snapshot)}</Code>
              </Collapse>
            </li>
          );
        })}
      </ul>
    </div>
  );
};

const ArtifactView = (props: { block: ArtifactBlock; ctx: BlockContext }) => {
  const { artifact } = props.block;
  const { client } = props.ctx;
  const [downloading, setDownloading] = React.useState(false);
  const [preview, setPreview] = React.useState<string | undefined>();
  const [error, setError] = React.useState<string | undefined>();
  const isImage = artifact.mimeType.startsWith("image/");

  React.useEffect(() => {
    if (!isImage || artifact.size > 10 * 1024 * 1024) return;
    let url: string | undefined;
    let active = true;
    client
      .downloadArtifact(artifact)
      .then((blob) => {
        if (!active) return;
        url = URL.createObjectURL(blob);
        setPreview(url);
      })
      .catch(() => undefined);
    return () => {
      active = false;
      if (url) URL.revokeObjectURL(url);
    };
  }, [client, artifact, isImage]);

  const download = async () => {
    setDownloading(true);
    setError(undefined);
    try {
      saveBlob(await client.downloadArtifact(artifact), artifact.name);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div className="my-2 overflow-hidden rounded-xl border border-slate-200 bg-white">
      {preview && (
        <img
          src={preview}
          alt={artifact.title ?? artifact.name}
          className="max-h-[360px] w-full border-b border-slate-100 bg-slate-50 object-contain"
        />
      )}
      <div className="flex items-center gap-3 px-3 py-2.5">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-600">
          <FileText size={16} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-body font-semibold text-slate-800">
            {artifact.title ?? artifact.name}
          </p>
          <p className="truncate text-micro text-slate-500">
            {artifact.name} · {formatBytes(artifact.size)} · {artifact.mimeType}
          </p>
          {artifact.description && (
            <p className="mt-0.5 text-xs text-slate-600">
              {artifact.description}
            </p>
          )}
          {error && <p className="mt-0.5 text-xs text-red-600">{error}</p>}
        </div>
        <Button
          size="xs"
          variant="default"
          leftSection={<Download size={13} />}
          loading={downloading}
          onClick={() => void download()}
        >
          Download
        </Button>
      </div>
    </div>
  );
};

const NoticeView = (props: { block: NoticeBlock }) => (
  <div
    className={twMerge(
      "my-2 flex items-start gap-2 rounded-lg border px-3 py-2 text-xs",
      props.block.level === "warning"
        ? "border-amber-200 bg-amber-50 text-amber-800"
        : "border-slate-200 bg-slate-50 text-slate-600",
    )}
  >
    {props.block.level === "warning" ? (
      <TriangleAlert size={14} className="mt-0.5 shrink-0" />
    ) : (
      <Info size={14} className="mt-0.5 shrink-0" />
    )}
    <span className="min-w-0 break-words">{props.block.text}</span>
  </div>
);

const ErrorView = (props: { block: ErrorBlock }) => (
  <div className="my-2 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold text-red-700">
    <CircleAlert size={14} className="mt-0.5 shrink-0" />
    <span className="min-w-0 break-words">{props.block.message}</span>
  </div>
);

export const BlockView = (props: {
  block: Block;
  ctx: BlockContext;
  isLast: boolean;
}) => {
  const { block, ctx } = props;

  switch (block.type) {
    case "markdown":
      return <AgentMarkdown>{block.text}</AgentMarkdown>;
    case "thinking":
      return (
        <ThinkingView block={block} streaming={ctx.streaming && props.isLast} />
      );
    case "tool":
      return <ToolView block={block} />;
    case "approval":
      return <ApprovalView block={block} ctx={ctx} />;
    case "table":
      return <TableView block={block} />;
    case "chart":
      return <ChartView block={block} />;
    case "resources":
      return <ResourcesView block={block} />;
    case "artifact":
      return <ArtifactView block={block} ctx={ctx} />;
    case "notice":
      return <NoticeView block={block} />;
    case "error":
      return <ErrorView block={block} />;
    default:
      return null;
  }
};
