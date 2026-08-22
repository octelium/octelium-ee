import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Duration } from "@/apis/metav1/metav1";
import ItemMessage from "@/components/ItemMessage";
import DurationPicker from "@/components/DurationPicker";
import EditItem from "@/components/EditItem";

import SelectResource from "@/components/ResourceLayout/SelectResource";
import {
  Alert,
  CloseButton,
  Group,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  Tabs,
  TextInput,
  Textarea,
} from "@mantine/core";
import { AlertTriangle } from "lucide-react";
import * as React from "react";
import { match } from "ts-pattern";

type SecretSelector = {
  type:
    | { oneofKind: "fromSecret"; fromSecret: string }
    | { oneofKind: undefined };
};

type GrpcStyleAuth = {
  type:
    | { oneofKind: "bearer"; bearer: SecretSelector }
    | {
        oneofKind: "basic";
        basic: { username: string; password?: SecretSelector };
      }
    | {
        oneofKind: "custom";
        custom: { header: string; value?: SecretSelector };
      }
    | { oneofKind: undefined };
};

const freshSecret = (): SecretSelector => ({
  type: { oneofKind: "fromSecret", fromSecret: "" },
});

const freshBearer = (): GrpcStyleAuth["type"] => ({
  oneofKind: "bearer",
  bearer: freshSecret(),
});

const freshBasic = (): GrpcStyleAuth["type"] => ({
  oneofKind: "basic",
  basic: { username: "", password: freshSecret() },
});

const freshCustom = (): GrpcStyleAuth["type"] => ({
  oneofKind: "custom",
  custom: { header: "", value: freshSecret() },
});

const normalizeSecretSelectors = (item: EnterpriseP.CollectorExporter) => {
  const type = item.spec?.type;
  if (!type?.oneofKind) return item;
  const normalizeAuth = (auth?: GrpcStyleAuth) => {
    if (auth && !auth.type.oneofKind) auth.type = freshBearer();
    if (auth?.type.oneofKind === "bearer" && !auth.type.bearer) auth.type.bearer = freshSecret();
    if (auth?.type.oneofKind === "basic" && !auth.type.basic.password) auth.type.basic.password = freshSecret();
    if (auth?.type.oneofKind === "custom" && !auth.type.custom.value) auth.type.custom.value = freshSecret();
  };
  if (type.oneofKind === "otlp") {
    if (!type.otlp.auth) type.otlp.auth = EnterpriseP.CollectorExporter_Spec_OTLP_Auth.create({ type: freshBearer() });
    normalizeAuth(type.otlp.auth as unknown as GrpcStyleAuth);
  }
  if (type.oneofKind === "otlpHTTP") {
    if (!type.otlpHTTP.auth) type.otlpHTTP.auth = EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth.create({ type: freshBearer() });
    normalizeAuth(type.otlpHTTP.auth as unknown as GrpcStyleAuth);
  }
  if (type.oneofKind === "prometheusRemoteWrite") {
    if (!type.prometheusRemoteWrite.auth) type.prometheusRemoteWrite.auth = EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth.create({ type: freshBearer() });
    normalizeAuth(type.prometheusRemoteWrite.auth as unknown as GrpcStyleAuth);
  }
  if (type.oneofKind === "clickhouse" && !type.clickhouse.password) type.clickhouse.password = freshSecret();
  if (type.oneofKind === "elasticsearch" && type.elasticsearch.auth?.type.oneofKind === "apiKey" && !type.elasticsearch.auth.type.apiKey) type.elasticsearch.auth.type.apiKey = freshSecret();
  if (type.oneofKind === "elasticsearch" && type.elasticsearch.auth?.type.oneofKind === "basic" && !type.elasticsearch.auth.type.basic.password) type.elasticsearch.auth.type.basic.password = freshSecret();
  if (type.oneofKind === "elasticsearch" && !type.elasticsearch.auth?.type.oneofKind) type.elasticsearch.auth = EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth.create({ type: { oneofKind: "apiKey", apiKey: freshSecret() } });
  if (type.oneofKind === "logzio" && !type.logzio.token) type.logzio.token = freshSecret();
  if (type.oneofKind === "influxDB" && !type.influxDB.token) type.influxDB.token = freshSecret();
  if (type.oneofKind === "kafka" && type.kafka.auth?.type.oneofKind === "sasl" && !type.kafka.auth.type.sasl.password) type.kafka.auth.type.sasl.password = freshSecret();
  if (type.oneofKind === "datadog") {
    if (!type.datadog.api) type.datadog.api = EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
    if (!type.datadog.api.key) type.datadog.api.key = freshSecret();
  }
  if (type.oneofKind === "splunk" && !type.splunk.token) type.splunk.token = freshSecret();
  if (type.oneofKind === "azureMonitor") {
    if (!type.azureMonitor.connectionString) type.azureMonitor.connectionString = freshSecret();
    if (!type.azureMonitor.instrumentationKey) type.azureMonitor.instrumentationKey = freshSecret();
  }
  if (type.oneofKind === "azureDataExplorer" && type.azureDataExplorer.auth?.type.oneofKind === "servicePrincipal" && !type.azureDataExplorer.auth.type.servicePrincipal.applicationKey) type.azureDataExplorer.auth.type.servicePrincipal.applicationKey = freshSecret();
  if (type.oneofKind === "azureDataExplorer" && !type.azureDataExplorer.auth?.type.oneofKind) type.azureDataExplorer.auth = EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth.create({ type: { oneofKind: "servicePrincipal", servicePrincipal: { applicationID: "", applicationKey: freshSecret(), tenantID: "" } } });
  return item;
};

const SecretSelectField = (props: {
  label: string;
  sel?: SecretSelector;
  description?: string;
  onUpdate: () => void;
}) => {
  const { label, sel, onUpdate } = props;
  const secretDescription = `Secret containing the ${label.replace(/\s+secret$/i, "").toLowerCase()}.`;

  if (!sel || sel.type.oneofKind !== "fromSecret") {
    return <></>;
  }

  return (
    <SelectResource
      api="enterprise"
      kind="Secret"
      required
      label={label}
      description={props.description ?? secretDescription}
      defaultValue={sel.type.fromSecret}
      onChange={(v) => {
        if (sel.type.oneofKind === "fromSecret") {
          sel.type.fromSecret = v?.metadata?.name ?? "";
          onUpdate();
        }
      }}
    />
  );
};

const SecretAuthTabs = (props: {
  auth?: GrpcStyleAuth;
  ensureAuth: () => GrpcStyleAuth;
  initAuth?: GrpcStyleAuth;
  onUpdate: () => void;
}) => {
  const { auth, ensureAuth, initAuth, onUpdate } = props;
  const configurations = React.useRef<Partial<Record<string, GrpcStyleAuth["type"]>>>({});

  React.useEffect(() => {
    if (auth?.type.oneofKind) {
      configurations.current = {
        [auth.type.oneofKind]: structuredClone(auth.type),
      };
    }
  }, [initAuth]);

  const changeAuth = (value: string) => {
    const a = ensureAuth();
    if (a.type.oneofKind) {
      configurations.current[a.type.oneofKind] = structuredClone(a.type);
    }
    const cached = configurations.current[value];
    if (cached) {
      a.type = structuredClone(cached);
    } else if (value === "bearer") {
      a.type = freshBearer();
    } else if (value === "basic") {
      a.type = freshBasic();
    } else if (value === "custom") {
      a.type = freshCustom();
    } else a.type = freshBearer();
    onUpdate();
  };

  return (
    <div className="space-y-3 rounded-xl border border-slate-200 bg-slate-50/50 p-3">
      <SegmentedControl
        fullWidth
        value={auth?.type.oneofKind ?? "bearer"}
        onChange={changeAuth}
        data={[
          { label: "Bearer", value: "bearer" },
          { label: "Basic", value: "basic" },
          { label: "Custom header", value: "custom" },
        ]}
      />

      {auth?.type.oneofKind === "bearer" && (
        match(auth?.type)
          .with({ oneofKind: "bearer" }, (b) => (
            <SecretSelectField
              label="Bearer Token Secret"
              sel={b.bearer}
              onUpdate={onUpdate}
            />
          ))
          .otherwise(() => (
            <></>
          ))
      )}

      {auth?.type.oneofKind === "basic" && (
        match(auth?.type)
          .with({ oneofKind: "basic" }, (b) => (
            <Group grow>
              <TextInput
                label="Username"
                description="Username used for HTTP Basic authentication."
                placeholder="username"
                value={b.basic.username}
                onChange={(v) => {
                  b.basic.username = v.target.value;
                  onUpdate();
                }}
              />
              <SecretSelectField
                label="Password Secret"
                sel={b.basic.password}
                onUpdate={onUpdate}
              />
            </Group>
          ))
          .otherwise(() => (
            <></>
          ))
      )}

      {auth?.type.oneofKind === "custom" && (
        match(auth?.type)
          .with({ oneofKind: "custom" }, (c) => (
            <Group grow>
              <TextInput
                required
                label="Header"
                description="HTTP header used to carry the custom credential."
                placeholder="X-Custom-Auth"
                value={c.custom.header}
                onChange={(v) => {
                  c.custom.header = v.target.value;
                  onUpdate();
                }}
              />
              <SecretSelectField
                label="Header Value Secret"
                sel={c.custom.value}
                onUpdate={onUpdate}
              />
            </Group>
          ))
          .otherwise(() => (
            <></>
          ))
      )}
    </div>
  );
};

