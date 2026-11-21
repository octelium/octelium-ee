import * as AccessP from "@/apis/accessv1/accessv1";
import { Session_Status_Authentication_Info_AAL } from "@/apis/corev1/corev1";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import SelectInlinePolicies from "@/components/ResourceLayout/SelectInlinePolicies";
import SelectPolicies from "@/components/ResourceLayout/SelectPolicies";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { useListKeys } from "@/utils/forms";
import {
  ActionIcon,
  Button,
  NumberInput,
  SegmentedControl,
  Switch,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { ArrowDown, ArrowUp, Library, PanelTop, Plus, Trash2 } from "lucide-react";
import * as React from "react";

const LEVEL_NAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

const newAccessLevel = (
  name: string,
  restricted?: boolean,
): AccessP.Catalog_Spec_AccessLevel =>
  AccessP.Catalog_Spec_AccessLevel.create({
    name,
    displayName: restricted ? "Restricted access" : "Full access",
    type: restricted
      ? { oneofKind: "restricted", restricted: AccessP.Catalog_Spec_AccessLevel_Restricted.create() }
      : { oneofKind: "full", full: AccessP.Catalog_Spec_AccessLevel_Full.create() },
  });

const nextLevelName = (levels: AccessP.Catalog_Spec_AccessLevel[], base: string) => {
  const names = new Set(levels.map((level) => level.name));
  if (!names.has(base)) return base;
  for (let i = 2; ; i++) if (!names.has(`${base}-${i}`)) return `${base}-${i}`;
};

const accessLevelIssues = (
  level: AccessP.Catalog_Spec_AccessLevel,
  levels: AccessP.Catalog_Spec_AccessLevel[],
): string[] => {
  const issues: string[] = [];
  if (!LEVEL_NAME_PATTERN.test(level.name))
    issues.push("Use lowercase letters, digits and dashes in the name");
  else if (levels.filter((other) => other.name === level.name).length > 1)
    issues.push("Access level names must be unique");
  if (!level.type.oneofKind) issues.push("Choose what the access level grants");
  if (
    level.type.oneofKind === "restricted" &&
    level.type.restricted.policies.length === 0 &&
    level.type.restricted.inlinePolicies.length === 0
  )
    issues.push("Attach at least one Policy to a restricted access level");
  const priority = level.type.oneofKind === "full" ? (level.type.full.priority ?? 0) : 0;
  if (priority < -4 || priority > 0) issues.push("The priority must be between -4 and 0");
  return issues;
};

const AccessLevelEdit = (props: {
  level: AccessP.Catalog_Spec_AccessLevel;
  index: number;
  levels: AccessP.Catalog_Spec_AccessLevel[];
  onUpdate: () => void;
  onMove: (delta: number) => void;
  onRemove: () => void;
}) => {
  const { level, index, levels, onUpdate } = props;
  const issues = accessLevelIssues(level, levels);
  const full = level.type.oneofKind === "full" ? level.type.full : undefined;
  const restricted = level.type.oneofKind === "restricted" ? level.type.restricted : undefined;

  return (
    <li className="flex flex-col gap-3 rounded-xl border border-slate-200 bg-white p-3.5 shadow-card">
      <div className="flex flex-wrap items-end gap-3">
        <TextInput
          className="min-w-[160px] flex-1"
          label="Name"
          description={index === 0 ? "Requested when the requester does not choose one." : "Used in requests and Policy conditions."}
          placeholder="read-only"
          value={level.name}
          onChange={(event) => {
            level.name = event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "-");
            onUpdate();
          }}
        />
        <TextInput
          className="min-w-[160px] flex-1"
          label="Display name"
          description="Shown to requesters and reviewers."
          placeholder="Read only"
          value={level.displayName}
          onChange={(event) => {
            level.displayName = event.target.value;
            onUpdate();
          }}
        />
        <div className="flex items-center gap-0.5 pb-1">
          <Tooltip label="Move up" withArrow>
            <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move access level up" disabled={index === 0} onClick={() => props.onMove(-1)}>
              <ArrowUp size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Move down" withArrow>
            <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Move access level down" disabled={index === levels.length - 1} onClick={() => props.onMove(1)}>
              <ArrowDown size={13} />
            </ActionIcon>
          </Tooltip>
          <Tooltip label="Remove access level" withArrow>
            <ActionIcon variant="subtle" color="red" size="sm" aria-label="Remove access level" onClick={props.onRemove}>
              <Trash2 size={13} />
            </ActionIcon>
          </Tooltip>
        </div>
      </div>

      <Textarea
        label="Description"
        description="Explain what this access level allows."
        autosize
        minRows={1}
        maxRows={4}
        value={level.description}
        onChange={(event) => {
          level.description = event.target.value;
          onUpdate();
        }}
      />

      <SegmentedControl
        size="xs"
        aria-label="Access level type"
        value={restricted ? "restricted" : "full"}
        data={[
          { label: "Full access", value: "full" },
          { label: "Restricted by Policies", value: "restricted" },
        ]}
        onChange={(value) => {
          level.type =
            value === "restricted"
              ? { oneofKind: "restricted", restricted: AccessP.Catalog_Spec_AccessLevel_Restricted.create() }
              : { oneofKind: "full", full: AccessP.Catalog_Spec_AccessLevel_Full.create() };
          onUpdate();
        }}
      />

      {full && (
        <div className="flex flex-col gap-1.5">
          <NumberInput
            className="max-w-[220px]"
            label="Priority"
            description="0 lets deny guardrails win. A negative priority overrides them."
            min={-4}
            max={0}
            value={full.priority ?? 0}
            onChange={(value) => {
              const next = Math.trunc(Number(value) || 0);
              full.priority = next === 0 ? undefined : next;
              onUpdate();
            }}
          />
          {(full.priority ?? 0) < 0 && (
            <span className="text-micro font-semibold leading-4 text-amber-700">
              This access level takes precedence over Policies that deny the
              access, such as break-glass access.
            </span>
          )}
        </div>
      )}

      {restricted && (
        <div className="flex flex-col gap-3 rounded-lg border border-dashed border-slate-200 p-3">
          <span className="text-micro leading-4 text-slate-500">
            The access allows exactly what these Policies allow, for instance
            only read-only HTTP methods or a single database role.
          </span>
          <SelectPolicies
            policies={restricted.policies}
            onUpdate={(value) => {
              restricted.policies = value ?? [];
              onUpdate();
            }}
          />
          <SelectInlinePolicies
            inlinePolicies={restricted.inlinePolicies}
            onUpdate={(value) => {
              restricted.inlinePolicies = value;
              onUpdate();
            }}
          />
        </div>
      )}

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <DurationPicker
          title="Default duration"
          description="Used when the requester does not choose one."
          placeholder="1 hour"
          value={level.defaultDuration}
          onChange={(value) => {
            level.defaultDuration = value;
            onUpdate();
          }}
        />
        <DurationPicker
          title="Longest access"
          description="Rules can only lower it."
          placeholder="24 hours"
          value={level.maxDuration}
          onChange={(value) => {
            level.maxDuration = value;
            onUpdate();
          }}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-xs font-semibold text-slate-700">Authentication assurance</span>
        <SegmentedControl
          size="xs"
          aria-label="Authentication assurance"
          value={String(level.verification?.minAAL ?? 0)}
          data={[
            { label: "Any", value: "0" },
            { label: "AAL1", value: String(Session_Status_Authentication_Info_AAL.AAL1) },
            { label: "AAL2", value: String(Session_Status_Authentication_Info_AAL.AAL2) },
            { label: "AAL3", value: String(Session_Status_Authentication_Info_AAL.AAL3) },
          ]}
          onChange={(value) => {
            const aal = Number(value) as Session_Status_Authentication_Info_AAL;
            level.verification = aal
              ? AccessP.Catalog_Spec_AccessLevel_Verification.create({ minAAL: aal })
              : undefined;
            onUpdate();
          }}
        />
        <span className="text-micro leading-4 text-slate-500">
          The granted access only applies to Sessions that authenticated at
          least at this level.
        </span>
      </div>

      {issues.length > 0 && (
        <ul className="flex flex-col gap-0.5 rounded-lg bg-amber-50 px-3 py-2 text-micro font-medium text-amber-800">
          {issues.map((issue) => (
            <li key={issue}>{issue}</li>
          ))}
        </ul>
      )}
    </li>
  );
};

