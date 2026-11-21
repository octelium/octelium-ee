import * as AccessP from "@/apis/accessv1/accessv1";
import DurationPicker from "@/components/DurationPicker";
import RpcError from "@/components/RpcError";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import { twMerge } from "@/lib/utils";
import { getClientAccess } from "@/utils/client";
import { useListKeys } from "@/utils/forms";
import { invalidateKey } from "@/utils/pb";
import { Alert, Button, Loader, Switch, Textarea } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  BellRing,
  FileClock,
  KeyRound,
  ShieldCheck,
  ShieldX,
  Users,
} from "lucide-react";
import * as React from "react";
import { notifications } from "@mantine/notifications";
import { ReviewersEdit } from "../Policy/edit/Step";
import { SurfacesEdit } from "../Policy/edit/Surface";

const DEFAULT_PRIORITY = 0;
const PRIORITIES = [-4, -3, -2, -1, 0];

const describePriority = (priority: number) => {
  if (priority === 0) {
    return "The granted access behaves like a regular ALLOW rule. DENY rules of the default priority (0) in your core Policies, such as guardrails, still block it.";
  }
  if (priority === -1) {
    return "The granted access takes precedence over every core Policy rule of the default priority (0), including DENY guardrails. Only rules with priority -1 or lower can still block it.";
  }
  return `The granted access takes precedence over every core Policy rule with a priority above ${priority}, including DENY guardrails. Only DENY rules with priority ${priority} or lower can still block it.`;
};

const ConfigSection = (props: {
  icon: React.ReactNode;
  title: string;
  description: string;
  children: React.ReactNode;
}) => (
  <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
    <header className="flex items-start gap-3 border-b border-slate-100 px-4 py-3.5">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
        {props.icon}
      </span>
      <div className="min-w-0">
        <h2 className="text-body font-semibold text-slate-900">{props.title}</h2>
        <p className="mt-0.5 max-w-3xl text-xs font-normal leading-5 text-slate-500">
          {props.description}
        </p>
      </div>
    </header>
    <div className="flex flex-col gap-4 px-4 py-4">{props.children}</div>
  </section>
);

const PriorityLadder = (props: {
  value: number;
  disabled: boolean;
  onChange: (priority: number) => void;
}) => (
  <div className="flex flex-col gap-2">
    <div
      role="radiogroup"
      aria-label="Granted access priority"
      className="grid grid-cols-5 gap-1.5"
    >
      {PRIORITIES.map((priority) => {
        const isGrant = priority === props.value;
        const isAbove = priority < props.value;
        return (
          <button
            key={priority}
            type="button"
            role="radio"
            aria-checked={isGrant}
            disabled={props.disabled}
            onClick={() => !isGrant && props.onChange(priority)}
            className={twMerge(
              "flex cursor-pointer flex-col items-center gap-1 rounded-lg border px-2 py-2 text-center outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 disabled:cursor-not-allowed",
              isGrant
                ? "border-emerald-300 bg-emerald-50"
                : isAbove
                  ? "border-slate-200 bg-slate-50 enabled:hover:border-slate-300"
                  : "border-slate-200 bg-white enabled:hover:border-slate-300",
              props.disabled && !isGrant && "opacity-70",
            )}
          >
            <span className="text-sm font-bold tabular-nums text-slate-800">
              {priority}
              {priority === DEFAULT_PRIORITY && (
                <span className="ml-1 text-micro font-normal text-slate-500">
                  default
                </span>
              )}
            </span>
            <span
              className={twMerge(
                "text-micro font-semibold",
                isGrant
                  ? "text-emerald-700"
                  : isAbove
                    ? "text-slate-500"
                    : "text-slate-400",
              )}
            >
              {isGrant
                ? "Granted access"
                : isAbove
                  ? "Can override"
                  : priority === 0
                    ? "Default rules"
                    : "Overridden"}
            </span>
          </button>
        );
      })}
    </div>
    <div className="flex items-center justify-between text-micro font-normal text-slate-500">
      <span>Higher precedence</span>
      <span>Default precedence</span>
    </div>
  </div>
);

