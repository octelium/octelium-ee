import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import {
  CloseButton,
  Group,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  TagsInput,
  TextInput,
} from "@mantine/core";

import Cond from "@/components/Condition";
import DurationPicker from "@/components/DurationPicker";
import Editor from "@/components/Editor";
import ItemMessage from "@/components/ItemMessage";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import TextAreaCustom from "@/components/TextAreaCustom";
import { strToNum } from "@/utils/convert";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

type SegmentedTabsContextValue = {
  value: string | null;
  onChange: (value: string) => void;
};

const SegmentedTabsContext = React.createContext<SegmentedTabsContextValue>({
  value: null,
  onChange: () => {},
});

const SegmentedTabsRoot = (props: {
  value?: string | null;
  defaultValue?: string | null;
  onChange?: (value: string | null) => void;
  className?: string;
  children: React.ReactNode;
}) => {
  const [internalValue, setInternalValue] = React.useState(
    props.defaultValue ?? null,
  );
  const value = props.value ?? internalValue;
  const onChange = (next: string) => {
    if (props.value === undefined) setInternalValue(next);
    props.onChange?.(next);
  };

  return (
    <SegmentedTabsContext.Provider value={{ value, onChange }}>
      <div className={props.className}>{props.children}</div>
    </SegmentedTabsContext.Provider>
  );
};

const SegmentedTabsTab = (_props: {
  value: string;
  children: React.ReactNode;
}) => null;

const SegmentedTabsList = (props: {
  className?: string;
  children: React.ReactNode;
}) => {
  const { value, onChange } = React.useContext(SegmentedTabsContext);
  const data = React.Children.toArray(props.children)
    .filter(React.isValidElement)
    .map((child) => {
      const tab = child as React.ReactElement<{
        value: string;
        children: React.ReactNode;
      }>;
      return { value: tab.props.value, label: tab.props.children };
    });

  return (
    <div className={`w-full ${props.className ?? ""}`}>
      <SegmentedControl
        size="sm"
        value={value ?? data[0]?.value}
        onChange={onChange}
        data={data}
        fullWidth
        className="w-full"
      />
    </div>
  );
};

const SegmentedTabsPanel = (props: {
  value: string;
  children: React.ReactNode;
}) => {
  const { value } = React.useContext(SegmentedTabsContext);
  return value === props.value ? <>{props.children}</> : null;
};

const Tabs = Object.assign(SegmentedTabsRoot, {
  List: SegmentedTabsList,
  Tab: SegmentedTabsTab,
  Panel: SegmentedTabsPanel,
});

const ContainerProbe = (props: {
  title: string;
  probe?: CoreP.Service_Spec_Config_Upstream_Container_Probe;
  onUnset: () => void;
  onSet: () => void;
  onChange: () => void;
}) => {
  const { title, probe, onUnset, onSet, onChange } = props;

  return (
    <EditItem
      title={title}
      description="Set the container health probe"
      onUnset={onUnset}
      obj={probe}
      onSet={onSet}
    >
      {probe && (
        <div>
          <Tabs
            className="mb-4"
            value={probe.type.oneofKind}
            onChange={(v) => {
              match(v)
                .with("httpGet", () => {
                  probe.type = {
                    oneofKind: "httpGet",
                    httpGet:
                      CoreP.Service_Spec_Config_Upstream_Container_Probe_HTTPGet.create(),
                  };
                })
                .with("tcpSocket", () => {
                  probe.type = {
                    oneofKind: "tcpSocket",
                    tcpSocket:
                      CoreP.Service_Spec_Config_Upstream_Container_Probe_TCPSocket.create(),
                  };
                })
                .with("grpc", () => {
                  probe.type = {
                    oneofKind: "grpc",
                    grpc: CoreP.Service_Spec_Config_Upstream_Container_Probe_GRPC.create(),
                  };
                })
                .otherwise(() => {});
              onChange();
            }}
          >
            <Tabs.List>
              <Tabs.Tab value="httpGet">HTTP Get</Tabs.Tab>
              <Tabs.Tab value="tcpSocket">TCP Socket</Tabs.Tab>
              <Tabs.Tab value="grpc">gRPC</Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="httpGet">
              {match(probe.type)
                .when(
                  (x) => x.oneofKind === "httpGet",
                  (httpGet) => (
                    <Group grow>
                      <TextInput
                        label="Path"
                        description="HTTP path requested to check the container's health."
                        placeholder="/healthz"
                        value={httpGet.httpGet.path}
                        onChange={(v) => {
                          httpGet.httpGet.path = v.target.value;
                          onChange();
                        }}
                      />
                      <NumberInput
                        label="Port"
                        description="Port on which the container health endpoint listens."
                        min={0}
                        max={65535}
                        value={httpGet.httpGet.port}
                        onChange={(v) => {
                          httpGet.httpGet.port = strToNum(v);
                          onChange();
                        }}
                      />
                    </Group>
                  ),
                )
                .otherwise(() => (
                  <></>
                ))}
            </Tabs.Panel>

            <Tabs.Panel value="tcpSocket">
              {match(probe.type)
                .when(
                  (x) => x.oneofKind === "tcpSocket",
                  (tcpSocket) => (
                    <NumberInput
                      label="Port"
                      description="Port checked by the TCP socket probe."
                      min={0}
                      max={65535}
                      value={tcpSocket.tcpSocket.port}
                      onChange={(v) => {
                        tcpSocket.tcpSocket.port = strToNum(v);
                        onChange();
                      }}
                    />
                  ),
                )
                .otherwise(() => (
                  <></>
                ))}
            </Tabs.Panel>

            <Tabs.Panel value="grpc">
              {match(probe.type)
                .when(
                  (x) => x.oneofKind === "grpc",
                  (grpc) => (
                    <NumberInput
                      label="Port"
                      description="Port checked by the gRPC health probe."
                      min={0}
                      max={65535}
                      value={grpc.grpc.port}
                      onChange={(v) => {
                        grpc.grpc.port = strToNum(v);
                        onChange();
                      }}
                    />
                  ),
                )
                .otherwise(() => (
                  <></>
                ))}
            </Tabs.Panel>
          </Tabs>

          <Group grow>
            <NumberInput
              label="Initial delay (s)"
              description="Delay before the first health check runs."
              min={0}
              value={probe.initialDelaySeconds}
              onChange={(v) => {
                probe.initialDelaySeconds = strToNum(v);
                onChange();
              }}
            />
            <NumberInput
              label="Timeout (s)"
              description="Maximum time allowed for each health check."
              min={0}
              value={probe.timeoutSeconds}
              onChange={(v) => {
                probe.timeoutSeconds = strToNum(v);
                onChange();
              }}
            />
            <NumberInput
              label="Period (s)"
              description="Interval between consecutive health checks."
              min={0}
              value={probe.periodSeconds}
              onChange={(v) => {
                probe.periodSeconds = strToNum(v);
                onChange();
              }}
            />
            <NumberInput
              label="Success threshold"
              description="Consecutive successes required to become healthy."
              min={0}
              value={probe.successThreshold}
              onChange={(v) => {
                probe.successThreshold = strToNum(v);
                onChange();
              }}
            />
            <NumberInput
              label="Failure threshold"
              description="Consecutive failures required to become unhealthy."
              min={0}
              value={probe.failureThreshold}
              onChange={(v) => {
                probe.failureThreshold = strToNum(v);
                onChange();
              }}
            />
          </Group>
        </div>
      )}
    </EditItem>
  );
};

const createDirectResponseBody = () =>
  CoreP.Service_Spec_Config_HTTP_Plugin_Direct_Body.create({
    type: { oneofKind: "inline", inline: "" },
  });

const createConfigTypeForMode = (
  mode?: CoreP.Service_Spec_Mode,
): CoreP.Service_Spec_Config["type"] => {
  switch (mode) {
    case CoreP.Service_Spec_Mode.HTTP:
    case CoreP.Service_Spec_Mode.WEB:
    case CoreP.Service_Spec_Mode.GRPC:
      return {
        oneofKind: "http",
        http: CoreP.Service_Spec_Config_HTTP.create(),
      };
    case CoreP.Service_Spec_Mode.SSH:
      return {
        oneofKind: "ssh",
        ssh: CoreP.Service_Spec_Config_SSH.create({
          auth: {
            type: {
              oneofKind: "password",
              password: {
                type: { oneofKind: "fromSecret", fromSecret: "" },
              },
            },
          },
        }),
      };
    case CoreP.Service_Spec_Mode.POSTGRES:
      return {
        oneofKind: "postgres",
        postgres: CoreP.Service_Spec_Config_Postgres.create({
          auth: {
            type: {
              oneofKind: "password",
              password: {
                type: { oneofKind: "fromSecret", fromSecret: "" },
              },
            },
          },
        }),
      };
    case CoreP.Service_Spec_Mode.MYSQL:
      return {
        oneofKind: "mysql",
        mysql: CoreP.Service_Spec_Config_MySQL.create({
          auth: {
            type: {
              oneofKind: "password",
              password: {
                type: { oneofKind: "fromSecret", fromSecret: "" },
              },
            },
          },
        }),
      };
    case CoreP.Service_Spec_Mode.KUBERNETES:
      return {
        oneofKind: "kubernetes",
        kubernetes: CoreP.Service_Spec_Config_Kubernetes.create({
          type: {
            oneofKind: "kubeconfig",
            kubeconfig: CoreP.Service_Spec_Config_Kubernetes_Kubeconfig.create({
              type: { oneofKind: "fromSecret", fromSecret: "" },
            }),
          },
        }),
      };
    case CoreP.Service_Spec_Mode.SOCKS5:
      return {
        oneofKind: "socks5",
        socks5: CoreP.Service_Spec_Config_SOCKS5.create({
          auth: { type: { oneofKind: "noAuth", noAuth: true } },
        }),
      };
    case CoreP.Service_Spec_Mode.RDP_WEB:
      return {
        oneofKind: "rdp",
        rdp: CoreP.Service_Spec_Config_RDP.create({
          auth: {
            password: {
              type: { oneofKind: "fromSecret", fromSecret: "" },
            },
          },
        }),
      };
    case CoreP.Service_Spec_Mode.MCP:
      return {
        oneofKind: "mcp",
        mcp: CoreP.Service_Spec_Config_MCP.create(),
      };
    case CoreP.Service_Spec_Mode.LLM:
      return {
        oneofKind: "llm",
        llm: CoreP.Service_Spec_Config_LLM.create(),
      };
    default:
      return { oneofKind: undefined };
  }
};

const configTypeForMode = (mode?: CoreP.Service_Spec_Mode) =>
  createConfigTypeForMode(mode).oneofKind;

const cloneConfigForMode = (
  item: CoreP.Service_Spec_Config,
  mode?: CoreP.Service_Spec_Mode,
) => {
  const ret = CoreP.Service_Spec_Config.clone(item);
  const expectedType = configTypeForMode(mode);
  if (ret.type.oneofKind !== expectedType) {
    ret.type = createConfigTypeForMode(mode);
  }
  if (ret.type.oneofKind === "http") {
    ret.type.http.plugins.forEach((plugin) => {
      if (plugin.type.oneofKind === undefined) {
        plugin.type = {
          oneofKind: "direct",
          direct: CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create({
            body: createDirectResponseBody(),
          }),
        };
      } else if (
        plugin.type.oneofKind === "direct" &&
        !plugin.type.direct.body
      ) {
        plugin.type.direct.body = createDirectResponseBody();
      }
    });
  }
  return ret;
};

const configTypeTitle = (
  type: CoreP.Service_Spec_Config["type"]["oneofKind"],
) => {
  switch (type) {
    case "http":
      return "HTTP-specific configuration";
    case "kubernetes":
      return "Kubernetes-specific configuration";
    case "ssh":
      return "SSH-specific configuration";
    case "postgres":
      return "PostgreSQL-specific configuration";
    case "mysql":
      return "MySQL-specific configuration";
    case "socks5":
      return "SOCKS5-specific configuration";
    case "rdp":
      return "RDP-specific configuration";
    case "mcp":
      return "MCP-specific configuration";
    case "llm":
      return "LLM/AI-specific configuration";
    default:
      return "Mode-specific configuration";
  }
};

type GatewayConfig =
  | CoreP.Service_Spec_Config_MCP
  | CoreP.Service_Spec_Config_LLM;

type GatewayPluginKind = "mcp" | "llm";

const StringListEditor = (props: {
  title: string;
  values: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  description?: string;
}) => (
  <ItemMessage
    title={props.title}
    obj={props.values}
    isList
    onSet={() => props.onChange([""])}
    onAddListItem={() => props.onChange([...props.values, ""])}
  >
    {props.values.map((value, index) => (
      <div className="mb-3 flex w-full items-center" key={`${props.title}-${index}`}>
        <CloseButton
          size="sm"
          variant="subtle"
          className="mr-2"
          aria-label={`Remove ${props.title} item ${index + 1}`}
          onClick={() =>
            props.onChange(props.values.filter((_, itemIndex) => itemIndex !== index))
          }
        />
        <TextInput
          className="flex-1"
          label={`${props.title} ${index + 1}`}
          description={props.description}
          placeholder={props.placeholder}
          value={value}
          onChange={(event) => {
            const next = [...props.values];
            next[index] = event.target.value;
            props.onChange(next);
          }}
        />
      </div>
    ))}
  </ItemMessage>
);

const CORSConfigEditor = (props: {
  cors?: CoreP.Service_Spec_Config_HTTP_CORS;
  onChange: (cors: CoreP.Service_Spec_Config_HTTP_CORS | undefined) => void;
}) => (
  <EditItem
    title="CORS"
    description="Allow browser applications from selected origins to call this Service"
    obj={props.cors}
    onUnset={() => props.onChange(undefined)}
    onSet={() =>
      props.onChange(CoreP.Service_Spec_Config_HTTP_CORS.create())
    }
  >
    {props.cors && (
      <div className="space-y-3">
        <Group grow>
          <TextInput
            label="Allow methods"
            description="Methods sent in the access-control-allow-methods response header."
            placeholder="POST, GET, OPTIONS"
            value={props.cors.allowMethods}
            onChange={(event) => {
              props.cors!.allowMethods = event.currentTarget.value;
              props.onChange(props.cors);
            }}
          />
          <TextInput
            label="Allow headers"
            description="Request headers permitted by the CORS policy."
            placeholder="X-PINGOTHER, Content-Type"
            value={props.cors.allowHeaders}
            onChange={(event) => {
              props.cors!.allowHeaders = event.currentTarget.value;
              props.onChange(props.cors);
            }}
          />
          <Switch
            label="Allow credentials"
            description="Allow cookies and HTTP authentication"
            checked={props.cors.allowCredentials}
            onChange={(event) => {
              props.cors!.allowCredentials = event.currentTarget.checked;
              props.onChange(props.cors);
            }}
          />
        </Group>
        <Group grow>
          <TextInput
            label="Expose headers"
            description="Response headers browser clients are allowed to read."
            placeholder="Content-Encoding, Kuma-Revision"
            value={props.cors.exposeHeaders}
            onChange={(event) => {
              props.cors!.exposeHeaders = event.currentTarget.value;
              props.onChange(props.cors);
            }}
          />
          <TextInput
            label="Max age"
            description="How long browsers may cache the CORS preflight result."
            placeholder="86400"
            value={props.cors.maxAge}
            onChange={(event) => {
              props.cors!.maxAge = event.currentTarget.value;
              props.onChange(props.cors);
            }}
          />
          <Switch
            label="Allow Cluster Services"
            description="Trust origins from Services in this Cluster"
            checked={props.cors.allowClusterServices}
            onChange={(event) => {
              props.cors!.allowClusterServices = event.currentTarget.checked;
              props.onChange(props.cors);
            }}
          />
        </Group>
        <StringListEditor
          title="Allowed origin patterns"
          description="Origins allowed by the CORS policy; use an exact origin or * for all origins."
          values={props.cors.allowOriginStringMatch}
          onChange={(values) => {
            props.cors!.allowOriginStringMatch = values;
            props.onChange(props.cors);
          }}
          placeholder="https://example.com or *"
        />
      </div>
    )}
  </EditItem>
);

const GatewayAuthEditor = (props: {
  config: GatewayConfig;
  onChange: () => void;
}) => {
  const { config, onChange } = props;

  return (
    <EditItem
      title="Upstream authentication"
      description="Authenticate requests sent to the MCP or model provider upstream"
      obj={config.auth}
      onUnset={() => {
        config.auth = undefined;
        onChange();
      }}
      onSet={() => {
        config.auth = CoreP.Service_Spec_Config_HTTP_Auth.create({
          type: {
            oneofKind: "bearer",
            bearer: CoreP.Service_Spec_Config_HTTP_Auth_Bearer.create({
              type: { oneofKind: "fromSecret", fromSecret: "" },
            }),
          },
        });
        onChange();
      }}
    >
      {config.auth && (
        <>
          <Tabs
            className="mb-4"
            value={config.auth.type.oneofKind ?? "bearer"}
            onChange={(value) => {
              if (!value) return;
              const type =
                value === "basic"
                  ? {
                      oneofKind: "basic" as const,
                      basic: CoreP.Service_Spec_Config_HTTP_Auth_Basic.create({
                        password: {
                          type: { oneofKind: "fromSecret", fromSecret: "" },
                        },
                      }),
                    }
                  : value === "custom"
                    ? {
                        oneofKind: "custom" as const,
                        custom:
                          CoreP.Service_Spec_Config_HTTP_Auth_Custom.create({
                            value: {
                              type: { oneofKind: "fromSecret", fromSecret: "" },
                            },
                          }),
                      }
                    : value === "oauth2ClientCredentials"
                      ? {
                          oneofKind: "oauth2ClientCredentials" as const,
                          oauth2ClientCredentials:
                            CoreP.Service_Spec_Config_HTTP_Auth_OAuth2ClientCredentials.create(
                              {
                                clientSecret: {
                                  type: {
                                    oneofKind: "fromSecret",
                                    fromSecret: "",
                                  },
                                },
                              },
                            ),
                        }
                      : value === "sigv4"
                        ? {
                            oneofKind: "sigv4" as const,
                            sigv4:
                              CoreP.Service_Spec_Config_HTTP_Auth_Sigv4.create({
                                secretAccessKey: {
                                  type: {
                                    oneofKind: "fromSecret",
                                    fromSecret: "",
                                  },
                                },
                              }),
                          }
                        : {
                            oneofKind: "bearer" as const,
                            bearer:
                              CoreP.Service_Spec_Config_HTTP_Auth_Bearer.create({
                                type: {
                                  oneofKind: "fromSecret",
                                  fromSecret: "",
                                },
                              }),
                          };
              config.auth!.type = type;
              onChange();
            }}
          >
            <Tabs.List>
              <Tabs.Tab value="bearer">Bearer</Tabs.Tab>
              <Tabs.Tab value="basic">Basic</Tabs.Tab>
              <Tabs.Tab value="custom">Custom header</Tabs.Tab>
              <Tabs.Tab value="oauth2ClientCredentials">OAuth2</Tabs.Tab>
              <Tabs.Tab value="sigv4">AWS SigV4</Tabs.Tab>
            </Tabs.List>
          </Tabs>

          {match(config.auth.type)
            .with({ oneofKind: "bearer" }, (auth) => (
              <SelectResource
                api="core"
                kind="Secret"
                label="Bearer token Secret"
                description="Secret whose value is sent as the upstream bearer token."
                defaultValue={
                  auth.bearer.type.oneofKind === "fromSecret"
                    ? auth.bearer.type.fromSecret
                    : undefined
                }
                onChange={(value) => {
                  if (auth.bearer.type.oneofKind === "fromSecret") {
                    auth.bearer.type.fromSecret = value?.metadata?.name ?? "";
                  }
                  onChange();
                }}
              />
            ))
            .with({ oneofKind: "basic" }, (auth) => (
              <Group grow align="flex-start">
                <TextInput
                  label="Username"
                  description="Username sent to the upstream for HTTP Basic authentication."
                  value={auth.basic.username}
                  onChange={(event) => {
                    auth.basic.username = event.target.value;
                    onChange();
                  }}
                />
                {auth.basic.password?.type.oneofKind === "fromSecret" && (
                  <SelectResource
                    api="core"
                    kind="Secret"
                    label="Password Secret"
                    description="Secret whose value is sent as the Basic authentication password."
                    defaultValue={auth.basic.password.type.fromSecret}
                    onChange={(value) => {
                      if (auth.basic.password?.type.oneofKind === "fromSecret") {
                        auth.basic.password.type.fromSecret =
                          value?.metadata?.name ?? "";
                      }
                      onChange();
                    }}
                  />
                )}
              </Group>
            ))
            .with({ oneofKind: "custom" }, (auth) => (
              <Group grow align="flex-start">
                <TextInput
                  label="Header name"
                  description="Header that carries the custom upstream credential."
                  placeholder="X-API-Key"
                  value={auth.custom.header}
                  onChange={(event) => {
                    auth.custom.header = event.target.value;
                    onChange();
                  }}
                />
                {auth.custom.value?.type.oneofKind === "fromSecret" && (
                  <SelectResource
                    api="core"
                    kind="Secret"
                    label="Header value Secret"
                    description="Secret whose value is sent in the custom authentication header."
                    defaultValue={auth.custom.value.type.fromSecret}
                    onChange={(value) => {
                      if (auth.custom.value?.type.oneofKind === "fromSecret") {
                        auth.custom.value.type.fromSecret =
                          value?.metadata?.name ?? "";
                      }
                      onChange();
                    }}
                  />
                )}
              </Group>
            ))
            .with({ oneofKind: "oauth2ClientCredentials" }, (auth) => (
              <div>
                <Group grow align="flex-start">
                  <TextInput
                    label="Client ID"
                    description="OAuth2 client identifier sent to the token endpoint."
                    value={auth.oauth2ClientCredentials.clientID}
                    onChange={(event) => {
                      auth.oauth2ClientCredentials.clientID = event.target.value;
                      onChange();
                    }}
                  />
                  <TextInput
                    label="Token endpoint URL"
                    description="OAuth2 endpoint used to obtain an access token."
                    value={auth.oauth2ClientCredentials.tokenURL}
                    onChange={(event) => {
                      auth.oauth2ClientCredentials.tokenURL = event.target.value;
                      onChange();
                    }}
                  />
                </Group>
                {auth.oauth2ClientCredentials.clientSecret?.type.oneofKind ===
                  "fromSecret" && (
                  <SelectResource
                    api="core"
                    kind="Secret"
                    label="Client Secret"
                    description="Secret containing the OAuth2 client secret."
                    defaultValue={
                      auth.oauth2ClientCredentials.clientSecret.type.fromSecret
                    }
                    onChange={(value) => {
                      if (
                        auth.oauth2ClientCredentials.clientSecret?.type
                          .oneofKind === "fromSecret"
                      ) {
                        auth.oauth2ClientCredentials.clientSecret.type.fromSecret =
                          value?.metadata?.name ?? "";
                      }
                      onChange();
                    }}
                  />
                )}
                <StringListEditor
                  title="OAuth2 scopes"
                  description="Scopes requested from the OAuth2 provider."
                  values={auth.oauth2ClientCredentials.scopes}
                  onChange={(values) => {
                    auth.oauth2ClientCredentials.scopes = values;
                    onChange();
                  }}
                  placeholder="scope.read"
                />
              </div>
            ))
            .with({ oneofKind: "sigv4" }, (auth) => (
              <div>
                <Group grow align="flex-start">
                  <TextInput
                    label="Access key ID"
                    description="AWS access key ID used to sign upstream requests."
                    value={auth.sigv4.accessKeyID}
                    onChange={(event) => {
                      auth.sigv4.accessKeyID = event.target.value;
                      onChange();
                    }}
                  />
                  <TextInput
                    label="Region"
                    description="AWS region used when generating the SigV4 signature."
                    value={auth.sigv4.region}
                    onChange={(event) => {
                      auth.sigv4.region = event.target.value;
                      onChange();
                    }}
                  />
                  <TextInput
                    label="Service"
                    description="AWS service name used when generating the SigV4 signature."
                    value={auth.sigv4.service}
                    onChange={(event) => {
                      auth.sigv4.service = event.target.value;
                      onChange();
                    }}
                  />
                </Group>
                {auth.sigv4.secretAccessKey?.type.oneofKind === "fromSecret" && (
                  <SelectResource
                    api="core"
                    kind="Secret"
                    label="Secret access key"
                    description="Secret containing the AWS SigV4 secret access key."
                    defaultValue={auth.sigv4.secretAccessKey.type.fromSecret}
                    onChange={(value) => {
                      if (auth.sigv4.secretAccessKey?.type.oneofKind === "fromSecret") {
                        auth.sigv4.secretAccessKey.type.fromSecret =
                          value?.metadata?.name ?? "";
                      }
                      onChange();
                    }}
                  />
                )}
              </div>
            ))
            .otherwise(() => null)}
        </>
      )}
    </EditItem>
  );
};

const GatewayPathEditor = (props: {
  config: GatewayConfig;
  onChange: () => void;
}) => (
  <EditItem
    title="Upstream path"
    description="Rewrite the path before forwarding requests"
    obj={props.config.path}
    onUnset={() => {
      props.config.path = undefined;
      props.onChange();
    }}
    onSet={() => {
      props.config.path = CoreP.Service_Spec_Config_HTTP_Path.create();
      props.onChange();
    }}
  >
    {props.config.path && (
      <Group grow>
        <TextInput
          label="Add prefix"
          description="Prefix added to the request path before forwarding."
          placeholder="/api"
          value={props.config.path.addPrefix}
          onChange={(event) => {
            props.config.path!.addPrefix = event.target.value;
            props.onChange();
          }}
        />
        <TextInput
          label="Remove prefix"
          description="Prefix removed from the request path before forwarding."
          placeholder="/v1"
          value={props.config.path.removePrefix}
          onChange={(event) => {
            props.config.path!.removePrefix = event.target.value;
            props.onChange();
          }}
        />
      </Group>
    )}
  </EditItem>
);

const GatewayHeaderEditor = (props: {
  config: GatewayConfig;
  onChange: () => void;
}) => (
  <EditItem
    title="Headers"
    description="Control forwarded, added, and removed request headers"
    obj={props.config.header}
    onUnset={() => {
      props.config.header = undefined;
      props.onChange();
    }}
    onSet={() => {
      props.config.header = CoreP.Service_Spec_Config_HTTP_Header.create();
      props.onChange();
    }}
  >
    {props.config.header && (
      <Group grow>
        <Select
          label="Forwarded headers"
          description="How the downstream Forwarded header is handled."
          data={["DROP", "OBFUSCATE", "TRANSPARENT"]}
          value={
            CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode[
              props.config.header.forwardedMode
            ] ?? "DROP"
          }
          onChange={(value) => {
            if (!value) return;
            props.config.header!.forwardedMode =
              CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode[
                value as keyof typeof CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode
              ];
            props.onChange();
          }}
        />
        <Select
          label="Authorization header"
          description="Whether the downstream Authorization header is removed or passed upstream."
          data={["DELETE", "PASS"]}
          value={
            CoreP.Service_Spec_Config_HTTP_Header_AuthorizationMode[
              props.config.header.authorizationMode
            ] ?? "DELETE"
          }
          onChange={(value) => {
            if (!value) return;
            props.config.header!.authorizationMode =
              CoreP.Service_Spec_Config_HTTP_Header_AuthorizationMode[
                value as keyof typeof CoreP.Service_Spec_Config_HTTP_Header_AuthorizationMode
              ];
            props.onChange();
          }}
        />
      </Group>
    )}
  </EditItem>
);

const createGatewayPlugin = (): any =>
  CoreP.Service_Spec_Config_HTTP_Plugin.create({
    type: {
      oneofKind: "direct",
      direct: CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create(),
    },
  });

const createHTTPPlugin = () =>
  CoreP.Service_Spec_Config_HTTP_Plugin.create({
    type: {
      oneofKind: "direct",
      direct: CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create({
        body: createDirectResponseBody(),
      }),
    },
  });

const LuaScriptEditor = (props: {
  value: string;
  onChange: (value: string) => void;
}) => (
  <div className="space-y-2">
    <div>
      <p className="text-sm font-semibold text-slate-700">Lua script</p>
      <p className="text-xs font-medium text-slate-500">
        Write the inline Lua script executed by this plugin.
      </p>
    </div>
    <Editor
      mode="lua"
      value={props.value}
      onChange={props.onChange}
      minHeight="260px"
      maxHeight="520px"
    />
  </div>
);

const createValueEval = (kind: string): any =>
  kind === "eval"
    ? { oneofKind: "eval", eval: "" }
    : kind === "opa"
      ? { oneofKind: "opa", opa: "" }
      : { oneofKind: "value", value: "" };

const valueEvalText = (value: any): string =>
  match(value?.oneofKind)
    .with("value", () => value.value as string)
    .with("eval", () => value.eval as string)
    .with("opa", () => value.opa as string)
    .otherwise(() => "");

const ValueEvalEditor = (props: {
  label: string;
  description?: string;
  placeholder?: string;
  value: any;
  multiline?: boolean;
  onChange: (value: any) => void;
}) => {
  const kind = props.value?.oneofKind ?? "value";
  const setText = (text: string) =>
    props.onChange({ oneofKind: kind, [kind]: text });

  return (
    <div className="w-full space-y-2">
      <Select
        label="Source"
        description="Use a literal value, a CEL expression or a Rego policy."
        data={[
          { value: "value", label: "Literal value" },
          { value: "eval", label: "CEL expression" },
          { value: "opa", label: "Rego policy" },
        ]}
        value={kind}
        onChange={(next) => next && props.onChange(createValueEval(next))}
      />
      {props.multiline || kind === "opa" ? (
        <TextAreaCustom
          label={props.label}
          description={props.description}
          placeholder={props.placeholder}
          value={valueEvalText(props.value)}
          onChange={(text) => setText(text ?? "")}
        />
      ) : (
        <TextInput
          label={props.label}
          description={props.description}
          placeholder={props.placeholder}
          value={valueEvalText(props.value)}
          onChange={(event) => setText(event.target.value)}
        />
      )}
    </div>
  );
};

const EnumSelect = (props: {
  label: string;
  description?: string;
  values: string[];
  enumObj: any;
  value: number;
  onChange: (value: number) => void;
}) => (
  <Select
    label={props.label}
    description={props.description}
    data={props.values}
    value={props.enumObj[props.value] ?? props.values[0]}
    onChange={(value) => {
      if (!value) return;
      props.onChange(props.enumObj[value]);
    }}
  />
);

const EnumMultiSelect = (props: {
  label: string;
  description?: string;
  values: string[];
  enumObj: any;
  selected: number[];
  onChange: (values: number[]) => void;
}) => (
  <div className="w-full">
    <p className="text-sm font-semibold text-slate-700">{props.label}</p>
    {props.description && (
      <p className="mb-2 text-xs font-medium text-slate-500">
        {props.description}
      </p>
    )}
    <div className="flex flex-wrap gap-2">
      {props.values.map((value) => {
        const numeric = props.enumObj[value] as number;
        const checked = props.selected.includes(numeric);
        return (
          <button
            key={value}
            type="button"
            onClick={() =>
              props.onChange(
                checked
                  ? props.selected.filter((item) => item !== numeric)
                  : [...props.selected, numeric],
              )
            }
            className={twMerge(
              "rounded-lg border px-2.5 py-1.5 text-[0.7rem] font-bold transition-colors duration-300",
              checked
                ? "border-slate-800 bg-slate-800 text-white"
                : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:text-slate-700",
            )}
          >
            {value}
          </button>
        );
      })}
    </div>
  </div>
);

const guardrailPatternMatchKinds = ["regex", "type", "secrets"];

