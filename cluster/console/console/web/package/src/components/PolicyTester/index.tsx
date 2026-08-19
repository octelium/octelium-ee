import {
  IsAuthorizedRequest,
  IsAuthorizedRequest_Additional,
  IsAuthorizedResponse,
} from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { onError } from "@/utils";
import { getClientPolicyPortal } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { Button, SegmentedControl } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  ArrowRight,
  Braces,
  Check,
  FlaskConical,
  Loader2,
  Network,
  Plus,
  Server,
  ShieldCheck,
  ShieldX,
  Trash2,
  UserRound,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";
import { getPolicyReason } from "../AccessLogViewer/Old";
import SelectInlinePolicies from "../ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "../ResourceLayout/SelectPolicies";
import SelectResource from "../ResourceLayout/SelectResource";
import { ResourceListLabel } from "../ResourceList";
import AnimatedConnector from "./Connector";
import RequestContextEditor, {
  createRequestContext,
} from "./RequestContextEditor";

type DownstreamKind = "userRef" | "sessionRef" | "deviceRef";
type UpstreamKind = "serviceRef" | "namespaceRef";

const downstreamReference = (request: IsAuthorizedRequest) => {
  switch (request.downstream.oneofKind) {
    case "userRef":
      return request.downstream.userRef;
    case "sessionRef":
      return request.downstream.sessionRef;
    case "deviceRef":
      return request.downstream.deviceRef;
    default:
      return undefined;
  }
};

const upstreamReference = (request: IsAuthorizedRequest) => {
  switch (request.upstream.oneofKind) {
    case "serviceRef":
      return request.upstream.serviceRef;
    case "namespaceRef":
      return request.upstream.namespaceRef;
    default:
      return undefined;
  }
};