const Edit = (props: {
  item: AccessP.ClusterConfig;
  onUpdate: (item: AccessP.ClusterConfig) => void;
}) => {
  const cloneForEdit = React.useCallback((value: AccessP.ClusterConfig) => {
    const next = AccessP.ClusterConfig.clone(value);
    if (!next.spec) next.spec = AccessP.ClusterConfig_Spec.create();
    if (!next.spec.notification)
      next.spec.notification = AccessP.ClusterConfig_Spec_Notification.create();
    return next;
  }, []);
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;

  React.useEffect(() => {
    setReq(cloneForEdit(props.item));
  }, [cloneForEdit, itemKey]);

  const update = (mutate: (next: AccessP.ClusterConfig) => void) => {
    const next = AccessP.ClusterConfig.clone(req);
    if (!next.spec) next.spec = AccessP.ClusterConfig_Spec.create();
    mutate(next);
    setReq(next);
    const out = AccessP.ClusterConfig.clone(next);
    if (out.spec?.notification && out.spec.notification.defaultSurfaces.length === 0)
      out.spec.notification = undefined;
    props.onUpdate(out);
  };

  const rows = useListKeys();
  const configured = req.spec?.authorization?.priority;
  const isCustom = configured !== undefined;
  const effective = configured ?? DEFAULT_PRIORITY;
  const noMatchTooLong = Array.from(req.spec?.request?.noMatchMessage ?? "").length > 500;

  const updateRequest = (mutate: (value: AccessP.ClusterConfig_Spec_Request) => void) =>
    update((next) => {
      const value =
        next.spec!.request ?? AccessP.ClusterConfig_Spec_Request.create();
      mutate(value);
      next.spec!.request =
        value.defaultDeadline || value.maxStartDelay || value.noMatchMessage
          ? value
          : undefined;
    });


  return (
    <div className="w-full space-y-4 py-2">
      <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
        <header className="flex items-start gap-3 border-b border-slate-100 px-4 py-3.5">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
            <KeyRound size={15} strokeWidth={2.2} />
          </span>
          <div className="min-w-0">
            <h2 className="text-body font-semibold text-slate-900">
              Priority of the granted access
            </h2>
            <p className="mt-0.5 max-w-3xl text-xs font-normal leading-5 text-slate-500">
              Every approved Request is enforced by a hidden PolicyTrigger that
              allows the subject to reach the requested Services until the
              access ends. This sets the priority of that ALLOW rule for the
              full access levels that do not set their own priority. Like any
              core Policy rule, the lower the number, the higher the
              precedence, and DENY wins over ALLOW at the same priority.
              Restricted access levels grant exactly their own Policies.
            </p>
          </div>
        </header>

        <div className="flex flex-col gap-4 px-4 py-4">
          <Switch
            label="Use a custom priority"
            description={`When off, the Cluster default (${DEFAULT_PRIORITY}) is used.`}
            checked={isCustom}
            onChange={(event) =>
              update((next) => {
                if (event.currentTarget.checked) {
                  next.spec!.authorization =
                    AccessP.ClusterConfig_Spec_Authorization.create({
                      priority: DEFAULT_PRIORITY,
                    });
                } else {
                  next.spec!.authorization = undefined;
                }
              })
            }
          />

          <PriorityLadder
            value={effective}
            disabled={!isCustom}
            onChange={(priority) =>
              update((next) => {
                next.spec!.authorization =
                  AccessP.ClusterConfig_Spec_Authorization.create({
                    priority,
                  });
              })
            }
          />

          <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
            <div className="flex items-start gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
              <ShieldCheck
                size={14}
                strokeWidth={2.3}
                className="mt-0.5 shrink-0 text-emerald-600"
              />
              <p className="text-xs leading-5 text-slate-700">
                {describePriority(effective)}
              </p>
            </div>
            <div className="flex items-start gap-2.5 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5">
              <ShieldX
                size={14}
                strokeWidth={2.3}
                className="mt-0.5 shrink-0 text-slate-500"
              />
              <p className="text-xs leading-5 text-slate-700">
                A change applies to the access that is already granted as well:
                the PolicyTriggers of every approved Request are updated right
                away.
              </p>
            </div>
          </div>
        </div>
      </section>

      <ConfigSection
        icon={<FileClock size={15} strokeWidth={2.2} />}
        title="Requests"
        description="Defaults that apply to every new Request."
      >
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <DurationPicker
            title="Default deadline"
            description="A Request that is not decided in time expires. 7 days when unset."
            placeholder="7 days"
            value={req.spec?.request?.defaultDeadline}
            onChange={(value) => updateRequest((next) => (next.defaultDeadline = value))}
          />
          <DurationPicker
            title="Longest start delay"
            description="How far in the future a Request may schedule its access. 30 days when unset."
            placeholder="30 days"
            value={req.spec?.request?.maxStartDelay}
            onChange={(value) => updateRequest((next) => (next.maxStartDelay = value))}
          />
        </div>
        <Textarea
          label="Message when no rule matches"
          description="Shown to requesters before submitting and on Requests that no rule matched, for instance where to ask instead."
          autosize
          minRows={2}
          maxRows={5}
          value={req.spec?.request?.noMatchMessage ?? ""}
          error={noMatchTooLong ? "Use at most 500 characters" : undefined}
          onChange={(event) => {
            const value = event.currentTarget.value;
            updateRequest((next) => (next.noMatchMessage = value));
          }}
        />
      </ConfigSection>

      <ConfigSection
        icon={<Users size={15} strokeWidth={2.2} />}
        title="Fallback reviewers"
        description="When a review step has nobody who can review it and no escalation, these reviewers are used instead of leaving the Request stuck."
      >
        <div className="@container">
          <ReviewersEdit
            reviewers={req.spec?.review?.fallbackReviewers ?? []}
            groupDescription="Members of these Groups review unroutable steps."
            userDescription="These Users review unroutable steps."
            onChange={(reviewers) =>
              update((next) => {
                next.spec!.review = reviewers.length
                  ? AccessP.ClusterConfig_Spec_Review.create({ fallbackReviewers: reviewers })
                  : undefined;
              })
            }
          />
        </div>
      </ConfigSection>

      <ConfigSection
        icon={<BellRing size={15} strokeWidth={2.2} />}
        title="Default notifications"
        description="Announce the outcome of every Request on these Integrations, in addition to the notifications of the rule that decided it."
      >
        <SurfacesEdit
          surfaces={req.spec!.notification!.defaultSurfaces}
          isReview={false}
          keyAt={(i) => rows.keyAt("defaultSurfaces", i)}
          removeKeyAt={(i) => rows.removeAt("defaultSurfaces", i)}
          onUpdate={() => update(() => {})}
          addLabel="Add notification"
          emptyLabel="No default notification."
        />
      </ConfigSection>
    </div>
  );
};

