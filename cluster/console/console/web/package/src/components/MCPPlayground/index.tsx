import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { getServicePublicURL } from "@/utils/octelium";
import { getDomain } from "@/utils";
import {
  Autocomplete,
  Badge,
  Button,
  Drawer,
  Group,
  Select,
  Text,
  TextInput,
  Textarea,
} from "@mantine/core";
import {
  Braces,
  Check,
  CircleAlert,
  Play,
  Plus,
  RefreshCw,
  Send,
  Trash2,
  Wrench,
} from "lucide-react";
import * as React from "react";

type MCPMethod =
  | "initialize"
  | "tools/list"
  | "tools/call"
  | "resources/list"
  | "resources/read"
  | "prompts/list"
  | "prompts/get"
  | "ping"
  | "server/discover";
type ArgumentType = "string" | "number" | "boolean" | "null" | "json";

type ArgumentRow = {
  id: string;
  key: string;
  value: string;
  type: ArgumentType;
};

const DEFAULT_PROTOCOL_VERSION = "2025-03-26";
const PROTOCOL_VERSIONS = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];
const SERVER_DISCOVER_VERSION = "2026-07-28";

const newArgument = (): ArgumentRow => ({
  id: crypto.randomUUID(),
  key: "",
  value: "",
  type: "string",
});

const getMCPConfig = (service: Service) => {
  const type = service.spec?.config?.type;
  return type?.oneofKind === "mcp" ? type.mcp : undefined;
};

const parseEventStream = (body: string): unknown => {
  const data = body
    .split(/\r?\n/)
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trim())
    .filter((line) => line && line !== "[DONE]");
  const last = data.at(-1);
  if (!last) return body;
  try {
    return JSON.parse(last);
  } catch {
    return data.join("\n");
  }
};

const parseResponse = (body: string, contentType: string): unknown => {
  if (contentType.includes("text/event-stream") || body.includes("data:")) {
    return parseEventStream(body);
  }
  try {
    return JSON.parse(body);
  } catch {
    return body;
  }
};

const formatResult = (value: unknown) =>
  typeof value === "string" ? value : JSON.stringify(value, null, 2);

const ArgumentEditor = (props: {
  title: string;
  description: string;
  rows: ArgumentRow[];
  onAdd: () => void;
  onUpdate: (id: string, patch: Partial<ArgumentRow>) => void;
  onRemove: (id: string) => void;
}) => (
  <div className="space-y-3">
    <div className="flex items-center justify-between gap-3">
      <div>
        <p className="text-xs font-bold text-slate-700">{props.title}</p>
        <p className="text-[0.68rem] font-semibold text-slate-500">
          {props.description}
        </p>
      </div>
      <Button
        type="button"
        size="compact-xs"
        variant="light"
        leftSection={<Plus size={13} />}
        onClick={props.onAdd}
      >
        Add value
      </Button>
    </div>
    {props.rows.length === 0 ? (
      <div className="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-3 py-4 text-center text-xs font-semibold text-slate-500">
        No values. This is valid when the method accepts an empty argument object.
      </div>
    ) : (
      <div className="space-y-2">
        {props.rows.map((row) => (
          <div
            key={row.id}
            className="grid gap-2 rounded-lg border border-slate-200 bg-slate-50/70 p-2 sm:grid-cols-[1fr_115px_1.3fr_auto] sm:items-end"
          >
            <TextInput
              label="Key"
              size="xs"
              placeholder="argument name"
              value={row.key}
              onChange={(event) =>
                props.onUpdate(row.id, { key: event.currentTarget.value })
              }
            />
            <Select
              label="Type"
              size="xs"
              data={[
                { value: "string", label: "Text" },
                { value: "number", label: "Number" },
                { value: "boolean", label: "Boolean" },
                { value: "null", label: "Null" },
                { value: "json", label: "JSON" },
              ]}
              value={row.type}
              onChange={(value) =>
                props.onUpdate(row.id, {
                  type: (value as ArgumentType) || "string",
                  value: value === "boolean" ? "false" : "",
                })
              }
              allowDeselect={false}
            />
            {row.type === "boolean" ? (
              <Select
                label="Value"
                size="xs"
                data={["true", "false"]}
                value={row.value || "false"}
                onChange={(value) =>
                  props.onUpdate(row.id, { value: value || "false" })
                }
                allowDeselect={false}
              />
            ) : row.type === "null" ? (
              <Text size="xs" fw={600} c="dimmed" className="pb-2">
                This value is sent as null.
              </Text>
            ) : row.type === "json" ? (
              <Textarea
                label="Value"
                size="xs"
                autosize
                minRows={1}
                placeholder='e.g. {"limit": 10}'
                value={row.value}
                onChange={(event) =>
                  props.onUpdate(row.id, { value: event.currentTarget.value })
                }
              />
            ) : (
              <TextInput
                label="Value"
                size="xs"
                type={row.type === "number" ? "number" : "text"}
                placeholder={row.type === "number" ? "0" : "value"}
                value={row.value}
                onChange={(event) =>
                  props.onUpdate(row.id, { value: event.currentTarget.value })
                }
              />
            )}
            <Button
              type="button"
              size="compact-xs"
              variant="subtle"
              color="red"
              aria-label={`Remove ${row.key || "argument"}`}
              onClick={() => props.onRemove(row.id)}
            >
              <Trash2 size={14} />
            </Button>
          </div>
        ))}
      </div>
    )}
  </div>
);