const OptionalPanel = ({
  title,
  description,
  enabled,
  onEnable,
  onRemove,
  children,
}: {
  title: string;
  description: string;
  enabled: boolean;
  onEnable: () => void;
  onRemove: () => void;
  children: React.ReactNode;
}) => (
  <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
    <div className="flex flex-wrap items-center justify-between gap-3 bg-slate-50/60 px-4 py-3.5 sm:px-5">
      <div className="min-w-0">
        <h3 className="text-[0.8rem] font-bold text-slate-800">{title}</h3>
        <p className="mt-0.5 text-[0.7rem] font-semibold text-slate-400">
          {description}
        </p>
      </div>
      {enabled ? (
        <button
          type="button"
          onClick={onRemove}
          className="flex items-center gap-1.5 rounded-lg px-2.5 py-2 text-[0.68rem] font-bold text-slate-500 transition-colors duration-500 hover:bg-red-50 hover:text-red-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-300"
        >
          <Trash2 size={13} strokeWidth={2.25} />
          Remove
        </button>
      ) : (
        <button
          type="button"
          onClick={onEnable}
          className="flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-2 text-[0.68rem] font-bold text-slate-600 shadow-sm transition-colors duration-500 hover:border-slate-300 hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <Plus size={13} strokeWidth={2.5} />
          Add
        </button>
      )}
    </div>
    <AnimatePresence initial={false}>
      {enabled && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: "auto", opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
          className="overflow-hidden"
        >
          <div className="border-t border-slate-100 px-4 py-4 sm:px-5">
            {children}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  </section>
);

const EndpointCard = ({
  step,
  title,
  description,
  icon: Icon,
  children,
}: {
  step: string;
  title: string;
  description: string;
  icon: React.ComponentType<{ size?: number; strokeWidth?: number }>;
  children: React.ReactNode;
}) => (
  <div className="min-w-0 rounded-xl border border-slate-200 bg-white p-4 shadow-[0_1px_3px_rgba(15,23,42,0.04)] sm:p-5">
    <div className="mb-4 flex items-start gap-3">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white shadow-sm">
        <Icon size={16} strokeWidth={2.25} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-[0.58rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            Step {step}
          </span>
        </div>
        <h3 className="mt-0.5 text-sm font-bold text-slate-800">{title}</h3>
        <p className="mt-0.5 text-[0.68rem] font-semibold text-slate-400">
          {description}
        </p>
      </div>
    </div>
    <div className="space-y-3">{children}</div>
  </div>
);

const PolicyTester = () => {
  const [req, setReq] = React.useState(() =>
    IsAuthorizedRequest.create({
      downstream: { oneofKind: "userRef", userRef: {} as ObjectReference },
      upstream: {
        oneofKind: "serviceRef",
        serviceRef: {} as ObjectReference,
      },
    }),
  );
  const [resp, setResp] = React.useState<IsAuthorizedResponse>();
  const [testedReq, setTestedReq] = React.useState<IsAuthorizedRequest>();

  const mutation = useMutation({
    mutationFn: async (request: IsAuthorizedRequest) =>
      (await getClientPolicyPortal().isAuthorized(request)).response,
    onMutate: () => {
      setResp(undefined);
      setTestedReq(undefined);
    },
    onSuccess: (response, request) => {
      setResp(response);
      setTestedReq(request);
    },
    onError,
  });

  const updateReq = React.useCallback(
    (update: (request: IsAuthorizedRequest) => void) => {
      setReq((current) => {
        const next = IsAuthorizedRequest.clone(current);
        update(next);
        return next;
      });
      setResp(undefined);
      setTestedReq(undefined);
      mutation.reset();
    },
    [mutation],
  );

  const downstreamKind = (req.downstream.oneofKind ||
    "userRef") as DownstreamKind;
  const upstreamKind = (req.upstream.oneofKind ||
    "serviceRef") as UpstreamKind;
  const selectedDownstream = downstreamReference(req);
  const selectedUpstream = upstreamReference(req);
  const canTest = Boolean(
    selectedDownstream?.name && selectedUpstream?.name,
  );

  const setDownstreamKind = (kind: DownstreamKind) => {
    updateReq((request) => {
      request.downstream = match(kind)
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
        .exhaustive();
    });
  };

  const setUpstreamKind = (kind: UpstreamKind) => {
    updateReq((request) => {
      request.upstream = match(kind)
        .with("serviceRef", () => ({
          oneofKind: "serviceRef" as const,
          serviceRef: {} as ObjectReference,
        }))
        .with("namespaceRef", () => ({
          oneofKind: "namespaceRef" as const,
          namespaceRef: {} as ObjectReference,
        }))
        .exhaustive();
    });
  };

  const resultRequest = testedReq ?? req;
  const resultDownstream = downstreamReference(resultRequest);
  const resultUpstream = upstreamReference(resultRequest);

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 pb-8">
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        <div className="flex items-start gap-3 border-b border-slate-100 bg-slate-50/60 px-4 py-4 sm:px-5">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
            <FlaskConical size={18} strokeWidth={2.25} />
          </span>
          <div>
            <h2 className="text-base font-bold tracking-tight text-slate-900">
              Policy authorization tester
            </h2>
            <p className="mt-1 max-w-2xl text-xs font-semibold leading-relaxed text-slate-500">
              Simulate a connection between a downstream identity and an
              upstream resource before applying policy changes.
            </p>
          </div>
        </div>

        <div className="grid items-stretch gap-3 p-4 md:grid-cols-[minmax(0,1fr)_80px_minmax(0,1fr)] md:gap-2 sm:p-5">
          <EndpointCard
            step="1"
            title="Downstream identity"
            description="Choose who is requesting access"
            icon={UserRound}
          >
            <div>
              <div className="mb-2">
                <p className="text-[0.72rem] font-bold text-slate-700">
                  Identity type
                </p>
                <p className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
                  Choose the identity that is making the request
                </p>
              </div>
              <SegmentedControl
                fullWidth
                value={downstreamKind}
                data={[
                  { label: "User", value: "userRef" },
                  { label: "Session", value: "sessionRef" },
                  { label: "Device", value: "deviceRef" },
                ]}
                onChange={(value) =>
                  setDownstreamKind(value as DownstreamKind)
                }
              />
            </div>
            <SelectResource
              key={downstreamKind}
              api="core"
              kind={
                downstreamKind === "userRef"
                  ? "User"
                  : downstreamKind === "sessionRef"
                    ? "Session"
                    : "Device"
              }
              label="Resource"
              required
              clearable
              defaultValue={selectedDownstream?.name}
              onChange={(resource) => {
                updateReq((request) => {
                  const reference = resource
                    ? getResourceRef(resource)
                    : ({} as ObjectReference);
                  request.downstream =
                    downstreamKind === "userRef"
                      ? { oneofKind: "userRef", userRef: reference }
                      : downstreamKind === "sessionRef"
                        ? { oneofKind: "sessionRef", sessionRef: reference }
                        : { oneofKind: "deviceRef", deviceRef: reference };
                });
              }}
            />
          </EndpointCard>

          <div className="hidden items-center px-2 md:flex">
            <AnimatedConnector />
          </div>
          <div className="flex h-10 justify-center md:hidden">
            <AnimatedConnector orientation="vertical" className="h-full" />
          </div>

          <EndpointCard
            step="2"
            title="Upstream resource"
            description="Choose what is being accessed"
            icon={Server}
          >
            <div>
              <div className="mb-2">
                <p className="text-[0.72rem] font-bold text-slate-700">
                  Resource type
                </p>
                <p className="mt-0.5 text-[0.66rem] font-semibold text-slate-400">
                  Choose the resource being accessed
                </p>
              </div>
              <SegmentedControl
                fullWidth
                value={upstreamKind}
                data={[
                  { label: "Service", value: "serviceRef" },
                  { label: "Namespace", value: "namespaceRef" },
                ]}
                onChange={(value) => setUpstreamKind(value as UpstreamKind)}
              />
            </div>
            <SelectResource
              key={upstreamKind}
              api="core"
              kind={upstreamKind === "serviceRef" ? "Service" : "Namespace"}
              label="Resource"
              required
              clearable
              defaultValue={selectedUpstream?.name}
              onChange={(resource) => {
                updateReq((request) => {
                  const reference = resource
                    ? getResourceRef(resource)
                    : ({} as ObjectReference);
                  request.upstream =
                    upstreamKind === "serviceRef"
                      ? { oneofKind: "serviceRef", serviceRef: reference }
                      : { oneofKind: "namespaceRef", namespaceRef: reference };
                });
              }}
            />
          </EndpointCard>
        </div>
      </section>

      <OptionalPanel
        title="Additional policies"
        description="Evaluate policies without attaching them to a resource"
        enabled={Boolean(req.additional)}
        onEnable={() =>
          updateReq((request) => {
            request.additional = IsAuthorizedRequest_Additional.create({});
          })
        }
        onRemove={() =>
          updateReq((request) => {
            request.additional = undefined;
          })
        }
      >
        {req.additional && (
          <div className="space-y-4">
            <SelectPolicies
              policies={req.additional.policies}
              onUpdate={(policies) =>
                updateReq((request) => {
                  if (request.additional) {
                    request.additional.policies = policies ?? [];
                  }
                })
              }
            />
            <SelectInlinePolicies
              inlinePolicies={req.additional.inlinePolicies}
              onUpdate={(inlinePolicies) =>
                updateReq((request) => {
                  if (request.additional) {
                    request.additional.inlinePolicies = inlinePolicies;
                  }
                })
              }
            />
          </div>
        )}
      </OptionalPanel>

      <OptionalPanel
        title="Request context"
        description="Add protocol-specific information to the authorization request"
        enabled={Boolean(req.request)}
        onEnable={() =>
          updateReq((request) => {
            request.request = createRequestContext("http");
          })
        }
        onRemove={() =>
          updateReq((request) => {
            request.request = undefined;
          })
        }
      >
        {req.request && (
          <RequestContextEditor
            value={req.request}
            onChange={(update) =>
              updateReq((request) => {
                if (request.request) update(request.request);
              })
            }
          />
        )}
      </OptionalPanel>

      <section className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
        <div className="flex items-center gap-2.5">
          <Network size={15} className="shrink-0 text-slate-500" />
          <div>
            <p className="text-[0.72rem] font-bold text-slate-700">
              Ready to evaluate
            </p>
            <p className="mt-0.5 text-[0.65rem] font-semibold text-slate-400">
              {canTest
                ? "Both connection endpoints are selected."
                : "Select both a downstream identity and upstream resource."}
            </p>
          </div>
        </div>
        <Button
          onClick={() => mutation.mutate(IsAuthorizedRequest.clone(req))}
          disabled={mutation.isPending || !canTest}
          loading={mutation.isPending}
          loaderProps={{ children: <Loader2 size={14} /> }}
          rightSection={!mutation.isPending && <ArrowRight size={14} />}
          color="dark"
          radius="md"
          size="sm"
          styles={{
            root: {
              fontSize: "0.78rem",
              fontWeight: 700,
              boxShadow: "0 3px 10px rgba(15,23,42,0.16)",
              transition:
                "background-color 500ms, box-shadow 500ms, opacity 500ms",
            },
          }}
        >
          {mutation.isPending ? "Evaluating…" : "Test authorization"}
        </Button>
      </section>

      {mutation.isError && (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-xs font-semibold text-red-700">
          The authorization test could not be completed. Review the request and
          try again.
        </div>
      )}

      <AnimatePresence mode="wait">
        {resp && testedReq && mutation.isSuccess && (
          <motion.section
            key={resp.isAuthorized ? "authorized" : "unauthorized"}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
            className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-[0_4px_18px_rgba(15,23,42,0.06)]"
          >
            <div
              className={twMerge(
                "flex items-center justify-between gap-3 border-b px-4 py-4 sm:px-5",
                resp.isAuthorized
                  ? "border-emerald-100 bg-emerald-50/80"
                  : "border-red-100 bg-red-50/80",
              )}
            >
              <div className="flex items-center gap-3">
                <span
                  className={twMerge(
                    "flex h-9 w-9 items-center justify-center rounded-lg text-white shadow-sm",
                    resp.isAuthorized ? "bg-emerald-600" : "bg-red-600",
                  )}
                >
                  {resp.isAuthorized ? (
                    <ShieldCheck size={18} />
                  ) : (
                    <ShieldX size={18} />
                  )}
                </span>
                <div>
                  <p
                    className={twMerge(
                      "text-sm font-bold",
                      resp.isAuthorized ? "text-emerald-900" : "text-red-900",
                    )}
                  >
                    {resp.isAuthorized
                      ? "Access authorized"
                      : "Access denied"}
                  </p>
                  <p
                    className={twMerge(
                      "mt-0.5 text-[0.65rem] font-semibold",
                      resp.isAuthorized ? "text-emerald-700" : "text-red-700",
                    )}
                  >
                    Authorization evaluation completed
                  </p>
                </div>
              </div>
              <Check
                size={16}
                className={
                  resp.isAuthorized ? "text-emerald-600" : "text-red-500"
                }
              />
            </div>

            <div className="p-4 sm:p-5">
              <div className="grid grid-cols-[minmax(0,1fr)_70px_minmax(0,1fr)] items-center gap-2 sm:grid-cols-[minmax(0,1fr)_160px_minmax(0,1fr)] sm:gap-4">
                <div className="min-w-0 rounded-lg border border-slate-200 bg-slate-50/60 p-2.5">
                  <span className="mb-2 block text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                    Downstream
                  </span>
                  {resultDownstream && (
                    <ResourceListLabel itemRef={resultDownstream} />
                  )}
                </div>
                <AnimatedConnector
                  color={resp.isAuthorized ? "#059669" : "#dc2626"}
                />
                <div className="min-w-0 rounded-lg border border-slate-200 bg-slate-50/60 p-2.5">
                  <span className="mb-2 block text-[0.58rem] font-bold uppercase tracking-[0.07em] text-slate-400">
                    Upstream
                  </span>
                  {resultUpstream && (
                    <ResourceListLabel itemRef={resultUpstream} />
                  )}
                </div>
              </div>

              {resp.reason && (
                <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
                  <div className="flex items-center gap-2">
                    <Braces size={14} className="text-slate-500" />
                    <span className="text-[0.62rem] font-bold uppercase tracking-[0.08em] text-slate-500">
                      Decision reason
                    </span>
                  </div>
                  <p className="mt-2 text-[0.8rem] font-bold text-slate-700">
                    {getPolicyReason(resp.reason.type)}
                  </p>

                  {match(resp.reason.details?.type)
                    .when(
                      (details) => details?.oneofKind === "policyMatch",
                      (details) => (
                        <div className="mt-3 border-t border-slate-200 pt-3">
                          {details.policyMatch.type.oneofKind === "policy" && (
                            <ResourceListLabel
                              label="Policy"
                              itemRef={
                                details.policyMatch.type.policy.policyRef
                              }
                            />
                          )}
                          {details.policyMatch.type.oneofKind ===
                            "inlinePolicy" && (
                            <ResourceListLabel
                              label="Inline policy"
                              itemRef={
                                details.policyMatch.type.inlinePolicy.resourceRef
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
          </motion.section>
        )}
      </AnimatePresence>
    </div>
  );
};

export default PolicyTester;