const HeaderList = (props: {
  headers: { key: string; value: string }[];
  makeHeader: () => { key: string; value: string };
  onUpdate: () => void;
}) => {
  const { headers, makeHeader, onUpdate } = props;
  const nextID = React.useRef(0);
  const itemIDs = React.useRef<string[]>([]);
  while (itemIDs.current.length < headers.length) {
    itemIDs.current.push(`header-${nextID.current++}`);
  }
  if (itemIDs.current.length > headers.length) {
    itemIDs.current.length = headers.length;
  }

  return (
    <Group grow>
      <ItemMessage
        title="Add Headers"
        obj={headers}
        isList
        onSet={() => {
          headers.push(makeHeader());
          itemIDs.current.push(`header-${nextID.current++}`);
          onUpdate();
        }}
        onAddListItem={() => {
          headers.push(makeHeader());
          itemIDs.current.push(`header-${nextID.current++}`);
          onUpdate();
        }}
      >
        {headers.map((x, idx) => (
          <div className="w-full flex mb-3" key={itemIDs.current[idx]}>
            <CloseButton
              size="sm"
              variant="subtle"
              className="mr-2"
              onClick={() => {
                headers.splice(idx, 1);
                itemIDs.current.splice(idx, 1);
                onUpdate();
              }}
            ></CloseButton>
            <Group className="flex w-full" grow>
              <TextInput
                required
                label="Key"
                description="Set the Header key"
                placeholder="MY_KEY"
                value={x.key}
                onChange={(v) => {
                  x.key = v.target.value;
                  onUpdate();
                }}
              />
              <TextInput
                required
                label="Value"
                description="Set the Header value"
                placeholder="my-value"
                value={x.value}
                onChange={(v) => {
                  x.value = v.target.value;
                  onUpdate();
                }}
              />
            </Group>
          </div>
        ))}
      </ItemMessage>
    </Group>
  );
};

const EnumSelect = (props: {
  label: string;
  description?: string;
  enumObj: Record<string, string | number>;
  value: number;
  options: { label: string; value: number }[];
  onChange: (v: number) => void;
}) => {
  const { label, description, enumObj, value, options, onChange } = props;

  return (
    <Select
      label={label}
      description={description ?? `Select the ${label.toLowerCase()}.`}
      data={options.map((o) => ({
        label: o.label,
        value: String(enumObj[o.value]),
      }))}
      value={String(enumObj[value])}
      onChange={(v) => {
        if (v === null) {
          return;
        }
        onChange(Number(enumObj[v as keyof typeof enumObj]));
      }}
    />
  );
};

const DurationField = (props: {
  label: string;
  description?: string;
  value?: Duration;
  onChange: (value?: Duration) => void;
}) => (
  <DurationPicker
    title={props.label}
    description={props.description ?? `Duration for ${props.label.toLowerCase()}.`}
    value={props.value}
    onChange={props.onChange}
  />
);

const StringListField = (props: {
  label: string;
  description?: string;
  value: string[];
  onChange: (value: string[]) => void;
}) => (
  <TextInput
    label={props.label}
    description={props.description ?? "Separate multiple values with commas."}
    value={props.value.join(", ")}
    onChange={(event) =>
      props.onChange(
        event.currentTarget.value
          .split(",")
          .map((value) => value.trim())
          .filter(Boolean),
      )
    }
  />
);

