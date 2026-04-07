import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import ItemMessage from "@/components/ItemMessage";

import SelectSecret from "@/components/ResourceLayout/SelectSecret";
import {
  CloseButton,
  Group,
  Select,
  Switch,
  Tabs,
  TextInput,
} from "@mantine/core";
import * as React from "react";
import { match } from "ts-pattern";

const Edit = (props: {
  item: EnterpriseP.CollectorExporter;
  onUpdate: (item: EnterpriseP.CollectorExporter) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(
    EnterpriseP.CollectorExporter.clone(item),
  );
  let [init, _] = React.useState(EnterpriseP.CollectorExporter.clone(req));
  const updateReq = () => {
    setReq(EnterpriseP.CollectorExporter.clone(req));
    onUpdate(req);
  };

  if (!req.spec) {
    return <></>;
  }

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
            .with("clickhouse", () => {
              match(init.spec!.type.oneofKind)
                .with(`clickhouse`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "clickhouse",
                    clickhouse: {
                      password: {
                        type: {
                          oneofKind: "fromSecret",
                          fromSecret: "",
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Clickhouse,
                  };
                });

              updateReq();
            })
            .with("elasticsearch", () => {
              match(init.spec!.type.oneofKind)
                .with(`elasticsearch`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "elasticsearch",
                    elasticsearch: {
                      auth: {
                        type: {
                          oneofKind: `apiKey`,
                          apiKey: {
                            type: {
                              oneofKind: "fromSecret",
                              fromSecret: "",
                            },
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Elasticsearch,
                  };
                });

              updateReq();
            })
            .with("kafka", () => {
              match(init.spec!.type.oneofKind)
                .with(`kafka`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "kafka",
                    kafka: {
                      auth: {
                        type: {
                          oneofKind: "plain",
                          plain: {
                            password: {
                              type: {
                                oneofKind: "fromSecret",
                                fromSecret: "",
                              },
                            },
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
                .with(`datadog`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "datadog",
                    datadog: {
                      apiKey: {
                        type: {
                          oneofKind: "fromSecret",
                          fromSecret: "",
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Datadog,
                  };
                });

              updateReq();
            })
            .with("logzio", () => {
              match(init.spec!.type.oneofKind)
                .with(`logzio`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "logzio",
                    logzio: {
                      token: {
                        type: {
                          oneofKind: "fromSecret",
                          fromSecret: "",
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Logzio,
                  };
                });

              updateReq();
            })
            .with("otlp", () => {
              match(init.spec!.type.oneofKind)
                .with(`otlp`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlp",
                    otlp: {
                      auth: {
                        type: {
                          oneofKind: "bearer",
                          bearer: {
                            type: {
                              oneofKind: "fromSecret",
                              fromSecret: "",
                            },
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_OTLP,
                  };
                });

              updateReq();
            })
            .with("otlpHTTP", () => {
              match(init.spec!.type.oneofKind)
                .with(`otlpHTTP`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlpHTTP",
                    otlpHTTP: {
                      auth: {
                        type: {
                          oneofKind: "bearer",
                          bearer: {
                            type: {
                              oneofKind: "fromSecret",
                              fromSecret: "",
                            },
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP,
                  };
                });

              updateReq();
            })
            .with("prometheusRemoteWrite", () => {
              match(init.spec!.type.oneofKind)
                .with(`prometheusRemoteWrite`, () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "prometheusRemoteWrite",
                    prometheusRemoteWrite: {
                      auth: {
                        type: {
                          oneofKind: "bearer",
                          bearer: {
                            type: {
                              oneofKind: "fromSecret",
                              fromSecret: "",
                            },
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite,
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
          <Tabs.Tab value="kafka">Kafka</Tabs.Tab>
          <Tabs.Tab value="datadog">Datadog</Tabs.Tab>
          <Tabs.Tab value="logzio">Logz.io</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="clickhouse">
          {req.spec!.type.oneofKind === `clickhouse` && (
            <>
              <Group grow>
                <TextInput
                  required
                  label="Endpoint"
                  placeholder={`"tcp://addr1:port", "http://addr1:port", "clickhouse://addr1:port"`}
                  value={req.spec!.type.clickhouse.endpoint}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `clickhouse`,
                      (x) => {
                        x.clickhouse.endpoint = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
                <TextInput
                  required
                  label="Username"
                  placeholder={`clickhouse`}
                  value={req.spec!.type.clickhouse.username}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `clickhouse`,
                      (x) => {
                        x.clickhouse.username = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
              </Group>
              <Group grow>
                <TextInput
                  label="Database"
                  placeholder={`clickhouse`}
                  value={req.spec!.type.clickhouse.database}
                  onChange={(v) => {
                    match(req.spec?.type).when(
                      (x) => x?.oneofKind === `clickhouse`,
                      (x) => {
                        x.clickhouse.database = v.target.value;
                        updateReq();
                      },
                    );
                  }}
                />
                {req.spec!.type.clickhouse.password &&
                  req.spec.type.clickhouse.password.type.oneofKind ===
                    `fromSecret` && (
                    <SelectSecret
                      api="enterprise"
                      label="Password Secret"
                      defaultValue={
                        req.spec!.type.clickhouse.password!.type.fromSecret
                      }
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `clickhouse`,
                          (x) => {
                            match(x.clickhouse.password?.type).when(
                              (x) => x?.oneofKind === `fromSecret`,
                              (x) => {
                                x.fromSecret = v ?? "";
                                updateReq();
                              },
                            );
                          },
                        );
                      }}
                    />
                  )}
              </Group>
            </>
          )}
        </Tabs.Panel>

        <Tabs.Panel value="prometheusRemoteWrite">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `prometheusRemoteWrite`,
              (type) => (
                <>
                  <Group grow>
                    <TextInput
                      required
                      label="Endpoint"
                      placeholder={`"tcp://addr1:port", "http://addr1:port", "clickhouse://addr1:port"`}
                      value={type.prometheusRemoteWrite.endpoint}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `prometheusRemoteWrite`,
                          (x) => {
                            x.prometheusRemoteWrite.endpoint = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      required
                      label="Namespace"
                      placeholder={`default`}
                      value={type.prometheusRemoteWrite.namespace}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `prometheusRemoteWrite`,
                          (x) => {
                            x.prometheusRemoteWrite.namespace = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />
                  </Group>

                  <Tabs
                    defaultValue={
                      type.prometheusRemoteWrite.auth?.type.oneofKind
                    }
                    onChange={(v) => {
                      match(v)
                        .with("bearer", () => {
                          match(init.spec?.type)
                            .when(
                              (x) => x?.oneofKind === `prometheusRemoteWrite`,
                              (x) => {
                                match(x.prometheusRemoteWrite.auth?.type)
                                  .when(
                                    (x) => x?.oneofKind === `bearer`,
                                    (x) => {
                                      type.prometheusRemoteWrite.auth!.type = x;
                                      updateReq();
                                    },
                                  )
                                  .otherwise(() => {
                                    type.prometheusRemoteWrite.auth!.type = {
                                      oneofKind: "bearer",
                                      bearer: {
                                        type: {
                                          oneofKind: "fromSecret",
                                          fromSecret: "",
                                        },
                                      } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Bearer,
                                    };
                                    updateReq();
                                  });
                              },
                            )
                            .otherwise(() => {
                              type.prometheusRemoteWrite.auth!.type = {
                                oneofKind: "bearer",
                                bearer: {
                                  type: {
                                    oneofKind: "fromSecret",
                                    fromSecret: "",
                                  },
                                } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Bearer,
                              };
                              updateReq();
                            });
                        })
                        .with("basic", () => {
                          match(init.spec?.type)
                            .when(
                              (x) => x?.oneofKind === `prometheusRemoteWrite`,
                              (x) => {
                                match(x.prometheusRemoteWrite.auth?.type)
                                  .when(
                                    (x) => x?.oneofKind === `basic`,
                                    (x) => {
                                      type.prometheusRemoteWrite.auth!.type = x;
                                      updateReq();
                                    },
                                  )
                                  .otherwise(() => {
                                    type.prometheusRemoteWrite.auth!.type = {
                                      oneofKind: "basic",
                                      basic: {
                                        password: {
                                          type: {
                                            oneofKind: "fromSecret",
                                            fromSecret: "",
                                          },
                                        },
                                      } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic,
                                    };
                                    updateReq();
                                  });
                              },
                            )
                            .otherwise(() => {
                              type.prometheusRemoteWrite.auth!.type = {
                                oneofKind: "basic",
                                basic: {
                                  password: {
                                    type: {
                                      oneofKind: "fromSecret",
                                      fromSecret: "",
                                    },
                                  },
                                } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Basic,
                              };
                              updateReq();
                            });
                        })
                        .with("custom", () => {
                          match(init.spec?.type)
                            .when(
                              (x) => x?.oneofKind === `prometheusRemoteWrite`,
                              (x) => {
                                match(x.prometheusRemoteWrite.auth?.type)
                                  .when(
                                    (x) => x?.oneofKind === `custom`,
                                    (x) => {
                                      type.prometheusRemoteWrite.auth!.type = x;
                                      updateReq();
                                    },
                                  )
                                  .otherwise(() => {
                                    type.prometheusRemoteWrite.auth!.type = {
                                      oneofKind: "custom",
                                      custom: {
                                        value: {
                                          type: {
                                            oneofKind: "fromSecret",
                                            fromSecret: "",
                                          },
                                        },
                                      } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom,
                                    };
                                    updateReq();
                                  });
                              },
                            )
                            .otherwise(() => {
                              type.prometheusRemoteWrite.auth!.type = {
                                oneofKind: "custom",
                                custom: {
                                  value: {
                                    type: {
                                      oneofKind: "fromSecret",
                                      fromSecret: "",
                                    },
                                  },
                                } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom,
                              };
                              updateReq();
                            });
                        })
                        .otherwise(() => {
                          type.prometheusRemoteWrite.auth!.type = {
                            oneofKind: "custom",
                            custom: {
                              value: {
                                type: {
                                  oneofKind: "fromSecret",
                                  fromSecret: "",
                                },
                              },
                            } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth_Custom,
                          };
                          updateReq();
                        });
                    }}
                  >
                    <Tabs.List>
                      <Tabs.Tab value="bearer">Bearer Authentication</Tabs.Tab>
                      <Tabs.Tab value="basic">Basic Authentication</Tabs.Tab>
                      <Tabs.Tab value="custom">Custom Header</Tabs.Tab>
                    </Tabs.List>

                    <Tabs.Panel value="bearer">
                      {match(type.prometheusRemoteWrite.auth?.type)
                        .when(
                          (x) => x?.oneofKind === `bearer`,
                          (bearer) => {
                            return (
                              <SelectSecret
                                api="enterprise"
                                label="Bearer Token Secret"
                                defaultValue={match(bearer.bearer.type)
                                  .when(
                                    (x) => x.oneofKind === `fromSecret`,
                                    (x) => x.fromSecret,
                                  )
                                  .otherwise(() => undefined)}
                                onChange={(v) => {
                                  match(bearer.bearer.type).when(
                                    (x) => x.oneofKind === `fromSecret`,
                                    (x) => {
                                      x.fromSecret = v ?? "";
                                      updateReq();
                                    },
                                  );
                                }}
                              />
                            );
                          },
                        )
                        .otherwise(() => (
                          <></>
                        ))}
                    </Tabs.Panel>

                    <Tabs.Panel value="basic">
                      {match(type.prometheusRemoteWrite.auth?.type)
                        .when(
                          (x) => x?.oneofKind === `basic`,
                          (basic) => {
                            return (
                              <Group grow>
                                <TextInput
                                  label="Username"
                                  placeholder={`username`}
                                  value={basic.basic.user}
                                  onChange={(v) => {
                                    basic.basic.user = v.target.value;
                                    updateReq();
                                  }}
                                />

                                <SelectSecret
                                  api="enterprise"
                                  label="Password Secret"
                                  defaultValue={match(
                                    basic.basic.password?.type,
                                  )
                                    .when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => x.fromSecret,
                                    )
                                    .otherwise(() => undefined)}
                                  onChange={(v) => {
                                    match(basic.basic.password?.type).when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => {
                                        x.fromSecret = v ?? "";
                                        updateReq();
                                      },
                                    );
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

                    <Tabs.Panel value="custom">
                      {match(type.prometheusRemoteWrite.auth?.type)
                        .when(
                          (x) => x?.oneofKind === `custom`,
                          (custom) => {
                            return (
                              <Group grow>
                                <TextInput
                                  required
                                  label="Header"
                                  placeholder={`X-Custom-Auth`}
                                  value={custom.custom.header}
                                  onChange={(v) => {
                                    custom.custom.header = v.target.value;
                                    updateReq();
                                  }}
                                />

                                <SelectSecret
                                  api="enterprise"
                                  label="Header Value Secret"
                                  defaultValue={match(custom.custom.value?.type)
                                    .when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => x.fromSecret,
                                    )
                                    .otherwise(() => undefined)}
                                  onChange={(v) => {
                                    match(custom.custom.value?.type).when(
                                      (x) => x?.oneofKind === `fromSecret`,
                                      (x) => {
                                        x.fromSecret = v ?? "";
                                        updateReq();
                                      },
                                    );
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
                  </Tabs>
                </>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="elasticsearch">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `elasticsearch`,
              (type) => (
                <>
                  <Group grow>
                    <TextInput
                      label="Cloud ID"
                      placeholder={`my-cloud-id`}
                      value={type.elasticsearch.cloudID}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `elasticsearch`,
                          (x) => {
                            x.elasticsearch.cloudID = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      label="Logs Index"
                      placeholder={`my-log-index`}
                      value={type.elasticsearch.logsIndex}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `elasticsearch`,
                          (x) => {
                            x.elasticsearch.logsIndex = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      label="Pipeline"
                      placeholder={`pipeline`}
                      value={type.elasticsearch.pipeline}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `elasticsearch`,
                          (x) => {
                            x.elasticsearch.pipeline = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />
                  </Group>
                </>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="logzio">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `logzio`,
              (type) => (
                <>
                  <Group grow>
                    <TextInput
                      label="Endpoint"
                      placeholder="https://listener.logz.io:8053"
                      required
                      value={type.logzio.endpoint}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `logzio`,
                          (x) => {
                            x.logzio.endpoint = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      label="Region"
                      placeholder="us"
                      value={type.logzio.region}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `logzio`,
                          (x) => {
                            x.logzio.region = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    {type.logzio.token?.type.oneofKind === `fromSecret` && (
                      <SelectSecret
                        api="enterprise"
                        label="Token Secret"
                        defaultValue={type.logzio.token.type.fromSecret}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `logzio`,
                            (x) => {
                              match(x.logzio.token?.type).when(
                                (x) => x?.oneofKind === `fromSecret`,
                                (x) => {
                                  x.fromSecret = v ?? "";
                                  updateReq();
                                },
                              );
                            },
                          );
                        }}
                      />
                    )}
                  </Group>
                </>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="kafka">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `kafka`,
              (type) => (
                <>
                  <TextInput
                    label="Brokers"
                    description="Set one or more brokers separated by a comma"
                    placeholder={`localhost:9092, localhost:9093`}
                    value={type.kafka.brokers.join(",")}
                    onChange={(v) => {
                      match(req.spec?.type).when(
                        (x) => x?.oneofKind === `kafka`,
                        (x) => {
                          x.kafka.brokers = v.target.value
                            .split(",")
                            .map((x) => x.trim());
                          updateReq();
                        },
                      );
                    }}
                  />
                  <Group grow>
                    <TextInput
                      label="Topic"
                      placeholder={`otlp_logs`}
                      value={type.kafka.topic}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `kafka`,
                          (x) => {
                            x.kafka.topic = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      label="Encoding"
                      value={type.kafka.encoding}
                      placeholder={`"otlp_json" or ""otlp_proto`}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `kafka`,
                          (x) => {
                            x.kafka.encoding = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    <TextInput
                      label="Protocol Version"
                      value={type.kafka.protocolVersion}
                      placeholder="2.1.0"
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `kafka`,
                          (x) => {
                            x.kafka.protocolVersion = v.target.value;

                            updateReq();
                          },
                        );
                      }}
                    />
                  </Group>

                  {type.kafka.auth?.type.oneofKind === `plain` &&
                    type.kafka.auth.type.plain && (
                      <Group grow>
                        <TextInput
                          label="Username"
                          value={type.kafka.auth.type.plain.username}
                          onChange={(v) => {
                            match(req.spec?.type).when(
                              (x) => x?.oneofKind === `kafka`,
                              (x) => {
                                match(x.kafka.auth?.type).when(
                                  (x) => x?.oneofKind === `plain`,
                                  (x) => {
                                    x.plain.username = v.target.value;
                                    updateReq();
                                  },
                                );
                              },
                            );
                          }}
                        />
                        {type.kafka.auth.type.plain.password?.type.oneofKind ===
                          `fromSecret` && (
                          <SelectSecret
                            api="enterprise"
                            label="Token Secret"
                            defaultValue={
                              type.kafka.auth.type.plain.password.type
                                .fromSecret
                            }
                            onChange={(v) => {
                              match(req.spec?.type).when(
                                (x) => x?.oneofKind === `kafka`,
                                (x) => {
                                  match(x.kafka.auth?.type).when(
                                    (x) => x?.oneofKind === `plain`,
                                    (x) => {
                                      match(x.plain.password?.type).when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret = v ?? "";
                                          updateReq();
                                        },
                                      );
                                    },
                                  );
                                },
                              );
                            }}
                          />
                        )}
                      </Group>
                    )}
                </>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="datadog">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `datadog`,
              (type) => (
                <>
                  <Group grow>
                    <TextInput
                      label="Site"
                      required
                      value={type.datadog.site}
                      onChange={(v) => {
                        match(req.spec?.type).when(
                          (x) => x?.oneofKind === `datadog`,
                          (x) => {
                            x.datadog.site = v.target.value;
                            updateReq();
                          },
                        );
                      }}
                    />

                    {type.datadog.apiKey?.type.oneofKind === `fromSecret` && (
                      <SelectSecret
                        api="enterprise"
                        label="Bearer Token Secret"
                        defaultValue={match(type.datadog.apiKey.type)
                          .when(
                            (x) => x.oneofKind === `fromSecret`,
                            (x) => x.fromSecret,
                          )
                          .otherwise(() => undefined)}
                        onChange={(v) => {
                          match(type.datadog.apiKey?.type).when(
                            (x) => x?.oneofKind === `fromSecret`,
                            (x) => {
                              x.fromSecret = v ?? "";
                              updateReq();
                            },
                          );
                        }}
                      />
                    )}
                  </Group>
                </>
              ),
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="otlp">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `otlp`,
              (otlp) => {
                return (
                  <>
                    <Group grow>
                      <TextInput
                        required
                        label="Endpoint"
                        placeholder={`otlp-receiver.example.com:8443`}
                        value={otlp.otlp.endpoint}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `otlp`,
                            (x) => {
                              x.otlp.endpoint = v.target.value;
                              updateReq();
                            },
                          );
                        }}
                      />
                      <Switch
                        label="No TLS"
                        description="Disable using TLS"
                        checked={otlp.otlp.noTLS}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `otlp`,
                            (x) => {
                              x.otlp.noTLS = v.target.checked;
                              updateReq();
                            },
                          );
                        }}
                      />
                    </Group>

                    <Group grow>
                      <ItemMessage
                        title="Add Headers"
                        obj={otlp.otlp.headers}
                        isList
                        onSet={() => {
                          otlp.otlp.headers.push(
                            EnterpriseP.CollectorExporter_Spec_OTLP_KeyValue.create(),
                          );

                          updateReq();
                        }}
                        onAddListItem={() => {
                          otlp.otlp.headers.push(
                            EnterpriseP.CollectorExporter_Spec_OTLP_KeyValue.create(),
                          );

                          updateReq();
                        }}
                      >
                        {otlp.otlp.headers.map((x, idx) => (
                          <div className="w-full flex mb-3" key={idx}>
                            <CloseButton
                              size={"sm"}
                              variant="subtle"
                              className="mr-2"
                              onClick={() => {
                                otlp.otlp.headers.splice(idx, 1);
                                updateReq();
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
                                  updateReq();
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
                                  updateReq();
                                }}
                              />
                            </Group>
                          </div>
                        ))}
                      </ItemMessage>
                    </Group>

                    <Tabs
                      defaultValue={otlp.otlp.auth?.type.oneofKind}
                      onChange={(v) => {
                        match(v)
                          .with("bearer", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlp`,
                                (x) => {
                                  match(x.otlp.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `bearer`,
                                      (x) => {
                                        otlp.otlp.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlp.auth!.type = {
                                        oneofKind: "bearer",
                                        bearer: {
                                          type: {
                                            oneofKind: "fromSecret",
                                            fromSecret: "",
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Bearer,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlp.auth!.type = {
                                  oneofKind: "bearer",
                                  bearer: {
                                    type: {
                                      oneofKind: "fromSecret",
                                      fromSecret: "",
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Bearer,
                                };
                                updateReq();
                              });
                          })
                          .with("basic", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlp`,
                                (x) => {
                                  match(x.otlp.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `basic`,
                                      (x) => {
                                        otlp.otlp.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlp.auth!.type = {
                                        oneofKind: "basic",
                                        basic: {
                                          password: {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Basic,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlp.auth!.type = {
                                  oneofKind: "basic",
                                  basic: {
                                    password: {
                                      type: {
                                        oneofKind: "fromSecret",
                                        fromSecret: "",
                                      },
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Basic,
                                };
                                updateReq();
                              });
                          })
                          .with("custom", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlp`,
                                (x) => {
                                  match(x.otlp.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `custom`,
                                      (x) => {
                                        otlp.otlp.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlp.auth!.type = {
                                        oneofKind: "custom",
                                        custom: {
                                          value: {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Custom,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlp.auth!.type = {
                                  oneofKind: "custom",
                                  custom: {
                                    value: {
                                      type: {
                                        oneofKind: "fromSecret",
                                        fromSecret: "",
                                      },
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLP_Auth_Custom,
                                };
                                updateReq();
                              });
                          });
                      }}
                    >
                      <Tabs.List>
                        <Tabs.Tab value="bearer">
                          Bearer Authentication
                        </Tabs.Tab>
                        <Tabs.Tab value="basic">Basic Authentication</Tabs.Tab>
                        <Tabs.Tab value="custom">Custom Header</Tabs.Tab>
                      </Tabs.List>

                      <Tabs.Panel value="bearer">
                        {match(otlp.otlp.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `bearer`,
                            (bearer) => {
                              return (
                                <SelectSecret
                                  api="enterprise"
                                  label="Bearer Token Secret"
                                  defaultValue={match(bearer.bearer.type)
                                    .when(
                                      (x) => x.oneofKind === `fromSecret`,
                                      (x) => x.fromSecret,
                                    )
                                    .otherwise(() => undefined)}
                                  onChange={(v) => {
                                    match(bearer.bearer.type).when(
                                      (x) => x.oneofKind === `fromSecret`,
                                      (x) => {
                                        x.fromSecret = v ?? "";
                                        updateReq();
                                      },
                                    );
                                  }}
                                />
                              );
                            },
                          )
                          .otherwise(() => (
                            <></>
                          ))}
                      </Tabs.Panel>

                      <Tabs.Panel value="basic">
                        {match(otlp.otlp.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `basic`,
                            (basic) => {
                              return (
                                <Group grow>
                                  <TextInput
                                    label="Username"
                                    placeholder={`username`}
                                    value={basic.basic.username}
                                    onChange={(v) => {
                                      basic.basic.username = v.target.value;
                                      updateReq();
                                    }}
                                  />

                                  <SelectSecret
                                    api="enterprise"
                                    label="Password Secret"
                                    defaultValue={match(
                                      basic.basic.password?.type,
                                    )
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => x.fromSecret,
                                      )
                                      .otherwise(() => undefined)}
                                    onChange={(v) => {
                                      match(basic.basic.password?.type).when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret = v ?? "";
                                          updateReq();
                                        },
                                      );
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

                      <Tabs.Panel value="custom">
                        {match(otlp.otlp.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `custom`,
                            (custom) => {
                              return (
                                <Group grow>
                                  <TextInput
                                    required
                                    label="Header"
                                    placeholder={`X-Custom-Auth`}
                                    value={custom.custom.header}
                                    onChange={(v) => {
                                      custom.custom.header = v.target.value;
                                      updateReq();
                                    }}
                                  />

                                  <SelectSecret
                                    api="enterprise"
                                    label="Header Value Secret"
                                    defaultValue={match(
                                      custom.custom.value?.type,
                                    )
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => x.fromSecret,
                                      )
                                      .otherwise(() => undefined)}
                                    onChange={(v) => {
                                      match(custom.custom.value?.type).when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret = v ?? "";
                                          updateReq();
                                        },
                                      );
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
                    </Tabs>
                  </>
                );
              },
            )
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="otlpHTTP">
          {match(req.spec.type)
            .when(
              (x) => x.oneofKind === `otlpHTTP`,
              (otlp) => {
                return (
                  <>
                    <Group grow>
                      <TextInput
                        required
                        label="Endpoint"
                        placeholder={`https://otlp-receiver.example.com`}
                        description={`base URL where /v1/logs and /v1/metrics paths are automatically added to it`}
                        value={otlp.otlpHTTP.endpoint}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `otlpHTTP`,
                            (x) => {
                              x.otlpHTTP.endpoint = v.target.value;

                              updateReq();
                            },
                          );
                        }}
                      />
                    </Group>

                    <Group grow>
                      <TextInput
                        label="Log Endpoint"
                        description="Override the logs endpoint"
                        placeholder={`https://otlp-receiver.example.com/v1/logs`}
                        value={otlp.otlpHTTP.logsEndpoint}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `otlpHTTP`,
                            (x) => {
                              x.otlpHTTP.logsEndpoint = v.target.value;
                              updateReq();
                            },
                          );
                        }}
                      />

                      <TextInput
                        label="Metrics Endpoint"
                        description="Override the metrics endpoint"
                        placeholder={`https://otlp-receiver.example.com/v1/metrics`}
                        value={otlp.otlpHTTP.metricsEndpoint}
                        onChange={(v) => {
                          match(req.spec?.type).when(
                            (x) => x?.oneofKind === `otlpHTTP`,
                            (x) => {
                              x.otlpHTTP.metricsEndpoint = v.target.value;
                              updateReq();
                            },
                          );
                        }}
                      />
                    </Group>

                    <Group grow>
                      <Select
                        label="Mode"
                        description="Set the message mode"
                        data={[
                          {
                            label: "JSON",
                            value:
                              EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode[
                                EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode
                                  .JSON
                              ],
                          },
                          {
                            label: "Proto",
                            value:
                              EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode[
                                EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode
                                  .PROTO
                              ],
                          },
                        ]}
                        value={
                          EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode[
                            otlp.otlpHTTP.mode
                          ]
                        }
                        onChange={(v) => {
                          otlp.otlpHTTP.mode =
                            EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Mode[
                              v as "PROTO"
                            ];
                          updateReq();
                        }}
                      />

                      <Select
                        label="Compression"
                        description="Set the compression type"
                        data={[
                          {
                            label: "Gzip",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_OTLPHTTP_Compression[
                                EnterpriseP
                                  .CollectorExporter_Spec_OTLPHTTP_Compression
                                  .GZIP
                              ],
                          },
                          {
                            label: "None",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_OTLPHTTP_Compression[
                                EnterpriseP
                                  .CollectorExporter_Spec_OTLPHTTP_Compression
                                  .NONE
                              ],
                          },
                        ]}
                        value={
                          EnterpriseP
                            .CollectorExporter_Spec_OTLPHTTP_Compression[
                            otlp.otlpHTTP.compression
                          ]
                        }
                        onChange={(v) => {
                          otlp.otlpHTTP.compression =
                            EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Compression[
                              v as "NONE"
                            ];
                          updateReq();
                        }}
                      />
                    </Group>

                    <Group grow>
                      <ItemMessage
                        title="Add Headers"
                        obj={otlp.otlpHTTP.headers}
                        isList
                        onSet={() => {
                          otlp.otlpHTTP.headers.push(
                            EnterpriseP.CollectorExporter_Spec_OTLPHTTP_KeyValue.create(),
                          );

                          updateReq();
                        }}
                        onAddListItem={() => {
                          otlp.otlpHTTP.headers.push(
                            EnterpriseP.CollectorExporter_Spec_OTLPHTTP_KeyValue.create(),
                          );

                          updateReq();
                        }}
                      >
                        {otlp.otlpHTTP.headers.map((x, idx) => (
                          <div className="w-full flex mb-3" key={idx}>
                            <CloseButton
                              size={"sm"}
                              variant="subtle"
                              className="mr-2"
                              onClick={() => {
                                otlp.otlpHTTP.headers.splice(idx, 1);
                                updateReq();
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
                                  updateReq();
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
                                  updateReq();
                                }}
                              />
                            </Group>
                          </div>
                        ))}
                      </ItemMessage>
                    </Group>

                    <Tabs
                      defaultValue={otlp.otlpHTTP.auth?.type.oneofKind}
                      onChange={(v) => {
                        match(v)
                          .with("bearer", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlpHTTP`,
                                (x) => {
                                  match(x.otlpHTTP.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `bearer`,
                                      (x) => {
                                        otlp.otlpHTTP.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlpHTTP.auth!.type = {
                                        oneofKind: "bearer",
                                        bearer: {
                                          type: {
                                            oneofKind: "fromSecret",
                                            fromSecret: "",
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlpHTTP.auth!.type = {
                                  oneofKind: "bearer",
                                  bearer: {
                                    type: {
                                      oneofKind: "fromSecret",
                                      fromSecret: "",
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Bearer,
                                };
                                updateReq();
                              });
                          })
                          .with("basic", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlpHTTP`,
                                (x) => {
                                  match(x.otlpHTTP.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `basic`,
                                      (x) => {
                                        otlp.otlpHTTP.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlpHTTP.auth!.type = {
                                        oneofKind: "basic",
                                        basic: {
                                          password: {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Basic,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlpHTTP.auth!.type = {
                                  oneofKind: "basic",
                                  basic: {
                                    password: {
                                      type: {
                                        oneofKind: "fromSecret",
                                        fromSecret: "",
                                      },
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Basic,
                                };
                                updateReq();
                              });
                          })
                          .with("custom", () => {
                            match(init.spec?.type)
                              .when(
                                (x) => x?.oneofKind === `otlpHTTP`,
                                (x) => {
                                  match(x.otlpHTTP.auth?.type)
                                    .when(
                                      (x) => x?.oneofKind === `custom`,
                                      (x) => {
                                        otlp.otlpHTTP.auth!.type = x;
                                        updateReq();
                                      },
                                    )
                                    .otherwise(() => {
                                      otlp.otlpHTTP.auth!.type = {
                                        oneofKind: "custom",
                                        custom: {
                                          value: {
                                            type: {
                                              oneofKind: "fromSecret",
                                              fromSecret: "",
                                            },
                                          },
                                        } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Custom,
                                      };
                                      updateReq();
                                    });
                                },
                              )
                              .otherwise(() => {
                                otlp.otlpHTTP.auth!.type = {
                                  oneofKind: "custom",
                                  custom: {
                                    value: {
                                      type: {
                                        oneofKind: "fromSecret",
                                        fromSecret: "",
                                      },
                                    },
                                  } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth_Custom,
                                };
                                updateReq();
                              });
                          });
                      }}
                    >
                      <Tabs.List>
                        <Tabs.Tab value="bearer">
                          Bearer Authentication
                        </Tabs.Tab>
                        <Tabs.Tab value="basic">Basic Authentication</Tabs.Tab>
                        <Tabs.Tab value="custom">Custom Header</Tabs.Tab>
                      </Tabs.List>

                      <Tabs.Panel value="bearer">
                        {match(otlp.otlpHTTP.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `bearer`,
                            (bearer) => {
                              return (
                                <SelectSecret
                                  api="enterprise"
                                  label="Bearer Token Secret"
                                  defaultValue={match(bearer.bearer.type)
                                    .when(
                                      (x) => x.oneofKind === `fromSecret`,
                                      (x) => x.fromSecret,
                                    )
                                    .otherwise(() => undefined)}
                                  onChange={(v) => {
                                    match(bearer.bearer.type).when(
                                      (x) => x.oneofKind === `fromSecret`,
                                      (x) => {
                                        x.fromSecret = v ?? "";
                                        updateReq();
                                      },
                                    );
                                  }}
                                />
                              );
                            },
                          )
                          .otherwise(() => (
                            <></>
                          ))}
                      </Tabs.Panel>

                      <Tabs.Panel value="basic">
                        {match(otlp.otlpHTTP.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `basic`,
                            (basic) => {
                              return (
                                <Group grow>
                                  <TextInput
                                    label="Username"
                                    placeholder={`username`}
                                    value={basic.basic.username}
                                    onChange={(v) => {
                                      basic.basic.username = v.target.value;
                                      updateReq();
                                    }}
                                  />

                                  <SelectSecret
                                    api="enterprise"
                                    label="Password Secret"
                                    defaultValue={match(
                                      basic.basic.password?.type,
                                    )
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => x.fromSecret,
                                      )
                                      .otherwise(() => undefined)}
                                    onChange={(v) => {
                                      match(basic.basic.password?.type).when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret = v ?? "";
                                          updateReq();
                                        },
                                      );
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

                      <Tabs.Panel value="custom">
                        {match(otlp.otlpHTTP.auth?.type)
                          .when(
                            (x) => x?.oneofKind === `custom`,
                            (custom) => {
                              return (
                                <Group grow>
                                  <TextInput
                                    required
                                    label="Header"
                                    placeholder={`X-Custom-Auth`}
                                    value={custom.custom.header}
                                    onChange={(v) => {
                                      custom.custom.header = v.target.value;
                                      updateReq();
                                    }}
                                  />

                                  <SelectSecret
                                    api="enterprise"
                                    label="Header Value Secret"
                                    defaultValue={match(
                                      custom.custom.value?.type,
                                    )
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => x.fromSecret,
                                      )
                                      .otherwise(() => undefined)}
                                    onChange={(v) => {
                                      match(custom.custom.value?.type).when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          x.fromSecret = v ?? "";
                                          updateReq();
                                        },
                                      );
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
                    </Tabs>
                  </>
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
};

export default Edit;