const MCPPlayground = (props: { service: Service }) => {
  const { service } = props;
  const [opened, setOpened] = React.useState(false);
  const [method, setMethod] = React.useState<MCPMethod>("tools/list");
  const [protocolVersion, setProtocolVersion] = React.useState(
    getMCPConfig(service)?.protocol?.versions[0] || DEFAULT_PROTOCOL_VERSION,
  );
  const [toolName, setToolName] = React.useState("");
  const [promptName, setPromptName] = React.useState("");
  const [resourceURI, setResourceURI] = React.useState("");
  const [cursor, setCursor] = React.useState("");
  const [argumentsList, setArgumentsList] = React.useState<ArgumentRow[]>([]);
  const [clientName, setClientName] = React.useState("Octelium Console");
  const [clientVersion, setClientVersion] = React.useState("1.0.0");
  const [capabilitiesJSON, setCapabilitiesJSON] = React.useState("{}");
  const [sessionID, setSessionID] = React.useState("");
  const [pending, setPending] = React.useState(false);
  const [error, setError] = React.useState<string>();
  const [response, setResponse] = React.useState<unknown>();
  const [request, setRequest] = React.useState<unknown>();

  const config = getMCPConfig(service);
  const endpoint = config?.endpoint?.trim() || "/mcp";
  const isPublic = service.spec?.isPublic === true;
  const protocolDate = Date.parse(`${protocolVersion}T00:00:00Z`);
  const serverDiscoverDate = Date.parse(`${SERVER_DISCOVER_VERSION}T00:00:00Z`);
  const supportsServerDiscover =
    Number.isFinite(protocolDate) && protocolDate >= serverDiscoverDate;
  const supportsInitialize = !supportsServerDiscover;
  const methodGroups = React.useMemo(
    () =>
      [
        {
          group: "Session",
          items: supportsInitialize
            ? [{ value: "initialize", label: "initialize" }]
            : [],
        },
        {
          group: "Tools",
          items: [
            { value: "tools/list", label: "tools/list" },
            { value: "tools/call", label: "tools/call" },
          ],
        },
        {
          group: "Resources",
          items: [
            { value: "resources/list", label: "resources/list" },
            { value: "resources/read", label: "resources/read" },
          ],
        },
        {
          group: "Prompts",
          items: [
            { value: "prompts/list", label: "prompts/list" },
            { value: "prompts/get", label: "prompts/get" },
          ],
        },
        {
          group: "Diagnostics",
          items: [
            { value: "ping", label: "ping" },
            ...(supportsServerDiscover
              ? [{ value: "server/discover", label: "server/discover" }]
              : []),
          ],
        },
      ].filter((group) => group.items.length > 0),
    [supportsInitialize, supportsServerDiscover],
  );
  const availableTools = React.useMemo(() => {
    const tools = (response as { result?: { tools?: unknown[] } } | undefined)?.result?.tools;
    if (!Array.isArray(tools)) return [];
    return tools.flatMap((tool) => {
      if (!tool || typeof tool !== "object") return [];
      const name = (tool as { name?: unknown }).name;
      return typeof name === "string" && name.length > 0 ? [name] : [];
    });
  }, [response]);
  const availablePrompts = React.useMemo(() => {
    const prompts = (response as { result?: { prompts?: unknown[] } } | undefined)?.result?.prompts;
    if (!Array.isArray(prompts)) return [];
    return prompts.flatMap((prompt) => {
      if (!prompt || typeof prompt !== "object") return [];
      const name = (prompt as { name?: unknown }).name;
      return typeof name === "string" && name.length > 0 ? [name] : [];
    });
  }, [response]);
  const availableResources = React.useMemo(() => {
    const resources = (response as { result?: { resources?: unknown[] } } | undefined)?.result?.resources;
    if (!Array.isArray(resources)) return [];
    return resources.flatMap((resource) => {
      if (!resource || typeof resource !== "object") return [];
      const uri = (resource as { uri?: unknown }).uri;
      return typeof uri === "string" && uri.length > 0 ? [uri] : [];
    });
  }, [response]);
  const targetURL = React.useMemo(() => {
    try {
      return new URL(
        endpoint,
        `${getServicePublicURL(service, getDomain())}/`,
      ).toString();
    } catch {
      return endpoint;
    }
  }, [endpoint, service]);

  React.useEffect(() => {
    const configured = getMCPConfig(service)?.protocol?.versions[0];
    setProtocolVersion(configured || DEFAULT_PROTOCOL_VERSION);
  }, [service]);

  React.useEffect(() => {
    if (method === "initialize" && !supportsInitialize) {
      setMethod("tools/list");
    }
    if (method === "server/discover" && !supportsServerDiscover) {
      setMethod("tools/list");
    }
  }, [method, supportsInitialize, supportsServerDiscover]);

  const updateArgument = (id: string, patch: Partial<ArgumentRow>) => {
    setArgumentsList((rows) =>
      rows.map((row) => (row.id === id ? { ...row, ...patch } : row)),
    );
  };

  const buildArguments = (): Record<string, unknown> => {
    const args: Record<string, unknown> = {};
    for (const row of argumentsList) {
      const key = row.key.trim();
      if (!key) throw new Error("Every argument needs a name.");
      if (key in args) throw new Error(`The argument “${key}” is duplicated.`);
      if (row.type === "string") args[key] = row.value;
      else if (row.type === "number") {
        const number = Number(row.value);
        if (!Number.isFinite(number)) {
          throw new Error(`The value for “${key}” must be a number.`);
        }
        args[key] = number;
      } else if (row.type === "boolean") {
        args[key] = row.value === "true";
      } else if (row.type === "null") {
        args[key] = null;
      } else {
        try {
          args[key] = JSON.parse(row.value || "null");
        } catch {
          throw new Error(`The JSON value for “${key}” is invalid.`);
        }
      }
    }
    return args;
  };

  const sendRequest = async () => {
    setError(undefined);
    setResponse(undefined);
    let params: Record<string, unknown> = {};
    try {
      if (method === "initialize") {
        let capabilities: unknown;
        try {
          capabilities = JSON.parse(capabilitiesJSON || "{}");
        } catch {
          throw new Error("Capabilities must contain valid JSON.");
        }
        if (!capabilities || typeof capabilities !== "object" || Array.isArray(capabilities)) {
          throw new Error("Capabilities must be a JSON object.");
        }
        params = {
          protocolVersion: protocolVersion || DEFAULT_PROTOCOL_VERSION,
          capabilities,
          clientInfo: {
            name: clientName.trim() || "Octelium Console",
            version: clientVersion.trim() || "1.0.0",
          },
        };
      } else if (method === "tools/call") {
        const name = toolName.trim();
        if (!name) throw new Error("Enter a tool name before sending the request.");
        params = { name, arguments: buildArguments() };
      } else if (method === "resources/read") {
        const uri = resourceURI.trim();
        if (!uri) throw new Error("Enter a resource URI before sending the request.");
        params = { uri };
      } else if (method === "prompts/get") {
        const name = promptName.trim();
        if (!name) throw new Error("Enter a prompt name before sending the request.");
        params = { name, arguments: buildArguments() };
      } else if (
        (method === "tools/list" ||
          method === "resources/list" ||
          method === "prompts/list") &&
        cursor.trim()
      ) {
        params = { cursor: cursor.trim() };
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "The request is invalid.");
      return;
    }

    const payload = {
      jsonrpc: "2.0",
      id: Date.now(),
      method,
      params,
    };
    setRequest(payload);
    setPending(true);
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), 30_000);
    try {
      const result = await fetch(targetURL, {
        method: "POST",
        credentials: "include",
        headers: {
          Accept: "application/json, text/event-stream",
          "Content-Type": "application/json",
          "MCP-Protocol-Version": protocolVersion || DEFAULT_PROTOCOL_VERSION,
          ...(sessionID ? { "Mcp-Session-Id": sessionID } : {}),
        },
        body: JSON.stringify(payload),
        signal: controller.signal,
      });
      const body = await result.text();
      const parsed = parseResponse(body, result.headers.get("content-type") || "");
      const returnedSessionID = result.headers.get("Mcp-Session-Id");
      if (returnedSessionID) setSessionID(returnedSessionID);
      if (!result.ok) {
        throw new Error(
          `${result.status} ${result.statusText}${body ? `: ${String(formatResult(parsed))}` : ""}`,
        );
      }
      setResponse(parsed);
    } catch (err) {
      setError(
        err instanceof DOMException && err.name === "AbortError"
          ? "The request timed out after 30 seconds."
          : err instanceof Error
            ? err.message
            : "The MCP request failed.",
      );
    } finally {
      window.clearTimeout(timeout);
      setPending(false);
    }
  };

  const close = () => setOpened(false);

  if (service.spec?.mode !== Service_Spec_Mode.MCP || !isPublic) return null;

  return (
    <>
      <Button
        type="button"
        size="compact-xs"
        variant="default"
        leftSection={<Play size={12} strokeWidth={2.5} />}
        onClick={() => setOpened(true)}
      >
        MCP playground
      </Button>

      <Drawer
        opened={opened}
        onClose={close}
        position="right"
        size="min(760px, 100vw)"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <Wrench size={15} className="shrink-0 text-slate-400" />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              MCP playground
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {service.metadata?.name}
            </span>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: "16px",
            backgroundColor: "#f8fafc",
          },
          content: { borderLeft: "1px solid #e2e8f0" },
        }}
      >
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex items-start gap-3">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Braces size={16} />
              </span>
              <div className="min-w-0">
                <p className="text-sm font-bold text-slate-800">Call the MCP endpoint</p>
                <p className="mt-1 text-xs font-semibold leading-5 text-slate-500">
                  Build a tools request without writing JSON. The playground sends a JSON-RPC
                  request directly to this Service.
                </p>
              </div>
            </div>
            <div className="mt-4 grid gap-3 sm:grid-cols-[1fr_1.4fr]">
              <TextInput
                label="Endpoint"
                value={targetURL}
                readOnly
                leftSection={<Send size={13} />}
              />
              <Select
                label="Protocol version"
                searchable
                data={Array.from(
                  new Set([
                    ...PROTOCOL_VERSIONS,
                    ...(config?.protocol?.versions ?? []),
                    protocolVersion,
                  ]),
                )}
                value={protocolVersion}
                onChange={(value) => setProtocolVersion(value || DEFAULT_PROTOCOL_VERSION)}
                allowDeselect={false}
              />
            </div>
            {sessionID && (
              <div className="mt-3 flex flex-wrap items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2">
                <Badge size="sm" variant="light" color="teal">
                  Initialized session
                </Badge>
                <span className="min-w-0 flex-1 truncate text-[0.68rem] font-semibold text-emerald-800">
                  {sessionID}
                </span>
                <Button
                  type="button"
                  size="compact-xs"
                  variant="subtle"
                  color="teal"
                  onClick={() => setSessionID("")}
                >
                  Clear session
                </Button>
              </div>
            )}
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <Select
              label="MCP method"
              searchable
              data={methodGroups}
              value={method}
              onChange={(value) => {
                setMethod((value as MCPMethod) || "tools/list");
                setError(undefined);
              }}
              allowDeselect={false}
            />

            {method === "initialize" && (
              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <TextInput
                  label="Client name"
                  value={clientName}
                  onChange={(event) => setClientName(event.currentTarget.value)}
                />
                <TextInput
                  label="Client version"
                  value={clientVersion}
                  onChange={(event) => setClientVersion(event.currentTarget.value)}
                />
                <Textarea
                  className="sm:col-span-2"
                  label="Capabilities (JSON object)"
                  description="Leave as {} when the dashboard does not need optional capabilities."
                  autosize
                  minRows={2}
                  value={capabilitiesJSON}
                  onChange={(event) => setCapabilitiesJSON(event.currentTarget.value)}
                />
              </div>
            )}

            {(method === "tools/list" ||
              method === "resources/list" ||
              method === "prompts/list") && (
              <TextInput
                className="mt-4"
                label="Cursor (optional)"
                description={`Use a cursor returned by a previous ${method} response to fetch the next page.`}
                placeholder="Leave empty for the first page"
                value={cursor}
                onChange={(event) => setCursor(event.currentTarget.value)}
              />
            )}

            {method === "tools/call" && (
              <div className="mt-4">
                <Autocomplete
                  label="Tool name"
                  description="The exact name returned by tools/list."
                  placeholder="e.g. search_documents"
                  data={availableTools}
                  limit={20}
                  value={toolName}
                  onChange={setToolName}
                  required
                />
                <div className="mt-4">
                  <ArgumentEditor
                    title="Tool arguments"
                    description="Add the key/value pairs expected by this tool."
                    rows={argumentsList}
                    onAdd={() => setArgumentsList((rows) => [...rows, newArgument()])}
                    onUpdate={updateArgument}
                    onRemove={(id) =>
                      setArgumentsList((rows) => rows.filter((row) => row.id !== id))
                    }
                  />
                </div>
              </div>
            )}

            {method === "resources/read" && (
              <Autocomplete
                className="mt-4"
                label="Resource URI"
                description="The exact URI returned by resources/list."
                placeholder="e.g. file:///reports/today"
                data={availableResources}
                limit={20}
                value={resourceURI}
                onChange={setResourceURI}
                required
              />
            )}

            {method === "prompts/get" && (
              <div className="mt-4">
                <Autocomplete
                  label="Prompt name"
                  description="The exact name returned by prompts/list."
                  placeholder="e.g. summarize_document"
                  data={availablePrompts}
                  limit={20}
                  value={promptName}
                  onChange={setPromptName}
                  required
                />
                <div className="mt-4">
                  <ArgumentEditor
                    title="Prompt arguments"
                    description="Add the string arguments expected by this prompt."
                    rows={argumentsList}
                    onAdd={() => setArgumentsList((rows) => [...rows, newArgument()])}
                    onUpdate={updateArgument}
                    onRemove={(id) =>
                      setArgumentsList((rows) => rows.filter((row) => row.id !== id))
                    }
                  />
                </div>
              </div>
            )}

            {(method === "ping" || method === "server/discover") && (
              <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 px-3 py-3 text-xs font-semibold text-slate-500">
                This method does not require additional parameters.
              </div>
            )}

            {error && (
              <div className="mt-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-xs font-semibold text-red-700" role="alert">
                <CircleAlert size={15} className="mt-0.5 shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <Group justify="flex-end" mt="md">
              <Button
                type="button"
                variant="default"
                loading={pending}
                leftSection={pending ? undefined : <Send size={13} />}
                onClick={() => void sendRequest()}
              >
                {pending ? "Sending…" : "Send request"}
              </Button>
            </Group>
          </div>

          {(request !== undefined || response !== undefined) && (
            <div className="grid gap-4 lg:grid-cols-2">
              {request !== undefined && (
                <div className="rounded-xl border border-slate-200 bg-white p-3 shadow-sm">
                  <div className="mb-2 flex items-center gap-2 text-xs font-bold uppercase tracking-[0.05em] text-slate-500">
                    <Send size={13} /> Request
                  </div>
                  <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-slate-50 p-3 text-[0.68rem] font-semibold leading-5 text-slate-700">
                    {formatResult(request)}
                  </pre>
                </div>
              )}
              {response !== undefined && (
                <div className="rounded-xl border border-emerald-200 bg-white p-3 shadow-sm">
                  <div className="mb-2 flex items-center gap-2 text-xs font-bold uppercase tracking-[0.05em] text-emerald-700">
                    <Check size={13} /> Response
                  </div>
                  <pre className="max-h-80 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-emerald-50/60 p-3 text-[0.68rem] font-semibold leading-5 text-slate-700">
                    {formatResult(response)}
                  </pre>
                </div>
              )}
            </div>
          )}

          {response !== undefined && (
            <Button
              type="button"
              variant="subtle"
              size="compact-xs"
              leftSection={<RefreshCw size={12} />}
              onClick={() => {
                setRequest(undefined);
                setResponse(undefined);
                setError(undefined);
              }}
            >
              Clear result
            </Button>
          )}
        </div>
      </Drawer>
    </>
  );
};

export default MCPPlayground;
