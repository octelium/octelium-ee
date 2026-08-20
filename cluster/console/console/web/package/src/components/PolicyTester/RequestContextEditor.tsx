import * as Core from "@/apis/corev1/corev1";
import { Struct } from "@/apis/google/protobuf/struct";
import {
  ActionIcon,
  Autocomplete,
  Button,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  TagsInput,
  Textarea,
  TextInput,
} from "@mantine/core";
import { Plus, X } from "lucide-react";
import * as React from "react";

export type RequestKind =
  | "http"
  | "mcp"
  | "llm"
  | "kubernetes"
  | "grpc"
  | "ssh"
  | "postgres"
  | "dns"
  | "socks5";

const fieldStyles = {
  label: {
    color: "#475569",
    fontSize: "0.68rem",
    fontWeight: 700,
    letterSpacing: "0.04em",
    marginBottom: "5px",
  },
  description: {
    color: "#94a3b8",
    fontSize: "0.65rem",
    fontWeight: 600,
  },
  input: {
    minHeight: "38px",
    borderColor: "#e2e8f0",
    borderRadius: "8px",
    color: "#1e293b",
    fontSize: "0.78rem",
    fontWeight: 600,
  },
  option: {
    fontSize: "0.78rem",
    fontWeight: 600,
  },
};

const protocols = [
  { label: "HTTP", value: "http" },
  { label: "MCP", value: "mcp" },
  { label: "LLM / AI", value: "llm" },
  { label: "Kubernetes", value: "kubernetes" },
  { label: "gRPC", value: "grpc" },
  { label: "SSH", value: "ssh" },
  { label: "PostgreSQL", value: "postgres" },
  { label: "DNS", value: "dns" },
  { label: "SOCKS5", value: "socks5" },
];

const mcpProtocolVersions = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];

const mcpKnownMethods = [
  "server/discover",
  "initialize",
  "ping",
  "tools/list",
  "tools/call",
  "prompts/list",
  "prompts/get",
  "resources/list",
  "resources/read",
  "resources/templates/list",
  "resources/subscribe",
  "resources/unsubscribe",
  "completion/complete",
  "subscriptions/listen",
  "elicitation/create",
  "roots/list",
  "sampling/createMessage",
  "logging/setLevel",
  "tasks/get",
  "tasks/update",
  "tasks/cancel",
  "notifications/initialized",
  "notifications/cancelled",
  "notifications/progress",
  "notifications/message",
  "notifications/roots/list_changed",
  "notifications/tools/list_changed",
  "notifications/prompts/list_changed",
  "notifications/resources/list_changed",
  "notifications/resources/updated",
  "notifications/subscriptions/acknowledged",
  "notifications/tasks",
];

const createHTTP = () =>
  Core.RequestContext_Request_HTTP.create({ size: -1 });

export const createRequestContext = (
  kind: RequestKind,
): Core.RequestContext_Request => {
  switch (kind) {
    case "http":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "http",
          http: createHTTP(),
        },
      });
    case "mcp":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "mcp",
          mcp: Core.RequestContext_Request_MCP.create(),
        },
      });
    case "llm":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "llm",
          llm: Core.RequestContext_Request_LLM.create({
            protocol: Core.Service_Spec_Config_LLM_Protocol.OPENAI,
            operation: Core.RequestContext_Request_LLM_Operation.CHAT_COMPLETIONS,
            estimateQuality:
              Core.RequestContext_Request_LLM_EstimateQuality.COMPLETE,
          }),
        },
      });
    case "kubernetes":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "kubernetes",
          kubernetes: Core.RequestContext_Request_Kubernetes.create(),
        },
      });
    case "grpc":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "grpc",
          grpc: Core.RequestContext_Request_GRPC.create(),
        },
      });
    case "ssh":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "ssh",
          ssh: Core.RequestContext_Request_SSH.create({
            type: {
              oneofKind: "connect",
              connect: Core.RequestContext_Request_SSH_Connect.create(),
            },
          }),
        },
      });
    case "postgres":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "postgres",
          postgres: Core.RequestContext_Request_Postgres.create({
            type: {
              oneofKind: "connect",
              connect:
                Core.RequestContext_Request_Postgres_Connect.create(),
            },
          }),
        },
      });
    case "dns":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "dns",
          dns: Core.RequestContext_Request_DNS.create(),
        },
      });
    case "socks5":
      return Core.RequestContext_Request.create({
        type: {
          oneofKind: "socks5",
          socks5: Core.RequestContext_Request_SOCKS5.create({
            type: {
              oneofKind: "connect",
              connect: Core.RequestContext_Request_SOCKS5_Connect.create(),
            },
          }),
        },
      });
  }
};