const AdvancedSettings = (props: {
  type: EnterpriseP.CollectorExporter_Spec["type"];
  onUpdate: () => void;
}) => {
  const { type, onUpdate } = props;

  if (type.oneofKind === "otlp") {
    const value = type.otlp;
    return (
      <EditItem title="Advanced delivery" obj={value}>
        <div className="grid gap-4 md:grid-cols-2">
          <DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} />
          <TextInput label="Authority" description="Override the gRPC :authority pseudo-header." value={value.authority} onChange={(event) => { value.authority = event.currentTarget.value; onUpdate(); }} />
          <TextInput label="User agent" description="Override the user-agent sent to the upstream endpoint." value={value.userAgent} onChange={(event) => { value.userAgent = event.currentTarget.value; onUpdate(); }} />
          <TextInput label="Balancer name" description="gRPC load-balancing policy used for the upstream connection." value={value.balancerName} onChange={(event) => { value.balancerName = event.currentTarget.value; onUpdate(); }} />
          <NumberInput min={0} label="Read buffer size" description="Size in bytes of the upstream connection read buffer." value={value.readBufferSize} onChange={(next) => { value.readBufferSize = Number(next) || 0; onUpdate(); }} />
          <NumberInput min={0} label="Write buffer size" description="Size in bytes of the upstream connection write buffer." value={value.writeBufferSize} onChange={(next) => { value.writeBufferSize = Number(next) || 0; onUpdate(); }} />
        </div>
        <EditItem
          title="TLS details"
          obj={value.tls}
          onSet={() => { value.tls = EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create(); onUpdate(); }}
          onUnset={() => { value.tls = undefined; onUpdate(); }}
        >
          {value.tls && <div className="grid gap-4 md:grid-cols-2"><TextInput label="Server name override" description="Optional hostname used when verifying the upstream certificate." value={value.tls.serverNameOverride} onChange={(event) => { value.tls!.serverNameOverride = event.currentTarget.value; onUpdate(); }} /><Textarea label="CA certificate PEM" description="Optional PEM-encoded CA used to verify the upstream certificate." autosize minRows={3} value={value.tls.caPEM} onChange={(event) => { value.tls!.caPEM = event.currentTarget.value; onUpdate(); }} /></div>}
        </EditItem>
        <EditItem title="Keepalive" obj={value.keepalive} onSet={() => { value.keepalive = EnterpriseP.CollectorExporter_Spec_OTLP_Keepalive.create(); onUpdate(); }} onUnset={() => { value.keepalive = undefined; onUpdate(); }}>
          {value.keepalive && <div className="grid gap-4 md:grid-cols-2"><DurationField label="Keepalive time" value={value.keepalive.time} onChange={(next) => { value.keepalive!.time = next; onUpdate(); }} /><DurationField label="Keepalive timeout" value={value.keepalive.timeout} onChange={(next) => { value.keepalive!.timeout = next; onUpdate(); }} /><Switch label="Permit without stream" description="Allow keepalive pings when no streams are active." checked={value.keepalive.permitWithoutStream} onChange={(event) => { value.keepalive!.permitWithoutStream = event.currentTarget.checked; onUpdate(); }} /></div>}
        </EditItem>
      </EditItem>
    );
  }

  if (type.oneofKind === "otlpHTTP") {
    const value = type.otlpHTTP;
    return <EditItem title="Advanced delivery" obj={value}><div className="grid gap-4 md:grid-cols-2"><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><NumberInput min={0} label="Read buffer size" description="Size in bytes of the upstream connection read buffer." value={value.readBufferSize} onChange={(next) => { value.readBufferSize = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Write buffer size" description="Size in bytes of the upstream connection write buffer." value={value.writeBufferSize} onChange={(next) => { value.writeBufferSize = Number(next) || 0; onUpdate(); }} /></div><EditItem title="TLS" obj={value.tls} onSet={() => { value.tls = EnterpriseP.CollectorExporter_Spec_OTLPHTTP_TLS.create(); onUpdate(); }} onUnset={() => { value.tls = undefined; onUpdate(); }}>{value.tls && <div className="grid gap-4 md:grid-cols-2"><Switch label="No TLS" description="Disable TLS and connect over plaintext." checked={value.tls.insecure} onChange={(event) => { value.tls!.insecure = event.currentTarget.checked; onUpdate(); }} /><Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls.insecureSkipVerify} onChange={(event) => { value.tls!.insecureSkipVerify = event.currentTarget.checked; onUpdate(); }} /><TextInput label="Server name override" description="Optional hostname used when verifying the upstream certificate." value={value.tls.serverNameOverride} onChange={(event) => { value.tls!.serverNameOverride = event.currentTarget.value; onUpdate(); }} /><Textarea label="CA certificate PEM" description="Optional PEM-encoded CA used to verify the upstream certificate." autosize minRows={3} value={value.tls.caPEM} onChange={(event) => { value.tls!.caPEM = event.currentTarget.value; onUpdate(); }} />{value.tls.insecureSkipVerify && <Alert color="red" icon={<AlertTriangle size={14} />}>Server certificates will not be verified.</Alert>}</div>}</EditItem></EditItem>;
  }

  if (type.oneofKind === "prometheusRemoteWrite") {
    const value = type.prometheusRemoteWrite;
    return <EditItem title="Advanced delivery" obj={value}><HeaderList headers={value.headers} makeHeader={() => EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Header.create()} onUpdate={onUpdate} /><HeaderList headers={value.externalLabels} makeHeader={() => EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel.create()} onUpdate={onUpdate} /><div className="grid gap-4 md:grid-cols-2"><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><NumberInput min={0} label="Maximum batch size (bytes)" description="Maximum size of a single remote-write request." value={value.maxBatchSizeBytes} onChange={(next) => { value.maxBatchSizeBytes = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Maximum parallel requests" description="Maximum concurrent requests sent for one batch." value={value.maxBatchRequestParallelism} onChange={(next) => { value.maxBatchRequestParallelism = Number(next) || 0; onUpdate(); }} /></div><EditItem title="Remote write queue" obj={value.remoteWriteQueue} onSet={() => { value.remoteWriteQueue = EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue.create(); onUpdate(); }} onUnset={() => { value.remoteWriteQueue = undefined; onUpdate(); }}>{value.remoteWriteQueue && <div className="grid gap-4 md:grid-cols-3"><Switch label="Enabled" description="Buffer metric batches before sending them." checked={value.remoteWriteQueue.enabled} onChange={(event) => { value.remoteWriteQueue!.enabled = event.currentTarget.checked; onUpdate(); }} /><NumberInput min={0} label="Queue size" description="Maximum number of metric batches buffered in the queue." value={value.remoteWriteQueue.queueSize} onChange={(next) => { value.remoteWriteQueue!.queueSize = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Consumers" description="Number of workers sending queued batches." value={value.remoteWriteQueue.numConsumers} onChange={(next) => { value.remoteWriteQueue!.numConsumers = Number(next) || 0; onUpdate(); }} /></div>}</EditItem><EditItem title="Resource conversion" obj={value.resourceToTelemetryConversion} onSet={() => { value.resourceToTelemetryConversion = EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_ResourceToTelemetryConversion.create(); onUpdate(); }} onUnset={() => { value.resourceToTelemetryConversion = undefined; onUpdate(); }}>{value.resourceToTelemetryConversion && <Group><Switch label="Enabled" description="Convert OpenTelemetry resource attributes into Prometheus labels." checked={value.resourceToTelemetryConversion.enabled} onChange={(event) => { value.resourceToTelemetryConversion!.enabled = event.currentTarget.checked; onUpdate(); }} /><Switch label="Exclude service attributes" description="Exclude service.* resource attributes from the conversion." checked={value.resourceToTelemetryConversion.excludeServiceAttributes} onChange={(event) => { value.resourceToTelemetryConversion!.excludeServiceAttributes = event.currentTarget.checked; onUpdate(); }} /></Group>}</EditItem><Switch label="Target information" description="Export the target_info metric with resource attributes." checked={value.targetInfo?.enabled ?? false} onChange={(event) => { value.targetInfo = EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TargetInfo.create({ enabled: event.currentTarget.checked }); onUpdate(); }} /></EditItem>;
  }

  if (type.oneofKind === "clickhouse") {
    const value = type.clickhouse;
    return <EditItem title="Advanced delivery" obj={value}><HeaderList headers={value.connectionParams} makeHeader={() => EnterpriseP.CollectorExporter_Spec_Clickhouse_ConnectionParam.create()} onUpdate={onUpdate} /><div className="grid gap-4 md:grid-cols-2"><DurationField label="TTL" description="Retention duration for exported telemetry rows." value={value.ttl} onChange={(next) => { value.ttl = next; onUpdate(); }} /><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /></div><EditItem title="Metrics tables" obj={value.metricsTables} onSet={() => { value.metricsTables = EnterpriseP.CollectorExporter_Spec_Clickhouse_MetricsTables.create(); onUpdate(); }} onUnset={() => { value.metricsTables = undefined; onUpdate(); }}>{value.metricsTables && <div className="grid gap-4 md:grid-cols-2">{(["gauge", "sum", "summary", "histogram", "exponentialHistogram"] as const).map((field) => <TextInput key={field} label={field === "exponentialHistogram" ? "Exponential histogram" : field[0].toUpperCase() + field.slice(1)} description="ClickHouse table name for this metric type." value={value.metricsTables![field]} onChange={(event) => { value.metricsTables![field] = event.currentTarget.value; onUpdate(); }} />)}</div>}</EditItem><EditItem title="Table engine" obj={value.tableEngine} onSet={() => { value.tableEngine = EnterpriseP.CollectorExporter_Spec_Clickhouse_TableEngine.create(); onUpdate(); }} onUnset={() => { value.tableEngine = undefined; onUpdate(); }}>{value.tableEngine && <div className="grid gap-4 md:grid-cols-2"><TextInput label="Engine name" description="ClickHouse table engine name, such as MergeTree." value={value.tableEngine.name} onChange={(event) => { value.tableEngine!.name = event.currentTarget.value; onUpdate(); }} /><TextInput label="Parameters" description="Parameters passed to the ClickHouse table engine." value={value.tableEngine.params} onChange={(event) => { value.tableEngine!.params = event.currentTarget.value; onUpdate(); }} /></div>}</EditItem><EditItem title="TLS" obj={value.tls} onSet={() => { value.tls = EnterpriseP.CollectorExporter_Spec_Clickhouse_TLS.create(); onUpdate(); }} onUnset={() => { value.tls = undefined; onUpdate(); }}>{value.tls && <div className="grid gap-4 md:grid-cols-2"><Switch label="No TLS" description="Disable TLS and connect over plaintext." checked={value.tls.insecure} onChange={(event) => { value.tls!.insecure = event.currentTarget.checked; onUpdate(); }} /><Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls.insecureSkipVerify} onChange={(event) => { value.tls!.insecureSkipVerify = event.currentTarget.checked; onUpdate(); }} /><TextInput label="Server name override" description="Optional hostname used when verifying the upstream certificate." value={value.tls.serverNameOverride} onChange={(event) => { value.tls!.serverNameOverride = event.currentTarget.value; onUpdate(); }} /><Textarea label="CA certificate PEM" description="Optional PEM-encoded CA used to verify the upstream certificate." autosize minRows={3} value={value.tls.caPEM} onChange={(event) => { value.tls!.caPEM = event.currentTarget.value; onUpdate(); }} /></div>}</EditItem></EditItem>;
  }

  if (type.oneofKind === "elasticsearch") {
    const value = type.elasticsearch;
    return <EditItem title="Advanced delivery" obj={value}><HeaderList headers={value.headers} makeHeader={() => EnterpriseP.CollectorExporter_Spec_Elasticsearch_Header.create()} onUpdate={onUpdate} /><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls?.insecureSkipVerify ?? false} onChange={(event) => { value.tls = EnterpriseP.CollectorExporter_Spec_Elasticsearch_TLS.create({ insecureSkipVerify: event.currentTarget.checked }); onUpdate(); }} />{value.tls?.insecureSkipVerify && <Alert color="red" icon={<AlertTriangle size={14} />}>Server certificates will not be verified.</Alert>}</EditItem>;
  }

  if (type.oneofKind === "logzio") return <EditItem title="Advanced delivery" obj={type.logzio}><DurationField label="Timeout" value={type.logzio.timeout} onChange={(next) => { type.logzio.timeout = next; onUpdate(); }} /></EditItem>;

  if (type.oneofKind === "influxDB") {
    const value = type.influxDB;
    return <EditItem title="Advanced delivery" obj={value}><HeaderList headers={value.headers} makeHeader={() => EnterpriseP.CollectorExporter_Spec_InfluxDB_Header.create()} onUpdate={onUpdate} /><div className="grid gap-4 md:grid-cols-2"><NumberInput min={0} label="Maximum payload lines" description="Maximum number of lines sent in one export request." value={value.payloadMaxLines} onChange={(next) => { value.payloadMaxLines = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Maximum payload bytes" description="Maximum size in bytes of one export request." value={value.payloadMaxBytes} onChange={(next) => { value.payloadMaxBytes = Number(next) || 0; onUpdate(); }} /><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><StringListField label="Log record dimensions" value={value.logRecordDimensions} onChange={(next) => { value.logRecordDimensions = next; onUpdate(); }} /></div><EditItem title="InfluxDB v1 compatibility" obj={value.v1Compatibility} onSet={() => { value.v1Compatibility = EnterpriseP.CollectorExporter_Spec_InfluxDB_V1Compatibility.create(); onUpdate(); }} onUnset={() => { value.v1Compatibility = undefined; onUpdate(); }}>{value.v1Compatibility && <div className="grid gap-4 md:grid-cols-2"><Switch label="Enabled" description="Enable InfluxDB v1.x compatibility mode." checked={value.v1Compatibility.enabled} onChange={(event) => { value.v1Compatibility!.enabled = event.currentTarget.checked; onUpdate(); }} /><TextInput label="Database" description="InfluxDB v1.x database receiving telemetry." value={value.v1Compatibility.db} onChange={(event) => { value.v1Compatibility!.db = event.currentTarget.value; onUpdate(); }} /><TextInput label="Username" description="InfluxDB v1.x username." value={value.v1Compatibility.username} onChange={(event) => { value.v1Compatibility!.username = event.currentTarget.value; onUpdate(); }} /><SecretSelectField label="Password Secret" sel={value.v1Compatibility.password} onUpdate={onUpdate} /></div>}</EditItem></EditItem>;
  }

  if (type.oneofKind === "kafka") {
    const value = type.kafka;
    return <EditItem title="Advanced delivery" obj={value}><HeaderList headers={value.recordHeaders} makeHeader={() => EnterpriseP.CollectorExporter_Spec_Kafka_Header.create()} onUpdate={onUpdate} /><div className="grid gap-4 md:grid-cols-2"><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><DurationField label="Connection idle timeout" value={value.connIdleTimeout} onChange={(next) => { value.connIdleTimeout = next; onUpdate(); }} /><Switch label="Partition logs by resource attributes" description="Use resource attributes when selecting the Kafka partition for logs." checked={value.partitionLogsByResourceAttributes} onChange={(event) => { value.partitionLogsByResourceAttributes = event.currentTarget.checked; onUpdate(); }} /><Switch label="Partition metrics by resource attributes" description="Use resource attributes when selecting the Kafka partition for metrics." checked={value.partitionMetricsByResourceAttributes} onChange={(event) => { value.partitionMetricsByResourceAttributes = event.currentTarget.checked; onUpdate(); }} /></div><EditItem title="TLS" obj={value.tls} onSet={() => { value.tls = EnterpriseP.CollectorExporter_Spec_Kafka_TLS.create(); onUpdate(); }} onUnset={() => { value.tls = undefined; onUpdate(); }}>{value.tls && <Group><Switch label="No TLS" description="Disable TLS and connect over plaintext." checked={value.tls.insecure} onChange={(event) => { value.tls!.insecure = event.currentTarget.checked; onUpdate(); }} /><Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls.insecureSkipVerify} onChange={(event) => { value.tls!.insecureSkipVerify = event.currentTarget.checked; onUpdate(); }} /></Group>}</EditItem><EditItem title="Producer" obj={value.producer} onSet={() => { value.producer = EnterpriseP.CollectorExporter_Spec_Kafka_Producer.create(); onUpdate(); }} onUnset={() => { value.producer = undefined; onUpdate(); }}>{value.producer && <div className="grid gap-4 md:grid-cols-2"><NumberInput min={0} label="Maximum message bytes" description="Maximum size of a Kafka message in bytes." value={value.producer.maxMessageBytes} onChange={(next) => { value.producer!.maxMessageBytes = Number(next) || 0; onUpdate(); }} /><NumberInput label="Required acknowledgements" description="Number of broker acknowledgements required before a message is considered written." value={value.producer.requiredAcks} onChange={(next) => { value.producer!.requiredAcks = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Flush maximum messages" description="Maximum messages accumulated before a producer flush." value={value.producer.flushMaxMessages} onChange={(next) => { value.producer!.flushMaxMessages = Number(next) || 0; onUpdate(); }} /><Switch label="Allow automatic topic creation" description="Allow the producer to create missing Kafka topics automatically." checked={value.producer.allowAutoTopicCreation} onChange={(event) => { value.producer!.allowAutoTopicCreation = event.currentTarget.checked; onUpdate(); }} /><DurationField label="Linger" value={value.producer.linger} onChange={(next) => { value.producer!.linger = next; onUpdate(); }} /><EnumSelect label="Compression" enumObj={EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression} value={value.producer.compression} options={[{ label: "None", value: EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression.NONE }, { label: "Gzip", value: EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression.GZIP }, { label: "Snappy", value: EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression.SNAPPY }, { label: "LZ4", value: EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression.LZ4 }, { label: "Zstd", value: EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression.ZSTD }]} onChange={(next) => { value.producer!.compression = next; onUpdate(); }} /></div>}</EditItem></EditItem>;
  }

  if (type.oneofKind === "datadog") {
    const value = type.datadog;
    return <EditItem title="Advanced delivery" obj={value}><EditItem title="Metrics" obj={value.metrics} onSet={() => { value.metrics = EnterpriseP.CollectorExporter_Spec_Datadog_Metrics.create(); onUpdate(); }} onUnset={() => { value.metrics = undefined; onUpdate(); }}>{value.metrics && <div className="grid gap-4 md:grid-cols-2"><TextInput label="Metrics endpoint" description="Optional Datadog metrics intake endpoint override." value={value.metrics.endpoint} onChange={(event) => { value.metrics!.endpoint = event.currentTarget.value; onUpdate(); }} /><Switch label="Resource attributes as tags" description="Export resource attributes as Datadog metric tags." checked={value.metrics.resourceAttributesAsTags} onChange={(event) => { value.metrics!.resourceAttributesAsTags = event.currentTarget.checked; onUpdate(); }} /><Switch label="Instrumentation scope metadata as tags" description="Export instrumentation-scope metadata as Datadog metric tags." checked={value.metrics.instrumentationScopeMetadataAsTags} onChange={(event) => { value.metrics!.instrumentationScopeMetadataAsTags = event.currentTarget.checked; onUpdate(); }} /></div>}</EditItem><EditItem title="Logs" obj={value.logs} onSet={() => { value.logs = EnterpriseP.CollectorExporter_Spec_Datadog_Logs.create(); onUpdate(); }} onUnset={() => { value.logs = undefined; onUpdate(); }}>{value.logs && <div className="grid gap-4 md:grid-cols-2"><TextInput label="Logs endpoint" description="Optional Datadog logs intake endpoint override." value={value.logs.endpoint} onChange={(event) => { value.logs!.endpoint = event.currentTarget.value; onUpdate(); }} /><Switch label="Use compression" description="Compress log payloads before sending them." checked={value.logs.useCompression} onChange={(event) => { value.logs!.useCompression = event.currentTarget.checked; onUpdate(); }} /><NumberInput min={0} label="Compression level" description="Compression level used for Datadog log payloads." value={value.logs.compressionLevel} onChange={(next) => { value.logs!.compressionLevel = Number(next) || 0; onUpdate(); }} /><DurationField label="Batch wait" value={value.logs.batchWait} onChange={(next) => { value.logs!.batchWait = next; onUpdate(); }} /></div>}</EditItem><EditItem title="Host metadata" obj={value.hostMetadata} onSet={() => { value.hostMetadata = EnterpriseP.CollectorExporter_Spec_Datadog_HostMetadata.create(); onUpdate(); }} onUnset={() => { value.hostMetadata = undefined; onUpdate(); }}>{value.hostMetadata && <div className="grid gap-4 md:grid-cols-2"><Switch label="Enabled" description="Enable Datadog host metadata reporting." checked={value.hostMetadata.enabled} onChange={(event) => { value.hostMetadata!.enabled = event.currentTarget.checked; onUpdate(); }} /><DurationField label="Reporter period" value={value.hostMetadata.reporterPeriod} onChange={(next) => { value.hostMetadata!.reporterPeriod = next; onUpdate(); }} /></div>}</EditItem><DurationField label="Hostname detection timeout" value={value.hostnameDetectionTimeout} onChange={(next) => { value.hostnameDetectionTimeout = next; onUpdate(); }} /></EditItem>;
  }

  if (type.oneofKind === "splunk") {
    const value = type.splunk;
    return <EditItem title="Advanced delivery" obj={value}><div className="grid gap-4 md:grid-cols-2"><NumberInput min={0} label="Maximum log content length" description="Maximum log payload size sent to Splunk." value={value.maxContentLengthLogs} onChange={(next) => { value.maxContentLengthLogs = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Maximum metric content length" description="Maximum metric payload size sent to Splunk." value={value.maxContentLengthMetrics} onChange={(next) => { value.maxContentLengthMetrics = Number(next) || 0; onUpdate(); }} /><NumberInput min={0} label="Maximum idle connections" description="Maximum number of idle connections kept in the pool." value={value.maxIdleConns} onChange={(next) => { value.maxIdleConns = Number(next) || 0; onUpdate(); }} /><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /><Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls?.insecureSkipVerify ?? false} onChange={(event) => { value.tls = EnterpriseP.CollectorExporter_Spec_Splunk_TLS.create({ insecureSkipVerify: event.currentTarget.checked }); onUpdate(); }} /></div>{value.tls?.insecureSkipVerify && <Alert color="red" icon={<AlertTriangle size={14} />}>Server certificates will not be verified.</Alert>}</EditItem>;
  }

  if (type.oneofKind === "azureMonitor") {
    const value = type.azureMonitor;
    return <EditItem title="Advanced delivery" obj={value}><div className="grid gap-4 md:grid-cols-2"><DurationField label="Maximum batch interval" value={value.maxBatchInterval} onChange={(next) => { value.maxBatchInterval = next; onUpdate(); }} /><DurationField label="Shutdown timeout" value={value.shutdownTimeout} onChange={(next) => { value.shutdownTimeout = next; onUpdate(); }} /></div></EditItem>;
  }

  if (type.oneofKind === "azureDataExplorer") {
    const value = type.azureDataExplorer;
    return <EditItem title="Advanced delivery" obj={value}><div className="grid gap-4 md:grid-cols-2"><TextInput label="Metrics table mapping" description="Optional mapping that routes metrics to an Azure Data Explorer table." value={value.metricsTableMapping} onChange={(event) => { value.metricsTableMapping = event.currentTarget.value; onUpdate(); }} /><TextInput label="Logs table mapping" description="Optional mapping that routes logs to an Azure Data Explorer table." value={value.logsTableMapping} onChange={(event) => { value.logsTableMapping = event.currentTarget.value; onUpdate(); }} /><DurationField label="Timeout" value={value.timeout} onChange={(next) => { value.timeout = next; onUpdate(); }} /></div></EditItem>;
  }

  return null;
};

const PrometheusTLSSettings = (props: {
  type: EnterpriseP.CollectorExporter_Spec["type"];
  onUpdate: () => void;
}) => {
  if (props.type.oneofKind !== "prometheusRemoteWrite") return null;
  const value = props.type.prometheusRemoteWrite;
  return (
    <EditItem
      title="TLS"
      obj={value.tls}
      onSet={() => {
        value.tls =
          EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TLS.create();
        props.onUpdate();
      }}
      onUnset={() => {
        value.tls = undefined;
        props.onUpdate();
      }}
    >
      {value.tls && (
        <div className="grid gap-4 md:grid-cols-2">
          <Switch label="No TLS" description="Disable TLS and connect over plaintext." checked={value.tls.insecure} onChange={(event) => { value.tls!.insecure = event.currentTarget.checked; props.onUpdate(); }} />
          <Switch label="Skip TLS verification" description="Skip upstream certificate-chain and hostname verification." checked={value.tls.insecureSkipVerify} onChange={(event) => { value.tls!.insecureSkipVerify = event.currentTarget.checked; props.onUpdate(); }} />
          <TextInput label="Server name override" description="Optional hostname used when verifying the upstream certificate." value={value.tls.serverNameOverride} onChange={(event) => { value.tls!.serverNameOverride = event.currentTarget.value; props.onUpdate(); }} />
          <Textarea label="CA certificate PEM" description="Optional PEM-encoded CA used to verify the upstream certificate." autosize minRows={3} value={value.tls.caPEM} onChange={(event) => { value.tls!.caPEM = event.currentTarget.value; props.onUpdate(); }} />
          {value.tls.insecureSkipVerify && <Alert color="red" icon={<AlertTriangle size={14} />}>Server certificates will not be verified.</Alert>}
        </div>
      )}
    </EditItem>
  );
};

const Edit = (props: {
  item: EnterpriseP.CollectorExporter;
  onUpdate: (item: EnterpriseP.CollectorExporter) => void;
}) => {
  const { item, onUpdate } = props;
  const [req, setReq] = React.useState(
    normalizeSecretSelectors(EnterpriseP.CollectorExporter.clone(item)),
  );
  const [init, setInit] = React.useState(
    EnterpriseP.CollectorExporter.clone(req),
  );
  const configurations = React.useRef<
    Partial<Record<string, EnterpriseP.CollectorExporter_Spec["type"]>>
  >({
    [req.spec?.type.oneofKind ?? "otlp"]: structuredClone(
      req.spec?.type ?? { oneofKind: undefined },
    ),
  });
  const authenticationConfigurations = React.useRef<Record<string, unknown>>({});
  const itemKey = item.metadata?.uid || item.apiVersion || item.kind;

  React.useEffect(() => {
    const next = normalizeSecretSelectors(EnterpriseP.CollectorExporter.clone(item));
    if (!next.spec) {
      next.spec = EnterpriseP.CollectorExporter_Spec.create({
        type: {
          oneofKind: "otlp",
          otlp: EnterpriseP.CollectorExporter_Spec_OTLP.create({
            auth: { type: freshBearer() },
          }),
        },
      });
    }
    setReq(next);
    setInit(EnterpriseP.CollectorExporter.clone(next));
    configurations.current = {
      [next.spec.type.oneofKind ?? "otlp"]: structuredClone(next.spec.type),
    };
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.CollectorExporter.clone(req);
    setReq(next);
    onUpdate(EnterpriseP.CollectorExporter.clone(next));
  };

  if (!req.spec) {
    return <></>;
  }

  const grpcInitAuth = (
    kind: "otlp" | "otlpHTTP" | "prometheusRemoteWrite",
  ): GrpcStyleAuth | undefined =>
    match(init.spec?.type)
      .with({ oneofKind: "otlp" }, (t) =>
        kind === "otlp"
          ? (t.otlp.auth as unknown as GrpcStyleAuth | undefined)
          : undefined,
      )
      .with({ oneofKind: "otlpHTTP" }, (t) =>
        kind === "otlpHTTP"
          ? (t.otlpHTTP.auth as unknown as GrpcStyleAuth | undefined)
          : undefined,
      )
      .with({ oneofKind: "prometheusRemoteWrite" }, (t) =>
        kind === "prometheusRemoteWrite"
          ? (t.prometheusRemoteWrite.auth as unknown as
              | GrpcStyleAuth
              | undefined)
          : undefined,
      )
      .otherwise(() => undefined);

  return (
    <div className="space-y-4">
      <div className="rounded-xl border border-slate-200 bg-white p-3">
        <Switch
          label="Disabled"
          description="Disable this exporter so the Collector skips it even when referenced by a pipeline."
          checked={req.spec!.isDisabled}
          onChange={(v) => {
            req.spec!.isDisabled = v.target.checked;
            updateReq();
          }}
        />
      </div>
      <Tabs
        value={req.spec!.type.oneofKind}
        onChange={(v) => {
          const currentKind = req.spec!.type.oneofKind;
          if (currentKind) {
            configurations.current[currentKind] = structuredClone(req.spec!.type);
          }
          if (v && configurations.current[v]) {
            req.spec!.type = structuredClone(configurations.current[v]!);
            updateReq();
            return;
          }
          match(v)
            .with("otlp", () => {
              match(init.spec!.type.oneofKind)
                .with("otlp", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlp",
                    otlp: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_OTLP,
                  };
                });
              updateReq();
            })
            .with("otlpHTTP", () => {
              match(init.spec!.type.oneofKind)
                .with("otlpHTTP", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "otlpHTTP",
                    otlpHTTP: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_OTLPHTTP,
                  };
                });
              updateReq();
            })
            .with("clickhouse", () => {
              match(init.spec!.type.oneofKind)
                .with("clickhouse", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "clickhouse",
                    clickhouse: {
                      password: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Clickhouse,
                  };
                });
              updateReq();
            })
            .with("prometheusRemoteWrite", () => {
              match(init.spec!.type.oneofKind)
                .with("prometheusRemoteWrite", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "prometheusRemoteWrite",
                    prometheusRemoteWrite: {
                      auth: { type: freshBearer() },
                    } as EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite,
                  };
                });
              updateReq();
            })
            .with("elasticsearch", () => {
              match(init.spec!.type.oneofKind)
                .with("elasticsearch", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "elasticsearch",
                    elasticsearch: {
                      auth: {
                        type: {
                          oneofKind: "apiKey",
                          apiKey: freshSecret(),
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Elasticsearch,
                  };
                });
              updateReq();
            })
            .with("logzio", () => {
              match(init.spec!.type.oneofKind)
                .with("logzio", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "logzio",
                    logzio: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Logzio,
                  };
                });
              updateReq();
            })
            .with("influxDB", () => {
              match(init.spec!.type.oneofKind)
                .with("influxDB", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "influxDB",
                    influxDB: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_InfluxDB,
                  };
                });
              updateReq();
            })
            .with("kafka", () => {
              match(init.spec!.type.oneofKind)
                .with("kafka", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "kafka",
                    kafka: {
                      auth: {
                        type: {
                          oneofKind: "sasl",
                          sasl: {
                            username: "",
                            password: freshSecret(),
                            mechanism:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .PLAIN,
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Kafka,
                  };
                });
              updateReq();
            })
            .with("datadog", () => {
              match(init.spec!.type.oneofKind)
                .with("datadog", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "datadog",
                    datadog: {
                      api: {
                        key: freshSecret(),
                        site: "",
                        failOnInvalidKey: false,
                      },
                    } as EnterpriseP.CollectorExporter_Spec_Datadog,
                  };
                });
              updateReq();
            })
            .with("splunk", () => {
              match(init.spec!.type.oneofKind)
                .with("splunk", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "splunk",
                    splunk: {
                      token: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_Splunk,
                  };
                });
              updateReq();
            })
            .with("azureMonitor", () => {
              match(init.spec!.type.oneofKind)
                .with("azureMonitor", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "azureMonitor",
                    azureMonitor: {
                      connectionString: freshSecret(),
                    } as EnterpriseP.CollectorExporter_Spec_AzureMonitor,
                  };
                });
              updateReq();
            })
            .with("azureDataExplorer", () => {
              match(init.spec!.type.oneofKind)
                .with("azureDataExplorer", () => {
                  req.spec!.type = init.spec!.type;
                })
                .otherwise(() => {
                  req.spec!.type = {
                    oneofKind: "azureDataExplorer",
                    azureDataExplorer: {
                      auth: {
                        type: {
                          oneofKind: "servicePrincipal",
                          servicePrincipal: {
                            applicationID: "",
                            applicationKey: freshSecret(),
                            tenantID: "",
                          },
                        },
                      },
                    } as EnterpriseP.CollectorExporter_Spec_AzureDataExplorer,
                  };
                });
              updateReq();
            });
        }}
      >
        <Tabs.List className="flex-nowrap overflow-x-auto rounded-xl border border-slate-200 bg-slate-50/60 p-1">
          <Tabs.Tab value="otlp">OTLP</Tabs.Tab>
          <Tabs.Tab value="otlpHTTP">OTLP HTTP</Tabs.Tab>
          <Tabs.Tab value="clickhouse">ClickHouse</Tabs.Tab>
          <Tabs.Tab value="prometheusRemoteWrite">
            Prometheus Remote Write
          </Tabs.Tab>
          <Tabs.Tab value="elasticsearch">Elasticsearch</Tabs.Tab>
          <Tabs.Tab value="kafka">Kafka</Tabs.Tab>
          <Tabs.Tab value="datadog">Datadog</Tabs.Tab>
          <Tabs.Tab value="logzio">Logz.io</Tabs.Tab>
          <Tabs.Tab value="influxDB">InfluxDB</Tabs.Tab>
          <Tabs.Tab value="splunk">Splunk</Tabs.Tab>
          <Tabs.Tab value="azureMonitor">Azure Monitor</Tabs.Tab>
          <Tabs.Tab value="azureDataExplorer">Azure Data Explorer</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="otlp">
          {match(req.spec!.type)
            .with({ oneofKind: "otlp" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="Address of the upstream OTLP gRPC endpoint."
                    placeholder="otlp-receiver.example.com:8443"
                    value={type.otlp.endpoint}
                    onChange={(v) => {
                      type.otlp.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    description="Compression algorithm used for OTLP payloads."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                    }
                    value={type.otlp.compression}
                    options={[
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .GZIP,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .NONE,
                      },
                      {
                        label: "Snappy",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .SNAPPY,
                      },
                      {
                        label: "Zstd",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLP_Compression
                            .ZSTD,
                      },
                    ]}
                    onChange={(c) => {
                      type.otlp.compression = c;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <Switch
                    label="No TLS"
                    description="Connect to the endpoint without TLS"
                    checked={type.otlp.tls?.insecure ?? false}
                    onChange={(v) => {
                      if (!type.otlp.tls) {
                        type.otlp.tls =
                          EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create();
                      }
                      type.otlp.tls.insecure = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Skip TLS Verify"
                    description="Skip upstream certificate-chain and hostname verification."
                    checked={type.otlp.tls?.insecureSkipVerify ?? false}
                    onChange={(v) => {
                      if (!type.otlp.tls) {
                        type.otlp.tls =
                          EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create();
                      }
                      type.otlp.tls.insecureSkipVerify = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Wait For Ready"
                    description="Wait for the upstream connection instead of failing immediately."
                    checked={type.otlp.waitForReady}
                    onChange={(v) => {
                      type.otlp.waitForReady = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>

                <HeaderList
                  headers={type.otlp.headers}
                  makeHeader={() =>
                    EnterpriseP.CollectorExporter_Spec_OTLP_Header.create()
                  }
                  onUpdate={updateReq}
                />

                <SecretAuthTabs
                  auth={type.otlp.auth as unknown as GrpcStyleAuth | undefined}
                  ensureAuth={() => {
                    if (!type.otlp.auth) {
                      type.otlp.auth =
                        EnterpriseP.CollectorExporter_Spec_OTLP_Auth.create();
                    }
                    return type.otlp.auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("otlp")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="otlpHTTP">
          {match(req.spec!.type)
            .with({ oneofKind: "otlpHTTP" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="Base URL; /v1/logs and /v1/metrics are appended automatically"
                    placeholder="https://otlp-receiver.example.com"
                    value={type.otlpHTTP.endpoint}
                    onChange={(v) => {
                      type.otlpHTTP.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Logs Endpoint"
                    description="Optional URL override for exported logs."
                    placeholder="https://otlp-receiver.example.com/v1/logs"
                    value={type.otlpHTTP.logsEndpoint}
                    onChange={(v) => {
                      type.otlpHTTP.logsEndpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Endpoint"
                    description="Optional URL override for exported metrics."
                    placeholder="https://otlp-receiver.example.com/v1/metrics"
                    value={type.otlpHTTP.metricsEndpoint}
                    onChange={(v) => {
                      type.otlpHTTP.metricsEndpoint = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <EnumSelect
                    label="Encoding"
                    description="Wire encoding used for OTLP HTTP payloads."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                    }
                    value={type.otlpHTTP.encoding}
                    options={[
                      {
                        label: "Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                            .PROTO,
                      },
                      {
                        label: "JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding
                            .JSON,
                      },
                    ]}
                    onChange={(e) => {
                      type.otlpHTTP.encoding = e;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    description="Compression algorithm used for OTLP HTTP payloads."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Compression
                    }
                    value={type.otlpHTTP.compression}
                    options={[
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_OTLPHTTP_Compression.GZIP,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_OTLPHTTP_Compression.NONE,
                      },
                    ]}
                    onChange={(c) => {
                      type.otlpHTTP.compression = c;
                      updateReq();
                    }}
                  />
                </Group>

                <HeaderList
                  headers={type.otlpHTTP.headers}
                  makeHeader={() =>
                    EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Header.create()
                  }
                  onUpdate={updateReq}
                />

                <SecretAuthTabs
                  auth={
                    type.otlpHTTP.auth as unknown as GrpcStyleAuth | undefined
                  }
                  ensureAuth={() => {
                    if (!type.otlpHTTP.auth) {
                      type.otlpHTTP.auth =
                        EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth.create();
                    }
                    return type.otlpHTTP.auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("otlpHTTP")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="clickhouse">
          {match(req.spec!.type)
            .with({ oneofKind: "clickhouse" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="Address of the upstream ClickHouse server."
                    placeholder='"tcp://addr:port", "http://addr:port", "clickhouse://addr:port"'
                    value={type.clickhouse.endpoint}
                    onChange={(v) => {
                      type.clickhouse.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Username"
                    description="Username of the ClickHouse database user."
                    placeholder="clickhouse"
                    value={type.clickhouse.username}
                    onChange={(v) => {
                      type.clickhouse.username = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Database"
                    description="ClickHouse database that receives the telemetry."
                    placeholder="otel"
                    value={type.clickhouse.database}
                    onChange={(v) => {
                      type.clickhouse.database = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Password Secret"
                    sel={type.clickhouse.password}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Logs Table Name"
                    description="Table where exported logs are stored."
                    placeholder="otel_logs"
                    value={type.clickhouse.logsTableName}
                    onChange={(v) => {
                      type.clickhouse.logsTableName = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Cluster Name"
                    description="Optional ClickHouse cluster used for ON CLUSTER table creation."
                    placeholder="cluster"
                    value={type.clickhouse.clusterName}
                    onChange={(v) => {
                      type.clickhouse.clusterName = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Compression"
                    description="Compression algorithm used for exported ClickHouse payloads."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_Clickhouse_Compression
                    }
                    value={type.clickhouse.compression}
                    options={[
                      {
                        label: "LZ4",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.LZ4,
                      },
                      {
                        label: "ZSTD",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.ZSTD,
                      },
                      {
                        label: "Gzip",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.GZIP,
                      },
                      {
                        label: "Deflate",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression
                            .DEFLATE,
                      },
                      {
                        label: "Br",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.BR,
                      },
                      {
                        label: "None",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_Clickhouse_Compression.NONE,
                      },
                    ]}
                    onChange={(c) => {
                      type.clickhouse.compression = c;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Async Insert"
                    description="Buffer inserts on the ClickHouse server before flushing them."
                    checked={type.clickhouse.asyncInsert}
                    onChange={(v) => {
                      type.clickhouse.asyncInsert = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Create Schema"
                    description="Create the database and tables automatically when missing."
                    checked={type.clickhouse.createSchema}
                    onChange={(v) => {
                      type.clickhouse.createSchema = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="JSON"
                    description="Use ClickHouse JSON columns instead of map columns for attributes."
                    checked={type.clickhouse.json}
                    onChange={(v) => {
                      type.clickhouse.json = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="prometheusRemoteWrite">
          {match(req.spec!.type)
            .with({ oneofKind: "prometheusRemoteWrite" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="URL of the Prometheus remote write endpoint."
                    placeholder="https://prometheus.example.com/api/v1/write"
                    value={type.prometheusRemoteWrite.endpoint}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Namespace"
                    description="Prefix prepended to every exported metric name."
                    placeholder="default"
                    value={type.prometheusRemoteWrite.namespace}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.namespace = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group>
                  <Switch
                    label="Disable Scope Info"
                    description="Do not export OpenTelemetry scope labels or the scope-info metric."
                    checked={type.prometheusRemoteWrite.disableScopeInfo}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.disableScopeInfo =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Send Metadata"
                    description="Send metric help, type, and unit metadata with the samples."
                    checked={type.prometheusRemoteWrite.sendMetadata}
                    onChange={(v) => {
                      type.prometheusRemoteWrite.sendMetadata =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>

                <EnumSelect
                  label="Translation Strategy"
                  description="How OpenTelemetry metric and label names are translated to Prometheus names."
                  enumObj={
                    EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                  }
                  value={type.prometheusRemoteWrite.translationStrategy}
                  options={[
                    {
                      label: "Underscore Escaping With Suffixes",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                          .UNDERSCORE_ESCAPING_WITH_SUFFIXES,
                    },
                    {
                      label: "Underscore Escaping Without Suffixes",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy
                          .UNDERSCORE_ESCAPING_WITHOUT_SUFFIXES,
                    },
                  ]}
                  onChange={(s) => {
                    type.prometheusRemoteWrite.translationStrategy = s;
                    updateReq();
                  }}
                />

                <SecretAuthTabs
                  auth={
                    type.prometheusRemoteWrite.auth as unknown as
                      | GrpcStyleAuth
                      | undefined
                  }
                  ensureAuth={() => {
                    if (!type.prometheusRemoteWrite.auth) {
                      type.prometheusRemoteWrite.auth =
                        EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth.create();
                    }
                    return type.prometheusRemoteWrite
                      .auth as unknown as GrpcStyleAuth;
                  }}
                  initAuth={grpcInitAuth("prometheusRemoteWrite")}
                  onUpdate={updateReq}
                />
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="elasticsearch">
          {match(req.spec!.type)
            .with({ oneofKind: "elasticsearch" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    label="Endpoint"
                    description="URL of the upstream Elasticsearch cluster."
                    placeholder="https://es.example.com:9200"
                    value={type.elasticsearch.endpoint}
                    onChange={(v) => {
                      type.elasticsearch.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Cloud ID"
                    description="Elastic Cloud ID; used instead of endpoint addresses."
                    placeholder="my-cloud-id"
                    value={type.elasticsearch.cloudID}
                    onChange={(v) => {
                      type.elasticsearch.cloudID = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <TextInput
                  label="Endpoints"
                  description="Optional Elasticsearch node URLs, separated by commas."
                  placeholder="https://es1:9200, https://es2:9200"
                  value={type.elasticsearch.endpoints.join(",")}
                  onChange={(v) => {
                    type.elasticsearch.endpoints = v.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter((x) => x !== "");
                    updateReq();
                  }}
                />

                <Group grow>
                  <TextInput
                    label="Logs Index"
                    description="Index or data stream where logs are written."
                    placeholder="my-log-index"
                    value={type.elasticsearch.logsIndex}
                    onChange={(v) => {
                      type.elasticsearch.logsIndex = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Index"
                    description="Index or data stream where metrics are written."
                    placeholder="my-metrics-index"
                    value={type.elasticsearch.metricsIndex}
                    onChange={(v) => {
                      type.elasticsearch.metricsIndex = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Pipeline"
                    description="Optional Elasticsearch ingest pipeline for exported documents."
                    placeholder="pipeline"
                    value={type.elasticsearch.pipeline}
                    onChange={(v) => {
                      type.elasticsearch.pipeline = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <EnumSelect
                  label="Compression"
                  description="Compression algorithm used for Elasticsearch payloads."
                  enumObj={
                    EnterpriseP.CollectorExporter_Spec_Elasticsearch_Compression
                  }
                  value={type.elasticsearch.compression}
                  options={[
                    {
                      label: "Gzip",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_Elasticsearch_Compression
                          .GZIP,
                    },
                    {
                      label: "None",
                      value:
                        EnterpriseP
                          .CollectorExporter_Spec_Elasticsearch_Compression
                          .NONE,
                    },
                  ]}
                  onChange={(c) => {
                    type.elasticsearch.compression = c;
                    updateReq();
                  }}
                />

                <Tabs
                  value={type.elasticsearch.auth?.type.oneofKind ?? "apiKey"}
                  onChange={(v) => {
                    if (!type.elasticsearch.auth) {
                      type.elasticsearch.auth =
                        EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth.create();
                    }
                    const currentKind = type.elasticsearch.auth.type.oneofKind;
                    if (currentKind) {
                      authenticationConfigurations.current[`elasticsearch.${currentKind}`] = structuredClone(type.elasticsearch.auth.type);
                    }
                    const cached = authenticationConfigurations.current[`elasticsearch.${v}`] as typeof type.elasticsearch.auth.type | undefined;
                    if (cached) {
                      type.elasticsearch.auth.type = structuredClone(cached);
                      updateReq();
                      return;
                    }
                    match(v)
                      .with("apiKey", () => {
                        type.elasticsearch.auth!.type = {
                          oneofKind: "apiKey",
                          apiKey: freshSecret(),
                        };
                      })
                      .with("basic", () => {
                        type.elasticsearch.auth!.type = {
                          oneofKind: "basic",
                          basic: { user: "", password: freshSecret() },
                        };
                      })
                      .otherwise(() => {});
                    updateReq();
                  }}
                >
                  <SegmentedControl
                    fullWidth
                    value={type.elasticsearch.auth?.type.oneofKind ?? "apiKey"}
                    data={[
                      { label: "API key", value: "apiKey" },
                      { label: "Basic", value: "basic" },
                    ]}
                    onChange={(value) => {
                      if (!type.elasticsearch.auth) type.elasticsearch.auth = EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth.create();
                      const current = type.elasticsearch.auth.type.oneofKind;
                      if (current) authenticationConfigurations.current[`elasticsearch.${current}`] = structuredClone(type.elasticsearch.auth.type);
                      const cached = authenticationConfigurations.current[`elasticsearch.${value}`] as typeof type.elasticsearch.auth.type | undefined;
                      type.elasticsearch.auth.type = cached ? structuredClone(cached) : value === "basic" ? { oneofKind: "basic", basic: { user: "", password: freshSecret() } } : { oneofKind: "apiKey", apiKey: freshSecret() };
                      updateReq();
                    }}
                  />

                  <Tabs.Panel value="apiKey">
                    {match(type.elasticsearch.auth?.type)
                      .with({ oneofKind: "apiKey" }, (a) => (
                        <SecretSelectField
                          label="API Key Secret"
                          sel={a.apiKey}
                          onUpdate={updateReq}
                        />
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="basic">
                    {match(type.elasticsearch.auth?.type)
                      .with({ oneofKind: "basic" }, (a) => (
                        <Group grow>
                          <TextInput
                            label="User"
                            description="Username used for Elasticsearch Basic authentication."
                            placeholder="elastic"
                            value={a.basic.user}
                            onChange={(v) => {
                              a.basic.user = v.target.value;
                              updateReq();
                            }}
                          />
                          <SecretSelectField
                            label="Password Secret"
                            sel={a.basic.password}
                            onUpdate={updateReq}
                          />
                        </Group>
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>
                </Tabs>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="kafka">
          {match(req.spec!.type)
            .with({ oneofKind: "kafka" }, (type) => (
              <>
                <TextInput
                  label="Brokers"
                  description="Kafka broker addresses, separated by commas."
                  placeholder="localhost:9092, localhost:9093"
                  value={type.kafka.brokers.join(",")}
                  onChange={(v) => {
                    type.kafka.brokers = v.target.value
                      .split(",")
                      .map((x) => x.trim())
                      .filter((x) => x !== "");
                    updateReq();
                  }}
                />

                <Group grow>
                  <TextInput
                    label="Protocol Version"
                    description="Kafka protocol version negotiated with the brokers."
                    placeholder="2.1.0"
                    value={type.kafka.protocolVersion}
                    onChange={(v) => {
                      type.kafka.protocolVersion = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Client ID"
                    description="Client identifier reported to the Kafka brokers."
                    placeholder="otel-collector"
                    value={type.kafka.clientID}
                    onChange={(v) => {
                      type.kafka.clientID = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Logs Topic"
                    description="Kafka topic for exported logs; leave empty to disable log export."
                    placeholder="otlp_logs"
                    value={type.kafka.logs?.topic ?? ""}
                    onChange={(v) => {
                      if (!type.kafka.logs) {
                        type.kafka.logs =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.logs.topic = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Logs Encoding"
                    description="Wire encoding used for exported log records."
                    enumObj={EnterpriseP.CollectorExporter_Spec_Kafka_Encoding}
                    value={type.kafka.logs?.encoding ?? 0}
                    options={[
                      {
                        label: "OTLP Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_PROTO,
                      },
                      {
                        label: "OTLP JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_JSON,
                      },
                      {
                        label: "Raw",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding.RAW,
                      },
                    ]}
                    onChange={(e) => {
                      if (!type.kafka.logs) {
                        type.kafka.logs =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.logs.encoding = e;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Metrics Topic"
                    description="Kafka topic for exported metrics; leave empty to disable metric export."
                    placeholder="otlp_metrics"
                    value={type.kafka.metrics?.topic ?? ""}
                    onChange={(v) => {
                      if (!type.kafka.metrics) {
                        type.kafka.metrics =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.metrics.topic = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Metrics Encoding"
                    description="Wire encoding used for exported metric records."
                    enumObj={EnterpriseP.CollectorExporter_Spec_Kafka_Encoding}
                    value={type.kafka.metrics?.encoding ?? 0}
                    options={[
                      {
                        label: "OTLP Proto",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_PROTO,
                      },
                      {
                        label: "OTLP JSON",
                        value:
                          EnterpriseP.CollectorExporter_Spec_Kafka_Encoding
                            .OTLP_JSON,
                      },
                    ]}
                    onChange={(e) => {
                      if (!type.kafka.metrics) {
                        type.kafka.metrics =
                          EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
                      }
                      type.kafka.metrics.encoding = e;
                      updateReq();
                    }}
                  />
                </Group>

                {match(type.kafka.auth?.type)
                  .with({ oneofKind: "sasl" }, (a) => (
                    <Group grow>
                      <TextInput
                        label="SASL Username"
                        description="Username used for Kafka SASL authentication."
                        value={a.sasl.username}
                        onChange={(v) => {
                          a.sasl.username = v.target.value;
                          updateReq();
                        }}
                      />
                      <SecretSelectField
                        label="SASL Password Secret"
                        sel={a.sasl.password}
                        onUpdate={updateReq}
                      />
                      <EnumSelect
                        label="Mechanism"
                        description="SASL mechanism used to authenticate to Kafka."
                        enumObj={
                          EnterpriseP.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                        }
                        value={a.sasl.mechanism}
                        options={[
                          {
                            label: "PLAIN",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .PLAIN,
                          },
                          {
                            label: "SCRAM-SHA-256",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .SCRAM_SHA_256,
                          },
                          {
                            label: "SCRAM-SHA-512",
                            value:
                              EnterpriseP
                                .CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism
                                .SCRAM_SHA_512,
                          },
                        ]}
                        onChange={(m) => {
                          a.sasl.mechanism = m;
                          updateReq();
                        }}
                      />
                    </Group>
                  ))
                  .otherwise(() => (
                    <></>
                  ))}
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="datadog">
          {match(req.spec!.type)
            .with({ oneofKind: "datadog" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Site"
                    description="Datadog site receiving the exported telemetry."
                    placeholder="datadoghq.com"
                    value={type.datadog.api?.site ?? ""}
                    onChange={(v) => {
                      if (!type.datadog.api) {
                        type.datadog.api =
                          EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
                      }
                      type.datadog.api.site = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="API Key Secret"
                    sel={type.datadog.api?.key}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Hostname"
                    description="Hostname attributed to exported Datadog telemetry; detected automatically when empty."
                    value={type.datadog.hostname}
                    onChange={(v) => {
                      type.datadog.hostname = v.target.value;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Fail On Invalid Key"
                    description="Fail Collector startup when Datadog rejects the API key."
                    checked={type.datadog.api?.failOnInvalidKey ?? false}
                    onChange={(v) => {
                      if (!type.datadog.api) {
                        type.datadog.api =
                          EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
                      }
                      type.datadog.api.failOnInvalidKey = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="logzio">
          {match(req.spec!.type)
            .with({ oneofKind: "logzio" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="Logz.io listener URL; derived from Region when empty."
                    placeholder="https://listener.logz.io:8053"
                    value={type.logzio.endpoint}
                    onChange={(v) => {
                      type.logzio.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Region"
                    description="Logz.io account region used to derive the listener URL."
                    placeholder="us"
                    value={type.logzio.region}
                    onChange={(v) => {
                      type.logzio.region = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.logzio.token}
                    onUpdate={updateReq}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="influxDB">
          {match(req.spec!.type)
            .with({ oneofKind: "influxDB" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="URL of the upstream InfluxDB server."
                    placeholder="https://influxdb.example.com:8086"
                    value={type.influxDB.endpoint}
                    onChange={(v) => {
                      type.influxDB.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.influxDB.token}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    required
                    label="Org"
                    description="InfluxDB organization that owns the bucket."
                    value={type.influxDB.org}
                    onChange={(v) => {
                      type.influxDB.org = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    required
                    label="Bucket"
                    description="InfluxDB bucket receiving the telemetry."
                    value={type.influxDB.bucket}
                    onChange={(v) => {
                      type.influxDB.bucket = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <EnumSelect
                    label="Metrics Schema"
                    description="Schema used to map OpenTelemetry metrics into InfluxDB points."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_InfluxDB_MetricsSchema
                    }
                    value={type.influxDB.metricsSchema}
                    options={[
                      {
                        label: "Telegraf Prometheus V1",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_InfluxDB_MetricsSchema
                            .TELEGRAF_PROMETHEUS_V1,
                      },
                      {
                        label: "Telegraf Prometheus V2",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_InfluxDB_MetricsSchema
                            .TELEGRAF_PROMETHEUS_V2,
                      },
                    ]}
                    onChange={(s) => {
                      type.influxDB.metricsSchema = s;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Precision"
                    description="Timestamp precision used for exported InfluxDB points."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                    }
                    value={type.influxDB.precision}
                    options={[
                      {
                        label: "ns",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .NS,
                      },
                      {
                        label: "us",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .US,
                      },
                      {
                        label: "ms",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .MS,
                      },
                      {
                        label: "s",
                        value:
                          EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision
                            .S,
                      },
                    ]}
                    onChange={(p) => {
                      type.influxDB.precision = p;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="splunk">
          {match(req.spec!.type)
            .with({ oneofKind: "splunk" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Endpoint"
                    description="URL of the upstream Splunk HTTP Event Collector endpoint."
                    placeholder="https://splunk.example.com:8088/services/collector"
                    value={type.splunk.endpoint}
                    onChange={(v) => {
                      type.splunk.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <SecretSelectField
                    label="Token Secret"
                    sel={type.splunk.token}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Source"
                    description="Splunk source field assigned to exported events."
                    value={type.splunk.source}
                    onChange={(v) => {
                      type.splunk.source = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Source Type"
                    description="Splunk sourcetype field assigned to exported events."
                    value={type.splunk.sourceType}
                    onChange={(v) => {
                      type.splunk.sourceType = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Index"
                    description="Splunk index receiving the exported events."
                    value={type.splunk.index}
                    onChange={(v) => {
                      type.splunk.index = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="App Name"
                    description="Application name reported to Splunk."
                    value={type.splunk.appName}
                    onChange={(v) => {
                      type.splunk.appName = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="App Version"
                    description="Application version reported to Splunk."
                    value={type.splunk.appVersion}
                    onChange={(v) => {
                      type.splunk.appVersion = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Use Multi Metric Format"
                    description="Combine metrics with shared dimensions into a single event."
                    checked={type.splunk.useMultiMetricFormat}
                    onChange={(v) => {
                      type.splunk.useMultiMetricFormat = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Disable Compression"
                    description="Disable compression for exported Splunk payloads."
                    checked={type.splunk.disableCompression}
                    onChange={(v) => {
                      type.splunk.disableCompression = v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="azureMonitor">
          {match(req.spec!.type)
            .with({ oneofKind: "azureMonitor" }, (type) => (
              <>
                <Group grow>
                  <SecretSelectField
                    label="Connection String Secret"
                    sel={type.azureMonitor.connectionString}
                    onUpdate={updateReq}
                  />
                  <SecretSelectField
                    label="Instrumentation Key Secret"
                    sel={type.azureMonitor.instrumentationKey}
                    onUpdate={updateReq}
                  />
                </Group>
                <Group grow>
                  <TextInput
                    label="Endpoint"
                    description="Optional Azure Monitor ingestion endpoint override."
                    value={type.azureMonitor.endpoint}
                    onChange={(v) => {
                      type.azureMonitor.endpoint = v.target.value;
                      updateReq();
                    }}
                  />
                  <NumberInput
                    label="Max Batch Size"
                    description="Maximum number of telemetry items sent in one batch."
                    value={type.azureMonitor.maxBatchSize}
                    onChange={(v) => {
                      type.azureMonitor.maxBatchSize =
                        typeof v === "number" ? v : Number(v) || 0;
                      updateReq();
                    }}
                  />
                </Group>
                <Group>
                  <Switch
                    label="Custom Events Enabled"
                    description="Export custom-event log records to Application Insights."
                    checked={type.azureMonitor.customEventsEnabled}
                    onChange={(v) => {
                      type.azureMonitor.customEventsEnabled = v.target.checked;
                      updateReq();
                    }}
                  />
                  <Switch
                    label="Exception Events Enabled"
                    description="Export exception attributes as Application Insights telemetry."
                    checked={type.azureMonitor.exceptionEventsEnabled}
                    onChange={(v) => {
                      type.azureMonitor.exceptionEventsEnabled =
                        v.target.checked;
                      updateReq();
                    }}
                  />
                </Group>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>

        <Tabs.Panel value="azureDataExplorer">
          {match(req.spec!.type)
            .with({ oneofKind: "azureDataExplorer" }, (type) => (
              <>
                <Group grow>
                  <TextInput
                    required
                    label="Cluster URI"
                    description="URI of the upstream Azure Data Explorer cluster."
                    placeholder="https://cluster.region.kusto.windows.net"
                    value={type.azureDataExplorer.clusterURI}
                    onChange={(v) => {
                      type.azureDataExplorer.clusterURI = v.target.value;
                      updateReq();
                    }}
                  />
                  <EnumSelect
                    label="Ingestion Type"
                    description="Ingestion mode used to write telemetry to Azure Data Explorer."
                    enumObj={
                      EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_IngestionType
                    }
                    value={type.azureDataExplorer.ingestionType}
                    options={[
                      {
                        label: "Queued",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_AzureDataExplorer_IngestionType
                            .QUEUED,
                      },
                      {
                        label: "Managed",
                        value:
                          EnterpriseP
                            .CollectorExporter_Spec_AzureDataExplorer_IngestionType
                            .MANAGED,
                      },
                    ]}
                    onChange={(t) => {
                      type.azureDataExplorer.ingestionType = t;
                      updateReq();
                    }}
                  />
                </Group>

                <Group grow>
                  <TextInput
                    label="Database"
                    description="Azure Data Explorer database receiving the telemetry."
                    value={type.azureDataExplorer.database}
                    onChange={(v) => {
                      type.azureDataExplorer.database = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Logs Table"
                    description="Azure Data Explorer table receiving logs."
                    value={type.azureDataExplorer.logsTable}
                    onChange={(v) => {
                      type.azureDataExplorer.logsTable = v.target.value;
                      updateReq();
                    }}
                  />
                  <TextInput
                    label="Metrics Table"
                    description="Azure Data Explorer table receiving metrics."
                    value={type.azureDataExplorer.metricsTable}
                    onChange={(v) => {
                      type.azureDataExplorer.metricsTable = v.target.value;
                      updateReq();
                    }}
                  />
                </Group>

                <Tabs
                  value={type.azureDataExplorer.auth?.type.oneofKind ?? "servicePrincipal"}
                  onChange={(v) => {
                    if (!type.azureDataExplorer.auth) {
                      type.azureDataExplorer.auth =
                        EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth.create();
                    }
                    const currentKind = type.azureDataExplorer.auth.type.oneofKind;
                    if (currentKind) {
                      authenticationConfigurations.current[`azureDataExplorer.${currentKind}`] = structuredClone(type.azureDataExplorer.auth.type);
                    }
                    const cached = authenticationConfigurations.current[`azureDataExplorer.${v}`] as typeof type.azureDataExplorer.auth.type | undefined;
                    if (cached) {
                      type.azureDataExplorer.auth.type = structuredClone(cached);
                      updateReq();
                      return;
                    }
                    match(v)
                      .with("servicePrincipal", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "servicePrincipal",
                          servicePrincipal: {
                            applicationID: "",
                            applicationKey: freshSecret(),
                            tenantID: "",
                          },
                        };
                      })
                      .with("managedIdentity", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "managedIdentity",
                          managedIdentity: { id: "" },
                        };
                      })
                      .with("azureDefault", () => {
                        type.azureDataExplorer.auth!.type = {
                          oneofKind: "azureDefault",
                          azureDefault: {},
                        };
                      })
                      .otherwise(() => {});
                    updateReq();
                  }}
                >
                  <SegmentedControl
                    fullWidth
                    value={type.azureDataExplorer.auth?.type.oneofKind ?? "servicePrincipal"}
                    data={[
                      { label: "Service principal", value: "servicePrincipal" },
                      { label: "Managed identity", value: "managedIdentity" },
                      { label: "Azure default", value: "azureDefault" },
                    ]}
                    onChange={(value) => {
                      if (!type.azureDataExplorer.auth) type.azureDataExplorer.auth = EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth.create();
                      const current = type.azureDataExplorer.auth.type.oneofKind;
                      if (current) authenticationConfigurations.current[`azureDataExplorer.${current}`] = structuredClone(type.azureDataExplorer.auth.type);
                      const cached = authenticationConfigurations.current[`azureDataExplorer.${value}`] as typeof type.azureDataExplorer.auth.type | undefined;
                      type.azureDataExplorer.auth.type = cached ? structuredClone(cached) : value === "managedIdentity" ? { oneofKind: "managedIdentity", managedIdentity: { id: "" } } : value === "azureDefault" ? { oneofKind: "azureDefault", azureDefault: {} } : { oneofKind: "servicePrincipal", servicePrincipal: { applicationID: "", applicationKey: freshSecret(), tenantID: "" } };
                      updateReq();
                    }}
                  />

                  <Tabs.Panel value="servicePrincipal">
                    {match(type.azureDataExplorer.auth?.type)
                      .with({ oneofKind: "servicePrincipal" }, (a) => (
                        <Group grow>
                          <TextInput
                            required
                            label="Application ID"
                            description="Client ID of the Entra ID application."
                            value={a.servicePrincipal.applicationID}
                            onChange={(v) => {
                              a.servicePrincipal.applicationID = v.target.value;
                              updateReq();
                            }}
                          />
                          <TextInput
                            required
                            label="Tenant ID"
                            description="Entra ID tenant that owns the application."
                            value={a.servicePrincipal.tenantID}
                            onChange={(v) => {
                              a.servicePrincipal.tenantID = v.target.value;
                              updateReq();
                            }}
                          />
                          <SecretSelectField
                            label="Application Key Secret"
                            sel={a.servicePrincipal.applicationKey}
                            onUpdate={updateReq}
                          />
                        </Group>
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="managedIdentity">
                    {match(type.azureDataExplorer.auth?.type)
                      .with({ oneofKind: "managedIdentity" }, (a) => (
                        <TextInput
                          required
                          label="Managed Identity ID"
                          description='"system" or a client UUID'
                          value={a.managedIdentity.id}
                          onChange={(v) => {
                            a.managedIdentity.id = v.target.value;
                            updateReq();
                          }}
                        />
                      ))
                      .otherwise(() => (
                        <></>
                      ))}
                  </Tabs.Panel>

                  <Tabs.Panel value="azureDefault">
                    <ItemMessage
                      title="Azure Default Credentials"
                      obj={{}}
                    ></ItemMessage>
                  </Tabs.Panel>
                </Tabs>
              </>
            ))
            .otherwise(() => (
              <></>
            ))}
        </Tabs.Panel>
      </Tabs>
      <div className="mt-4">
        <PrometheusTLSSettings type={req.spec.type} onUpdate={updateReq} />
        <AdvancedSettings type={req.spec.type} onUpdate={updateReq} />
      </div>
    </div>
  );
};

export default Edit;
