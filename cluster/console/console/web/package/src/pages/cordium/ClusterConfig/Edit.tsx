import * as CordiumP from "@/apis/cordiumv1/cordiumv1";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";
import { strToNum } from "@/utils/convert";
import { NumberInput, Select, Switch, TagsInput, TextInput } from "@mantine/core";
import * as React from "react";
import Agent from "./Agent";
import { ComputeLimit, RuleList } from "./fields";

const newCondition = () =>
  CordiumP.Condition.create({ type: { oneofKind: "match", match: "" } });

const computeLimits = [
  { key: "buildLimit", title: "Template build resources", description: "Compute resources allocated to Template pre-build Workspaces." },
  { key: "defaultOrganizationSpaceLimit", title: "Organization Space defaults", description: "Default resources for Workspaces in organization Spaces." },
  { key: "defaultUserSpaceLimit", title: "Personal Space defaults", description: "Default resources for Workspaces in personal Spaces." },
  { key: "maxLimit", title: "Maximum Workspace resources", description: "Hard caps that no Workspace in the Cluster can exceed. Leave a field empty to apply no cap for that resource." },
] as const;

const timeouts = [
  { key: "defaultDuration", title: "Default inactivity timeout", description: "Used when no Space-specific timeout is configured." },
  { key: "userSpaceDuration", title: "Personal Space inactivity timeout", description: "Inactivity timeout for Workspaces in personal Spaces." },
  { key: "organizationSpaceDuration", title: "Organization Space inactivity timeout", description: "Inactivity timeout for Workspaces in organization Spaces." },
  { key: "maxActiveDuration", title: "Maximum active duration", description: "Maximum time a Workspace can run, regardless of activity." },
] as const;

const ownershipEffects = CordiumP.ClusterConfig_Spec_Space_Ownership_Rule_Effect;

