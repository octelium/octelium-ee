import type {
  APIRisk,
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
  useChartColorScheme,
} from "@/utils/charts/palette";
import {
  ActionIcon,
  Badge,
  Button,
  Loader,
  Modal,
  Textarea,
  Tooltip,
} from "@mantine/core";
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
  BookOpen,
  Boxes,
  Brain,
  ChartColumn,
  Check,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  CircleSlash,
  Download,
  File,
  FileArchive,
  FileCode,
  FileImage,
  FileOutput,
  FilePen,
  FileSpreadsheet,
  FileText,
  FolderOpen,
  FolderSearch,
  Globe,
  ImageDown,
  Info,
  ListChecks,
  Maximize2,
  Network,
  Search,
  ShieldAlert,
  ShieldCheck,
  ShieldQuestion,
  ShieldX,
  SquareTerminal,
  Table as TableIcon,
  TextSearch,
  TriangleAlert,
  Wrench,
  X,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { AgentClient } from "./client";
import { CopyButton, Shimmer } from "./ui";
import {
  formatBytes,
  formatDuration,
  formatValue,
  getResourceRoute,
  getResourceURIRoute,
  riskColor,
  saveBlob,
  toCSV,
  toFileName,
  toJSON,
  toolDuration,
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
    reason?: string,
  ) => Promise<void>;
}

const ease = [0.22, 1, 0.36, 1] as const;

const Collapse = (props: { opened: boolean; children: React.ReactNode }) => (
  <AnimatePresence initial={false}>
    {props.opened && (
      <motion.div
        initial={{ height: 0, opacity: 0 }}
        animate={{ height: "auto", opacity: 1 }}
        exit={{ height: 0, opacity: 0 }}
        transition={{ duration: 0.25, ease }}
        className="overflow-hidden"
      >
        {props.children}
      </motion.div>
    )}
  </AnimatePresence>
);

const Code = (props: { children: string; className?: string }) => (
  <div className="group/code relative">
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
    <span className="absolute right-1.5 top-1.5 rounded-md bg-white opacity-0 shadow-sm transition-opacity focus-within:opacity-100 group-hover/code:opacity-100 [@media(hover:none)]:opacity-100">
      <CopyButton value={() => props.children} />
    </span>
  </div>
);

const TerminalView = (props: {
  command: string;
  output?: string;
  exitCode?: number;
  truncated?: boolean;
}) => (
  <div className="group/code relative overflow-hidden rounded-lg border border-terminal-line bg-terminal">
    <div className="flex items-center gap-2 border-b border-terminal-line px-3 py-1.5">
      <SquareTerminal size={12} className="text-terminal-muted" />
      <span className="flex-1 text-micro font-semibold uppercase tracking-[0.08em] text-terminal-muted">
        Shell{props.truncated ? " · truncated" : ""}
      </span>
      {props.exitCode !== undefined && (
        <span
          className={`font-mono text-micro font-semibold ${props.exitCode === 0 ? "text-emerald-500" : "text-red-500"}`}
        >
          exit {props.exitCode}
        </span>
      )}
    </div>
    <pre className="max-h-[320px] overflow-auto px-3 py-2 font-mono text-xs leading-5 text-terminal-text">
      <span className="select-none text-terminal-muted">$ </span>
      {props.command}
      {props.output ? `\n${props.output}` : ""}
    </pre>
  </div>
);

const diffLineClass = (line: string): string => {
  if (line.startsWith("+") && !line.startsWith("+++")) {
    return "bg-emerald-50 text-emerald-800";
  }
  if (line.startsWith("-") && !line.startsWith("---")) {
    return "bg-red-50 text-red-800";
  }
  if (line.startsWith("@@")) {
    return "text-blue-700";
  }
  return "text-slate-600";
};

const DiffView = (props: { diff: string }) => (
  <pre className="max-h-[320px] overflow-auto rounded-lg border border-slate-200 bg-white py-1.5 font-mono text-xs leading-5">
    {props.diff
      .split("\n")
      .slice(0, 2000)
      .map((line, idx) => (
        <div key={idx} className={twMerge("px-3", diffLineClass(line))}>
          {line || " "}
        </div>
      ))}
  </pre>
);

const Section = (props: { label: string; children: React.ReactNode }) => (
  <div className="mt-3 first:mt-0">
    <p className="mb-1 text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
      {props.label}
    </p>
    {props.children}
  </div>
);

