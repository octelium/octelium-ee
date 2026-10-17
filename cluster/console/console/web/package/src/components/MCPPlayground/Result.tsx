import { formatBytes, formatDuration } from "@/pages/Agent/format";
import { CopyButton, Shimmer } from "@/pages/Agent/ui";
import { Loader, SegmentedControl } from "@mantine/core";
import {
  Check,
  CircleAlert,
  Download,
  ExternalLink,
  FileText,
  Link2,
  TriangleAlert,
  X,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";
import { Exchange, isObject, toJSONText } from "./client";
import { CodeBlock } from "./ui";

export type Run = {
  pending: boolean;
  exchange?: Exchange;
  result?: unknown;
  error?: string;
};

export type ResultKind = "tool" | "resource" | "prompt";

const asString = (value: unknown) =>
  typeof value === "string" ? value : undefined;

const isWebURL = (value: string) => /^https?:\/\//i.test(value);

const prettyText = (text: string) => {
  const trimmed = text.trim();
  if (!/^[[{]/.test(trimmed)) return undefined;
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return undefined;
  }
};

const base64Size = (data: string) => Math.floor((data.length * 3) / 4);

const TextContent = (props: { text: string }) => {
  const pretty = prettyText(props.text);
  return pretty ? (
    <CodeBlock>{pretty}</CodeBlock>
  ) : (
    <CodeBlock wrap>{props.text}</CodeBlock>
  );
};

const MediaFrame = (props: { label: string; children: React.ReactNode }) => (
  <figure className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
    <div className="flex justify-center p-2">{props.children}</div>
    <figcaption className="border-t border-slate-200 px-3 py-1.5 font-mono text-micro text-slate-500">
      {props.label}
    </figcaption>
  </figure>
);

const BlobContent = (props: {
  data: string;
  mimeType?: string;
  name: string;
}) => {
  const mimeType = props.mimeType || "application/octet-stream";
  const url = `data:${mimeType};base64,${props.data}`;
  const label = `${mimeType} · ${formatBytes(base64Size(props.data))}`;

  if (mimeType.startsWith("image/")) {
    return (
      <MediaFrame label={label}>
        <img
          src={url}
          alt={props.name}
          className="max-h-80 max-w-full rounded-md object-contain"
        />
      </MediaFrame>
    );
  }

  if (mimeType.startsWith("audio/")) {
    return (
      <MediaFrame label={label}>
        <audio controls src={url} className="w-full" />
      </MediaFrame>
    );
  }

  return (
    <div className="flex items-center gap-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2">
      <FileText size={15} className="shrink-0 text-slate-500" />
      <span className="min-w-0 flex-1 truncate font-mono text-micro text-slate-600">
        {label}
      </span>
      <a
        href={url}
        download={props.name}
        className="inline-flex shrink-0 items-center gap-1 rounded-md px-1.5 py-1 text-xs font-semibold text-slate-600 transition-colors hover:bg-slate-100 hover:text-slate-900"
      >
        <Download size={13} />
        Download
      </a>
    </div>
  );
};

const ResourceLink = (props: { item: Record<string, unknown> }) => {
  const uri = asString(props.item.uri) ?? "";
  const name = asString(props.item.title) ?? asString(props.item.name) ?? uri;
  const description = asString(props.item.description);

  return (
    <div className="flex items-start gap-2.5 rounded-lg border border-slate-200 bg-white px-3 py-2">
      <Link2 size={14} className="mt-0.5 shrink-0 text-slate-500" />
      <div className="min-w-0 flex-1">
        <p className="truncate text-xs font-semibold text-slate-800">{name}</p>
        <p className="truncate font-mono text-micro text-slate-500">{uri}</p>
        {description && (
          <p className="mt-0.5 text-micro leading-4 text-slate-500">
            {description}
          </p>
        )}
      </div>
      {isWebURL(uri) && (
        <a
          href={uri}
          target="_blank"
          rel="noreferrer noopener"
          aria-label={`Open ${uri}`}
          className="shrink-0 rounded-md p-1 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900"
        >
          <ExternalLink size={13} />
        </a>
      )}
    </div>
  );
};

const ResourceContents = (props: { item: unknown }) => {
  if (!isObject(props.item))
    return <CodeBlock>{toJSONText(props.item)}</CodeBlock>;
  const uri = asString(props.item.uri) ?? "";
  const mimeType = asString(props.item.mimeType);
  const text = asString(props.item.text);
  const blob = asString(props.item.blob);

  return (
    <div className="flex flex-col gap-1.5">
      {(uri || mimeType) && (
        <p className="flex min-w-0 items-center gap-2 font-mono text-micro text-slate-500">
          <span className="min-w-0 truncate">{uri}</span>
          {mimeType && (
            <span className="shrink-0 rounded border border-slate-200 bg-slate-50 px-1 text-slate-500">
              {mimeType}
            </span>
          )}
        </p>
      )}
      {text !== undefined ? (
        <TextContent text={text} />
      ) : blob !== undefined ? (
        <BlobContent
          data={blob}
          mimeType={mimeType}
          name={uri.split("/").at(-1) || "resource"}
        />
      ) : (
        <CodeBlock>{toJSONText(props.item)}</CodeBlock>
      )}
    </div>
  );
};

const ContentItem = (props: { item: unknown }) => {
  const { item } = props;
  if (!isObject(item)) return <CodeBlock>{toJSONText(item)}</CodeBlock>;

  switch (item.type) {
    case "text":
      return <TextContent text={asString(item.text) ?? ""} />;
    case "image":
    case "audio":
      return (
        <BlobContent
          data={asString(item.data) ?? ""}
          mimeType={asString(item.mimeType)}
          name={String(item.type)}
        />
      );
    case "resource":
      return <ResourceContents item={item.resource} />;
    case "resource_link":
      return <ResourceLink item={item} />;
    default:
      return <CodeBlock>{toJSONText(item)}</CodeBlock>;
  }
};

const contentList = (value: unknown): unknown[] =>
  Array.isArray(value) ? value : value === undefined ? [] : [value];

const ToolResult = (props: { result: unknown }) => {
  if (!isObject(props.result)) {
    return <CodeBlock>{toJSONText(props.result ?? null)}</CodeBlock>;
  }
  const content = contentList(props.result.content);
  const structured = props.result.structuredContent;

  return (
    <div className="flex flex-col gap-2.5">
      {content.map((item, idx) => (
        <ContentItem key={idx} item={item} />
      ))}
      {structured !== undefined && (
        <div className="flex flex-col gap-1.5">
          <p className="text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
            Structured content
          </p>
          <CodeBlock>{toJSONText(structured)}</CodeBlock>
        </div>
      )}
      {content.length === 0 && structured === undefined && (
        <p className="text-xs text-slate-500">The tool returned no content.</p>
      )}
    </div>
  );
};

const ResourceResult = (props: { result: unknown }) => {
  const contents = isObject(props.result)
    ? contentList(props.result.contents)
    : [];
  return contents.length > 0 ? (
    <div className="flex flex-col gap-3">
      {contents.map((item, idx) => (
        <ResourceContents key={idx} item={item} />
      ))}
    </div>
  ) : (
    <p className="text-xs text-slate-500">The resource has no contents.</p>
  );
};

const PromptResult = (props: { result: unknown }) => {
  if (!isObject(props.result)) {
    return <CodeBlock>{toJSONText(props.result ?? null)}</CodeBlock>;
  }
  const description = asString(props.result.description);
  const messages = contentList(props.result.messages);

  return (
    <div className="flex flex-col gap-3">
      {description && (
        <p className="text-xs leading-5 text-slate-600">{description}</p>
      )}
      {messages.map((message, idx) => {
        const role = isObject(message) ? asString(message.role) : undefined;
        return (
          <div key={idx} className="flex flex-col gap-1.5">
            <span
              className={twMerge(
                "w-fit rounded-md border px-1.5 py-0.5 text-micro font-semibold",
                role === "assistant"
                  ? "border-slate-900 bg-slate-900 text-white"
                  : "border-slate-200 bg-slate-50 text-slate-600",
              )}
            >
              {role === "assistant"
                ? "Assistant"
                : role === "user"
                  ? "User"
                  : (role ?? "Message")}
            </span>
            {contentList(isObject(message) ? message.content : message).map(
              (item, itemIdx) => (
                <ContentItem key={itemIdx} item={item} />
              ),
            )}
          </div>
        );
      })}
      {messages.length === 0 && (
        <p className="text-xs text-slate-500">The prompt has no messages.</p>
      )}
    </div>
  );
};

const resultTitles: Record<ResultKind, string> = {
  tool: "Result",
  resource: "Contents",
  prompt: "Messages",
};

const ResultCard = (props: { run: Run; kind: ResultKind }) => {
  const { run } = props;
  const [view, setView] = React.useState<"content" | "json">("content");
  const toolError =
    props.kind === "tool" &&
    isObject(run.result) &&
    run.result.isError === true;
  const raw = run.exchange?.response;
  const duration = run.exchange?.durationMs;
  const exchangeID = run.exchange?.id;
  const ref = React.useRef<HTMLElement>(null);
  const seenRef = React.useRef(exchangeID);

  React.useEffect(() => {
    if (run.pending || exchangeID === seenRef.current) return;
    seenRef.current = exchangeID;
    ref.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [run.pending, exchangeID]);

  const status = run.pending ? (
    <Loader size={11} color="gray" />
  ) : run.error ? (
    <X size={13} strokeWidth={3} className="text-red-600" />
  ) : toolError ? (
    <TriangleAlert size={13} className="text-amber-600" />
  ) : (
    <Check size={13} strokeWidth={3} className="text-emerald-600" />
  );

  return (
    <section
      ref={ref}
      aria-live="polite"
      className={twMerge(
        "scroll-mb-4 overflow-hidden rounded-xl border bg-white",
        run.error ? "border-red-200" : "border-slate-200",
      )}
    >
      <div className="flex min-h-10 items-center gap-2 border-b border-slate-100 py-1.5 pl-3 pr-1.5">
        <span className="flex h-4 w-4 shrink-0 items-center justify-center">
          {status}
        </span>
        <span className="min-w-0 flex-1 truncate text-xs font-semibold text-slate-700">
          {run.pending ? (
            <Shimmer>Waiting for the server</Shimmer>
          ) : run.error ? (
            "The request failed"
          ) : toolError ? (
            "The tool reported an error"
          ) : (
            resultTitles[props.kind]
          )}
        </span>
        {!run.pending && duration !== undefined && (
          <span className="shrink-0 font-mono text-micro text-slate-400">
            {run.exchange?.httpStatus ? `${run.exchange.httpStatus} · ` : ""}
            {formatDuration(duration)}
          </span>
        )}
        {!run.pending && !run.error && (
          <SegmentedControl
            size="xs"
            aria-label="Result view"
            value={view}
            onChange={(value) => setView(value as "content" | "json")}
            data={[
              { label: "Content", value: "content" },
              { label: "JSON", value: "json" },
            ]}
          />
        )}
        {!run.pending && raw !== undefined && (
          <CopyButton value={() => toJSONText(raw)} label="Copy the response" />
        )}
      </div>

      {!run.pending && (
        <div className="p-3">
          {run.error ? (
            <div className="flex flex-col gap-2.5">
              <p className="flex items-start gap-2 text-xs font-semibold leading-5 text-red-700">
                <CircleAlert size={14} className="mt-0.5 shrink-0" />
                <span className="min-w-0 break-words">{run.error}</span>
              </p>
              {raw !== undefined && <CodeBlock>{toJSONText(raw)}</CodeBlock>}
            </div>
          ) : view === "json" ? (
            <CodeBlock>{toJSONText(raw ?? run.result ?? null)}</CodeBlock>
          ) : props.kind === "tool" ? (
            <ToolResult result={run.result} />
          ) : props.kind === "resource" ? (
            <ResourceResult result={run.result} />
          ) : (
            <PromptResult result={run.result} />
          )}
        </div>
      )}
    </section>
  );
};

export default ResultCard;