const MapEditor = ({
  label,
  description,
  value,
  onChange,
}: {
  label: string;
  description: string;
  value: Record<string, string>;
  onChange: (value: Record<string, string>) => void;
}) => {
  const id = React.useId().replace(/:/g, "");
  const nextID = React.useRef(0);
  const [entries, setEntries] = React.useState(() =>
    Object.entries(value).map(([key, itemValue]) => ({
      id: `${id}-${nextID.current++}`,
      key,
      value: itemValue,
    })),
  );

  const commit = (
    next: Array<{ id: string; key: string; value: string }>,
  ) => {
    setEntries(next);
    onChange(Object.fromEntries(next.map((entry) => [entry.key, entry.value])));
  };

  const updateEntry = (
    index: number,
    nextKey: string,
    nextValue: string,
  ) => {
    commit(
      entries.map((entry, entryIndex) =>
        entryIndex === index
          ? { ...entry, key: nextKey, value: nextValue }
          : entry,
      ),
    );
  };

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-3.5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-[0.68rem] font-bold text-slate-700">{label}</p>
          <p className="mt-0.5 text-[0.63rem] font-semibold text-slate-400">
            {description}
          </p>
        </div>
        <Button
          type="button"
          size="compact-xs"
          variant="default"
          leftSection={<Plus size={11} strokeWidth={2.5} />}
          onClick={() =>
            commit([
              ...entries,
              { id: `${id}-${nextID.current++}`, key: "", value: "" },
            ])
          }
        >
          Add
        </Button>
      </div>

      {entries.length === 0 ? (
        <p className="mt-3 rounded-lg bg-slate-50 px-3 py-2 text-[0.65rem] font-semibold text-slate-400">
          No entries added
        </p>
      ) : (
        <div className="mt-3 space-y-2">
          {entries.map((entry, index) => (
            <div
              key={entry.id}
              className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_30px] items-center gap-2"
            >
              <TextInput
                aria-label={`${label} key`}
                placeholder="Name"
                value={entry.key}
                styles={fieldStyles}
                onChange={(event) =>
                  updateEntry(index, event.target.value, entry.value)
                }
              />
              <TextInput
                aria-label={`${label} value`}
                placeholder="Value"
                value={entry.value}
                styles={fieldStyles}
                onChange={(event) =>
                  updateEntry(index, entry.key, event.target.value)
                }
              />
              <ActionIcon
                type="button"
                variant="subtle"
                color="red"
                aria-label={`Remove ${label} entry`}
                onClick={() =>
                  commit(entries.filter((_, entryIndex) => entryIndex !== index))
                }
              >
                <X size={13} />
              </ActionIcon>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

const StructEditor = ({
  value,
  onChange,
}: {
  value?: Struct;
  onChange: (value?: Struct) => void;
}) => {
  const [text, setText] = React.useState(() =>
    value ? JSON.stringify(Struct.toJson(value), null, 2) : "",
  );
  const [error, setError] = React.useState<string>();

  return (
    <Textarea
      label="Structured body"
      description="JSON representation used by structured-body policies"
      placeholder={'{\n  "role": "admin"\n}'}
      autosize
      minRows={4}
      maxRows={10}
      value={text}
      error={error}
      styles={fieldStyles}
      onChange={(event) => {
        const nextText = event.target.value;
        setText(nextText);

        if (!nextText.trim()) {
          setError(undefined);
          onChange(undefined);
          return;
        }

        try {
          const parsed = JSON.parse(nextText);
          if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
            setError("Enter a JSON object");
            return;
          }
          setError(undefined);
          onChange(Struct.fromJson(parsed));
        } catch {
          setError("Enter valid JSON");
        }
      }}
    />
  );
};

const HTTPFields = ({
  value,
  onChange,
}: {
  value: Core.RequestContext_Request_HTTP;
  onChange: (update: (value: Core.RequestContext_Request_HTTP) => void) => void;
}) => (
  <div className="space-y-4">
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <TextInput
        label="Method"
        placeholder="GET"
        value={value.method}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.method = event.target.value;
          })
        }
      />
      <TextInput
        label="Scheme"
        placeholder="https"
        value={value.scheme}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.scheme = event.target.value;
          })
        }
      />
      <TextInput
        label="Host"
        placeholder="api.example.com"
        value={value.host}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.host = event.target.value;
          })
        }
      />
      <Select
        label="Protocol"
        placeholder="Select protocol"
        clearable
        searchable
        value={value.protocol || null}
        data={["HTTP/1.0", "HTTP/1.1", "HTTP/2"]}
        styles={fieldStyles}
        onChange={(protocol) =>
          onChange((http) => {
            http.protocol = protocol ?? "";
          })
        }
      />
      <TextInput
        label="Path"
        placeholder="/apis/v1/sessions"
        value={value.path}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.path = event.target.value;
          })
        }
      />
      <TextInput
        label="URI"
        placeholder="/apis/v1/sessions?user=john"
        value={value.uri}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.uri = event.target.value;
          })
        }
      />
      <NumberInput
        label="Request size"
        description="Bytes; use -1 when unknown"
        placeholder="-1"
        min={-1}
        allowDecimal={false}
        value={Number(value.size)}
        styles={fieldStyles}
        onChange={(size) =>
          onChange((http) => {
            http.size = Number(size === "" ? -1 : size);
          })
        }
      />
    </div>

    <div className="grid gap-4 lg:grid-cols-2">
      <MapEditor
        label="Headers"
        description="Single-valued HTTP request headers"
        value={value.headers}
        onChange={(headers) =>
          onChange((http) => {
            http.headers = headers;
          })
        }
      />
      <MapEditor
        label="Query parameters"
        description="Single-valued query parameters"
        value={value.queryParams}
        onChange={(queryParams) =>
          onChange((http) => {
            http.queryParams = queryParams;
          })
        }
      />
    </div>

    <div className="grid gap-4 lg:grid-cols-2">
      <Textarea
        label="Raw body"
        description="UTF-8 request body"
        placeholder="Request body"
        autosize
        minRows={4}
        maxRows={10}
        value={new TextDecoder().decode(value.body)}
        styles={fieldStyles}
        onChange={(event) =>
          onChange((http) => {
            http.body = new TextEncoder().encode(event.target.value);
          })
        }
      />
      <StructEditor
        value={value.bodyMap}
        onChange={(bodyMap) =>
          onChange((http) => {
            http.bodyMap = bodyMap;
          })
        }
      />
    </div>
  </div>
);