const riskLabel: Record<APIRisk, string> = {
  read: "Read",
  write: "Write",
  destructive: "Destructive",
  sensitive: "Sensitive",
};

const RiskBadge = (props: { risk: APIRisk }) => (
  <Badge size="xs" variant="light" color={riskColor(props.risk)}>
    {riskLabel[props.risk]}
  </Badge>
);

const CardHeader = (props: {
  icon: LucideIcon;
  title: React.ReactNode;
  meta?: React.ReactNode;
  children?: React.ReactNode;
}) => (
  <div className="flex min-h-10 items-center gap-2 border-b border-slate-100 py-1.5 pl-3 pr-1.5">
    <props.icon size={13} className="shrink-0 text-slate-500" />
    <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
      {props.title}
    </span>
    {props.meta && (
      <span className="shrink-0 text-micro font-semibold text-slate-500">
        {props.meta}
      </span>
    )}
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

export const AgentMarkdown = React.memo((props: { children: string }) => (
  <div className="text-body">
    <Markdown urlTransform={urlTransform} components={{ a: AgentLink }}>
      {props.children}
    </Markdown>
  </div>
));

const toolIcons: Record<string, LucideIcon> = {
  octelium_api_call: Network,
  octelium_api_search: Search,
  octelium_api_describe: BookOpen,
  bash: SquareTerminal,
  read: FileText,
  write: FilePen,
  edit: FilePen,
  grep: TextSearch,
  find: FolderSearch,
  ls: FolderOpen,
  web_fetch: Globe,
  web_search: Search,
  present_table: TableIcon,
  present_chart: ChartColumn,
  present_resources: Boxes,
  publish_artifact: FileOutput,
};

const ToolStatusIcon = (props: { block: ToolBlock }) => {
  switch (props.block.status) {
    case "pending":
    case "running":
      return <Loader size={10} color="gray" />;
    case "awaiting_approval":
      return <ShieldQuestion size={11} className="text-amber-600" />;
    case "completed":
      return <Check size={11} strokeWidth={3} className="text-emerald-600" />;
    case "failed":
      return <X size={11} strokeWidth={3} className="text-red-600" />;
    case "rejected":
      return <Ban size={11} strokeWidth={2.5} className="text-red-600" />;
    case "cancelled":
      return (
        <CircleSlash size={11} strokeWidth={2.5} className="text-slate-500" />
      );
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

const isToolActive = (block: ToolBlock) =>
  block.status === "pending" || block.status === "running";

const ResourceChip = (props: {
  resource: { apiVersion: string; kind: string; name?: string };
}) => {
  const route = getResourceRoute(props.resource);
  const className =
    "inline-flex max-w-full items-center gap-1 rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-micro text-slate-800";
  const label = (
    <>
      <span className="text-slate-500">{props.resource.kind}</span>
      <span className="truncate font-semibold">{props.resource.name}</span>
    </>
  );

  return route ? (
    <Link
      to={route}
      className={twMerge(
        className,
        "transition-colors duration-150 hover:border-slate-300 hover:bg-slate-50",
      )}
    >
      {label}
    </Link>
  ) : (
    <span className={className}>{label}</span>
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
            <code className="break-all font-mono text-xs font-semibold text-slate-800">
              {details.method}
            </code>
            <RiskBadge risk={details.risk} />
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
            <code className="break-all font-mono text-xs text-slate-700">
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

  if (details?.kind === "command") {
    return (
      <TerminalView
        command={details.command}
        output={block.output}
        exitCode={details.exitCode}
        truncated={block.outputTruncated}
      />
    );
  }

  return (
    <>
      {details?.kind === "file" ? (
        <Section label={details.operation}>
          <code className="break-all font-mono text-xs text-slate-700">
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
          <DiffView diff={details.diff} />
        </Section>
      )}
      {block.output && (
        <Section
          label={block.outputTruncated ? "Output (truncated)" : "Output"}
        >
          <Code>{block.output}</Code>
        </Section>
      )}
    </>
  );
};

type StepBlock = ThinkingBlock | ToolBlock;

const StepIcon = (props: { block: StepBlock; live: boolean }) => {
  if (props.block.type === "thinking") {
    return props.live ? (
      <Loader size={10} color="gray" />
    ) : (
      <Brain size={11} strokeWidth={2.4} className="text-violet-600" />
    );
  }
  return <ToolStatusIcon block={props.block} />;
};

const thinkingLabel = (live: boolean, duration?: number) => {
  if (live) return "Thinking";
  const formatted = duration !== undefined ? formatDuration(duration) : "";
  return formatted && duration! >= 1000
    ? `Thought for ${formatted}`
    : "Thought briefly";
};

const StepRow = (props: {
  block: StepBlock;
  live: boolean;
  endAt?: string;
  isLast: boolean;
}) => {
  const { block, live } = props;
  const [opened, setOpened] = React.useState(false);

  const duration =
    block.type === "tool"
      ? toolDuration(block)
      : props.endAt
        ? new Date(props.endAt).getTime() - new Date(block.createdAt).getTime()
        : undefined;
  const KindIcon =
    block.type === "thinking" ? undefined : (toolIcons[block.name] ?? Wrench);
  const title =
    block.type === "thinking"
      ? thinkingLabel(live, duration)
      : block.title || block.name;
  const active = block.type === "thinking" ? live : isToolActive(block);
  const hasDetails =
    block.type === "thinking"
      ? block.redacted || block.text.trim() !== ""
      : true;
  const risk =
    block.type === "tool" &&
    block.details?.kind === "octelium_api" &&
    block.details.risk !== "read"
      ? block.details.risk
      : undefined;
  const failed =
    block.type === "tool" &&
    (block.status === "failed" || block.status === "rejected");

  return (
    <li className="relative">
      {!props.isLast && (
        <span className="absolute bottom-0 left-[9.5px] top-7 w-px bg-slate-200" />
      )}
      <div className="flex items-start gap-2">
        <span className="relative z-[1] mt-[7px] flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white ring-1 ring-slate-200">
          <StepIcon block={block} live={live} />
        </span>
        <div className="min-w-0 flex-1 pb-1">
          <button
            type="button"
            disabled={!hasDetails}
            aria-expanded={opened}
            onClick={() => setOpened((value) => !value)}
            className="group/step flex min-h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-2 py-1 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:bg-slate-100 disabled:cursor-default disabled:hover:bg-transparent"
          >
            {KindIcon && (
              <KindIcon size={13} className="shrink-0 text-slate-400" />
            )}
            <span
              className={twMerge(
                "min-w-0 flex-1 truncate text-xs font-semibold",
                failed ? "text-red-700" : "text-slate-700",
              )}
            >
              {active ? <Shimmer>{title}</Shimmer> : title}
            </span>
            {risk && <RiskBadge risk={risk} />}
            {block.type === "tool" && block.status !== "completed" && (
              <span className="hidden shrink-0 text-micro font-semibold text-slate-500 sm:inline">
                {toolStatusLabel[block.status]}
              </span>
            )}
            {block.type === "tool" && duration !== undefined && (
              <span className="shrink-0 font-mono text-micro text-slate-400">
                {formatDuration(duration)}
              </span>
            )}
            {hasDetails && (
              <ChevronRight
                size={13}
                strokeWidth={2.4}
                className={twMerge(
                  "shrink-0 text-slate-400 transition-transform duration-200",
                  opened && "rotate-90",
                )}
              />
            )}
          </button>
          <Collapse opened={opened}>
            <div className="mb-1.5 ml-2 mt-1 rounded-lg border border-slate-200 bg-white p-3">
              {block.type === "thinking" ? (
                <p className="max-h-[320px] overflow-y-auto whitespace-pre-wrap text-xs leading-5 text-slate-600">
                  {block.redacted ? "The reasoning is redacted." : block.text}
                </p>
              ) : (
                <ToolDetailsView block={block} />
              )}
            </div>
          </Collapse>
        </div>
      </div>
    </li>
  );
};

const stepsSummary = (blocks: StepBlock[]): string => {
  const tools = blocks.filter((b) => b.type === "tool").length;
  if (tools === 0) return "Reasoning";
  const thought = blocks.some((b) => b.type === "thinking");
  return `${thought ? "Thought and used" : "Used"} ${tools} tool${tools === 1 ? "" : "s"}`;
};

const liveStepLabel = (blocks: StepBlock[]): string => {
  const last = blocks.at(-1);
  if (!last || last.type === "thinking") return "Thinking";
  if (last.status === "awaiting_approval") return "Waiting for your approval";
  return last.title || last.name;
};

export const StepsView = React.memo(
  (props: {
    blocks: StepBlock[];
    live: boolean;
    ends: Record<string, string | undefined>;
  }) => {
    const { blocks, live } = props;
    const [manual, setManual] = React.useState<boolean>();
    const opened = manual ?? live;

    const tools = blocks.filter((b): b is ToolBlock => b.type === "tool");
    const failed = tools.filter(
      (b) => b.status === "failed" || b.status === "rejected",
    ).length;
    const waiting = tools.some((b) => b.status === "awaiting_approval");
    const active = live || tools.some(isToolActive);
    const first = blocks[0];
    const last = blocks.at(-1)!;
    const end =
      last.type === "tool"
        ? (last.completedAt ?? props.ends[last.id])
        : props.ends[last.id];
    const duration =
      !active && end
        ? new Date(end).getTime() - new Date(first.createdAt).getTime()
        : undefined;

    if (blocks.length === 1) {
      return (
        <ol className="my-1.5">
          <StepRow
            block={first}
            live={live}
            endAt={props.ends[first.id]}
            isLast
          />
        </ol>
      );
    }

    return (
      <div className="my-1.5">
        <button
          type="button"
          aria-expanded={opened}
          onClick={() => setManual(!opened)}
          className="flex min-h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-1 py-1 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:bg-slate-100"
        >
          <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-white ring-1 ring-slate-200">
            {active ? (
              <Loader size={10} color="gray" />
            ) : waiting ? (
              <ShieldQuestion size={11} className="text-amber-600" />
            ) : failed > 0 ? (
              <CircleAlert size={11} className="text-red-600" />
            ) : (
              <ListChecks
                size={11}
                strokeWidth={2.4}
                className="text-slate-500"
              />
            )}
          </span>
          <span className="min-w-0 flex-1 truncate pl-1 text-xs font-semibold text-slate-600">
            {active ? (
              <Shimmer>{liveStepLabel(blocks)}</Shimmer>
            ) : (
              stepsSummary(blocks)
            )}
          </span>
          {failed > 0 && (
            <Badge size="xs" variant="light" color="red">
              {failed} failed
            </Badge>
          )}
          {duration !== undefined && (
            <span className="shrink-0 font-mono text-micro text-slate-400">
              {formatDuration(duration)}
            </span>
          )}
          <ChevronDown
            size={13}
            strokeWidth={2.4}
            className={twMerge(
              "mr-1 shrink-0 text-slate-400 transition-transform duration-200",
              opened && "rotate-180",
            )}
          />
        </button>
        <Collapse opened={opened}>
          <ol className="pb-1 pt-1">
            {blocks.map((block, idx) => (
              <StepRow
                key={block.id}
                block={block}
                live={live && idx === blocks.length - 1}
                endAt={props.ends[block.id]}
                isLast={idx === blocks.length - 1}
              />
            ))}
          </ol>
        </Collapse>
      </div>
    );
  },
);

const ApprovalView = (props: { block: ApprovalBlock; ctx: BlockContext }) => {
  const { block, ctx } = props;
  const [pending, setPending] = React.useState<ApprovalDecision | undefined>();
  const [rejecting, setRejecting] = React.useState(false);
  const [reason, setReason] = React.useState("");

  const decide = async (decision: ApprovalDecision) => {
    if (!ctx.runId || !ctx.onDecide) return;
    setPending(decision);
    try {
      await ctx.onDecide(
        ctx.runId,
        block.approvalId,
        decision,
        decision === "reject" && reason.trim() !== ""
          ? reason.trim()
          : undefined,
      );
    } finally {
      setPending(undefined);
    }
  };

  if (block.status !== "pending") {
    const approved = block.status === "approved";
    const Icon = approved
      ? ShieldCheck
      : block.status === "rejected"
        ? ShieldX
        : CircleSlash;
    return (
      <div className="my-2 flex items-start gap-2.5 rounded-xl border border-slate-200 bg-slate-50/60 px-3 py-2">
        <Icon
          size={14}
          className={twMerge(
            "mt-0.5 shrink-0",
            approved
              ? "text-emerald-600"
              : block.status === "rejected"
                ? "text-red-600"
                : "text-slate-500",
          )}
        />
        <div className="min-w-0 flex-1 text-xs leading-5">
          <span className="font-semibold text-slate-700">
            {approved
              ? "Approved"
              : block.status === "rejected"
                ? "Rejected"
                : "Cancelled"}
          </span>
          <span className="text-slate-500"> · {block.title}</span>
          {block.reason && (
            <p className="text-slate-500">Reason: {block.reason}</p>
          )}
        </div>
        <RiskBadge risk={block.risk} />
      </div>
    );
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 6, scale: 0.99 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.3, ease }}
      data-approval-pending
      className="my-3 overflow-hidden rounded-xl border border-amber-300 bg-white shadow-raised ring-4 ring-amber-100/60"
    >
      <div className="flex items-start gap-3 border-b border-amber-200/70 bg-amber-50/70 px-4 py-3">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-500 text-white shadow-sm">
          <ShieldAlert size={15} strokeWidth={2.25} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-micro font-semibold uppercase tracking-[0.08em] text-amber-700">
            Approval required
          </p>
          <p className="mt-0.5 break-words text-body font-semibold text-slate-900">
            {block.title}
          </p>
        </div>
        <RiskBadge risk={block.risk} />
      </div>
      <div className="px-4 py-3">
        {block.description && (
          <p className="mb-3 text-xs leading-5 text-slate-600">
            {block.description}
          </p>
        )}
        {block.preview?.method && (
          <Section label="Method">
            <code className="break-all font-mono text-xs font-semibold text-slate-800">
              {block.preview.method}
            </code>
          </Section>
        )}
        {block.preview?.request !== undefined && (
          <Section label="Request">
            <Code>{toJSON(block.preview.request)}</Code>
          </Section>
        )}
        {block.preview?.command && (
          <Section label="Command">
            <TerminalView command={block.preview.command} />
          </Section>
        )}
        <Collapse opened={rejecting}>
          <Textarea
            className="mt-3"
            size="xs"
            label="Reason"
            description="Optional. The agent sees it and can adjust its plan."
            placeholder="e.g. Use the staging Namespace instead"
            autosize
            minRows={1}
            maxRows={4}
            maxLength={2000}
            value={reason}
            onChange={(event) => setReason(event.currentTarget.value)}
            styles={{ input: { resize: "none" } }}
          />
        </Collapse>
      </div>
      <div className="flex flex-wrap items-center justify-end gap-2 border-t border-slate-100 bg-slate-50/60 px-4 py-2.5">
        <span className="mr-auto text-micro text-slate-500">
          Authorization is still enforced by your Policies
        </span>
        {rejecting ? (
          <>
            <Button
              size="xs"
              variant="subtle"
              color="gray"
              disabled={!!pending}
              onClick={() => setRejecting(false)}
            >
              Cancel
            </Button>
            <Button
              size="xs"
              color="red"
              loading={pending === "reject"}
              disabled={!!pending || !ctx.runId}
              onClick={() => void decide("reject")}
            >
              Reject
            </Button>
          </>
        ) : (
          <>
            <Button
              size="xs"
              variant="default"
              leftSection={<X size={13} strokeWidth={2.5} />}
              disabled={!!pending || !ctx.runId}
              onClick={() => setRejecting(true)}
            >
              Reject
            </Button>
            <Button
              size="xs"
              color="dark"
              leftSection={<Check size={13} strokeWidth={2.5} />}
              loading={pending === "approve"}
              disabled={!!pending || !ctx.runId}
              onClick={() => void decide("approve")}
            >
              Approve
            </Button>
          </>
        )}
      </div>
    </motion.div>
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

const DataTable = (props: {
  block: TableBlock;
  limit: number;
  className?: string;
}) => {
  const { block } = props;
  const rows = block.rows.slice(0, props.limit);

  return (
    <div className={twMerge("overflow-auto", props.className)}>
      <table className="w-full border-collapse text-xs">
        <thead className="sticky top-0 z-[1] bg-slate-50 text-slate-600">
          <tr>
            {block.columns.map((column) => (
              <th
                key={column.key}
                className={twMerge(
                  "whitespace-nowrap border-b border-slate-200 px-3 py-2 text-left font-bold",
                  column.type === "number" && "text-right",
                )}
              >
                {column.label ?? column.key}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, idx) => (
            <tr
              key={idx}
              className="transition-colors duration-100 hover:bg-slate-50/80"
            >
              {block.columns.map((column) => (
                <td
                  key={column.key}
                  title={formatValue(row[column.key])}
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
  );
};

const TableView = (props: { block: TableBlock }) => {
  const { block } = props;
  const [limit, setLimit] = React.useState(tablePageSize);
  const [expanded, setExpanded] = React.useState(false);

  const download = () =>
    saveBlob(
      new Blob([toCSV(block.columns, block.rows)], { type: "text/csv" }),
      toFileName(block.title ?? "table", "csv"),
    );

  return (
    <div className="my-3 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      <CardHeader
        icon={TableIcon}
        title={block.title ?? "Table"}
        meta={`${block.totalRows.toLocaleString()} rows`}
      >
        <Tooltip label="Download as CSV" withArrow>
          <ActionIcon
            size="sm"
            variant="subtle"
            color="gray"
            aria-label="Download as CSV"
            onClick={download}
          >
            <Download size={13} />
          </ActionIcon>
        </Tooltip>
        <Tooltip label="Expand" withArrow>
          <ActionIcon
            size="sm"
            variant="subtle"
            color="gray"
            aria-label="Expand the table"
            onClick={() => setExpanded(true)}
          >
            <Maximize2 size={13} />
          </ActionIcon>
        </Tooltip>
      </CardHeader>
      <DataTable block={block} limit={limit} className="max-h-[420px]" />
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
      <Modal
        opened={expanded}
        onClose={() => setExpanded(false)}
        size="calc(100vw - 48px)"
        centered
        padding={0}
        withCloseButton={false}
        overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
        transitionProps={{ transition: "pop", duration: 200 }}
        styles={{
          content: {
            border: "1px solid var(--color-slate-200)",
            borderRadius: "14px",
            boxShadow: "var(--shadow-modal)",
            overflow: "hidden",
          },
        }}
      >
        <CardHeader
          icon={TableIcon}
          title={block.title ?? "Table"}
          meta={`${block.totalRows.toLocaleString()} rows`}
        >
          <Button
            size="compact-xs"
            variant="subtle"
            color="gray"
            leftSection={<Download size={12} />}
            onClick={download}
          >
            CSV
          </Button>
          <ActionIcon
            size="sm"
            variant="subtle"
            color="gray"
            aria-label="Close"
            onClick={() => setExpanded(false)}
          >
            <X size={14} />
          </ActionIcon>
        </CardHeader>
        <DataTable
          block={block}
          limit={block.rows.length}
          className="max-h-[calc(100dvh-140px)]"
        />
      </Modal>
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
  const scheme = useChartColorScheme();
  const ref = React.useRef<ReactEChartsCore>(null);
  const option = React.useMemo(() => buildChartOption(chart), [chart, scheme]);

  const download = async () => {
    const url = ref.current?.getEchartsInstance().getDataURL({
      type: "png",
      pixelRatio: 2,
      backgroundColor: CHART_INK.surface,
    });
    if (!url) return;
    saveBlob(
      await (await fetch(url)).blob(),
      toFileName(chart.title ?? "chart", "png"),
    );
  };

  return (
    <div className="my-3 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      <CardHeader icon={ChartColumn} title={chart.title ?? "Chart"}>
        <Tooltip label="Download as PNG" withArrow>
          <ActionIcon
            size="sm"
            variant="subtle"
            color="gray"
            aria-label="Download as PNG"
            onClick={() => void download()}
          >
            <ImageDown size={13} />
          </ActionIcon>
        </Tooltip>
      </CardHeader>
      <div className="px-3 pb-2 pt-2.5">
        {chart.description && (
          <p className="mb-1 text-micro text-slate-500">{chart.description}</p>
        )}
        <ReactEChartsCore
          ref={ref}
          echarts={echarts}
          option={option}
          notMerge
          style={{ height: 280, width: "100%" }}
        />
      </div>
    </div>
  );
};

const ResourcesView = (props: { block: ResourcesBlock }) => {
  const { block } = props;
  const [opened, setOpened] = React.useState<string | undefined>();

  return (
    <div className="my-3 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      <CardHeader
        icon={Boxes}
        title={block.title ?? "Resources"}
        meta={block.resources.length.toLocaleString()}
      />
      <ul className="max-h-[420px] divide-y divide-slate-100 overflow-y-auto">
        {block.resources.map((item) => {
          const route = getResourceRoute(item.ref);
          const isOpened = opened === item.uri;
          return (
            <li key={item.uri} className="px-3 py-1.5">
              <div className="flex min-h-7 items-center gap-2">
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
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color="gray"
                    aria-label="Show details"
                    aria-expanded={isOpened}
                    onClick={() => setOpened(isOpened ? undefined : item.uri)}
                  >
                    <ChevronDown
                      size={13}
                      className={twMerge(
                        "transition-transform duration-200",
                        isOpened && "rotate-180",
                      )}
                    />
                  </ActionIcon>
                )}
              </div>
              <Collapse opened={isOpened}>
                <div className="pb-1 pt-1.5">
                  <Code>{toJSON(item.snapshot)}</Code>
                </div>
              </Collapse>
            </li>
          );
        })}
      </ul>
    </div>
  );
};

export const fileIcon = (mimeType: string, name = ""): LucideIcon => {
  if (mimeType.startsWith("image/")) return FileImage;
  if (mimeType === "text/csv" || /\.(csv|tsv|xlsx?)$/i.test(name)) {
    return FileSpreadsheet;
  }
  if (/(zip|tar|gzip|compressed)/.test(mimeType)) return FileArchive;
  if (
    /(json|yaml|xml|javascript|typescript)/.test(mimeType) ||
    /\.(json|ya?ml|ts|js|go|py|sh|toml)$/i.test(name)
  ) {
    return FileCode;
  }
  if (mimeType.startsWith("text/") || mimeType === "application/pdf") {
    return FileText;
  }
  return File;
};

const ArtifactView = (props: { block: ArtifactBlock; ctx: BlockContext }) => {
  const { artifact } = props.block;
  const { client } = props.ctx;
  const [downloading, setDownloading] = React.useState(false);
  const [preview, setPreview] = React.useState<string | undefined>();
  const [zoomed, setZoomed] = React.useState(false);
  const [error, setError] = React.useState<string | undefined>();
  const isImage = artifact.mimeType.startsWith("image/");
  const Icon = fileIcon(artifact.mimeType, artifact.name);

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
    <div className="my-3 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card">
      {preview && (
        <button
          type="button"
          aria-label="Enlarge the image"
          onClick={() => setZoomed(true)}
          className="block w-full cursor-zoom-in border-b border-slate-100 bg-slate-50"
        >
          <img
            src={preview}
            alt={artifact.title ?? artifact.name}
            className="max-h-[360px] w-full object-contain"
          />
        </button>
      )}
      <div className="flex items-center gap-3 px-3 py-2.5">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-600">
          <Icon size={16} />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-body font-semibold text-slate-800">
            {artifact.title ?? artifact.name}
          </p>
          <p className="truncate text-micro text-slate-500">
            {artifact.name} · {formatBytes(artifact.size)}
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
          <span className="hidden sm:inline">Download</span>
          <span className="sm:hidden">Save</span>
        </Button>
      </div>
      {preview && (
        <Modal
          opened={zoomed}
          onClose={() => setZoomed(false)}
          size="auto"
          centered
          padding={0}
          withCloseButton={false}
          overlayProps={{ backgroundOpacity: 0.55, blur: 2 }}
          transitionProps={{ transition: "pop", duration: 200 }}
          styles={{
            content: { background: "transparent", boxShadow: "none" },
          }}
        >
          <img
            src={preview}
            alt={artifact.title ?? artifact.name}
            onClick={() => setZoomed(false)}
            className="max-h-[88dvh] max-w-[92vw] cursor-zoom-out rounded-xl object-contain shadow-modal"
          />
        </Modal>
      )}
    </div>
  );
};

const NoticeView = (props: { block: NoticeBlock }) => (
  <div
    className={twMerge(
      "my-2 flex items-start gap-2 rounded-lg border px-3 py-2 text-xs leading-5",
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
  <div className="my-2 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold leading-5 text-red-700">
    <CircleAlert size={14} className="mt-0.5 shrink-0" />
    <span className="min-w-0 break-words">{props.block.message}</span>
  </div>
);

export const BlockView = React.memo(
  (props: { block: Block; ctx: BlockContext }) => {
    const { block, ctx } = props;

    switch (block.type) {
      case "markdown":
        return <AgentMarkdown>{block.text}</AgentMarkdown>;
      case "thinking":
      case "tool":
        return <StepsView blocks={[block]} live={false} ends={{}} />;
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
  },
);
