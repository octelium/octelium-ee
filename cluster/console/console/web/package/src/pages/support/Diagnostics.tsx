import { ComponentLog, ComponentLog_Entry_Level } from "@/apis/corev1/corev1";
import { Struct } from "@/apis/google/protobuf/struct";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import {
  ComponentSelector,
  ListComponentLogRequest,
  ListComponentLogTopComponentRequest,
} from "@/apis/visibilityv1/visibilityv1";
import RpcError from "@/components/RpcError";
import { twMerge } from "@/lib/utils";
import { saveBlob } from "@/pages/Agent/files";
import { formatBytes, formatCount } from "@/pages/Agent/format";
import { getClientVisibilityComponentLog } from "@/utils/client";
import {
  Alert,
  Button,
  Checkbox,
  Chip,
  Drawer,
  Progress,
  SegmentedControl,
  Select,
  Switch,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import dayjs from "dayjs";
import {
  ArrowLeft,
  Download,
  EyeOff,
  Paperclip,
  ScanSearch,
  ShieldCheck,
  X,
} from "lucide-react";
import * as React from "react";
import {
  buildDiagnostics,
  countRedactions,
  DEFAULT_DIAGNOSTIC_LEVELS,
  DIAGNOSTIC_LEVELS,
  DiagnosticEntry,
  DiagnosticsBundle,
  getWindow,
  REDACTION_LABELS,
  RedactionRule,
} from "./diagnostics";

export interface DiagnosticsPreset {
  time?: string;
  level?: string;
  component?: { namespace?: string; type?: string; uid?: string };
}

const PAGE_SIZE = 500;

const PREVIEW_SIZE = 100;

const LEVEL_LABEL: Record<string, string> = {
  WARN: "Warn",
  ERROR: "Error",
  PANIC: "Panic",
  FATAL: "Fatal",
};

const LEVEL_STYLE: Record<string, string> = {
  WARN: "border-amber-200 bg-amber-50 text-amber-700",
  ERROR: "border-red-200 bg-red-50 text-red-700",
  PANIC: "border-red-300 bg-red-100 text-red-800",
  FATAL: "border-red-400 bg-red-200 text-red-900",
};

const toEntry = (log: ComponentLog): DiagnosticEntry => {
  const entry = log.entry;
  return {
    ...(entry?.time
      ? { time: Timestamp.toDate(entry.time).toISOString() }
      : {}),
    level: ComponentLog_Entry_Level[entry?.level ?? 0] ?? "LEVEL_UNSET",
    ...(entry?.component
      ? {
          component: {
            namespace: entry.component.namespace,
            type: entry.component.type,
            uid: entry.component.uid,
          },
        }
      : {}),
    ...(entry?.function ? { function: entry.function } : {}),
    ...(entry?.file ? { file: entry.file } : {}),
    ...(entry?.line ? { line: entry.line } : {}),
    message: entry?.message ?? "",
    ...(entry?.fields && Object.keys(entry.fields.fields).length > 0
      ? { fields: Struct.toJson(entry.fields) as Record<string, unknown> }
      : {}),
  };
};

const toLevel = (level: string) =>
  ComponentLog_Entry_Level[level as keyof typeof ComponentLog_Entry_Level];

const Tile = (props: {
  label: string;
  value: React.ReactNode;
  hint?: string;
}) => (
  <div className="flex min-w-0 flex-col gap-0.5 rounded-lg border border-slate-200 bg-white px-3 py-2.5">
    <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {props.label}
    </span>
    <span className="truncate text-body font-semibold tabular-nums text-slate-800">
      {props.value}
    </span>
    {props.hint && (
      <span className="truncate text-micro font-normal text-slate-500">
        {props.hint}
      </span>
    )}
  </div>
);

const Section = (props: { title: string; children: React.ReactNode }) => (
  <section className="flex flex-col gap-2.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3.5">
    <h3 className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {props.title}
    </h3>
    {props.children}
  </section>
);

const Preview = (props: { bundle: DiagnosticsBundle }) => {
  const { bundle } = props;
  const { manifest } = bundle;
  const redactions = countRedactions(manifest.redaction.counts);
  const rules = (
    Object.entries(manifest.redaction.counts) as [RedactionRule, number][]
  ).filter(([, count]) => count > 0);

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Tile
          label="Entries"
          value={formatCount(manifest.included)}
          hint={
            manifest.truncated
              ? `${formatCount(manifest.matched)} matched`
              : "All matching"
          }
        />
        <Tile
          label="Window"
          value={dayjs(manifest.window.from).format("MMM D, HH:mm")}
          hint={`to ${dayjs(manifest.window.to).format("MMM D, HH:mm")}`}
        />
        <Tile
          label="Size"
          value={formatBytes(bundle.data.byteLength)}
          hint={bundle.compressed ? "gzip compressed" : "Uncompressed"}
        />
        <Tile
          label="Redactions"
          value={formatCount(redactions)}
          hint={redactions > 0 ? "Removed before sending" : "Nothing found"}
        />
      </div>

      {rules.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          <EyeOff size={12} strokeWidth={2.4} className="text-slate-400" />
          {rules.map(([rule, count]) => (
            <span
              key={rule}
              className="rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-micro font-semibold text-slate-600"
            >
              {REDACTION_LABELS[rule]} · {count}
            </span>
          ))}
        </div>
      )}

      {manifest.truncated && (
        <p className="text-micro font-normal leading-5 text-slate-500">
          Only the most recent entries are included. Narrow the time window or
          the levels to send other entries.
        </p>
      )}

      {bundle.entries.length > PREVIEW_SIZE && (
        <p className="text-micro font-normal leading-5 text-slate-500">
          The preview shows the latest {PREVIEW_SIZE} of{" "}
          {formatCount(bundle.entries.length)} entries. Save a copy to review
          all of them before you agree to send them.
        </p>
      )}

      {bundle.entries.length === 0 ? (
        <div className="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-slate-200 bg-white px-4 text-center">
          <p className="text-xs font-normal text-slate-500">
            No component logs match this selection. Widen the time window or add
            levels.
          </p>
        </div>
      ) : (
        <div
          className="max-h-[min(420px,45dvh)] overflow-auto rounded-lg border border-slate-200 bg-white"
          aria-label="Diagnostics preview"
          tabIndex={0}
        >
          <ol className="divide-y divide-slate-100">
            {bundle.entries
              .slice(-PREVIEW_SIZE)
              .reverse()
              .map((entry, index) => (
                <li key={index} className="flex flex-col gap-1 px-3 py-2">
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-micro">
                    <span
                      className={twMerge(
                        "rounded border px-1 py-px font-semibold",
                        LEVEL_STYLE[entry.level] ??
                          "border-slate-200 bg-slate-50 text-slate-600",
                      )}
                    >
                      {LEVEL_LABEL[entry.level] ?? entry.level}
                    </span>
                    <span className="font-mono tabular-nums text-slate-500">
                      {entry.time
                        ? dayjs(entry.time).format("HH:mm:ss.SSS")
                        : "—"}
                    </span>
                    {entry.component?.type && (
                      <span className="font-mono font-semibold text-slate-600">
                        {entry.component.type}
                      </span>
                    )}
                  </div>
                  <p className="break-words font-mono text-xs leading-5 text-slate-800">
                    {entry.message}
                  </p>
                  {entry.fields && (
                    <p className="line-clamp-2 break-all font-mono text-micro leading-4 text-slate-500">
                      {JSON.stringify(entry.fields)}
                    </p>
                  )}
                </li>
              ))}
          </ol>
        </div>
      )}
    </div>
  );
};