const NestedHTTP = ({
  value,
  onSet,
  onRemove,
  onChange,
}: {
  value?: Core.RequestContext_Request_HTTP;
  onSet: () => void;
  onRemove: () => void;
  onChange: (update: (value: Core.RequestContext_Request_HTTP) => void) => void;
}) => (
  <div className="rounded-xl border border-slate-200 bg-white">
    <div className="flex items-center justify-between gap-3 px-3.5 py-3">
      <div>
        <p className="text-[0.68rem] font-bold text-slate-700">
          Underlying HTTP request
        </p>
        <p className="mt-0.5 text-[0.63rem] font-semibold text-slate-400">
          Add transport-level HTTP attributes when relevant
        </p>
      </div>
      <Button
        type="button"
        size="compact-xs"
        variant="default"
        leftSection={value ? <X size={11} /> : <Plus size={11} />}
        onClick={value ? onRemove : onSet}
      >
        {value ? "Remove" : "Add HTTP"}
      </Button>
    </div>
    {value && (
      <div className="border-t border-slate-100 p-3.5">
        <HTTPFields value={value} onChange={onChange} />
      </div>
    )}
  </div>
);

const MCPClientEditor = ({
  value,
  onSet,
  onRemove,
  onChange,
}: {
  value?: Core.RequestContext_Request_MCP_Client;
  onSet: () => void;
  onRemove: () => void;
  onChange: (
    update: (value: Core.RequestContext_Request_MCP_Client) => void,
  ) => void;
}) => (
  <div className="rounded-xl border border-slate-200 bg-white">
    <div className="flex items-center justify-between gap-3 px-3.5 py-3">
      <div>
        <p className="text-[0.68rem] font-bold text-slate-700">MCP client information</p>
        <p className="mt-0.5 text-[0.63rem] font-semibold text-slate-400">
          Self-reported client metadata, not an identity signal
        </p>
      </div>
      <Button
        type="button"
        size="compact-xs"
        variant="default"
        leftSection={value ? <X size={11} /> : <Plus size={11} />}
        onClick={value ? onRemove : onSet}
      >
        {value ? "Remove" : "Add client"}
      </Button>
    </div>
    {value && (
      <div className="grid gap-3 border-t border-slate-100 p-3.5 sm:grid-cols-3">
        {[
          ["Name", "name", "Example Client"],
          ["Version", "version", "1.0.0"],
          ["Title", "title", "Example MCP client"],
        ].map(([label, field, placeholder]) => (
          <TextInput
            key={field}
            label={label}
            placeholder={placeholder}
            value={String(value[field as keyof Core.RequestContext_Request_MCP_Client] ?? "")}
            styles={fieldStyles}
            onChange={(event) =>
              onChange((client) => {
                Object.assign(client, { [field]: event.target.value });
              })
            }
          />
        ))}
      </div>
    )}
  </div>
);

