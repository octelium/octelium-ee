import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import EditItem from "@/components/EditItem";
import ItemMessage from "@/components/ItemMessage";
import { ResourceEdit } from "@/components/ResourceLayout/ResourceEdit";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { getClientEnterprise } from "@/utils/client";
import { strToNum } from "@/utils/convert";
import { invalidateKey } from "@/utils/pb";
import {
  Alert,
  Button,
  Group,
  Loader,
  NumberInput,
  SegmentedControl,
  Switch,
  TextInput,
} from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangle, Trash2 } from "lucide-react";
import * as React from "react";
import { toast } from "sonner";

const pipelineTypes = EnterpriseP.ClusterConfig_Spec_Collector_Pipeline_Type;

const Edit = ({
  item,
  onUpdate,
}: {
  item: EnterpriseP.ClusterConfig;
  onUpdate: (item: EnterpriseP.ClusterConfig) => void;
}) => {
  const cloneForEdit = React.useCallback((value: EnterpriseP.ClusterConfig) => {
    const next = EnterpriseP.ClusterConfig.clone(value);
    if (!next.spec) next.spec = EnterpriseP.ClusterConfig_Spec.create();
    return next;
  }, []);
  const [req, setReq] = React.useState(() => cloneForEdit(item));
  const pipelineIDs = React.useRef<string[]>([]);
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    setReq(cloneForEdit(item));
    pipelineIDs.current = [];
  }, [cloneForEdit, itemKey]);

  const update = (mutate: (next: EnterpriseP.ClusterConfig) => void) => {
    const next = EnterpriseP.ClusterConfig.clone(req);
    if (!next.spec) next.spec = EnterpriseP.ClusterConfig_Spec.create();
    mutate(next);
    setReq(next);
    onUpdate(EnterpriseP.ClusterConfig.clone(next));
  };

  const pipelineID = (index: number) => {
    while (pipelineIDs.current.length <= index) {
      pipelineIDs.current.push(crypto.randomUUID());
    }
    return pipelineIDs.current[index];
  };

  return (
    <div className="w-full space-y-5 py-2">
      <EditItem
        title="Scaler"
        description="Set replica counts for enterprise cluster components"
        obj={req.spec?.scaler}
        onUnset={() => update((next) => (next.spec!.scaler = undefined))}
        onSet={() =>
          update((next) => {
            next.spec!.scaler = EnterpriseP.ClusterConfig_Spec_Scaler.create({
              collector: {},
              ingress: {},
              octovigil: {},
            });
          })
        }
      >
        {req.spec?.scaler && (
          <div className="grid gap-4 md:grid-cols-3">
            <NumberInput label="Collector replicas" description="OpenTelemetry Collector instances" min={0} max={32} value={req.spec.scaler.collector?.replicas ?? 0} onChange={(value) => update((next) => { next.spec!.scaler!.collector ??= EnterpriseP.ClusterConfig_Spec_Scaler_Collector.create(); next.spec!.scaler!.collector.replicas = strToNum(value); })} />
            <NumberInput label="Ingress replicas" description="Envoy ingress instances" min={0} max={32} value={req.spec.scaler.ingress?.replicas ?? 0} onChange={(value) => update((next) => { next.spec!.scaler!.ingress ??= EnterpriseP.ClusterConfig_Spec_Scaler_Ingress.create(); next.spec!.scaler!.ingress.replicas = strToNum(value); })} />
            <NumberInput label="Octovigil replicas" description="Octovigil instances" min={0} max={32} value={req.spec.scaler.octovigil?.replicas ?? 0} onChange={(value) => update((next) => { next.spec!.scaler!.octovigil ??= EnterpriseP.ClusterConfig_Spec_Scaler_Octovigil.create(); next.spec!.scaler!.octovigil.replicas = strToNum(value); })} />
          </div>
        )}
      </EditItem>

      <EditItem
        title="Collector"
        description="Configure telemetry pipelines and their exporters"
        obj={req.spec?.collector}
        onUnset={() => update((next) => (next.spec!.collector = undefined))}
        onSet={() => update((next) => { next.spec!.collector = EnterpriseP.ClusterConfig_Spec_Collector.create(); })}
      >
        {req.spec?.collector && (
          <ItemMessage
            title="Pipelines"
            obj={req.spec.collector.pipelines}
            isList
            onSet={() => update((next) => { next.spec!.collector!.pipelines = [EnterpriseP.ClusterConfig_Spec_Collector_Pipeline.create()]; })}
            onAddListItem={() => update((next) => { next.spec!.collector!.pipelines.push(EnterpriseP.ClusterConfig_Spec_Collector_Pipeline.create()); })}
          >
            <div className="space-y-3">
              {req.spec.collector.pipelines.map((pipeline, index) => (
                <div key={pipelineID(index)} className="rounded-xl border border-slate-200 bg-slate-50/60 p-3 sm:p-4">
                  <div className="mb-3 flex items-center justify-between gap-3">
                    <div className="text-xs font-bold uppercase tracking-wide text-slate-500">Pipeline {index + 1}</div>
                    <Button type="button" size="compact-xs" variant="subtle" color="red" leftSection={<Trash2 size={13} />} onClick={() => { pipelineIDs.current.splice(index, 1); update((next) => { next.spec!.collector!.pipelines.splice(index, 1); }); }}>Remove</Button>
                  </div>
                  <div className="grid items-start gap-4 lg:grid-cols-3">
                    <TextInput required label="Name" description="A unique pipeline name" placeholder="application-logs" value={pipeline.name} onChange={(event) => update((next) => { next.spec!.collector!.pipelines[index].name = event.currentTarget.value; })} />
                    <div>
                      <div className="mb-1 text-sm font-semibold text-slate-700">Signal type <span className="text-red-500">*</span></div>
                      <div className="mb-2 text-xs text-slate-500">Choose the telemetry signal handled by this pipeline</div>
                      <SegmentedControl fullWidth value={pipelineTypes[pipeline.type]} data={[{ label: "Logs", value: "LOGS" }, { label: "Metrics", value: "METRICS" }]} onChange={(value) => update((next) => { next.spec!.collector!.pipelines[index].type = pipelineTypes[value as "LOGS" | "METRICS"]; })} />
                    </div>
                    <SelectResourceMultiple api="enterprise" kind="CollectorExporter" label="Collector exporters" description="Destinations that receive this pipeline" defaultValue={pipeline.exporters} clearable onChange={(value) => update((next) => { next.spec!.collector!.pipelines[index].exporters = value?.map((resource) => resource.metadata!.name) ?? []; })} />
                  </div>
                  <Switch className="mt-4" label="Disable this pipeline" checked={pipeline.isDisabled} onChange={(event) => update((next) => { next.spec!.collector!.pipelines[index].isDisabled = event.currentTarget.checked; })} />
                </div>
              ))}
            </div>
          </ItemMessage>
        )}
      </EditItem>

      <EditItem
        title="Certificate"
        description="Choose the default certificate lifecycle mode"
        obj={req.spec?.certificate}
        onUnset={() => update((next) => (next.spec!.certificate = undefined))}
        onSet={() => update((next) => { next.spec!.certificate = EnterpriseP.ClusterConfig_Spec_Certificate.create({ defaultMode: EnterpriseP.Certificate_Spec_Mode.MANAGED }); })}
      >
        {req.spec?.certificate && (
          <div className="max-w-md">
            <div className="mb-1 text-sm font-semibold text-slate-700">Default mode</div>
            <div className="mb-2 text-xs text-slate-500">Apply managed or manual handling to certificates by default</div>
            <SegmentedControl fullWidth value={EnterpriseP.Certificate_Spec_Mode[req.spec.certificate.defaultMode] || "MANAGED"} data={[{ label: "Managed", value: "MANAGED" }, { label: "Manual", value: "MANUAL" }]} onChange={(value) => update((next) => { next.spec!.certificate!.defaultMode = EnterpriseP.Certificate_Spec_Mode[value as "MANAGED" | "MANUAL"]; })} />
          </div>
        )}
      </EditItem>
    </div>
  );
};

