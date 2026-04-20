import * as React from "react";

import * as CoreP from "@/apis/corev1/corev1";

import EditItem from "@/components/EditItem";
import {
  CloseButton,
  Group,
  NumberInput,
  Select,
  Switch,
  TextInput,
} from "@mantine/core";

import Cond from "@/components/Condition";
import DurationPicker from "@/components/DurationPicker";
import ItemMessage from "@/components/ItemMessage";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import SelectSecret from "@/components/ResourceLayout/SelectSecret";
import TextAreaCustom from "@/components/TextAreaCustom";
import { strToNum } from "@/utils/convert";
import { Tabs } from "@mantine/core";
import { match } from "ts-pattern";

const Config = (props: {
  item: CoreP.Service_Spec_Config;

  onUpdate: (item: CoreP.Service_Spec_Config) => void;
  default?: boolean;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(CoreP.Service_Spec_Config.clone(item));
  const [init, setInit] = React.useState(CoreP.Service_Spec_Config.clone(item));

  React.useEffect(() => {
    setReq(CoreP.Service_Spec_Config.clone(item));
    setInit(CoreP.Service_Spec_Config.clone(item));
  }, [item]);

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
              defaultValue={req.upstream.type.oneofKind}
              onChange={(v) => {
                match(v)
                  .with("url", () => {
                    match(init.upstream?.type.oneofKind)
                      .with(`url`, () => {
                        req.upstream!.type = init!.upstream!.type;
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
                        req.upstream!.type = init!.upstream!.type;
                      })
                      .otherwise(() => {
                        req.upstream!.type = {
                          oneofKind: "container",
                          container:
                            CoreP.Service_Spec_Config_Upstream_Container.create(),
                        };
                      });

                    updateReq();
                  });
              }}
            >
              <Tabs.List>
                <Tabs.Tab value="url">URL</Tabs.Tab>
                <Tabs.Tab value="container">Managed Container</Tabs.Tab>
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
                                CoreP.Service_Spec_Config_Upstream_Container_Env.create(),
                              ];
                              updateReq();
                            }}
                            onAddListItem={() => {
                              container.container.env.push(
                                CoreP.Service_Spec_Config_Upstream_Container_Env.create(),
                              );
                              updateReq();
                            }}
                          >
                            {container.container.env.map((x, idx) => (
                              <Group key={idx} grow>
                                <div className="flex">
                                  <CloseButton
                                    size={"sm"}
                                    variant="subtle"
                                    onClick={() => {
                                      container.container.env.splice(idx, 1);
                                      updateReq();
                                    }}
                                  ></CloseButton>
                                  <TextInput
                                    required
                                    label="Key"
                                    description="Set the environment variable key"
                                    placeholder="MY_KEY"
                                    className="flex-1"
                                    value={
                                      container.container.env[idx].type
                                        .oneofKind === `value`
                                        ? container.container.env[idx].name
                                        : undefined
                                    }
                                    onChange={(v) => {
                                      container.container.env[idx].name =
                                        v.target.value;
                                      updateReq();
                                    }}
                                  />
                                </div>
                                <TextInput
                                  required
                                  label="Value"
                                  description="Set the environment variable value"
                                  placeholder="my-value"
                                  value={
                                    container.container.env[idx].type
                                      .oneofKind === `value`
                                      ? container.container.env[idx].type.value
                                      : undefined
                                  }
                                  onChange={(v) => {
                                    let g = container.container.env[idx]
                                      .type as {
                                      oneofKind: "value";
                                      value: string;
                                    };
                                    g.value = v.target.value;
                                    updateReq();
                                  }}
                                />
                              </Group>
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
                                        <SelectSecret
                                          api="core"
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
                                                x.fromSecret = val ?? "";
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
                                    cpu: {
                                      millicores: 0,
                                    },
                                    memory: {
                                      megabytes: 0,
                                    },
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
                                        v as number;
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
                                        v as number;
                                      updateReq();
                                    }}
                                  />
                                </Group>
                              </div>
                            )}
                          </EditItem>
                        </div>
                      );
                    },
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </Tabs.Panel>
            </Tabs>
          </>
        )}
      </EditItem>

      {match(req.type)
        .when(
          (x) => x.oneofKind === `kubernetes`,
          (kubernetes) => {
            return (
              <div>
                <Tabs
                  defaultValue={kubernetes.kubernetes.type.oneofKind}
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
                                  kubernetes.kubernetes.type = x.type;
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
                                  kubernetes.kubernetes.type = x.type;
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
                                (x) => x.type.oneofKind === `clientCertificate`,
                                (x) => {
                                  kubernetes.kubernetes.type = x.type;
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
                              <SelectSecret
                                api="core"
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
                                      x.fromSecret = v ?? "";
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
                              <SelectSecret
                                api="core"
                                defaultValue={
                                  bearerToken.bearerToken.type.oneofKind ===
                                  `fromSecret`
                                    ? bearerToken.bearerToken.type.fromSecret
                                    : undefined
                                }
                                onChange={(v) => {
                                  match(bearerToken.bearerToken.type).when(
                                    (x) => x.oneofKind === `fromSecret`,
                                    (x) => {
                                      x.fromSecret = v ?? "";
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
                          defaultValue={
                            CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode[
                              http.http.header.forwardedMode
                            ] ??
                            CoreP.Service_Spec_Config_HTTP_Header_ForwardedMode[
                              CoreP
                                .Service_Spec_Config_HTTP_Header_ForwardedMode
                                .OBFUSCATE
                            ]
                          }
                          onChange={(v) => {
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
                          defaultValue={
                            CoreP
                              .Service_Spec_Config_HTTP_Header_AuthorizationMode[
                              http.http.header.authorizationMode
                            ]
                          }
                          onChange={(v) => {
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
                            CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(),
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
                        {http.http.header!.addRequestHeaders.map((x, idx) => (
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
                                  http.http.header!.addRequestHeaders[idx].key
                                }
                                onChange={(v) => {
                                  http.http.header!.addRequestHeaders[idx].key =
                                    v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                required
                                label="Value"
                                description="Set the Header value"
                                placeholder="my-value"
                                value={match(
                                  http.http.header!.addRequestHeaders[idx].type,
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
                                    http.http.header!.addRequestHeaders[idx]
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
                        ))}
                      </ItemMessage>

                      <ItemMessage
                        title="Add Response Headers"
                        obj={http.http.header.addResponseHeaders}
                        isList
                        onSet={() => {
                          http.http.header!.addResponseHeaders = [
                            CoreP.Service_Spec_Config_HTTP_Header_KeyValue.create(),
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
                        {http.http.header!.addResponseHeaders.map((x, idx) => (
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
                                  http.http.header!.addResponseHeaders[idx].key
                                }
                                onChange={(v) => {
                                  http.http.header!.addResponseHeaders[
                                    idx
                                  ].key = v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                required
                                label="Value"
                                description="Set the Header value"
                                placeholder="my-value"
                                value={match(
                                  http.http.header!.addResponseHeaders[idx]
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
                                  let f = req.type as {
                                    oneofKind: "http";
                                    http: CoreP.Service_Spec_Config_HTTP;
                                  };

                                  match(
                                    f.http.header!.addResponseHeaders[idx].type,
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
                        ))}
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
                                    http.http.header!.removeRequestHeaders[idx]
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
                                    http.http.header!.removeResponseHeaders[idx]
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
                            CoreP.Service_Spec_Config_HTTP_Header_Host.create();

                          updateReq();
                        }}
                      >
                        {http.http.header.host && <div></div>}
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
                          http.http.body!.maxRequestSize = v as number;
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
                                CoreP.Service_Spec_Config_HTTP_Body_Mode.JSON
                              ],
                          },
                        ]}
                        value={
                          CoreP.Service_Spec_Config_HTTP_Body_Mode[
                            http.http.body!.mode
                          ]
                        }
                        onChange={(v) => {
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
                    http.http.auth = CoreP.Service_Spec_Config_HTTP_Auth.create(
                      {
                        type: {
                          oneofKind: `bearer`,
                          bearer: {
                            type: {
                              oneofKind: `fromSecret`,
                              fromSecret: ``,
                            },
                          },
                        },
                      },
                    );
                    updateReq();
                  }}
                >
                  {http.http.auth && (
                    <div>
                      <Tabs
                        defaultValue={http.http.auth!.type.oneofKind}
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
                                      ? init.type.http.auth!.type
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
                                      ? init.type.http.auth!.type
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
                                      ? init!.type.http.auth!.type
                                      : {
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
                                      ? init!.type.http.auth!.type
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
                                      ? init.type.http.auth!.type
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
                                    <SelectSecret
                                      api="core"
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
                                          (x) => x.oneofKind === `fromSecret`,
                                          (x) => {
                                            x.fromSecret = val ?? "";
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
                              (x) => x.oneofKind == `oauth2ClientCredentials`,
                              (oauth2ClientCredentials) => {
                                return (
                                  <Group grow>
                                    <TextInput
                                      required
                                      label="Client ID"
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
                                        .oauth2ClientCredentials.clientSecret
                                        ?.type,
                                    )
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          return (
                                            <SelectSecret
                                              api="core"
                                              label="Client Secret"
                                              description="Select the Secret of the OAuth2 client secret"
                                              defaultValue={x.fromSecret}
                                              onChange={(v) => {
                                                x.fromSecret = v ?? "";
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
                                      placeholder="user1234"
                                      value={basic.basic.username}
                                      onChange={(v) => {
                                        basic.basic.username = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    {match(basic.basic.password?.type)
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          return (
                                            <SelectSecret
                                              api="core"
                                              label="Password Secret"
                                              description="Select the Secret of the basic authentication password"
                                              defaultValue={x.fromSecret}
                                              onChange={(v) => {
                                                x.fromSecret = v ?? "";
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
                                      placeholder="X-CUSTOM-AUTH-HEADER"
                                      value={custom.custom.header}
                                      onChange={(v) => {
                                        custom.custom.header = v.target.value;
                                        updateReq();
                                      }}
                                    />
                                    {match(custom.custom.value?.type)
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          return (
                                            <SelectSecret
                                              api="core"
                                              label="Header value Secret"
                                              description="Select the Secret of the header value"
                                              defaultValue={x.fromSecret}
                                              onChange={(v) => {
                                                x.fromSecret = v ?? "";
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
                                      placeholder="s3"
                                      value={sigv4.sigv4.service}
                                      onChange={(v) => {
                                        sigv4.sigv4.service = v.target.value;
                                        updateReq();
                                      }}
                                    />

                                    {match(sigv4.sigv4.secretAccessKey?.type)
                                      .when(
                                        (x) => x?.oneofKind === `fromSecret`,
                                        (x) => {
                                          return (
                                            <SelectSecret
                                              api="core"
                                              label="Secret Access Key"
                                              description="Set the Secret of the Sigv4 Secret Access Key"
                                              defaultValue={x.fromSecret}
                                              onChange={(v) => {
                                                x.fromSecret = v ?? "";
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
                            http.http.retry!.maxRetries = v as number;

                            updateReq();
                          }}
                        />
                      </Group>

                      <Group grow>
                        <DurationPicker
                          value={http.http.retry!.initialInterval}
                          title="Initial interval"
                          onChange={(v) => {
                            http.http.retry!.initialInterval = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={http.http.retry!.maxInterval}
                          title="Max interval"
                          onChange={(v) => {
                            http.http.retry!.maxInterval = v;
                            updateReq();
                          }}
                        />

                        <DurationPicker
                          value={http.http.retry!.maxElapsedTime}
                          title="Max elapsed time"
                          onChange={(v) => {
                            http.http.retry!.maxElapsedTime = v;
                            updateReq();
                          }}
                        />
                      </Group>
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
                            http.http.cors!.allowCredentials = v.target.checked;
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
                          description="Set the allowed headers"
                          value={http.http.cors.maxAge}
                          onChange={(v) => {
                            http.http.cors!.maxAge = v.target.value;
                            updateReq();
                          }}
                        />
                      </Group>
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
                          checked={http.http.visibility!.enableRequestBodyMap}
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
                          checked={http.http.visibility!.enableResponseBodyMap}
                          description="Capture the response JSON body map"
                          onChange={(v) => {
                            http.http.visibility!.enableResponseBodyMap =
                              v.target.checked;
                            updateReq();
                          }}
                        />
                      </Group>
                      <Group grow></Group>
                    </div>
                  )}
                </EditItem>
                <ItemMessage
                  title="Plugins"
                  obj={http.http.plugins}
                  isList
                  onSet={() => {
                    http.http.plugins = [
                      CoreP.Service_Spec_Config_HTTP_Plugin.create(),
                    ];
                    updateReq();
                  }}
                  onAddListItem={() => {
                    http.http.plugins.push(
                      CoreP.Service_Spec_Config_HTTP_Plugin.create(),
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
                                  CoreP.Service_Spec_Config_HTTP_Plugin_Phase
                                    .POST_AUTH
                                ],
                            },
                            {
                              label: "Pre-Authorization",
                              value:
                                CoreP.Service_Spec_Config_HTTP_Plugin_Phase[
                                  CoreP.Service_Spec_Config_HTTP_Plugin_Phase
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
                        defaultValue={plugin.type.oneofKind}
                      >
                        <Tabs.List>
                          <Tabs.Tab value="direct">Direct Response</Tabs.Tab>
                          <Tabs.Tab value="rateLimit">Rate Limit</Tabs.Tab>
                          <Tabs.Tab value="cache">Cache</Tabs.Tab>
                        </Tabs.List>

                        <Tabs.Panel value="direct">
                          {match(plugin.type)
                            .when(
                              (x) => x.oneofKind === `direct`,
                              (direct) => <div></div>,
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
                        CoreP.Service_Spec_Config_SSH_UpstreamHostKey.create({
                          type: {
                            oneofKind: `key`,
                            key: "",
                          },
                        });

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
                    ssh.ssh.auth = CoreP.Service_Spec_Config_SSH_Auth.create();

                    updateReq();
                  }}
                >
                  {ssh.ssh.auth && (
                    <Tabs
                      className="mb-8"
                      defaultValue={ssh.ssh.auth!.type.oneofKind}
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
                                      <SelectSecret
                                        api="core"
                                        label="Password Secret"
                                        description="Select the Secret of the password"
                                        defaultValue={x.fromSecret}
                                        onChange={(v) => {
                                          x.fromSecret = v ?? "";
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
                                      <SelectSecret
                                        api="core"
                                        label="Private key Secret"
                                        description="Select the Secret of the private key"
                                        defaultValue={x.fromSecret}
                                        onChange={(v) => {
                                          x.fromSecret = v ?? "";
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
                            CoreP.Service_Spec_Config_Postgres_SSLMode.REQUIRE
                          ],
                      },
                      {
                        label: "Disable",
                        value:
                          CoreP.Service_Spec_Config_Postgres_SSLMode[
                            CoreP.Service_Spec_Config_Postgres_SSLMode.DISABLE
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
                                <SelectSecret
                                  api="core"
                                  label="Password Secret"
                                  description="Select the Secret of the Password"
                                  defaultValue={x.fromSecret}
                                  onChange={(v) => {
                                    x.fromSecret = v ?? "";
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
                    defaultChecked={mysql.mysql.isTLS}
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
                                <SelectSecret
                                  api="core"
                                  defaultValue={x.fromSecret}
                                  onChange={(v) => {
                                    x.fromSecret = v ?? "";
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
        .otherwise(() => (
          <></>
        ))}

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
                        <SelectSecret
                          api="core"
                          defaultValue={x.fromSecret}
                          onChange={(v) => {
                            x.fromSecret = v ?? "";
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
  const [init, _] = React.useState(CoreP.Service.clone(item));
  const updateReq = () => {
    setReq(CoreP.Service.clone(req));
    onUpdate(req);
  };

  return (
    <div>
      <Group grow>
        <Select
          label="Service Mode"
          required
          description="Set the Service mode (e.g. HTTP, SSH, PostgreSQL)"
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
          ]}
          value={CoreP.Service_Spec_Mode[req.spec!.mode]}
          onChange={(v) => {
            // @ts-ignore
            req.spec!.mode = CoreP.Service_Spec_Mode[v];

            match(req.spec!.mode)
              .when(
                (x) =>
                  x === CoreP.Service_Spec_Mode.HTTP ||
                  x === CoreP.Service_Spec_Mode.GRPC ||
                  x === CoreP.Service_Spec_Mode.WEB,
                () => {
                  match(init.spec!.config?.type)
                    .when(
                      (x) => x?.oneofKind === `http`,
                      (x) => {
                        req.spec!.config = init.spec!.config;
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

                  updateReq();
                },
              )
              .with(CoreP.Service_Spec_Mode.SSH, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `ssh`,
                    (x) => {
                      req.spec!.config = init.spec!.config;
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

                updateReq();
              })
              .with(CoreP.Service_Spec_Mode.POSTGRES, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `postgres`,
                    (x) => {
                      req.spec!.config = init.spec!.config;
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

                updateReq();
              })
              .with(CoreP.Service_Spec_Mode.MYSQL, () => {
                match(init.spec!.config?.type)
                  .when(
                    (x) => x?.oneofKind === `postgres`,
                    (x) => {
                      req.spec!.config = init.spec!.config;
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

                updateReq();
              })
              .with(CoreP.Service_Spec_Mode.KUBERNETES, () => {
                req.spec!.config = CoreP.Service_Spec_Config.create({
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
              .when(
                (x) =>
                  x === CoreP.Service_Spec_Mode.TCP ||
                  x === CoreP.Service_Spec_Mode.DNS ||
                  x === CoreP.Service_Spec_Mode.UDP,
                () => {
                  match(init.spec!.config)
                    .when(
                      (x) => !!x,
                      (x) => {
                        req.spec!.config = init.spec!.config;
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
                          oneofKind: undefined,
                        },
                      });
                    });

                  updateReq();
                },
              )
              .otherwise(() => {});
            updateReq();
          }}
        />

        <NumberInput
          label="Port"
          placeholder="8080"
          description="Port listen number"
          min={0}
          max={65535}
          value={req.spec!.port}
          onChange={(v) => {
            req.spec!.port = v as number;
            updateReq();
          }}
        />

        <Switch
          label="Disabled"
          description="Disable/deactivate the Service to stop serving requests"
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
        req.spec!.mode === CoreP.Service_Spec_Mode.KUBERNETES) && (
        <Group className="my-2" grow>
          <Switch
            className="my-2"
            label="Enable Public"
            description="This enables Client-less BeyondCorp mode"
            checked={req.spec!.isPublic}
            onChange={(v) => {
              req.spec!.isPublic = v.target.checked;
              updateReq();
            }}
          />

          <Switch
            label="Enable TLS"
            description="Set the Service to listen over TLS"
            checked={req.spec!.isTLS}
            onChange={(v) => {
              req.spec!.isTLS = v.target.checked;
              updateReq();
            }}
          />

          <Switch
            label="Enable Anonymous Access"
            description="Set the Service to be publicly and anonymously accessible over the internet"
            checked={req.spec!.isAnonymous}
            disabled={!req.spec?.isPublic}
            onChange={(v) => {
              req.spec!.isAnonymous = v.target.checked;
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
                req.spec!.config = x;
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
            default
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
                <div key={`${idx}`}>
                  <Config
                    item={req.spec!.dynamicConfig!.configs[idx]}
                    onUpdate={(v) => {
                      req.spec!.dynamicConfig!.configs[idx] = v;
                      updateReq();
                    }}
                  />
                </div>
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