const Principals = (props: {
  label: string;
  users: string[];
  groups: string[];
  description: string;
  onChange: (users: string[], groups: string[]) => void;
}) => (
  <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
    <SelectResourceMultiple
      api="core"
      kind="Group"
      label={`${props.label} Groups`}
      description={props.description}
      clearable
      defaultValue={props.groups}
      onChange={(resources) =>
        props.onChange(props.users, resources?.map((resource) => resource.metadata!.name) ?? [])
      }
    />
    <SelectResourceMultiple
      api="core"
      kind="User"
      label={`${props.label} Users`}
      description={props.description}
      clearable
      defaultValue={props.users}
      onChange={(resources) =>
        props.onChange(resources?.map((resource) => resource.metadata!.name) ?? [], props.groups)
      }
    />
  </div>
);

const Edit = (props: {
  item: AccessP.Catalog;
  onUpdate: (item: AccessP.Catalog) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(AccessP.Catalog.clone(item));
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(AccessP.Catalog.clone(item));
  }, [itemKey]);

  const rows = useListKeys();

  const updateReq = () => {
    const next = AccessP.Catalog.clone(req);
    setReq(next);
    onUpdate(AccessP.Catalog.clone(next));
  };

  const levels = req.spec?.accessLevels ?? [];

  const audience = () => {
    if (!req.spec!.audience) req.spec!.audience = AccessP.Catalog_Spec_Audience.create();
    return req.spec!.audience;
  };

  const pruneAudience = () => {
    const value = req.spec?.audience;
    if (
      value &&
      value.users.length === 0 &&
      value.groups.length === 0 &&
      !value.match &&
      !value.onBehalf?.users.length &&
      !value.onBehalf?.groups.length
    )
      req.spec!.audience = undefined;
  };

  const requirements = () => {
    if (!req.spec!.requirements)
      req.spec!.requirements = AccessP.Catalog_Spec_Requirements.create();
    return req.spec!.requirements;
  };

  const ensureCollection = () => {
    const spec = req.spec!;
    if (!spec.resourceCollection) {
      spec.resourceCollection =
        AccessP.Catalog_Spec_ResourceCollection.create();
    }
    if (!spec.resourceCollection.service) {
      spec.resourceCollection.service =
        AccessP.Catalog_Spec_ResourceCollection_Service.create();
    }
    return spec.resourceCollection.service;
  };

  const collection = req.spec?.resourceCollection?.service;

  return (
    <div className="flex w-full flex-col gap-4">
      <EditItem
        title="Access levels"
        description="What an approved request grants. Requesters choose one of these levels; the first one is the default."
      >
        <div className="flex flex-col gap-3">
          {levels.length === 0 && (
            <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800">
              This Catalog has no access level, so it cannot be requested.
            </p>
          )}
          <ol className="flex flex-col gap-3">
            {levels.map((level, index) => (
              <AccessLevelEdit
                key={rows.keyAt("levels", index)}
                level={level}
                index={index}
                levels={levels}
                onUpdate={updateReq}
                onMove={(delta) => {
                  const target = index + delta;
                  if (target < 0 || target >= levels.length) return;
                  rows.move("levels", index, target);
                  const [moved] = levels.splice(index, 1);
                  levels.splice(target, 0, moved);
                  updateReq();
                }}
                onRemove={() => {
                  rows.removeAt("levels", index);
                  levels.splice(index, 1);
                  updateReq();
                }}
              />
            ))}
          </ol>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="default"
              size="xs"
              leftSection={<Plus size={13} />}
              disabled={levels.length >= 16}
              onClick={() => {
                req.spec!.accessLevels = [...levels, newAccessLevel(nextLevelName(levels, "full"))];
                updateReq();
              }}
            >
              Add full access
            </Button>
            <Button
              variant="default"
              size="xs"
              leftSection={<Plus size={13} />}
              disabled={levels.length >= 16}
              onClick={() => {
                req.spec!.accessLevels = [
                  ...levels,
                  newAccessLevel(nextLevelName(levels, "restricted"), true),
                ];
                updateReq();
              }}
            >
              Add restricted access
            </Button>
          </div>
        </div>
      </EditItem>

      <EditItem
        title="Who can request"
        description="Leave empty to let every User request this Catalog for themselves."
      >
        <div className="flex flex-col gap-4">
          <Principals
            label="Requester"
            description="Only these can request this Catalog."
            users={req.spec?.audience?.users ?? []}
            groups={req.spec?.audience?.groups ?? []}
            onChange={(users, groups) => {
              audience().users = users;
              audience().groups = groups;
              pruneAudience();
              updateReq();
            }}
          />
          <TextInput
            label="Requester condition"
            description="Optional CEL expression over ctx.user. Matching Users can request too."
            placeholder={'"sre" in ctx.user.spec.groups'}
            classNames={{ input: "font-mono" }}
            value={req.spec?.audience?.match ?? ""}
            onChange={(event) => {
              audience().match = event.target.value;
              pruneAudience();
              updateReq();
            }}
          />
          <Principals
            label="On behalf"
            description="These can request this Catalog for other Users."
            users={req.spec?.audience?.onBehalf?.users ?? []}
            groups={req.spec?.audience?.onBehalf?.groups ?? []}
            onChange={(users, groups) => {
              audience().onBehalf = AccessP.Catalog_Spec_Audience_OnBehalf.create({ users, groups });
              pruneAudience();
              updateReq();
            }}
          />
        </div>
      </EditItem>

      <EditItem
        title="Owners"
        description="The people responsible for this Catalog. Review steps can route requests to them."
      >
        <Principals
          label="Owner"
          description="Owners of this Catalog."
          users={req.spec?.owners?.users ?? []}
          groups={req.spec?.owners?.groups ?? []}
          onChange={(users, groups) => {
            req.spec!.owners =
              users.length || groups.length
                ? AccessP.Catalog_Spec_Owners.create({ users, groups })
                : undefined;
            updateReq();
          }}
        />
      </EditItem>

      <EditItem
        title="Requirements"
        description="What every request for this Catalog must include."
      >
        <div className="flex flex-col gap-4">
          <div className="flex flex-wrap items-end gap-3">
            <Switch
              size="xs"
              label="Require a justification"
              checked={!!req.spec?.requirements?.justification?.isRequired}
              onChange={(event) => {
                requirements().justification = event.target.checked
                  ? AccessP.Catalog_Spec_Requirements_Justification.create({ isRequired: true })
                  : undefined;
                updateReq();
              }}
            />
            {req.spec?.requirements?.justification?.isRequired && (
              <NumberInput
                size="xs"
                className="w-48"
                label="Minimum length"
                min={0}
                max={1500}
                value={req.spec.requirements.justification.minLength}
                onChange={(value) => {
                  requirements().justification!.minLength = Math.max(0, Math.trunc(Number(value) || 0));
                  updateReq();
                }}
              />
            )}
          </div>
          <div className="flex flex-col gap-3">
            <Switch
              size="xs"
              label="Require a ticket"
              checked={!!req.spec?.requirements?.ticket?.isRequired}
              onChange={(event) => {
                requirements().ticket = event.target.checked
                  ? AccessP.Catalog_Spec_Requirements_Ticket.create({ isRequired: true })
                  : undefined;
                updateReq();
              }}
            />
            {req.spec?.requirements?.ticket?.isRequired && (
              <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                <TextInput
                  label="Label"
                  description="How the ticket is called, for instance Change ticket."
                  placeholder="Ticket"
                  value={req.spec.requirements.ticket.label}
                  onChange={(event) => {
                    requirements().ticket!.label = event.target.value;
                    updateReq();
                  }}
                />
                <TextInput
                  label="Pattern"
                  description="Optional regular expression the ticket must match."
                  placeholder="^CHG-[0-9]+$"
                  classNames={{ input: "font-mono" }}
                  value={req.spec.requirements.ticket.pattern}
                  onChange={(event) => {
                    requirements().ticket!.pattern = event.target.value;
                    updateReq();
                  }}
                />
              </div>
            )}
          </div>
        </div>
      </EditItem>

      <EditItem title="Options">
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <span className="text-xs font-semibold text-slate-700">Namespace membership</span>
            <SegmentedControl
              size="xs"
              aria-label="Namespace membership"
              className="w-fit"
              value={req.spec?.membership === AccessP.Catalog_Spec_Membership.DYNAMIC ? "dynamic" : "fixed"}
              data={[
                { label: "Fixed at approval", value: "fixed" },
                { label: "Follows the Namespace", value: "dynamic" },
              ]}
              onChange={(value) => {
                req.spec!.membership =
                  value === "dynamic"
                    ? AccessP.Catalog_Spec_Membership.DYNAMIC
                    : AccessP.Catalog_Spec_Membership.MEMBERSHIP_UNSET;
                updateReq();
              }}
            />
            <span className="text-micro leading-4 text-slate-500">
              {req.spec?.membership === AccessP.Catalog_Spec_Membership.DYNAMIC
                ? "Services added to a Namespace during an approved access are included too."
                : "An approved access covers the Services of the Namespaces at the time it is approved."}
            </span>
          </div>
          <Switch
            size="xs"
            label="Let requesters pick individual Services"
            checked={!!req.spec?.allowPartialRequests}
            onChange={(event) => {
              req.spec!.allowPartialRequests = event.target.checked;
              updateReq();
            }}
          />
          <Switch
            size="xs"
            label="Include system Services"
            description="System and hidden Services, such as the API server, are excluded unless this is enabled."
            checked={!!req.spec?.allowSystemResources}
            onChange={(event) => {
              req.spec!.allowSystemResources = event.target.checked;
              updateReq();
            }}
          />
          <Switch
            size="xs"
            label="Disable this Catalog"
            description="Nobody can request it. Approved access stays until it ends or is revoked."
            checked={!!req.spec?.isDisabled}
            onChange={(event) => {
              req.spec!.isDisabled = event.target.checked;
              updateReq();
            }}
          />
        </div>
      </EditItem>

      <EditItem
        title="Catalog resources"
        description="Include specific Services and every Service belonging to selected Namespaces. At least one must be configured."
        obj={req.spec?.resourceCollection}
        onSet={() => {
          ensureCollection();
          updateReq();
        }}
        onUnset={() => {
          req.spec!.resourceCollection = undefined;
          updateReq();
        }}
      >
        {req.spec?.resourceCollection && (
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            <section className="rounded-xl border border-slate-200 bg-slate-50/50 p-3.5">
              <div className="mb-3 flex items-start gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm">
                  <PanelTop size={14} strokeWidth={2.2} />
                </span>
                <div>
                  <p className="text-body font-semibold text-slate-700">
                    Individual Services
                  </p>
                  <p className="mt-0.5 text-micro font-normal leading-4 text-slate-500">
                    Include only the selected Services.
                  </p>
                </div>
              </div>
              <SelectResourceMultiple
                api="core"
                kind="Service"
                label="Services"
                description="Services included directly in this Catalog."
                clearable
                defaultValue={collection?.services ?? []}
                onChange={(resources) => {
                  ensureCollection().services =
                    resources?.map((resource) => resource.metadata!.name) ?? [];
                  updateReq();
                }}
              />
            </section>

            <section className="rounded-xl border border-slate-200 bg-slate-50/50 p-3.5">
              <div className="mb-3 flex items-start gap-2.5">
                <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 shadow-sm">
                  <Library size={14} strokeWidth={2.2} />
                </span>
                <div>
                  <p className="text-body font-semibold text-slate-700">
                    Namespace Services
                  </p>
                  <p className="mt-0.5 text-micro font-normal leading-4 text-slate-500">
                    Include Services belonging to the selected Namespaces.
                  </p>
                </div>
              </div>
              <SelectResourceMultiple
                api="core"
                kind="Namespace"
                label="Namespaces"
                description="Include every Service belonging to these Namespaces."
                clearable
                defaultValue={collection?.namespaces ?? []}
                onChange={(resources) => {
                  ensureCollection().namespaces =
                    resources?.map((resource) => resource.metadata!.name) ?? [];
                  updateReq();
                }}
              />
            </section>
          </div>
        )}
      </EditItem>
    </div>
  );
};

export default Edit;
