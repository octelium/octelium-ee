import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import ItemMessage from "@/components/ItemMessage";

import SelectSecret from "@/components/ResourceLayout/SelectSecret";
import {
  CloseButton,
  Group,
  NumberInput,
  Select,
  Switch,
  Tabs,
  TextInput,
} from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";

type SecretSelector = {
  type:
    | { oneofKind: "fromSecret"; fromSecret: string }
    | { oneofKind: undefined };
};

type GrpcStyleAuth = {
  type:
    | { oneofKind: "bearer"; bearer: SecretSelector }
    | {
        oneofKind: "basic";
        basic: { username: string; password?: SecretSelector };
      }
    | {
        oneofKind: "custom";
        custom: { header: string; value?: SecretSelector };
      }
    | { oneofKind: undefined };
};

const freshSecret = (): SecretSelector => ({
  type: { oneofKind: "fromSecret", fromSecret: "" },
});

const freshBearer = (): GrpcStyleAuth["type"] => ({
  oneofKind: "bearer",
  bearer: freshSecret(),
});

const freshBasic = (): GrpcStyleAuth["type"] => ({
  oneofKind: "basic",
  basic: { username: "", password: freshSecret() },
});

const freshCustom = (): GrpcStyleAuth["type"] => ({
  oneofKind: "custom",
  custom: { header: "", value: freshSecret() },
});

const SecretSelectField = (props: {
  label: string;
  sel?: SecretSelector;
  onUpdate: () => void;
}) => {
  const { label, sel, onUpdate } = props;

  if (!sel || sel.type.oneofKind !== "fromSecret") {
    return <></>;
  }

  return (
    <SelectSecret
      api="enterprise"
      label={label}
      defaultValue={sel.type.fromSecret}
      onChange={(v) => {
        if (sel.type.oneofKind === "fromSecret") {
          sel.type.fromSecret = v ?? "";
          onUpdate();
        }
      }}
    />
  );
};

const SecretAuthTabs = (props: {
  auth?: GrpcStyleAuth;
  ensureAuth: () => GrpcStyleAuth;
  initAuth?: GrpcStyleAuth;
  onUpdate: () => void;
}) => {
  const { auth, ensureAuth, initAuth, onUpdate } = props;

  return (
    <Tabs
      defaultValue={auth?.type.oneofKind}
      onChange={(v) => {
        const a = ensureAuth();
        match(v)
          .with("bearer", () => {
            a.type =
              initAuth?.type.oneofKind === "bearer"
                ? initAuth.type
                : freshBearer();
          })
          .with("basic", () => {
            a.type =
              initAuth?.type.oneofKind === "basic"
                ? initAuth.type
                : freshBasic();
          })
          .with("custom", () => {
            a.type =
              initAuth?.type.oneofKind === "custom"
                ? initAuth.type
                : freshCustom();
          })
          .otherwise(() => {});
        onUpdate();
      }}
    >
      <Tabs.List>
        <Tabs.Tab value="bearer">Bearer Authentication</Tabs.Tab>
        <Tabs.Tab value="basic">Basic Authentication</Tabs.Tab>
        <Tabs.Tab value="custom">Custom Header</Tabs.Tab>
      </Tabs.List>

      <Tabs.Panel value="bearer">
        {match(auth?.type)
          .with({ oneofKind: "bearer" }, (b) => (
            <SecretSelectField
              label="Bearer Token Secret"
              sel={b.bearer}
              onUpdate={onUpdate}
            />
          ))
          .otherwise(() => (
            <></>
          ))}
      </Tabs.Panel>

      <Tabs.Panel value="basic">
        {match(auth?.type)
          .with({ oneofKind: "basic" }, (b) => (
            <Group grow>
              <TextInput
                label="Username"
                placeholder="username"
                value={b.basic.username}
                onChange={(v) => {
                  b.basic.username = v.target.value;
                  onUpdate();
                }}
              />
              <SecretSelectField
                label="Password Secret"
                sel={b.basic.password}
                onUpdate={onUpdate}
              />
            </Group>
          ))
          .otherwise(() => (
            <></>
          ))}
      </Tabs.Panel>

      <Tabs.Panel value="custom">
        {match(auth?.type)
          .with({ oneofKind: "custom" }, (c) => (
            <Group grow>
              <TextInput
                required
                label="Header"
                placeholder="X-Custom-Auth"
                value={c.custom.header}
                onChange={(v) => {
                  c.custom.header = v.target.value;
                  onUpdate();
                }}
              />
              <SecretSelectField
                label="Header Value Secret"
                sel={c.custom.value}
                onUpdate={onUpdate}
              />
            </Group>
          ))
          .otherwise(() => (
            <></>
          ))}
      </Tabs.Panel>
    </Tabs>
  );
};

