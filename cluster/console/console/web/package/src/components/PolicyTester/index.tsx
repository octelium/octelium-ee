import {
  RequestContext_Request,
  RequestContext_Request_GRPC,
  RequestContext_Request_HTTP,
  RequestContext_Request_Kubernetes,
  RequestContext_Request_Postgres,
  RequestContext_Request_SSH,
  RequestContext_Request_SSH_Connect,
} from "@/apis/corev1/corev1";
import {
  IsAuthorizedRequest,
  IsAuthorizedRequest_Additional,
  IsAuthorizedResponse,
} from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { onError } from "@/utils";
import { getClientPolicyPortal } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { Button, Group, Select, Tabs, TextInput } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getPolicyReason } from "../AccessLogViewer";
import EditItem from "../EditItem";
import SelectInlinePolicies from "../ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "../ResourceLayout/SelectPolicies";
import SelectResource from "../ResourceLayout/SelectResource";
import { ResourceListLabel } from "../ResourceList";
import AnimatedConnector from "./Connector";

const PolicyTester = (props: {}) => {
  let [req, setReq] = React.useState(
    IsAuthorizedRequest.create({
      downstream: {
        oneofKind: `userRef`,
        userRef: {} as ObjectReference,
      },
      upstream: {
        oneofKind: `serviceRef`,
        serviceRef: {} as ObjectReference,
      },
    }),
  );
  let [resp, setResp] = React.useState<IsAuthorizedResponse | undefined>(
    undefined,
  );

  const updateReq = () => {
    setReq(IsAuthorizedRequest.clone(req));
  };
  const mutation = useMutation({
    mutationFn: async () => {
      let r = await getClientPolicyPortal().isAuthorized(req);

      return r.response;
    },
    onSuccess: (r) => {
      setResp(r);
    },
    onError: onError,
  });

  return (
    <div className="w-full">
      <div className="w-full flex items-center my-8">
        <div className="font-bold text-lg mr-3 min-w-[120px]">Downstream</div>
        <div className="flex items-center">
          <Select
            defaultValue={`userRef`}
            data={[
              {
                label: "User",
                value: `userRef`,
              },
              {
                label: "Session",
                value: `sessionRef`,
              },
              {
                label: "Device",
                value: `deviceRef`,
              },
            ]}
            onChange={(v) => {
              if (!v) {
                return;
              }

              match(v)
                .with(`userRef`, () => {
                  req.downstream = {
                    oneofKind: "userRef",
                    userRef: {} as ObjectReference,
                  };
                })
                .with(`sessionRef`, () => {
                  req.downstream = {
                    oneofKind: "sessionRef",
                    sessionRef: {} as ObjectReference,
                  };
                })
                .with(`deviceRef`, () => {
                  req.downstream = {
                    oneofKind: "deviceRef",
                    deviceRef: {} as ObjectReference,
                  };
                });

              updateReq();
            }}
          />
        </div>
        <div className="w-full flex-1 ml-4">
          {match(req.downstream)
            .when(
              (v) => v.oneofKind === `userRef`,
              () => {
                return (
                  <>
                    <SelectResource
                      api="core"
                      kind="User"
                      onChange={(v) => {
                        if (!v) {
                          req.downstream = {
                            oneofKind: undefined,
                          };

                          updateReq();
                          return;
                        }
                        req.downstream = {
                          oneofKind: `userRef`,
                          userRef: getResourceRef(v),
                        };
                        updateReq();
                      }}
                    />
                  </>
                );
              },
            )
            .when(
              (v) => v.oneofKind === `sessionRef`,
              () => {
                return (
                  <>
                    <SelectResource
                      api="core"
                      kind="Session"
                      onChange={(v) => {
                        if (!v) {
                          req.downstream = {
                            oneofKind: undefined,
                          };

                          updateReq();
                          return;
                        }
                        req.downstream = {
                          oneofKind: `sessionRef`,
                          sessionRef: getResourceRef(v),
                        };
                        updateReq();
                      }}
                    />
                  </>
                );
              },
            )
            .when(
              (v) => v.oneofKind === `deviceRef`,
              () => {
                return (
                  <>
                    <SelectResource
                      api="core"
                      kind="Device"
                      onChange={(v) => {
                        if (!v) {
                          req.downstream = {
                            oneofKind: undefined,
                          };

                          updateReq();
                          return;
                        }
                        req.downstream = {
                          oneofKind: `deviceRef`,
                          deviceRef: getResourceRef(v),
                        };
                        updateReq();
                      }}
                    />
                  </>
                );
              },
            )
            .otherwise(() => (
              <></>
            ))}
        </div>

        <div></div>
      </div>

      <div className="w-full flex items-center">
        <div className="font-bold text-lg mr-3 min-w-[120px]">Resource</div>
        <div className="flex items-center">
          <Select
            defaultValue={`serviceRef`}
            data={[
              {
                label: "Service",
                value: `serviceRef`,
              },
              {
                label: "Namespace",
                value: `namespaceRef`,
              },
            ]}
            onChange={(v) => {
              if (!v) {
                return;
              }

              match(v)
                .with(`serviceRef`, () => {
                  req.upstream = {
                    oneofKind: `serviceRef`,
                    serviceRef: {} as ObjectReference,
                  };
                })
                .with(`namespaceRef`, () => {
                  req.upstream = {
                    oneofKind: "namespaceRef",
                    namespaceRef: {} as ObjectReference,
                  };
                });

              updateReq();
            }}
          />
        </div>
        <div className="w-full flex-1 ml-4">
          {match(req.upstream)
            .when(
              (v) => v.oneofKind === `serviceRef`,
              () => {
                return (
                  <>
                    <SelectResource
                      api="core"
                      kind="Service"
                      onChange={(v) => {
                        if (!v) {
                          req.upstream = {
                            oneofKind: undefined,
                          };

                          updateReq();
                          return;
                        }
                        req.upstream = {
                          oneofKind: `serviceRef`,
                          serviceRef: getResourceRef(v),
                        };
                        updateReq();
                      }}
                    />
                  </>
                );
              },
            )
            .when(
              (v) => v.oneofKind === `namespaceRef`,
              () => {
                return (
                  <>
                    <SelectResource
                      api="core"
                      kind="Namespace"
                      onChange={(v) => {
                        if (!v) {
                          req.upstream = {
                            oneofKind: undefined,
                          };

                          updateReq();
                          return;
                        }
                        req.upstream = {
                          oneofKind: `namespaceRef`,
                          namespaceRef: getResourceRef(v),
                        };
                        updateReq();
                      }}
                    />
                  </>
                );
              },
            )
            .otherwise(() => (
              <></>
            ))}
        </div>
      </div>

      <div className="w-full my-10">
        <EditItem
          title="Additional Policies"
          description="Test additional Policies/InLinePolicies without adding them to your Resources"
          obj={req.additional}
          onSet={() => {
            req.additional = IsAuthorizedRequest_Additional.create({});
            updateReq();
          }}
          onUnset={() => {
            req.additional = undefined;
            updateReq();
          }}
        >
          {req.additional && (
            <div className="w-full">
              <SelectPolicies
                policies={req.additional.policies}
                onUpdate={(v) => {
                  if (!v) {
                    req.additional!.policies = [];
                  } else {
                    req.additional!.policies = v;
                  }

                  updateReq();
                }}
              />

              <SelectInlinePolicies
                inlinePolicies={req.additional.inlinePolicies}
                onUpdate={(v) => {
                  req.additional!.inlinePolicies = v;
                  updateReq();
                }}
              />
            </div>
          )}
        </EditItem>
      </div>

      <div className="w-full my-10">
        <EditItem
          title="Context"
          obj={req.request}
          onSet={() => {
            req.request = RequestContext_Request.create({
              type: {
                oneofKind: `http`,
                http: RequestContext_Request_HTTP.create(),
              },
            });
            updateReq();
          }}
          onUnset={() => {
            req.request = undefined;
            updateReq();
          }}
        >
          {req.request && (
            <div className="w-full flex">
              <Tabs
                onChange={(v) => {
                  match(v)
                    .with(`http`, () => {
                      req.request = RequestContext_Request.create({
                        type: {
                          oneofKind: `http`,
                          http: RequestContext_Request_HTTP.create(),
                        },
                      });
                      updateReq();
                    })
                    .with(`kubernetes`, () => {
                      req.request = RequestContext_Request.create({
                        type: {
                          oneofKind: `kubernetes`,
                          kubernetes:
                            RequestContext_Request_Kubernetes.create(),
                        },
                      });
                      updateReq();
                    })
                    .with(`grpc`, () => {
                      req.request = RequestContext_Request.create({
                        type: {
                          oneofKind: `grpc`,
                          grpc: RequestContext_Request_GRPC.create(),
                        },
                      });
                      updateReq();
                    })
                    .with(`postgres`, () => {
                      req.request = RequestContext_Request.create({
                        type: {
                          oneofKind: `postgres`,
                          postgres: RequestContext_Request_Postgres.create(),
                        },
                      });
                      updateReq();
                    })

                    .with(`ssh`, () => {
                      req.request = RequestContext_Request.create({
                        type: {
                          oneofKind: `ssh`,
                          ssh: RequestContext_Request_SSH.create({
                            type: {
                              oneofKind: `connect`,
                              connect:
                                RequestContext_Request_SSH_Connect.create(),
                            },
                          }),
                        },
                      });
                      updateReq();
                    })
                    .otherwise(() => {});
                }}
                defaultValue="http"
              >
                <Tabs.List>
                  <Tabs.Tab value="http">HTTP</Tabs.Tab>
                  <Tabs.Tab value="kubernetes">Kubernetes</Tabs.Tab>
                  <Tabs.Tab value="grpc">gRPC</Tabs.Tab>
                  <Tabs.Tab value="ssh">SSH</Tabs.Tab>
                </Tabs.List>

                <Tabs.Panel value="http">
                  {match(req.request?.type)
                    .when(
                      (x) => x?.oneofKind === `http`,
                      (http) => {
                        return (
                          <div className="w-full">
                            <Group grow>
                              <TextInput
                                label="URI"
                                placeholder="/apis/v1/sessions?user=john"
                                value={http.http.uri}
                                onChange={(v) => {
                                  http.http.uri = v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Path"
                                placeholder="/apis/v1/sessions"
                                value={http.http.path}
                                onChange={(v) => {
                                  http.http.path = v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Method"
                                placeholder="GET"
                                value={http.http.method}
                                onChange={(v) => {
                                  http.http.method = v.target.value;
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
                </Tabs.Panel>

                <Tabs.Panel value="kubernetes">
                  {match(req.request?.type)
                    .when(
                      (x) => x?.oneofKind === `kubernetes`,
                      (kubernetes) => {
                        return (
                          <div className="w-full">
                            <Group grow>
                              <TextInput
                                label="Name"
                                placeholder="pod-123456"
                                value={kubernetes.kubernetes.name}
                                onChange={(v) => {
                                  kubernetes.kubernetes.name = v.target.value;
                                  updateReq();
                                }}
                              />

                              <TextInput
                                label="Resource"
                                placeholder="pods"
                                value={kubernetes.kubernetes.resource}
                                onChange={(v) => {
                                  kubernetes.kubernetes.resource =
                                    v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Sub-resource"
                                placeholder="portforward"
                                value={kubernetes.kubernetes.subresource}
                                onChange={(v) => {
                                  kubernetes.kubernetes.subresource =
                                    v.target.value;
                                  updateReq();
                                }}
                              />
                            </Group>
                            <Group grow>
                              <TextInput
                                label="API Group"
                                placeholder="rbac.authorization.k8s.io"
                                value={kubernetes.kubernetes.apiGroup}
                                onChange={(v) => {
                                  kubernetes.kubernetes.apiGroup =
                                    v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="API Version"
                                placeholder="apps/v1"
                                value={kubernetes.kubernetes.apiVersion}
                                onChange={(v) => {
                                  kubernetes.kubernetes.apiVersion =
                                    v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Namespace"
                                placeholder="kube-system"
                                value={kubernetes.kubernetes.namespace}
                                onChange={(v) => {
                                  kubernetes.kubernetes.namespace =
                                    v.target.value;
                                  updateReq();
                                }}
                              />

                              <TextInput
                                label="Verb"
                                placeholder="get"
                                value={kubernetes.kubernetes.verb}
                                onChange={(v) => {
                                  kubernetes.kubernetes.verb = v.target.value;
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
                </Tabs.Panel>

                <Tabs.Panel value="grpc">
                  {match(req.request?.type)
                    .when(
                      (x) => x?.oneofKind === `grpc`,
                      (grpc) => {
                        return (
                          <div className="w-full">
                            <Group grow>
                              <TextInput
                                label="Method"
                                placeholder="GetUser"
                                value={grpc.grpc.method}
                                onChange={(v) => {
                                  grpc.grpc.method = v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Package"
                                placeholder="octelium.api.main.core.v1"
                                value={grpc.grpc.package}
                                onChange={(v) => {
                                  grpc.grpc.package = v.target.value;
                                  updateReq();
                                }}
                              />
                            </Group>
                            <Group grow>
                              <TextInput
                                label="Service"
                                placeholder="MainService"
                                value={grpc.grpc.service}
                                onChange={(v) => {
                                  grpc.grpc.service = v.target.value;
                                  updateReq();
                                }}
                              />
                              <TextInput
                                label="Service Full Name"
                                placeholder="octelium.api.main.core.v1.MainService"
                                value={grpc.grpc.serviceFullName}
                                onChange={(v) => {
                                  grpc.grpc.serviceFullName = v.target.value;
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
                </Tabs.Panel>

                <Tabs.Panel value="ssh">
                  {match(req.request?.type)
                    .when(
                      (x) => x?.oneofKind === `ssh`,
                      (ssh) => {
                        return (
                          <div className="w-full">
                            {match(ssh.ssh.type)
                              .when(
                                (x) => x.oneofKind === `connect`,
                                (connect) => {
                                  return (
                                    <div>
                                      <TextInput
                                        label="User"
                                        placeholder="root"
                                        value={connect.connect.user}
                                        onChange={(v) => {
                                          connect.connect.user = v.target.value;
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
      </div>

      <div className="mt-8 w-full flex items-end justify-end">
        <Button
          size="xl"
          onClick={(v) => {
            mutation.mutate();
          }}
          loading={mutation.isPending}
        >
          Test
        </Button>
      </div>

      {resp && mutation.isSuccess && (
        <div className="my-8">
          <div className="w-full flex items-center justify-center">
            {req.downstream && (
              <div className="my-4">
                {match(req.downstream)
                  .when(
                    (x) => x.oneofKind === `userRef`,
                    (x) => {
                      return (
                        <ResourceListLabel
                          itemRef={x.userRef}
                        ></ResourceListLabel>
                      );
                    },
                  )
                  .when(
                    (x) => x.oneofKind === `sessionRef`,
                    (x) => {
                      return (
                        <ResourceListLabel
                          itemRef={x.sessionRef}
                        ></ResourceListLabel>
                      );
                    },
                  )
                  .when(
                    (x) => x.oneofKind === `deviceRef`,
                    (x) => {
                      return (
                        <ResourceListLabel
                          itemRef={x.deviceRef}
                        ></ResourceListLabel>
                      );
                    },
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </div>
            )}

            <AnimatedConnector className="w-[200px]" />

            {req.upstream && (
              <div className="my-4">
                {match(req.upstream)
                  .when(
                    (x) => x.oneofKind === `serviceRef`,
                    (x) => {
                      return (
                        <ResourceListLabel
                          itemRef={x.serviceRef}
                        ></ResourceListLabel>
                      );
                    },
                  )
                  .when(
                    (x) => x.oneofKind === `namespaceRef`,
                    (x) => {
                      return (
                        <ResourceListLabel
                          itemRef={x.namespaceRef}
                        ></ResourceListLabel>
                      );
                    },
                  )
                  .otherwise(() => (
                    <></>
                  ))}
              </div>
            )}
          </div>

          <div
            className={twMerge(
              resp.isAuthorized ? `bg-green-800` : `bg-red-600`,
              `font-bold text-white p-2 rounded-lg shadow`,
            )}
          >
            <div className="w-full mb-8 text-2xl">
              {resp.isAuthorized ? `Authorized` : `Unauthorized`}
            </div>

            <div>{getPolicyReason(resp.reason?.type)}</div>
            {match(resp.reason?.details?.type)
              .when(
                (c) => c?.oneofKind === `policyMatch`,
                (c) => {
                  return (
                    <div>
                      {c.policyMatch.type.oneofKind === `policy` && (
                        <ResourceListLabel
                          label="Policy"
                          itemRef={c.policyMatch.type.policy.policyRef}
                        ></ResourceListLabel>
                      )}
                      {c.policyMatch.type.oneofKind === `inlinePolicy` && (
                        <ResourceListLabel
                          label="Inline Policy"
                          itemRef={c.policyMatch.type.inlinePolicy.resourceRef}
                        ></ResourceListLabel>
                      )}
                    </div>
                  );
                },
              )
              .otherwise(() => (
                <></>
              ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default PolicyTester;