const MCPFields = ({
  value,
  onChange,
}: {
  value: Core.RequestContext_Request_MCP;
  onChange: (update: (value: Core.RequestContext_Request_MCP) => void) => void;
}) => (
  <div className="space-y-4">
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <Autocomplete
        label="Protocol version"
        description="Select a canonical version or enter a custom value"
        placeholder="2025-06-18"
        data={mcpProtocolVersions}
        value={value.protocolVersion}
        styles={fieldStyles}
        onChange={(protocolVersion) =>
          onChange((mcp) => {
            mcp.protocolVersion = protocolVersion;
          })
        }
      />
      <Autocomplete
        label="Method"
        description="Select a known JSON-RPC method or enter a custom value"
        placeholder="tools/call"
        data={mcpKnownMethods}
        value={value.method}
        styles={fieldStyles}
        onChange={(method) =>
          onChange((mcp) => {
            mcp.method = method;
          })
        }
      />
      {[
        ["Target name", "name", "search"],
        ["Request ID", "requestID", "42"],
        ["MCP session ID", "sessionID", "session-123"],
      ].map(([label, field, placeholder]) => (
        <TextInput
          key={field}
          label={label}
          placeholder={placeholder}
          value={String(value[field as keyof Core.RequestContext_Request_MCP] ?? "")}
          styles={fieldStyles}
          onChange={(event) =>
            onChange((mcp) => {
              Object.assign(mcp, { [field]: event.target.value });
            })
          }
        />
      ))}
      <Switch
        label="JSON-RPC notification"
        description="No response is expected for this request"
        checked={value.isNotification}
        styles={fieldStyles}
        onChange={(event) => {
          const checked = event.currentTarget.checked;
          onChange((mcp) => {
            mcp.isNotification = checked;
          });
        }}
      />
    </div>

    <TagsInput
      label="Capabilities"
      description="Self-reported MCP capabilities"
      placeholder="Add capability"
      value={value.capabilities}
      styles={fieldStyles}
      onChange={(capabilities) =>
        onChange((mcp) => {
          mcp.capabilities = capabilities;
        })
      }
    />

    <MCPClientEditor
      value={value.client}
      onSet={() =>
        onChange((mcp) => {
          mcp.client = Core.RequestContext_Request_MCP_Client.create();
        })
      }
      onRemove={() =>
        onChange((mcp) => {
          mcp.client = undefined;
        })
      }
      onChange={(update) =>
        onChange((mcp) => {
          if (mcp.client) update(mcp.client);
        })
      }
    />

    <NestedHTTP
      value={value.http}
      onSet={() =>
        onChange((mcp) => {
          mcp.http = createHTTP();
        })
      }
      onRemove={() =>
        onChange((mcp) => {
          mcp.http = undefined;
        })
      }
      onChange={(update) =>
        onChange((mcp) => {
          if (mcp.http) update(mcp.http);
        })
      }
    />
  </div>
);