const HeaderList = (props: {
  headers: { key: string; value: string }[];
  makeHeader: () => { key: string; value: string };
  onUpdate: () => void;
}) => {
  const { headers, makeHeader, onUpdate } = props;

  return (
    <Group grow>
      <ItemMessage
        title="Add Headers"
        obj={headers}
        isList
        onSet={() => {
          headers.push(makeHeader());
          onUpdate();
        }}
        onAddListItem={() => {
          headers.push(makeHeader());
          onUpdate();
        }}
      >
        {headers.map((x, idx) => (
          <div className="w-full flex mb-3" key={idx}>
            <CloseButton
              size="sm"
              variant="subtle"
              className="mr-2"
              onClick={() => {
                headers.splice(idx, 1);
                onUpdate();
              }}
            ></CloseButton>
            <Group className="flex w-full" grow>
              <TextInput
                required
                label="Key"
                description="Set the Header key"
                placeholder="MY_KEY"
                value={x.key}
                onChange={(v) => {
                  x.key = v.target.value;
                  onUpdate();
                }}
              />
              <TextInput
                required
                label="Value"
                description="Set the Header value"
                placeholder="my-value"
                value={x.value}
                onChange={(v) => {
                  x.value = v.target.value;
                  onUpdate();
                }}
              />
            </Group>
          </div>
        ))}
      </ItemMessage>
    </Group>
  );
};

const EnumSelect = (props: {
  label: string;
  description?: string;
  enumObj: Record<string, string | number>;
  value: number;
  options: { label: string; value: number }[];
  onChange: (v: number) => void;
}) => {
  const { label, description, enumObj, value, options, onChange } = props;

  return (
    <Select
      label={label}
      description={description}
      data={options.map((o) => ({
        label: o.label,
        value: String(enumObj[o.value]),
      }))}
      value={String(enumObj[value])}
      onChange={(v) => {
        if (v === null) {
          return;
        }
        onChange(Number(enumObj[v as keyof typeof enumObj]));
      }}
    />
  );
};

