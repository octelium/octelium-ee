import * as CoreP from "@/apis/corev1/corev1";
import {
  IsAuthorizedRequest,
  IsAuthorizedRequest_Additional,
} from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { getPolicyReason } from "@/components/PolicyTester/policyReason";
import RequestContextEditor, {
  createRequestContext,
} from "@/components/PolicyTester/RequestContextEditor";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { ResourceListLabel } from "@/components/ResourceList";
import { twMerge } from "@/lib/utils";
import { onError } from "@/utils";
import { getClientPolicyPortal } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { Button, Collapse, SegmentedControl } from "@mantine/core";
import { useMutation } from "@tanstack/react-query";
import { ChevronDown, Play, ShieldCheck, ShieldX } from "lucide-react";
import * as React from "react";

type Downstream = "userRef" | "sessionRef" | "deviceRef";
type Upstream = "serviceRef" | "namespaceRef";

const DOWNSTREAM_KIND: Record<Downstream, string> = {
  userRef: "User",
  sessionRef: "Session",
  deviceRef: "Device",
};

const UPSTREAM_KIND: Record<Upstream, string> = {
  serviceRef: "Service",
  namespaceRef: "Namespace",
};

const TryPolicy = (props: { policy: CoreP.Policy; blocked?: string }) => {
  const [downstream, setDownstream] = React.useState<Downstream>("userRef");
  const [upstream, setUpstream] = React.useState<Upstream>("serviceRef");
  const [who, setWho] = React.useState<ObjectReference>();
  const [what, setWhat] = React.useState<ObjectReference>();
  const [details, setDetails] = React.useState(false);
  const [request, setRequest] = React.useState(() =>
    createRequestContext("http"),
  );
  const draftName = props.policy.metadata?.name || "draft";

  const mutation = useMutation({
    mutationFn: async () => {
      const req = IsAuthorizedRequest.create({
        downstream:
          downstream === "userRef"
            ? { oneofKind: "userRef", userRef: who! }
            : downstream === "sessionRef"
              ? { oneofKind: "sessionRef", sessionRef: who! }
              : { oneofKind: "deviceRef", deviceRef: who! },
        upstream:
          upstream === "serviceRef"
            ? { oneofKind: "serviceRef", serviceRef: what! }
            : { oneofKind: "namespaceRef", namespaceRef: what! },
        request: details
          ? CoreP.RequestContext_Request.clone(request)
          : undefined,
        additional: IsAuthorizedRequest_Additional.create({
          inlinePolicies: [
            CoreP.InlinePolicy.create({
              name: draftName,
              spec: CoreP.Policy_Spec.clone(
                props.policy.spec ?? CoreP.Policy_Spec.create(),
              ),
            }),
          ],
        }),
      });
      return (await getClientPolicyPortal().isAuthorized(req)).response;
    },
    onError,
  });

  const reset = () => mutation.reset();
  const resp = mutation.data;
  const match =
    resp?.reason?.details?.type.oneofKind === "policyMatch"
      ? resp.reason.details.type.policyMatch
      : undefined;
  const byDraft =
    match?.type.oneofKind === "inlinePolicy" &&
    match.type.inlinePolicy.name === draftName;

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-3 @2xl:grid-cols-2">
        <div className="flex min-w-0 flex-col gap-2">
          <SegmentedControl
            size="xs"
            value={downstream}
            data={Object.entries(DOWNSTREAM_KIND).map(([value, label]) => ({
              value,
              label,
            }))}
            onChange={(value) => {
              setDownstream(value as Downstream);
              setWho(undefined);
              reset();
            }}
          />
          <SelectResource
            key={downstream}
            api="core"
            kind={DOWNSTREAM_KIND[downstream]}
            label="Who"
            clearable
            defaultValue={who?.name}
            onChange={(item) => {
              setWho(item ? getResourceRef(item) : undefined);
              reset();
            }}
          />
        </div>
        <div className="flex min-w-0 flex-col gap-2">
          <SegmentedControl
            size="xs"
            value={upstream}
            data={Object.entries(UPSTREAM_KIND).map(([value, label]) => ({
              value,
              label,
            }))}
            onChange={(value) => {
              setUpstream(value as Upstream);
              setWhat(undefined);
              reset();
            }}
          />
          <SelectResource
            key={upstream}
            api="core"
            kind={UPSTREAM_KIND[upstream]}
            label="Accessing"
            clearable
            defaultValue={what?.name}
            onChange={(item) => {
              setWhat(item ? getResourceRef(item) : undefined);
              reset();
            }}
          />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <button
          type="button"
          aria-expanded={details}
          onClick={() => {
            setDetails((value) => !value);
            reset();
          }}
          className="flex w-fit cursor-pointer items-center gap-1 text-xs font-semibold text-slate-600 outline-none hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <ChevronDown
            size={13}
            strokeWidth={2.5}
            className={twMerge(
              "transition-transform duration-150",
              !details && "-rotate-90",
            )}
          />
          Request details (method, path, query…)
        </button>
        <Collapse expanded={details}>
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-3">
            <RequestContextEditor
              value={request}
              onChange={(update) => {
                setRequest((current) => {
                  const next = CoreP.RequestContext_Request.clone(current);
                  update(next);
                  return next;
                });
                reset();
              }}
            />
          </div>
        </Collapse>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button
          color="ink"
          leftSection={<Play size={13} />}
          loading={mutation.isPending}
          disabled={!who?.name || !what?.name || !!props.blocked}
          onClick={() => mutation.mutate()}
        >
          Evaluate the draft
        </Button>
        <span className="text-micro leading-4 text-slate-500">
          {props.blocked ??
            "The draft is added to every Policy that already applies, including a saved version of this one."}
        </span>
      </div>

      {resp && (
        <div
          role="status"
          className={twMerge(
            "flex items-start gap-3 rounded-xl border px-4 py-3",
            resp.isAuthorized
              ? "border-emerald-200 bg-emerald-50/70"
              : "border-red-200 bg-red-50/70",
          )}
        >
          <span
            className={twMerge(
              "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg",
              resp.isAuthorized
                ? "bg-emerald-600 text-white"
                : "bg-red-600 text-white",
            )}
          >
            {resp.isAuthorized ? (
              <ShieldCheck size={17} />
            ) : (
              <ShieldX size={17} />
            )}
          </span>
          <div className="flex min-w-0 flex-col gap-1">
            <span
              className={twMerge(
                "text-body font-bold",
                resp.isAuthorized ? "text-emerald-800" : "text-red-800",
              )}
            >
              {resp.isAuthorized ? "Allowed" : "Denied"}
              <span className="ml-2 text-xs font-medium opacity-80">
                {getPolicyReason(resp.reason?.type)}
              </span>
            </span>
            {match && (
              <span className="flex flex-wrap items-center gap-1.5 text-xs text-slate-700">
                {byDraft ? (
                  <span className="font-semibold">Decided by this draft</span>
                ) : match.type.oneofKind === "policy" ? (
                  <ResourceListLabel
                    label="Decided by"
                    itemRef={match.type.policy.policyRef}
                  />
                ) : match.type.oneofKind === "inlinePolicy" ? (
                  <span>
                    Decided by the inline Policy {match.type.inlinePolicy.name}
                  </span>
                ) : null}
                {match.ruleName && (
                  <span>
                    · rule <b>{match.ruleName}</b>
                  </span>
                )}
                <span className="text-slate-500">
                  · priority {match.priority}
                </span>
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default TryPolicy;