const Diagnostics = (props: {
  opened: boolean;
  onClose: () => void;
  onAttach: (file: File) => void;
  maxBytes: number;
  clusterDomain?: string;
  preset?: DiagnosticsPreset;
}) => {
  const anchor = props.preset?.time;
  const [range, setRange] = React.useState(anchor ? "around" : "1h");
  const [levels, setLevels] = React.useState<string[]>(
    props.preset?.level &&
      DIAGNOSTIC_LEVELS.includes(props.preset.level as never)
      ? Array.from(new Set([...DEFAULT_DIAGNOSTIC_LEVELS, props.preset.level]))
      : DEFAULT_DIAGNOSTIC_LEVELS,
  );
  const [componentType, setComponentType] = React.useState<string | null>(
    props.preset?.component?.type ?? null,
  );
  const [onlyInstance, setOnlyInstance] = React.useState(
    !!props.preset?.component?.uid,
  );
  const [maxEntries, setMaxEntries] = React.useState("1000");
  const [keepEmails, setKeepEmails] = React.useState(false);
  const [progress, setProgress] = React.useState<number | null>(null);
  const [error, setError] = React.useState<unknown>(null);
  const [bundle, setBundle] = React.useState<DiagnosticsBundle | null>(null);
  const [consent, setConsent] = React.useState(false);
  const abort = React.useRef<AbortController | null>(null);
  const [wasOpened, setWasOpened] = React.useState(props.opened);
  const [period, setPeriod] = React.useState(() => getWindow(range, anchor));

  if (wasOpened !== props.opened) {
    setWasOpened(props.opened);
    if (props.opened) {
      setPeriod(getWindow(range, anchor));
    } else {
      setProgress(null);
      setError(null);
      setBundle(null);
      setConsent(false);
    }
  }

  const presetComponent = props.preset?.component;
  const presetSelected =
    !!componentType && componentType === presetComponent?.type;

  const components = useQuery({
    queryKey: ["support", "diagnostics", "components", period.from, period.to],
    queryFn: async () => {
      const { response } =
        await getClientVisibilityComponentLog().listComponentLogTopComponent(
          ListComponentLogTopComponentRequest.create({
            from: Timestamp.fromDate(new Date(period.from)),
            to: Timestamp.fromDate(new Date(period.to)),
          }),
        );
      return response;
    },
    enabled: props.opened,
    retry: false,
    staleTime: 60_000,
  });

  const typeOptions = Array.from(
    new Set(
      [
        ...(components.data?.items ?? []).map((item) => item.component?.type),
        props.preset?.component?.type,
      ].filter((value): value is string => !!value),
    ),
  ).sort();

  const collecting = progress !== null;

  const collect = async () => {
    const controller = new AbortController();
    abort.current = controller;
    setError(null);
    setBundle(null);
    setConsent(false);
    setProgress(0);

    const max = Number(maxEntries);
    const bounds = getWindow(range, anchor);
    setPeriod(bounds);
    const component = ComponentSelector.create({
      ...(componentType ? { type: componentType } : {}),
      ...(presetSelected && presetComponent?.namespace
        ? { namespace: presetComponent.namespace }
        : {}),
      ...(presetSelected && onlyInstance && presetComponent?.uid
        ? { uid: presetComponent.uid }
        : {}),
    });
    const selection = {
      from: bounds.from,
      to: bounds.to,
      levels,
      component: {
        ...(component.namespace ? { namespace: component.namespace } : {}),
        ...(component.type ? { type: component.type } : {}),
        ...(component.uid ? { uid: component.uid } : {}),
      },
      maxEntries: max,
      keepEmails,
    };

    try {
      const entries: DiagnosticEntry[] = [];
      let matched = 0;
      for (let page = 0; entries.length < max; page++) {
        const { response } =
          await getClientVisibilityComponentLog().listComponentLog(
            ListComponentLogRequest.create({
              common: {
                page,
                itemsPerPage: Math.min(PAGE_SIZE, max - entries.length),
              },
              from: Timestamp.fromDate(new Date(bounds.from)),
              to: Timestamp.fromDate(new Date(bounds.to)),
              levels: levels.map(toLevel),
              component,
            }),
            { abort: controller.signal },
          );
        matched = Number(response.listResponseMeta?.totalCount ?? 0);
        entries.push(...response.items.map(toEntry));
        setProgress(entries.length);
        if (!response.listResponseMeta?.hasMore || response.items.length === 0)
          break;
      }

      if (controller.signal.aborted) return;
      setBundle(
        await buildDiagnostics({
          entries,
          selection,
          matched,
          maxBytes: props.maxBytes,
          clusterDomain: props.clusterDomain,
        }),
      );
    } catch (err) {
      if (!controller.signal.aborted) setError(err);
    } finally {
      if (abort.current === controller) {
        abort.current = null;
        setProgress(null);
      }
    }
  };

  const attach = () => {
    if (!bundle) return;
    props.onAttach(
      new File([bundle.data], bundle.name, {
        type: bundle.compressed ? "application/gzip" : "application/x-ndjson",
      }),
    );
    props.onClose();
  };

  return (
    <Drawer
      opened={props.opened}
      onClose={() => {
        abort.current?.abort();
        props.onClose();
      }}
      position="right"
      size="min(720px, 100vw)"
      padding={0}
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      transitionProps={{
        transition: "slide-left",
        duration: 260,
        exitDuration: 220,
      }}
      title={
        <div className="flex min-w-0 flex-col">
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            Support
          </span>
          <span className="truncate text-sm font-bold text-slate-900">
            Collect diagnostics
          </span>
        </div>
      }
      styles={{
        header: {
          borderBottom: "1px solid var(--color-slate-200)",
          minHeight: "56px",
          paddingInline: "16px",
        },
        body: {
          height: "calc(100dvh - 56px)",
          padding: 0,
          display: "flex",
          flexDirection: "column",
          backgroundColor: "var(--color-slate-50)",
        },
      }}
    >
      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-4">
        <div className="flex items-start gap-2.5 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
          <ShieldCheck
            size={14}
            strokeWidth={2.4}
            className="mt-0.5 shrink-0 text-slate-500"
          />
          <p className="text-xs font-normal leading-5 text-slate-600">
            The component logs are read with your own permissions and processed
            in this browser. Tokens, passwords, cookies, private keys, URL query
            strings and email addresses are detected and removed automatically,
            but the detection cannot recognize every secret: review the entries
            before you agree to send them. Nothing is sent until you attach the
            result and submit.
          </p>
        </div>

        {bundle ? (
          <Section title="Review">
            <Preview bundle={bundle} />
          </Section>
        ) : (
          <>
            <Section title="Time window">
              <SegmentedControl
                aria-label="Time window"
                size="xs"
                fullWidth
                value={range}
                onChange={(value) => {
                  setRange(value);
                  setPeriod(getWindow(value, anchor));
                }}
                disabled={collecting}
                data={[
                  ...(anchor
                    ? [{ value: "around", label: "Around entry" }]
                    : []),
                  { value: "15m", label: "15 min" },
                  { value: "1h", label: "1 hour" },
                  { value: "6h", label: "6 hours" },
                  { value: "24h", label: "24 hours" },
                ]}
              />
              <span className="text-micro font-normal tabular-nums text-slate-500">
                {dayjs(period.from).format("MMM D, HH:mm")} –{" "}
                {dayjs(period.to).format("MMM D, HH:mm")}
              </span>
            </Section>

            <Section title="Entries">
              <Chip.Group multiple value={levels} onChange={setLevels}>
                <div
                  className="flex flex-wrap gap-1.5"
                  role="group"
                  aria-label="Levels"
                >
                  {DIAGNOSTIC_LEVELS.map((level) => (
                    <Chip
                      key={level}
                      value={level}
                      size="xs"
                      variant="outline"
                      disabled={collecting}
                    >
                      {LEVEL_LABEL[level]}
                    </Chip>
                  ))}
                </div>
              </Chip.Group>

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <Select
                  label="Component"
                  placeholder="All components"
                  data={typeOptions}
                  value={componentType}
                  onChange={setComponentType}
                  clearable
                  searchable
                  disabled={collecting}
                  nothingFoundMessage="No component logged in this window"
                  comboboxProps={{ withinPortal: true }}
                />
                <Select
                  label="Maximum entries"
                  data={["250", "1000", "2500", "5000"]}
                  value={maxEntries}
                  onChange={(value) => setMaxEntries(value ?? "1000")}
                  allowDeselect={false}
                  disabled={collecting}
                  comboboxProps={{ withinPortal: true }}
                />
              </div>

              {presetSelected && presetComponent?.uid && (
                <Switch
                  size="sm"
                  checked={onlyInstance}
                  disabled={collecting}
                  onChange={(event) =>
                    setOnlyInstance(event.currentTarget.checked)
                  }
                  label={
                    <span className="text-body font-normal text-slate-600">
                      Only the instance that logged the entry
                    </span>
                  }
                />
              )}

              <Switch
                size="sm"
                checked={keepEmails}
                disabled={collecting}
                onChange={(event) => setKeepEmails(event.currentTarget.checked)}
                label={
                  <span className="text-body font-normal text-slate-600">
                    Keep email addresses
                  </span>
                }
              />
            </Section>

            {collecting && (
              <div className="flex flex-col gap-2 rounded-xl border border-slate-200 bg-white px-3.5 py-3">
                <span className="text-xs font-semibold text-slate-700">
                  Reading component logs… {formatCount(progress ?? 0)} entries
                </span>
                <Progress
                  value={Math.min(
                    100,
                    ((progress ?? 0) / Number(maxEntries)) * 100,
                  )}
                  size="sm"
                  color="ink"
                  aria-label="Collection progress"
                />
              </div>
            )}
          </>
        )}

        {!!error && (
          <Alert color="red" title="Could not collect the component logs">
            <RpcError error={error} />
          </Alert>
        )}

        {bundle && bundle.entries.length > 0 && (
          <Checkbox
            checked={consent}
            onChange={(event) => setConsent(event.currentTarget.checked)}
            label={
              <span className="text-body font-normal text-slate-700">
                I reviewed these diagnostics and agree to send them to Octelium
                Support with this request.
              </span>
            }
          />
        )}
      </div>

      <footer className="flex flex-wrap items-center justify-end gap-2 border-t border-slate-200 bg-white px-4 py-3.5">
        {bundle ? (
          <>
            <Button
              variant="default"
              size="sm"
              leftSection={<ArrowLeft size={13} strokeWidth={2.5} />}
              onClick={() => {
                setBundle(null);
                setConsent(false);
              }}
            >
              Back
            </Button>
            {bundle.entries.length > 0 && (
              <Button
                variant="default"
                size="sm"
                leftSection={<Download size={13} strokeWidth={2.5} />}
                onClick={() =>
                  saveBlob(
                    new Blob([bundle.data], {
                      type: "application/octet-stream",
                    }),
                    bundle.name,
                  )
                }
              >
                Save a copy
              </Button>
            )}
            <Button
              variant="filled"
              color="ink"
              size="sm"
              leftSection={<Paperclip size={13} strokeWidth={2.5} />}
              disabled={!consent || bundle.entries.length === 0}
              onClick={attach}
            >
              Attach diagnostics
            </Button>
          </>
        ) : (
          <>
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.5} />}
              onClick={() => {
                if (collecting) abort.current?.abort();
                else props.onClose();
              }}
            >
              {collecting ? "Stop" : "Cancel"}
            </Button>
            <Button
              variant="filled"
              color="ink"
              size="sm"
              leftSection={<ScanSearch size={13} strokeWidth={2.5} />}
              disabled={levels.length === 0}
              loading={collecting}
              onClick={() => void collect()}
            >
              Collect and review
            </Button>
          </>
        )}
      </footer>
    </Drawer>
  );
};

export default Diagnostics;
