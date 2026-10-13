import * as CoreP from "@/apis/corev1/corev1";
import { getResourceComponentInfoFromResource } from "@/pages/utils/resourceRegistry";
import { useUpdateResource } from "@/pages/utils/resource";
import { ResourceStatusInfo, ResourceStatusTone } from "@/pages/utils/types";
import { cloneResource, Resource } from "@/utils/pb";
import { Select, Switch } from "@mantine/core";
import StatusPill from "@/components/StatusBadge";
import EditItemWrap from "./EditItemWrap";

export { StatusPill };

const STATE_KINDS = new Set([
  "core/Session",
  "core/Device",
  "core/Authenticator",
]);

const STATE_TONES: Record<number, ResourceStatusTone> = {
  [CoreP.Session_Spec_State.ACTIVE]: "success",
  [CoreP.Session_Spec_State.PENDING]: "warning",
  [CoreP.Session_Spec_State.REJECTED]: "danger",
};

const STATE_LABELS: Record<number, string> = {
  [CoreP.Session_Spec_State.ACTIVE]: "Active",
  [CoreP.Session_Spec_State.PENDING]: "Pending",
  [CoreP.Session_Spec_State.REJECTED]: "Rejected",
};

const STATE_OPTIONS = [
  { label: "Active", value: `${CoreP.Session_Spec_State.ACTIVE}` },
  { label: "Pending", value: `${CoreP.Session_Spec_State.PENDING}` },
  { label: "Rejected", value: `${CoreP.Session_Spec_State.REJECTED}` },
];

type Spec = { isDisabled?: boolean; state?: number } | undefined;

const getSpec = (item: Resource): Spec => (item as { spec?: Spec }).spec;

const usesState = (item: Resource) =>
  STATE_KINDS.has(`${item.apiVersion.split("/")[0]}/${item.kind}`) &&
  typeof getSpec(item)?.state === "number";

const usesEnabled = (item: Resource) =>
  typeof getSpec(item)?.isDisabled === "boolean";

export const deriveResourceStatus = (
  item: Resource,
): ResourceStatusInfo | undefined => {
  const spec = getSpec(item);
  if (!spec) return undefined;

  if (usesState(item)) {
    const state = spec.state as number;
    const label = STATE_LABELS[state];
    if (!label) return undefined;
    return { label, tone: STATE_TONES[state] ?? "neutral" };
  }

  if (usesEnabled(item)) {
    return spec.isDisabled
      ? { label: "Disabled", tone: "danger" }
      : { label: "Active", tone: "success" };
  }

  return undefined;
};

const ResourceStatus = (props: {
  item: Resource;
  status?: ResourceStatusInfo;
}) => {
  const { item } = props;
  const mutationUpdate = useUpdateResource();
  const status = props.status ?? deriveResourceStatus(item);

  if (!status) return null;

  const info = getResourceComponentInfoFromResource(item);
  const readOnly =
    !!item.metadata?.isSystem || !!info?.unEditable || !!info?.readOnlyEdit;

  const pill = <StatusPill label={status.label} tone={status.tone} />;

  let control = status.control;

  if (!control && !readOnly && !props.status) {
    if (usesState(item)) {
      control = (
        <Select
          size="xs"
          allowDeselect={false}
          data={STATE_OPTIONS}
          value={`${getSpec(item)!.state}`}
          onChange={(value) => {
            if (!value) return;
            const next = cloneResource(item) as Resource & { spec: Spec };
            next.spec!.state = Number(value);
            mutationUpdate.mutate(next);
          }}
        />
      );
    } else if (usesEnabled(item)) {
      control = (
        <Switch
          size="sm"
          aria-label="Enabled"
          checked={!getSpec(item)!.isDisabled}
          onChange={(event) => {
            const next = cloneResource(item) as Resource & { spec: Spec };
            next.spec!.isDisabled = !event.currentTarget.checked;
            mutationUpdate.mutate(next);
          }}
        />
      );
    }
  }

  const body = control ? (
    <EditItemWrap
      mutation={mutationUpdate}
      label="status"
      showComponent={pill}
      editComponent={control}
    />
  ) : (
    pill
  );

  if (!status.hint) return body;

  return (
    <span className="flex flex-col items-start gap-0.5">
      {body}
      <span className="text-xs font-normal text-slate-500">{status.hint}</span>
    </span>
  );
};

export default ResourceStatus;