const LLMFields = ({
  value,
  onChange,
}: {
  value: Core.RequestContext_Request_LLM;
  onChange: (update: (value: Core.RequestContext_Request_LLM) => void) => void;
}) => {
  const protocolData = [
    { label: "OpenAI", value: String(Core.Service_Spec_Config_LLM_Protocol.OPENAI) },
    { label: "Anthropic", value: String(Core.Service_Spec_Config_LLM_Protocol.ANTHROPIC) },
  ];
  const operationData = [
    ["Chat completions", Core.RequestContext_Request_LLM_Operation.CHAT_COMPLETIONS],
    ["Responses", Core.RequestContext_Request_LLM_Operation.RESPONSES],
    ["Completions", Core.RequestContext_Request_LLM_Operation.COMPLETIONS],
    ["Embeddings", Core.RequestContext_Request_LLM_Operation.EMBEDDINGS],
    ["Moderations", Core.RequestContext_Request_LLM_Operation.MODERATIONS],
    ["Models list", Core.RequestContext_Request_LLM_Operation.MODELS_LIST],
    ["Models get", Core.RequestContext_Request_LLM_Operation.MODELS_GET],
    ["Messages", Core.RequestContext_Request_LLM_Operation.MESSAGES],
    ["Count tokens", Core.RequestContext_Request_LLM_Operation.COUNT_TOKENS],
  ].map(([label, operation]) => ({ label: String(label), value: String(operation) }));
  const qualityData = [
    ["Complete", Core.RequestContext_Request_LLM_EstimateQuality.COMPLETE],
    ["Partial", Core.RequestContext_Request_LLM_EstimateQuality.PARTIAL],
    ["Unavailable", Core.RequestContext_Request_LLM_EstimateQuality.UNAVAILABLE],
  ].map(([label, quality]) => ({ label: String(label), value: String(quality) }));

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <Select
          label="Protocol"
          value={String(value.protocol)}
          data={protocolData}
          allowDeselect={false}
          styles={fieldStyles}
          onChange={(protocol) =>
            onChange((llm) => {
              llm.protocol = Number(protocol);
            })
          }
        />
        <Select
          label="Operation"
          value={String(value.operation)}
          data={operationData}
          allowDeselect={false}
          searchable
          styles={fieldStyles}
          onChange={(operation) =>
            onChange((llm) => {
              llm.operation = Number(operation);
            })
          }
        />
        <TextInput
          label="Model"
          placeholder="gpt-4o"
          value={value.model}
          styles={fieldStyles}
          onChange={(event) =>
            onChange((llm) => {
              llm.model = event.target.value;
            })
          }
        />
        <Switch
          label="Streaming response"
          checked={value.stream}
          styles={fieldStyles}
          onChange={(event) => {
            const checked = event.currentTarget.checked;
            onChange((llm) => {
              llm.stream = checked;
            });
          }}
        />
        <NumberInput
          label="Estimated input tokens"
          min={0}
          allowDecimal={false}
          value={value.estimatedInputTokens}
          styles={fieldStyles}
          onChange={(tokens) =>
            onChange((llm) => {
              llm.estimatedInputTokens = Number(tokens || 0);
            })
          }
        />
        <Select
          label="Estimate quality"
          value={String(value.estimateQuality)}
          data={qualityData}
          allowDeselect={false}
          styles={fieldStyles}
          onChange={(quality) =>
            onChange((llm) => {
              llm.estimateQuality = Number(quality);
            })
          }
        />
        <NumberInput
          label="Maximum output tokens"
          min={0}
          allowDecimal={false}
          value={value.maxOutputTokens}
          styles={fieldStyles}
          onChange={(tokens) =>
            onChange((llm) => {
              llm.maxOutputTokens = Number(tokens || 0);
            })
          }
        />
        <Switch
          label="Request has tools"
          checked={value.hasTools}
          styles={fieldStyles}
          onChange={(event) => {
            const checked = event.currentTarget.checked;
            onChange((llm) => {
              llm.hasTools = checked;
            });
          }}
        />
        <NumberInput
          label="Tool count"
          min={0}
          allowDecimal={false}
          value={value.toolCount}
          styles={fieldStyles}
          onChange={(count) =>
            onChange((llm) => {
              llm.toolCount = Number(count || 0);
            })
          }
        />
        <NumberInput
          label="Input item count"
          min={0}
          allowDecimal={false}
          value={value.inputItemCount}
          styles={fieldStyles}
          onChange={(count) =>
            onChange((llm) => {
              llm.inputItemCount = Number(count || 0);
            })
          }
        />
        <Switch
          label="Image input"
          checked={value.hasImageInput}
          styles={fieldStyles}
          onChange={(event) => {
            const checked = event.currentTarget.checked;
            onChange((llm) => {
              llm.hasImageInput = checked;
            });
          }}
        />
        <Switch
          label="Audio input"
          checked={value.hasAudioInput}
          styles={fieldStyles}
          onChange={(event) => {
            const checked = event.currentTarget.checked;
            onChange((llm) => {
              llm.hasAudioInput = checked;
            });
          }}
        />
      </div>

      <TagsInput
        label="Tool names"
        placeholder="Add tool name"
        value={value.toolNames}
        styles={fieldStyles}
        onChange={(toolNames) =>
          onChange((llm) => {
            llm.toolNames = toolNames;
          })
        }
      />

      <NestedHTTP
        value={value.http}
        onSet={() =>
          onChange((llm) => {
            llm.http = createHTTP();
          })
        }
        onRemove={() =>
          onChange((llm) => {
            llm.http = undefined;
          })
        }
        onChange={(update) =>
          onChange((llm) => {
            if (llm.http) update(llm.http);
          })
        }
      />
    </div>
  );
};

