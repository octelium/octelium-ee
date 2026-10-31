import { Struct } from "@/apis/google/protobuf/struct";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ResourceListLabel } from "@/components/ResourceList";
import { ResourceInfoMainItem } from "@/pages/utils/types";
import { Shield } from "lucide-react";
import { MutedValue } from "./overview";

export const hasRef = (ref?: ObjectReference): ref is ObjectReference =>
  !!(ref?.name || ref?.uid);

export const RefLabel = (props: { itemRef?: ObjectReference; label?: string }) =>
  hasRef(props.itemRef) ? (
    <ResourceListLabel itemRef={props.itemRef} label={props.label} />
  ) : null;

export const authorizationItems = (
  authorization:
    | { policies: string[]; inlinePolicies: { name: string }[] }
    | undefined,
  group = "Authorization",
): ResourceInfoMainItem[] => {
  const policies = authorization?.policies ?? [];
  const inlinePolicies = authorization?.inlinePolicies ?? [];

  if (policies.length === 0 && inlinePolicies.length === 0) {
    return [
      {
        label: "Policies",
        group,
        span: "full",
        value: <MutedValue>No Policies are attached directly.</MutedValue>,
      },
    ];
  }

  return [
    ...(policies.length > 0
      ? [
          {
            label: "Policies",
            group,
            span: "full" as const,
            value: (
              <div className="flex flex-wrap gap-1">
                {policies.map((policy) => (
                  <ResourceListLabel
                    key={policy}
                    itemRef={ObjectReference.create({
                      apiVersion: "core/v1",
                      kind: "Policy",
                      name: policy,
                    })}
                  />
                ))}
              </div>
            ),
          },
        ]
      : []),
    ...(inlinePolicies.length > 0
      ? [
          {
            label: "Inline policies",
            group,
            span: "full" as const,
            value: (
              <div className="flex flex-wrap gap-1">
                {inlinePolicies.map((policy, index) => (
                  <ResourceListLabel
                    key={`${policy.name}-${index}`}
                    label="Inline policy"
                  >
                    <Shield size={12} strokeWidth={2.5} />
                    {policy.name || `Inline policy ${index + 1}`}
                  </ResourceListLabel>
                ))}
              </div>
            ),
          },
        ]
      : []),
  ];
};

const attrValue = (value: unknown) =>
  typeof value === "string" ? value : JSON.stringify(value);

export const attrsItems = (
  attrs: Struct | undefined,
  group = "Attributes",
): ResourceInfoMainItem[] => {
  if (!attrs) return [];
  const json = Struct.toJson(attrs);
  const entries =
    json && typeof json === "object" && !Array.isArray(json)
      ? Object.entries(json)
      : [];
  if (entries.length === 0) return [];

  return [
    {
      label: "Attributes",
      hint: "Available to Policy conditions",
      group,
      span: "full",
      value: (
        <div className="flex flex-wrap gap-1.5">
          {entries.map(([key, value]) => (
            <span
              key={key}
              className="inline-flex max-w-full items-center gap-1 rounded-md border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-xs leading-none"
            >
              <span className="shrink-0 font-semibold text-slate-600">
                {key}
              </span>
              <span aria-hidden="true" className="h-3 w-px bg-slate-200" />
              <span className="min-w-0 truncate text-slate-700">
                {attrValue(value)}
              </span>
            </span>
          ))}
        </div>
      ),
    },
  ];
};