const Edit = (props: {
  item: CordiumP.ClusterConfig;
  onUpdate: (item: CordiumP.ClusterConfig) => void;
}) => {
  const cloneForEdit = (value: CordiumP.ClusterConfig) => {
    const next = CordiumP.ClusterConfig.clone(value);
    next.spec ??= CordiumP.ClusterConfig_Spec.create();
    return next;
  };
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));

  const update = (mutate: (spec: CordiumP.ClusterConfig_Spec) => void) => {
    const next = cloneForEdit(req);
    mutate(next.spec!);
    setReq(next);
    props.onUpdate(CordiumP.ClusterConfig.clone(next));
  };

  const spec = req.spec!;

  return (
    <div className="min-w-0 w-full space-y-5 py-2">
      <EditItem
        title="Spaces"
        description="Control which Users can create and own Spaces."
        obj={spec.space}
        onSet={() => update((next) => { next.space = CordiumP.ClusterConfig_Spec_Space.create(); })}
        onUnset={() => update((next) => { next.space = undefined; })}
      >
        {spec.space && (
          <RuleList
            title="Space ownership rules"
            description="Conditions can use ctx.user and ctx.space. DENY rules are checked before ALLOW rules. If no rule matches, or ownership is not configured, Space creation is denied."
            items={spec.space.ownership?.rules ?? []}
            create={() => CordiumP.ClusterConfig_Spec_Space_Ownership_Rule.create({ effect: ownershipEffects.ALLOW, condition: newCondition() })}
            onChange={(rules) => update((next) => { next.space!.ownership = CordiumP.ClusterConfig_Spec_Space_Ownership.create({ rules }); })}
          >
            {(rule, onChange) => (
              <Select label="Effect" required allowDeselect={false} data={[{ label: "Allow", value: "ALLOW" }, { label: "Deny", value: "DENY" }]} value={ownershipEffects[rule.effect]} onChange={(value) => value && onChange({ ...rule, effect: ownershipEffects[value as "ALLOW" | "DENY"] })} />
            )}
          </RuleList>
        )}
      </EditItem>

      <EditItem
        title="Workspaces"
        description="Configure storage, resource limits, inactivity timeouts and runtime capabilities."
        obj={spec.workspace}
        onSet={() => update((next) => { next.workspace = CordiumP.ClusterConfig_Spec_Workspace.create(); })}
        onUnset={() => update((next) => { next.workspace = undefined; })}
      >
        {spec.workspace && (
          <>
            <EditItem
              title="Workspace storage"
              description="Choose Kubernetes storage and snapshot classes using ordered rules."
              obj={spec.workspace.storage}
              onSet={() => update((next) => { next.workspace!.storage = CordiumP.ClusterConfig_Spec_Workspace_Storage.create(); })}
              onUnset={() => update((next) => { next.workspace!.storage = undefined; })}
            >
              {spec.workspace.storage && (
                <>
                  <RuleList
                    title="Workspace storage class rules"
                    description="The first matching rule selects the StorageClass. Conditions can use workspace."
                    items={spec.workspace.storage.storageClass?.rules ?? []}
                    create={() => CordiumP.ClusterConfig_Spec_Workspace_Storage_StorageClass_Rule.create({ condition: newCondition() })}
                    onChange={(rules) => update((next) => { next.workspace!.storage!.storageClass = CordiumP.ClusterConfig_Spec_Workspace_Storage_StorageClass.create({ rules }); })}
                  >
                    {(rule, onChange) => <TextInput label="Storage class" required placeholder="standard" value={rule.storageClass} onChange={(event) => onChange({ ...rule, storageClass: event.currentTarget.value })} />}
                  </RuleList>
                  <RuleList
                    title="Volume snapshot class rules"
                    description="The first matching rule selects the VolumeSnapshotClass. Conditions can use ctx.workspace and ctx.template. When no rule matches, Kubernetes uses the CSI driver's default snapshot class."
                    items={spec.workspace.storage.volumeSnapshotClass?.rules ?? []}
                    create={() => CordiumP.ClusterConfig_Spec_Workspace_Storage_VolumeSnapshotClass_Rule.create({ condition: newCondition() })}
                    onChange={(rules) => update((next) => { next.workspace!.storage!.volumeSnapshotClass = CordiumP.ClusterConfig_Spec_Workspace_Storage_VolumeSnapshotClass.create({ rules }); })}
                  >
                    {(rule, onChange) => <TextInput label="Volume snapshot class" required placeholder="default-snapshots" value={rule.volumeSnapshotClass} onChange={(event) => onChange({ ...rule, volumeSnapshotClass: event.currentTarget.value })} />}
                  </RuleList>
                </>
              )}
            </EditItem>

            <EditItem
              title="Workspace limits"
              description="Cluster-wide quotas and compute resources. A quota of 0 applies no Cluster-level restriction."
              obj={spec.workspace.limit}
              onSet={() => update((next) => { next.workspace!.limit = CordiumP.ClusterConfig_Spec_Workspace_Limit.create(); })}
              onUnset={() => update((next) => { next.workspace!.limit = undefined; })}
            >
              {spec.workspace.limit && (
                <>
                  <div className="grid gap-4 md:grid-cols-3">
                    <NumberInput label="Maximum Workspaces per User" min={0} max={4294967295} allowDecimal={false} value={spec.workspace.limit.maxPerUser} onChange={(value) => update((next) => { next.workspace!.limit!.maxPerUser = strToNum(value); })} />
                    <NumberInput label="Maximum active Workspaces per User" min={0} max={4294967295} allowDecimal={false} value={spec.workspace.limit.maxActivePerUser} onChange={(value) => update((next) => { next.workspace!.limit!.maxActivePerUser = strToNum(value); })} />
                    <NumberInput label="Maximum snapshots per User" min={0} max={4294967295} allowDecimal={false} value={spec.workspace.limit.maxSnapshotsPerUser} onChange={(value) => update((next) => { next.workspace!.limit!.maxSnapshotsPerUser = strToNum(value); })} />
                  </div>
                  {computeLimits.map(({ key, title, description }) => (
                    <ComputeLimit key={key} title={title} description={description} value={spec.workspace!.limit![key]} onChange={(value) => update((next) => { next.workspace!.limit![key] = value; })} />
                  ))}
                </>
              )}
            </EditItem>

            <EditItem
              title="Workspace timeouts"
              description="Stop inactive Workspaces automatically and cap how long they can run."
              obj={spec.workspace.timeout}
              onSet={() => update((next) => { next.workspace!.timeout = CordiumP.ClusterConfig_Spec_Workspace_Timeout.create(); })}
              onUnset={() => update((next) => { next.workspace!.timeout = undefined; })}
            >
              {spec.workspace.timeout && (
                <div className="space-y-4">
                  <div className="grid gap-4 md:grid-cols-2">
                    {timeouts.map(({ key, title, description }) => (
                      <DurationPicker key={key} title={title} description={description} value={spec.workspace!.timeout![key]} onChange={(value) => update((next) => { next.workspace!.timeout![key] = value; })} />
                    ))}
                  </div>
                  <Switch label="Allow no inactivity timeout" description="Let Workspaces disable their inactivity timeout. The maximum active duration still applies." checked={spec.workspace.timeout.allowNoTimeout} onChange={(event) => update((next) => { next.workspace!.timeout!.allowNoTimeout = event.currentTarget.checked; })} />
                </div>
              )}
            </EditItem>

            <EditItem
              title="Workspace runtime"
              description="Linux capabilities merged into every Workspace."
              obj={spec.workspace.runtime}
              onSet={() => update((next) => { next.workspace!.runtime = CordiumP.ClusterConfig_Spec_Workspace_Runtime.create(); })}
              onUnset={() => update((next) => { next.workspace!.runtime = undefined; })}
            >
              {spec.workspace.runtime && (
                <EditItem
                  title="Capabilities"
                  obj={spec.workspace.runtime.capabilities}
                  onSet={() => update((next) => { next.workspace!.runtime!.capabilities = CordiumP.Workspace_Spec_Runtime_Capabilities.create(); })}
                  onUnset={() => update((next) => { next.workspace!.runtime!.capabilities = undefined; })}
                >
                  {spec.workspace.runtime.capabilities && (
                    <div className="grid gap-4 md:grid-cols-2">
                      <TagsInput label="Add capabilities" placeholder="NET_ADMIN" value={spec.workspace.runtime.capabilities.add} onChange={(value) => update((next) => { next.workspace!.runtime!.capabilities!.add = value; })} />
                      <TagsInput label="Drop capabilities" placeholder="ALL" value={spec.workspace.runtime.capabilities.drop} onChange={(value) => update((next) => { next.workspace!.runtime!.capabilities!.drop = value; })} />
                    </div>
                  )}
                </EditItem>
              )}
            </EditItem>
          </>
        )}
      </EditItem>

      <EditItem
        title="Volumes"
        description="Configure persistent Volume storage and Cluster-wide limits."
        obj={spec.volume}
        onSet={() => update((next) => { next.volume = CordiumP.ClusterConfig_Spec_Volume.create(); })}
        onUnset={() => update((next) => { next.volume = undefined; })}
      >
        {spec.volume && (
          <>
            <EditItem
              title="Volume storage"
              obj={spec.volume.storage}
              onSet={() => update((next) => { next.volume!.storage = CordiumP.ClusterConfig_Spec_Volume_Storage.create(); })}
              onUnset={() => update((next) => { next.volume!.storage = undefined; })}
            >
              {spec.volume.storage && (
                <RuleList
                  title="Volume storage class rules"
                  description="The first matching rule selects the StorageClass. Conditions can use ctx.volume to distinguish shared and exclusive Volumes. When no rule matches, Kubernetes uses its default storage class."
                  items={spec.volume.storage.storageClass?.rules ?? []}
                  create={() => CordiumP.ClusterConfig_Spec_Volume_Storage_StorageClass_Rule.create({ condition: newCondition() })}
                  onChange={(rules) => update((next) => { next.volume!.storage!.storageClass = CordiumP.ClusterConfig_Spec_Volume_Storage_StorageClass.create({ rules }); })}
                >
                  {(rule, onChange) => <TextInput label="Storage class" required placeholder="shared-storage" value={rule.storageClass} onChange={(event) => onChange({ ...rule, storageClass: event.currentTarget.value })} />}
                </RuleList>
              )}
            </EditItem>

            <EditItem
              title="Volume limits"
              description="A quota of 0 applies no Cluster-level restriction. Leave the maximum size empty to apply no size cap."
              obj={spec.volume.limit}
              onSet={() => update((next) => { next.volume!.limit = CordiumP.ClusterConfig_Spec_Volume_Limit.create(); })}
              onUnset={() => update((next) => { next.volume!.limit = undefined; })}
            >
              {spec.volume.limit && (
                <div className="grid gap-4 md:grid-cols-2">
                  <NumberInput label="Maximum Volumes per Space" min={0} max={4294967295} allowDecimal={false} value={spec.volume.limit.maxPerSpace} onChange={(value) => update((next) => { next.volume!.limit!.maxPerSpace = strToNum(value); })} />
                  <NumberInput label="Maximum mounts per Workspace" min={0} max={4294967295} allowDecimal={false} value={spec.volume.limit.maxMountsPerWorkspace} onChange={(value) => update((next) => { next.volume!.limit!.maxMountsPerWorkspace = strToNum(value); })} />
                  <NumberInput label="Default Volume size (MB)" min={0} max={4294967295} allowDecimal={false} placeholder="Default" value={spec.volume.limit.defaultSize?.megabytes ?? ""} onChange={(value) => update((next) => { next.volume!.limit!.defaultSize = value === "" ? undefined : { megabytes: strToNum(value) }; })} />
                  <NumberInput label="Maximum Volume size (MB)" min={0} max={4294967295} allowDecimal={false} placeholder="No Cluster limit" value={spec.volume.limit.maxSize?.megabytes ?? ""} onChange={(value) => update((next) => { next.volume!.limit!.maxSize = value === "" ? undefined : { megabytes: strToNum(value) }; })} />
                </div>
              )}
            </EditItem>
          </>
        )}
      </EditItem>

      <EditItem
        title="AI agent"
        description="Defaults for the Cordium AI agent and its Workspaces."
        obj={spec.agent}
        onSet={() => update((next) => { next.agent = CordiumP.ClusterConfig_Spec_Agent.create(); })}
        onUnset={() => update((next) => { next.agent = undefined; })}
      >
        {spec.agent && <Agent value={spec.agent} onChange={(value) => update((next) => { next.agent = value; })} />}
      </EditItem>
    </div>
  );
};

export default Edit;