const RequestContextEditor = ({
  value,
  onChange,
}: {
  value: Core.RequestContext_Request;
  onChange: (
    update: (request: Core.RequestContext_Request) => void,
  ) => void;
}) => {
  const kind = (value.type.oneofKind ?? "http") as RequestKind;

  const changeKind = (nextKind: string) => {
    const next = createRequestContext(nextKind as RequestKind);
    next.ip = value.ip;
    onChange((request) => Object.assign(request, next));
  };

  return (
    <div>
      <div className="sm:hidden">
        <Select
          label="Protocol"
          value={kind}
          allowDeselect={false}
          data={protocols}
          styles={fieldStyles}
          onChange={(nextKind) => nextKind && changeKind(nextKind)}
        />
      </div>
      <div className="hidden overflow-x-auto sm:block">
        <SegmentedControl
          fullWidth
          value={kind}
          data={protocols}
          onChange={changeKind}
          className="min-w-[640px]"
          styles={{
            root: {
              backgroundColor: "#f1f5f9",
              border: "1px solid #e2e8f0",
              borderRadius: "10px",
              padding: "4px",
            },
            label: { fontSize: "0.7rem", fontWeight: 700 },
          }}
        />
      </div>

      <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50/50 p-4">
        <TextInput
          label="Client IP address"
          description="Source address used during policy evaluation"
          placeholder="192.0.2.10"
          value={value.ip}
          styles={fieldStyles}
          className="mb-4 max-w-md"
          onChange={(event) =>
            onChange((request) => {
              request.ip = event.target.value;
            })
          }
        />

        {value.type.oneofKind === "http" && (
          <HTTPFields
            value={value.type.http}
            onChange={(update) =>
              onChange((request) => {
                if (request.type.oneofKind === "http") {
                  update(request.type.http);
                }
              })
            }
          />
        )}

        {value.type.oneofKind === "mcp" && (
          <MCPFields
            value={value.type.mcp}
            onChange={(update) =>
              onChange((request) => {
                if (request.type.oneofKind === "mcp") {
                  update(request.type.mcp);
                }
              })
            }
          />
        )}

        {value.type.oneofKind === "llm" && (
          <LLMFields
            value={value.type.llm}
            onChange={(update) =>
              onChange((request) => {
                if (request.type.oneofKind === "llm") {
                  update(request.type.llm);
                }
              })
            }
          />
        )}

        {value.type.oneofKind === "kubernetes" && (
          <div className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {[
                ["Verb", "verb", "get"],
                ["API prefix", "apiPrefix", "/apis"],
                ["API group", "apiGroup", "apps"],
                ["API version", "apiVersion", "v1"],
                ["Namespace", "namespace", "kube-system"],
                ["Resource", "resource", "pods"],
                ["Sub-resource", "subresource", "portforward"],
                ["Name", "name", "pod-123456"],
              ].map(([label, field, placeholder]) => (
                <TextInput
                  key={field}
                  label={label}
                  placeholder={placeholder}
                  value={String(
                    value.type.oneofKind === "kubernetes"
                      ? value.type.kubernetes[
                          field as keyof Core.RequestContext_Request_Kubernetes
                        ] ?? ""
                      : "",
                  )}
                  styles={fieldStyles}
                  onChange={(event) =>
                    onChange((request) => {
                      if (request.type.oneofKind === "kubernetes") {
                        Object.assign(request.type.kubernetes, {
                          [field]: event.target.value,
                        });
                      }
                    })
                  }
                />
              ))}
            </div>
            <NestedHTTP
              value={value.type.kubernetes.http}
              onSet={() =>
                onChange((request) => {
                  if (request.type.oneofKind === "kubernetes") {
                    request.type.kubernetes.http = createHTTP();
                  }
                })
              }
              onRemove={() =>
                onChange((request) => {
                  if (request.type.oneofKind === "kubernetes") {
                    request.type.kubernetes.http = undefined;
                  }
                })
              }
              onChange={(update) =>
                onChange((request) => {
                  if (
                    request.type.oneofKind === "kubernetes" &&
                    request.type.kubernetes.http
                  ) {
                    update(request.type.kubernetes.http);
                  }
                })
              }
            />
          </div>
        )}

        {value.type.oneofKind === "grpc" && (
          <div className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-2">
              {[
                ["Method", "method", "GetUser"],
                ["Package", "package", "octelium.api.main.core.v1"],
                ["Service", "service", "MainService"],
                [
                  "Service full name",
                  "serviceFullName",
                  "octelium.api.main.core.v1.MainService",
                ],
              ].map(([label, field, placeholder]) => (
                <TextInput
                  key={field}
                  label={label}
                  placeholder={placeholder}
                  value={String(
                    value.type.oneofKind === "grpc"
                      ? value.type.grpc[
                          field as keyof Core.RequestContext_Request_GRPC
                        ] ?? ""
                      : "",
                  )}
                  styles={fieldStyles}
                  onChange={(event) =>
                    onChange((request) => {
                      if (request.type.oneofKind === "grpc") {
                        Object.assign(request.type.grpc, {
                          [field]: event.target.value,
                        });
                      }
                    })
                  }
                />
              ))}
            </div>
            <NestedHTTP
              value={value.type.grpc.http}
              onSet={() =>
                onChange((request) => {
                  if (request.type.oneofKind === "grpc") {
                    request.type.grpc.http = createHTTP();
                  }
                })
              }
              onRemove={() =>
                onChange((request) => {
                  if (request.type.oneofKind === "grpc") {
                    request.type.grpc.http = undefined;
                  }
                })
              }
              onChange={(update) =>
                onChange((request) => {
                  if (
                    request.type.oneofKind === "grpc" &&
                    request.type.grpc.http
                  ) {
                    update(request.type.grpc.http);
                  }
                })
              }
            />
          </div>
        )}

        {value.type.oneofKind === "ssh" &&
          value.type.ssh.type.oneofKind === "connect" && (
            <TextInput
              label="User"
              placeholder="root"
              value={value.type.ssh.type.connect.user}
              styles={fieldStyles}
              className="max-w-md"
              onChange={(event) =>
                onChange((request) => {
                  if (
                    request.type.oneofKind === "ssh" &&
                    request.type.ssh.type.oneofKind === "connect"
                  ) {
                    request.type.ssh.type.connect.user = event.target.value;
                  }
                })
              }
            />
          )}

        {value.type.oneofKind === "postgres" && (
          <div className="space-y-4">
            <SegmentedControl
              value={value.type.postgres.type.oneofKind ?? "connect"}
              data={[
                { label: "Connect", value: "connect" },
                { label: "Query", value: "query" },
                { label: "Parse", value: "parse" },
              ]}
              onChange={(postgresKind) =>
                onChange((request) => {
                  if (request.type.oneofKind !== "postgres") return;
                  request.type.postgres.type =
                    postgresKind === "connect"
                      ? {
                          oneofKind: "connect",
                          connect:
                            Core.RequestContext_Request_Postgres_Connect.create(),
                        }
                      : postgresKind === "query"
                        ? {
                            oneofKind: "query",
                            query:
                              Core.RequestContext_Request_Postgres_Query.create(),
                          }
                        : {
                            oneofKind: "parse",
                            parse:
                              Core.RequestContext_Request_Postgres_Parse.create(),
                          };
                })
              }
              styles={{ label: { fontSize: "0.7rem", fontWeight: 700 } }}
            />

            {value.type.postgres.type.oneofKind === "connect" && (
              <div className="grid gap-3 sm:grid-cols-3">
                {[
                  ["User", "user", "postgres"],
                  ["Database", "database", "app"],
                  ["Application name", "applicationName", "psql"],
                ].map(([label, field, placeholder]) => (
                  <TextInput
                    key={field}
                    label={label}
                    placeholder={placeholder}
                    value={String(
                      value.type.oneofKind === "postgres" &&
                        value.type.postgres.type.oneofKind === "connect"
                        ? value.type.postgres.type.connect[
                            field as keyof Core.RequestContext_Request_Postgres_Connect
                          ] ?? ""
                        : "",
                    )}
                    styles={fieldStyles}
                    onChange={(event) =>
                      onChange((request) => {
                        if (
                          request.type.oneofKind === "postgres" &&
                          request.type.postgres.type.oneofKind === "connect"
                        ) {
                          Object.assign(request.type.postgres.type.connect, {
                            [field]: event.target.value,
                          });
                        }
                      })
                    }
                  />
                ))}
              </div>
            )}
            {value.type.postgres.type.oneofKind === "query" && (
              <Textarea
                label="Query"
                placeholder="SELECT * FROM users"
                autosize
                minRows={3}
                value={value.type.postgres.type.query.query}
                styles={fieldStyles}
                onChange={(event) =>
                  onChange((request) => {
                    if (
                      request.type.oneofKind === "postgres" &&
                      request.type.postgres.type.oneofKind === "query"
                    ) {
                      request.type.postgres.type.query.query =
                        event.target.value;
                    }
                  })
                }
              />
            )}
            {value.type.postgres.type.oneofKind === "parse" && (
              <div className="grid gap-3 sm:grid-cols-[minmax(0,220px)_minmax(0,1fr)]">
                <TextInput
                  label="Statement name"
                  placeholder="find-user"
                  value={value.type.postgres.type.parse.name}
                  styles={fieldStyles}
                  onChange={(event) =>
                    onChange((request) => {
                      if (
                        request.type.oneofKind === "postgres" &&
                        request.type.postgres.type.oneofKind === "parse"
                      ) {
                        request.type.postgres.type.parse.name =
                          event.target.value;
                      }
                    })
                  }
                />
                <Textarea
                  label="Query"
                  placeholder="SELECT * FROM users WHERE id = $1"
                  autosize
                  minRows={3}
                  value={value.type.postgres.type.parse.query}
                  styles={fieldStyles}
                  onChange={(event) =>
                    onChange((request) => {
                      if (
                        request.type.oneofKind === "postgres" &&
                        request.type.postgres.type.oneofKind === "parse"
                      ) {
                        request.type.postgres.type.parse.query =
                          event.target.value;
                      }
                    })
                  }
                />
              </div>
            )}
          </div>
        )}

        {value.type.oneofKind === "dns" && (
          <div className="grid gap-3 sm:grid-cols-2">
            <TextInput
              label="DNS name"
              placeholder="api.example.com"
              value={value.type.dns.name}
              styles={fieldStyles}
              onChange={(event) =>
                onChange((request) => {
                  if (request.type.oneofKind === "dns") {
                    request.type.dns.name = event.target.value;
                  }
                })
              }
            />
            <NumberInput
              label="Record type ID"
              description="DNS numeric record type, such as 1 for A"
              placeholder="1"
              min={0}
              allowDecimal={false}
              value={value.type.dns.typeID}
              styles={fieldStyles}
              onChange={(typeID) =>
                onChange((request) => {
                  if (request.type.oneofKind === "dns") {
                    request.type.dns.typeID = Number(typeID || 0);
                  }
                })
              }
            />
          </div>
        )}

        {value.type.oneofKind === "socks5" &&
          value.type.socks5.type.oneofKind === "connect" && (
            <div className="grid gap-3 sm:grid-cols-3">
              <TextInput
                label="Host"
                placeholder="db.internal.example"
                value={value.type.socks5.type.connect.host}
                styles={fieldStyles}
                onChange={(event) =>
                  onChange((request) => {
                    if (
                      request.type.oneofKind === "socks5" &&
                      request.type.socks5.type.oneofKind === "connect"
                    ) {
                      request.type.socks5.type.connect.host = event.target.value;
                    }
                  })
                }
              />
              <NumberInput
                label="Port"
                placeholder="443"
                min={0}
                max={65535}
                allowDecimal={false}
                value={value.type.socks5.type.connect.port}
                styles={fieldStyles}
                onChange={(port) =>
                  onChange((request) => {
                    if (
                      request.type.oneofKind === "socks5" &&
                      request.type.socks5.type.oneofKind === "connect"
                    ) {
                      request.type.socks5.type.connect.port = Number(port || 0);
                    }
                  })
                }
              />
              <Select
                label="Address type"
                value={String(value.type.socks5.type.connect.addressType)}
                allowDeselect={false}
                data={[
                  { label: "Unspecified", value: "0" },
                  { label: "IPv4", value: "1" },
                  { label: "Domain", value: "2" },
                  { label: "IPv6", value: "3" },
                ]}
                styles={fieldStyles}
                onChange={(addressType) =>
                  onChange((request) => {
                    if (
                      request.type.oneofKind === "socks5" &&
                      request.type.socks5.type.oneofKind === "connect"
                    ) {
                      request.type.socks5.type.connect.addressType = Number(
                        addressType,
                      );
                    }
                  })
                }
              />
            </div>
          )}
      </div>
    </div>
  );
};

export default RequestContextEditor;