export default () => {
  const query = useQuery({
    queryKey: ["access", "clusterconfig"],
    queryFn: async () => getClientAccess().getClusterConfig({}),
  });

  if (query.isLoading)
    return (
      <div className="flex min-h-64 items-center justify-center">
        <Loader size="sm" color="gray" />
      </div>
    );

  if (query.isError)
    return (
      <div className="mx-auto mt-10 max-w-xl">
        <Alert
          color="red"
          title="Could not load the access cluster configuration"
          icon={<AlertTriangle size={16} />}
        >
          <div className="space-y-3">
            <RpcError error={query.error} />
            <Button
              type="button"
              size="compact-sm"
              variant="outline"
              onClick={() => query.refetch()}
            >
              Retry
            </Button>
          </div>
        </Alert>
      </div>
    );

  if (!query.data?.response)
    return (
      <div className="py-16 text-center text-sm font-semibold text-slate-500">
        The access cluster configuration is unavailable.
      </div>
    );

  return (
    <ResourceEdit
      item={query.data.response}
      specComponent={({ item, onUpdate }) => (
        <Edit
          item={item as AccessP.ClusterConfig}
          onUpdate={(next) => onUpdate(next)}
        />
      )}
      noPostUpdateNavigation
      noPostUpdateToast
      noMetadata
      onUpdateDone={() => {
        invalidateKey(["access", "clusterconfig"]);
        notifications.show({
          color: "green",
          message: "Access ClusterConfig successfully updated",
        });
      }}
    />
  );
};
