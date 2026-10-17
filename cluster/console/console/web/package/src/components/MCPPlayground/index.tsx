import { Service } from "@/apis/corev1/corev1";
import { formatDuration } from "@/pages/Agent/format";
import { CopyButton, StatusDot } from "@/pages/Agent/ui";
import { getDomain } from "@/utils";
import { getServicePublicURL } from "@/utils/octelium";
import {
  ActionIcon,
  Button,
  Popover,
  SegmentedControl,
  Select,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  Activity,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  FileText,
  History,
  MessageSquareText,
  MousePointerClick,
  Play,
  Plug,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  Trash2,
  Unplug,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";
import Arguments, { ArgumentsMode } from "./Arguments";
import {
  ClientConfig,
  DEFAULT_CLIENT_NAME,
  DEFAULT_CLIENT_VERSION,
  DEFAULT_PROTOCOL_VERSION,
  Exchange,
  isObject,
  JSONObject,
  parseCapabilities,
  PROTOCOL_VERSIONS,
  Reply,
  sendMCP,
  toJSONText,
  usesServerDiscover,
} from "./client";
import ResultCard, { ResultKind, Run } from "./Result";
import {
  buildArguments,
  FieldValues,
  parseArguments,
  promptFields,
  schemaFields,
} from "./schema";
import { monospace } from "./styles";
import { BackButton, CodeBlock, Collapse, Description, EmptyState } from "./ui";

type Kind = "tools" | "resources" | "prompts";
type Tab = Kind | "history";

type Listing = {
  items: JSONObject[];
  cursor?: string;
  loaded: boolean;
  loading: boolean;
  error?: string;
};

type Server = {
  name?: string;
  version?: string;
  protocolVersion?: string;
  capabilities?: JSONObject;
  instructions?: string;
};

type State = {
  tab: Tab;
  protocolVersion: string;
  clientName: string;
  clientVersion: string;
  capabilities: string;
  sessionID?: string;
  server?: Server;
  connecting: boolean;
  connectError?: string;
  lists: Record<Kind, Listing>;
  filters: Record<Kind, string>;
  selected: Partial<Record<Kind, string>>;
  custom: Record<Kind, string>;
  values: Record<string, FieldValues>;
  json: Record<string, string>;
  modes: Record<string, ArgumentsMode>;
  runs: Record<string, Run>;
  formErrors: Record<string, string | undefined>;
  history: Exchange[];
  ping?: { ms?: number; error?: string };
};

const CUSTOM = "\u0000custom";
const HISTORY_LIMIT = 100;

const kinds: Kind[] = ["tools", "resources", "prompts"];

const kindInfo: Record<
  Kind,
  {
    icon: LucideIcon;
    label: string;
    singular: string;
    method: string;
    action: string;
    result: ResultKind;
    custom: string;
    customLabel: string;
    customPlaceholder: string;
  }
> = {
  tools: {
    icon: Wrench,
    label: "Tools",
    singular: "tool",
    method: "tools/call",
    action: "Run tool",
    result: "tool",
    custom: "Call a tool by name",
    customLabel: "Tool name",
    customPlaceholder: "search_documents",
  },
  resources: {
    icon: FileText,
    label: "Resources",
    singular: "resource",
    method: "resources/read",
    action: "Read resource",
    result: "resource",
    custom: "Read a resource by URI",
    customLabel: "Resource URI",
    customPlaceholder: "file:///reports/today.md",
  },
  prompts: {
    icon: MessageSquareText,
    label: "Prompts",
    singular: "prompt",
    method: "prompts/get",
    action: "Get prompt",
    result: "prompt",
    custom: "Get a prompt by name",
    customLabel: "Prompt name",
    customPlaceholder: "summarize_document",
  },
};

const sessions = new Map<string, State>();

const emptyListing = (): Listing => ({
  items: [],
  loaded: false,
  loading: false,
});

const getMCPConfig = (service: Service) => {
  const type = service.spec?.config?.type;
  return type?.oneofKind === "mcp" ? type.mcp : undefined;
};

const initialState = (service: Service): State => ({
  tab: "tools",
  protocolVersion:
    getMCPConfig(service)?.protocol?.versions[0] || DEFAULT_PROTOCOL_VERSION,
  clientName: DEFAULT_CLIENT_NAME,
  clientVersion: DEFAULT_CLIENT_VERSION,
  capabilities: "{}",
  connecting: false,
  lists: {
    tools: emptyListing(),
    resources: emptyListing(),
    prompts: emptyListing(),
  },
  filters: { tools: "", resources: "", prompts: "" },
  selected: {},
  custom: { tools: "", resources: "", prompts: "" },
  values: {},
  json: {},
  modes: {},
  runs: {},
  formErrors: {},
  history: [],
});

const restore = (state: State): State => ({
  ...state,
  connecting: false,
  lists: Object.fromEntries(
    kinds.map((kind) => [kind, { ...state.lists[kind], loading: false }]),
  ) as Record<Kind, Listing>,
  runs: Object.fromEntries(
    Object.entries(state.runs).filter(([, run]) => !run.pending),
  ),
});

const itemID = (kind: Kind, item: JSONObject) =>
  String((kind === "resources" ? item.uri : item.name) ?? "");

const itemTitle = (item: JSONObject) =>
  typeof item.title === "string" ? item.title : undefined;

const itemDescription = (item: JSONObject) =>
  typeof item.description === "string" ? item.description : undefined;

const runKey = (kind: Kind, id: string) => `${kind}:${id}`;

const parseServer = (result: unknown, fallbackVersion: string): Server => {
  const body = isObject(result) ? result : {};
  const info = isObject(body.serverInfo) ? body.serverInfo : {};
  return {
    name:
      (typeof info.title === "string" && info.title) ||
      (typeof info.name === "string" ? info.name : undefined),
    version: typeof info.version === "string" ? info.version : undefined,
    protocolVersion:
      typeof body.protocolVersion === "string"
        ? body.protocolVersion
        : fallbackVersion,
    capabilities: isObject(body.capabilities) ? body.capabilities : undefined,
    instructions:
      typeof body.instructions === "string" ? body.instructions : undefined,
  };
};

const hintTones = {
  green: "border-emerald-200 bg-emerald-50 text-emerald-700",
  red: "border-red-200 bg-red-50 text-red-700",
  gray: "border-slate-200 bg-slate-50 text-slate-600",
};

type Hint = { label: string; tone: keyof typeof hintTones };

const toolHints = (item: JSONObject): Hint[] => {
  const annotations = isObject(item.annotations) ? item.annotations : {};
  return [
    annotations.readOnlyHint === true && {
      label: "Read-only",
      tone: "green" as const,
    },
    annotations.destructiveHint === true &&
      annotations.readOnlyHint !== true && {
        label: "Destructive",
        tone: "red" as const,
      },
    annotations.idempotentHint === true && {
      label: "Idempotent",
      tone: "gray" as const,
    },
    annotations.openWorldHint === true && {
      label: "Open world",
      tone: "gray" as const,
    },
  ].filter((hint): hint is Hint => !!hint);
};

const HintChip = (props: { hint: Hint }) => (
  <span
    className={twMerge(
      "inline-flex shrink-0 items-center rounded-md border px-1.5 py-px text-micro font-semibold",
      hintTones[props.hint.tone],
    )}
  >
    {props.hint.label}
  </span>
);

const formatTime = (ms: number) =>
  new Date(ms).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

const StatusPill = (props: { connected: boolean; connecting: boolean }) => (
  <span className="flex shrink-0 items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2 py-0.5 text-micro font-semibold text-slate-600">
    <StatusDot
      tone={props.connecting ? "blue" : props.connected ? "green" : "gray"}
      pulse={props.connecting}
    />
    {props.connecting
      ? "Connecting"
      : props.connected
        ? "Connected"
        : "Not connected"}
  </span>
);

const ListSkeleton = () => (
  <div
    className="flex flex-col gap-1 p-1.5 motion-safe:animate-pulse"
    role="status"
    aria-label="Loading"
  >
    {Array.from({ length: 5 }).map((_, idx) => (
      <div key={idx} className="flex flex-col gap-1.5 rounded-lg px-2.5 py-2">
        <div className="h-3 w-2/5 rounded bg-slate-200" />
        <div className="h-2.5 w-4/5 rounded bg-slate-100" />
      </div>
    ))}
  </div>
);

const ConnectionSettings = (props: {
  state: State;
  versions: string[];
  onChange: (patch: Partial<State>) => void;
}) => {
  const { state } = props;
  let capabilitiesError: string | undefined;
  try {
    parseCapabilities(state.capabilities);
  } catch (err) {
    capabilitiesError = (err as Error).message;
  }

  return (
    <Popover position="bottom-end" width={320} withinPortal>
      <Popover.Target>
        <Tooltip label="Connection settings" withArrow>
          <ActionIcon
            variant="default"
            size={26}
            aria-label="Connection settings"
          >
            <Settings2 size={14} />
          </ActionIcon>
        </Tooltip>
      </Popover.Target>
      <Popover.Dropdown>
        <div className="flex flex-col gap-3">
          <div>
            <p className="text-body font-semibold text-slate-800">
              Connection settings
            </p>
            <p className="mt-0.5 text-xs leading-5 text-slate-500">
              The changes apply to the next connection.
            </p>
          </div>
          <Select
            label="Protocol version"
            searchable
            allowDeselect={false}
            comboboxProps={{ withinPortal: false }}
            data={props.versions}
            value={state.protocolVersion}
            onChange={(value) =>
              props.onChange({
                protocolVersion: value || DEFAULT_PROTOCOL_VERSION,
              })
            }
          />
          <div className="grid grid-cols-2 gap-3">
            <TextInput
              label="Client name"
              value={state.clientName}
              onChange={(event) =>
                props.onChange({ clientName: event.currentTarget.value })
              }
            />
            <TextInput
              label="Client version"
              value={state.clientVersion}
              onChange={(event) =>
                props.onChange({ clientVersion: event.currentTarget.value })
              }
            />
          </div>
          <Textarea
            label="Client capabilities"
            description="A JSON object sent with the handshake"
            autosize
            minRows={2}
            maxRows={8}
            spellCheck={false}
            styles={monospace}
            error={capabilitiesError}
            value={state.capabilities}
            onChange={(event) =>
              props.onChange({ capabilities: event.currentTarget.value })
            }
          />
        </div>
      </Popover.Dropdown>
    </Popover>
  );
};

const HistoryRow = (props: { exchange: Exchange }) => {
  const { exchange } = props;
  const [opened, setOpened] = React.useState(false);
  const failed = !!exchange.error;
  const toolError =
    isObject(exchange.response) &&
    isObject(exchange.response.result) &&
    exchange.response.result.isError === true;

  return (
    <li className="border-b border-slate-100 last:border-b-0">
      <button
        type="button"
        aria-expanded={opened}
        onClick={() => setOpened((value) => !value)}
        className="flex w-full cursor-pointer items-center gap-3 px-4 py-2.5 text-left outline-none transition-colors duration-150 hover:bg-slate-50 focus-visible:bg-slate-50"
      >
        <StatusDot tone={failed ? "red" : toolError ? "amber" : "green"} />
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-baseline gap-2">
            <span className="shrink-0 font-mono text-xs font-semibold text-slate-800">
              {exchange.method}
            </span>
            {exchange.target && (
              <span className="min-w-0 truncate font-mono text-xs text-slate-500">
                {exchange.target}
              </span>
            )}
          </span>
          {failed && (
            <span className="mt-0.5 block truncate text-micro font-semibold text-red-700">
              {exchange.error}
            </span>
          )}
        </span>
        <span className="hidden shrink-0 font-mono text-micro text-slate-400 @md:inline">
          {exchange.httpStatus ? `${exchange.httpStatus} · ` : ""}
          {exchange.durationMs !== undefined
            ? formatDuration(exchange.durationMs)
            : ""}
        </span>
        <span className="shrink-0 font-mono text-micro text-slate-400">
          {formatTime(exchange.startedAt)}
        </span>
        <ChevronRight
          size={13}
          strokeWidth={2.4}
          className={twMerge(
            "shrink-0 text-slate-400 transition-transform duration-200",
            opened && "rotate-90",
          )}
        />
      </button>
      <Collapse opened={opened}>
        <div className="grid gap-3 px-4 pb-4 @3xl:grid-cols-2">
          <div className="flex min-w-0 flex-col gap-1.5">
            <p className="text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
              Request
            </p>
            <CodeBlock>{toJSONText(exchange.request)}</CodeBlock>
          </div>
          <div className="flex min-w-0 flex-col gap-1.5">
            <p className="text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
              Response
              {exchange.httpStatus ? ` · HTTP ${exchange.httpStatus}` : ""}
            </p>
            {exchange.response !== undefined ? (
              <CodeBlock>{toJSONText(exchange.response)}</CodeBlock>
            ) : (
              <p className="rounded-lg border border-dashed border-slate-200 px-3 py-2 text-xs text-slate-500">
                {exchange.error ?? "No body"}
              </p>
            )}
          </div>
        </div>
      </Collapse>
    </li>
  );
};

const MCPPlayground = (props: { service: Service }) => {
  const { service } = props;
  const key = service.metadata?.uid || service.metadata?.name || "";
  const config = getMCPConfig(service);
  const endpoint = config?.endpoint?.trim() || "/mcp";
  const url = React.useMemo(() => {
    try {
      return new URL(
        endpoint,
        `${getServicePublicURL(service, getDomain())}/`,
      ).toString();
    } catch {
      return endpoint;
    }
  }, [endpoint, service]);

  const [state, setState] = React.useState<State>(() => {
    const cached = sessions.get(key);
    return cached ? restore(cached) : initialState(service);
  });
  const [instructionsOpened, setInstructionsOpened] = React.useState(false);
  const sessionRef = React.useRef(state.sessionID);
  const controllersRef = React.useRef(new Set<AbortController>());

  const versions = Array.from(
    new Set([
      ...PROTOCOL_VERSIONS,
      ...(config?.protocol?.versions ?? []),
      state.protocolVersion,
    ]),
  );
  const connected = !!state.server;

  React.useEffect(() => {
    sessions.set(key, state);
  }, [key, state]);

  React.useEffect(() => {
    const controllers = controllersRef.current;
    return () => controllers.forEach((controller) => controller.abort());
  }, []);

  const patch = (
    value: Partial<State> | ((current: State) => Partial<State>),
  ) =>
    setState((current) => ({
      ...current,
      ...(typeof value === "function" ? value(current) : value),
    }));

  const patchList = (kind: Kind, value: Partial<Listing>) =>
    patch((current) => ({
      lists: {
        ...current.lists,
        [kind]: { ...current.lists[kind], ...value },
      },
    }));

  const request = async (
    method: string,
    params?: JSONObject,
    options?: { target?: string; notification?: boolean },
  ): Promise<Reply> => {
    const clientConfig: ClientConfig = {
      url,
      protocolVersion: state.protocolVersion,
      clientName: state.clientName,
      clientVersion: state.clientVersion,
      capabilities: state.capabilities,
      sessionID: sessionRef.current,
    };
    const controller = new AbortController();
    controllersRef.current.add(controller);
    try {
      const reply = await sendMCP({
        config: clientConfig,
        method,
        params,
        target: options?.target,
        notification: options?.notification,
        signal: controller.signal,
      });
      if (controller.signal.aborted) return reply;
      patch((current) => ({
        history: [reply.exchange, ...current.history].slice(0, HISTORY_LIMIT),
      }));
      if (reply.expired) {
        sessionRef.current = undefined;
        patch({
          sessionID: undefined,
          server: undefined,
          connectError: "The session has expired. Connect again.",
        });
      }
      return reply;
    } finally {
      controllersRef.current.delete(controller);
    }
  };

  const list = async (kind: Kind, more = false) => {
    const cursor = more ? state.lists[kind].cursor : undefined;
    patchList(kind, { loading: true, error: undefined });
    const reply = await request(`${kind}/list`, cursor ? { cursor } : {});
    if (reply.error) {
      patchList(kind, { loading: false, error: reply.error });
      return;
    }
    const result = isObject(reply.result) ? reply.result : {};
    const items = (Array.isArray(result[kind]) ? result[kind] : []).filter(
      isObject,
    );
    patch((current) => ({
      lists: {
        ...current.lists,
        [kind]: {
          items: more ? [...current.lists[kind].items, ...items] : items,
          cursor:
            typeof result.nextCursor === "string" && result.nextCursor
              ? result.nextCursor
              : undefined,
          loaded: true,
          loading: false,
        },
      },
    }));
  };

  const connect = async () => {
    try {
      parseCapabilities(state.capabilities);
    } catch (err) {
      patch({ connectError: (err as Error).message });
      return;
    }

    patch({ connecting: true, connectError: undefined, ping: undefined });
    sessionRef.current = undefined;
    const discover = usesServerDiscover(state.protocolVersion);
    const reply = discover
      ? await request("server/discover", {})
      : await request("initialize", {
          protocolVersion: state.protocolVersion,
          capabilities: parseCapabilities(state.capabilities),
          clientInfo: {
            name: state.clientName.trim() || DEFAULT_CLIENT_NAME,
            version: state.clientVersion.trim() || DEFAULT_CLIENT_VERSION,
          },
        });

    if (reply.error) {
      patch({
        connecting: false,
        server: undefined,
        sessionID: undefined,
        connectError: `Could not connect: ${reply.error}`,
      });
      return;
    }

    sessionRef.current = reply.sessionID;
    const server = parseServer(reply.result, state.protocolVersion);
    patch({ connecting: false, server, sessionID: reply.sessionID });

    if (!discover) {
      await request("notifications/initialized", undefined, {
        notification: true,
      });
    }

    const supported = kinds.filter(
      (kind) => !server.capabilities || isObject(server.capabilities[kind]),
    );
    await Promise.all(supported.map((kind) => list(kind)));
  };

  const disconnect = () => {
    const sessionID = sessionRef.current;
    sessionRef.current = undefined;
    patch({ sessionID: undefined, server: undefined, ping: undefined });
    if (!sessionID) return;
    void fetch(url, {
      method: "DELETE",
      credentials: "include",
      headers: {
        "MCP-Protocol-Version": state.protocolVersion,
        "Mcp-Session-Id": sessionID,
      },
    }).catch(() => undefined);
  };

  const ping = async () => {
    const reply = await request("ping", {});
    patch({
      ping: reply.error
        ? { error: reply.error }
        : { ms: reply.exchange.durationMs },
    });
  };

  const execute = async (kind: Kind, id: string) => {
    const isCustom = id === CUSTOM;
    const target = (isCustom ? state.custom[kind] : id).trim();
    const keyOf = runKey(kind, id);
    const item = state.lists[kind].items.find(
      (itm) => itemID(kind, itm) === id,
    );
    const fields =
      kind === "tools"
        ? schemaFields(item?.inputSchema)
        : kind === "prompts"
          ? promptFields(item?.arguments)
          : [];

    let params: JSONObject;
    try {
      if (!target) {
        throw new Error(
          `Enter the ${kindInfo[kind].customLabel.toLowerCase()}.`,
        );
      }
      if (kind === "resources") {
        params = { uri: target };
      } else {
        const mode = isCustom ? "json" : (state.modes[keyOf] ?? "form");
        const args =
          mode === "json"
            ? parseArguments(state.json[keyOf] ?? "")
            : buildArguments(fields, state.values[keyOf] ?? {});
        params = { name: target, arguments: args };
      }
    } catch (err) {
      patch((current) => ({
        formErrors: { ...current.formErrors, [keyOf]: (err as Error).message },
      }));
      return;
    }

    patch((current) => ({
      formErrors: { ...current.formErrors, [keyOf]: undefined },
      runs: { ...current.runs, [keyOf]: { pending: true } },
    }));
    const reply = await request(kindInfo[kind].method, params, { target });
    patch((current) => ({
      runs: {
        ...current.runs,
        [keyOf]: {
          pending: false,
          exchange: reply.exchange,
          result: reply.result,
          error: reply.error,
        },
      },
    }));
  };

  const select = (kind: Kind, id?: string) =>
    patch((current) => ({ selected: { ...current.selected, [kind]: id } }));

  const renderList = (kind: Kind) => {
    const info = kindInfo[kind];
    const listing = state.lists[kind];
    const filter = state.filters[kind].trim().toLowerCase();
    const selected = state.selected[kind];
    const items = filter
      ? listing.items.filter((item) =>
          `${itemID(kind, item)} ${itemTitle(item) ?? ""} ${typeof item.name === "string" ? item.name : ""} ${itemDescription(item) ?? ""}`
            .toLowerCase()
            .includes(filter),
        )
      : listing.items;
    const declared =
      !state.server?.capabilities || isObject(state.server.capabilities[kind]);

    return (
      <div
        className={twMerge(
          "min-h-0 flex-col @3xl:flex @3xl:w-72 @3xl:shrink-0 @3xl:border-r @3xl:border-slate-200",
          selected !== undefined ? "hidden" : "flex",
        )}
      >
        <div className="flex items-center gap-1 border-b border-slate-100 p-2">
          <TextInput
            className="min-w-0 flex-1"
            aria-label={`Filter the ${info.label.toLowerCase()}`}
            placeholder={`Filter ${info.label.toLowerCase()}`}
            leftSection={<Search size={13} />}
            disabled={listing.items.length === 0}
            value={state.filters[kind]}
            onChange={(event) => {
              const value = event.currentTarget.value;
              patch((current) => ({
                filters: { ...current.filters, [kind]: value },
              }));
            }}
          />
          <Tooltip label={`List the ${info.label.toLowerCase()}`} withArrow>
            <ActionIcon
              variant="subtle"
              color="gray"
              aria-label={`List the ${info.label.toLowerCase()}`}
              loading={listing.loading}
              onClick={() => void list(kind)}
            >
              <RefreshCw size={14} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label={info.custom} withArrow>
            <ActionIcon
              variant={selected === CUSTOM ? "light" : "subtle"}
              color="gray"
              aria-label={info.custom}
              onClick={() => select(kind, CUSTOM)}
            >
              <Plus size={15} />
            </ActionIcon>
          </Tooltip>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
          {listing.loading && !listing.loaded ? (
            <ListSkeleton />
          ) : listing.error && listing.items.length === 0 ? (
            <div className="p-3">
              <p
                role="alert"
                className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold leading-5 text-red-700"
              >
                <CircleAlert size={14} className="mt-0.5 shrink-0" />
                <span className="min-w-0 break-words">{listing.error}</span>
              </p>
            </div>
          ) : !listing.loaded ? (
            <EmptyState
              icon={info.icon}
              title={`No ${info.label.toLowerCase()} loaded`}
              description={
                connected
                  ? declared
                    ? `List the ${info.label.toLowerCase()} that the server exposes.`
                    : `The server does not declare ${info.label.toLowerCase()}.`
                  : `Connect to the server first, or list the ${info.label.toLowerCase()} directly if it does not use sessions.`
              }
              action={
                <Button
                  variant="default"
                  size="compact-sm"
                  leftSection={<RefreshCw size={13} />}
                  onClick={() => void list(kind)}
                >
                  List {info.label.toLowerCase()}
                </Button>
              }
            />
          ) : items.length === 0 ? (
            <EmptyState
              icon={info.icon}
              title={
                filter
                  ? `No ${info.label.toLowerCase()} match`
                  : `The server has no ${info.label.toLowerCase()}`
              }
            />
          ) : (
            <ul className="flex flex-col gap-0.5 p-1.5">
              {items.map((item) => {
                const id = itemID(kind, item);
                const title = itemTitle(item);
                const description = itemDescription(item);
                const hints = kind === "tools" ? toolHints(item) : [];
                const active = selected === id;
                return (
                  <li key={id}>
                    <button
                      type="button"
                      aria-current={active || undefined}
                      onClick={() => select(kind, id)}
                      className={twMerge(
                        "flex w-full cursor-pointer flex-col gap-0.5 rounded-lg px-2.5 py-2 text-left outline-none transition-colors duration-150 hover:bg-slate-100 focus-visible:bg-slate-100",
                        active && "bg-slate-100",
                      )}
                    >
                      <span className="flex min-w-0 items-center gap-1.5">
                        <span className="min-w-0 truncate font-mono text-xs font-semibold text-slate-800">
                          {kind === "resources"
                            ? (title ??
                              (typeof item.name === "string" ? item.name : id))
                            : id}
                        </span>
                        {hints.slice(0, 1).map((hint) => (
                          <HintChip key={hint.label} hint={hint} />
                        ))}
                      </span>
                      {kind === "resources" ? (
                        <span className="truncate font-mono text-micro text-slate-500">
                          {id}
                        </span>
                      ) : (
                        description && (
                          <span className="line-clamp-2 text-micro leading-4 text-slate-500">
                            {description}
                          </span>
                        )
                      )}
                    </button>
                  </li>
                );
              })}
              {listing.cursor && !filter && (
                <li className="px-1 pt-1">
                  <Button
                    fullWidth
                    variant="subtle"
                    color="gray"
                    size="compact-sm"
                    loading={listing.loading}
                    onClick={() => void list(kind, true)}
                  >
                    Load more
                  </Button>
                </li>
              )}
            </ul>
          )}
          {listing.error && listing.items.length > 0 && (
            <p className="mx-2 mb-2 rounded-lg border border-red-200 bg-red-50 px-2.5 py-1.5 text-micro font-semibold text-red-700">
              {listing.error}
            </p>
          )}
        </div>
      </div>
    );
  };

  const renderDetail = (kind: Kind) => {
    const info = kindInfo[kind];
    const id = state.selected[kind];

    if (id === undefined) {
      return (
        <div className="hidden min-h-0 flex-1 @3xl:flex">
          <EmptyState
            className="w-full"
            icon={MousePointerClick}
            title={`Select a ${info.singular}`}
            description={
              kind === "tools"
                ? "Inspect its input schema, fill in the arguments and run it."
                : kind === "resources"
                  ? "Inspect it and read its contents."
                  : "Fill in its arguments and get its messages."
            }
          />
        </div>
      );
    }

    const isCustom = id === CUSTOM;
    const keyOf = runKey(kind, id);
    const item = state.lists[kind].items.find(
      (itm) => itemID(kind, itm) === id,
    );
    const fields =
      kind === "tools"
        ? schemaFields(item?.inputSchema)
        : kind === "prompts"
          ? promptFields(item?.arguments)
          : [];
    const title = item ? itemTitle(item) : undefined;
    const description = item ? itemDescription(item) : undefined;
    const run = state.runs[keyOf];
    const hints = item && kind === "tools" ? toolHints(item) : [];
    const name =
      kind === "resources" && item && typeof item.name === "string"
        ? item.name
        : undefined;
    const meta =
      kind === "resources" && item
        ? [
            typeof item.mimeType === "string" ? item.mimeType : undefined,
            typeof item.size === "number"
              ? `${item.size.toLocaleString()} bytes`
              : undefined,
          ].filter(Boolean)
        : [];

    const action = (
      <Button
        size="compact-sm"
        leftSection={<Play size={12} fill="currentColor" />}
        loading={run?.pending}
        onClick={() => void execute(kind, id)}
      >
        {info.action}
      </Button>
    );

    return (
      <div
        key={keyOf}
        className="flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto overscroll-contain"
      >
        <div className="flex flex-col gap-4 p-4">
          <BackButton
            label={`All ${info.label.toLowerCase()}`}
            onClick={() => select(kind, undefined)}
          />

          {isCustom ? (
            <div>
              <h3 className="text-body font-bold text-slate-900">
                {info.custom}
              </h3>
              <p className="mt-1 text-xs leading-5 text-slate-500">
                Use this when the {info.singular} is not listed, for example
                when a Policy hides the listing.
              </p>
              <TextInput
                className="mt-3"
                label={info.customLabel}
                placeholder={info.customPlaceholder}
                spellCheck={false}
                styles={monospace}
                value={state.custom[kind]}
                onChange={(event) => {
                  const value = event.currentTarget.value;
                  patch((current) => ({
                    custom: { ...current.custom, [kind]: value },
                  }));
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && kind === "resources") {
                    event.preventDefault();
                    void execute(kind, id);
                  }
                }}
              />
            </div>
          ) : (
            <div className="min-w-0">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <h3 className="min-w-0 break-all font-mono text-body font-bold text-slate-900">
                  {kind === "resources" ? (title ?? name ?? id) : id}
                </h3>
                {hints.map((hint) => (
                  <HintChip key={hint.label} hint={hint} />
                ))}
              </div>
              {kind !== "resources" && title && title !== id && (
                <p className="mt-0.5 text-xs font-semibold text-slate-600">
                  {title}
                </p>
              )}
              {kind === "resources" && (
                <p className="mt-1 flex min-w-0 items-center gap-1">
                  <span className="min-w-0 break-all font-mono text-micro text-slate-500">
                    {id}
                  </span>
                  <CopyButton value={() => id} label="Copy the URI" />
                </p>
              )}
              {meta.length > 0 && (
                <p className="mt-1 text-micro text-slate-500">
                  {meta.join(" · ")}
                </p>
              )}
              {description && <Description text={description} />}
            </div>
          )}

          {kind === "resources" ? (
            <div className="flex justify-end">{action}</div>
          ) : (
            <Arguments
              fields={fields}
              values={state.values[keyOf] ?? {}}
              onValues={(values) =>
                patch((current) => ({
                  values: { ...current.values, [keyOf]: values },
                  formErrors: { ...current.formErrors, [keyOf]: undefined },
                }))
              }
              mode={state.modes[keyOf] ?? "form"}
              onMode={(mode) =>
                patch((current) => ({
                  modes: { ...current.modes, [keyOf]: mode },
                }))
              }
              json={state.json[keyOf] ?? ""}
              onJSON={(value) =>
                patch((current) => ({
                  json: { ...current.json, [keyOf]: value },
                  formErrors: { ...current.formErrors, [keyOf]: undefined },
                }))
              }
              jsonOnly={isCustom}
              emptyLabel={`This ${info.singular} takes no arguments.`}
              error={state.formErrors[keyOf]}
              onSubmit={() => void execute(kind, id)}
            >
              {action}
            </Arguments>
          )}

          {kind === "resources" && state.formErrors[keyOf] && (
            <p
              role="alert"
              className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold leading-5 text-red-700"
            >
              <CircleAlert size={14} className="mt-0.5 shrink-0" />
              <span className="min-w-0 break-words">
                {state.formErrors[keyOf]}
              </span>
            </p>
          )}

          {run && <ResultCard run={run} kind={info.result} />}
        </div>
      </div>
    );
  };

  const tabs: {
    value: Tab;
    label: string;
    icon: LucideIcon;
    count?: number;
  }[] = [
    ...kinds.map((kind) => ({
      value: kind,
      label: kindInfo[kind].label,
      icon: kindInfo[kind].icon,
      count: state.lists[kind].loaded
        ? state.lists[kind].items.length
        : undefined,
    })),
    {
      value: "history" as const,
      label: "History",
      icon: History,
      count: state.history.length || undefined,
    },
  ];

  const capabilityNames = state.server?.capabilities
    ? Object.keys(state.server.capabilities)
    : [];

  return (
    <div className="@container w-full">
      <div className="flex w-full flex-col overflow-hidden rounded-xl border border-slate-200 bg-white shadow-card @3xl:h-[calc(100dvh-11.125rem)] @3xl:min-h-[30rem]">
        <header className="flex flex-col gap-3 border-b border-slate-200 bg-slate-50/60 px-4 py-3 @xl:flex-row @xl:items-center">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
              <Plug size={16} strokeWidth={2.2} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                <h2 className="min-w-0 truncate text-sm font-bold text-slate-800">
                  {state.server?.name || "MCP server"}
                </h2>
                {state.server?.version && (
                  <span className="shrink-0 font-mono text-micro text-slate-500">
                    v{state.server.version}
                  </span>
                )}
                <StatusPill
                  connected={connected}
                  connecting={state.connecting}
                />
              </div>
              <div className="mt-0.5 flex min-w-0 items-center gap-0.5">
                <span
                  className="min-w-0 truncate font-mono text-micro text-slate-500"
                  title={url}
                >
                  {url}
                </span>
                <CopyButton value={() => url} label="Copy the endpoint" />
              </div>
            </div>
          </div>
          <div className="flex shrink-0 items-center justify-end gap-1.5">
            <ConnectionSettings
              state={state}
              versions={versions}
              onChange={(value) => patch(value)}
            />
            {connected && (
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<Activity size={13} />}
                onClick={() => void ping()}
              >
                Ping
              </Button>
            )}
            {connected ? (
              <Button
                variant="default"
                size="compact-sm"
                leftSection={<Unplug size={13} />}
                onClick={disconnect}
              >
                Disconnect
              </Button>
            ) : (
              <Button
                size="compact-sm"
                leftSection={<Plug size={13} />}
                loading={state.connecting}
                onClick={() => void connect()}
              >
                Connect
              </Button>
            )}
          </div>
        </header>

        {state.connectError && !state.server && (
          <p
            role="alert"
            className="flex items-start gap-2 border-b border-red-200 bg-red-50 px-4 py-2 text-xs font-semibold leading-5 text-red-700"
          >
            <CircleAlert size={14} className="mt-0.5 shrink-0" />
            <span className="min-w-0 break-words">{state.connectError}</span>
          </p>
        )}

        {state.server && (
          <div className="border-b border-slate-200">
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5 px-4 py-2 text-micro text-slate-500">
              <span>
                Protocol{" "}
                <span className="font-mono font-semibold text-slate-700">
                  {state.server.protocolVersion ?? state.protocolVersion}
                </span>
              </span>
              {state.sessionID && (
                <span className="flex min-w-0 items-center gap-1">
                  Session
                  <span
                    className="max-w-[10rem] truncate font-mono font-semibold text-slate-700"
                    title={state.sessionID}
                  >
                    {state.sessionID}
                  </span>
                </span>
              )}
              {state.ping && (
                <span
                  className={twMerge(
                    state.ping.error && "font-semibold text-red-700",
                  )}
                  title={state.ping.error}
                >
                  Ping{" "}
                  <span className="font-mono font-semibold">
                    {state.ping.error
                      ? "failed"
                      : formatDuration(state.ping.ms ?? 0)}
                  </span>
                </span>
              )}
              {capabilityNames.length > 0 && (
                <span className="flex flex-wrap items-center gap-1">
                  {capabilityNames.map((name) => (
                    <span
                      key={name}
                      className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-px font-semibold text-slate-600"
                    >
                      {name}
                    </span>
                  ))}
                </span>
              )}
              {state.server.instructions && (
                <button
                  type="button"
                  aria-expanded={instructionsOpened}
                  onClick={() => setInstructionsOpened((value) => !value)}
                  className="ml-auto flex cursor-pointer items-center gap-1 font-semibold text-slate-600 hover:text-slate-900"
                >
                  Instructions
                  <ChevronDown
                    size={12}
                    strokeWidth={2.5}
                    className={twMerge(
                      "transition-transform duration-200",
                      instructionsOpened && "rotate-180",
                    )}
                  />
                </button>
              )}
            </div>
            {state.server.instructions && (
              <Collapse opened={instructionsOpened}>
                <p className="mx-4 mb-3 max-h-48 overflow-y-auto whitespace-pre-wrap rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-600">
                  {state.server.instructions}
                </p>
              </Collapse>
            )}
          </div>
        )}

        <div className="overflow-x-auto border-b border-slate-200 bg-white p-2">
          <SegmentedControl
            fullWidth
            aria-label="Playground content"
            value={state.tab}
            onChange={(value) => patch({ tab: value as Tab })}
            styles={{ root: { minWidth: "max-content" } }}
            data={tabs.map((tab) => ({
              value: tab.value,
              label: (
                <span className="inline-flex items-center gap-1.5 whitespace-nowrap">
                  <tab.icon size={13} className="hidden @md:block" />
                  {tab.label}
                  {tab.count !== undefined && (
                    <span className="hidden rounded-full bg-slate-200/70 px-1.5 text-micro font-semibold tabular-nums text-slate-600 @sm:inline">
                      {tab.count}
                    </span>
                  )}
                </span>
              ),
            }))}
          />
        </div>

        {state.tab === "history" ? (
          <div className="flex min-h-0 flex-1 flex-col">
            {state.history.length === 0 ? (
              <EmptyState
                className="flex-1"
                icon={History}
                title="No requests yet"
                description="Every JSON-RPC request and response of this playground shows up here."
              />
            ) : (
              <>
                <div className="flex items-center justify-between gap-2 border-b border-slate-100 py-1.5 pl-4 pr-2">
                  <span className="text-xs font-semibold text-slate-600">
                    {state.history.length}{" "}
                    {state.history.length === 1 ? "request" : "requests"}
                  </span>
                  <Button
                    variant="subtle"
                    color="gray"
                    size="compact-xs"
                    leftSection={<Trash2 size={12} />}
                    onClick={() => patch({ history: [] })}
                  >
                    Clear
                  </Button>
                </div>
                <ul className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
                  {state.history.map((exchange) => (
                    <HistoryRow key={exchange.id} exchange={exchange} />
                  ))}
                </ul>
              </>
            )}
          </div>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col @3xl:flex-row">
            {renderList(state.tab)}
            {renderDetail(state.tab)}
          </div>
        )}
      </div>
    </div>
  );
};

export default MCPPlayground;