const createGuardrailPatternMatch = (kind: string): any =>
  kind === "type"
    ? {
        oneofKind: "type",
        type: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Type.EMAIL,
      }
    : kind === "secrets"
      ? {
          oneofKind: "secrets",
          secrets:
            CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Secrets.create(),
        }
      : { oneofKind: "regex", regex: "" };

const GuardrailPatternsEditor = (props: {
  patterns: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern[];
  onChange: () => void;
  onSet: (
    patterns: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern[],
  ) => void;
}) => (
  <ItemMessage
    title="Patterns"
    obj={props.patterns}
    isList
    onSet={() =>
      props.onSet([
        CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern.create({
          match: { oneofKind: "regex", regex: "" },
          action:
            CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Action.REDACT,
        }),
      ])
    }
    onAddListItem={() =>
      props.onSet([
        ...props.patterns,
        CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern.create({
          match: { oneofKind: "regex", regex: "" },
          action:
            CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Action.REDACT,
        }),
      ])
    }
  >
    {props.patterns.map((pattern, index) => (
      <EditItem
        key={`guardrail-pattern-${index}`}
        title={`Pattern ${index + 1}`}
        obj={pattern}
        onUnset={() =>
          props.onSet(props.patterns.filter((_, item) => item !== index))
        }
      >
        <Group grow align="flex-start">
          <Select
            label="Match"
            description="Match a regular expression, a built-in type or the Cluster's secret rules."
            data={guardrailPatternMatchKinds}
            value={pattern.match.oneofKind ?? "regex"}
            onChange={(value) => {
              if (!value) return;
              pattern.match = createGuardrailPatternMatch(value);
              props.onChange();
            }}
          />
          <EnumSelect
            label="Action"
            description="What happens to the content that the pattern matches."
            values={["DENY", "REDACT", "STRIP", "REPLACE"]}
            enumObj={
              CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Action
            }
            value={pattern.action}
            onChange={(value) => {
              pattern.action = value;
              props.onChange();
            }}
          />
        </Group>
        {pattern.match.oneofKind === "regex" && (
          <TextInput
            label="Regular expression"
            description="RE2 regular expression matched against the inspected content."
            placeholder="[0-9]{16}"
            value={pattern.match.regex}
            onChange={(event) => {
              if (pattern.match.oneofKind === "regex") {
                pattern.match.regex = event.target.value;
              }
              props.onChange();
            }}
          />
        )}
        {pattern.match.oneofKind === "type" && (
          <EnumSelect
            label="Built-in type"
            description="Built-in detector applied to the inspected content."
            values={["EMAIL", "CREDIT_CARD", "IBAN", "US_SSN"]}
            enumObj={
              CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Type
            }
            value={pattern.match.type}
            onChange={(value) => {
              if (pattern.match.oneofKind === "type") {
                pattern.match.type = value;
              }
              props.onChange();
            }}
          />
        )}
        {pattern.match.oneofKind === "secrets" && (
          <StringListEditor
            title="Excluded secret rules"
            description="Secret detection rules that this pattern does not apply."
            values={pattern.match.secrets.excludeRules}
            onChange={(values) => {
              if (pattern.match.oneofKind === "secrets") {
                pattern.match.secrets.excludeRules = values;
              }
              props.onChange();
            }}
          />
        )}
        {pattern.action ===
          CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Action
            .REPLACE && (
          <EditItem
            title="Replacement"
            description="Value written in place of the matched content"
            obj={pattern.replace}
            onUnset={() => {
              pattern.replace = undefined;
              props.onChange();
            }}
            onSet={() => {
              pattern.replace =
                CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Replace.create(
                  { type: { oneofKind: "value", value: "" } },
                );
              props.onChange();
            }}
          >
            {pattern.replace && (
              <ValueEvalEditor
                label="Replacement value"
                placeholder="[REDACTED]"
                value={pattern.replace.type}
                onChange={(value) => {
                  pattern.replace!.type = value;
                  props.onChange();
                }}
              />
            )}
          </EditItem>
        )}
      </EditItem>
    ))}
  </ItemMessage>
);

const RateLimitKeyEditor = (props: {
  keyValue?: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Key;
  onChange: (
    keyValue?: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Key,
  ) => void;
}) => (
  <EditItem
    title="Bucket key"
    description="How the requests are grouped into rate-limit buckets"
    obj={props.keyValue}
    onUnset={() => props.onChange(undefined)}
    onSet={() =>
      props.onChange(
        CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Key.create({
          type: { oneofKind: "perUser", perUser: true },
        }),
      )
    }
  >
    {props.keyValue && (
      <Group grow align="flex-start">
        <Select
          label="Key type"
          description="Bucket per User, per Session, or by a CEL expression."
          data={["perUser", "perSession", "eval"]}
          value={props.keyValue.type.oneofKind ?? "perUser"}
          onChange={(value) => {
            if (!value) return;
            props.keyValue!.type =
              value === "eval"
                ? { oneofKind: "eval", eval: "" }
                : value === "perSession"
                  ? { oneofKind: "perSession", perSession: true }
                  : { oneofKind: "perUser", perUser: true };
            props.onChange(props.keyValue);
          }}
        />
        {props.keyValue.type.oneofKind === "eval" && (
          <TextInput
            label="CEL expression"
            description="CEL expression that resolves to the bucket key."
            placeholder="ctx.user.metadata.name"
            value={props.keyValue.type.eval}
            onChange={(event) => {
              if (props.keyValue?.type.oneofKind === "eval") {
                props.keyValue.type.eval = event.target.value;
              }
              props.onChange(props.keyValue);
            }}
          />
        )}
      </Group>
    )}
  </EditItem>
);

const EmbeddingEditor = (props: {
  embedding?: CoreP.Service_Spec_Config_LLM_Embedding;
  onChange: (embedding?: CoreP.Service_Spec_Config_LLM_Embedding) => void;
}) => (
  <EditItem
    title="Embedding"
    description="Model used to embed the request in order to compare it semantically"
    obj={props.embedding}
    onUnset={() => props.onChange(undefined)}
    onSet={() =>
      props.onChange(
        CoreP.Service_Spec_Config_LLM_Embedding.create({
          source: CoreP.Service_Spec_Config_LLM_Embedding_Source.create({
            type: { oneofKind: "currentUpstream", currentUpstream: true },
          }),
        }),
      )
    }
  >
    {props.embedding && (
      <>
        <Group grow align="flex-start">
          <TextInput
            label="Embedding model"
            description="Model name requested from the embedding upstream."
            placeholder="text-embedding-3-small"
            value={props.embedding.model}
            onChange={(event) => {
              props.embedding!.model = event.target.value;
              props.onChange(props.embedding);
            }}
          />
          <NumberInput
            label="Dimensions"
            description="Vector dimensions requested from the embedding model."
            min={0}
            value={props.embedding.dimensions}
            onChange={(value) => {
              props.embedding!.dimensions = strToNum(value);
              props.onChange(props.embedding);
            }}
          />
        </Group>
        <EditItem
          title="Embedding source"
          description="Where the embeddings are generated"
          obj={props.embedding.source}
          onUnset={() => {
            props.embedding!.source = undefined;
            props.onChange(props.embedding);
          }}
          onSet={() => {
            props.embedding!.source =
              CoreP.Service_Spec_Config_LLM_Embedding_Source.create({
                type: { oneofKind: "currentUpstream", currentUpstream: true },
              });
            props.onChange(props.embedding);
          }}
        >
          {props.embedding.source && (
            <>
              <Select
                label="Source"
                description="Embed on the Service's own upstream or on a dedicated one."
                data={["currentUpstream", "upstream"]}
                value={props.embedding.source.type.oneofKind ?? "currentUpstream"}
                onChange={(value) => {
                  if (!value) return;
                  props.embedding!.source!.type =
                    value === "upstream"
                      ? {
                          oneofKind: "upstream",
                          upstream:
                            CoreP.Service_Spec_Config_LLM_Embedding_Source_Upstream.create(),
                        }
                      : { oneofKind: "currentUpstream", currentUpstream: true };
                  props.onChange(props.embedding);
                }}
              />
              {props.embedding.source.type.oneofKind === "upstream" && (
                <Group grow align="flex-start">
                  <TextInput
                    label="Upstream URL"
                    description="Base URL of the dedicated embedding upstream."
                    placeholder="https://api.openai.com"
                    value={props.embedding.source.type.upstream.url}
                    onChange={(event) => {
                      if (
                        props.embedding?.source?.type.oneofKind === "upstream"
                      ) {
                        props.embedding.source.type.upstream.url =
                          event.target.value;
                      }
                      props.onChange(props.embedding);
                    }}
                  />
                  <EnumSelect
                    label="Upstream protocol"
                    description="Inference protocol spoken by the embedding upstream."
                    values={llmProtocolNames}
                    enumObj={CoreP.Service_Spec_Config_LLM_Protocol}
                    value={props.embedding.source.type.upstream.protocol}
                    onChange={(value) => {
                      if (
                        props.embedding?.source?.type.oneofKind === "upstream"
                      ) {
                        props.embedding.source.type.upstream.protocol = value;
                      }
                      props.onChange(props.embedding);
                    }}
                  />
                </Group>
              )}
            </>
          )}
        </EditItem>
      </>
    )}
  </EditItem>
);

const llmProtocolNames = ["OPENAI", "ANTHROPIC", "GEMINI", "BEDROCK"];

const sharedPluginTypes = [
  { value: "direct", label: "Direct response" },
  { value: "rateLimit", label: "Rate limit" },
  { value: "lua", label: "Lua" },
  { value: "path", label: "Path" },
  { value: "jsonSchema", label: "JSON Schema" },
  { value: "extProc", label: "Ext Proc" },
];

const mcpPluginTypes = [
  ...sharedPluginTypes,
  { value: "guardrail", label: "Guardrail" },
];

const llmPluginTypes = [
  ...sharedPluginTypes,
  { value: "prompt", label: "Prompt" },
  { value: "tools", label: "Tools" },
  { value: "guardrail", label: "Guardrail" },
  { value: "model", label: "Model" },
  { value: "reasoning", label: "Reasoning" },
  { value: "tokenRateLimit", label: "Token rate limit" },
  { value: "semanticCache", label: "Semantic cache" },
  { value: "semanticRouter", label: "Semantic router" },
];

const createGatewayPluginType = (
  value: string,
  kind: GatewayPluginKind,
): any => {
  switch (value) {
    case "rateLimit":
      return {
        oneofKind: "rateLimit",
        rateLimit: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit.create(),
      };
    case "lua":
      return {
        oneofKind: "lua",
        lua: CoreP.Service_Spec_Config_HTTP_Plugin_Lua.create({
          type: { oneofKind: "inline", inline: "" },
        }),
      };
    case "path":
      return {
        oneofKind: "path",
        path: CoreP.Service_Spec_Config_HTTP_Plugin_Path.create(),
      };
    case "jsonSchema":
      return {
        oneofKind: "jsonSchema",
        jsonSchema: CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema.create({
          type: { oneofKind: "inline", inline: "" },
        }),
      };
    case "extProc":
      return {
        oneofKind: "extProc",
        extProc: CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc.create({
          type: { oneofKind: "address", address: "" },
        }),
      };
    case "guardrail":
      return kind === "mcp"
        ? {
            oneofKind: "guardrail",
            guardrail: CoreP.Service_Spec_Config_MCP_Plugin_Guardrail.create({
              leg: CoreP.Service_Spec_Config_MCP_Plugin_Guardrail_Leg.BOTH,
            }),
          }
        : {
            oneofKind: "guardrail",
            guardrail: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail.create({
              leg: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Leg.BOTH,
            }),
          };
    case "prompt":
      return {
        oneofKind: "prompt",
        prompt: CoreP.Service_Spec_Config_LLM_Plugin_Prompt.create({
          type: {
            oneofKind: "system",
            system: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System.create({
              mode: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System_Mode
                .PREPEND,
              content:
                CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create({
                  type: { oneofKind: "value", value: "" },
                }),
            }),
          },
        }),
      };
    case "tools":
      return {
        oneofKind: "tools",
        tools: CoreP.Service_Spec_Config_LLM_Plugin_Tools.create(),
      };
    case "model":
      return {
        oneofKind: "model",
        model: CoreP.Service_Spec_Config_LLM_Model.create({
          type: { oneofKind: "value", value: "" },
        }),
      };
    case "reasoning":
      return {
        oneofKind: "reasoning",
        reasoning: CoreP.Service_Spec_Config_LLM_Reasoning.create({
          type: {
            oneofKind: "level",
            level: CoreP.Service_Spec_Config_LLM_Reasoning_Level.MEDIUM,
          },
        }),
      };
    case "tokenRateLimit":
      return {
        oneofKind: "tokenRateLimit",
        tokenRateLimit:
          CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit.create({
            scope:
              CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope.TOTAL,
          }),
      };
    case "semanticCache":
      return {
        oneofKind: "semanticCache",
        semanticCache:
          CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache.create(),
      };
    case "semanticRouter":
      return {
        oneofKind: "semanticRouter",
        semanticRouter:
          CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter.create(),
      };
    default:
      return {
        oneofKind: "direct",
        direct: CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create(),
      };
  }
};

const SharedPluginTypeEditor = (props: {
  type: any;
  onChange: () => void;
}) => {
  const type = props.type;

  return (
    <>
      {type.oneofKind === "direct" && (
        <NumberInput
          label="Status code"
          description="HTTP status returned by the direct response."
          min={100}
          max={599}
          value={type.direct.statusCode}
          onChange={(value) => {
            type.direct.statusCode = strToNum(value);
            props.onChange();
          }}
        />
      )}
      {type.oneofKind === "rateLimit" && (
        <>
          <Group grow align="flex-start">
            <NumberInput
              label="Request limit"
              description="Maximum requests allowed during the configured rate-limit window."
              min={0}
              value={Number(type.rateLimit.limit)}
              onChange={(value) => {
                type.rateLimit.limit = strToNum(value);
                props.onChange();
              }}
            />
            <DurationPicker
              title="Window"
              description="Duration of the rate-limit window."
              value={type.rateLimit.window}
              onChange={(value) => {
                type.rateLimit.window = value;
                props.onChange();
              }}
            />
          </Group>
          <RateLimitKeyEditor
            keyValue={type.rateLimit.key}
            onChange={(value) => {
              type.rateLimit.key = value;
              props.onChange();
            }}
          />
        </>
      )}
      {type.oneofKind === "lua" && (
        <LuaScriptEditor
          value={
            type.lua.type.oneofKind === "inline" ? type.lua.type.inline : ""
          }
          onChange={(value) => {
            type.lua.type = { oneofKind: "inline", inline: value ?? "" };
            props.onChange();
          }}
        />
      )}
      {type.oneofKind === "path" && (
        <Group grow>
          <TextInput
            label="Add prefix"
            description="Prefix added to the request path before forwarding."
            value={type.path.addPrefix}
            onChange={(event) => {
              type.path.addPrefix = event.target.value;
              props.onChange();
            }}
          />
          <TextInput
            label="Remove prefix"
            description="Prefix removed from the request path before forwarding."
            value={type.path.removePrefix}
            onChange={(event) => {
              type.path.removePrefix = event.target.value;
              props.onChange();
            }}
          />
        </Group>
      )}
      {type.oneofKind === "jsonSchema" && (
        <div>
          <NumberInput
            label="Status code"
            description="HTTP status returned when JSON validation fails."
            min={100}
            max={599}
            value={type.jsonSchema.statusCode}
            onChange={(value) => {
              type.jsonSchema.statusCode = strToNum(value);
              props.onChange();
            }}
          />
          <TextAreaCustom
            description="Inline JSON Schema used to validate the request body."
            value={
              type.jsonSchema.type.oneofKind === "inline"
                ? type.jsonSchema.type.inline
                : ""
            }
            onChange={(value) => {
              type.jsonSchema.type = {
                oneofKind: "inline",
                inline: value ?? "",
              };
              props.onChange();
            }}
          />
        </div>
      )}
      {type.oneofKind === "extProc" && (
        <Group grow>
          <Select
            label="Endpoint type"
            description="Use a fixed address or a managed container for ext_proc."
            data={["address", "container"]}
            value={type.extProc.type.oneofKind ?? "address"}
            onChange={(value) => {
              if (!value) return;
              type.extProc.type =
                value === "container"
                  ? {
                      oneofKind: "container",
                      container:
                        CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_Container.create(),
                    }
                  : { oneofKind: "address", address: "" };
              props.onChange();
            }}
          />
          {type.extProc.type.oneofKind === "address" ? (
            <TextInput
              label="Address"
              description="Address of the Envoy ext_proc gRPC server."
              value={type.extProc.type.address}
              onChange={(event) => {
                if (type.extProc.type.oneofKind === "address") {
                  type.extProc.type.address = event.target.value;
                }
                props.onChange();
              }}
            />
          ) : (
            <TextInput
              label="Container image"
              description="Container image that serves the ext_proc gRPC endpoint."
              value={
                type.extProc.type.oneofKind === "container"
                  ? type.extProc.type.container.image
                  : ""
              }
              onChange={(event) => {
                if (type.extProc.type.oneofKind === "container") {
                  type.extProc.type.container.image = event.target.value;
                }
                props.onChange();
              }}
            />
          )}
        </Group>
      )}
    </>
  );
};

const MCPGuardrailEditor = (props: {
  guardrail: CoreP.Service_Spec_Config_MCP_Plugin_Guardrail;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <EnumSelect
        label="Leg"
        description="Which side of the exchange is inspected."
        values={["REQUEST", "RESPONSE", "BOTH"]}
        enumObj={CoreP.Service_Spec_Config_MCP_Plugin_Guardrail_Leg}
        value={props.guardrail.leg}
        onChange={(value) => {
          props.guardrail.leg = value;
          props.onChange();
        }}
      />
      <TextInput
        label="Deny message"
        description="Message returned to the downstream when the guardrail denies the request."
        placeholder="This request was blocked by policy."
        value={props.guardrail.denyMessage}
        onChange={(event) => {
          props.guardrail.denyMessage = event.target.value;
          props.onChange();
        }}
      />
    </Group>
    <EnumMultiSelect
      label="Scopes"
      description="Parts of the MCP exchange that the patterns are applied to."
      values={[
        "TOOL_ARGUMENTS",
        "TOOL_RESULTS",
        "RESOURCE_CONTENTS",
        "PROMPT_MESSAGES",
        "TOOL_DEFINITIONS",
        "ALL",
      ]}
      enumObj={CoreP.Service_Spec_Config_MCP_Plugin_Guardrail_Scope}
      selected={props.guardrail.scopes}
      onChange={(values) => {
        props.guardrail.scopes = values;
        props.onChange();
      }}
    />
    <GuardrailPatternsEditor
      patterns={props.guardrail.patterns}
      onChange={props.onChange}
      onSet={(patterns) => {
        props.guardrail.patterns = patterns;
        props.onChange();
      }}
    />
  </>
);

const LLMGuardrailEditor = (props: {
  guardrail: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <EnumSelect
        label="Leg"
        description="Which side of the exchange is inspected."
        values={["REQUEST", "RESPONSE", "BOTH"]}
        enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Leg}
        value={props.guardrail.leg}
        onChange={(value) => {
          props.guardrail.leg = value;
          props.onChange();
        }}
      />
      <TextInput
        label="Deny message"
        description="Message returned to the downstream when the guardrail denies the request."
        placeholder="This request was blocked by policy."
        value={props.guardrail.denyMessage}
        onChange={(event) => {
          props.guardrail.denyMessage = event.target.value;
          props.onChange();
        }}
      />
    </Group>
    <EnumMultiSelect
      label="Scopes"
      description="Parts of the inference exchange that the patterns are applied to."
      values={[
        "CONTENT",
        "INSTRUCTIONS",
        "TOOL_DEFINITIONS",
        "TOOL_RESULTS",
        "ALL",
      ]}
      enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Scope}
      selected={props.guardrail.scopes}
      onChange={(values) => {
        props.guardrail.scopes = values;
        props.onChange();
      }}
    />
    <GuardrailPatternsEditor
      patterns={props.guardrail.patterns}
      onChange={props.onChange}
      onSet={(patterns) => {
        props.guardrail.patterns = patterns;
        props.onChange();
      }}
    />
  </>
);

const LLMPromptEditor = (props: {
  prompt: CoreP.Service_Spec_Config_LLM_Plugin_Prompt;
  onChange: () => void;
}) => (
  <>
    <Select
      label="Target"
      description="Rewrite the system instructions or inject a conversation message."
      data={[
        { value: "system", label: "System instructions" },
        { value: "message", label: "Conversation message" },
      ]}
      value={props.prompt.type.oneofKind ?? "system"}
      onChange={(value) => {
        if (!value) return;
        props.prompt.type =
          value === "message"
            ? {
                oneofKind: "message",
                message:
                  CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message.create({
                    role: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Role
                      .USER,
                    position:
                      CoreP
                        .Service_Spec_Config_LLM_Plugin_Prompt_Message_Position
                        .PREPEND,
                    selector:
                      CoreP
                        .Service_Spec_Config_LLM_Plugin_Prompt_Message_Selector
                        .LAST,
                    content:
                      CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create(
                        { type: { oneofKind: "value", value: "" } },
                      ),
                  }),
              }
            : {
                oneofKind: "system",
                system:
                  CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System.create({
                    mode: CoreP
                      .Service_Spec_Config_LLM_Plugin_Prompt_System_Mode.PREPEND,
                    content:
                      CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create(
                        { type: { oneofKind: "value", value: "" } },
                      ),
                  }),
              };
        props.onChange();
      }}
    />
    {props.prompt.type.oneofKind === "system" && (
      <>
        <EnumSelect
          label="Mode"
          description="How the configured content is combined with the request's own system instructions."
          values={["PREPEND", "APPEND", "REPLACE", "STRIP", "REJECT"]}
          enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System_Mode}
          value={props.prompt.type.system.mode}
          onChange={(value) => {
            if (props.prompt.type.oneofKind === "system") {
              props.prompt.type.system.mode = value;
            }
            props.onChange();
          }}
        />
        {props.prompt.type.system.mode !==
          CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System_Mode.STRIP &&
          props.prompt.type.system.mode !==
            CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System_Mode.REJECT && (
            <ValueEvalEditor
              label="System content"
              description="Instructions written into the request."
              placeholder="You are a helpful assistant."
              multiline
              value={props.prompt.type.system.content?.type}
              onChange={(value) => {
                if (props.prompt.type.oneofKind === "system") {
                  props.prompt.type.system.content =
                    CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create({
                      type: value,
                    });
                }
                props.onChange();
              }}
            />
          )}
      </>
    )}
    {props.prompt.type.oneofKind === "message" && (
      <>
        <Group grow align="flex-start">
          <EnumSelect
            label="Role"
            description="Role of the message that the plugin writes."
            values={["USER", "ASSISTANT"]}
            enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Role}
            value={props.prompt.type.message.role}
            onChange={(value) => {
              if (props.prompt.type.oneofKind === "message") {
                props.prompt.type.message.role = value;
              }
              props.onChange();
            }}
          />
          <EnumSelect
            label="Position"
            description="Where the content is written relative to the selected message."
            values={["PREPEND", "APPEND", "NEW_BEFORE", "NEW_AFTER"]}
            enumObj={
              CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Position
            }
            value={props.prompt.type.message.position}
            onChange={(value) => {
              if (props.prompt.type.oneofKind === "message") {
                props.prompt.type.message.position = value;
              }
              props.onChange();
            }}
          />
          <EnumSelect
            label="Selector"
            description="Which of the matching messages the plugin acts on."
            values={["LAST", "FIRST", "ALL"]}
            enumObj={
              CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Selector
            }
            value={props.prompt.type.message.selector}
            onChange={(value) => {
              if (props.prompt.type.oneofKind === "message") {
                props.prompt.type.message.selector = value;
              }
              props.onChange();
            }}
          />
        </Group>
        <ValueEvalEditor
          label="Message content"
          description="Content written into the conversation."
          placeholder="Answer only from the provided context."
          multiline
          value={props.prompt.type.message.content?.type}
          onChange={(value) => {
            if (props.prompt.type.oneofKind === "message") {
              props.prompt.type.message.content =
                CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create({
                  type: value,
                });
            }
            props.onChange();
          }}
        />
      </>
    )}
  </>
);

const LLMToolsEditor = (props: {
  tools: CoreP.Service_Spec_Config_LLM_Plugin_Tools;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <EnumSelect
        label="Tool choice"
        description="Whether the request's own tool choice is preserved or overwritten."
        values={["PRESERVE", "NONE", "AUTO"]}
        enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Tools_Choice}
        value={props.tools.choice}
        onChange={(value) => {
          props.tools.choice = value;
          props.onChange();
        }}
      />
      <TextInput
        label="Deny message"
        description="Message returned when a filter denies the request."
        placeholder="This tool is not allowed."
        value={props.tools.denyMessage}
        onChange={(event) => {
          props.tools.denyMessage = event.target.value;
          props.onChange();
        }}
      />
    </Group>
    <ItemMessage
      title="Filters"
      obj={props.tools.filters}
      isList
      onSet={() => {
        props.tools.filters = [
          CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter.create({
            match: { oneofKind: "name", name: "" },
            decision:
              CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Decision.REMOVE,
          }),
        ];
        props.onChange();
      }}
      onAddListItem={() => {
        props.tools.filters.push(
          CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter.create({
            match: { oneofKind: "name", name: "" },
            decision:
              CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Decision.REMOVE,
          }),
        );
        props.onChange();
      }}
    >
      {props.tools.filters.map((filter, index) => (
        <EditItem
          key={`tool-filter-${index}`}
          title={`Filter ${index + 1}`}
          obj={filter}
          onUnset={() => {
            props.tools.filters.splice(index, 1);
            props.onChange();
          }}
        >
          <Group grow align="flex-start">
            <Select
              label="Match"
              description="Match the tool by its name or by its declared type."
              data={["name", "type"]}
              value={filter.match.oneofKind ?? "name"}
              onChange={(value) => {
                if (!value) return;
                filter.match =
                  value === "type"
                    ? { oneofKind: "type", type: "" }
                    : { oneofKind: "name", name: "" };
                props.onChange();
              }}
            />
            <TextInput
              label={filter.match.oneofKind === "type" ? "Tool type" : "Tool name"}
              description="Matched verbatim, or as a `*` suffixed prefix."
              placeholder={filter.match.oneofKind === "type" ? "function" : "exec_*"}
              value={
                filter.match.oneofKind === "type"
                  ? filter.match.type
                  : filter.match.oneofKind === "name"
                    ? filter.match.name
                    : ""
              }
              onChange={(event) => {
                if (filter.match.oneofKind === "type") {
                  filter.match.type = event.target.value;
                } else if (filter.match.oneofKind === "name") {
                  filter.match.name = event.target.value;
                }
                props.onChange();
              }}
            />
            <EnumSelect
              label="Decision"
              description="What happens to the tools that the filter matches."
              values={["ALLOW", "REMOVE", "DENY", "REPLACE"]}
              enumObj={
                CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Decision
              }
              value={filter.decision}
              onChange={(value) => {
                filter.decision = value;
                props.onChange();
              }}
            />
          </Group>
          {filter.decision ===
            CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Decision
              .REPLACE && (
            <EditItem
              title="Replacement"
              description="Tool definition written in place of the matched one"
              obj={filter.replace}
              onUnset={() => {
                filter.replace = undefined;
                props.onChange();
              }}
              onSet={() => {
                filter.replace =
                  CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Replace.create(
                    { type: { oneofKind: "value", value: "" } },
                  );
                props.onChange();
              }}
            >
              {filter.replace && (
                <ValueEvalEditor
                  label="Replacement tool"
                  description="JSON tool definition sent to the upstream."
                  multiline
                  value={filter.replace.type}
                  onChange={(value) => {
                    filter.replace!.type = value;
                    props.onChange();
                  }}
                />
              )}
            </EditItem>
          )}
        </EditItem>
      ))}
    </ItemMessage>
    <ItemMessage
      title="Injected tools"
      obj={props.tools.tools}
      isList
      onSet={() => {
        props.tools.tools = [
          CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool.create({
            type: { oneofKind: "value", value: "" },
            position:
              CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool_Position.APPEND,
          }),
        ];
        props.onChange();
      }}
      onAddListItem={() => {
        props.tools.tools.push(
          CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool.create({
            type: { oneofKind: "value", value: "" },
            position:
              CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool_Position.APPEND,
          }),
        );
        props.onChange();
      }}
    >
      {props.tools.tools.map((tool, index) => (
        <EditItem
          key={`tool-${index}`}
          title={`Tool ${index + 1}`}
          obj={tool}
          onUnset={() => {
            props.tools.tools.splice(index, 1);
            props.onChange();
          }}
        >
          <EnumSelect
            label="Position"
            description="Where the tool is written into the request's tool list."
            values={["PREPEND", "APPEND"]}
            enumObj={CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool_Position}
            value={tool.position}
            onChange={(value) => {
              tool.position = value;
              props.onChange();
            }}
          />
          <ValueEvalEditor
            label="Tool definition"
            description="JSON tool definition added to the request."
            multiline
            value={tool.type}
            onChange={(value) => {
              tool.type = value;
              props.onChange();
            }}
          />
        </EditItem>
      ))}
    </ItemMessage>
  </>
);

const LLMReasoningEditor = (props: {
  reasoning: CoreP.Service_Spec_Config_LLM_Reasoning;
  onChange: () => void;
}) => (
  <>
    <Select
      label="Reasoning control"
      description="How the reasoning budget of the request is decided."
      data={[
        { value: "level", label: "Level" },
        { value: "tokenBudget", label: "Token budget" },
        { value: "effort", label: "Provider effort" },
        { value: "eval", label: "CEL expression" },
        { value: "opa", label: "Rego policy" },
      ]}
      value={props.reasoning.type.oneofKind ?? "level"}
      onChange={(value) => {
        if (!value) return;
        props.reasoning.type = match(value)
          .with("tokenBudget", () => ({
            oneofKind: "tokenBudget" as const,
            tokenBudget: 0,
          }))
          .with("effort", () => ({ oneofKind: "effort" as const, effort: "" }))
          .with("eval", () => ({ oneofKind: "eval" as const, eval: "" }))
          .with("opa", () => ({ oneofKind: "opa" as const, opa: "" }))
          .otherwise(() => ({
            oneofKind: "level" as const,
            level: CoreP.Service_Spec_Config_LLM_Reasoning_Level.MEDIUM,
          }));
        props.onChange();
      }}
    />
    {props.reasoning.type.oneofKind === "level" && (
      <EnumSelect
        label="Level"
        description="Portable reasoning level mapped onto each provider's own control."
        values={["NONE", "MINIMAL", "LOW", "MEDIUM", "HIGH", "XHIGH", "MAX"]}
        enumObj={CoreP.Service_Spec_Config_LLM_Reasoning_Level}
        value={props.reasoning.type.level}
        onChange={(value) => {
          if (props.reasoning.type.oneofKind === "level") {
            props.reasoning.type.level = value;
          }
          props.onChange();
        }}
      />
    )}
    {props.reasoning.type.oneofKind === "tokenBudget" && (
      <NumberInput
        label="Token budget"
        description="Reasoning token budget requested from the upstream."
        min={0}
        value={Number(props.reasoning.type.tokenBudget)}
        onChange={(value) => {
          if (props.reasoning.type.oneofKind === "tokenBudget") {
            props.reasoning.type.tokenBudget = strToNum(value);
          }
          props.onChange();
        }}
      />
    )}
    {props.reasoning.type.oneofKind === "effort" && (
      <TextInput
        label="Provider effort"
        description="Provider-native effort value passed through verbatim."
        placeholder="high"
        value={props.reasoning.type.effort}
        onChange={(event) => {
          if (props.reasoning.type.oneofKind === "effort") {
            props.reasoning.type.effort = event.target.value;
          }
          props.onChange();
        }}
      />
    )}
    {props.reasoning.type.oneofKind === "eval" && (
      <TextInput
        label="CEL expression"
        description="CEL expression that resolves to a reasoning level or token budget."
        placeholder='ctx.user.spec.type == "HUMAN" ? "high" : "low"'
        value={props.reasoning.type.eval}
        onChange={(event) => {
          if (props.reasoning.type.oneofKind === "eval") {
            props.reasoning.type.eval = event.target.value;
          }
          props.onChange();
        }}
      />
    )}
    {props.reasoning.type.oneofKind === "opa" && (
      <TextAreaCustom
        label="Rego policy"
        description="Rego policy that resolves to a reasoning level or token budget."
        value={props.reasoning.type.opa}
        onChange={(value) => {
          if (props.reasoning.type.oneofKind === "opa") {
            props.reasoning.type.opa = value ?? "";
          }
          props.onChange();
        }}
      />
    )}
  </>
);