const Edit = (props: {
  item: EnterpriseP.CollectorExporter;
  onUpdate: (item: EnterpriseP.CollectorExporter) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(
    EnterpriseP.CollectorExporter.clone(item),
  );
  const [init] = React.useState(EnterpriseP.CollectorExporter.clone(req));
  const updateReq = () => {
    setReq(EnterpriseP.CollectorExporter.clone(req));
    onUpdate(req);
  };

  if (!req.spec) {
    return <></>;
  }

  const grpcInitAuth = (
    kind: "otlp" | "otlpHTTP" | "prometheusRemoteWrite",
  ): GrpcStyleAuth | undefined =>
    match(init.spec?.type)
      .with({ oneofKind: "otlp" }, (t) =>
        kind === "otlp"
          ? (t.otlp.auth as unknown as GrpcStyleAuth | undefined)
          : undefined,
      )
      .with({ oneofKind: "otlpHTTP" }, (t) =>
        kind === "otlpHTTP"
          ? (t.otlpHTTP.auth as unknown as GrpcStyleAuth | undefined)
          : undefined,
      )
      .with({ oneofKind: "prometheusRemoteWrite" }, (t) =>
        kind === "prometheusRemoteWrite"
          ? (t.prometheusRemoteWrite.auth as unknown as
              | GrpcStyleAuth
              | undefined)
          : undefined,
      )
      .otherwise(() => undefined);

  return (
    <div>
      <Group>
        <Switch
          label="Disabled"
          description="Disable/deactivate the Collector Exporter"
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </Group>
      <Tabs
        defaultValue={req.spec!.type.oneofKind}
        onChange={(v) => {
          match(v)
            .with("otlp", () => {
              match(init.spec!.type.oneofKind)
                .with("otlp", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlp",
                    otlp: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_OTLP,
                  };
                });
              updateReq();
            })
            .with("otlpHTTP", () => {
              match(init.spec!.type.oneofKind)
                .with("otlpHTTP", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlpHTTP",
                    otlpHTTP: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP,
                  };
                });
              updateReq();
            })
            .with("clickhouse", () => {
              match(init.spec!.type.oneofKind)
                .with("clickhouse", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "clickhouse",
                    clickhouse: {
                      password: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Clickhouse,
                  };
                });
              updateReq();
            })
            .with("prometheusRemoteWrite", () => {
              match(init.spec!.type.oneofKind)
                .with("prometheusRemoteWrite", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "prometheusRemoteWrite",
                    prometheusRemoteWrite: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite,
                  };
                });
              updateReq();
            })
            .with("elasticsearch", () => {
              match(init.spec!.type.oneofKind)
                .with("elasticsearch", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "elasticsearch",
                    elasticsearch: {
                      auth: {
                        type: {
                          oneofKind: "apiKey",
                          apiKey: freshSecret(),
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Elasticsearch,
                  };
                });
              updateReq();
            })
            .with("logzio", () => {
              match(init.spec!.type.oneofKind)
                .with("logzio", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "logzio",
                    logzio: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Logzio,
                  };
                });
              updateReq();
            })
            .with("influxDB", () => {
              match(init.spec!.type.oneofKind)
                .with("influxDB", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "influxDB",
                    influxDB: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_InfluxDB,
                  };
                });
              updateReq();
            })
            .with("kafka", () => {
              match(init.spec!.type.oneofKind)
                .with("kafka", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "kafka",
                    kafka: {
                      auth: {
                        type: {
                          oneofKind: "sasl",
                          sasl: {
                            username: "",
                            password: freshSecret(),
                            mechanism:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .PLAIN,
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Kafka,
                  };
                });
              updateReq();
            })
            .with("datadog", () => {
              match(init.spec!.type.oneofKind)
                .with("datadog", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "datadog",
                    datadog: {
                      api: {
                        key: freshSecret(),
                        site: "",
                        failOnInvalidKey: false,
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Datadog,
                  };
                });
              updateReq();
            })
            .with("splunk", () => {
              match(init.spec!.type.oneofKind)
                .with("splunk", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "splunk",
                    splunk: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Splunk,
                  };
                });
              updateReq();
            })
            .with("azureMonitor", () => {
              match(init.spec!.type.oneofKind)
                .with("azureMonitor", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "azureMonitor",
                    azureMonitor: {
                      connectionString: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_AzureMonitor,
                  };
                });
              updateReq();
            })
            .with("azureDataExplorer", () => {
              match(init.spec!.type.oneofKind)
                .with("azureDataExplorer", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "azureDataExplorer",
                    azureDataExplorer: {
                      auth: {
                        type: {
                          oneofKind: "servicePrincipal",
                          servicePrincipal: {
                            applicationID: "",
                            applicationKey: freshSecret(),
                            tenantID: "",
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_AzureDataExplorer,
                  };
                });
              updateReq();
            });
        }}
      >
        <Tabs.List>
          <Tabs.Tab value="otlp">OTLP</Tabs.Tab>
          <Tabs.Tab value="otlpHTTP">OTLP HTTP</Tabs.Tab>
          <Tabs.Tab value="clickhouse">ClickHouse</Tabs.Tab>
          <Tabs.Tab value="prometheusRemoteWrite">
            Prometheus Remote Write
          </Tabs.Tab>
          <Tabs.Tab value="elasticsearch">Elasticsearch</Tabs.Tab>
          <Tabs.Tab value="kafka">Kafka</Tabs.Tab>
          <Tabs.Tab value="datadog">Datadog</Tabs.Tab>
          <Tabs.Tab value="logzio">Logz.io</Tabs.Tab>
          <Tabs.Tab value="influxDB">InfluxDB</Tabs.Tab>
          <Tabs.Tab value="splunk">Splunk</Tabs.Tab>
          <Tabs.Tab value="azureMonitor">Azure Monitor</Tabs.Tab>
          <Tabs.Tab value="azureDataExplorer">Azure Data Explorer</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="otlp">
          {match(req.spec!.type)
            .with({ oneofKind: "otlp" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder="otlp-receiver.example.com:8443"
                    value={type.otlp.endpoint}
                    onChange={(v) => {
                      type.otlp.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                    }
                    value={type.otlp.compression}
                    options={[
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .GZIP,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .NONE,
                      },
                      {
                        label: "Snappy",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .SNAPPY,
                      },
                      {
                        label: "Zstd",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .ZSTD,
                      },
                    ]}
                    onChange={(c) => {
                      type.otlp.compression = c;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <Switch
                    label="No TLS"
                    description="Connect to the endpoint without TLS"
                    checked={type.otlp.tls?.insecure ?? false}
                    onChange={(v) => {
                      if (!type.otlp.tls) {
                        type.otlp.tls =
                          EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create();
                      }
                      type.otlp.tls.insecure = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Skip TLS Verify"
                    description="Do not verify the server certificate"
                    checked={type.otlp.tls?.insecureSkipVerify ?? false}
                    onChange={(v) => {
                      if (!type.otlp.tls) {
                        type.otlp.tls =
                          EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create();
                      }
                      type.otlp.tls.insecureSkipVerify = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Wait For Ready"
                    description="Block until the connection is ready"
                    checked={type.otlp.waitForReady}
                    onChange={(v) => {
                      type.otlp.waitForReady = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>

                <HeaderList
                  headers={type.otlp.headers}
                  makeHeader={() =>
                    EnterpriseP.CollectorExporter_Spec_OTLP_Header.create()
                  }
                  onUpdate={updateReq}
                />

                <SecretAuthTabs
                  auth={type.otlp.auth as unknown as GrpcStyleAuth | undefined}
                  ensureAuth={() => {
                    if (!type.otlp.auth) {
                      type.otlp.auth =
                        EnterpriseP.CollectorExporter_Spec_OTLP_Auth.create();
                    }
                    return type.otlp.auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("otlp")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="otlpHTTP">
          {match(req.spec!.type)
            .with({ oneofKind: "otlpHTTP" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="Base URL; /v1/logs and /v1/metrics are appended automatically"
                    placeholder="https://otlp-receiver.example.com"
                    value={type.otlpHTTP.endpoint}
                    onChange={(v) => {
                      type.otlpHTTP.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Logs Endpoint"
                    description="Override the logs endpoint"
                    placeholder="https://otlp-receiver.example.com/v1/logs"
                    value={type.otlpHTTP.logsEndpoint}
                    onChange={(v) => {
                      type.otlpHTTP.logsEndpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Endpoint"
                    description="Override the metrics endpoint"
                    placeholder="https://otlp-receiver.example.com/v1/metrics"
                    value={type.otlpHTTP.metricsEndpoint}
                    onChange={(v) => {
                      type.otlpHTTP.metricsEndpoint = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <EnumSelect
                    label="Encoding"
                    description="Set the payload encoding"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                    }
                    value={type.otlpHTTP.encoding}
                    options={[
                      {
                        label: "Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                            .PROTO,
                      },
                      {
                        label: "JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                            .JSON,
                      },
                    ]}
                    onChange={(e) => {
                      type.otlpHTTP.encoding = e;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Compression
                    }
                    value={type.otlpHTTP.compression}
                    options={[
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_OTLPHTTP_Compression.GZIP,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_OTLPHTTP_Compression.NONE,
                      },
                    ]}
                    onChange={(c) => {
                      type.otlpHTTP.compression = c;
                      updateReq();
                    }}
                  />
                </Group>

                <HeaderList
                  headers={type.otlpHTTP.headers}
                  makeHeader={() =>
                    EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Header.create()
                  }
                  onUpdate={updateReq}
                />

                <SecretAuthTabs
                  auth={
                    type.otlpHTTP.auth as unknown as GrpcStyleAuth | undefined
                  }
                  ensureAuth={() => {
                    if (!type.otlpHTTP.auth) {
                      type.otlpHTTP.auth =
                        EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth.create();
                    }
                    return type.otlpHTTP.auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("otlpHTTP")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="clickhouse">
          {match(req.spec!.type)
            .with({ oneofKind: "clickhouse" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder='"tcp://addr:port", "http://addr:port", "clickhouse://addr:port"'
                    value={type.clickhouse.endpoint}
                    onChange={(v) => {
                      type.clickhouse.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Username"
                    placeholder="clickhouse"
                    value={type.clickhouse.username}
                    onChange={(v) => {
                      type.clickhouse.username = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Database"
                    placeholder="otel"
                    value={type.clickhouse.database}
                    onChange={(v) => {
                      type.clickhouse.database = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Password Secret"
                    sel={type.clickhouse.password}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Logs Table Name"
                    placeholder="otel_logs"
                    value={type.clickhouse.logsTableName}
                    onChange={(v) => {
                      type.clickhouse.logsTableName = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Cluster Name"
                    placeholder="cluster"
                    value={type.clickhouse.clusterName}
                    onChange={(v) => {
                      type.clickhouse.clusterName = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_Clickhouse_Compression
                    }
                    value={type.clickhouse.compression}
                    options={[
                      {
                        label: "LZ4",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.LZ4,
                      },
                      {
                        label: "ZSTD",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.ZSTD,
                      },
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.GZIP,
                      },
                      {
                        label: "Deflate",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression
                            .DEFLATE,
                      },
                      {
                        label: "Br",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.BR,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.NONE,
                      },
                    ]}
                    onChange={(c) => {
                      type.clickhouse.compression = c;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Async Insert"
                    checked={type.clickhouse.asyncInsert}
                    onChange={(v) => {
                      type.clickhouse.asyncInsert = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Create Schema"
                    checked={type.clickhouse.createSchema}
                    onChange={(v) => {
                      type.clickhouse.createSchema = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="JSON"
                    checked={type.clickhouse.json}
                    onChange={(v) => {
                      type.clickhouse.json = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="prometheusRemoteWrite">
          {match(req.spec!.type)
            .with({ oneofKind: "prometheusRemoteWrite" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder="https://prometheus.example.com/api/v1/write"
                    value={type.prometheusRemoteWrite.endpoint}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Namespace"
                    placeholder="default"
                    value={type.prometheusRemoteWrite.namespace}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.namespace = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group>
                  <Switch
                    label="Disable Scope Info"
                    checked={type.prometheusRemoteWrite.disableScopeInfo}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.disableScopeInfo =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Send Metadata"
                    checked={type.prometheusRemoteWrite.sendMetadata}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.sendMetadata =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>

                <EnumSelect
                  label="Translation Strategy"
                  enumObj={
                    EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                  }
                  value={type.prometheusRemoteWrite.translationStrategy}
                  options={[
                    {
                      label: "Underscore Escaping With Suffixes",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                          .UNDERSCORE_ESCAPING_WITH_SUFFIXES,
                    },
                    {
                      label: "Underscore Escaping Without Suffixes",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                          .UNDERSCORE_ESCAPING_WITHOUT_SUFFIXES,
                    },
                  ]}
                  onChange={(s) => {
                    type.prometheusRemoteWrite.translationStrategy = s;
                    updateReq();
                  }}
                />

                <SecretAuthTabs
                  auth={
                    type.prometheusRemoteWrite.auth as unknown as
                      | GrpcStyleAuth
                      | undefined
                  }
                  ensureAuth={() => {
                    if (!type.prometheusRemoteWrite.auth) {
                      type.prometheusRemoteWrite.auth =
                        EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth.create();
                    }
                    return type.prometheusRemoteWrite
                      .auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("prometheusRemoteWrite")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="elasticsearch">
          {match(req.spec!.type)
            .with({ oneofKind: "elasticsearch" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    label="Endpoint"
                    placeholder="https://es.example.com:9200"
                    value={type.elasticsearch.endpoint}
                    onChange={(v) => {
                      type.elasticsearch.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Cloud ID"
                    placeholder="my-cloud-id"
                    value={type.elasticsearch.cloudID}
                    onChange={(v) => {
                      type.elasticsearch.cloudID = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <TextInput
                  label="Endpoints"
                  description="One or more endpoints separated by a comma"
                  placeholder="https://es1:9200, https://es2:9200"
                  value={type.elasticsearch.endpoints.join(",")}
                  onChange={(v) => {
                    type.elasticsearch.endpoints = v.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter((x) => x !== "");
                    updateReq();
                  }}
                />

                <Group grow>
                  <TextInput
                    label="Logs Index"
                    placeholder="my-log-index"
                    value={type.elasticsearch.logsIndex}
                    onChange={(v) => {
                      type.elasticsearch.logsIndex = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Index"
                    placeholder="my-metrics-index"
                    value={type.elasticsearch.metricsIndex}
                    onChange={(v) => {
                      type.elasticsearch.metricsIndex = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Pipeline"
                    placeholder="pipeline"
                    value={type.elasticsearch.pipeline}
                    onChange={(v) => {
                      type.elasticsearch.pipeline = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <EnumSelect
                  label="Compression"
                  enumObj={
                    EnterpriseP.CollectorExporter_Spec_Elasticsearch_Compression
                  }
                  value={type.elasticsearch.compression}
                  options={[
                    {
                      label: "Gzip",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_Elasticsearch_Compression
                          .GZIP,
                    },
                    {
                      label: "None",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_Elasticsearch_Compression
                          .NONE,
                    },
                  ]}
                  onChange={(c) => {
                    type.elasticsearch.compression = c;
                    updateReq();
                  }}
                />

                <Tabs
                  defaultValue={type.elasticsearch.auth?.type.oneofKind}
                  onChange={(v) => {
                    if (!type.elasticsearch.auth) {
                      type.elasticsearch.auth =
                        EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth.create();
                    }
                    match(v)
                      .with("apiKey", () => {
                        type.elasticsearch.auth!.type = {
                          oneofKind: "apiKey",
                          apiKey: freshSecret(),
                        };
                      })
                      .with("basic", () => {
                        type.elasticsearch.auth!.type = {
                          oneofKind: "basic",
                          basic: { user: "", password: freshSecret() },
                        };
                      })
                      .otherwise(() => {});
                    updateReq();
                  }}
                >
                  <Tabs.List>
                    <Tabs.Tab value="apiKey">API Key</Tabs.Tab>
                    <Tabs.Tab value="basic">Basic Authentication</Tabs.Tab>
                  </Tabs.List>

                  <Tabs.Panel value="apiKey">
                    {match(type.elasticsearch.auth?.type)
                      .with({ oneofKind: "apiKey" }, (a) => (
                        <SecretSelectField
                          label="API Key Secret"
                          sel={a.apiKey}
                          onUpdate={updateReq}
                        />
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="basic">
                    {match(type.elasticsearch.auth?.type)
                      .with({ oneofKind: "basic" }, (a) => (
                        <Group grow>
                          <TextInput
                            label="User"
                            placeholder="elastic"
                            value={a.basic.user}
                            onChange={(v) => {
                              a.basic.user = v.target.value;
                              updateReq();
                            }}
                          />
                          <SecretSelectField
                            label="Password Secret"
                            sel={a.basic.password}
                            onUpdate={updateReq}
                          />
                        </Group>
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>
                </Tabs>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="kafka">
          {match(req.spec!.type)
            .with({ oneofKind: "kafka" }, (type) => (
              <>
                <TextInput
                  label="Brokers"
                  description="One or more brokers separated by a comma"
                  placeholder="localhost:9092, localhost:9093"
                  value={type.kafka.brokers.join(",")}
                  onChange={(v) => {
                    type.kafka.brokers = v.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter((x) => x !== "");
                    updateReq();
                  }}
                />

                <Group grow>
                  <TextInput
                    label="Protocol Version"
                    placeholder="2.1.0"
                    value={type.kafka.protocolVersion}
                    onChange={(v) => {
                      type.kafka.protocolVersion = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Client ID"
                    placeholder="otel-collector"
                    value={type.kafka.clientID}
                    onChange={(v) => {
                      type.kafka.clientID = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Logs Topic"
                    placeholder="otlp_logs"
                    value={type.kafka.logs?.topic ?? ""}
                    onChange={(v) => {
                      if (!type.kafka.logs) {
                        type.kafka.logs =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.logs.topic = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Logs Encoding"
                    enumObj={EnterpriseP.CollectorExporter_Spec_Kafka_Encoding}
                    value={type.kafka.logs?.encoding ?? 0}
                    options={[
                      {
                        label: "OTLP Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_PROTO,
                      },
                      {
                        label: "OTLP JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_JSON,
                      },
                      {
                        label: "Raw",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding.RAW,
                      },
                    ]}
                    onChange={(e) => {
                      if (!type.kafka.logs) {
                        type.kafka.logs =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.logs.encoding = e;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Metrics Topic"
                    placeholder="otlp_metrics"
                    value={type.kafka.metrics?.topic ?? ""}
                    onChange={(v) => {
                      if (!type.kafka.metrics) {
                        type.kafka.metrics =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.metrics.topic = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Metrics Encoding"
                    enumObj={EnterpriseP.CollectorExporter_Spec_Kafka_Encoding}
                    value={type.kafka.metrics?.encoding ?? 0}
                    options={[
                      {
                        label: "OTLP Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_PROTO,
                      },
                      {
                        label: "OTLP JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_JSON,
                      },
                    ]}
                    onChange={(e) => {
                      if (!type.kafka.metrics) {
                        type.kafka.metrics =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.metrics.encoding = e;
                      updateReq();
                    }}
                  />
                </Group>

                {match(type.kafka.auth?.type)
                  .with({ oneofKind: "sasl" }, (a) => (
                    <Group grow>
                      <TextInput
                        label="SASL Username"
                        value={a.sasl.username}
                        onChange={(v) => {
                          a.sasl.username = v.target.value;
                          updateReq();
                        }}
                      />
                      <SecretSelectField
                        label="SASL Password Secret"
                        sel={a.sasl.password}
                        onUpdate={updateReq}
                      />
                      <EnumSelect
                        label="Mechanism"
                        enumObj={
                          EnterpriseP.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                        }
                        value={a.sasl.mechanism}
                        options={[
                          {
                            label: "PLAIN",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .PLAIN,
                          },
                          {
                            label: "SCRAM-SHA-256",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .SCRAM_SHA_256,
                          },
                          {
                            label: "SCRAM-SHA-512",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .SCRAM_SHA_512,
                          },
                        ]}
                        onChange={(m) => {
                          a.sasl.mechanism = m;
                          updateReq();
                        }}
                      />
                    </Group>
                  ))
                  .otherwise(() => (
                    <></>
                  ))}
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="datadog">
          {match(req.spec!.type)
            .with({ oneofKind: "datadog" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Site"
                    placeholder="datadoghq.com"
                    value={type.datadog.api?.site ?? ""}
                    onChange={(v) => {
                      if (!type.datadog.api) {
                        type.datadog.api =
                          EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
                      }
                      type.datadog.api.site = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="API Key Secret"
                    sel={type.datadog.api?.key}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Hostname"
                    value={type.datadog.hostname}
                    onChange={(v) => {
                      type.datadog.hostname = v.target.value;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Fail On Invalid Key"
                    checked={type.datadog.api?.failOnInvalidKey ?? false}
                    onChange={(v) => {
                      if (!type.datadog.api) {
                        type.datadog.api =
                          EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
                      }
                      type.datadog.api.failOnInvalidKey = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="logzio">
          {match(req.spec!.type)
            .with({ oneofKind: "logzio" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder="https://listener.logz.io:8053"
                    value={type.logzio.endpoint}
                    onChange={(v) => {
                      type.logzio.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Region"
                    placeholder="us"
                    value={type.logzio.region}
                    onChange={(v) => {
                      type.logzio.region = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.logzio.token}
                    onUpdate={updateReq}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="influxDB">
          {match(req.spec!.type)
            .with({ oneofKind: "influxDB" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder="https://influxdb.example.com:8086"
                    value={type.influxDB.endpoint}
                    onChange={(v) => {
                      type.influxDB.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.influxDB.token}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    required
                    label="Org"
                    value={type.influxDB.org}
                    onChange={(v) => {
                      type.influxDB.org = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    required
                    label="Bucket"
                    value={type.influxDB.bucket}
                    onChange={(v) => {
                      type.influxDB.bucket = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <EnumSelect
                    label="Metrics Schema"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_InfluxDB_MetricsSchema
                    }
                    value={type.influxDB.metricsSchema}
                    options={[
                      {
                        label: "Telegraf Prometheus V1",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_InfluxDB_MetricsSchema
                            .TELEGRAF_PROMETHEUS_V1,
                      },
                      {
                        label: "Telegraf Prometheus V2",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_InfluxDB_MetricsSchema
                            .TELEGRAF_PROMETHEUS_V2,
                      },
                    ]}
                    onChange={(s) => {
                      type.influxDB.metricsSchema = s;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Precision"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                    }
                    value={type.influxDB.precision}
                    options={[
                      {
                        label: "ns",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .NS,
                      },
                      {
                        label: "us",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .US,
                      },
                      {
                        label: "ms",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .MS,
                      },
                      {
                        label: "s",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .S,
                      },
                    ]}
                    onChange={(p) => {
                      type.influxDB.precision = p;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="splunk">
          {match(req.spec!.type)
            .with({ oneofKind: "splunk" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    placeholder="https://splunk.example.com:8088/services/collector"
                    value={type.splunk.endpoint}
                    onChange={(v) => {
                      type.splunk.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.splunk.token}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Source"
                    value={type.splunk.source}
                    onChange={(v) => {
                      type.splunk.source = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Source Type"
                    value={type.splunk.sourceType}
                    onChange={(v) => {
                      type.splunk.sourceType = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Index"
                    value={type.splunk.index}
                    onChange={(v) => {
                      type.splunk.index = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="App Name"
                    value={type.splunk.appName}
                    onChange={(v) => {
                      type.splunk.appName = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="App Version"
                    value={type.splunk.appVersion}
                    onChange={(v) => {
                      type.splunk.appVersion = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Use Multi Metric Format"
                    checked={type.splunk.useMultiMetricFormat}
                    onChange={(v) => {
                      type.splunk.useMultiMetricFormat = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Disable Compression"
                    checked={type.splunk.disableCompression}
                    onChange={(v) => {
                      type.splunk.disableCompression = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="azureMonitor">
          {match(req.spec!.type)
            .with({ oneofKind: "azureMonitor" }, (type) => (
              <>
                <Group grow>
                  <SecretSelectField
                    label="Connection String Secret"
                    sel={type.azureMonitor.connectionString}
                    onUpdate={updateReq}
                  />
                  <SecretSelectField
                    label="Instrumentation Key Secret"
                    sel={type.azureMonitor.instrumentationKey}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Endpoint"
                    value={type.azureMonitor.endpoint}
                    onChange={(v) => {
                      type.azureMonitor.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <NumberInput
                    label="Max Batch Size"
                    value={type.azureMonitor.maxBatchSize}
                    onChange={(v) => {
                      type.azureMonitor.maxBatchSize =
                        typeof v === "number" ? v : Number(v) || 0;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Custom Events Enabled"
                    checked={type.azureMonitor.customEventsEnabled}
                    onChange={(v) => {
                      type.azureMonitor.customEventsEnabled = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Exception Events Enabled"
                    checked={type.azureMonitor.exceptionEventsEnabled}
                    onChange={(v) => {
                      type.azureMonitor.exceptionEventsEnabled =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="azureDataExplorer">
          {match(req.spec!.type)
            .with({ oneofKind: "azureDataExplorer" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Cluster URI"
                    placeholder="https://cluster.region.kusto.windows.net"
                    value={type.azureDataExplorer.clusterURI}
                    onChange={(v) => {
                      type.azureDataExplorer.clusterURI = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Ingestion Type"
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_IngestionType
                    }
                    value={type.azureDataExplorer.ingestionType}
                    options={[
                      {
                        label: "Queued",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_AzureDataExplorer_IngestionType
                            .QUEUED,
                      },
                      {
                        label: "Managed",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_AzureDataExplorer_IngestionType
                            .MANAGED,
                      },
                    ]}
                    onChange={(t) => {
                      type.azureDataExplorer.ingestionType = t;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Database"
                    value={type.azureDataExplorer.database}
                    onChange={(v) => {
                      type.azureDataExplorer.database = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Logs Table"
                    value={type.azureDataExplorer.logsTable}
                    onChange={(v) => {
                      type.azureDataExplorer.logsTable = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Table"
                    value={type.azureDataExplorer.metricsTable}
                    onChange={(v) => {
                      type.azureDataExplorer.metricsTable = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Tabs
                  defaultValue={type.azureDataExplorer.auth?.type.oneofKind}
                  onChange={(v) => {
                    if (!type.azureDataExplorer.auth) {
                      type.azureDataExplorer.auth =
                        EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth.create();
                    }
                    match(v)
                      .with("servicePrincipal", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "servicePrincipal",
                          servicePrincipal: {
                            applicationID: "",
                            applicationKey: freshSecret(),
                            tenantID: "",
                          },
                        };
                      })
                      .with("managedIdentity", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "managedIdentity",
                          managedIdentity: { id: "" },
                        };
                      })
                      .with("azureDefault", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "azureDefault",
                          azureDefault: {},
                        };
                      })
                      .otherwise(() => {});
                    updateReq();
                  }}
                >
                  <Tabs.List>
                    <Tabs.Tab value="servicePrincipal">
                      Service Principal
                    </Tabs.Tab>
                    <Tabs.Tab value="managedIdentity">
                      Managed Identity
                    </Tabs.Tab>
                    <Tabs.Tab value="azureDefault">Azure Default</Tabs.Tab>
                  </Tabs.List>

                  <Tabs.Panel value="servicePrincipal">
                    {match(type.azureDataExplorer.auth?.type)
                      .with({ oneofKind: "servicePrincipal" }, (a) => (
                        <Group grow>
                          <TextInput
                            required
                            label="Application ID"
                            value={a.servicePrincipal.applicationID}
                            onChange={(v) => {
                              a.servicePrincipal.applicationID = v.target.value;
                              updateReq();
                            }}
                          />
                          <TextInput
                            required
                            label="Tenant ID"
                            value={a.servicePrincipal.tenantID}
                            onChange={(v) => {
                              a.servicePrincipal.tenantID = v.target.value;
                              updateReq();
                            }}
                          />
                          <SecretSelectField
                            label="Application Key Secret"
                            sel={a.servicePrincipal.applicationKey}
                            onUpdate={updateReq}
                          />
                        </Group>
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="managedIdentity">
                    {match(type.azureDataExplorer.auth?.type)
                      .with({ oneofKind: "managedIdentity" }, (a) => (
                        <TextInput
                          required
                          label="Managed Identity ID"
                          description='"system" or a client UUID'
                          value={a.managedIdentity.id}
                          onChange={(v) => {
                            a.managedIdentity.id = v.target.value;
                            updateReq();
                          }}
                        />
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="azureDefault">
                    <ItemMessage
                      title="Azure Default Credentials"
                      obj={{}}
                    ></ItemMessage>
                  </Tabs.Panel>
                </Tabs>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>
      </Tabs>
    </div>
  );
};

export default Edit;
