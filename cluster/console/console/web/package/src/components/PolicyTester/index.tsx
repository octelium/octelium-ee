import {
  RequestContext_Request,
  RequestContext_Request_GRPC,
  RequestContext_Request_HTTP,
  RequestContext_Request_Kubernetes,
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
import { Group, Select, Tabs, TextInput } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight, Loader2, ShieldCheck, ShieldX } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getPolicyReason } from "../AccessLogViewer/Old";
import EditItem from "../EditItem";
import SelectInlinePolicies from "../ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "../ResourceLayout/SelectPolicies";
import SelectResource from "../ResourceLayout/SelectResource";
import { ResourceListLabel } from "../ResourceList";
import AnimatedConnector from "./Connector";

const SectionLabel = ({ children }: { children: React.ReactNode }) => (
  <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500 shrink-0 w-[110px]">
    {children}
  </span>
);

const PolicyTester = () => {
  const [req, setReq] = React.useState(
    IsAuthorizedRequest.create({
      downstream: { oneofKind: "userRef", userRef: {} as ObjectReference },
      upstream: { oneofKind: "serviceRef", serviceRef: {} as ObjectReference },
    }),
  );
  const [resp, setResp] = React.useState<IsAuthorizedResponse | undefined>(
    undefined,
  );

  const updateReq = () => setReq(IsAuthorizedRequest.clone(req));

  const mutation = useMutation({
    mutationFn: async () =>
      (await getClientPolicyPortal().isAuthorized(req)).response,
    onSuccess: setResp,
    onError,
  });

  return (
    <div className="w-full flex flex-col gap-6">
      <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-[0_1px_4px_rgba(15,23,42,0.06)]">
        <div className="px-5 py-3 border-b border-slate-100 bg-slate-50/60">
          <span className="text-[0.72rem] font-bold uppercase tracking-[0.05em] text-slate-500">
            Authorization request
          </span>
        </div>

        <div className="px-5 py-4 flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <SectionLabel>Downstream</SectionLabel>
            <Select
              w={140}
              size="xs"
              defaultValue="userRef"
              data={[
                { label: "User", value: "userRef" },
                { label: "Session", value: "sessionRef" },
                { label: "Device", value: "deviceRef" },
              ]}
              onChange={(v) => {
                if (!v) return;
                req.downstream = match(v)
                  .with("userRef", () => ({
                    oneofKind: "userRef" as const,
                    userRef: {} as ObjectReference,
                  }))
                  .with("sessionRef", () => ({
                    oneofKind: "sessionRef" as const,
                    sessionRef: {} as ObjectReference,
                  }))
                  .with("deviceRef", () => ({
                    oneofKind: "deviceRef" as const,
                    deviceRef: {} as ObjectReference,
                  }))
                  .otherwise(() => ({ oneofKind: undefined }));
                updateReq();
              }}
            />
            <div className="flex-1">
              {match(req.downstream)
                .when(
                  (v) => v.oneofKind === "userRef",
                  () => (
                    <SelectResource
                      api="core"
                      kind="User"
                      onChange={(v) => {
                        req.downstream = v
                          ? { oneofKind: "userRef", userRef: getResourceRef(v) }
                          : { oneofKind: undefined };
                        updateReq();
                      }}
                    />
                  ),
                )
                .when(
                  (v) => v.oneofKind === "sessionRef",
                  () => (
                    <SelectResource
                      api="core"
                      kind="Session"
                      onChange={(v) => {
                        req.downstream = v
                          ? {
                              oneofKind: "sessionRef",
                              sessionRef: getResourceRef(v),
                            }
                          : { oneofKind: undefined };
                        updateReq();
                      }}
                    />
                  ),
                )
                .when(
                  (v) => v.oneofKind === "deviceRef",
                  () => (
                    <SelectResource
                      api="core"
                      kind="Device"
                      onChange={(v) => {
                        req.downstream = v
                          ? {
                              oneofKind: "deviceRef",
                              deviceRef: getResourceRef(v),
                            }
                          : { oneofKind: undefined };
                        updateReq();
                      }}
                    />
                  ),
                )
                .otherwise(() => null)}
            </div>
          </div>

          <div className="flex items-center gap-3">
            <SectionLabel>Resource</SectionLabel>
            <Select
              w={140}
              size="xs"
              defaultValue="serviceRef"
              data={[
                { label: "Service", value: "serviceRef" },
                { label: "Namespace", value: "namespaceRef" },
              ]}
              onChange={(v) => {
                if (!v) return;
                req.upstream = match(v)
                  .with("serviceRef", () => ({
                    oneofKind: "serviceRef" as const,
                    serviceRef: {} as ObjectReference,
                  }))
                  .with("namespaceRef", () => ({
                    oneofKind: "namespaceRef" as const,
                    namespaceRef: {} as ObjectReference,
                  }))
                  .otherwise(() => ({ oneofKind: undefined }));
                updateReq();
              }}
            />
            <div className="flex-1">
              {match(req.upstream)
                .when(
                  (v) => v.oneofKind === "serviceRef",
                  () => (
                    <SelectResource
                      api="core"
                      kind="Service"
                      onChange={(v) => {
                        req.upstream = v
                          ? {
                              oneofKind: "serviceRef",
                              serviceRef: getResourceRef(v),
                            }
                          : { oneofKind: undefined };
                        updateReq();
                      }}
                    />
                  ),
                )
                .when(
                  (v) => v.oneofKind === "namespaceRef",
                  () => (
                    <SelectResource
                      api="core"
                      kind="Namespace"
                      onChange={(v) => {
                        req.upstream = v
                          ? {
                              oneofKind: "namespaceRef",
                              namespaceRef: getResourceRef(v),
                            }
                          : { oneofKind: undefined };
                        updateReq();
                      }}
                    />
                  ),
                )
                .otherwise(() => null)}
            </div>
          </div>
        </div>
      </div>

      <EditItem
        title="Additional policies"
        description="Test additional policies without attaching them to resources"
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
          <div className="flex flex-col gap-3">
            <SelectPolicies
              policies={req.additional.policies}
              onUpdate={(v) => {
                req.additional!.policies = v ?? [];
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

      <EditItem
        title="Request context"
        obj={req.request}
        onSet={() => {
          req.request = RequestContext_Request.create({
            type: {
              oneofKind: "http",
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
          <Tabs
            defaultValue="http"
            onChange={(v) => {
              req.request = match(v)
                .with("http", () =>
                  RequestContext_Request.create({
                    type: {
                      oneofKind: "http",
                      http: RequestContext_Request_HTTP.create(),
                    },
                  }),
                )
                .with("kubernetes", () =>
                  RequestContext_Request.create({
                    type: {
                      oneofKind: "kubernetes",
                      kubernetes: RequestContext_Request_Kubernetes.create(),
                    },
                  }),
                )
                .with("grpc", () =>
                  RequestContext_Request.create({
                    type: {
                      oneofKind: "grpc",
                      grpc: RequestContext_Request_GRPC.create(),
                    },
                  }),
                )
                .with("ssh", () =>
                  RequestContext_Request.create({
                    type: {
                      oneofKind: "ssh",
                      ssh: RequestContext_Request_SSH.create({
                        type: {
                          oneofKind: "connect",
                          connect: RequestContext_Request_SSH_Connect.create(),
                        },
                      }),
                    },
                  }),
                )
                .otherwise(() => undefined);
              updateReq();
            }}
          >
            <Tabs.List mb="md">
              <Tabs.Tab value="http">HTTP</Tabs.Tab>
              <Tabs.Tab value="kubernetes">Kubernetes</Tabs.Tab>
              <Tabs.Tab value="grpc">gRPC</Tabs.Tab>
              <Tabs.Tab value="ssh">SSH</Tabs.Tab>
            </Tabs.List>

            <Tabs.Panel value="http">
              {match(req.request?.type)
                .when(
                  (x) => x?.oneofKind === "http",
                  (t) => (
                    <Group grow>
                      <TextInput
                        label="URI"
                        placeholder="/apis/v1/sessions?user=john"
                        value={t.http.uri}
                        onChange={(e) => {
                          t.http.uri = e.target.value;
                          updateReq();
                        }}
                      />
                      <TextInput
                        label="Path"
                        placeholder="/apis/v1/sessions"
                        value={t.http.path}
                        onChange={(e) => {
                          t.http.path = e.target.value;
                          updateReq();
                        }}
                      />
                      <TextInput
                        label="Method"
                        placeholder="GET"
                        value={t.http.method}
                        onChange={(e) => {
                          t.http.method = e.target.value;
                          updateReq();
                        }}
                      />
                    </Group>
                  ),
                )
                .otherwise(() => null)}
            </Tabs.Panel>

            <Tabs.Panel value="kubernetes">
              {match(req.request?.type)
                .when(
                  (x) => x?.oneofKind === "kubernetes",
                  (t) => (
                    <div className="flex flex-col gap-3">
                      <Group grow>
                        <TextInput
                          label="Name"
                          placeholder="pod-123456"
                          value={t.kubernetes.name}
                          onChange={(e) => {
                            t.kubernetes.name = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Resource"
                          placeholder="pods"
                          value={t.kubernetes.resource}
                          onChange={(e) => {
                            t.kubernetes.resource = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Sub-resource"
                          placeholder="portforward"
                          value={t.kubernetes.subresource}
                          onChange={(e) => {
                            t.kubernetes.subresource = e.target.value;
                            updateReq();
                          }}
                        />
                      </Group>
                      <Group grow>
                        <TextInput
                          label="API Group"
                          placeholder="rbac.authorization.k8s.io"
                          value={t.kubernetes.apiGroup}
                          onChange={(e) => {
                            t.kubernetes.apiGroup = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="API Version"
                          placeholder="apps/v1"
                          value={t.kubernetes.apiVersion}
                          onChange={(e) => {
                            t.kubernetes.apiVersion = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Namespace"
                          placeholder="kube-system"
                          value={t.kubernetes.namespace}
                          onChange={(e) => {
                            t.kubernetes.namespace = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Verb"
                          placeholder="get"
                          value={t.kubernetes.verb}
                          onChange={(e) => {
                            t.kubernetes.verb = e.target.value;
                            updateReq();
                          }}
                        />
                      </Group>
                    </div>
                  ),
                )
                .otherwise(() => null)}
            </Tabs.Panel>

            <Tabs.Panel value="grpc">
              {match(req.request?.type)
                .when(
                  (x) => x?.oneofKind === "grpc",
                  (t) => (
                    <div className="flex flex-col gap-3">
                      <Group grow>
                        <TextInput
                          label="Method"
                          placeholder="GetUser"
                          value={t.grpc.method}
                          onChange={(e) => {
                            t.grpc.method = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Package"
                          placeholder="octelium.api.main.core.v1"
                          value={t.grpc.package}
                          onChange={(e) => {
                            t.grpc.package = e.target.value;
                            updateReq();
                          }}
                        />
                      </Group>
                      <Group grow>
                        <TextInput
                          label="Service"
                          placeholder="MainService"
                          value={t.grpc.service}
                          onChange={(e) => {
                            t.grpc.service = e.target.value;
                            updateReq();
                          }}
                        />
                        <TextInput
                          label="Service full name"
                          placeholder="octelium.api.main.core.v1.MainService"
                          value={t.grpc.serviceFullName}
                          onChange={(e) => {
                            t.grpc.serviceFullName = e.target.value;
                            updateReq();
                          }}
                        />
                      </Group>
                    </div>
                  ),
                )
                .otherwise(() => null)}
            </Tabs.Panel>

            <Tabs.Panel value="ssh">
              {match(req.request?.type)
                .when(
                  (x) => x?.oneofKind === "ssh",
                  (t) =>
                    match(t.ssh.type)
                      .when(
                        (x) => x.oneofKind === "connect",
                        (c) => (
                          <TextInput
                            label="User"
                            placeholder="root"
                            value={c.connect.user}
                            onChange={(e) => {
                              c.connect.user = e.target.value;
                              updateReq();
                            }}
                          />
                        ),
                      )
                      .otherwise(() => null),
                )
                .otherwise(() => null)}
            </Tabs.Panel>
          </Tabs>
        )}
      </EditItem>

      <div className="flex justify-end">
        <button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
          className={twMerge(
            "flex items-center gap-2 px-5 py-2 rounded-lg text-[0.82rem] font-bold",
            "bg-slate-900 text-white border border-slate-900",
            "hover:bg-slate-800 transition-colors duration-150",
            "disabled:opacity-50 disabled:cursor-not-allowed",
            "shadow-[0_2px_8px_rgba(15,23,42,0.18)]",
          )}
        >
          {mutation.isPending ? (
            <>
              <Loader2 size={14} className="animate-spin" /> Testing…
            </>
          ) : (
            <>
              Test authorization <ArrowRight size={14} />
            </>
          )}
        </button>
      </div>

      <AnimatePresence>
        {resp && mutation.isSuccess && (
          <motion.div
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.22, ease: "easeOut" }}
          >
            <div className="bg-white border border-slate-200 rounded-xl overflow-hidden shadow-[0_1px_4px_rgba(15,23,42,0.06)]">
              <div
                className={twMerge(
                  "flex items-center gap-3 px-5 py-4 border-b",
                  resp.isAuthorized
                    ? "bg-emerald-50 border-emerald-100"
                    : "bg-red-50 border-red-100",
                )}
              >
                {resp.isAuthorized ? (
                  <ShieldCheck
                    size={20}
                    className="text-emerald-600 shrink-0"
                  />
                ) : (
                  <ShieldX size={20} className="text-red-500 shrink-0" />
                )}
                <span
                  className={twMerge(
                    "text-[0.92rem] font-bold",
                    resp.isAuthorized ? "text-emerald-800" : "text-red-700",
                  )}
                >
                  {resp.isAuthorized ? "Authorized" : "Unauthorized"}
                </span>
              </div>

              <div className="px-5 py-4 flex flex-col gap-4">
                <div className="flex items-center justify-center gap-4">
                  <div className="flex items-center justify-center min-w-[120px]">
                    {match(req.downstream)
                      .when(
                        (x) => x.oneofKind === "userRef",
                        (x) => <ResourceListLabel itemRef={x.userRef} />,
                      )
                      .when(
                        (x) => x.oneofKind === "sessionRef",
                        (x) => <ResourceListLabel itemRef={x.sessionRef} />,
                      )
                      .when(
                        (x) => x.oneofKind === "deviceRef",
                        (x) => <ResourceListLabel itemRef={x.deviceRef} />,
                      )
                      .otherwise(() => null)}
                  </div>

                  <AnimatedConnector
                    className="w-[160px]"
                    color={resp.isAuthorized ? "#10b981" : "#ef4444"}
                  />

                  <div className="flex items-center justify-center min-w-[120px]">
                    {match(req.upstream)
                      .when(
                        (x) => x.oneofKind === "serviceRef",
                        (x) => <ResourceListLabel itemRef={x.serviceRef} />,
                      )
                      .when(
                        (x) => x.oneofKind === "namespaceRef",
                        (x) => <ResourceListLabel itemRef={x.namespaceRef} />,
                      )
                      .otherwise(() => null)}
                  </div>
                </div>

                {resp.reason && (
                  <div className="border-t border-slate-100 pt-3 flex flex-col gap-1.5">
                    <span className="text-[0.68rem] font-bold uppercase tracking-[0.08em] text-slate-400">
                      Reason
                    </span>
                    <span className="text-[0.82rem] font-semibold text-slate-700">
                      {getPolicyReason(resp.reason.type)}
                    </span>

                    {match(resp.reason.details?.type)
                      .when(
                        (c) => c?.oneofKind === "policyMatch",
                        (c) => (
                          <div className="mt-1">
                            {c.policyMatch.type.oneofKind === "policy" && (
                              <ResourceListLabel
                                label="Policy"
                                itemRef={c.policyMatch.type.policy.policyRef}
                              />
                            )}
                            {c.policyMatch.type.oneofKind ===
                              "inlinePolicy" && (
                              <ResourceListLabel
                                label="Inline policy"
                                itemRef={
                                  c.policyMatch.type.inlinePolicy.resourceRef
                                }
                              />
                            )}
                          </div>
                        ),
                      )
                      .otherwise(() => null)}
                  </div>
                )}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default PolicyTester;