const LLMTokenRateLimitEditor = (props: {
  tokenRateLimit: CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <EnumSelect
        label="Scope"
        description="Which tokens are counted against the limit."
        values={["TOTAL", "INPUT", "OUTPUT"]}
        enumObj={CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope}
        value={props.tokenRateLimit.scope}
        onChange={(value) => {
          props.tokenRateLimit.scope = value;
          props.onChange();
        }}
      />
      <NumberInput
        label="Token limit"
        description="Maximum tokens allowed during the configured window."
        min={0}
        value={Number(props.tokenRateLimit.limit)}
        onChange={(value) => {
          props.tokenRateLimit.limit = strToNum(value);
          props.onChange();
        }}
      />
      <DurationPicker
        title="Window"
        description="Duration of the token rate-limit window."
        value={props.tokenRateLimit.window}
        onChange={(value) => {
          props.tokenRateLimit.window = value;
          props.onChange();
        }}
      />
    </Group>
    <Group grow align="flex-start">
      <NumberInput
        label="Default output tokens"
        description="Output tokens reserved before the response's own usage is known."
        min={0}
        value={Number(props.tokenRateLimit.defaultOutputTokens)}
        onChange={(value) => {
          props.tokenRateLimit.defaultOutputTokens = strToNum(value);
          props.onChange();
        }}
      />
      <TextInput
        label="Deny message"
        description="Message returned to the downstream when the quota is exhausted."
        placeholder="Token quota exceeded."
        value={props.tokenRateLimit.denyMessage}
        onChange={(event) => {
          props.tokenRateLimit.denyMessage = event.target.value;
          props.onChange();
        }}
      />
    </Group>
    <RateLimitKeyEditor
      keyValue={props.tokenRateLimit.key}
      onChange={(value) => {
        props.tokenRateLimit.key = value;
        props.onChange();
      }}
    />
    <ItemMessage
      title="Response headers"
      obj={props.tokenRateLimit.headers}
      isList
      onSet={() => {
        props.tokenRateLimit.headers = [
          CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_KeyValue.create(),
        ];
        props.onChange();
      }}
      onAddListItem={() => {
        props.tokenRateLimit.headers.push(
          CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_KeyValue.create(),
        );
        props.onChange();
      }}
    >
      {props.tokenRateLimit.headers.map((header, index) => (
        <div
          className="mb-3 flex w-full items-center"
          key={`token-rate-limit-header-${index}`}
        >
          <CloseButton
            size="sm"
            variant="subtle"
            className="mr-2"
            aria-label={`Remove header ${index + 1}`}
            onClick={() => {
              props.tokenRateLimit.headers.splice(index, 1);
              props.onChange();
            }}
          />
          <Group grow className="flex-1">
            <TextInput
              label="Header"
              placeholder="x-ratelimit-remaining-tokens"
              value={header.key}
              onChange={(event) => {
                header.key = event.target.value;
                props.onChange();
              }}
            />
            <TextInput
              label="Value"
              placeholder="{remaining}"
              value={header.value}
              onChange={(event) => {
                header.value = event.target.value;
                props.onChange();
              }}
            />
          </Group>
        </div>
      ))}
    </ItemMessage>
  </>
);

const LLMSemanticCacheEditor = (props: {
  semanticCache: CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <NumberInput
        label="Minimum similarity"
        description="Cosine similarity above which a cached response is served."
        min={0}
        max={1}
        step={0.01}
        decimalScale={3}
        value={props.semanticCache.minSimilarity}
        onChange={(value) => {
          props.semanticCache.minSimilarity = Number(value) || 0;
          props.onChange();
        }}
      />
      <NumberInput
        label="Maximum entries"
        description="Maximum number of cached responses kept for this Service."
        min={0}
        value={Number(props.semanticCache.maxSize)}
        onChange={(value) => {
          props.semanticCache.maxSize = strToNum(value);
          props.onChange();
        }}
      />
      <DurationPicker
        title="TTL"
        description="How long a cached response stays valid."
        value={props.semanticCache.ttl}
        onChange={(value) => {
          props.semanticCache.ttl = value;
          props.onChange();
        }}
      />
    </Group>
    <Switch
      label="Set the X-Cache header"
      description="Tell the downstream whether the response came from the cache."
      checked={props.semanticCache.useXCacheHeader}
      onChange={(event) => {
        props.semanticCache.useXCacheHeader = event.currentTarget.checked;
        props.onChange();
      }}
    />
    <EditItem
      title="Cache scope"
      description="Who shares the cached responses"
      obj={props.semanticCache.scope}
      onUnset={() => {
        props.semanticCache.scope = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.semanticCache.scope =
          CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache_Scope.create({
            type: { oneofKind: "perUser", perUser: true },
          });
        props.onChange();
      }}
    >
      {props.semanticCache.scope && (
        <Group grow align="flex-start">
          <Select
            label="Scope"
            description="Cache per User, per Session, Cluster-wide, or by a CEL expression."
            data={["perUser", "perSession", "shared", "eval"]}
            value={props.semanticCache.scope.type.oneofKind ?? "perUser"}
            onChange={(value) => {
              if (!value) return;
              props.semanticCache.scope!.type = match(value)
                .with("perSession", () => ({
                  oneofKind: "perSession" as const,
                  perSession: true,
                }))
                .with("shared", () => ({
                  oneofKind: "shared" as const,
                  shared: true,
                }))
                .with("eval", () => ({ oneofKind: "eval" as const, eval: "" }))
                .otherwise(() => ({
                  oneofKind: "perUser" as const,
                  perUser: true,
                }));
              props.onChange();
            }}
          />
          {props.semanticCache.scope.type.oneofKind === "eval" && (
            <TextInput
              label="CEL expression"
              description="CEL expression that resolves to the cache partition key."
              placeholder="ctx.user.metadata.name"
              value={props.semanticCache.scope.type.eval}
              onChange={(event) => {
                if (props.semanticCache.scope?.type.oneofKind === "eval") {
                  props.semanticCache.scope.type.eval = event.target.value;
                }
                props.onChange();
              }}
            />
          )}
        </Group>
      )}
    </EditItem>
    <EmbeddingEditor
      embedding={props.semanticCache.embedding}
      onChange={(value) => {
        props.semanticCache.embedding = value;
        props.onChange();
      }}
    />
  </>
);

const LLMSemanticRouterEditor = (props: {
  semanticRouter: CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter;
  onChange: () => void;
}) => (
  <>
    <Group grow align="flex-start">
      <NumberInput
        label="Minimum similarity"
        description="Default similarity above which a route matches."
        min={0}
        max={1}
        step={0.01}
        decimalScale={3}
        value={props.semanticRouter.minSimilarity}
        onChange={(value) => {
          props.semanticRouter.minSimilarity = Number(value) || 0;
          props.onChange();
        }}
      />
      <TextInput
        label="Fallback model"
        description="Model used when no route matches the request."
        placeholder="gpt-5-mini"
        value={props.semanticRouter.fallbackModel}
        onChange={(event) => {
          props.semanticRouter.fallbackModel = event.target.value;
          props.onChange();
        }}
      />
    </Group>
    <ItemMessage
      title="Routes"
      obj={props.semanticRouter.routes}
      isList
      onSet={() => {
        props.semanticRouter.routes = [
          CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter_Route.create(),
        ];
        props.onChange();
      }}
      onAddListItem={() => {
        props.semanticRouter.routes.push(
          CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter_Route.create(),
        );
        props.onChange();
      }}
    >
      {props.semanticRouter.routes.map((route, index) => (
        <EditItem
          key={`semantic-route-${index}`}
          title={route.name || `Route ${index + 1}`}
          obj={route}
          onUnset={() => {
            props.semanticRouter.routes.splice(index, 1);
            props.onChange();
          }}
        >
          <Group grow align="flex-start">
            <TextInput
              label="Name"
              required
              description="Name recorded in the AccessLogs when this route matches."
              placeholder="coding"
              value={route.name}
              onChange={(event) => {
                route.name = event.target.value;
                props.onChange();
              }}
            />
            <TextInput
              label="Model"
              required
              description="Model that the matched requests are routed to."
              placeholder="claude-sonnet-5"
              value={route.model}
              onChange={(event) => {
                route.model = event.target.value;
                props.onChange();
              }}
            />
            <NumberInput
              label="Minimum similarity"
              description="Overrides the plugin's own similarity threshold."
              min={0}
              max={1}
              step={0.01}
              decimalScale={3}
              value={route.minSimilarity}
              onChange={(value) => {
                route.minSimilarity = Number(value) || 0;
                props.onChange();
              }}
            />
          </Group>
          <TextInput
            label="Description"
            description="Describes the requests that belong to this route."
            placeholder="Software engineering and code review questions"
            value={route.description}
            onChange={(event) => {
              route.description = event.target.value;
              props.onChange();
            }}
          />
          <StringListEditor
            title="Examples"
            description="Example prompts embedded and compared against the request."
            placeholder="Refactor this function"
            values={route.examples}
            onChange={(values) => {
              route.examples = values;
              props.onChange();
            }}
          />
        </EditItem>
      ))}
    </ItemMessage>
    <EmbeddingEditor
      embedding={props.semanticRouter.embedding}
      onChange={(value) => {
        props.semanticRouter.embedding = value;
        props.onChange();
      }}
    />
  </>
);

const GatewayPluginsEditor = (props: {
  config: GatewayConfig;
  kind: GatewayPluginKind;
  onChange: () => void;
}) => {
  const plugins = props.config.plugins as any[];
  const types = props.kind === "mcp" ? mcpPluginTypes : llmPluginTypes;

  return (
    <ItemMessage
      title="Plugins"
      obj={plugins}
      isList
      onSet={() => {
        props.config.plugins = [createGatewayPlugin()];
        props.onChange();
      }}
      onAddListItem={() => {
        plugins.push(createGatewayPlugin());
        props.onChange();
      }}
    >
      {plugins.map((plugin, index) => (
        <EditItem
          key={`${plugin.name || "plugin"}-${index}`}
          title={plugin.name || `Plugin ${index + 1}`}
          obj={plugin}
          onUnset={() => {
            plugins.splice(index, 1);
            props.onChange();
          }}
        >
          <Group grow align="flex-start">
            <TextInput
              label="Name"
              required
              description="Unique name used to identify this plugin."
              placeholder="my-plugin"
              value={plugin.name}
              onChange={(event) => {
                plugin.name = event.target.value;
                props.onChange();
              }}
            />
            <EnumSelect
              label="Phase"
              description="Run the plugin before or after authentication and authorization."
              values={["PRE_AUTH", "POST_AUTH"]}
              enumObj={CoreP.Service_Spec_Config_HTTP_Plugin_Phase}
              value={plugin.phase}
              onChange={(value) => {
                plugin.phase = value;
                props.onChange();
              }}
            />
            <Switch
              label="Disabled"
              description="Disable this plugin without removing its configuration."
              checked={plugin.isDisabled}
              onChange={(event) => {
                plugin.isDisabled = event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
          <Cond
            item={
              plugin.condition ??
              CoreP.Condition.create({
                type: { oneofKind: "matchAny", matchAny: true },
              })
            }
            onChange={(condition) => {
              plugin.condition = condition;
              props.onChange();
            }}
          />
          <Tabs
            className="mb-4"
            value={plugin.type.oneofKind ?? "direct"}
            onChange={(value) => {
              if (!value) return;
              plugin.type = createGatewayPluginType(value, props.kind);
              props.onChange();
            }}
          >
            <Tabs.List className="w-full">
              {types.map((type) => (
                <Tabs.Tab key={type.value} value={type.value}>
                  {type.label}
                </Tabs.Tab>
              ))}
            </Tabs.List>
          </Tabs>
          <SharedPluginTypeEditor type={plugin.type} onChange={props.onChange} />
          {plugin.type.oneofKind === "guardrail" &&
            (props.kind === "mcp" ? (
              <MCPGuardrailEditor
                guardrail={plugin.type.guardrail}
                onChange={props.onChange}
              />
            ) : (
              <LLMGuardrailEditor
                guardrail={plugin.type.guardrail}
                onChange={props.onChange}
              />
            ))}
          {plugin.type.oneofKind === "prompt" && (
            <LLMPromptEditor
              prompt={plugin.type.prompt}
              onChange={props.onChange}
            />
          )}
          {plugin.type.oneofKind === "tools" && (
            <LLMToolsEditor tools={plugin.type.tools} onChange={props.onChange} />
          )}
          {plugin.type.oneofKind === "model" && (
            <ValueEvalEditor
              label="Model name"
              description="Model name sent to the upstream provider."
              placeholder="gpt-5-mini"
              value={plugin.type.model.type}
              onChange={(value) => {
                plugin.type.model.type = value;
                props.onChange();
              }}
            />
          )}
          {plugin.type.oneofKind === "reasoning" && (
            <LLMReasoningEditor
              reasoning={plugin.type.reasoning}
              onChange={props.onChange}
            />
          )}
          {plugin.type.oneofKind === "tokenRateLimit" && (
            <LLMTokenRateLimitEditor
              tokenRateLimit={plugin.type.tokenRateLimit}
              onChange={props.onChange}
            />
          )}
          {plugin.type.oneofKind === "semanticCache" && (
            <LLMSemanticCacheEditor
              semanticCache={plugin.type.semanticCache}
              onChange={props.onChange}
            />
          )}
          {plugin.type.oneofKind === "semanticRouter" && (
            <LLMSemanticRouterEditor
              semanticRouter={plugin.type.semanticRouter}
              onChange={props.onChange}
            />
          )}
        </EditItem>
      ))}
    </ItemMessage>
  );
};

const GatewayCommonEditor = (props: {
  config: GatewayConfig;
  kind: GatewayPluginKind;
  onChange: () => void;
}) => (
  <>
    <Group grow>
      <Switch
        label="HTTP/2 upstream"
        description="Connect to the upstream over HTTP/2"
        checked={props.config.isUpstreamHTTP2}
        onChange={(event) => {
          props.config.isUpstreamHTTP2 = event.currentTarget.checked;
          props.onChange();
        }}
      />
      <Switch
        label="Listen over HTTP/2"
        description="Enable HTTP/2 on the Service listener"
        checked={props.config.listenHTTP2}
        onChange={(event) => {
          props.config.listenHTTP2 = event.currentTarget.checked;
          props.onChange();
        }}
      />
    </Group>
    <GatewayAuthEditor {...props} />
    <GatewayHeaderEditor {...props} />
    <GatewayPathEditor {...props} />
    <GatewayPluginsEditor {...props} />
  </>
);

const mcpProtocolVersions = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];

const MCPConfigEditor = (props: {
  config: CoreP.Service_Spec_Config_MCP;
  onChange: () => void;
}) => (
  <div className="w-full space-y-4">
    <TextInput
      label="MCP endpoint"
      description="Only requests matching this exact path are accepted when set"
      placeholder="/mcp"
      value={props.config.endpoint}
      onChange={(event) => {
        props.config.endpoint = event.target.value;
        props.onChange();
      }}
    />
    <EditItem
      title="Protocol validation"
      description="Restrict MCP protocol versions and method handling"
      obj={props.config.protocol}
      onUnset={() => {
        props.config.protocol = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.protocol =
          CoreP.Service_Spec_Config_MCP_Protocol.create();
        props.onChange();
      }}
    >
      {props.config.protocol && (
        <>
          <TagsInput
            label="Accepted protocol versions"
            description="Select a canonical MCP version or enter a custom version"
            placeholder="Choose a version or type a custom value"
            data={mcpProtocolVersions}
            value={props.config.protocol.versions}
            onChange={(values) => {
              props.config.protocol!.versions = values;
              props.onChange();
            }}
          />
          <Group grow>
            <Switch
              label="Require protocol version"
              description="Reject requests that do not provide an accepted MCP-Protocol-Version."
              checked={props.config.protocol.requireVersion}
              onChange={(event) => {
                props.config.protocol!.requireVersion = event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Reject unknown methods"
              description="Reject MCP methods that are not part of the known method set."
              checked={props.config.protocol.rejectUnknownMethods}
              onChange={(event) => {
                props.config.protocol!.rejectUnknownMethods =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
        </>
      )}
    </EditItem>
    <EditItem
      title="MCP limits"
      description="Bound request and stream event inspection sizes"
      obj={props.config.limits}
      onUnset={() => {
        props.config.limits = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.limits = CoreP.Service_Spec_Config_MCP_Limits.create();
        props.onChange();
      }}
    >
      {props.config.limits && (
        <Group grow>
          <NumberInput
            label="Max request bytes"
            description="Maximum MCP request body size inspected by Vigil."
            min={0}
            value={props.config.limits.maxRequestBytes}
            onChange={(value) => {
              props.config.limits!.maxRequestBytes = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max stream event bytes"
            description="Maximum size of an individual MCP stream event inspected by Vigil."
            min={0}
            value={props.config.limits.maxStreamEventBytes}
            onChange={(value) => {
              props.config.limits!.maxStreamEventBytes = strToNum(value);
              props.onChange();
            }}
          />
        </Group>
      )}
    </EditItem>
    <CORSConfigEditor
      cors={props.config.cors}
      onChange={(cors) => {
        props.config.cors = cors;
        props.onChange();
      }}
    />
    <Switch
      label="Disable Origin validation"
      description="Allow requests from any Origin, including untrusted origins"
      checked={props.config.disableOriginCheck}
      onChange={(event) => {
        props.config.disableOriginCheck = event.currentTarget.checked;
        props.onChange();
      }}
    />
    <EditItem
      title="MCP visibility"
      description="Choose which MCP payloads and headers are recorded in AccessLogs"
      obj={props.config.visibility}
      onUnset={() => {
        props.config.visibility = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.visibility =
          CoreP.Service_Spec_Config_MCP_Visibility.create();
        props.onChange();
      }}
    >
      {props.config.visibility && (
        <>
          <Group grow>
            <Switch
              label="Disable request body"
              description="Do not record MCP request bodies in visibility logs."
              checked={props.config.visibility.disableRequestBody}
              onChange={(event) => {
                props.config.visibility!.disableRequestBody =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Disable response body"
              description="Do not record MCP response bodies in visibility logs."
              checked={props.config.visibility.disableResponseBody}
              onChange={(event) => {
                props.config.visibility!.disableResponseBody =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
          <StringListEditor
            title="Request header allowlist"
            description="Request headers to include in visibility logs."
            values={props.config.visibility.includeRequestHeaders}
            onChange={(values) => {
              props.config.visibility!.includeRequestHeaders = values;
              props.onChange();
            }}
          />
          <StringListEditor
            title="Response header allowlist"
            description="Response headers to include in visibility logs."
            values={props.config.visibility.includeResponseHeaders}
            onChange={(values) => {
              props.config.visibility!.includeResponseHeaders = values;
              props.onChange();
            }}
          />
          <Group grow>
            <Switch
              label="Include all request headers"
              description="Record every request header instead of only the allowlist."
              checked={props.config.visibility.includeAllRequestHeaders}
              onChange={(event) => {
                props.config.visibility!.includeAllRequestHeaders =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Include all response headers"
              description="Record every response header instead of only the allowlist."
              checked={props.config.visibility.includeAllResponseHeaders}
              onChange={(event) => {
                props.config.visibility!.includeAllResponseHeaders =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
          <StringListEditor
            title="Request header denylist"
            description="Request headers to omit from visibility logs."
            values={props.config.visibility.excludeRequestHeaders}
            onChange={(values) => {
              props.config.visibility!.excludeRequestHeaders = values;
              props.onChange();
            }}
          />
          <StringListEditor
            title="Response header denylist"
            description="Response headers to omit from visibility logs."
            values={props.config.visibility.excludeResponseHeaders}
            onChange={(values) => {
              props.config.visibility!.excludeResponseHeaders = values;
              props.onChange();
            }}
          />
        </>
      )}
    </EditItem>
    <GatewayCommonEditor config={props.config} kind="mcp" onChange={props.onChange} />
  </div>
);

const LLMConfigEditor = (props: {
  config: CoreP.Service_Spec_Config_LLM;
  onChange: () => void;
}) => (
  <div className="w-full space-y-4">
    <EnumSelect
      label="Inference protocol"
      description="The API protocol spoken by the downstreams and the upstream"
      values={llmProtocolNames}
      enumObj={CoreP.Service_Spec_Config_LLM_Protocol}
      value={props.config.protocol}
      onChange={(value) => {
        props.config.protocol = value;
        props.onChange();
      }}
    />
    <EditItem
      title="Model rewrite"
      description="Optionally replace the downstream model name before proxying"
      obj={props.config.model}
      onUnset={() => {
        props.config.model = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.model = CoreP.Service_Spec_Config_LLM_Model.create({
          type: { oneofKind: "value", value: "" },
        });
        props.onChange();
      }}
    >
      {props.config.model && (
        <Group grow align="flex-start">
          <Select
            label="Rewrite source"
            description="Use a fixed model name or evaluate a CEL expression."
            data={["value", "eval"]}
            value={props.config.model.type.oneofKind ?? "value"}
            onChange={(value) => {
              if (!value) return;
              props.config.model!.type =
                value === "eval"
                  ? { oneofKind: "eval", eval: "" }
                  : { oneofKind: "value", value: "" };
              props.onChange();
            }}
          />
          {props.config.model.type.oneofKind === "value" ? (
            <TextInput
              label="Model name"
              description="Model name sent to the upstream provider."
              placeholder="gpt-4.1"
              value={props.config.model.type.value}
              onChange={(event) => {
                if (props.config.model?.type.oneofKind === "value") {
                  props.config.model.type.value = event.target.value;
                }
                props.onChange();
              }}
            />
          ) : (
            <TextInput
              label="CEL expression"
              description="CEL expression that resolves to the upstream model name."
              placeholder="ctx.request.llm.model"
              value={
                props.config.model.type.oneofKind === "eval"
                  ? props.config.model.type.eval
                  : ""
              }
              onChange={(event) => {
                if (props.config.model?.type.oneofKind === "eval") {
                  props.config.model.type.eval = event.target.value;
                }
                props.onChange();
              }}
            />
          )}
        </Group>
      )}
    </EditItem>
    <EditItem
      title="LLM limits"
      description="Bound request parsing and inference request semantics"
      obj={props.config.limits}
      onUnset={() => {
        props.config.limits = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.limits = CoreP.Service_Spec_Config_LLM_Limits.create();
        props.onChange();
      }}
    >
      {props.config.limits && (
        <Group grow>
          <NumberInput
            label="Max request bytes"
            description="Maximum LLM request body size inspected by Vigil."
            min={0}
            value={props.config.limits.maxRequestBytes}
            onChange={(value) => {
              props.config.limits!.maxRequestBytes = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max stream event bytes"
            description="Maximum size of an individual streamed LLM event."
            min={0}
            value={props.config.limits.maxStreamEventBytes}
            onChange={(value) => {
              props.config.limits!.maxStreamEventBytes = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max estimated input tokens"
            description="Maximum estimated input tokens accepted in a request."
            min={0}
            value={props.config.limits.maxEstimatedInputTokens}
            onChange={(value) => {
              props.config.limits!.maxEstimatedInputTokens = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max output tokens"
            description="Maximum output tokens requested from the upstream provider."
            min={0}
            value={props.config.limits.maxOutputTokens}
            onChange={(value) => {
              props.config.limits!.maxOutputTokens = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max tools"
            description="Maximum number of tools accepted in an LLM request."
            min={0}
            value={props.config.limits.maxTools}
            onChange={(value) => {
              props.config.limits!.maxTools = strToNum(value);
              props.onChange();
            }}
          />
          <NumberInput
            label="Max tool schema bytes"
            description="Maximum size of an individual tool's JSON schema."
            min={0}
            value={props.config.limits.maxToolSchemaBytes}
            onChange={(value) => {
              props.config.limits!.maxToolSchemaBytes = strToNum(value);
              props.onChange();
            }}
          />
        </Group>
      )}
    </EditItem>
    <EditItem
      title="Reasoning"
      description="Set the reasoning budget of every request that the Service serves"
      obj={props.config.reasoning}
      onUnset={() => {
        props.config.reasoning = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.reasoning = CoreP.Service_Spec_Config_LLM_Reasoning.create({
          type: {
            oneofKind: "level",
            level: CoreP.Service_Spec_Config_LLM_Reasoning_Level.MEDIUM,
          },
        });
        props.onChange();
      }}
    >
      {props.config.reasoning && (
        <LLMReasoningEditor
          reasoning={props.config.reasoning}
          onChange={props.onChange}
        />
      )}
    </EditItem>
    <EmbeddingEditor
      embedding={props.config.embedding}
      onChange={(value) => {
        props.config.embedding = value;
        props.onChange();
      }}
    />
    <EditItem
      title="LLM visibility"
      description="Control recording of sensitive prompts, completions, and headers"
      obj={props.config.visibility}
      onUnset={() => {
        props.config.visibility = undefined;
        props.onChange();
      }}
      onSet={() => {
        props.config.visibility =
          CoreP.Service_Spec_Config_LLM_Visibility.create();
        props.onChange();
      }}
    >
      {props.config.visibility && (
        <>
          <Group grow>
            <Switch
              label="Record request body"
              description="Include the raw request body in visibility logs."
              checked={props.config.visibility.enableRequestBody}
              onChange={(event) => {
                props.config.visibility!.enableRequestBody =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Record request body map"
              description="Include the parsed request body map in visibility logs."
              checked={props.config.visibility.enableRequestBodyMap}
              onChange={(event) => {
                props.config.visibility!.enableRequestBodyMap =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Record response body"
              description="Include the raw response body in visibility logs."
              checked={props.config.visibility.enableResponseBody}
              onChange={(event) => {
                props.config.visibility!.enableResponseBody =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Record response body map"
              description="Include the parsed response body map in visibility logs."
              checked={props.config.visibility.enableResponseBodyMap}
              onChange={(event) => {
                props.config.visibility!.enableResponseBodyMap =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
          <StringListEditor
            title="Request header allowlist"
            description="Request headers to include in visibility logs."
            values={props.config.visibility.includeRequestHeaders}
            onChange={(values) => {
              props.config.visibility!.includeRequestHeaders = values;
              props.onChange();
            }}
          />
          <StringListEditor
            title="Response header allowlist"
            description="Response headers to include in visibility logs."
            values={props.config.visibility.includeResponseHeaders}
            onChange={(values) => {
              props.config.visibility!.includeResponseHeaders = values;
              props.onChange();
            }}
          />
          <Group grow>
            <Switch
              label="Include all request headers"
              description="Record every request header instead of only the allowlist."
              checked={props.config.visibility.includeAllRequestHeaders}
              onChange={(event) => {
                props.config.visibility!.includeAllRequestHeaders =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
            <Switch
              label="Include all response headers"
              description="Record every response header instead of only the allowlist."
              checked={props.config.visibility.includeAllResponseHeaders}
              onChange={(event) => {
                props.config.visibility!.includeAllResponseHeaders =
                  event.currentTarget.checked;
                props.onChange();
              }}
            />
          </Group>
          <StringListEditor
            title="Request header denylist"
            description="Request headers to omit from visibility logs."
            values={props.config.visibility.excludeRequestHeaders}
            onChange={(values) => {
              props.config.visibility!.excludeRequestHeaders = values;
              props.onChange();
            }}
          />
          <StringListEditor
            title="Response header denylist"
            description="Response headers to omit from visibility logs."
            values={props.config.visibility.excludeResponseHeaders}
            onChange={(values) => {
              props.config.visibility!.excludeResponseHeaders = values;
              props.onChange();
            }}
          />
        </>
      )}
    </EditItem>
    <CORSConfigEditor
      cors={props.config.cors}
      onChange={(cors) => {
        props.config.cors = cors;
        props.onChange();
      }}
    />
    <GatewayCommonEditor config={props.config} kind="llm" onChange={props.onChange} />
  </div>
);

const Config = (props: {
  item: CoreP.Service_Spec_Config;

  onUpdate: (item: CoreP.Service_Spec_Config) => void;
  default?: boolean;
  mode?: CoreP.Service_Spec_Mode;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(cloneConfigForMode(item, props.mode));
  const [init] = React.useState(cloneConfigForMode(item, props.mode));

  React.useEffect(() => {
    setReq(cloneConfigForMode(item, props.mode));
  }, [item, props.mode]);

  const updateReq = () => {
    setReq(CoreP.Service_Spec_Config.clone(req));
    onUpdate(CoreP.Service_Spec_Config.clone(req));
  };

  return (
    <div className="w-full">
      {!props.default && (
        <div className="mb-6">
          <TextInput
            required
            label="Name"
            description="Set a unique name for the configuration"
            placeholder="my-config"
            value={req.name}
            onChange={(v) => {
              req.name = v.target.value;
              updateReq();
            }}
          />
        </div>
      )}
      <EditItem
        title="Upstream"
        description="Set Service Upstream"
        onUnset={() => {
          req.upstream = undefined;
          updateReq();
        }}
        obj={req.upstream}
        onSet={() => {
          if (!req.upstream) {
            req.upstream = CoreP.Service_Spec_Config_Upstream.create({
              type: {
                oneofKind: "url",
                url: "",
              },
            });
            updateReq();
          }
        }}
      >
        {req.upstream && (
          <>
            <Tabs
              className="mb-8"
              value={req.upstream.type.oneofKind}
              onChange={(v) => {
                match(v)
                  .with("url", () => {
                    match(init.upstream?.type.oneofKind)
                      .with(`url`, () => {
                        req.upstream!.type = structuredClone(
                          init!.upstream!.type,
                        );
                      })
                      .otherwise(() => {
                        req.upstream!.type = {
                          oneofKind: "url",
                          url: "",
                        };
                      });

                    updateReq();
                  })
                  .with("container", () => {
                    match(init.upstream?.type.oneofKind)
                      .with(`container`, () => {
                        req.upstream!.type = structuredClone(
                          init!.upstream!.type,
                        );
                      })
                      .otherwise(() => {
                        req.upstream!.type = {
                          oneofKind: "container",
                          container:
                            CoreP.Service_Spec_Config_Upstream_Container.create(),
                        };
                      });

                    updateReq();
                  })
                  .with("loadbalance", () => {
                    match(init.upstream?.type.oneofKind)
                      .with("loadbalance", () => {
                        req.upstream!.type = structuredClone(
                          init!.upstream!.type,
                        );
                      })
                      .otherwise(() => {
                        req.upstream!.type = {
                          oneofKind: "loadbalance",
                          loadbalance:
                            CoreP.Service_Spec_Config_Upstream_Loadbalance.create(
                              {
                                endpoints: [
                                  CoreP.Service_Spec_Config_Upstream_Loadbalance_Endpoint.create(),
                                ],
                              },
                            ),
                        };
                      });
                    updateReq();
                  });
              }}
            >
              <Tabs.List>
                <Tabs.Tab value="url">URL</Tabs.Tab>
                <Tabs.Tab value="container">Managed Container</Tabs.Tab>
                <Tabs.Tab value="loadbalance">Load Balance</Tabs.Tab>
              </Tabs.List>

              <Tabs.Panel value="url">
                {match(req.upstream.type)
                  .when(
                    (x) => x.oneofKind === `url`,
                    (url) => {
                      return (
                        <Group grow>
                          <TextInput
                            required
                            label="URL"
                            description="The upstream canonical URL"
                            placeholder="https://example.com"
                            value={url.url}
                            onChange={(v) => {
                              url.url = v.target.value;
                              updateReq();
                            }}
                          />
                          <SelectResource
                            api="core"
                            kind="User"
                            defaultValue={req.upstream!.user}
                            label="Serve by User"
                            clearable
                            description="Serve the upstream from a connected client by a User"
                            onChange={(v) => {
                              req.upstream!.user = v?.metadata?.name ?? "";
                              updateReq();
                            }}
                          />
                        </Group>
                      );
                    },
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </Tabs.Panel>

              <Tabs.Panel value="container">
                {match(req.upstream.type)
                  .when(
                    (x) => x.oneofKind === `container`,
                    (container) => {
                      return (
                        <div>
                          <Group grow>
                            <TextInput
                              required
                              label="Image"
                              description="The Docker/container image URL"
                              placeholder="postgres:latest"
                              value={container.container.image}
                              onChange={(v) => {
                                container.container.image = v.target.value;
                                updateReq();
                              }}
                            />
                            <NumberInput
                              label="Port"
                              description="Set the exposed port number of the container"
                              required
                              placeholder="8080"
                              min={0}
                              max={65535}
                              value={container.container.port}
                              onChange={(v) => {
                                container.container.port = strToNum(v);
                                updateReq();
                              }}
                            />
                            <NumberInput
                              label="Replicas"
                              description="Set the number of containers replicas to be deployed"
                              placeholder="3"
                              min={0}
                              max={1000}
                              value={container.container.replicas}
                              onChange={(v) => {
                                container.container.replicas = strToNum(v);
                                updateReq();
                              }}
                            />
                          </Group>

                          <ItemMessage
                            title="Environment Variables"
                            obj={container.container.env}
                            isList
                            onSet={() => {
                              container.container.env = [
                                CoreP.Service_Spec_Config_Upstream_Container_Env.create(
                                  {
                                    name: "",
                                    type: { oneofKind: "value", value: "" },
                                  },
                                ),
                              ];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.env.push(
                                CoreP.Service_Spec_Config_Upstream_Container_Env.create(
                                  {
                                    name: "",
                                    type: { oneofKind: "value", value: "" },
                                  },
                                ),
                              );
                              updateReq();
                            }}
                          >
                            {container.container.env.map((envVar, idx) => (
                              <div className="w-full flex mb-3" key={idx}>
                                <CloseButton
                                  size={"sm"}
                                  variant="subtle"
                                  className="mr-2"
                                  onClick={() => {
                                    container.container.env.splice(idx, 1);
                                    updateReq();
                                  }}
                                ></CloseButton>
                                <div className="flex-1">
                                  <Group grow align="flex-start">
                                    <TextInput
                                      required
                                      label="Key"
                                      description="Set the environment variable key"
                                      placeholder="MY_KEY"
                                      value={container.container.env[idx].name}
                                      onChange={(v) => {
                                        container.container.env[idx].name =
                                          v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <Select
                                      label="Value type"
                                      description="Choose whether the environment variable is a literal value or comes from a Secret."
                                      data={[
                                        { label: "Value", value: "value" },
                                        {
                                          label: "From Secret",
                                          value: "fromSecret",
                                        },
                                        {
                                          label: "Kubernetes Secret",
                                          value: "kubernetesSecretRef",
                                        },
                                      ]}
                                      value={
                                        container.container.env[idx].type
                                          .oneofKind
                                      }
                                      onChange={(val) => {
                                        container.container.env[idx].type =
                                          match(val)
                                            .with("fromSecret", () => ({
                                              oneofKind: "fromSecret" as const,
                                              fromSecret: "",
                                            }))
                                            .with(
                                              "kubernetesSecretRef",
                                              () => ({
                                                oneofKind:
                                                  "kubernetesSecretRef" as const,
                                                kubernetesSecretRef:
                                                  CoreP.Service_Spec_Config_Upstream_Container_Env_KubernetesSecretRef.create(),
                                              }),
                                            )
                                            .otherwise(() => ({
                                              oneofKind: "value" as const,
                                              value: "",
                                            }));
                                        updateReq();
                                      }}
                                    />
                                  </Group>

                                  {container.container.env[idx].type
                                    .oneofKind === `value` && (
                                    <TextInput
                                      required
                                      label="Value"
                                      description="Set the environment variable value"
                                      placeholder="my-value"
                                      value={
                                        container.container.env[idx].type
                                          .oneofKind === `value`
                                          ? container.container.env[idx].type
                                              .value
                                          : ""
                                      }
                                      onChange={(v) => {
                                        container.container.env[idx].type = {
                                          oneofKind: "value",
                                          value: v.target.value,
                                        };
                                        updateReq();
                                      }}
                                    />
                                  )}

                                  {container.container.env[idx].type
                                    .oneofKind === `fromSecret` && (
                                    <SelectResource
                                      api="core"
                                      kind="Secret"
                                      label="Value Secret"
                                      description="Select the Secret whose value is used"
                                      defaultValue={
                                        container.container.env[idx].type
                                          .oneofKind === `fromSecret`
                                          ? container.container.env[idx].type
                                              .fromSecret
                                          : undefined
                                      }
                                      onChange={(v) => {
                                        container.container.env[idx].type = {
                                          oneofKind: "fromSecret",
                                          fromSecret: v?.metadata?.name ?? "",
                                        };
                                        updateReq();
                                      }}
                                    />
                                  )}

                                  {container.container.env[idx].type
                                    .oneofKind === `kubernetesSecretRef` && (
                                    <Group grow>
                                      <TextInput
                                        required
                                        label="Kubernetes Secret name"
                                        description="Set the Kubernetes secret name"
                                        placeholder="my-secret"
                                        value={
                                          container.container.env[idx].type
                                            .oneofKind === `kubernetesSecretRef`
                                            ? container.container.env[idx].type
                                                .kubernetesSecretRef.name
                                            : ""
                                        }
                                        onChange={(v) => {
                                          const t =
                                            container.container.env[idx].type;
                                          if (
                                            t.oneofKind ===
                                            `kubernetesSecretRef`
                                          ) {
                                            t.kubernetesSecretRef.name =
                                              v.target.value;
                                            updateReq();
                                          }
                                        }}
                                      />
                                      <TextInput
                                        required
                                        label="Kubernetes Secret key"
                                        description="Set the Kubernetes secret data key"
                                        placeholder="my-key"
                                        value={
                                          container.container.env[idx].type
                                            .oneofKind === `kubernetesSecretRef`
                                            ? container.container.env[idx].type
                                                .kubernetesSecretRef.key
                                            : ""
                                        }
                                        onChange={(v) => {
                                          const t =
                                            container.container.env[idx].type;
                                          if (
                                            t.oneofKind ===
                                            `kubernetesSecretRef`
                                          ) {
                                            t.kubernetesSecretRef.key =
                                              v.target.value;
                                            updateReq();
                                          }
                                        }}
                                      />
                                    </Group>
                                  )}
                                </div>
                              </div>
                            ))}
                          </ItemMessage>

                          <EditItem
                            title="Credentials"
                            description="Set authentication-specific info required to pull the container image"
                            onUnset={() => {
                              container.container.credentials = undefined;
                              updateReq();
                            }}
                            obj={container.container.credentials}
                            onSet={() => {
                              container.container.credentials =
                                CoreP.Service_Spec_Config_Upstream_Container_Credentials.create(
                                  {
                                    type: {
                                      oneofKind: `usernamePassword`,
                                      usernamePassword: {
                                        password: {
                                          type: {
                                            oneofKind: `fromSecret`,
                                            fromSecret: "",
                                          },
                                        },
                                      } as CoreP.Service_Spec_Config_Upstream_Container_Credentials_UsernamePassword,
                                    },
                                  },
                                );

                              updateReq();
                            }}
                          >
                            {match(container.container.credentials?.type)
                              .when(
                                (x) => x?.oneofKind === `usernamePassword`,
                                (usernamePassword) => {
                                  return (
                                    <div>
                                      <Group grow>
                                        <TextInput
                                          required
                                          label="Username"
                                          description="Set the authentication username"
                                          placeholder="linus-torvalds"
                                          value={
                                            usernamePassword.usernamePassword
                                              .username
                                          }
                                          onChange={(v) => {
                                            usernamePassword.usernamePassword.username =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        <SelectResource
                                          api="core"
                                          kind="Secret"
                                          label="Password Secret"
                                          description="Select the secret of the password"
                                          defaultValue={
                                            usernamePassword.usernamePassword
                                              .password?.type.oneofKind ===
                                            `fromSecret`
                                              ? usernamePassword
                                                  .usernamePassword.password
                                                  .type.fromSecret
                                              : undefined
                                          }
                                          onChange={(val) => {
                                            match(
                                              usernamePassword.usernamePassword
                                                .password?.type,
                                            ).when(
                                              (x) =>
                                                x?.oneofKind === `fromSecret`,
                                              (x) => {
                                                x.fromSecret =
                                                  val?.metadata?.name ?? "";
                                              },
                                            );

                                            updateReq();
                                          }}
                                        />
                                        <TextInput
                                          label="Server"
                                          description="Set the registry server"
                                          placeholder="ghcr.io"
                                          value={
                                            usernamePassword.usernamePassword
                                              .server
                                          }
                                          onChange={(v) => {
                                            usernamePassword.usernamePassword.server =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                      </Group>
                                    </div>
                                  );
                                },
                              )
                              .otherwise(() => (
                                <></>
                              ))}
                          </EditItem>

                          <EditItem
                            title="Resource Limit"
                            description="Set the container runtime resource limits (e.g. CPU, memory)"
                            onUnset={() => {
                              container.container.resourceLimit = undefined;
                              updateReq();
                            }}
                            obj={container.container.resourceLimit}
                            onSet={() => {
                              container.container.resourceLimit =
                                CoreP.Service_Spec_Config_Upstream_Container_ResourceLimit.create(
                                  {
                                    cpu: { millicores: 0 },
                                    memory: { megabytes: 0 },
                                  },
                                );

                              updateReq();
                            }}
                          >
                            {container.container.resourceLimit && (
                              <div>
                                <Group grow>
                                  <NumberInput
                                    label="CPU"
                                    placeholder="2000"
                                    description="Set the CPU millicores"
                                    min={0}
                                    value={
                                      container.container.resourceLimit!.cpu!
                                        .millicores
                                    }
                                    onChange={(v) => {
                                      container.container.resourceLimit!.cpu!.millicores =
                                        strToNum(v);
                                      updateReq();
                                    }}
                                  />

                                  <NumberInput
                                    label="Memory"
                                    placeholder="4000"
                                    description="Set the memory in megabytes"
                                    min={0}
                                    value={
                                      container.container.resourceLimit!.memory!
                                        .megabytes
                                    }
                                    onChange={(v) => {
                                      container.container.resourceLimit!.memory!.megabytes =
                                        strToNum(v);
                                      updateReq();
                                    }}
                                  />
                                </Group>

                                <ItemMessage
                                  title="Extended Resource Limits"
                                  obj={
                                    Object.keys(
                                      container.container.resourceLimit!.ext,
                                    ).length > 0
                                      ? container.container.resourceLimit!.ext
                                      : undefined
                                  }
                                  isList
                                  onSet={() => {
                                    const ext =
                                      container.container.resourceLimit!.ext;
                                    let idx = Object.keys(ext).length + 1;
                                    let key = `resource-${idx}`;
                                    while (Object.hasOwn(ext, key)) {
                                      key = `resource-${++idx}`;
                                    }
                                    ext[key] = "";
                                    updateReq();
                                  }}
                                  onAddListItem={() => {
                                    const ext =
                                      container.container.resourceLimit!.ext;
                                    let idx = Object.keys(ext).length + 1;
                                    let key = `resource-${idx}`;
                                    while (Object.hasOwn(ext, key)) {
                                      key = `resource-${++idx}`;
                                    }
                                    ext[key] = "";
                                    updateReq();
                                  }}
                                >
                                  {Object.entries(
                                    container.container.resourceLimit!.ext,
                                  ).map(([k, v]) => (
                                    <div className="w-full flex mb-3" key={k}>
                                      <CloseButton
                                        size={"sm"}
                                        variant="subtle"
                                        className="mr-2"
                                        onClick={() => {
                                          delete container.container
                                            .resourceLimit!.ext[k];
                                          updateReq();
                                        }}
                                      ></CloseButton>
                                      <Group className="flex w-full" grow>
                                        <TextInput
                                          required
                                          label="Resource"
                                          description="Set the resource name"
                                          placeholder="nvidia.com/gpu"
                                          value={k}
                                          onChange={(e) => {
                                            const nextKey = e.target.value;
                                            if (
                                              nextKey !== k &&
                                              Object.hasOwn(
                                                container.container
                                                  .resourceLimit!.ext,
                                                nextKey,
                                              )
                                            ) {
                                              return;
                                            }
                                            const val =
                                              container.container.resourceLimit!
                                                .ext[k];
                                            delete container.container
                                              .resourceLimit!.ext[k];
                                            container.container.resourceLimit!.ext[
                                              nextKey
                                            ] = val;
                                            updateReq();
                                          }}
                                        />
                                        <TextInput
                                          required
                                          label="Value"
                                          description="Set the resource value"
                                          placeholder="1"
                                          value={v}
                                          onChange={(e) => {
                                            container.container.resourceLimit!.ext[
                                              k
                                            ] = e.target.value;
                                            updateReq();
                                          }}
                                        />
                                      </Group>
                                    </div>
                                  ))}
                                </ItemMessage>
                              </div>
                            )}
                          </EditItem>

                          <ItemMessage
                            title="Command"
                            obj={container.container.command}
                            isList
                            onSet={() => {
                              container.container.command = [""];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.command.push("");
                              updateReq();
                            }}
                          >
                            {container.container.command.map((x, idx) => (
                              <div className="w-full flex mb-3" key={idx}>
                                <CloseButton
                                  size="sm"
                                  variant="subtle"
                                  onClick={() => {
                                    container.container.command.splice(idx, 1);
                                    updateReq();
                                  }}
                                />
                                <TextInput
                                  required
                                  label="Command"
                                  description="Executable started as the container entrypoint."
                                  placeholder="/bin/sh"
                                  className="flex-1"
                                  value={container.container.command[idx]}
                                  onChange={(v) => {
                                    container.container.command[idx] =
                                      v.target.value;
                                    updateReq();
                                  }}
                                />
                              </div>
                            ))}
                          </ItemMessage>

                          <ItemMessage
                            title="Args"
                            obj={container.container.args}
                            isList
                            onSet={() => {
                              container.container.args = [""];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.args.push("");
                              updateReq();
                            }}
                          >
                            {container.container.args.map((x, idx) => (
                              <div className="w-full flex mb-3" key={idx}>
                                <CloseButton
                                  size="sm"
                                  variant="subtle"
                                  onClick={() => {
                                    container.container.args.splice(idx, 1);
                                    updateReq();
                                  }}
                                />
                                <TextInput
                                  required
                                  label="Arg"
                                  description="Argument passed to the container command."
                                  placeholder="-c"
                                  className="flex-1"
                                  value={container.container.args[idx]}
                                  onChange={(v) => {
                                    container.container.args[idx] =
                                      v.target.value;
                                    updateReq();
                                  }}
                                />
                              </div>
                            ))}
                          </ItemMessage>

                          <EditItem
                            title="Security Context"
                            description="Set Kubernetes security context"
                            onUnset={() => {
                              container.container.securityContext = undefined;
                              updateReq();
                            }}
                            obj={container.container.securityContext}
                            onSet={() => {
                              container.container.securityContext =
                                CoreP.Service_Spec_Config_Upstream_Container_SecurityContext.create();
                              updateReq();
                            }}
                          >
                            {container.container.securityContext && (
                              <div>
                                <Group grow>
                                  <Switch
                                    label="Read-only root filesystem"
                                    description="Prevent processes from writing to the container root filesystem."
                                    checked={
                                      container.container.securityContext
                                        .readOnlyRootFilesystem
                                    }
                                    onChange={(v) => {
                                      container.container.securityContext!.readOnlyRootFilesystem =
                                        v.target.checked;
                                      updateReq();
                                    }}
                                  />
                                  <NumberInput
                                    label="Run as user (UID)"
                                    description="Numeric user ID used to run the container process."
                                    min={0}
                                    value={
                                      container.container.securityContext
                                        .runAsUser
                                    }
                                    onChange={(v) => {
                                      container.container.securityContext!.runAsUser =
                                        strToNum(v);
                                      updateReq();
                                    }}
                                  />
                                </Group>

                                <EditItem
                                  title="Capabilities"
                                  description="Add or drop Linux capabilities"
                                  onUnset={() => {
                                    container.container.securityContext!.capabilities =
                                      undefined;
                                    updateReq();
                                  }}
                                  obj={
                                    container.container.securityContext
                                      .capabilities
                                  }
                                  onSet={() => {
                                    container.container.securityContext!.capabilities =
                                      CoreP.Service_Spec_Config_Upstream_Container_SecurityContext_Capabilities.create();
                                    updateReq();
                                  }}
                                >
                                  {container.container.securityContext
                                    .capabilities && (
                                    <div>
                                      <ItemMessage
                                        title="Add"
                                        obj={
                                          container.container.securityContext
                                            .capabilities.add
                                        }
                                        isList
                                        onSet={() => {
                                          container.container.securityContext!.capabilities!.add =
                                            [""];
                                          updateReq();
                                        }}
                                        onAddListItem={() => {
                                          container.container.securityContext!.capabilities!.add.push(
                                            "",
                                          );
                                          updateReq();
                                        }}
                                      >
                                        {container.container.securityContext.capabilities.add.map(
                                          (x, idx) => (
                                            <div
                                              className="w-full flex mb-3"
                                              key={idx}
                                            >
                                              <CloseButton
                                                size="sm"
                                                variant="subtle"
                                                onClick={() => {
                                                  container.container.securityContext!.capabilities!.add.splice(
                                                    idx,
                                                    1,
                                                  );
                                                  updateReq();
                                                }}
                                              />
                                              <TextInput
                                                required
                                                label="Capability"
                                                description="Linux capability added to the container process."
                                                placeholder="NET_ADMIN"
                                                className="flex-1"
                                                value={
                                                  container.container
                                                    .securityContext!
                                                    .capabilities!.add[idx]
                                                }
                                                onChange={(v) => {
                                                  container.container.securityContext!.capabilities!.add[
                                                    idx
                                                  ] = v.target.value;
                                                  updateReq();
                                                }}
                                              />
                                            </div>
                                          ),
                                        )}
                                      </ItemMessage>

                                      <ItemMessage
                                        title="Drop"
                                        obj={
                                          container.container.securityContext
                                            .capabilities.drop
                                        }
                                        isList
                                        onSet={() => {
                                          container.container.securityContext!.capabilities!.drop =
                                            [""];
                                          updateReq();
                                        }}
                                        onAddListItem={() => {
                                          container.container.securityContext!.capabilities!.drop.push(
                                            "",
                                          );
                                          updateReq();
                                        }}
                                      >
                                        {container.container.securityContext.capabilities.drop.map(
                                          (x, idx) => (
                                            <div
                                              className="w-full flex mb-3"
                                              key={idx}
                                            >
                                              <CloseButton
                                                size="sm"
                                                variant="subtle"
                                                onClick={() => {
                                                  container.container.securityContext!.capabilities!.drop.splice(
                                                    idx,
                                                    1,
                                                  );
                                                  updateReq();
                                                }}
                                              />
                                              <TextInput
                                                required
                                                label="Capability"
                                                description="Linux capability removed from the container process."
                                                placeholder="ALL"
                                                className="flex-1"
                                                value={
                                                  container.container
                                                    .securityContext!
                                                    .capabilities!.drop[idx]
                                                }
                                                onChange={(v) => {
                                                  container.container.securityContext!.capabilities!.drop[
                                                    idx
                                                  ] = v.target.value;
                                                  updateReq();
                                                }}
                                              />
                                            </div>
                                          ),
                                        )}
                                      </ItemMessage>
                                    </div>
                                  )}
                                </EditItem>
                              </div>
                            )}
                          </EditItem>

                          <ItemMessage
                            title="Volumes"
                            obj={container.container.volumes}
                            isList
                            onSet={() => {
                              container.container.volumes = [
                                CoreP.Service_Spec_Config_Upstream_Container_Volume.create(
                                  {
                                    type: {
                                      oneofKind: "emptyDir",
                                      emptyDir:
                                        CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
                                    },
                                  },
                                ),
                              ];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.volumes.push(
                                CoreP.Service_Spec_Config_Upstream_Container_Volume.create(
                                  {
                                    type: {
                                      oneofKind: "emptyDir",
                                      emptyDir:
                                        CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
                                    },
                                  },
                                ),
                              );
                              updateReq();
                            }}
                          >
                            {container.container.volumes.map((volume, idx) => (
                              <EditItem
                                key={idx}
                                obj={volume}
                                onUnset={() => {
                                  container.container.volumes.splice(idx, 1);
                                  updateReq();
                                }}
                              >
                                <Group grow>
                                  <TextInput
                                    required
                                    label="Name"
                                    placeholder="my-volume"
                                    description="Set a unique volume name"
                                    value={volume.name}
                                    onChange={(v) => {
                                      volume.name = v.target.value;
                                      updateReq();
                                    }}
                                  />
                                </Group>
                                <Tabs
                                  className="mb-4"
                                  value={volume.type.oneofKind}
                                  onChange={(v) => {
                                    match(v)
                                      .with("emptyDir", () => {
                                        volume.type = {
                                          oneofKind: "emptyDir",
                                          emptyDir:
                                            CoreP.Service_Spec_Config_Upstream_Container_Volume_EmptyDir.create(),
                                        };
                                      })
                                      .with("persistentVolumeClaim", () => {
                                        volume.type = {
                                          oneofKind: "persistentVolumeClaim",
                                          persistentVolumeClaim:
                                            CoreP.Service_Spec_Config_Upstream_Container_Volume_PersistentVolumeClaim.create(),
                                        };
                                      })
                                      .otherwise(() => {});
                                    updateReq();
                                  }}
                                >
                                  <Tabs.List>
                                    <Tabs.Tab value="emptyDir">
                                      Empty Dir
                                    </Tabs.Tab>
                                    <Tabs.Tab value="persistentVolumeClaim">
                                      Persistent Volume Claim
                                    </Tabs.Tab>
                                  </Tabs.List>

                                  <Tabs.Panel value="emptyDir">
                                    {match(volume.type)
                                      .when(
                                        (x) => x.oneofKind === "emptyDir",
                                        (emptyDir) => (
                                          <NumberInput
                                            label="Size limit (megabytes)"
                                            description="Maximum size of the emptyDir volume."
                                            placeholder="100"
                                            min={0}
                                            value={
                                              emptyDir.emptyDir
                                                .sizeLimitMegabytes
                                            }
                                            onChange={(v) => {
                                              emptyDir.emptyDir.sizeLimitMegabytes =
                                                strToNum(v);
                                              updateReq();
                                            }}
                                          />
                                        ),
                                      )
                                      .otherwise(() => (
                                        <></>
                                      ))}
                                  </Tabs.Panel>

                                  <Tabs.Panel value="persistentVolumeClaim">
                                    {match(volume.type)
                                      .when(
                                        (x) =>
                                          x.oneofKind ===
                                          "persistentVolumeClaim",
                                        (pvc) => (
                                          <TextInput
                                            required
                                            label="Claim name"
                                            description="Name of the PersistentVolumeClaim mounted by the container."
                                            placeholder="my-pvc"
                                            value={
                                              pvc.persistentVolumeClaim.name
                                            }
                                            onChange={(v) => {
                                              pvc.persistentVolumeClaim.name =
                                                v.target.value;
                                              updateReq();
                                            }}
                                          />
                                        ),
                                      )
                                      .otherwise(() => (
                                        <></>
                                      ))}
                                  </Tabs.Panel>
                                </Tabs>
                              </EditItem>
                            ))}
                          </ItemMessage>

                          <ItemMessage
                            title="Volume Mounts"
                            obj={container.container.volumeMounts}
                            isList
                            onSet={() => {
                              container.container.volumeMounts = [
                                CoreP.Service_Spec_Config_Upstream_Container_VolumeMount.create(),
                              ];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.volumeMounts.push(
                                CoreP.Service_Spec_Config_Upstream_Container_VolumeMount.create(),
                              );
                              updateReq();
                            }}
                          >
                            {container.container.volumeMounts.map(
                              (volumeMount, idx) => (
                                <EditItem
                                  key={idx}
                                  obj={volumeMount}
                                  onUnset={() => {
                                    container.container.volumeMounts.splice(
                                      idx,
                                      1,
                                    );
                                    updateReq();
                                  }}
                                >
                                  <Group grow>
                                    <TextInput
                                      required
                                      label="Name"
                                      placeholder="my-volume"
                                      description="Set the volume name to mount"
                                      value={volumeMount.name}
                                      onChange={(v) => {
                                        volumeMount.name = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label="Mount path"
                                      description="Path inside the container where the volume is mounted."
                                      placeholder="/data"
                                      value={volumeMount.mountPath}
                                      onChange={(v) => {
                                        volumeMount.mountPath = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      label="Sub path"
                                      description="Optional subdirectory within the volume to mount."
                                      placeholder="subdir"
                                      value={volumeMount.subPath}
                                      onChange={(v) => {
                                        volumeMount.subPath = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <Switch
                                      label="Read-only"
                                      description="Mount the volume without write access."
                                      checked={volumeMount.readOnly}
                                      onChange={(v) => {
                                        volumeMount.readOnly = v.target.checked;
                                        updateReq();
                                      }}
                                    />
                                  </Group>
                                </EditItem>
                              ),
                            )}
                          </ItemMessage>

                          <ContainerProbe
                            title="Liveness Probe"
                            probe={container.container.livenessProbe}
                            onUnset={() => {
                              container.container.livenessProbe = undefined;
                              updateReq();
                            }}
                            onSet={() => {
                              container.container.livenessProbe =
                                CoreP.Service_Spec_Config_Upstream_Container_Probe.create(
                                  {
                                    type: {
                                      oneofKind: "httpGet",
                                      httpGet:
                                        CoreP.Service_Spec_Config_Upstream_Container_Probe_HTTPGet.create(),
                                    },
                                  },
                                );
                              updateReq();
                            }}
                            onChange={() => updateReq()}
                          />

                          <ContainerProbe
                            title="Readiness Probe"
                            probe={container.container.readinessProbe}
                            onUnset={() => {
                              container.container.readinessProbe = undefined;
                              updateReq();
                            }}
                            onSet={() => {
                              container.container.readinessProbe =
                                CoreP.Service_Spec_Config_Upstream_Container_Probe.create(
                                  {
                                    type: {
                                      oneofKind: "httpGet",
                                      httpGet:
                                        CoreP.Service_Spec_Config_Upstream_Container_Probe_HTTPGet.create(),
                                    },
                                  },
                                );
                              updateReq();
                            }}
                            onChange={() => updateReq()}
                          />
                        </div>
                      );
                    },
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </Tabs.Panel>

              <Tabs.Panel value="loadbalance">
                {match(req.upstream.type)
                  .when(
                    (x) => x.oneofKind === "loadbalance",
                    (loadbalance) => (
                      <ItemMessage
                        title="Endpoints"
                        obj={loadbalance.loadbalance.endpoints}
                        isList
                        onSet={() => {
                          loadbalance.loadbalance.endpoints = [
                            CoreP.Service_Spec_Config_Upstream_Loadbalance_Endpoint.create(),
                          ];
                          updateReq();
                        }}
                        onAddListItem={() => {
                          loadbalance.loadbalance.endpoints.push(
                            CoreP.Service_Spec_Config_Upstream_Loadbalance_Endpoint.create(),
                          );
                          updateReq();
                        }}
                      >
                        {loadbalance.loadbalance.endpoints.map(
                          (endpoint, idx) => (
                            <EditItem
                              key={idx}
                              obj={endpoint}
                              onUnset={() => {
                                loadbalance.loadbalance.endpoints.splice(
                                  idx,
                                  1,
                                );
                                updateReq();
                              }}
                            >
                              <Group grow>
                                <TextInput
                                  required
                                  label="URL"
                                  description="The upstream canonical URL"
                                  placeholder="https://example.com"
                                  value={endpoint.url}
                                  onChange={(v) => {
                                    endpoint.url = v.target.value;
                                    updateReq();
                                  }}
                                />
                                <SelectResource
                                  api="core"
                                  kind="User"
                                  defaultValue={endpoint.user}
                                  label="Serve by User"
                                  clearable
                                  description="Serve from a connected client by a User"
                                  onChange={(v) => {
                                    endpoint.user = v?.metadata?.name ?? "";
                                    updateReq();
                                  }}
                                />
                              </Group>
                            </EditItem>
                          ),
                        )}
                      </ItemMessage>
                    ),
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </Tabs.Panel>
            </Tabs>
          </>
        )}
      </EditItem>

      {req.type.oneofKind !== undefined && (
        <EditItem
          title={configTypeTitle(req.type.oneofKind)}
          description="Configure settings specific to this Service mode"
          obj={req.type}
          onUnset={() => {}}
          noDelete
        >
          {match(req.type)
            .when(
              (x) => x.oneofKind === `kubernetes`,
              (kubernetes) => {
                return (
                  <div>
                    <Tabs
                      value={kubernetes.kubernetes.type.oneofKind}
                      onChange={(v) => {
                        match(v)
                          .with("kubeconfig", () => {
                            match(init.type).when(
                              (x) => x.oneofKind === `kubernetes`,
                              (x) => {
                                match(x.kubernetes)
                                  .when(
                                    (x) => x.type.oneofKind === `kubeconfig`,
                                    (x) => {
                                      kubernetes.kubernetes.type =
                                        structuredClone(x.type);
                                    },
                                  )
                                  .otherwise(() => {
                                    kubernetes.kubernetes.type = {
                                      oneofKind: "kubeconfig",
                                      kubeconfig:
                                        CoreP.Service_Spec_Config_Kubernetes_Kubeconfig.create(
                                          {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        ),
                                    };
                                  });
                              },
                            );

                            updateReq();
                          })
                          .with("bearerToken", () => {
                            match(init.type).when(
                              (x) => x.oneofKind === `kubernetes`,
                              (x) => {
                                match(x.kubernetes)
                                  .when(
                                    (x) => x.type.oneofKind === `bearerToken`,
                                    (x) => {
                                      kubernetes.kubernetes.type =
                                        structuredClone(x.type);
                                    },
                                  )
                                  .otherwise(() => {
                                    kubernetes.kubernetes.type = {
                                      oneofKind: "bearerToken",
                                      bearerToken:
                                        CoreP.Service_Spec_Config_Kubernetes_BearerToken.create(
                                          {
                                            type: {
                                              oneofKind: `fromSecret`,
                                              fromSecret: "",
                                            },
                                          },
                                        ),
                                    };
                                  });
                              },
                            );

                            updateReq();
                          })
                          .with("clientCertificate", () => {
                            match(init.type).when(
                              (x) => x.oneofKind === `kubernetes`,
                              (x) => {
                                match(x.kubernetes)
                                  .when(
                                    (x) =>
                                      x.type.oneofKind === `clientCertificate`,
                                    (x) => {
                                      kubernetes.kubernetes.type =
                                        structuredClone(x.type);
                                    },
                                  )
                                  .otherwise(() => {
                                    kubernetes.kubernetes.type = {
                                      oneofKind: "clientCertificate",
                                      clientCertificate:
                                        CoreP.Service_Spec_Config_ClientCertificate.create(),
                                    };
                                  });
                              },
                            );

                            updateReq();
                          });
                      }}
                    >
                      <Tabs.List>
                        <Tabs.Tab value="kubeconfig">Kubeconfig</Tabs.Tab>
                        <Tabs.Tab value="bearerToken">Bearer Token</Tabs.Tab>
                      </Tabs.List>
                      <Tabs.Panel value="kubeconfig">
                        {match(kubernetes.kubernetes.type)
                          .when(
                            (x) => x.oneofKind === `kubeconfig`,
                            (kubeconfig) => {
                              return (
                                <Group grow>
                                  <SelectResource
                                    api="core"
                                    kind="Secret"
                                    description="Secret containing the Kubernetes kubeconfig."
                                    defaultValue={
                                      kubeconfig.kubeconfig.type.oneofKind ===
                                      `fromSecret`
                                        ? kubeconfig.kubeconfig.type.fromSecret
                                        : undefined
                                    }
                                    onChange={(v) => {
                                      match(kubeconfig.kubeconfig.type).when(
                                        (x) => x.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret =
                                            v?.metadata?.name ?? "";
                                        },
                                      );

                                      updateReq();
                                    }}
                                  />

                                  <TextInput
                                    label="Context"
                                    description="Set a context name in the Kubeconfig"
                                    placeholder="context-1"
                                    value={kubeconfig.kubeconfig.context}
                                    onChange={(v) => {
                                      kubeconfig.kubeconfig.context =
                                        v.target.value;
                                      updateReq();
                                    }}
                                  />
                                </Group>
                              );
                            },
                          )
                          .otherwise(() => (
                            <></>
                          ))}
                      </Tabs.Panel>

                      <Tabs.Panel value="bearerToken">
                        {match(kubernetes.kubernetes.type)
                          .when(
                            (x) => x.oneofKind === `bearerToken`,
                            (bearerToken) => {
                              return (
                                <div className="w-full">
                                  <SelectResource
                                    api="core"
                                    kind="Secret"
                                    description="Secret containing the Kubernetes bearer token."
                                    defaultValue={
                                      bearerToken.bearerToken.type.oneofKind ===
                                      `fromSecret`
                                        ? bearerToken.bearerToken.type
                                            .fromSecret
                                        : undefined
                                    }
                                    onChange={(v) => {
                                      match(bearerToken.bearerToken.type).when(
                                        (x) => x.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret =
                                            v?.metadata?.name ?? "";
                                        },
                                      );

                                      updateReq();
                                    }}
                                  />
                                </div>
                              );
                            },
                          )
                          .otherwise(() => (
                            <></>
                          ))}
                      </Tabs.Panel>
                    </Tabs>
                  </div>
                );
              },
            )
            .when(
              (x) => x.oneofKind === `mcp`,
              (mcp) => (
                <MCPConfigEditor
                  config={mcp.mcp}
                  onChange={updateReq}
                />
              ),
            )
            .when(
              (x) => x.oneofKind === `llm`,
              (llm) => (
                <LLMConfigEditor
                  config={llm.llm}
                  onChange={updateReq}
                />
              ),
            )
            .when(
              (x) => x.oneofKind === `http`,
              (http) => {
                return (
                  <div className="w-full">
                    <Group grow>
                      <Switch
                        label="HTTP 2/0 Upstream"
                        checked={http.http.isUpstreamHTTP2}
                        description="Connect to the upstream over HTTP 2/0"
                        onChange={(v) => {
                          http.http.isUpstreamHTTP2 = v.target.checked;
                          updateReq();
                        }}
                      />

                      <Switch
                        label="Listen over HTTP 2/0"
                        checked={http.http.listenHTTP2}
                        description="Force the Service to listen over HTTP 2/0"
                        onChange={(v) => {
                          http.http.listenHTTP2 = v.target.checked;
                          updateReq();
                        }}
                      />

                      <Switch
                        label="Enable Request Buffering"
                        checked={http.http.enableRequestBuffering}
                        description="Buffer the entire request's body before sending to the upstream"
                        onChange={(v) => {
                          http.http.enableRequestBuffering = v.target.checked;
                          updateReq();
                        }}
                      />
                    </Group>

                    <EditItem
                      title="Headers"
                      description="Set Request/Response header related configs"
                      onUnset={() => {
                        http.http.header = undefined;
                        updateReq();
                      }}
                      obj={http.http.header}
                      onSet={() => {
                        http.http.header =
                          CoreP.Service_Spec_Config_HTTP_Header.create();

                        updateReq();
                      }}
                    >
                      {http.http.header && (
                        <div>
                          <Group grow>
                            <Select
                              label="Forwarded Headers Mode"
                              required
                              description="Obfuscate, drop or pass the the X-Forwarded-* headers to the upstream"
                              data={[
                                {
                                  label: "Obfuscate",
                                  value:
                                    CoreP
                                      .Service_Spec_Config_HTTP_Header_ForwardedMode[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Header_ForwardedMode
                                        .OBFUSCATE
                                    ],
                                },
                                {
                                  label: "Transparent",
                                  value:
                                    CoreP
                                      .Service_Spec_Config_HTTP_Header_ForwardedMode[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Header_ForwardedMode
                                        .TRANSPARENT
                                    ],
                                },
                                {
                                  label: "Drop",
                                  value:
                                    CoreP
                                      .Service_Spec_Config_HTTP_Header_ForwardedMode[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Header_ForwardedMode
                                        .DROP
                                    ],
                                },
                              ]}
                              value={
                                CoreP
                                  .Service_Spec_Config_HTTP_Header_ForwardedMode[
                                  http.http.header.forwardedMode
                                ] ??
                                CoreP
                                  .Service_Spec_Config_HTTP_Header_ForwardedMode[
                                  CoreP
                                    .Service_Spec_Config_HTTP_Header_ForwardedMode
                                    .OBFUSCATE
                                ]
                              }
                              onChange={(v) => {
                                if (!v) return;
                                http.http.header!.forwardedMode =
                                  CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode[
                                    v as "OBFUSCATE"
                                  ];

                                updateReq();
                              }}
                            />

                            <Select
                              label="Authorization Header Mode"
                              required
                              description="Explicitly delete or pass the downstream Authorization request header"
                              data={[
                                {
                                  label: "Delete",
                                  value:
                                    CoreP
                                      .Service_Spec_Config_HTTP_Header_AuthorizationMode[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Header_AuthorizationMode
                                        .DELETE
                                    ],
                                },
                                {
                                  label: "Pass",
                                  value:
                                    CoreP
                                      .Service_Spec_Config_HTTP_Header_AuthorizationMode[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Header_AuthorizationMode
                                        .PASS
                                    ],
                                },
                              ]}
                              value={
                                CoreP
                                  .Service_Spec_Config_HTTP_Header_AuthorizationMode[
                                  http.http.header.authorizationMode
                                ]
                              }
                              onChange={(v) => {
                                if (!v) return;
                                http.http.header!.authorizationMode =
                                  CoreP.Service_Spec_Config_HTTP_Header_AuthorizationMode[
                                    v as "PASS"
                                  ];

                                updateReq();
                              }}
                            />
                          </Group>
                          <ItemMessage
                            title="Add Request Headers"
                            obj={http.http.header.addRequestHeaders}
                            isList
                            onSet={() => {
                              http.http.header!.addRequestHeaders = [
                                CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(
                                  {
                                    key: "",
                                    type: {
                                      oneofKind: "value",
                                      value: "",
                                    },
                                  },
                                ),
                              ];

                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.header!.addRequestHeaders.push(
                                CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(
                                  {
                                    key: "",
                                    type: {
                                      oneofKind: "value",
                                      value: "",
                                    },
                                  },
                                ),
                              );

                              updateReq();
                            }}
                          >
                            {http.http.header!.addRequestHeaders.map(
                              (x, idx) => (
                                <div className="w-full flex mb-3" key={idx}>
                                  <CloseButton
                                    size={"sm"}
                                    variant="subtle"
                                    className="mr-2"
                                    onClick={() => {
                                      http.http.header!.addRequestHeaders.splice(
                                        idx,
                                        1,
                                      );
                                      updateReq();
                                    }}
                                  ></CloseButton>
                                  <Group className="flex w-full" grow>
                                    <TextInput
                                      required
                                      label="Key"
                                      description="Set the Header key"
                                      placeholder="MY_KEY"
                                      value={
                                        http.http.header!.addRequestHeaders[idx]
                                          .key
                                      }
                                      onChange={(v) => {
                                        http.http.header!.addRequestHeaders[
                                          idx
                                        ].key = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <Select
                                      label="Value type"
                                      description="Use a literal header value or evaluate it with CEL."
                                      data={[
                                        { label: "Literal", value: "value" },
                                        { label: "Eval (CEL)", value: "eval" },
                                      ]}
                                      value={
                                        http.http.header!.addRequestHeaders[idx]
                                          .type.oneofKind ?? "value"
                                      }
                                      onChange={(v) => {
                                        if (!v) return;
                                        http.http.header!.addRequestHeaders[
                                          idx
                                        ].type =
                                          v === "eval"
                                            ? { oneofKind: "eval", eval: "" }
                                            : { oneofKind: "value", value: "" };
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label={
                                        http.http.header!.addRequestHeaders[idx]
                                          .type.oneofKind === "eval"
                                          ? "CEL expression"
                                          : "Value"
                                      }
                                      description="Set the Header value"
                                      placeholder="my-value"
                                      value={match(
                                        http.http.header!.addRequestHeaders[idx]
                                          .type,
                                      )
                                        .when(
                                          (v) => v.oneofKind === `value`,
                                          (v) => v.value,
                                        )
                                        .when(
                                          (v) => v.oneofKind === `eval`,
                                          (v) => v.eval,
                                        )
                                        .otherwise(() => undefined)}
                                      onChange={(val) => {
                                        match(
                                          http.http.header!.addRequestHeaders[
                                            idx
                                          ].type,
                                        )
                                          .when(
                                            (v) => v.oneofKind === `value`,
                                            (v) => {
                                              v.value = val.target.value;
                                            },
                                          )
                                          .when(
                                            (v) => v.oneofKind === `eval`,
                                            (v) => {
                                              v.eval = val.target.value;
                                            },
                                          );

                                        updateReq();
                                      }}
                                    />
                                  </Group>
                                </div>
                              ),
                            )}
                          </ItemMessage>

                          <ItemMessage
                            title="Add Response Headers"
                            obj={http.http.header.addResponseHeaders}
                            isList
                            onSet={() => {
                              http.http.header!.addResponseHeaders = [
                                CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(
                                  {
                                    key: "",
                                    type: {
                                      oneofKind: "value",
                                      value: "",
                                    },
                                  },
                                ),
                              ];

                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.header!.addResponseHeaders.push(
                                CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(
                                  {
                                    key: "",
                                    type: {
                                      oneofKind: "value",
                                      value: "",
                                    },
                                  },
                                ),
                              );

                              updateReq();
                            }}
                          >
                            {http.http.header!.addResponseHeaders.map(
                              (x, idx) => (
                                <div className="w-full flex mb-3" key={idx}>
                                  <CloseButton
                                    size={"sm"}
                                    variant="subtle"
                                    className="mr-2"
                                    onClick={() => {
                                      http.http.header!.addResponseHeaders.splice(
                                        idx,
                                        1,
                                      );
                                      updateReq();
                                    }}
                                  ></CloseButton>
                                  <Group className="flex w-full" grow>
                                    <TextInput
                                      required
                                      label="Key"
                                      description="Set the Header key"
                                      placeholder="MY_KEY"
                                      value={
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].key
                                      }
                                      onChange={(v) => {
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].key = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    <Select
                                      label="Value type"
                                      description="Use a literal header value or evaluate it with CEL."
                                      data={[
                                        { label: "Literal", value: "value" },
                                        { label: "Eval (CEL)", value: "eval" },
                                      ]}
                                      value={
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].type.oneofKind ?? "value"
                                      }
                                      onChange={(v) => {
                                        if (!v) return;
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].type =
                                          v === "eval"
                                            ? { oneofKind: "eval", eval: "" }
                                            : { oneofKind: "value", value: "" };
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label={
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].type.oneofKind === "eval"
                                          ? "CEL expression"
                                          : "Value"
                                      }
                                      description="Set the Header value"
                                      placeholder="my-value"
                                      value={match(
                                        http.http.header!.addResponseHeaders[
                                          idx
                                        ].type,
                                      )
                                        .when(
                                          (v) => v.oneofKind === `value`,
                                          (v) => v.value,
                                        )
                                        .when(
                                          (v) => v.oneofKind === `eval`,
                                          (v) => v.eval,
                                        )
                                        .otherwise(() => undefined)}
                                      onChange={(val) => {
                                        let f = req.type as {
                                          oneofKind: "http";
                                          http: CoreP.Service_Spec_Config_HTTP;
                                        };

                                        match(
                                          f.http.header!.addResponseHeaders[idx]
                                            .type,
                                        )
                                          .when(
                                            (v) => v.oneofKind === `value`,
                                            (v) => {
                                              v.value = val.target.value;
                                            },
                                          )
                                          .when(
                                            (v) => v.oneofKind === `eval`,
                                            (v) => {
                                              v.eval = val.target.value;
                                            },
                                          );
                                        updateReq();
                                      }}
                                    />
                                  </Group>
                                </div>
                              ),
                            )}
                          </ItemMessage>

                          <ItemMessage
                            title="Remove Request Headers"
                            obj={http.http.header.removeRequestHeaders}
                            isList
                            onSet={() => {
                              http.http.header!.removeRequestHeaders = [""];

                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.header!.removeRequestHeaders.push("");

                              updateReq();
                            }}
                          >
                            {http.http.header!.removeRequestHeaders.map(
                              (x, idx) => (
                                <div className="w-full flex mb-3" key={idx}>
                                  <CloseButton
                                    size={"sm"}
                                    variant="subtle"
                                    className="mr-2"
                                    onClick={() => {
                                      http.http.header!.removeRequestHeaders.splice(
                                        idx,
                                        1,
                                      );
                                      updateReq();
                                    }}
                                  ></CloseButton>
                                  <Group className="flex w-full" grow>
                                    <TextInput
                                      required
                                      label="Key"
                                      description="Set the Header key"
                                      placeholder="MY_KEY"
                                      value={
                                        http.http.header!.removeRequestHeaders[
                                          idx
                                        ]
                                      }
                                      onChange={(v) => {
                                        http.http.header!.removeRequestHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </Group>
                                </div>
                              ),
                            )}
                          </ItemMessage>

                          <ItemMessage
                            title="Remove Response Headers"
                            obj={http.http.header.removeResponseHeaders}
                            isList
                            onSet={() => {
                              http.http.header!.removeResponseHeaders = [""];

                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.header!.removeResponseHeaders.push("");

                              updateReq();
                            }}
                          >
                            {http.http.header!.removeResponseHeaders.map(
                              (x, idx) => (
                                <div className="w-full flex mb-3" key={idx}>
                                  <CloseButton
                                    size={"sm"}
                                    variant="subtle"
                                    className="mr-2"
                                    onClick={() => {
                                      http.http.header!.removeResponseHeaders.splice(
                                        idx,
                                        1,
                                      );
                                      updateReq();
                                    }}
                                  ></CloseButton>
                                  <Group className="flex w-full" grow>
                                    <TextInput
                                      required
                                      label="Key"
                                      description="Set the Header key"
                                      placeholder="MY_KEY"
                                      value={
                                        http.http.header!.removeResponseHeaders[
                                          idx
                                        ]
                                      }
                                      onChange={(v) => {
                                        http.http.header!.removeResponseHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </Group>
                                </div>
                              ),
                            )}
                          </ItemMessage>

                          <EditItem
                            title="Host header"
                            description="Set the Host header related configs"
                            onUnset={() => {
                              http.http.header!.host = undefined;
                              updateReq();
                            }}
                            obj={http.http.header.host}
                            onSet={() => {
                              http.http.header!.host =
                                CoreP.Service_Spec_Config_HTTP_Header_Host.create(
                                  {
                                    type: {
                                      oneofKind: "preserve",
                                      preserve: true,
                                    },
                                  },
                                );
                              updateReq();
                            }}
                          >
                            {http.http.header.host && (
                              <Tabs
                                value={http.http.header.host.type.oneofKind}
                                onChange={(v) => {
                                  match(v)
                                    .with("preserve", () => {
                                      http.http.header!.host!.type = {
                                        oneofKind: "preserve",
                                        preserve: true,
                                      };
                                    })
                                    .with("value", () => {
                                      http.http.header!.host!.type = {
                                        oneofKind: "value",
                                        value: "",
                                      };
                                    })
                                    .with("eval", () => {
                                      http.http.header!.host!.type = {
                                        oneofKind: "eval",
                                        eval: "",
                                      };
                                    })
                                    .otherwise(() => {});
                                  updateReq();
                                }}
                              >
                                <Tabs.List className="mb-2">
                                  <Tabs.Tab value="preserve">Preserve</Tabs.Tab>
                                  <Tabs.Tab value="value">Value</Tabs.Tab>
                                  <Tabs.Tab value="eval">Eval (CEL)</Tabs.Tab>
                                </Tabs.List>

                                <Tabs.Panel value="preserve">
                                  {match(http.http.header.host.type)
                                    .when(
                                      (x) => x.oneofKind === "preserve",
                                      (preserve) => (
                                        <Switch
                                          label="Preserve host header"
                                          description="Preserve the downstream Host header to the upstream"
                                          checked={preserve.preserve}
                                          onChange={(v) => {
                                            preserve.preserve =
                                              v.target.checked;
                                            updateReq();
                                          }}
                                        />
                                      ),
                                    )
                                    .otherwise(() => (
                                      <></>
                                    ))}
                                </Tabs.Panel>

                                <Tabs.Panel value="value">
                                  {match(http.http.header.host.type)
                                    .when(
                                      (x) => x.oneofKind === "value",
                                      (value) => (
                                        <TextInput
                                          label="Host value"
                                          description="Set a fixed Host header value sent to the upstream"
                                          placeholder="example.com"
                                          value={value.value}
                                          onChange={(v) => {
                                            value.value = v.target.value;
                                            updateReq();
                                          }}
                                        />
                                      ),
                                    )
                                    .otherwise(() => (
                                      <></>
                                    ))}
                                </Tabs.Panel>

                                <Tabs.Panel value="eval">
                                  {match(http.http.header.host.type)
                                    .when(
                                      (x) => x.oneofKind === "eval",
                                      (evalType) => (
                                        <TextInput
                                          label="Host eval (CEL)"
                                          description="Set a CEL expression that evaluates to the Host header value"
                                          placeholder='ctx.service.metadata.name + ".internal"'
                                          value={evalType.eval}
                                          onChange={(v) => {
                                            evalType.eval = v.target.value;
                                            updateReq();
                                          }}
                                        />
                                      ),
                                    )
                                    .otherwise(() => (
                                      <></>
                                    ))}
                                </Tabs.Panel>
                              </Tabs>
                            )}
                          </EditItem>
                        </div>
                      )}
                    </EditItem>

                    <EditItem
                      title="Path"
                      description="Set the request path related configs"
                      onUnset={() => {
                        http.http.path = undefined;
                        updateReq();
                      }}
                      obj={http.http.path}
                      onSet={() => {
                        http.http.path =
                          CoreP.Service_Spec_Config_HTTP_Path.create();

                        updateReq();
                      }}
                    >
                      {http.http.path && (
                        <div>
                          <Group grow>
                            <TextInput
                              label="Add prefix"
                              description="Add Prefix to the request path"
                              placeholder="/api/v1"
                              value={http.http.path.addPrefix}
                              onChange={(v) => {
                                http.http.path!.addPrefix = v.target.value;
                                updateReq();
                              }}
                            />
                            <TextInput
                              label="Remove prefix"
                              description="Remove prefix from the request path"
                              placeholder="/api/v2"
                              value={http.http.path.removePrefix}
                              onChange={(v) => {
                                http.http.path!.removePrefix = v.target.value;
                                updateReq();
                              }}
                            />
                          </Group>
                        </div>
                      )}
                    </EditItem>

                    <EditItem
                      title="Response"
                      description="Set a direct response returned by the Service"
                      onUnset={() => {
                        http.http.response = undefined;
                        updateReq();
                      }}
                      obj={http.http.response}
                      onSet={() => {
                        http.http.response =
                          CoreP.Service_Spec_Config_HTTP_Response.create({
                            type: {
                              oneofKind: "direct",
                              direct:
                                CoreP.Service_Spec_Config_HTTP_Response_Direct.create(
                                  {
                                    type: { oneofKind: "inline", inline: "" },
                                  },
                                ),
                            },
                          });
                        updateReq();
                      }}
                    >
                      {http.http.response &&
                        match(http.http.response.type)
                          .when(
                            (x) => x.oneofKind === "direct",
                            (direct) => (
                              <div>
                                <Group grow>
                                  <NumberInput
                                    label="Status Code"
                                    description="HTTP status code to return"
                                    min={100}
                                    max={599}
                                    value={direct.direct.statusCode}
                                    onChange={(v) => {
                                      direct.direct.statusCode = strToNum(v);
                                      updateReq();
                                    }}
                                  />
                                  <TextInput
                                    label="Content Type"
                                    description="Set the response Content-Type header"
                                    placeholder="application/json"
                                    value={direct.direct.contentType}
                                    onChange={(v) => {
                                      direct.direct.contentType =
                                        v.target.value;
                                      updateReq();
                                    }}
                                  />
                                </Group>

                                <Tabs
                                  className="mt-4"
                                  value={
                                    direct.direct.type.oneofKind ?? "inline"
                                  }
                                  onChange={(v) => {
                                    match(v)
                                      .with("inline", () => {
                                        direct.direct.type = {
                                          oneofKind: "inline",
                                          inline: "",
                                        };
                                      })
                                      .with("inlineBytes", () => {
                                        direct.direct.type = {
                                          oneofKind: "inlineBytes",
                                          inlineBytes: new Uint8Array(),
                                        };
                                      })
                                      .otherwise(() => {});
                                    updateReq();
                                  }}
                                >
                                  <Tabs.List>
                                    <Tabs.Tab value="inline">
                                      Inline Text
                                    </Tabs.Tab>
                                    <Tabs.Tab value="inlineBytes">
                                      Inline Bytes
                                    </Tabs.Tab>
                                  </Tabs.List>

                                  <Tabs.Panel value="inline">
                                    {match(direct.direct.type)
                                      .when(
                                        (x) => x.oneofKind === "inline",
                                        (inline) => (
                                          <div>
                                            <TextAreaCustom
                                              label="Inline response body"
                                              description="Body returned directly to the downstream client."
                                              placeholder='{ "message": "ok" }'
                                              value={inline.inline}
                                              onChange={(v) => {
                                                inline.inline = v ?? "";
                                                updateReq();
                                              }}
                                            />
                                          </div>
                                        ),
                                      )
                                      .otherwise(() => (
                                        <></>
                                      ))}
                                  </Tabs.Panel>

                                  <Tabs.Panel value="inlineBytes">
                                    {match(direct.direct.type)
                                      .when(
                                        (x) => x.oneofKind === "inlineBytes",
                                        (inlineBytes) => (
                                          <div>
                                            <TextAreaCustom
                                              label="Inline response bytes"
                                              description="Raw bytes returned directly to the downstream client."
                                              placeholder="Raw bytes content"
                                              value={new TextDecoder().decode(
                                                inlineBytes.inlineBytes,
                                              )}
                                              onChange={(v) => {
                                                inlineBytes.inlineBytes =
                                                  new TextEncoder().encode(
                                                    v ?? "",
                                                  );
                                                updateReq();
                                              }}
                                            />
                                          </div>
                                        ),
                                      )
                                      .otherwise(() => (
                                        <></>
                                      ))}
                                  </Tabs.Panel>
                                </Tabs>
                              </div>
                            ),
                          )
                          .otherwise(() => <></>)}
                    </EditItem>

                    <EditItem
                      title="Body"
                      description="Set Request body related configs"
                      onUnset={() => {
                        http.http.body = undefined;
                        updateReq();
                      }}
                      obj={http.http.body}
                      onSet={() => {
                        http.http.body =
                          CoreP.Service_Spec_Config_HTTP_Body.create();
                        updateReq();
                      }}
                    >
                      {http.http.body && (
                        <Group grow>
                          <NumberInput
                            label="Mox body size"
                            placeholder="8080"
                            description="Set the max request body size in Bytes"
                            min={0}
                            value={http.http.body.maxRequestSize}
                            onChange={(v) => {
                              http.http.body!.maxRequestSize = strToNum(v);
                              updateReq();
                            }}
                          />

                          <Select
                            label="Body Content Mode"
                            clearable
                            description="Set the request body mode (e.g. JSON)"
                            data={[
                              {
                                label: "JSON",
                                value:
                                  CoreP.Service_Spec_Config_HTTP_Body_Mode[
                                    CoreP.Service_Spec_Config_HTTP_Body_Mode
                                      .JSON
                                  ],
                              },
                            ]}
                            value={
                              CoreP.Service_Spec_Config_HTTP_Body_Mode[
                                http.http.body!.mode
                              ]
                            }
                            onChange={(v) => {
                              if (!v) return;
                              http.http.body!.mode =
                                CoreP.Service_Spec_Config_HTTP_Body_Mode[
                                  v as "JSON"
                                ];
                              updateReq();
                            }}
                          />
                        </Group>
                      )}
                    </EditItem>
                    <EditItem
                      title="Authentication"
                      description="Set authentication-related info required by the upstream to provide secretless access"
                      onUnset={() => {
                        http.http.auth = undefined;
                        updateReq();
                      }}
                      obj={http.http.auth}
                      onSet={() => {
                        http.http.auth =
                          CoreP.Service_Spec_Config_HTTP_Auth.create({
                            type: {
                              oneofKind: `bearer`,
                              bearer: {
                                type: {
                                  oneofKind: `fromSecret`,
                                  fromSecret: ``,
                                },
                              },
                            },
                          });
                        updateReq();
                      }}
                    >
                      {http.http.auth && (
                        <div>
                          <Tabs
                            value={http.http.auth!.type.oneofKind}
                            onChange={(v) => {
                              match(v)
                                .with("bearer", () => {
                                  match(
                                    init.type.oneofKind === `http`
                                      ? init.type.http.auth?.type.oneofKind
                                      : undefined,
                                  )
                                    .with(`bearer`, () => {
                                      http.http.auth!.type =
                                        init.type.oneofKind === `http`
                                          ? structuredClone(
                                              init.type.http.auth!.type,
                                            )
                                          : {
                                              oneofKind: "bearer",
                                              bearer:
                                                CoreP.Service_Spec_Config_HTTP_Auth_Bearer.create(
                                                  {
                                                    type: {
                                                      oneofKind: "fromSecret",
                                                      fromSecret: "",
                                                    },
                                                  },
                                                ),
                                            };
                                    })
                                    .otherwise(() => {
                                      http.http.auth!.type = {
                                        oneofKind: "bearer",
                                        bearer:
                                          CoreP.Service_Spec_Config_HTTP_Auth_Bearer.create(
                                            {
                                              type: {
                                                oneofKind: "fromSecret",
                                                fromSecret: "",
                                              },
                                            },
                                          ),
                                      };
                                    });

                                  updateReq();
                                })
                                .with("basic", () => {
                                  match(
                                    init.type.oneofKind === `http`
                                      ? init.type.http.auth?.type.oneofKind
                                      : undefined,
                                  )
                                    .with(`basic`, () => {
                                      http.http.auth!.type =
                                        init.type.oneofKind === `http`
                                          ? structuredClone(
                                              init.type.http.auth!.type,
                                            )
                                          : {
                                              oneofKind: "basic",
                                              basic:
                                                CoreP.Service_Spec_Config_HTTP_Auth_Basic.create(
                                                  {
                                                    password: {
                                                      type: {
                                                        oneofKind: "fromSecret",
                                                        fromSecret: "",
                                                      },
                                                    },
                                                  },
                                                ),
                                            };
                                    })
                                    .otherwise(() => {
                                      http.http.auth!.type = {
                                        oneofKind: "basic",
                                        basic:
                                          CoreP.Service_Spec_Config_HTTP_Auth_Basic.create(
                                            {
                                              password: {
                                                type: {
                                                  oneofKind: "fromSecret",
                                                  fromSecret: "",
                                                },
                                              },
                                            },
                                          ),
                                      };
                                    });

                                  updateReq();
                                })
                                .with("oauth2ClientCredentials", () => {
                                  let f = item.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };
                                  let ff = req.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };

                                  match(
                                    init.type.oneofKind === `http`
                                      ? init.type.http.auth?.type.oneofKind
                                      : undefined,
                                  )
                                    .with(`oauth2ClientCredentials`, () => {
                                      ff.http.auth!.type =
                                        init!.type.oneofKind === `http`
                                          ? structuredClone(
                                              init!.type.http.auth!.type,
                                            )
                                          : {
                                              oneofKind:
                                                "oauth2ClientCredentials",
                                              oauth2ClientCredentials:
                                                CoreP.Service_Spec_Config_HTTP_Auth_OAuth2ClientCredentials.create(
                                                  {
                                                    clientSecret: {
                                                      type: {
                                                        oneofKind: "fromSecret",
                                                        fromSecret: "",
                                                      },
                                                    },
                                                  },
                                                ),
                                            };
                                    })
                                    .otherwise(() => {
                                      ff.http.auth!.type = {
                                        oneofKind: "oauth2ClientCredentials",
                                        oauth2ClientCredentials:
                                          CoreP.Service_Spec_Config_HTTP_Auth_OAuth2ClientCredentials.create(
                                            {
                                              clientSecret: {
                                                type: {
                                                  oneofKind: "fromSecret",
                                                  fromSecret: "",
                                                },
                                              },
                                            },
                                          ),
                                      };
                                    });

                                  updateReq();
                                })
                                .with("custom", () => {
                                  let f = item.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };
                                  let ff = req.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };

                                  match(
                                    init.type.oneofKind === `http`
                                      ? init.type.http.auth?.type.oneofKind
                                      : undefined,
                                  )
                                    .with(`custom`, () => {
                                      ff.http.auth!.type =
                                        init!.type.oneofKind === `http`
                                          ? structuredClone(
                                              init!.type.http.auth!.type,
                                            )
                                          : {
                                              oneofKind: "custom",
                                              custom:
                                                CoreP.Service_Spec_Config_HTTP_Auth_Custom.create(
                                                  {
                                                    value: {
                                                      type: {
                                                        oneofKind: "fromSecret",
                                                        fromSecret: "",
                                                      },
                                                    },
                                                  },
                                                ),
                                            };
                                    })
                                    .otherwise(() => {
                                      ff.http.auth!.type = {
                                        oneofKind: "custom",
                                        custom:
                                          CoreP.Service_Spec_Config_HTTP_Auth_Custom.create(
                                            {
                                              value: {
                                                type: {
                                                  oneofKind: "fromSecret",
                                                  fromSecret: "",
                                                },
                                              },
                                            },
                                          ),
                                      };
                                    });

                                  updateReq();
                                })
                                .with(`sigv4`, () => {
                                  let f = item.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };
                                  let ff = req.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };

                                  match(
                                    init.type.oneofKind === `http`
                                      ? init.type.http.auth?.type.oneofKind
                                      : undefined,
                                  )
                                    .with(`sigv4`, () => {
                                      ff.http.auth!.type =
                                        init.type.oneofKind === `http`
                                          ? structuredClone(
                                              init.type.http.auth!.type,
                                            )
                                          : {
                                              oneofKind: "sigv4",
                                              sigv4:
                                                CoreP.Service_Spec_Config_HTTP_Auth_Sigv4.create(
                                                  {
                                                    secretAccessKey: {
                                                      type: {
                                                        oneofKind: "fromSecret",
                                                        fromSecret: "",
                                                      },
                                                    },
                                                  },
                                                ),
                                            };
                                    })
                                    .otherwise(() => {
                                      ff.http.auth!.type = {
                                        oneofKind: "sigv4",
                                        sigv4:
                                          CoreP.Service_Spec_Config_HTTP_Auth_Sigv4.create(
                                            {
                                              secretAccessKey: {
                                                type: {
                                                  oneofKind: "fromSecret",
                                                  fromSecret: "",
                                                },
                                              },
                                            },
                                          ),
                                      };
                                    });

                                  updateReq();
                                });
                            }}
                          >
                            <Tabs.List>
                              <Tabs.Tab value="bearer">
                                Bearer Authentication
                              </Tabs.Tab>
                              <Tabs.Tab value="basic">
                                Basic Authentication
                              </Tabs.Tab>
                              <Tabs.Tab value="oauth2ClientCredentials">
                                OAuth2 Client Credentials
                              </Tabs.Tab>
                              <Tabs.Tab value="custom">Custom Header</Tabs.Tab>
                              <Tabs.Tab value="sigv4">AWS SigV4</Tabs.Tab>
                            </Tabs.List>
                            <Tabs.Panel value="bearer">
                              {match(http.http.auth.type)
                                .when(
                                  (x) => x.oneofKind == `bearer`,
                                  (bearer) => {
                                    return (
                                      <div className="w-full">
                                        <SelectResource
                                          api="core"
                                          kind="Secret"
                                          label="Bearer access token Secret"
                                          description="Select the Secret of the bearer access token"
                                          defaultValue={
                                            bearer.bearer.type.oneofKind ===
                                            `fromSecret`
                                              ? bearer.bearer.type.fromSecret
                                              : undefined
                                          }
                                          onChange={(val) => {
                                            match(bearer.bearer.type).when(
                                              (x) =>
                                                x.oneofKind === `fromSecret`,
                                              (x) => {
                                                x.fromSecret =
                                                  val?.metadata?.name ?? "";
                                              },
                                            );

                                            updateReq();
                                          }}
                                        />
                                      </div>
                                    );
                                  },
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>
                            <Tabs.Panel value="oauth2ClientCredentials">
                              {match(http.http.auth.type)
                                .when(
                                  (x) =>
                                    x.oneofKind == `oauth2ClientCredentials`,
                                  (oauth2ClientCredentials) => {
                                    return (
                                      <Group grow>
                                        <TextInput
                                          required
                                          label="Client ID"
                                          description="OAuth2 client identifier sent to the token endpoint."
                                          placeholder="user1234"
                                          value={
                                            oauth2ClientCredentials
                                              .oauth2ClientCredentials.clientID
                                          }
                                          onChange={(v) => {
                                            oauth2ClientCredentials.oauth2ClientCredentials.clientID =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        {match(
                                          oauth2ClientCredentials
                                            .oauth2ClientCredentials
                                            .clientSecret?.type,
                                        )
                                          .when(
                                            (x) =>
                                              x?.oneofKind === `fromSecret`,
                                            (x) => {
                                              return (
                                                <SelectResource
                                                  api="core"
                                                  kind="Secret"
                                                  label="Client Secret"
                                                  description="Select the Secret of the OAuth2 client secret"
                                                  defaultValue={x.fromSecret}
                                                  onChange={(v) => {
                                                    x.fromSecret =
                                                      v?.metadata?.name ?? "";
                                                    updateReq();
                                                  }}
                                                />
                                              );
                                            },
                                          )
                                          .otherwise(() => (
                                            <></>
                                          ))}

                                      <TextInput
                                        required
                                        label="Token endpoint URL"
                                        description="OAuth2 endpoint used to obtain an upstream access token."
                                          placeholder="https://oauth2.example.com/token"
                                          value={
                                            oauth2ClientCredentials
                                              .oauth2ClientCredentials.tokenURL
                                          }
                                          onChange={(v) => {
                                            oauth2ClientCredentials.oauth2ClientCredentials.tokenURL =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                      </Group>
                                    );
                                  },
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="basic">
                              {match(http.http.auth.type)
                                .when(
                                  (x) => x.oneofKind == `basic`,
                                  (basic) => {
                                    return (
                                      <Group grow>
                                        <TextInput
                                          required
                                          label="Username"
                                          description="Username sent to the upstream for Basic authentication."
                                          placeholder="user1234"
                                          value={basic.basic.username}
                                          onChange={(v) => {
                                            basic.basic.username =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        {match(basic.basic.password?.type)
                                          .when(
                                            (x) =>
                                              x?.oneofKind === `fromSecret`,
                                            (x) => {
                                              return (
                                                <SelectResource
                                                  api="core"
                                                  kind="Secret"
                                                  label="Password Secret"
                                                  description="Select the Secret of the basic authentication password"
                                                  defaultValue={x.fromSecret}
                                                  onChange={(v) => {
                                                    x.fromSecret =
                                                      v?.metadata?.name ?? "";
                                                    updateReq();
                                                  }}
                                                />
                                              );
                                            },
                                          )
                                          .otherwise(() => (
                                            <></>
                                          ))}
                                      </Group>
                                    );
                                  },
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>
                            <Tabs.Panel value="custom">
                              {match(http.http.auth.type)
                                .when(
                                  (x) => x.oneofKind == `custom`,
                                  (custom) => {
                                    return (
                                      <Group grow>
                                        <TextInput
                                          required
                                          label="Header Name"
                                          description="Custom header used to carry the upstream credential."
                                          placeholder="X-CUSTOM-AUTH-HEADER"
                                          value={custom.custom.header}
                                          onChange={(v) => {
                                            custom.custom.header =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        {match(custom.custom.value?.type)
                                          .when(
                                            (x) =>
                                              x?.oneofKind === `fromSecret`,
                                            (x) => {
                                              return (
                                                <SelectResource
                                                  api="core"
                                                  kind="Secret"
                                                  label="Header value Secret"
                                                  description="Select the Secret of the header value"
                                                  defaultValue={x.fromSecret}
                                                  onChange={(v) => {
                                                    x.fromSecret =
                                                      v?.metadata?.name ?? "";
                                                    updateReq();
                                                  }}
                                                />
                                              );
                                            },
                                          )
                                          .otherwise(() => (
                                            <></>
                                          ))}
                                      </Group>
                                    );
                                  },
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="sigv4">
                              {match(http.http.auth.type)
                                .when(
                                  (x) => x.oneofKind == `sigv4`,
                                  (sigv4) => {
                                    return (
                                      <Group grow>
                                        <TextInput
                                          required
                                          label="Access Key ID"
                                          description="AWS access key ID used to sign upstream requests."
                                          placeholder="ABCDEDF123456"
                                          value={sigv4.sigv4.accessKeyID}
                                          onChange={(v) => {
                                            sigv4.sigv4.accessKeyID =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        <TextInput
                                          required
                                          label="Region"
                                          description="AWS region used to generate the SigV4 signature."
                                          placeholder="eu-west-1"
                                          value={sigv4.sigv4.region}
                                          onChange={(v) => {
                                            sigv4.sigv4.region = v.target.value;
                                            updateReq();
                                          }}
                                        />
                                        <TextInput
                                          required
                                          label="Service"
                                          description="AWS service name used to generate the SigV4 signature."
                                          placeholder="s3"
                                          value={sigv4.sigv4.service}
                                          onChange={(v) => {
                                            sigv4.sigv4.service =
                                              v.target.value;
                                            updateReq();
                                          }}
                                        />

                                        {match(
                                          sigv4.sigv4.secretAccessKey?.type,
                                        )
                                          .when(
                                            (x) =>
                                              x?.oneofKind === `fromSecret`,
                                            (x) => {
                                              return (
                                                <SelectResource
                                                  api="core"
                                                  kind="Secret"
                                                  label="Secret Access Key"
                                                  description="Set the Secret of the Sigv4 Secret Access Key"
                                                  defaultValue={x.fromSecret}
                                                  onChange={(v) => {
                                                    x.fromSecret =
                                                      v?.metadata?.name ?? "";
                                                    updateReq();
                                                  }}
                                                />
                                              );
                                            },
                                          )
                                          .otherwise(() => (
                                            <></>
                                          ))}
                                      </Group>
                                    );
                                  },
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>
                          </Tabs>
                        </div>
                      )}
                    </EditItem>

                    <EditItem
                      title="Retry"
                      description="Set retry-specific configs"
                      onUnset={() => {
                        http.http.retry = undefined;
                        updateReq();
                      }}
                      obj={http.http.retry}
                      onSet={() => {
                        http.http.retry =
                          CoreP.Service_Spec_Config_HTTP_Retry.create();
                        updateReq();
                      }}
                    >
                      {http.http.retry && (
                        <div>
                          <Group grow>
                            <NumberInput
                              label="Max retries"
                              placeholder="10"
                              description="Set the max number of retries"
                              min={0}
                              max={10000}
                              value={http.http.retry!.maxRetries}
                              onChange={(v) => {
                                http.http.retry!.maxRetries = strToNum(v);
                                updateReq();
                              }}
                            />

                            <NumberInput
                              label="Multiplier"
                              placeholder="2"
                              description="Set the backoff multiplier"
                              min={0}
                              step={0.1}
                              value={http.http.retry!.multiplier}
                              onChange={(v) => {
                                http.http.retry!.multiplier = strToNum(v);
                                updateReq();
                              }}
                            />

                            <Switch
                              label="Retry on server errors"
                              description="Retry on upstream 5xx server errors"
                              checked={http.http.retry!.retryOnServerErrors}
                              onChange={(v) => {
                                http.http.retry!.retryOnServerErrors =
                                  v.target.checked;
                                updateReq();
                              }}
                            />
                          </Group>

                          <Group grow>
                            <DurationPicker
                              value={http.http.retry!.initialInterval}
                              title="Initial interval"
                              description="Backoff delay before the first retry."
                              onChange={(v) => {
                                http.http.retry!.initialInterval = v;
                                updateReq();
                              }}
                            />

                            <DurationPicker
                              value={http.http.retry!.maxInterval}
                              title="Max interval"
                              description="Maximum backoff delay between retries."
                              onChange={(v) => {
                                http.http.retry!.maxInterval = v;
                                updateReq();
                              }}
                            />

                            <DurationPicker
                              value={http.http.retry!.maxElapsedTime}
                              title="Max elapsed time"
                              description="Maximum total time spent retrying a request."
                              onChange={(v) => {
                                http.http.retry!.maxElapsedTime = v;
                                updateReq();
                              }}
                            />
                          </Group>

                          <ItemMessage
                            title="Retry status codes"
                            obj={
                              http.http.retry.statusCodes.length > 0
                                ? http.http.retry.statusCodes
                                : undefined
                            }
                            isList
                            onSet={() => {
                              http.http.retry!.statusCodes = [0];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.retry!.statusCodes.push(0);
                              updateReq();
                            }}
                          >
                            {http.http.retry.statusCodes.map((x, idx) => (
                              <div className="w-full flex mb-3" key={idx}>
                                <CloseButton
                                  size="sm"
                                  variant="subtle"
                                  onClick={() => {
                                    http.http.retry!.statusCodes.splice(idx, 1);
                                    updateReq();
                                  }}
                                />
                                <NumberInput
                                  required
                                  label="Status code"
                                  description="HTTP status code that triggers an upstream retry."
                                  placeholder="503"
                                  className="flex-1"
                                  min={100}
                                  max={599}
                                  value={http.http.retry!.statusCodes[idx]}
                                  onChange={(v) => {
                                    http.http.retry!.statusCodes[idx] =
                                      strToNum(v);
                                    updateReq();
                                  }}
                                />
                              </div>
                            ))}
                          </ItemMessage>
                        </div>
                      )}
                    </EditItem>

                    <EditItem
                      title="CORS"
                      description="Set Cross-Origin Resource Sharing (CORS)-specific configs"
                      onUnset={() => {
                        http.http.cors = undefined;
                        updateReq();
                      }}
                      obj={http.http.cors}
                      onSet={() => {
                        http.http.cors =
                          CoreP.Service_Spec_Config_HTTP_CORS.create();
                        updateReq();
                      }}
                    >
                      {http.http.cors && (
                        <div>
                          <Group grow>
                            <TextInput
                              label="Allow Methods"
                              placeholder="POST, GET, OPTIONS"
                              description="Set the allowed methods"
                              value={http.http.cors.allowMethods}
                              onChange={(v) => {
                                http.http.cors!.allowMethods = v.target.value;
                                updateReq();
                              }}
                            />

                            <TextInput
                              label="Allow Headers"
                              placeholder="X-PINGOTHER, Content-Type"
                              description="Set the allowed headers"
                              value={http.http.cors.allowHeaders}
                              onChange={(v) => {
                                http.http.cors!.allowHeaders = v.target.value;
                                updateReq();
                              }}
                            />

                            <Switch
                              label="Allow Credentials"
                              checked={http.http.cors!.allowCredentials}
                              description="Allow credentials (such as Cookies and HTTP Authentication) to be sent with requests"
                              onChange={(v) => {
                                http.http.cors!.allowCredentials =
                                  v.target.checked;
                                updateReq();
                              }}
                            />
                          </Group>
                          <Group grow>
                            <TextInput
                              label="Expose Headers"
                              placeholder="Content-Encoding, Kuma-Revision"
                              description="Specify the content for the access-control-expose-headers header"
                              value={http.http.cors.exposeHeaders}
                              onChange={(v) => {
                                http.http.cors!.exposeHeaders = v.target.value;
                                updateReq();
                              }}
                            />

                            <TextInput
                              label="Max Age"
                              placeholder="86400"
                              description="Specify the content for the access-control-max-age header"
                              value={http.http.cors.maxAge}
                              onChange={(v) => {
                                http.http.cors!.maxAge = v.target.value;
                                updateReq();
                              }}
                            />

                            <Switch
                              label="Allow Cluster Services"
                              description="Trust origins from Services in this Cluster"
                              checked={http.http.cors.allowClusterServices}
                              onChange={(event) => {
                                http.http.cors!.allowClusterServices =
                                  event.currentTarget.checked;
                                updateReq();
                              }}
                            />
                          </Group>
                          <ItemMessage
                            title="Allow Origin String Match"
                            obj={
                              http.http.cors.allowOriginStringMatch.length > 0
                                ? http.http.cors.allowOriginStringMatch
                                : undefined
                            }
                            isList
                            onSet={() => {
                              http.http.cors!.allowOriginStringMatch = [""];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              http.http.cors!.allowOriginStringMatch.push("");
                              updateReq();
                            }}
                          >
                            {http.http.cors.allowOriginStringMatch.map(
                              (x, idx) => (
                                <div className="w-full flex mb-3" key={idx}>
                                  <CloseButton
                                    size="sm"
                                    variant="subtle"
                                    onClick={() => {
                                      http.http.cors!.allowOriginStringMatch.splice(
                                        idx,
                                        1,
                                      );
                                      updateReq();
                                    }}
                                  />
                                  <TextInput
                                    required
                                    label="Origin pattern"
                                    description="Exact origin or wildcard allowed to call this Service."
                                    placeholder="https://example.com"
                                    className="flex-1"
                                    value={
                                      http.http.cors!.allowOriginStringMatch[
                                        idx
                                      ]
                                    }
                                    onChange={(v) => {
                                      http.http.cors!.allowOriginStringMatch[
                                        idx
                                      ] = v.target.value;
                                      updateReq();
                                    }}
                                  />
                                </div>
                              ),
                            )}
                          </ItemMessage>
                        </div>
                      )}
                    </EditItem>

                    <EditItem
                      title="Visibility"
                      description="Set visibility-specific configs"
                      onUnset={() => {
                        http.http.visibility = undefined;
                        updateReq();
                      }}
                      obj={http.http.visibility}
                      onSet={() => {
                        http.http.visibility =
                          CoreP.Service_Spec_Config_HTTP_Visibility.create();
                        updateReq();
                      }}
                    >
                      {http.http.visibility && (
                        <div>
                          <Group grow>
                            <Switch
                              label="Enable request body"
                              checked={http.http.visibility!.enableRequestBody}
                              description="Capture the request body"
                              onChange={(v) => {
                                http.http.visibility!.enableRequestBody =
                                  v.target.checked;
                                updateReq();
                              }}
                            />
                            <Switch
                              label="Enable request body map"
                              checked={
                                http.http.visibility!.enableRequestBodyMap
                              }
                              description="Capture the request JSON body map"
                              onChange={(v) => {
                                http.http.visibility!.enableRequestBodyMap =
                                  v.target.checked;
                                updateReq();
                              }}
                            />

                            <Switch
                              label="Enable response body"
                              checked={http.http.visibility!.enableResponseBody}
                              description="Capture the response body"
                              onChange={(v) => {
                                http.http.visibility!.enableResponseBody =
                                  v.target.checked;
                                updateReq();
                              }}
                            />
                            <Switch
                              label="Enable response body map"
                              checked={
                                http.http.visibility!.enableResponseBodyMap
                              }
                              description="Capture the response JSON body map"
                              onChange={(v) => {
                                http.http.visibility!.enableResponseBodyMap =
                                  v.target.checked;
                                updateReq();
                              }}
                            />
                          </Group>
                          <div>
                            <ItemMessage
                              title="Include request headers"
                              obj={http.http.visibility!.includeRequestHeaders}
                              isList
                              onSet={() => {
                                http.http.visibility!.includeRequestHeaders = [
                                  "",
                                ];
                                updateReq();
                              }}
                              onAddListItem={() => {
                                http.http.visibility!.includeRequestHeaders.push(
                                  "",
                                );
                                updateReq();
                              }}
                            >
                              {http.http.visibility!.includeRequestHeaders.map(
                                (x, idx) => (
                                  <div className="w-full flex mb-3" key={idx}>
                                    <CloseButton
                                      size="sm"
                                      variant="subtle"
                                      onClick={() => {
                                        http.http.visibility!.includeRequestHeaders.splice(
                                          idx,
                                          1,
                                        );
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label="Header"
                                      description="Request header included in visibility logs."
                                      placeholder="X-Custom-Header"
                                      className="flex-1"
                                      value={
                                        http.http.visibility!
                                          .includeRequestHeaders[idx]
                                      }
                                      onChange={(v) => {
                                        http.http.visibility!.includeRequestHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </div>
                                ),
                              )}
                            </ItemMessage>

                            <ItemMessage
                              title="Include response headers"
                              obj={http.http.visibility!.includeResponseHeaders}
                              isList
                              onSet={() => {
                                http.http.visibility!.includeResponseHeaders = [
                                  "",
                                ];
                                updateReq();
                              }}
                              onAddListItem={() => {
                                http.http.visibility!.includeResponseHeaders.push(
                                  "",
                                );
                                updateReq();
                              }}
                            >
                              {http.http.visibility!.includeResponseHeaders.map(
                                (x, idx) => (
                                  <div className="w-full flex mb-3" key={idx}>
                                    <CloseButton
                                      size="sm"
                                      variant="subtle"
                                      onClick={() => {
                                        http.http.visibility!.includeResponseHeaders.splice(
                                          idx,
                                          1,
                                        );
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label="Header"
                                      description="Response header included in visibility logs."
                                      placeholder="X-Custom-Header"
                                      className="flex-1"
                                      value={
                                        http.http.visibility!
                                          .includeResponseHeaders[idx]
                                      }
                                      onChange={(v) => {
                                        http.http.visibility!.includeResponseHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </div>
                                ),
                              )}
                            </ItemMessage>

                            <ItemMessage
                              title="Exclude request headers"
                              obj={
                                http.http.visibility!.excludeRequestHeaders
                                  .length > 0
                                  ? http.http.visibility!.excludeRequestHeaders
                                  : undefined
                              }
                              isList
                              onSet={() => {
                                http.http.visibility!.excludeRequestHeaders = [
                                  "",
                                ];
                                updateReq();
                              }}
                              onAddListItem={() => {
                                http.http.visibility!.excludeRequestHeaders.push(
                                  "",
                                );
                                updateReq();
                              }}
                            >
                              {http.http.visibility!.excludeRequestHeaders.map(
                                (x, idx) => (
                                  <div className="w-full flex mb-3" key={idx}>
                                    <CloseButton
                                      size="sm"
                                      variant="subtle"
                                      onClick={() => {
                                        http.http.visibility!.excludeRequestHeaders.splice(
                                          idx,
                                          1,
                                        );
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label="Header"
                                      description="Request header excluded from visibility logs."
                                      placeholder="Authorization"
                                      className="flex-1"
                                      value={
                                        http.http.visibility!
                                          .excludeRequestHeaders[idx]
                                      }
                                      onChange={(v) => {
                                        http.http.visibility!.excludeRequestHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </div>
                                ),
                              )}
                            </ItemMessage>

                            <ItemMessage
                              title="Exclude response headers"
                              obj={
                                http.http.visibility!.excludeResponseHeaders
                                  .length > 0
                                  ? http.http.visibility!.excludeResponseHeaders
                                  : undefined
                              }
                              isList
                              onSet={() => {
                                http.http.visibility!.excludeResponseHeaders = [
                                  "",
                                ];
                                updateReq();
                              }}
                              onAddListItem={() => {
                                http.http.visibility!.excludeResponseHeaders.push(
                                  "",
                                );
                                updateReq();
                              }}
                            >
                              {http.http.visibility!.excludeResponseHeaders.map(
                                (x, idx) => (
                                  <div className="w-full flex mb-3" key={idx}>
                                    <CloseButton
                                      size="sm"
                                      variant="subtle"
                                      onClick={() => {
                                        http.http.visibility!.excludeResponseHeaders.splice(
                                          idx,
                                          1,
                                        );
                                        updateReq();
                                      }}
                                    />
                                    <TextInput
                                      required
                                      label="Header"
                                      description="Response header excluded from visibility logs."
                                      placeholder="Set-Cookie"
                                      className="flex-1"
                                      value={
                                        http.http.visibility!
                                          .excludeResponseHeaders[idx]
                                      }
                                      onChange={(v) => {
                                        http.http.visibility!.excludeResponseHeaders[
                                          idx
                                        ] = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                  </div>
                                ),
                              )}
                            </ItemMessage>

                            <Group grow>
                              <Switch
                                label="Include all request headers"
                                description="Record every request header instead of only the allowlist."
                                checked={
                                  http.http.visibility!.includeAllRequestHeaders
                                }
                                onChange={(v) => {
                                  http.http.visibility!.includeAllRequestHeaders =
                                    v.target.checked;
                                  updateReq();
                                }}
                              />
                              <Switch
                                label="Include all response headers"
                                description="Record every response header instead of only the allowlist."
                                checked={
                                  http.http.visibility!
                                    .includeAllResponseHeaders
                                }
                                onChange={(v) => {
                                  http.http.visibility!.includeAllResponseHeaders =
                                    v.target.checked;
                                  updateReq();
                                }}
                              />
                            </Group>
                          </div>
                        </div>
                      )}
                    </EditItem>

                    <ItemMessage
                      title="Plugins"
                      obj={http.http.plugins}
                      isList
                      onSet={() => {
                        http.http.plugins = [
                          createHTTPPlugin(),
                        ];
                        updateReq();
                      }}
                      onAddListItem={() => {
                        http.http.plugins.push(
                          createHTTPPlugin(),
                        );
                        updateReq();
                      }}
                    >
                      {http.http.plugins.map((plugin, idx) => (
                        <EditItem
                          key={`${idx}`}
                          obj={http.http.plugins[idx]}
                          onUnset={() => {
                            http.http.plugins.splice(idx, 1);
                            updateReq();
                          }}
                        >
                          <Group grow>
                            <TextInput
                              label="Name"
                              required
                              placeholder="my-plugin"
                              description="Set a unique name for the plugin"
                              value={plugin.name}
                              onChange={(v) => {
                                plugin.name = v.target.value;
                                updateReq();
                              }}
                            />

                            <Select
                              label="Phase"
                              required
                              description="Set the plugin phase"
                              data={[
                                {
                                  label: "Post-Authorization",
                                  value:
                                    CoreP.Service_Spec_Config_HTTP_Plugin_Phase[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Plugin_Phase
                                        .POST_AUTH
                                    ],
                                },
                                {
                                  label: "Pre-Authorization",
                                  value:
                                    CoreP.Service_Spec_Config_HTTP_Plugin_Phase[
                                      CoreP
                                        .Service_Spec_Config_HTTP_Plugin_Phase
                                        .PRE_AUTH
                                    ],
                                },
                              ]}
                              value={
                                CoreP.Service_Spec_Config_HTTP_Plugin_Phase[
                                  plugin.phase
                                ]
                              }
                              onChange={(v) => {
                                if (!v) return;
                                plugin.phase =
                                  CoreP.Service_Spec_Config_HTTP_Plugin_Phase[
                                    v as "PRE_AUTH"
                                  ];
                                updateReq();
                              }}
                            />

                            <Switch
                              label="Disabled"
                              description="Disable the plugin"
                              checked={plugin.isDisabled}
                              onChange={(v) => {
                                plugin.isDisabled = v.target.checked;
                                updateReq();
                              }}
                            />
                          </Group>
                          <Cond
                            item={
                              plugin.condition ??
                              CoreP.Condition.create({
                                type: {
                                  oneofKind: `match`,
                                  match: ``,
                                },
                              })
                            }
                            onChange={(v) => {
                              plugin.condition = v;
                              updateReq();
                            }}
                          />
                          <Tabs
                            className="mb-8"
                            value={plugin.type.oneofKind}
                            onChange={(v) => {
                              match(v)
                                .with("direct", () => {
                                  plugin.type = {
                                    oneofKind: "direct",
                                    direct:
                                      CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create({
                                        body: createDirectResponseBody(),
                                      }),
                                  };
                                })
                                .with("rateLimit", () => {
                                  plugin.type = {
                                    oneofKind: "rateLimit",
                                    rateLimit:
                                      CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit.create(),
                                  };
                                })
                                .with("cache", () => {
                                  plugin.type = {
                                    oneofKind: "cache",
                                    cache:
                                      CoreP.Service_Spec_Config_HTTP_Plugin_Cache.create(),
                                  };
                                })
                                .with("lua", () => {
                                  plugin.type = {
                                    oneofKind: "lua",
                                    lua: CoreP.Service_Spec_Config_HTTP_Plugin_Lua.create(
                                      {
                                        type: {
                                          oneofKind: "inline",
                                          inline: "",
                                        },
                                      },
                                    ),
                                  };
                                })
                                .with("path", () => {
                                  plugin.type = {
                                    oneofKind: "path",
                                    path: CoreP.Service_Spec_Config_HTTP_Plugin_Path.create(),
                                  };
                                })
                                .with("jsonSchema", () => {
                                  plugin.type = {
                                    oneofKind: "jsonSchema",
                                    jsonSchema:
                                      CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema.create(
                                        {
                                          type: {
                                            oneofKind: "inline",
                                            inline: "",
                                          },
                                        },
                                      ),
                                  };
                                })
                                .with("extProc", () => {
                                  plugin.type = {
                                    oneofKind: "extProc",
                                    extProc:
                                      CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc.create(
                                        {
                                          type: {
                                            oneofKind: "address",
                                            address: "",
                                          },
                                        },
                                      ),
                                  };
                                })
                                .otherwise(() => {});
                              updateReq();
                            }}
                          >
                            <Tabs.List className="w-full">
                              <Tabs.Tab value="direct">
                                Direct Response
                              </Tabs.Tab>
                              <Tabs.Tab value="rateLimit">Rate Limit</Tabs.Tab>
                              <Tabs.Tab value="cache">Cache</Tabs.Tab>
                              <Tabs.Tab value="lua">Lua</Tabs.Tab>
                              <Tabs.Tab value="path">Path</Tabs.Tab>
                              <Tabs.Tab value="jsonSchema">
                                JSON Schema
                              </Tabs.Tab>
                              <Tabs.Tab value="extProc">Ext Proc</Tabs.Tab>
                            </Tabs.List>

                            <Tabs.Panel value="direct">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "direct",
                                  (direct) => (
                                    <div>
                                      <Group grow>
                                        <NumberInput
                                          label="Status Code"
                                          description="HTTP status code to return"
                                          min={100}
                                          max={599}
                                          value={direct.direct.statusCode}
                                          onChange={(v) => {
                                            direct.direct.statusCode =
                                              strToNum(v);
                                            updateReq();
                                          }}
                                        />
                                      </Group>

                                      <EditItem
                                        title="Body"
                                        description="Set the direct response body"
                                        onUnset={() => {
                                          direct.direct.body = undefined;
                                          updateReq();
                                        }}
                                        obj={direct.direct.body}
                                        onSet={() => {
                                          direct.direct.body =
                                            CoreP.Service_Spec_Config_HTTP_Plugin_Direct_Body.create(
                                              {
                                                type: {
                                                  oneofKind: "inline",
                                                  inline: "",
                                                },
                                              },
                                            );
                                          updateReq();
                                        }}
                                      >
                                        {direct.direct.body && (
                                          <Tabs
                                            value={
                                              direct.direct.body.type
                                                .oneofKind ?? "inline"
                                            }
                                            onChange={(v) => {
                                              match(v)
                                                .with("inline", () => {
                                                  direct.direct.body!.type = {
                                                    oneofKind: "inline",
                                                    inline: "",
                                                  };
                                                })
                                                .with("inlineBytes", () => {
                                                  direct.direct.body!.type = {
                                                    oneofKind: "inlineBytes",
                                                    inlineBytes:
                                                      new Uint8Array(),
                                                  };
                                                })
                                                .otherwise(() => {});
                                              updateReq();
                                            }}
                                          >
                                            <Tabs.List>
                                              <Tabs.Tab value="inline">
                                                Inline Text
                                              </Tabs.Tab>
                                              <Tabs.Tab value="inlineBytes">
                                                Inline Bytes
                                              </Tabs.Tab>
                                            </Tabs.List>

                                            <Tabs.Panel value="inline">
                                              {match(direct.direct.body.type)
                                                .when(
                                                  (x) =>
                                                    x.oneofKind === "inline",
                                                  (inline) => (
                                                    <div>
                                                      <TextAreaCustom
                                                        label="Inline body"
                                                        description="Response body returned directly to the downstream client."
                                                        placeholder='{ "message": "ok" }'
                                                        value={inline.inline}
                                                        onChange={(v) => {
                                                          inline.inline =
                                                            v ?? "";
                                                          updateReq();
                                                        }}
                                                      />
                                                    </div>
                                                  ),
                                                )
                                                .otherwise(() => (
                                                  <></>
                                                ))}
                                            </Tabs.Panel>

                                            <Tabs.Panel value="inlineBytes">
                                              {match(direct.direct.body.type)
                                                .when(
                                                  (x) =>
                                                    x.oneofKind ===
                                                    "inlineBytes",
                                                  (inlineBytes) => (
                                                    <div>
                                                      <TextAreaCustom
                                                        label="Inline bytes"
                                                        description="Raw response bytes returned directly to the downstream client."
                                                        placeholder="Raw bytes content"
                                                        value={new TextDecoder().decode(
                                                          inlineBytes.inlineBytes,
                                                        )}
                                                        onChange={(v) => {
                                                          inlineBytes.inlineBytes =
                                                            new TextEncoder().encode(
                                                              v ?? "",
                                                            );
                                                          updateReq();
                                                        }}
                                                      />
                                                    </div>
                                                  ),
                                                )
                                                .otherwise(() => (
                                                  <></>
                                                ))}
                                            </Tabs.Panel>
                                          </Tabs>
                                        )}
                                      </EditItem>

                                      <ItemMessage
                                        title="Headers"
                                        obj={direct.direct.headers}
                                        isList
                                        onSet={() => {
                                          direct.direct.headers = [
                                            CoreP.Service_Spec_Config_HTTP_Plugin_Direct_KeyValue.create(),
                                          ];
                                          updateReq();
                                        }}
                                        onAddListItem={() => {
                                          direct.direct.headers.push(
                                            CoreP.Service_Spec_Config_HTTP_Plugin_Direct_KeyValue.create(),
                                          );
                                          updateReq();
                                        }}
                                      >
                                        {direct.direct.headers.map(
                                          (h, hIdx) => (
                                            <div
                                              className="w-full flex mb-3"
                                              key={hIdx}
                                            >
                                              <CloseButton
                                                size="sm"
                                                variant="subtle"
                                                className="mr-2"
                                                onClick={() => {
                                                  direct.direct.headers.splice(
                                                    hIdx,
                                                    1,
                                                  );
                                                  updateReq();
                                                }}
                                              />
                                              <Group
                                                className="flex w-full"
                                                grow
                                              >
                                                <TextInput
                                                  required
                                                  label="Key"
                                                  description="Response header name."
                                                  placeholder="Content-Type"
                                                  value={
                                                    direct.direct.headers[hIdx]
                                                      .key
                                                  }
                                                  onChange={(v) => {
                                                    direct.direct.headers[
                                                      hIdx
                                                    ].key = v.target.value;
                                                    updateReq();
                                                  }}
                                                />
                                                <TextInput
                                                  required
                                                  label="Value"
                                                  description="Response header value."
                                                  placeholder="application/json"
                                                  value={
                                                    direct.direct.headers[hIdx]
                                                      .value
                                                  }
                                                  onChange={(v) => {
                                                    direct.direct.headers[
                                                      hIdx
                                                    ].value = v.target.value;
                                                    updateReq();
                                                  }}
                                                />
                                              </Group>
                                            </div>
                                          ),
                                        )}
                                      </ItemMessage>
                                    </div>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="rateLimit">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "rateLimit",
                                  (rateLimit) => (
                                    <div>
                                      <Group grow>
                                        <NumberInput
                                          label="Limit"
                                          description="Maximum number of requests per window"
                                          min={0}
                                          value={Number(
                                            rateLimit.rateLimit.limit,
                                          )}
                                          onChange={(v) => {
                                            rateLimit.rateLimit.limit =
                                              strToNum(v);

                                            updateReq();
                                          }}
                                        />
                                        <NumberInput
                                          label="Status Code"
                                          description="HTTP status code when rate limited"
                                          min={100}
                                          max={599}
                                          value={rateLimit.rateLimit.statusCode}
                                          onChange={(v) => {
                                            rateLimit.rateLimit.statusCode =
                                              strToNum(v);
                                            updateReq();
                                          }}
                                        />
                                        <DurationPicker
                                          value={rateLimit.rateLimit.window}
                                          title="Window"
                                          description="Sliding time window used to count requests."
                                          onChange={(v) => {
                                            rateLimit.rateLimit.window = v;
                                            updateReq();
                                          }}
                                        />
                                      </Group>
                                    </div>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="cache">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "cache",
                                  (cache) => (
                                    <Group grow>
                                      <NumberInput
                                        label="Max size"
                                        description="Maximum number of cached entries"
                                        min={0}
                                        value={Number(cache.cache.maxSize)}
                                        onChange={(v) => {
                                          cache.cache.maxSize = strToNum(v);

                                          updateReq();
                                        }}
                                      />
                                      <DurationPicker
                                        value={cache.cache.ttl}
                                        title="TTL"
                                        description="How long a cached response remains valid."
                                        onChange={(v) => {
                                          cache.cache.ttl = v;
                                          updateReq();
                                        }}
                                      />
                                      <Switch
                                        label="Use X-Cache header"
                                        description="Add an X-Cache response header indicating cache status."
                                        checked={cache.cache.useXCacheHeader}
                                        onChange={(v) => {
                                          cache.cache.useXCacheHeader =
                                            v.target.checked;
                                          updateReq();
                                        }}
                                      />
                                      <Switch
                                        label="Allow unsafe methods"
                                        description="Allow caching methods beyond GET and HEAD."
                                        checked={cache.cache.allowUnsafeMethods}
                                        onChange={(v) => {
                                          cache.cache.allowUnsafeMethods =
                                            v.target.checked;
                                          updateReq();
                                        }}
                                      />
                                    </Group>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="lua">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "lua",
                                  (lua) => (
                                    <LuaScriptEditor
                                      value={
                                        lua.lua.type.oneofKind === "inline"
                                          ? lua.lua.type.inline
                                          : ""
                                      }
                                      onChange={(v) => {
                                        lua.lua.type = {
                                          oneofKind: "inline",
                                          inline: v,
                                        };
                                        updateReq();
                                      }}
                                    />
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="jsonSchema">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "jsonSchema",
                                  (jsonSchema) => (
                                    <div>
                                      <Group grow>
                                        <NumberInput
                                          label="Status Code"
                                          description="HTTP status code returned on validation failure"
                                          min={100}
                                          max={599}
                                          value={
                                            jsonSchema.jsonSchema.statusCode
                                          }
                                          onChange={(v) => {
                                            jsonSchema.jsonSchema.statusCode =
                                              strToNum(v);
                                            updateReq();
                                          }}
                                        />
                                      </Group>
                                      <TextAreaCustom
                                        label="JSON Schema"
                                        description="JSON Schema used to validate request bodies before forwarding."
                                        placeholder='{ "type": "object" }'
                                        value={
                                          jsonSchema.jsonSchema.type
                                            .oneofKind === "inline"
                                            ? jsonSchema.jsonSchema.type.inline
                                            : ""
                                        }
                                        onChange={(v) => {
                                          jsonSchema.jsonSchema.type = {
                                            oneofKind: "inline",
                                            inline: v ?? "",
                                          };
                                          updateReq();
                                        }}
                                      />

                                      <EditItem
                                        title="Body"
                                        description="Set the response body on validation failure"
                                        onUnset={() => {
                                          jsonSchema.jsonSchema.body =
                                            undefined;
                                          updateReq();
                                        }}
                                        obj={jsonSchema.jsonSchema.body}
                                        onSet={() => {
                                          jsonSchema.jsonSchema.body =
                                            CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema_Body.create(
                                              {
                                                type: {
                                                  oneofKind: "inline",
                                                  inline: "",
                                                },
                                              },
                                            );
                                          updateReq();
                                        }}
                                      >
                                        {jsonSchema.jsonSchema.body &&
                                          match(jsonSchema.jsonSchema.body.type)
                                            .when(
                                              (x) => x.oneofKind === "inline",
                                              (inline) => (
                                                <div>
                                                  <TextAreaCustom
                                                    label="Inline body"
                                                    description="Response body returned when validation fails."
                                                    placeholder="Invalid request body"
                                                    value={inline.inline}
                                                    onChange={(v) => {
                                                      inline.inline = v ?? "";
                                                      updateReq();
                                                    }}
                                                  />
                                                </div>
                                              ),
                                            )
                                            .otherwise(() => <></>)}
                                      </EditItem>

                                      <ItemMessage
                                        title="Headers"
                                        obj={jsonSchema.jsonSchema.headers}
                                        isList
                                        onSet={() => {
                                          jsonSchema.jsonSchema.headers = [
                                            CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema_KeyValue.create(),
                                          ];
                                          updateReq();
                                        }}
                                        onAddListItem={() => {
                                          jsonSchema.jsonSchema.headers.push(
                                            CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema_KeyValue.create(),
                                          );
                                          updateReq();
                                        }}
                                      >
                                        {jsonSchema.jsonSchema.headers.map(
                                          (h, hIdx) => (
                                            <div
                                              className="w-full flex mb-3"
                                              key={hIdx}
                                            >
                                              <CloseButton
                                                size="sm"
                                                variant="subtle"
                                                className="mr-2"
                                                onClick={() => {
                                                  jsonSchema.jsonSchema.headers.splice(
                                                    hIdx,
                                                    1,
                                                  );
                                                  updateReq();
                                                }}
                                              />
                                              <Group
                                                className="flex w-full"
                                                grow
                                              >
                                                <TextInput
                                                  required
                                                  label="Key"
                                                  description="Response header name returned when validation fails."
                                                  placeholder="Content-Type"
                                                  value={
                                                    jsonSchema.jsonSchema
                                                      .headers[hIdx].key
                                                  }
                                                  onChange={(v) => {
                                                    jsonSchema.jsonSchema.headers[
                                                      hIdx
                                                    ].key = v.target.value;
                                                    updateReq();
                                                  }}
                                                />
                                                <TextInput
                                                  required
                                                  label="Value"
                                                  description="Response header value returned when validation fails."
                                                  placeholder="application/json"
                                                  value={
                                                    jsonSchema.jsonSchema
                                                      .headers[hIdx].value
                                                  }
                                                  onChange={(v) => {
                                                    jsonSchema.jsonSchema.headers[
                                                      hIdx
                                                    ].value = v.target.value;
                                                    updateReq();
                                                  }}
                                                />
                                              </Group>
                                            </div>
                                          ),
                                        )}
                                      </ItemMessage>
                                    </div>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="extProc">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "extProc",
                                  (extProc) => (
                                    <div>
                                      <Tabs
                                        className="mb-4"
                                        value={
                                          extProc.extProc.type.oneofKind ??
                                          "address"
                                        }
                                        onChange={(v) => {
                                          match(v)
                                            .with("address", () => {
                                              extProc.extProc.type = {
                                                oneofKind: "address",
                                                address: "",
                                              };
                                            })
                                            .with("container", () => {
                                              extProc.extProc.type = {
                                                oneofKind: "container",
                                                container:
                                                  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_Container.create(),
                                              };
                                            })
                                            .otherwise(() => {});
                                          updateReq();
                                        }}
                                      >
                                        <Tabs.List>
                                          <Tabs.Tab value="address">
                                            Address
                                          </Tabs.Tab>
                                          <Tabs.Tab value="container">
                                            Container
                                          </Tabs.Tab>
                                        </Tabs.List>

                                        <Tabs.Panel value="address">
                                          {match(extProc.extProc.type)
                                            .when(
                                              (x) => x.oneofKind === "address",
                                              (address) => (
                                                <TextInput
                                                  required
                                                  label="Address"
                                                  description="Address of the Envoy ext_proc gRPC server."
                                                  placeholder="ext-proc.default.svc:9000"
                                                  value={address.address}
                                                  onChange={(v) => {
                                                    address.address =
                                                      v.target.value;
                                                    updateReq();
                                                  }}
                                                />
                                              ),
                                            )
                                            .otherwise(() => (
                                              <></>
                                            ))}
                                        </Tabs.Panel>

                                        <Tabs.Panel value="container">
                                          {match(extProc.extProc.type)
                                            .when(
                                              (x) =>
                                                x.oneofKind === "container",
                                              (container) => (
                                                <Group grow>
                                                  <TextInput
                                                    required
                                                    label="Image"
                                                    description="Container image that serves the ext_proc gRPC endpoint."
                                                    placeholder="ghcr.io/org/ext-proc:latest"
                                                    value={
                                                      container.container.image
                                                    }
                                                    onChange={(v) => {
                                                      container.container.image =
                                                        v.target.value;
                                                      updateReq();
                                                    }}
                                                  />
                                                  <NumberInput
                                                    label="Port"
                                                    description="Port on which the ext_proc container listens."
                                                    min={0}
                                                    max={65535}
                                                    value={
                                                      container.container.port
                                                    }
                                                    onChange={(v) => {
                                                      container.container.port =
                                                        strToNum(v);
                                                      updateReq();
                                                    }}
                                                  />
                                                </Group>
                                              ),
                                            )
                                            .otherwise(() => (
                                              <></>
                                            ))}
                                        </Tabs.Panel>
                                      </Tabs>

                                      <DurationPicker
                                        value={extProc.extProc.messageTimeout}
                                        title="Message timeout"
                                        description="Maximum time to wait for ext_proc processing."
                                        onChange={(v) => {
                                          extProc.extProc.messageTimeout = v;
                                          updateReq();
                                        }}
                                      />

                                      <EditItem
                                        title="Processing Mode"
                                        description="Set the ext-proc processing mode"
                                        onUnset={() => {
                                          extProc.extProc.processingMode =
                                            undefined;
                                          updateReq();
                                        }}
                                        obj={extProc.extProc.processingMode}
                                        onSet={() => {
                                          extProc.extProc.processingMode =
                                            CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode.create();
                                          updateReq();
                                        }}
                                      >
                                        {extProc.extProc.processingMode && (
                                          <Group grow>
                                            <Select
                                              label="Request header mode"
                                              description="Choose whether request headers are sent to ext_proc."
                                              data={[
                                                {
                                                  label: "Send",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode
                                                        .SEND
                                                    ],
                                                },
                                                {
                                                  label: "Skip",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode
                                                        .SKIP
                                                    ],
                                                },
                                              ]}
                                              value={
                                                CoreP
                                                  .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                  extProc.extProc.processingMode
                                                    .requestHeaderMode
                                                ]
                                              }
                                              onChange={(v) => {
                                                if (!v) return;
                                                extProc.extProc.processingMode!.requestHeaderMode =
                                                  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                    v as "SEND"
                                                  ];
                                                updateReq();
                                              }}
                                            />
                                            <Select
                                              label="Response header mode"
                                              description="Choose whether response headers are sent to ext_proc."
                                              data={[
                                                {
                                                  label: "Send",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode
                                                        .SEND
                                                    ],
                                                },
                                                {
                                                  label: "Skip",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode
                                                        .SKIP
                                                    ],
                                                },
                                              ]}
                                              value={
                                                CoreP
                                                  .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                  extProc.extProc.processingMode
                                                    .responseHeaderMode
                                                ]
                                              }
                                              onChange={(v) => {
                                                if (!v) return;
                                                extProc.extProc.processingMode!.responseHeaderMode =
                                                  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode[
                                                    v as "SEND"
                                                  ];
                                                updateReq();
                                              }}
                                            />
                                            <Select
                                              label="Request body mode"
                                              description="Choose whether request body content is sent to ext_proc."
                                              data={[
                                                {
                                                  label: "None",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode
                                                        .NONE
                                                    ],
                                                },
                                                {
                                                  label: "Buffered",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode
                                                        .BUFFERED
                                                    ],
                                                },
                                              ]}
                                              value={
                                                CoreP
                                                  .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                  extProc.extProc.processingMode
                                                    .requestBodyMode
                                                ]
                                              }
                                              onChange={(v) => {
                                                if (!v) return;
                                                extProc.extProc.processingMode!.requestBodyMode =
                                                  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                    v as "NONE"
                                                  ];
                                                updateReq();
                                              }}
                                            />
                                            <Select
                                              label="Response body mode"
                                              description="Choose whether response body content is sent to ext_proc."
                                              data={[
                                                {
                                                  label: "None",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode
                                                        .NONE
                                                    ],
                                                },
                                                {
                                                  label: "Buffered",
                                                  value:
                                                    CoreP
                                                      .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                      CoreP
                                                        .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode
                                                        .BUFFERED
                                                    ],
                                                },
                                              ]}
                                              value={
                                                CoreP
                                                  .Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                  extProc.extProc.processingMode
                                                    .responseBodyMode
                                                ]
                                              }
                                              onChange={(v) => {
                                                if (!v) return;
                                                extProc.extProc.processingMode!.responseBodyMode =
                                                  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode[
                                                    v as "NONE"
                                                  ];
                                                updateReq();
                                              }}
                                            />
                                          </Group>
                                        )}
                                      </EditItem>
                                    </div>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>

                            <Tabs.Panel value="path">
                              {match(plugin.type)
                                .when(
                                  (x) => x.oneofKind === "path",
                                  (path) => (
                                    <Group grow>
                                      <TextInput
                                        label="Add prefix"
                                        description="Prefix to add to the forwarded request path."
                                        placeholder="/api/v1"
                                        value={path.path.addPrefix}
                                        onChange={(v) => {
                                          path.path.addPrefix = v.target.value;
                                          updateReq();
                                        }}
                                      />
                                      <TextInput
                                        label="Remove prefix"
                                        description="Prefix to remove from the request path before forwarding."
                                        placeholder="/api/v2"
                                        value={path.path.removePrefix}
                                        onChange={(v) => {
                                          path.path.removePrefix =
                                            v.target.value;
                                          updateReq();
                                        }}
                                      />
                                    </Group>
                                  ),
                                )
                                .otherwise(() => (
                                  <></>
                                ))}
                            </Tabs.Panel>
                          </Tabs>
                        </EditItem>
                      ))}
                    </ItemMessage>
                  </div>
                );
              },
            )
            .when(
              (x) => x.oneofKind === `ssh`,
              (ssh) => {
                return (
                  <div>
                    <Group grow>
                      <TextInput
                        label="User"
                        placeholder="root"
                        description="Force a specific SSH user"
                        value={ssh.ssh.user}
                        onChange={(v) => {
                          ssh.ssh!.user = v.target.value;
                          updateReq();
                        }}
                      />

                      <Switch
                        className="my-2"
                        label="Enable local port forwarding"
                        description="This enables Client-less BeyondCorp mode"
                        checked={ssh.ssh.enableLocalPortForwarding}
                        onChange={(v) => {
                          ssh.ssh.enableLocalPortForwarding = v.target.checked;
                          updateReq();
                        }}
                      />

                      <Switch
                        className="my-2"
                        label="Embedded SSH Mode"
                        description="Switch to embedded SSH mode served by connected Octelium clients"
                        checked={ssh.ssh.eSSHMode}
                        onChange={(v) => {
                          ssh.ssh.eSSHMode = v.target.checked;
                          updateReq();
                        }}
                      />

                      <Switch
                        className="my-2"
                        label="Enable Subsystems"
                        description="Enable SSH subsystems"
                        checked={ssh.ssh.enableSubsystem}
                        onChange={(v) => {
                          ssh.ssh.enableSubsystem = v.target.checked;
                          updateReq();
                        }}
                      />
                    </Group>

                    <Group grow>
                      <EditItem
                        title="Upstream Host Key"
                        description="Set the upstream host key"
                        onUnset={() => {
                          ssh.ssh.upstreamHostKey = undefined;
                          updateReq();
                        }}
                        obj={ssh.ssh.upstreamHostKey}
                        onSet={() => {
                          ssh.ssh.upstreamHostKey =
                            CoreP.Service_Spec_Config_SSH_UpstreamHostKey.create(
                              {
                                type: {
                                  oneofKind: `key`,
                                  key: "",
                                },
                              },
                            );

                          updateReq();
                        }}
                      >
                        {ssh.ssh.upstreamHostKey && (
                          <div>
                            {match(ssh.ssh.upstreamHostKey?.type)
                              .when(
                                (x) => x?.oneofKind == `key`,
                                (key) => {
                                  return (
                                    <div>
                                      <TextAreaCustom
                                        label="Host public key"
                                        description="Public key used to verify the upstream SSH host."
                                        placeholder="ssh-rsa AAAAB3NzaC1y..."
                                        value={key.key}
                                        onChange={(v) => {
                                          key.key = v ?? "";
                                          updateReq();
                                        }}
                                      />
                                    </div>
                                  );
                                },
                              )
                              .when(
                                (x) => x?.oneofKind === `insecureIgnoreHostKey`,
                                (insecureIgnoreHostKey) => {
                                  return (
                                    <Switch
                                      required
                                      label="Ignore host key"
                                      description="Ignore checking the upstream's public key"
                                      checked={
                                        insecureIgnoreHostKey.insecureIgnoreHostKey
                                      }
                                      onChange={(v) => {
                                        insecureIgnoreHostKey.insecureIgnoreHostKey =
                                          v.target.checked;
                                        updateReq();
                                      }}
                                    />
                                  );
                                },
                              )
                              .otherwise(() => (
                                <></>
                              ))}
                          </div>
                        )}
                      </EditItem>
                    </Group>

                    <EditItem
                      title="Authentication"
                      description="Set the upstream User credential"
                      onUnset={() => {
                        ssh.ssh.auth = undefined;
                        updateReq();
                      }}
                      obj={ssh.ssh.auth}
                      onSet={() => {
                        ssh.ssh.auth =
                          CoreP.Service_Spec_Config_SSH_Auth.create();

                        updateReq();
                      }}
                    >
                      {ssh.ssh.auth && (
                        <Tabs
                          className="mb-8"
                          value={ssh.ssh.auth!.type.oneofKind}
                          onChange={(v) => {
                            match(v)
                              .with("password", () => {
                                match(init.type)
                                  .when(
                                    (x) => x.oneofKind === `ssh`,
                                    (x) => {
                                      match(x.ssh.auth?.type)
                                        .when(
                                          (x) => x?.oneofKind === `password`,
                                          (x) => {
                                            ssh.ssh.auth!.type = x;
                                          },
                                        )
                                        .otherwise(() => {
                                          ssh.ssh.auth!.type = {
                                            oneofKind: "password",
                                            password:
                                              CoreP.Service_Spec_Config_SSH_Auth_Password.create(
                                                {
                                                  type: {
                                                    oneofKind: "fromSecret",
                                                    fromSecret: "",
                                                  },
                                                },
                                              ),
                                          };
                                        });
                                    },
                                  )
                                  .otherwise(() => {
                                    ssh.ssh.auth!.type = {
                                      oneofKind: "password",
                                      password:
                                        CoreP.Service_Spec_Config_SSH_Auth_Password.create(
                                          {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        ),
                                    };
                                  });

                                updateReq();
                              })
                              .with("privateKey", () => {
                                match(init.type)
                                  .when(
                                    (x) => x.oneofKind === `ssh`,
                                    (x) => {
                                      match(x.ssh.auth?.type)
                                        .when(
                                          (x) => x?.oneofKind === `privateKey`,
                                          (x) => {
                                            ssh.ssh.auth!.type = x;
                                          },
                                        )
                                        .otherwise(() => {
                                          ssh.ssh.auth!.type = {
                                            oneofKind: "privateKey",
                                            privateKey:
                                              CoreP.Service_Spec_Config_SSH_Auth_PrivateKey.create(
                                                {
                                                  type: {
                                                    oneofKind: "fromSecret",
                                                    fromSecret: "",
                                                  },
                                                },
                                              ),
                                          };
                                        });
                                    },
                                  )
                                  .otherwise(() => {
                                    ssh.ssh.auth!.type = {
                                      oneofKind: "privateKey",
                                      privateKey:
                                        CoreP.Service_Spec_Config_SSH_Auth_PrivateKey.create(
                                          {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        ),
                                    };
                                  });

                                updateReq();
                              });
                          }}
                        >
                          <Tabs.List>
                            <Tabs.Tab value="password">Password</Tabs.Tab>
                            <Tabs.Tab value="privateKey">Private Key</Tabs.Tab>
                          </Tabs.List>
                          <Tabs.Panel value="password">
                            {match(ssh.ssh.auth.type)
                              .when(
                                (x) => x.oneofKind === `password`,
                                (password) => {
                                  return match(password.password.type)
                                    .when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => {
                                        return (
                                          <SelectResource
                                            api="core"
                                            kind="Secret"
                                            label="Password Secret"
                                            description="Select the Secret of the password"
                                            defaultValue={x.fromSecret}
                                            onChange={(v) => {
                                              x.fromSecret =
                                                v?.metadata?.name ?? "";
                                              updateReq();
                                            }}
                                          />
                                        );
                                      },
                                    )
                                    .otherwise(() => <></>);
                                },
                              )
                              .otherwise(() => (
                                <></>
                              ))}
                          </Tabs.Panel>

                          <Tabs.Panel value="privateKey">
                            {match(ssh.ssh.auth.type)
                              .when(
                                (x) => x.oneofKind === `privateKey`,
                                (privateKey) => {
                                  return match(privateKey.privateKey.type)
                                    .when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => {
                                        return (
                                          <SelectResource
                                            api="core"
                                            kind="Secret"
                                            label="Private key Secret"
                                            description="Select the Secret of the private key"
                                            defaultValue={x.fromSecret}
                                            onChange={(v) => {
                                              x.fromSecret =
                                                v?.metadata?.name ?? "";
                                              updateReq();
                                            }}
                                          />
                                        );
                                      },
                                    )
                                    .otherwise(() => <></>);
                                },
                              )
                              .otherwise(() => (
                                <></>
                              ))}
                          </Tabs.Panel>
                        </Tabs>
                      )}
                    </EditItem>

                    <EditItem
                      title="Visibility"
                      description="Set SSH session recording options"
                      onUnset={() => {
                        ssh.ssh.visibility = undefined;
                        updateReq();
                      }}
                      obj={ssh.ssh.visibility}
                      onSet={() => {
                        ssh.ssh.visibility =
                          CoreP.Service_Spec_Config_SSH_Visibility.create();
                        updateReq();
                      }}
                    >
                      {ssh.ssh.visibility && (
                        <Group grow>
                          <Switch
                            label="Disable session recording"
                            description="Do not record SSH sessions in access logs."
                            checked={ssh.ssh.visibility.disableSessionRecording}
                            onChange={(v) => {
                              ssh.ssh.visibility!.disableSessionRecording =
                                v.target.checked;
                              updateReq();
                            }}
                          />
                          <Switch
                            label="Enable stdin recording"
                            description="Also record stdin input in session recordings"
                            checked={
                              ssh.ssh.visibility.enableSessionStdinRecording
                            }
                            onChange={(v) => {
                              ssh.ssh.visibility!.enableSessionStdinRecording =
                                v.target.checked;
                              updateReq();
                            }}
                          />
                        </Group>
                      )}
                    </EditItem>
                  </div>
                );
              },
            )
            .when(
              (x) => x.oneofKind === `postgres`,
              (postgres) => {
                return (
                  <div>
                    <Group grow>
                      <TextInput
                        // required
                        label="User"
                        description="Force a specific User"
                        placeholder="root"
                        value={postgres.postgres.user}
                        onChange={(v) => {
                          postgres.postgres!.user = v.target.value;
                          updateReq();
                        }}
                      />

                      <TextInput
                        // required
                        label="Database"
                        description="Force a specific database"
                        placeholder="default"
                        value={postgres.postgres.database}
                        onChange={(v) => {
                          postgres.postgres!.database = v.target.value;
                          updateReq();
                        }}
                      />

                      <Select
                        label="TLS Mode"
                        clearable
                        description="Set the upstream TLS mode"
                        data={[
                          {
                            label: "Require",
                            value:
                              CoreP.Service_Spec_Config_Postgres_SSLMode[
                                CoreP.Service_Spec_Config_Postgres_SSLMode
                                  .REQUIRE
                              ],
                          },
                          {
                            label: "Disable",
                            value:
                              CoreP.Service_Spec_Config_Postgres_SSLMode[
                                CoreP.Service_Spec_Config_Postgres_SSLMode
                                  .DISABLE
                              ],
                          },
                        ]}
                        value={
                          CoreP.Service_Spec_Config_Postgres_SSLMode[
                            postgres.postgres.sslMode
                          ]
                        }
                        onChange={(v) => {
                          if (!v) {
                            postgres.postgres!.sslMode =
                              CoreP.Service_Spec_Config_Postgres_SSLMode.SSL_MODE_UNSET;

                            updateReq();
                            return;
                          }

                          postgres.postgres!.sslMode =
                            CoreP.Service_Spec_Config_Postgres_SSLMode[
                              v as "REQUIRE"
                            ];
                          updateReq();
                        }}
                      />

                      {match(postgres.postgres.auth?.type)
                        .when(
                          (x) => x?.oneofKind === `password`,
                          (password) => {
                            return match(password.password.type)
                              .when(
                                (x) => x?.oneofKind === `fromSecret`,
                                (x) => {
                                  return (
                                    <SelectResource
                                      api="core"
                                      kind="Secret"
                                      label="Password Secret"
                                      description="Select the Secret of the Password"
                                      defaultValue={x.fromSecret}
                                      onChange={(v) => {
                                        x.fromSecret = v?.metadata?.name ?? "";
                                        updateReq();
                                      }}
                                    />
                                  );
                                },
                              )
                              .otherwise(() => <></>);
                          },
                        )
                        .otherwise(() => (
                          <></>
                        ))}
                    </Group>

                    <EditItem
                      title="Authorization"
                      description="Set PostgreSQL-specific authorization configuration"
                      onUnset={() => {
                        postgres.postgres.authorization = undefined;
                        updateReq();
                      }}
                      obj={postgres.postgres.authorization}
                      onSet={() => {
                        postgres.postgres.authorization =
                          CoreP.Service_Spec_Config_Postgres_Authorization.create();
                        updateReq();
                      }}
                    >
                      {postgres.postgres.authorization && (
                        <Select
                          label="Authorization Mode"
                          description="Set when authorization is enforced"
                          data={[
                            {
                              label: "None (connection only)",
                              value:
                                CoreP
                                  .Service_Spec_Config_Postgres_Authorization_Mode[
                                  CoreP
                                    .Service_Spec_Config_Postgres_Authorization_Mode
                                    .NONE
                                ],
                            },
                            {
                              label: "All (every command)",
                              value:
                                CoreP
                                  .Service_Spec_Config_Postgres_Authorization_Mode[
                                  CoreP
                                    .Service_Spec_Config_Postgres_Authorization_Mode
                                    .ALL
                                ],
                            },
                          ]}
                          value={
                            CoreP
                              .Service_Spec_Config_Postgres_Authorization_Mode[
                              postgres.postgres.authorization.mode
                            ]
                          }
                          onChange={(v) => {
                            if (!v) return;
                            postgres.postgres.authorization!.mode =
                              CoreP.Service_Spec_Config_Postgres_Authorization_Mode[
                                v as "ALL"
                              ];
                            updateReq();
                          }}
                        />
                      )}
                    </EditItem>
                  </div>
                );
              },
            )
            .when(
              (x) => x.oneofKind === `mysql`,
              (mysql) => {
                return (
                  <div>
                    <Group grow>
                      <TextInput
                        required
                        label="User"
                        description="Force a specific user"
                        placeholder="root"
                        value={mysql.mysql.user}
                        onChange={(v) => {
                          mysql.mysql.user = v.target.value;

                          updateReq();
                        }}
                      />

                      <TextInput
                        required
                        label="Database"
                        placeholder="default"
                        description="Force a specific database"
                        value={mysql.mysql.database}
                        onChange={(v) => {
                          mysql.mysql.database = v.target.value;
                          updateReq();
                        }}
                      />
                      <Switch
                        label="Enable TLS"
                        description="Connect to the Upstream over TLS"
                        checked={mysql.mysql.isTLS}
                        onChange={(v) => {
                          mysql.mysql.isTLS = v.target.checked;
                          updateReq();
                        }}
                      />

                      {match(mysql.mysql.auth?.type)
                        .when(
                          (x) => x?.oneofKind === `password`,
                          (password) => {
                            return match(password.password.type)
                              .when(
                                (x) => x?.oneofKind === `fromSecret`,
                                (x) => {
                                  return (
                                    <SelectResource
                                      api="core"
                                      kind="Secret"
                                      label="Password Secret"
                                      description="Secret whose value contains the upstream MySQL password."
                                      defaultValue={x.fromSecret}
                                      onChange={(v) => {
                                        x.fromSecret = v?.metadata?.name ?? "";
                                        updateReq();
                                      }}
                                    />
                                  );
                                },
                              )
                              .otherwise(() => <></>);
                          },
                        )
                        .otherwise(() => (
                          <></>
                        ))}
                    </Group>
                  </div>
                );
              },
            )
            .when(
              (x) => x.oneofKind === "socks5",
              (socks5) => (
                <div>
                  <Group grow>
                    <Switch
                      label="Embedded mode"
                      description="Run the SOCKS5 proxy in embedded mode"
                      checked={socks5.socks5.isEmbeddedMode}
                      onChange={(v) => {
                        socks5.socks5.isEmbeddedMode = v.target.checked;
                        updateReq();
                      }}
                    />
                  </Group>

                  <EditItem
                    title="Authentication"
                    description="Set the SOCKS5 upstream server authentication method"
                    onUnset={() => {
                      socks5.socks5.auth = undefined;
                      updateReq();
                    }}
                    obj={socks5.socks5.auth}
                    onSet={() => {
                      socks5.socks5.auth =
                        CoreP.Service_Spec_Config_SOCKS5_Auth.create({
                          type: { oneofKind: "noAuth", noAuth: true },
                        });
                      updateReq();
                    }}
                  >
                    {socks5.socks5.auth && (
                      <Tabs
                        value={socks5.socks5.auth.type.oneofKind ?? "noAuth"}
                        onChange={(v) => {
                          match(v)
                            .with("noAuth", () => {
                              socks5.socks5.auth!.type = {
                                oneofKind: "noAuth",
                                noAuth: true,
                              };
                            })
                            .with("usernamePassword", () => {
                              socks5.socks5.auth!.type = {
                                oneofKind: "usernamePassword",
                                usernamePassword:
                                  CoreP.Service_Spec_Config_SOCKS5_Auth_UsernamePassword.create(
                                    {
                                      password: {
                                        type: {
                                          oneofKind: "fromSecret",
                                          fromSecret: "",
                                        },
                                      },
                                    },
                                  ),
                              };
                            })
                            .otherwise(() => {});
                          updateReq();
                        }}
                      >
                        <Tabs.List>
                          <Tabs.Tab value="noAuth">No Auth</Tabs.Tab>
                          <Tabs.Tab value="usernamePassword">
                            Username & Password
                          </Tabs.Tab>
                        </Tabs.List>

                        <Tabs.Panel value="noAuth">
                          <p className="text-[0.8rem] text-slate-500 mt-2">
                            The upstream does not require authentication.
                          </p>
                        </Tabs.Panel>

                        <Tabs.Panel value="usernamePassword">
                          {match(socks5.socks5.auth.type)
                            .when(
                              (x) => x.oneofKind === "usernamePassword",
                              (up) => (
                                <Group grow>
                                  <TextInput
                                    label="Username"
                                    description="Username presented to the SOCKS5 upstream."
                                    placeholder="user1234"
                                    value={up.usernamePassword.username}
                                    onChange={(v) => {
                                      up.usernamePassword.username =
                                        v.target.value;
                                      updateReq();
                                    }}
                                  />
                                  {match(up.usernamePassword.password?.type)
                                    .when(
                                      (x) => x?.oneofKind === "fromSecret",
                                      (x) => (
                                        <SelectResource
                                          api="core"
                                          kind="Secret"
                                          label="Password Secret"
                                          description="Select the Secret holding the password"
                                          defaultValue={x.fromSecret}
                                          onChange={(val) => {
                                            x.fromSecret =
                                              val?.metadata?.name ?? "";
                                            updateReq();
                                          }}
                                        />
                                      ),
                                    )
                                    .otherwise(() => (
                                      <></>
                                    ))}
                                </Group>
                              ),
                            )
                            .otherwise(() => (
                              <></>
                            ))}
                        </Tabs.Panel>
                      </Tabs>
                    )}
                  </EditItem>
                </div>
              ),
            )
            .when(
              (x) => x.oneofKind === "rdp",
              (rdp) => (
                <div>
                  <EditItem
                    title="Authentication"
                    description="Set the upstream RDP server authentication credentials"
                    onUnset={() => {
                      rdp.rdp.auth = undefined;
                      updateReq();
                    }}
                    obj={rdp.rdp.auth}
                    onSet={() => {
                      rdp.rdp.auth = CoreP.Service_Spec_Config_RDP_Auth.create({
                        password: {
                          type: { oneofKind: "fromSecret", fromSecret: "" },
                        },
                      });
                      updateReq();
                    }}
                  >
                    {rdp.rdp.auth && (
                      <div>
                        <Group grow>
                          <TextInput
                            label="User"
                            description="Username used to authenticate to the upstream RDP server."
                            placeholder="administrator"
                            value={rdp.rdp.auth.user}
                            onChange={(v) => {
                              rdp.rdp.auth!.user = v.target.value;
                              updateReq();
                            }}
                          />
                          <TextInput
                            label="Domain"
                            description="Optional Windows domain for the upstream RDP account."
                            placeholder="CORP"
                            value={rdp.rdp.auth.domain}
                            onChange={(v) => {
                              rdp.rdp.auth!.domain = v.target.value;
                              updateReq();
                            }}
                          />
                        </Group>

                        {match(rdp.rdp.auth.password?.type)
                          .when(
                            (x) => x?.oneofKind === "fromSecret",
                            (x) => (
                              <SelectResource
                                api="core"
                                kind="Secret"
                                label="Password Secret"
                                description="Select the Secret holding the upstream RDP password"
                                defaultValue={x.fromSecret}
                                onChange={(val) => {
                                  x.fromSecret = val?.metadata?.name ?? "";
                                  updateReq();
                                }}
                              />
                            ),
                          )
                          .otherwise(() => (
                            <></>
                          ))}
                      </div>
                    )}
                  </EditItem>

                  <EditItem
                    title="Upstream TLS"
                    description="Set the upstream RDP TLS verification options"
                    onUnset={() => {
                      rdp.rdp.upstreamTLS = undefined;
                      updateReq();
                    }}
                    obj={rdp.rdp.upstreamTLS}
                    onSet={() => {
                      rdp.rdp.upstreamTLS =
                        CoreP.Service_Spec_Config_RDP_UpstreamTLS.create();
                      updateReq();
                    }}
                  >
                    {rdp.rdp.upstreamTLS && (
                      <div>
                        <Group grow>
                          <Switch
                            label="Allow any certificate"
                            description="Skip upstream certificate verification (insecure)"
                            checked={rdp.rdp.upstreamTLS.allowAnyCert}
                            onChange={(v) => {
                              rdp.rdp.upstreamTLS!.allowAnyCert =
                                v.target.checked;
                              updateReq();
                            }}
                          />
                        </Group>

                        <ItemMessage
                          title="Pinned Certificate SHA256"
                          obj={
                            rdp.rdp.upstreamTLS.pinnedCertSHA256.length > 0
                              ? rdp.rdp.upstreamTLS.pinnedCertSHA256
                              : undefined
                          }
                          isList
                          onSet={() => {
                            rdp.rdp.upstreamTLS!.pinnedCertSHA256 = [""];
                            updateReq();
                          }}
                          onAddListItem={() => {
                            rdp.rdp.upstreamTLS!.pinnedCertSHA256.push("");
                            updateReq();
                          }}
                        >
                          {rdp.rdp.upstreamTLS.pinnedCertSHA256.map(
                            (x, idx) => (
                              <div className="w-full flex mb-3" key={idx}>
                                <CloseButton
                                  size="sm"
                                  variant="subtle"
                                  onClick={() => {
                                    rdp.rdp.upstreamTLS!.pinnedCertSHA256.splice(
                                      idx,
                                      1,
                                    );
                                    updateReq();
                                  }}
                                />
                                <TextInput
                                  required
                                  label="SHA256 fingerprint"
                                  description="SHA-256 fingerprint of a certificate trusted for the upstream RDP connection."
                                  placeholder="ab:cd:ef:12:34:..."
                                  className="flex-1"
                                  value={
                                    rdp.rdp.upstreamTLS!.pinnedCertSHA256[idx]
                                  }
                                  onChange={(v) => {
                                    rdp.rdp.upstreamTLS!.pinnedCertSHA256[idx] =
                                      v.target.value;
                                    updateReq();
                                  }}
                                />
                              </div>
                            ),
                          )}
                        </ItemMessage>
                      </div>
                    )}
                  </EditItem>
                </div>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </EditItem>
      )}

      <EditItem
        title="TLS"
        description="Set TLS-specific configs"
        onUnset={() => {
          req.tls = undefined;
          updateReq();
        }}
        obj={req.tls}
        onSet={() => {
          req.tls = CoreP.Service_Spec_Config_TLS.create();

          updateReq();
        }}
      >
        {req.tls && (
          <div>
            <Group grow>
              <Switch
                className="my-2"
                label="Skip verify"
                description="(INSECURE) Skip verifying the upstream server certificate"
                checked={req.tls.insecureSkipVerify}
                onChange={(v) => {
                  req.tls!.insecureSkipVerify = v.target.checked;
                  updateReq();
                }}
              />

              <Switch
                className="my-2"
                label="Append to system pool"
                description="Append your CAs to the system pool of CAs instead of overriding it"
                checked={req.tls.appendToSystemPool}
                onChange={(v) => {
                  req.tls!.appendToSystemPool = v.target.checked;
                  updateReq();
                }}
              />
            </Group>
            <ItemMessage
              title="Trusted CAs"
              obj={req.tls!.trustedCAs}
              isList
              onSet={() => {
                req.tls!.trustedCAs = [""];
                updateReq();
              }}
              onAddListItem={() => {
                req.tls!.trustedCAs.push("");
                updateReq();
              }}
            >
              {req.tls!.trustedCAs.map((rule: any, ruleIdx: number) => {
                return (
                  <EditItem
                    key={`${ruleIdx}`}
                    obj={{}}
                    onUnset={() => {
                      req.tls!.trustedCAs.splice(ruleIdx, 1);
                      updateReq();
                    }}
                  >
                    <TextAreaCustom
                      label="PEM certificate authority"
                      description="PEM-encoded root CA trusted for the upstream TLS connection."
                      value={req.tls!.trustedCAs[ruleIdx]}
                      placeholder={`-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUQyOS38lJDJ1dkt6oV5yal6UferUwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQ...wpk+geq0
-----END CERTIFICATE-----`}
                      onChange={(v) => {
                        req.tls!.trustedCAs[ruleIdx] = v ?? "";
                        updateReq();
                      }}
                    />
                  </EditItem>
                );
              })}
            </ItemMessage>

            <EditItem
              title="Client Certificate"
              description="Set client certificate info"
              onUnset={() => {
                req.tls!.clientCertificate = undefined;
                updateReq();
              }}
              obj={req.tls.clientCertificate}
              onSet={() => {
                req.tls!.clientCertificate =
                  CoreP.Service_Spec_Config_TLS_ClientCertificate.create({
                    type: {
                      oneofKind: `fromSecret`,
                      fromSecret: "",
                    },
                  });

                updateReq();
              }}
            >
              {match(req.tls!.clientCertificate?.type)
                .when(
                  (x) => x?.oneofKind === `fromSecret`,
                  (x) => {
                    return (
                      <div>
                        <SelectResource
                          api="core"
                          kind="Secret"
                          label="Client certificate Secret"
                          description="Secret containing the client certificate chain and private key for mTLS."
                          defaultValue={x.fromSecret}
                          onChange={(v) => {
                            x.fromSecret = v?.metadata?.name ?? "";
                            updateReq();
                          }}
                        />
                      </div>
                    );
                  },
                )
                .otherwise(() => (
                  <></>
                ))}
            </EditItem>
          </div>
        )}
      </EditItem>
    </div>
  );
};

const Edit = (props: {
  item: CoreP.Service;
  onUpdate: (item: CoreP.Service) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(CoreP.Service.clone(item));
  const [init, setInit] = React.useState(CoreP.Service.clone(item));
  const configsByMode = React.useRef<
    Partial<Record<number, CoreP.Service_Spec_Config>>
  >(
    item.spec?.config
      ? {
          [item.spec.mode]: CoreP.Service_Spec_Config.clone(item.spec.config),
        }
      : {},
  );
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;
  React.useEffect(() => {
    const next = CoreP.Service.clone(item);
    setReq(next);
    setInit(CoreP.Service.clone(item));
    configsByMode.current = item.spec?.config
      ? {
          [item.spec.mode]: CoreP.Service_Spec_Config.clone(item.spec.config),
        }
      : {};
  }, [itemKey]);
  const updateReq = () => {
    const next = CoreP.Service.clone(req);
    setReq(next);
    onUpdate(CoreP.Service.clone(next));
  };

  return (
    <div>
      <Group grow>
        <Select
          label="Service Mode"
          required
          description="Application-layer protocol used by this Service."
          data={[
            {
              label: "HTTP",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.HTTP],
            },
            {
              label: "TCP",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.TCP],
            },
            {
              label: "SSH",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.SSH],
            },
            {
              label: "Web App",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.WEB],
            },
            {
              label: "Kubernetes",
              value:
                CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.KUBERNETES],
            },
            {
              label: "PostgreSQL",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.POSTGRES],
            },
            {
              label: "MySQL",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.MYSQL],
            },
            {
              label: "UDP",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.UDP],
            },
            {
              label: "gRPC",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.GRPC],
            },
            {
              label: "DNS",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.DNS],
            },
            {
              label: "SOCKS5",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.SOCKS5],
            },
            {
              label: "RDP Web",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.RDP_WEB],
            },
            {
              label: "MCP Gateway",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.MCP],
            },
            {
              label: "LLM / AI Gateway",
              value: CoreP.Service_Spec_Mode[CoreP.Service_Spec_Mode.LLM],
            },
          ]}
          value={CoreP.Service_Spec_Mode[req.spec!.mode]}
          onChange={(v) => {
            if (!v) return;
            const previousMode = req.spec!.mode;
            if (req.spec!.config) {
              configsByMode.current[previousMode] =
                CoreP.Service_Spec_Config.clone(req.spec!.config);
            }
            const nextMode = CoreP.Service_Spec_Mode[
              v as keyof typeof CoreP.Service_Spec_Mode
            ] as CoreP.Service_Spec_Mode;
            req.spec!.mode = nextMode;

            match(req.spec!.mode)
              .when(
                (x) =>
                  x === CoreP.Service_Spec_Mode.HTTP ||
                  x === CoreP.Service_Spec_Mode.GRPC ||
                  x === CoreP.Service_Spec_Mode.WEB,
                () => {
                  if (
                    previousMode === CoreP.Service_Spec_Mode.HTTP ||
                    previousMode === CoreP.Service_Spec_Mode.GRPC ||
                    previousMode === CoreP.Service_Spec_Mode.WEB
                  ) {
                    return;
                  }

                  const previousConfig =
                    configsByMode.current[nextMode] ??
                    configsByMode.current[CoreP.Service_Spec_Mode.HTTP] ??
                    configsByMode.current[CoreP.Service_Spec_Mode.WEB] ??
                    configsByMode.current[CoreP.Service_Spec_Mode.GRPC];
                  if (previousConfig) {
                    req.spec!.config =
                      CoreP.Service_Spec_Config.clone(previousConfig);
                    return;
                  }

                  match(init.spec!.config?.type)
                    .when(
                      (x) => x?.oneofKind === `http`,
                      () => {
                        req.spec!.config = CoreP.Service_Spec_Config.clone(
                          init.spec!.config!,
                        );
                      },
                    )
                    .otherwise(() => {
                      req.spec!.config = CoreP.Service_Spec_Config.create({
                        upstream: {
                          type: {
                            oneofKind: "url",
                            url: "",
                          },
                        },
                        type: {
                          oneofKind: "http",
                          http: {} as CoreP.Service_Spec_Config_HTTP,
                        },
                      });
                    });
                },
              )
              .with(CoreP.Service_Spec_Mode.SSH, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `ssh`,
                    (x) => {
                      req.spec!.config = CoreP.Service_Spec_Config.clone(
                        init.spec!.config!,
                      );
                    },
                  )
                  .otherwise(() => {
                    req.spec!.config = CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "ssh",
                        ssh: {
                          auth: {
                            type: {
                              oneofKind: "password",
                              password: {
                                type: {
                                  oneofKind: "fromSecret",
                                  fromSecret: "",
                                },
                              },
                            },
                          },
                        } as CoreP.Service_Spec_Config_SSH,
                      },
                    });
                  });
              })
              .with(CoreP.Service_Spec_Mode.POSTGRES, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `postgres`,
                    (x) => {
                      req.spec!.config = CoreP.Service_Spec_Config.clone(
                        init.spec!.config!,
                      );
                    },
                  )
                  .otherwise(() => {
                    req.spec!.config = CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "postgres",
                        //@ts-ignore
                        postgres: {
                          auth: {
                            type: {
                              oneofKind: "password",
                              password: {
                                type: {
                                  oneofKind: "fromSecret",
                                  fromSecret: "",
                                },
                              },
                            },
                          },
                        },
                      },
                    });
                  });
              })
              .with(CoreP.Service_Spec_Mode.MYSQL, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `mysql`,
                    (x) => {
                      req.spec!.config = CoreP.Service_Spec_Config.clone(
                        init.spec!.config!,
                      );
                    },
                  )
                  .otherwise(() => {
                    req.spec!.config = CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "mysql",

                        //@ts-ignore
                        mysql: {
                          auth: {
                            type: {
                              oneofKind: "password",
                              password: {
                                type: {
                                  oneofKind: "fromSecret",
                                  fromSecret: "",
                                },
                              },
                            },
                          },
                        },
                      },
                    });
                  });
              })
              .with(CoreP.Service_Spec_Mode.KUBERNETES, () => {
                const previousConfig =
                  configsByMode.current[CoreP.Service_Spec_Mode.KUBERNETES];
                req.spec!.config = previousConfig
                  ? CoreP.Service_Spec_Config.clone(previousConfig)
                  : CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "kubernetes",
                        kubernetes: {
                          type: {
                            oneofKind: `kubeconfig`,
                            kubeconfig: {
                              type: {
                                oneofKind: `fromSecret`,
                                fromSecret: "",
                              },
                            },
                          },
                        } as CoreP.Service_Spec_Config_Kubernetes,
                      },
                    });
              })
              .with(CoreP.Service_Spec_Mode.SOCKS5, () => {
                const previousConfig =
                  configsByMode.current[CoreP.Service_Spec_Mode.SOCKS5];
                req.spec!.config = previousConfig
                  ? CoreP.Service_Spec_Config.clone(previousConfig)
                  : CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "socks5",
                        socks5: {
                          auth: { type: { oneofKind: "noAuth", noAuth: true } },
                        } as CoreP.Service_Spec_Config_SOCKS5,
                      },
                    });
              })
              .with(CoreP.Service_Spec_Mode.RDP_WEB, () => {
                const previousConfig =
                  configsByMode.current[CoreP.Service_Spec_Mode.RDP_WEB];
                req.spec!.config = previousConfig
                  ? CoreP.Service_Spec_Config.clone(previousConfig)
                  : CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: {
                          oneofKind: "url",
                          url: "",
                        },
                      },
                      type: {
                        oneofKind: "rdp",
                        rdp: {
                          auth: {
                            password: {
                              type: { oneofKind: "fromSecret", fromSecret: "" },
                            },
                          },
                        } as CoreP.Service_Spec_Config_RDP,
                      },
                    });
              })
              .with(CoreP.Service_Spec_Mode.MCP, () => {
                const previousConfig =
                  configsByMode.current[CoreP.Service_Spec_Mode.MCP];
                req.spec!.config = previousConfig
                  ? CoreP.Service_Spec_Config.clone(previousConfig)
                  : CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: { oneofKind: "url", url: "" },
                      },
                      type: {
                        oneofKind: "mcp",
                        mcp: CoreP.Service_Spec_Config_MCP.create(),
                      },
                    });
              })
              .with(CoreP.Service_Spec_Mode.LLM, () => {
                const previousConfig =
                  configsByMode.current[CoreP.Service_Spec_Mode.LLM];
                req.spec!.config = previousConfig
                  ? CoreP.Service_Spec_Config.clone(previousConfig)
                  : CoreP.Service_Spec_Config.create({
                      upstream: {
                        type: { oneofKind: "url", url: "" },
                      },
                      type: {
                        oneofKind: "llm",
                        llm: CoreP.Service_Spec_Config_LLM.create(),
                      },
                    });
              })

              .when(
                (x) =>
                  x === CoreP.Service_Spec_Mode.TCP ||
                  x === CoreP.Service_Spec_Mode.DNS ||
                  x === CoreP.Service_Spec_Mode.UDP,
                () => {
                  const previousConfig = configsByMode.current[nextMode];
                  req.spec!.config = previousConfig
                    ? CoreP.Service_Spec_Config.clone(previousConfig)
                    : CoreP.Service_Spec_Config.create({
                        upstream: {
                          type: {
                            oneofKind: "url",
                            url: "",
                          },
                        },
                        type: {
                          oneofKind: undefined,
                        },
                      });
                },
              )
              .otherwise(() => {});
            updateReq();
          }}
        />

        <NumberInput
          label="Port"
          placeholder="8080"
          description="Listener port; when unset, the port can be inferred from the upstream URL."
          min={0}
          max={65535}
          value={req.spec!.port}
          onChange={(v) => {
            req.spec!.port = strToNum(v);
            updateReq();
          }}
        />

        <Switch
          label="Disabled"
          description="Disable the Service so it stops serving requests."
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>

      {(req.spec!.mode === CoreP.Service_Spec_Mode.HTTP ||
        req.spec!.mode === CoreP.Service_Spec_Mode.WEB ||
        req.spec!.mode === CoreP.Service_Spec_Mode.GRPC ||
        req.spec!.mode === CoreP.Service_Spec_Mode.MCP ||
        req.spec!.mode === CoreP.Service_Spec_Mode.LLM ||
        req.spec!.mode === CoreP.Service_Spec_Mode.KUBERNETES) && (
        <Group className="my-2" grow>
          <Switch
            className="my-2"
            label="Enable Public"
            description="Allow access without a client connection (clientless/public mode)."
            checked={req.spec!.isPublic}
            onChange={(v) => {
              req.spec!.isPublic = v.target.checked;
              if (!v.target.checked) {
                req.spec!.isAnonymous = false;
                if (req.spec!.authorization) {
                  req.spec!.authorization.enableAnonymous = false;
                }
              }
              updateReq();
            }}
          />

          <Switch
            label="Enable TLS"
            description="Serve the public listener over TLS."
            checked={req.spec!.isTLS}
            onChange={(v) => {
              req.spec!.isTLS = v.target.checked;
              updateReq();
            }}
          />

          <Switch
            label="Enable Anonymous Access"
            description="Allow unauthenticated public access when the Service is public."
            checked={req.spec!.isAnonymous}
            disabled={!req.spec?.isPublic}
            onChange={(v) => {
              req.spec!.isAnonymous = v.target.checked;
              if (
                !v.target.checked &&
                req.spec!.authorization?.enableAnonymous
              ) {
                req.spec!.authorization.enableAnonymous = false;
              }
              updateReq();
            }}
          />
        </Group>
      )}

      <EditItem
        title="Configuration"
        description="Set the default Service configuration"
        onUnset={() => {
          req.spec!.config = undefined;
          updateReq();
        }}
        obj={req.spec!.config}
        onSet={() => {
          match(init.spec?.config)
            .when(
              (x) => !!x,
              (x) => {
                req.spec!.config = CoreP.Service_Spec_Config.clone(x);
              },
            )
            .otherwise(() => {
              req.spec!.config = CoreP.Service_Spec_Config.create({
                upstream: {
                  type: {
                    oneofKind: "url",
                    url: "",
                  },
                },
              });
            });

          updateReq();
        }}
      >
        {req.spec!.config && (
          <Config
            key={`${itemKey}-${req.spec!.mode}`}
            default
            mode={req.spec!.mode}
            item={req.spec!.config}
            onUpdate={(v) => {
              req.spec!.config = v;
              updateReq();
            }}
          />
        )}
      </EditItem>

      <EditItem
        title="Authorization"
        description="Set the Service Policies"
        onUnset={() => {
          req.spec!.authorization = undefined;
          updateReq();
        }}
        obj={req.spec!.authorization}
        onSet={() => {
          if (!req.spec!.authorization) {
            req.spec!.authorization = CoreP.Service_Spec_Authorization.create({
              policies: [],
            });
            updateReq();
          }
        }}
      >
        {req.spec!.authorization && (
          <>
            <Group>
              <Switch
                label="Enable Anonymous Authorization"
                description="Use Authorization Policies in Anonymous mode"
                checked={req.spec!.authorization.enableAnonymous}
                disabled={!req.spec?.isAnonymous}
                onChange={(v) => {
                  req.spec!.authorization!.enableAnonymous = v.target.checked;
                  updateReq();
                }}
              />
            </Group>

            <SelectPolicies
              policies={req.spec!.authorization.policies}
              onUpdate={(v) => {
                if (!v) {
                  req.spec!.authorization!.policies = [];
                } else {
                  req.spec!.authorization!.policies = v;
                }

                updateReq();
              }}
            />

            <SelectInlinePolicies
              inlinePolicies={req.spec!.authorization!.inlinePolicies}
              onUpdate={(v) => {
                req.spec!.authorization!.inlinePolicies = v;
                updateReq();
              }}
            />
          </>
        )}
      </EditItem>

      <EditItem
        title="Deployment"
        description="Set deployment-related configs such as replica number"
        onUnset={() => {
          req.spec!.deployment = undefined;
          updateReq();
        }}
        obj={req.spec!.deployment}
        onSet={() => {
          req.spec!.deployment = CoreP.Service_Spec_Deployment.create();
          updateReq();
        }}
      >
        {req.spec!.deployment && (
          <>
            <Group>
              <NumberInput
                label="Replicas"
                placeholder="1"
                description="Set the number of replicas of the Service deployment"
                min={0}
                max={100}
                value={req.spec!.deployment.replicas}
                onChange={(v) => {
                  req.spec!.deployment!.replicas = strToNum(v);
                  updateReq();
                }}
              />
            </Group>
          </>
        )}
      </EditItem>

      <EditItem
        title="Dynamic Configuration"
        description="Set multiple named dynamic Configurations"
        onUnset={() => {
          req.spec!.dynamicConfig = undefined;
          updateReq();
        }}
        obj={req.spec!.dynamicConfig}
        onSet={() => {
          req.spec!.dynamicConfig = CoreP.Service_Spec_DynamicConfig.create();
          updateReq();
        }}
      >
        {req.spec!.dynamicConfig && (
          <div className="w-full">
            <ItemMessage
              title="Configurations"
              obj={req.spec!.dynamicConfig.configs}
              isList
              onSet={() => {
                req.spec!.dynamicConfig!.configs = [
                  CoreP.Service_Spec_Config.create(),
                ];
                updateReq();
              }}
              onAddListItem={() => {
                req.spec!.dynamicConfig!.configs.push(
                  CoreP.Service_Spec_Config.create(),
                );
                updateReq();
              }}
            >
              {req.spec!.dynamicConfig.configs.map((x, idx) => (
                <EditItem
                  key={`${idx}`}
                  obj={{}}
                  title={x.name || `Configuration ${idx + 1}`}
                  onUnset={() => {
                    req.spec!.dynamicConfig!.configs.splice(idx, 1);
                    updateReq();
                  }}
                >
                  <Config
                    key={`${idx}-${req.spec!.mode}`}
                    item={req.spec!.dynamicConfig!.configs[idx]}
                    mode={req.spec!.mode}
                    onUpdate={(v) => {
                      req.spec!.dynamicConfig!.configs[idx] = v;
                      updateReq();
                    }}
                  />
                </EditItem>
              ))}
            </ItemMessage>

            <ItemMessage
              title="Rules"
              obj={req.spec!.dynamicConfig.rules}
              isList
              onSet={() => {
                req.spec!.dynamicConfig!.rules = [
                  CoreP.Service_Spec_DynamicConfig_Rule.create({
                    type: {
                      oneofKind: `configName`,
                      configName: ``,
                    },
                  }),
                ];
                updateReq();
              }}
              onAddListItem={() => {
                req.spec!.dynamicConfig!.rules.push(
                  CoreP.Service_Spec_DynamicConfig_Rule.create({
                    type: {
                      oneofKind: `configName`,
                      configName: ``,
                    },
                  }),
                );
                updateReq();
              }}
            >
              {req.spec!.dynamicConfig!.rules &&
                req.spec!.dynamicConfig!.rules.map(
                  (rule: any, ruleIdx: number) => (
                    <div key={`${ruleIdx}`}>
                      <EditItem
                        obj={req.spec!.dynamicConfig!.rules[ruleIdx]}
                        onUnset={() => {
                          req.spec!.dynamicConfig!.rules.splice(ruleIdx, 1);
                          updateReq();
                        }}
                      >
                        <Group grow>
                          <Cond
                            item={
                              req.spec!.dynamicConfig!.rules[ruleIdx]
                                .condition ??
                              CoreP.Condition.create({
                                type: {
                                  oneofKind: `match`,
                                  match: ``,
                                },
                              })
                            }
                            onChange={(v) => {
                              req.spec!.dynamicConfig!.rules[
                                ruleIdx
                              ].condition = v;
                              updateReq();
                            }}
                          />
                        </Group>

                        <div className="space-y-1.5">
                          <p className="text-sm font-semibold text-slate-700">
                            Rule type
                          </p>
                          <SegmentedControl
                            aria-label="Rule type"
                            size="sm"
                            fullWidth
                            className="w-full"
                            value={
                              req.spec!.dynamicConfig!.rules[ruleIdx].type
                                .oneofKind
                            }
                            data={[
                              { label: "Config name", value: "configName" },
                              { label: "Eval (CEL)", value: "eval" },
                              { label: "OPA (Rego)", value: "opa" },
                            ]}
                            onChange={(v) => {
                              req.spec!.dynamicConfig!.rules[ruleIdx].type =
                                match(v)
                                  .with("eval", () => ({
                                    oneofKind: "eval" as const,
                                    eval: "",
                                  }))
                                  .with("opa", () => ({
                                    oneofKind: "opa" as const,
                                    opa: "",
                                  }))
                                  .otherwise(() => ({
                                    oneofKind: "configName" as const,
                                    configName: "",
                                  }));
                              updateReq();
                            }}
                          />
                          <p className="text-xs font-medium text-slate-500">
                            Choose a named configuration or evaluate a complete service configuration object.
                          </p>
                        </div>
                      </EditItem>

                      {match(req.spec!.dynamicConfig!.rules[ruleIdx].type)
                        .when(
                          (x) => x.oneofKind === `configName`,
                          (configName) => {
                            return (
                              <div>
                                <Select
                                  label="Config name"
                                  required
                                  description="Select the config name"
                                  value={configName.configName}
                                  data={req.spec!.dynamicConfig!.configs.map(
                                    (x) => x.name,
                                  )}
                                  onChange={(v) => {
                                    configName.configName = v ?? "";
                                    updateReq();
                                  }}
                                />
                              </div>
                            );
                          },
                        )
                        .when(
                          (x) => x.oneofKind === `eval`,
                          (evalType) => {
                            return (
                              <div>
                                <TextAreaCustom
                                  label="Eval (CEL)"
                                  description="Return a service.spec.config-compatible object."
                                  placeholder={`{
  "upstream": {
    "url": "https://" + ctx.user.metadata.uid + ".example.com"
  }
}`}
                                  value={evalType.eval}
                                  onChange={(v) => {
                                    evalType.eval = v ?? "";
                                    updateReq();
                                  }}
                                />
                              </div>
                            );
                          },
                        )
                        .when(
                          (x) => x.oneofKind === `opa`,
                          (opa) => {
                            return (
                              <div>
                                <TextAreaCustom
                                  label="OPA (Rego)"
                                  description="Set result to a service.spec.config-compatible object."
                                  placeholder={`package octelium.eval

result := {
  "upstream": {
    "url": sprintf("https://%s.example.com", [input.ctx.service.metadata.name])
  }
}`}
                                  value={opa.opa}
                                  onChange={(v) => {
                                    opa.opa = v ?? "";
                                    updateReq();
                                  }}
                                />
                              </div>
                            );
                          },
                        )
                        .otherwise(() => (
                          <></>
                        ))}
                    </div>
                  ),
                )}
            </ItemMessage>
          </div>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