export default () => {
  const query = useQuery({
    queryKey: ["enterprise", "clusterconfig"],
    queryFn: async () => getClientEnterprise().getClusterConfig({}),
  });

  if (query.isLoading) return <div className="flex min-h-64 items-center justify-center"><Loader size="sm" color="gray" /></div>;
  if (query.isError) return <div className="mx-auto mt-10 max-w-xl"><Alert color="red" title="Could not load enterprise cluster configuration" icon={<AlertTriangle size={16} />}><div className="space-y-3"><div className="text-xs">{query.error.message}</div><Button type="button" size="compact-sm" variant="outline" onClick={() => query.refetch()}>Retry</Button></div></Alert></div>;
  if (!query.data?.response) return <div className="py-16 text-center text-sm font-semibold text-slate-500">Enterprise cluster configuration is unavailable.</div>;

  return (
    <ResourceEdit
      item={query.data.response}
      specComponent={({ item, onUpdate }) => <Edit item={item as EnterpriseP.ClusterConfig} onUpdate={(next) => onUpdate(next)} />}
      noPostUpdateNavigation
      noPostUpdateToast
      noMetadata
      onUpdateDone={() => {
        invalidateKey(["enterprise", "clusterconfig"]);
        toast.success("Enterprise ClusterConfig successfully updated");
      }}
    />
  );
};
