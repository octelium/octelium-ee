import * as AccessP from "@/apis/accessv1/accessv1";
import RpcError from "@/components/RpcError";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import { twMerge } from "@/lib/utils";
import { getClientAccess } from "@/utils/client";
import { invalidateKey } from "@/utils/pb";
import { Alert, Button, Loader, Switch } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, KeyRound, ShieldCheck, ShieldX } from "lucide-react";
import * as React from "react";
import { notifications } from "@mantine/notifications";

const DEFAULT_PRIORITY = -1;
const PRIORITIES = [-4, -3, -2, -1, 0];

const describePriority = (priority: number) => {
  if (priority === 0) {
    return "The granted access behaves like a regular ALLOW rule. DENY rules of the default priority (0) in your core Policies still block it.";
  }
  if (priority === -1) {
    return "The granted access takes precedence over every core Policy rule of the default priority (0), including DENY rules. Only rules with priority -1 or lower can still block it.";
  }
  return `The granted access takes precedence over every core Policy rule with a priority above ${priority}. Only DENY rules with priority ${priority} or lower can still block it.`;
};

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
    props.onUpdate(AccessP.ClusterConfig.clone(next));
  };

  const configured = req.spec?.authorization?.priority;
  const isCustom = configured !== undefined;
  const effective = configured ?? DEFAULT_PRIORITY;

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
              allows the subject to reach the requested resource until the
              access ends. This sets the priority of that ALLOW rule against
              the rules of your core Policies. Like any core Policy rule, the
              lower the number, the higher the precedence, and DENY wins over
              ALLOW at the same priority.
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
