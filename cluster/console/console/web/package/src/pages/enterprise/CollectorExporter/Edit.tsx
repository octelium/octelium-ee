import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Select, Switch, TagsInput, Textarea, TextInput } from "@mantine/core";
import {
  Database,
  KeyRound,
  Network,
  Power,
  Radio,
  Route,
  Tags,
} from "lucide-react";
import * as React from "react";
import DestinationPicker from "./DestinationPicker";
import {
  Advanced,
  AdvancedGroup,
  Callout,
  ChoiceGroup,
  countSet,
  DurationField,
  EnumField,
  FormSection,
  Grid,
  headerKeyError,
  identError,
  isUUID,
  KeyValueList,
  NumberField,
  SecretField,
  SecretRef,
  secretName,
  toSecret,
  ToggleList,
  ToggleRow,
  urlError,
} from "./fields";
import { ExporterKind } from "./providers";

type ExporterType = EnterpriseP.CollectorExporter_Spec["type"];

type Update = () => void;

type HTTPAuth = {
  type:
    | { oneofKind: "bearer"; bearer: SecretRef }
    | {
        oneofKind: "basic";
        basic: { username: string; password?: SecretRef };
      }
    | {
        oneofKind: "custom";
        custom: { header: string; value?: SecretRef };
      }
    | { oneofKind: undefined };
};

type TLSLike = {
  insecure?: boolean;
  insecureSkipVerify: boolean;
  serverNameOverride?: string;
  caPEM?: string;
};

type TransportMode = "verify" | "skip" | "plaintext";

const createType = (kind: ExporterKind): ExporterType => {
  switch (kind) {
    case "otlpHTTP":
      return {
        oneofKind: "otlpHTTP",
        otlpHTTP: EnterpriseP.CollectorExporter_Spec_OTLPHTTP.create(),
      };
    case "prometheusRemoteWrite":
      return {
        oneofKind: "prometheusRemoteWrite",
        prometheusRemoteWrite:
          EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite.create(),
      };
    case "clickhouse":
      return {
        oneofKind: "clickhouse",
        clickhouse: EnterpriseP.CollectorExporter_Spec_Clickhouse.create({
          createSchema: true,
        }),
      };
    case "elasticsearch":
      return {
        oneofKind: "elasticsearch",
        elasticsearch:
          EnterpriseP.CollectorExporter_Spec_Elasticsearch.create(),
      };
    case "logzio":
      return {
        oneofKind: "logzio",
        logzio: EnterpriseP.CollectorExporter_Spec_Logzio.create({
          token: toSecret(""),
        }),
      };
    case "influxDB":
      return {
        oneofKind: "influxDB",
        influxDB: EnterpriseP.CollectorExporter_Spec_InfluxDB.create(),
      };
    case "kafka":
      return {
        oneofKind: "kafka",
        kafka: EnterpriseP.CollectorExporter_Spec_Kafka.create(),
      };
    case "datadog":
      return {
        oneofKind: "datadog",
        datadog: EnterpriseP.CollectorExporter_Spec_Datadog.create({
          api: { key: toSecret(""), site: "", failOnInvalidKey: false },
        }),
      };
    case "splunk":
      return {
        oneofKind: "splunk",
        splunk: EnterpriseP.CollectorExporter_Spec_Splunk.create({
          token: toSecret(""),
        }),
      };
    case "azureMonitor":
      return {
        oneofKind: "azureMonitor",
        azureMonitor: EnterpriseP.CollectorExporter_Spec_AzureMonitor.create({
          connectionString: toSecret(""),
        }),
      };
    case "azureDataExplorer":
      return {
        oneofKind: "azureDataExplorer",
        azureDataExplorer:
          EnterpriseP.CollectorExporter_Spec_AzureDataExplorer.create({
            auth: {
              type: {
                oneofKind: "servicePrincipal",
                servicePrincipal: {
                  applicationID: "",
                  tenantID: "",
                  applicationKey: toSecret(""),
                },
              },
            },
          }),
      };
    default:
      return {
        oneofKind: "otlp",
        otlp: EnterpriseP.CollectorExporter_Spec_OTLP.create(),
      };
  }
};

const cloneForEdit = (item: EnterpriseP.CollectorExporter) => {
  const next = EnterpriseP.CollectorExporter.clone(item);
  if (!next.spec) {
    next.spec = EnterpriseP.CollectorExporter_Spec.create();
  }
  if (!next.spec.type.oneofKind) {
    next.spec.type = createType("otlp");
  }
  return next;
};

const grpcEndpointError = (value: string) => {
  const endpoint = value.trim();
  if (!endpoint) return undefined;
  if (endpoint.startsWith("unix://")) {
    return endpoint.length > "unix://".length
      ? undefined
      : "Set the unix socket path";
  }
  const rest = endpoint.replace(/^https?:\/\//, "").replace(/^dns:\/\/\/?/, "");
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(rest)) return "Unsupported scheme";
  return /^(\[[^\]]+\]|[^\s:/[\]]+):\d+$/.test(rest)
    ? undefined
    : "Use host:port, e.g. otel-collector.example.com:4317";
};

const clickhouseEndpointError = (value: string) => {
  if (!value) return undefined;
  try {
    const url = new URL(value);
    if (!url.host) return "Include the host, e.g. tcp://clickhouse:9000";
    return ["tcp:", "http:", "https:", "clickhouse:"].includes(url.protocol)
      ? undefined
      : "Use a tcp://, clickhouse://, http:// or https:// address";
  } catch {
    return "Use a tcp://, clickhouse://, http:// or https:// address";
  }
};

const optionalSecret = (name: string) => (name ? toSecret(name) : undefined);

const SecretsHint = () => (
  <p className="text-xs font-normal leading-5 text-slate-500">
    Credentials are read from enterprise Secrets and never stored in the
    exporter. Pick an existing Secret or create one from the picker.
  </p>
);

const transportMode = (
  tls?: TLSLike,
  plaintextByDefault?: boolean,
): TransportMode => {
  if (!tls) return plaintextByDefault ? "plaintext" : "verify";
  if (tls.insecure) return "plaintext";
  if (tls.insecureSkipVerify) return "skip";
  return "verify";
};

const TransportSelect = (props: {
  value: TransportMode;
  defaultMode: TransportMode;
  description: string;
  onChange: (mode: TransportMode) => void;
}) => {
  const label = (mode: TransportMode, text: string) =>
    mode === props.defaultMode ? `${text} (default)` : text;
  return (
    <Select
      label="Transport security"
      description={props.description}
      allowDeselect={false}
      data={[
        {
          value: "verify",
          label: label("verify", "TLS, verify the certificate"),
        },
        { value: "skip", label: label("skip", "TLS, skip verification") },
        { value: "plaintext", label: label("plaintext", "Plaintext, no TLS") },
      ]}
      value={props.value}
      onChange={(value) => value && props.onChange(value as TransportMode)}
    />
  );
};

const applyTransport = <T extends TLSLike>(
  tls: T | undefined,
  mode: TransportMode,
  create: () => T,
): T | undefined => {
  const next = tls ?? create();
  next.insecure = mode === "plaintext";
  next.insecureSkipVerify = mode === "skip";
  return pruneTLS(next);
};

const pruneTLS = <T extends TLSLike>(tls?: T) =>
  tls &&
  !tls.insecure &&
  !tls.insecureSkipVerify &&
  !tls.serverNameOverride &&
  !tls.caPEM
    ? undefined
    : tls;

const SkipVerifyWarning = () => (
  <Callout title="The server certificate is not verified">
    Anyone on the network path can impersonate the destination and read the
    exported telemetry. Prefer adding the CA certificate instead.
  </Callout>
);

const CertificateFields = <T extends TLSLike>(props: {
  tls?: T;
  create: () => T;
  onChange: (tls?: T) => void;
}) => {
  const set = (fn: (tls: T) => void) => {
    const next = props.tls ?? props.create();
    fn(next);
    props.onChange(pruneTLS(next));
  };
  return (
    <Grid>
      <TextInput
        label="Server name override"
        description="Hostname checked against the certificate when it differs from the endpoint."
        placeholder="collector.internal"
        value={props.tls?.serverNameOverride ?? ""}
        onChange={(event) => {
          const value = event.currentTarget.value.trim();
          set((tls) => {
            tls.serverNameOverride = value;
          });
        }}
      />
      <Textarea
        label="CA certificate"
        description="PEM-encoded CA used instead of the system pool."
        placeholder="-----BEGIN CERTIFICATE-----"
        autosize
        minRows={2}
        maxRows={8}
        styles={{
          input: {
            fontFamily: "var(--mantine-font-family-monospace)",
            fontSize: "12px",
          },
        }}
        value={props.tls?.caPEM ?? ""}
        onChange={(event) => {
          const value = event.currentTarget.value;
          set((tls) => {
            tls.caPEM = value;
          });
        }}
      />
    </Grid>
  );
};

type HTTPAuthMethod = "none" | "bearer" | "basic" | "custom";

const HTTPAuthFields = (props: {
  auth?: HTTPAuth;
  onChange: (auth?: HTTPAuth) => void;
}) => {
  const stash = React.useRef<Partial<Record<HTTPAuthMethod, HTTPAuth>>>({});
  const method: HTTPAuthMethod = props.auth?.type.oneofKind ?? "none";
  const update = () => props.onChange(props.auth);

  const change = (next: HTTPAuthMethod) => {
    if (props.auth) stash.current[method] = structuredClone(props.auth);
    if (next === "none") {
      props.onChange(undefined);
      return;
    }
    const cached = stash.current[next];
    if (cached) {
      props.onChange(structuredClone(cached));
      return;
    }
    props.onChange({
      type:
        next === "bearer"
          ? { oneofKind: "bearer", bearer: toSecret("") }
          : next === "basic"
            ? {
                oneofKind: "basic",
                basic: { username: "", password: toSecret("") },
              }
            : {
                oneofKind: "custom",
                custom: { header: "", value: toSecret("") },
              },
    });
  };

  const type = props.auth?.type;

  return (
    <>
      <ChoiceGroup
        label="Method"
        value={method}
        onChange={change}
        options={[
          { value: "none", label: "None", hint: "Open endpoint or mTLS" },
          {
            value: "bearer",
            label: "Bearer token",
            hint: "Authorization: Bearer",
          },
          { value: "basic", label: "Basic", hint: "Username and password" },
          { value: "custom", label: "API key header", hint: "Any header name" },
        ]}
      />
      {type?.oneofKind === "bearer" && (
        <Grid>
          <SecretField
            required
            label="Token"
            description="Secret holding the bearer token."
            value={secretName(type.bearer)}
            onChange={(name) => {
              type.bearer = toSecret(name);
              update();
            }}
          />
        </Grid>
      )}
      {type?.oneofKind === "basic" && (
        <Grid>
          <TextInput
            required
            label="Username"
            description="Sent with the password in the Authorization header."
            placeholder="otel"
            value={type.basic.username}
            error={type.basic.username.trim() ? undefined : "Required"}
            onChange={(event) => {
              type.basic.username = event.currentTarget.value;
              update();
            }}
          />
          <SecretField
            required
            label="Password"
            description="Secret holding the password."
            value={secretName(type.basic.password)}
            onChange={(name) => {
              type.basic.password = toSecret(name);
              update();
            }}
          />
        </Grid>
      )}
      {type?.oneofKind === "custom" && (
        <Grid>
          <TextInput
            required
            label="Header name"
            description="For example x-honeycomb-team or api-key."
            placeholder="X-API-Key"
            value={type.custom.header}
            error={
              !type.custom.header.trim()
                ? "Required"
                : headerKeyError([{ key: type.custom.header, value: "" }])(
                    type.custom.header,
                    0,
                  )
            }
            onChange={(event) => {
              type.custom.header = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            required
            label="Header value"
            description="Secret holding the header value."
            value={secretName(type.custom.value)}
            onChange={(name) => {
              type.custom.value = toSecret(name);
              update();
            }}
          />
        </Grid>
      )}
    </>
  );
};

const AuthSection = (props: {
  children: React.ReactNode;
  description?: string;
}) => (
  <FormSection
    icon={KeyRound}
    title="Authentication"
    description={
      props.description ??
      "How the Collector proves its identity to the destination."
    }
  >
    {props.children}
  </FormSection>
);

const OTLPForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_OTLP;
  update: Update;
}) => {
  const { value, update } = props;
  const C = EnterpriseP.CollectorExporter_Spec_OTLP_Compression;
  const keepalive = () => {
    if (!value.keepalive) {
      value.keepalive =
        EnterpriseP.CollectorExporter_Spec_OTLP_Keepalive.create();
    }
    return value.keepalive;
  };
  const pruneKeepalive = () => {
    const k = value.keepalive;
    if (k && !k.time && !k.timeout && !k.permitWithoutStream) {
      value.keepalive = undefined;
    }
  };
  const mode = transportMode(value.tls);

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The OTLP/gRPC receiver of your backend or of another Collector."
      >
        <TextInput
          required
          label="Endpoint"
          description="host:port of the receiver. dns:/// and unix:// targets are supported too."
          placeholder="otel-collector.example.com:4317"
          value={value.endpoint}
          error={grpcEndpointError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <TransportSelect
            value={mode}
            defaultMode="verify"
            description="Choose plaintext for in-cluster receivers without TLS."
            onChange={(next) => {
              value.tls = applyTransport(value.tls, next, () =>
                EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create(),
              );
              update();
            }}
          />
          <EnumField
            label="Compression"
            description="Gzip when not set."
            value={value.compression}
            options={[
              { label: "Gzip", value: C.GZIP },
              { label: "Zstandard", value: C.ZSTD },
              { label: "Snappy", value: C.SNAPPY },
              { label: "None", value: C.NONE },
            ]}
            onChange={(next) => {
              value.compression = next;
              update();
            }}
          />
        </Grid>
        {mode === "skip" && <SkipVerifyWarning />}
      </FormSection>

      <AuthSection>
        <HTTPAuthFields
          auth={value.auth as unknown as HTTPAuth | undefined}
          onChange={(auth) => {
            value.auth = auth as unknown as
              EnterpriseP.CollectorExporter_Spec_OTLP_Auth | undefined;
            update();
          }}
        />
      </AuthSection>

      <Advanced
        summary="Headers, certificates, timeouts, keepalive and connection tuning."
        count={countSet(
          value.headers,
          value.tls?.serverNameOverride,
          value.tls?.caPEM,
          value.timeout,
          value.waitForReady,
          value.authority,
          value.userAgent,
          value.balancerName,
          value.readBufferSize,
          value.writeBufferSize,
          value.keepalive,
        )}
      >
        <AdvancedGroup title="Headers">
          <KeyValueList
            label="Request headers"
            description="Sent as gRPC metadata with every export, e.g. a tenant ID."
            addLabel="Add header"
            keyPlaceholder="x-scope-orgid"
            valuePlaceholder="value"
            items={value.headers}
            make={() => EnterpriseP.CollectorExporter_Spec_OTLP_Header.create()}
            keyError={headerKeyError(value.headers)}
            onUpdate={update}
          />
        </AdvancedGroup>
        {mode !== "plaintext" && (
          <AdvancedGroup title="Certificates">
            <CertificateFields
              tls={value.tls}
              create={() =>
                EnterpriseP.CollectorExporter_Spec_OTLP_TLS.create()
              }
              onChange={(tls) => {
                value.tls = tls;
                update();
              }}
            />
          </AdvancedGroup>
        )}
        <AdvancedGroup title="Requests">
          <Grid>
            <DurationField
              label="Timeout"
              description="Per export request. 5 seconds when not set."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Wait for the connection"
              description="Block exports until the connection is ready instead of failing fast."
              checked={value.waitForReady}
              onChange={(checked) => {
                value.waitForReady = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Connection tuning">
          <Grid>
            <TextInput
              label="Authority"
              description="Overrides the :authority pseudo-header."
              placeholder="collector.example.com"
              value={value.authority}
              onChange={(event) => {
                value.authority = event.currentTarget.value.trim();
                update();
              }}
            />
            <TextInput
              label="User agent"
              description="Overrides the user agent of the requests."
              placeholder="Default"
              value={value.userAgent}
              onChange={(event) => {
                value.userAgent = event.currentTarget.value;
                update();
              }}
            />
            <Select
              label="Load balancing"
              description="gRPC policy across the resolved addresses."
              placeholder="Default (pick first)"
              clearable
              data={[
                { value: "pick_first", label: "Pick first" },
                { value: "round_robin", label: "Round robin" },
              ]}
              value={value.balancerName || null}
              onChange={(next) => {
                value.balancerName = next ?? "";
                update();
              }}
            />
            <NumberField
              label="Read buffer size"
              description="In bytes, up to 64 MiB."
              max={64 * 1024 * 1024}
              value={value.readBufferSize}
              onChange={(next) => {
                value.readBufferSize = next;
                update();
              }}
            />
            <NumberField
              label="Write buffer size"
              description="In bytes, up to 64 MiB."
              max={64 * 1024 * 1024}
              value={value.writeBufferSize}
              onChange={(next) => {
                value.writeBufferSize = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup
          title="Keepalive"
          description="Pings that keep long-lived connections through proxies and load balancers."
        >
          <Grid>
            <DurationField
              label="Ping after idle"
              description="Inactivity before a ping is sent. At least 1 second."
              value={value.keepalive?.time}
              onChange={(next) => {
                keepalive().time = next;
                pruneKeepalive();
                update();
              }}
            />
            <DurationField
              label="Ping timeout"
              description="Wait for the acknowledgement before closing the connection."
              value={value.keepalive?.timeout}
              onChange={(next) => {
                keepalive().timeout = next;
                pruneKeepalive();
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Ping without active streams"
              checked={value.keepalive?.permitWithoutStream ?? false}
              onChange={(checked) => {
                keepalive().permitWithoutStream = checked;
                pruneKeepalive();
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const OTLPHTTPForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_OTLPHTTP;
  update: Update;
}) => {
  const { value, update } = props;
  const E = EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding;
  const C = EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Compression;
  const mode = transportMode(value.tls);
  const noEndpoint =
    !value.endpoint.trim() &&
    !value.logsEndpoint.trim() &&
    !value.metricsEndpoint.trim();

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The OTLP/HTTP receiver of your backend or of another Collector."
      >
        <TextInput
          required={!value.logsEndpoint && !value.metricsEndpoint}
          type="url"
          label="Endpoint"
          description="Base URL. /v1/logs and /v1/metrics are appended to it."
          placeholder="https://otlp.example.com:4318"
          value={value.endpoint}
          error={urlError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <EnumField
            label="Encoding"
            description="Protobuf when not set."
            value={value.encoding}
            options={[
              { label: "Protobuf", value: E.PROTO },
              { label: "JSON", value: E.JSON },
            ]}
            onChange={(next) => {
              value.encoding = next;
              update();
            }}
          />
          <EnumField
            label="Compression"
            description="Gzip when not set."
            value={value.compression}
            options={[
              { label: "Gzip", value: C.GZIP },
              { label: "None", value: C.NONE },
            ]}
            onChange={(next) => {
              value.compression = next;
              update();
            }}
          />
        </Grid>
        {noEndpoint && (
          <Callout tone="info" title="Set an endpoint">
            Set the base URL, or per-signal URLs under Advanced options.
          </Callout>
        )}
      </FormSection>

      <AuthSection>
        <HTTPAuthFields
          auth={value.auth as unknown as HTTPAuth | undefined}
          onChange={(auth) => {
            value.auth = auth as unknown as
              EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Auth | undefined;
            update();
          }}
        />
      </AuthSection>

      <Advanced
        summary="Per-signal URLs, headers, TLS, timeouts and buffers."
        count={countSet(
          value.logsEndpoint,
          value.metricsEndpoint,
          value.headers,
          value.tls,
          value.timeout,
          value.readBufferSize,
          value.writeBufferSize,
        )}
      >
        <AdvancedGroup
          title="Per-signal URLs"
          description="Full URLs that replace the base endpoint for one signal."
        >
          <Grid>
            <TextInput
              type="url"
              label="Logs URL"
              placeholder="https://otlp.example.com/v1/logs"
              value={value.logsEndpoint}
              error={urlError(value.logsEndpoint)}
              onChange={(event) => {
                value.logsEndpoint = event.currentTarget.value.trim();
                update();
              }}
            />
            <TextInput
              type="url"
              label="Metrics URL"
              placeholder="https://otlp.example.com/v1/metrics"
              value={value.metricsEndpoint}
              error={urlError(value.metricsEndpoint)}
              onChange={(event) => {
                value.metricsEndpoint = event.currentTarget.value.trim();
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup title="Headers">
          <KeyValueList
            label="Request headers"
            description="Sent with every export request."
            addLabel="Add header"
            keyPlaceholder="X-Scope-OrgID"
            valuePlaceholder="value"
            items={value.headers}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Header.create()
            }
            keyError={headerKeyError(value.headers)}
            onUpdate={update}
          />
        </AdvancedGroup>
        <AdvancedGroup title="TLS">
          <Grid>
            <TransportSelect
              value={mode}
              defaultMode="verify"
              description="An http:// endpoint is always plaintext."
              onChange={(next) => {
                value.tls = applyTransport(value.tls, next, () =>
                  EnterpriseP.CollectorExporter_Spec_OTLPHTTP_TLS.create(),
                );
                update();
              }}
            />
          </Grid>
          {mode === "skip" && <SkipVerifyWarning />}
          {mode !== "plaintext" && (
            <CertificateFields
              tls={value.tls}
              create={() =>
                EnterpriseP.CollectorExporter_Spec_OTLPHTTP_TLS.create()
              }
              onChange={(tls) => {
                value.tls = tls;
                update();
              }}
            />
          )}
        </AdvancedGroup>
        <AdvancedGroup title="Requests">
          <Grid cols={3}>
            <DurationField
              label="Timeout"
              description="Per export request."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
            <NumberField
              label="Read buffer size"
              description="In bytes, up to 64 MiB."
              max={64 * 1024 * 1024}
              value={value.readBufferSize}
              onChange={(next) => {
                value.readBufferSize = next;
                update();
              }}
            />
            <NumberField
              label="Write buffer size"
              description="In bytes, up to 64 MiB."
              max={64 * 1024 * 1024}
              value={value.writeBufferSize}
              onChange={(next) => {
                value.writeBufferSize = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const PrometheusForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite;
  update: Update;
}) => {
  const { value, update } = props;
  const T =
    EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TranslationStrategy;
  const mode = transportMode(value.tls);
  const queueEnabled = value.remoteWriteQueue?.enabled ?? true;
  const queue = () => {
    if (!value.remoteWriteQueue) {
      value.remoteWriteQueue =
        EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue.create(
          {
            enabled: true,
            queueSize: 10000,
          },
        );
    }
    return value.remoteWriteQueue;
  };
  const pruneQueue = () => {
    const q = value.remoteWriteQueue;
    if (q?.enabled && q.queueSize === 10000 && !q.numConsumers) {
      value.remoteWriteQueue = undefined;
    }
  };

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The remote write receiver of Prometheus or of a compatible store. Metrics only."
      >
        <TextInput
          required
          type="url"
          label="Endpoint"
          description="Full remote write URL."
          placeholder="https://prometheus.example.com/api/v1/write"
          value={value.endpoint}
          error={urlError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <TextInput
            label="Metric prefix"
            description="Prepended to every metric name, e.g. octelium_."
            placeholder="None"
            value={value.namespace}
            error={
              value.namespace &&
              !/^[A-Za-z_:][A-Za-z0-9_:]*$/.test(value.namespace)
                ? "Use letters, digits, _ and :, not starting with a digit"
                : undefined
            }
            onChange={(event) => {
              value.namespace = event.currentTarget.value.trim();
              update();
            }}
          />
        </Grid>
      </FormSection>

      <AuthSection>
        <HTTPAuthFields
          auth={value.auth as unknown as HTTPAuth | undefined}
          onChange={(auth) => {
            value.auth = auth as unknown as
              | EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Auth
              | undefined;
            update();
          }}
        />
      </AuthSection>

      <FormSection
        icon={Tags}
        title="Labels"
        description="Added to every exported series, e.g. the cluster or the environment."
      >
        <KeyValueList
          label="External labels"
          description="Label names use letters, digits and underscores."
          addLabel="Add label"
          keyPlaceholder="cluster"
          valuePlaceholder="production"
          items={value.externalLabels}
          make={() =>
            EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_ExternalLabel.create()
          }
          keyError={(key, index) => {
            if (!key.trim()) return "Required";
            if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key.trim()))
              return "Invalid label name";
            return value.externalLabels.findIndex(
              (item) => item.key.trim() === key.trim(),
            ) !== index
              ? "Duplicate"
              : undefined;
          }}
          onUpdate={update}
        />
      </FormSection>

      <Advanced
        summary="Metric naming, resource attributes, headers, TLS, batching and queueing."
        count={countSet(
          value.translationStrategy,
          value.sendMetadata,
          value.disableScopeInfo,
          value.targetInfo,
          value.resourceToTelemetryConversion,
          value.headers,
          value.tls,
          value.timeout,
          value.maxBatchSizeBytes,
          value.maxBatchRequestParallelism,
          value.remoteWriteQueue,
        )}
      >
        <AdvancedGroup title="Metric naming">
          <Grid>
            <EnumField
              label="Name translation"
              description="How OpenTelemetry names become Prometheus names."
              value={value.translationStrategy}
              options={[
                {
                  label: "Escape to underscores, add unit suffixes",
                  value: T.UNDERSCORE_ESCAPING_WITH_SUFFIXES,
                },
                {
                  label: "Escape to underscores, no suffixes",
                  value: T.UNDERSCORE_ESCAPING_WITHOUT_SUFFIXES,
                },
                {
                  label:
                    "Keep UTF-8, add suffixes (Remote Write 2.0, not supported yet)",
                  value: T.NO_UTF8_ESCAPING_WITH_SUFFIXES,
                  disabled: true,
                },
                {
                  label: "No translation (Remote Write 2.0, not supported yet)",
                  value: T.NO_TRANSLATION,
                  disabled: true,
                },
              ]}
              onChange={(next) => {
                value.translationStrategy = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Send metric metadata"
              description="Include the help text, type and unit of every metric."
              checked={value.sendMetadata}
              onChange={(checked) => {
                value.sendMetadata = checked;
                update();
              }}
            />
            <ToggleRow
              label="Export target_info"
              description="A series carrying the resource attributes of every source."
              checked={value.targetInfo?.enabled ?? true}
              onChange={(checked) => {
                value.targetInfo = checked
                  ? undefined
                  : EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TargetInfo.create(
                      {
                        enabled: false,
                      },
                    );
                update();
              }}
            />
            <ToggleRow
              label="Export instrumentation scope"
              description="The otel_scope_* labels and the otel_scope_info series."
              checked={!value.disableScopeInfo}
              onChange={(checked) => {
                value.disableScopeInfo = !checked;
                update();
              }}
            />
            <ToggleRow
              label="Resource attributes as labels"
              description="Copy every resource attribute to every series. Raises cardinality."
              checked={value.resourceToTelemetryConversion?.enabled ?? false}
              onChange={(checked) => {
                value.resourceToTelemetryConversion = checked
                  ? EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_ResourceToTelemetryConversion.create(
                      {
                        enabled: true,
                      },
                    )
                  : undefined;
                update();
              }}
            >
              <Switch
                size="sm"
                label="Skip service.* attributes"
                checked={
                  value.resourceToTelemetryConversion
                    ?.excludeServiceAttributes ?? false
                }
                onChange={(event) => {
                  value.resourceToTelemetryConversion!.excludeServiceAttributes =
                    event.currentTarget.checked;
                  update();
                }}
              />
            </ToggleRow>
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Headers">
          <KeyValueList
            label="Request headers"
            description="Sent with every remote write request, e.g. X-Scope-OrgID."
            addLabel="Add header"
            keyPlaceholder="X-Scope-OrgID"
            valuePlaceholder="value"
            items={value.headers}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_Header.create()
            }
            keyError={headerKeyError(value.headers)}
            onUpdate={update}
          />
        </AdvancedGroup>
        <AdvancedGroup title="TLS">
          <Grid>
            <TransportSelect
              value={mode}
              defaultMode="verify"
              description="An http:// endpoint is always plaintext."
              onChange={(next) => {
                value.tls = applyTransport(value.tls, next, () =>
                  EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TLS.create(),
                );
                update();
              }}
            />
          </Grid>
          {mode === "skip" && <SkipVerifyWarning />}
          {mode !== "plaintext" && (
            <CertificateFields
              tls={value.tls}
              create={() =>
                EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_TLS.create()
              }
              onChange={(tls) => {
                value.tls = tls;
                update();
              }}
            />
          )}
        </AdvancedGroup>
        <AdvancedGroup title="Batching and queueing">
          <Grid cols={3}>
            <DurationField
              label="Timeout"
              description="Per remote write request."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
            <NumberField
              label="Maximum request size"
              description="Bigger batches are split. Up to 512 MiB."
              suffix=" bytes"
              max={512 * 1024 * 1024}
              value={value.maxBatchSizeBytes}
              onChange={(next) => {
                value.maxBatchSizeBytes = next;
                update();
              }}
            />
            <NumberField
              label="Parallel requests"
              description="Per batch, up to 1024."
              max={1024}
              value={value.maxBatchRequestParallelism}
              onChange={(next) => {
                value.maxBatchRequestParallelism = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Sending queue"
              description="Buffer batches in memory so short outages do not drop metrics."
              checked={queueEnabled}
              onChange={(checked) => {
                value.remoteWriteQueue = checked
                  ? undefined
                  : EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite_RemoteWriteQueue.create(
                      {
                        enabled: false,
                      },
                    );
                update();
              }}
            >
              <Grid>
                <NumberField
                  label="Queue size"
                  description="Batches kept in memory. 10,000 by default."
                  min={1}
                  max={10_000_000}
                  value={value.remoteWriteQueue?.queueSize ?? 0}
                  onChange={(next) => {
                    queue().queueSize = next || 10000;
                    pruneQueue();
                    update();
                  }}
                />
                <NumberField
                  label="Consumers"
                  description="Workers sending from the queue. 5 by default."
                  max={1024}
                  value={value.remoteWriteQueue?.numConsumers ?? 0}
                  onChange={(next) => {
                    queue().numConsumers = next;
                    pruneQueue();
                    update();
                  }}
                />
              </Grid>
            </ToggleRow>
          </ToggleList>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const ClickhouseForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Clickhouse;
  update: Update;
}) => {
  const { value, update } = props;
  const C = EnterpriseP.CollectorExporter_Spec_Clickhouse_Compression;
  const mode = transportMode(value.tls);
  const metricsTables = () => {
    if (!value.metricsTables) {
      value.metricsTables =
        EnterpriseP.CollectorExporter_Spec_Clickhouse_MetricsTables.create();
    }
    return value.metricsTables;
  };
  const tableEngine = () => {
    if (!value.tableEngine) {
      value.tableEngine =
        EnterpriseP.CollectorExporter_Spec_Clickhouse_TableEngine.create();
    }
    return value.tableEngine;
  };
  const pruneTables = () => {
    const t = value.metricsTables;
    if (
      t &&
      !t.gauge &&
      !t.sum &&
      !t.summary &&
      !t.histogram &&
      !t.exponentialHistogram
    ) {
      value.metricsTables = undefined;
    }
    if (
      value.tableEngine &&
      !value.tableEngine.name &&
      !value.tableEngine.params
    ) {
      value.tableEngine = undefined;
    }
  };
  const tableFields = [
    { key: "gauge", label: "Gauge", placeholder: "otel_metrics_gauge" },
    { key: "sum", label: "Sum", placeholder: "otel_metrics_sum" },
    {
      key: "histogram",
      label: "Histogram",
      placeholder: "otel_metrics_histogram",
    },
    {
      key: "exponentialHistogram",
      label: "Exponential histogram",
      placeholder: "otel_metrics_exponential_histogram",
    },
    { key: "summary", label: "Summary", placeholder: "otel_metrics_summary" },
  ] as const;

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The ClickHouse server or cluster that stores the telemetry."
      >
        <TextInput
          required
          label="Endpoint"
          description="Native protocol (tcp://, clickhouse://) or HTTP interface (http://, https://)."
          placeholder="tcp://clickhouse.example.com:9000"
          value={value.endpoint}
          error={clickhouseEndpointError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid cols={3}>
          <TextInput
            label="Database"
            description="default when not set."
            placeholder="default"
            value={value.database}
            error={identError(value.database)}
            onChange={(event) => {
              value.database = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            label="Username"
            description="default when not set."
            placeholder="default"
            value={value.username}
            onChange={(event) => {
              value.username = event.currentTarget.value.trim();
              update();
            }}
          />
          <SecretField
            label="Password"
            description="Secret holding the password. Optional."
            value={secretName(value.password)}
            onChange={(name) => {
              value.password = optionalSecret(name);
              update();
            }}
          />
        </Grid>
      </FormSection>

      <FormSection
        icon={Database}
        title="Tables"
        description="Where rows are written and how long they are kept."
      >
        <Grid>
          <TextInput
            label="Logs table"
            description="otel_logs when not set."
            placeholder="otel_logs"
            value={value.logsTableName}
            error={identError(value.logsTableName)}
            onChange={(event) => {
              value.logsTableName = event.currentTarget.value.trim();
              update();
            }}
          />
          <DurationField
            label="Retention"
            description="Rows older than this are deleted. Kept forever when not set."
            value={value.ttl}
            onChange={(next) => {
              value.ttl = next;
              update();
            }}
          />
        </Grid>
        <ToggleList>
          <ToggleRow
            label="Create the database and tables"
            description="Create them when they do not exist yet. Turn off when you manage the schema yourself."
            checked={value.createSchema}
            onChange={(checked) => {
              value.createSchema = checked;
              update();
            }}
          />
        </ToggleList>
      </FormSection>

      <Advanced
        summary="Metric tables, table engine, cluster, compression, TLS and connection parameters."
        count={countSet(
          value.metricsTables,
          value.tableEngine,
          value.clusterName,
          value.compression,
          value.asyncInsert,
          value.json,
          value.timeout,
          value.tls,
          value.connectionParams,
        )}
      >
        <AdvancedGroup
          title="Metric tables"
          description="One table per metric type."
        >
          <Grid cols={3}>
            {tableFields.map((field) => (
              <TextInput
                key={field.key}
                label={field.label}
                placeholder={field.placeholder}
                value={value.metricsTables?.[field.key] ?? ""}
                error={identError(value.metricsTables?.[field.key] ?? "")}
                onChange={(event) => {
                  metricsTables()[field.key] = event.currentTarget.value.trim();
                  pruneTables();
                  update();
                }}
              />
            ))}
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup title="Schema">
          <Grid cols={3}>
            <TextInput
              label="Cluster"
              description="Creates the tables ON CLUSTER."
              placeholder="None"
              value={value.clusterName}
              error={identError(value.clusterName)}
              onChange={(event) => {
                value.clusterName = event.currentTarget.value.trim();
                update();
              }}
            />
            <TextInput
              label="Table engine"
              description="MergeTree when not set."
              placeholder="MergeTree"
              value={value.tableEngine?.name ?? ""}
              error={identError(value.tableEngine?.name ?? "")}
              onChange={(event) => {
                tableEngine().name = event.currentTarget.value.trim();
                pruneTables();
                update();
              }}
            />
            <TextInput
              label="Engine parameters"
              description="Passed to the table engine."
              placeholder="None"
              value={value.tableEngine?.params ?? ""}
              onChange={(event) => {
                tableEngine().params = event.currentTarget.value;
                pruneTables();
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="JSON attribute columns"
              description="Use the JSON column type instead of maps for attributes."
              checked={value.json}
              onChange={(checked) => {
                value.json = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Inserts">
          <Grid>
            <EnumField
              label="Compression"
              description="LZ4 when not set."
              value={value.compression}
              options={[
                { label: "LZ4", value: C.LZ4 },
                { label: "Zstandard", value: C.ZSTD },
                { label: "Gzip", value: C.GZIP },
                { label: "Deflate", value: C.DEFLATE },
                { label: "Brotli", value: C.BR },
                { label: "None", value: C.NONE },
              ]}
              onChange={(next) => {
                value.compression = next;
                update();
              }}
            />
            <DurationField
              label="Timeout"
              description="Per insert."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Asynchronous inserts"
              description="The server buffers rows and flushes them in the background."
              checked={value.asyncInsert}
              onChange={(checked) => {
                value.asyncInsert = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="TLS">
          <Grid>
            <TransportSelect
              value={mode}
              defaultMode="verify"
              description="Applies to secure endpoints, such as https:// or port 9440."
              onChange={(next) => {
                value.tls = applyTransport(value.tls, next, () =>
                  EnterpriseP.CollectorExporter_Spec_Clickhouse_TLS.create(),
                );
                update();
              }}
            />
          </Grid>
          {mode === "skip" && <SkipVerifyWarning />}
          {mode !== "plaintext" && (
            <CertificateFields
              tls={value.tls}
              create={() =>
                EnterpriseP.CollectorExporter_Spec_Clickhouse_TLS.create()
              }
              onChange={(tls) => {
                value.tls = tls;
                update();
              }}
            />
          )}
        </AdvancedGroup>
        <AdvancedGroup title="Connection parameters">
          <KeyValueList
            label="Parameters"
            description="Extra connection string options. Set the username and password above instead."
            addLabel="Add parameter"
            keyPlaceholder="dial_timeout"
            valuePlaceholder="10s"
            items={value.connectionParams}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_Clickhouse_ConnectionParam.create()
            }
            keyError={(key, index) => {
              const trimmed = key.trim().toLowerCase();
              if (!trimmed) return "Required";
              if (trimmed === "username" || trimmed === "password")
                return "Set it above";
              return value.connectionParams.findIndex(
                (item) => item.key.trim().toLowerCase() === trimmed,
              ) !== index
                ? "Duplicate"
                : undefined;
            }}
            onUpdate={update}
          />
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

type ElasticMode = "url" | "nodes" | "cloud";

const ElasticsearchForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Elasticsearch;
  update: Update;
}) => {
  const { value, update } = props;
  const C = EnterpriseP.CollectorExporter_Spec_Elasticsearch_Compression;
  const [mode, setMode] = React.useState<ElasticMode>(() =>
    value.cloudID ? "cloud" : value.endpoints.length > 0 ? "nodes" : "url",
  );
  const stash = React.useRef<{
    endpoint: string;
    endpoints: string[];
    cloudID: string;
  }>({
    endpoint: "",
    endpoints: [],
    cloudID: "",
  });
  const authStash = React.useRef<
    Partial<
      Record<string, EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth>
    >
  >({});
  const auth = value.auth?.type;
  const authMethod = auth?.oneofKind ?? "none";

  const changeMode = (next: ElasticMode) => {
    stash.current = {
      endpoint: value.endpoint || stash.current.endpoint,
      endpoints: value.endpoints.length
        ? value.endpoints
        : stash.current.endpoints,
      cloudID: value.cloudID || stash.current.cloudID,
    };
    value.endpoint = next === "url" ? stash.current.endpoint : "";
    value.endpoints = next === "nodes" ? [...stash.current.endpoints] : [];
    value.cloudID = next === "cloud" ? stash.current.cloudID : "";
    setMode(next);
    update();
  };

  const changeAuth = (next: string) => {
    if (value.auth) authStash.current[authMethod] = structuredClone(value.auth);
    if (next === "none") {
      value.auth = undefined;
    } else {
      value.auth =
        authStash.current[next] ??
        EnterpriseP.CollectorExporter_Spec_Elasticsearch_Auth.create({
          type:
            next === "basic"
              ? {
                  oneofKind: "basic",
                  basic: { user: "", password: toSecret("") },
                }
              : { oneofKind: "apiKey", apiKey: toSecret("") },
        });
    }
    update();
  };

  const indexError = (index: string) => {
    if (!index) return undefined;
    if (/[\\/*?"<>| ,#]/.test(index)) return "Contains invalid characters";
    if (/^[-_+]/.test(index)) return "Must not start with -, _ or +";
    return undefined;
  };

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="A self-managed cluster or an Elastic Cloud deployment."
      >
        <ChoiceGroup
          label="Connect with"
          value={mode}
          onChange={changeMode}
          options={[
            {
              value: "url",
              label: "Cluster URL",
              hint: "One address, e.g. a load balancer",
            },
            {
              value: "nodes",
              label: "Node URLs",
              hint: "Spread requests over nodes",
            },
            {
              value: "cloud",
              label: "Elastic Cloud",
              hint: "Deployment Cloud ID",
            },
          ]}
        />
        {mode === "url" && (
          <TextInput
            required
            type="url"
            label="URL"
            placeholder="https://elasticsearch.example.com:9200"
            value={value.endpoint}
            error={urlError(value.endpoint)}
            onChange={(event) => {
              value.endpoint = event.currentTarget.value.trim();
              update();
            }}
          />
        )}
        {mode === "nodes" && (
          <TagsInput
            required
            label="Node URLs"
            description="Press Enter or comma after each URL. Up to 32."
            placeholder={
              value.endpoints.length ? "" : "https://es-1.example.com:9200"
            }
            splitChars={[",", " "]}
            acceptValueOnBlur
            maxTags={32}
            value={value.endpoints}
            error={value.endpoints.map((item) => urlError(item)).find(Boolean)}
            onChange={(next) => {
              value.endpoints = next;
              update();
            }}
          />
        )}
        {mode === "cloud" && (
          <TextInput
            required
            label="Cloud ID"
            description="From the deployment page of the Elastic Cloud console."
            placeholder="my-deployment:ZXUtd2VzdC0x..."
            value={value.cloudID}
            onChange={(event) => {
              value.cloudID = event.currentTarget.value.trim();
              update();
            }}
          />
        )}
      </FormSection>

      <AuthSection>
        <ChoiceGroup
          label="Method"
          value={authMethod}
          onChange={changeAuth}
          options={[
            { value: "none", label: "None", hint: "Security disabled" },
            { value: "apiKey", label: "API key", hint: "Recommended" },
            { value: "basic", label: "Basic", hint: "Username and password" },
          ]}
        />
        {auth?.oneofKind === "apiKey" && (
          <Grid>
            <SecretField
              required
              label="API key"
              description="Secret holding the encoded API key."
              value={secretName(auth.apiKey)}
              onChange={(name) => {
                auth.apiKey = toSecret(name);
                update();
              }}
            />
          </Grid>
        )}
        {auth?.oneofKind === "basic" && (
          <Grid>
            <TextInput
              required
              label="Username"
              description="An Elasticsearch user allowed to write the indexes."
              placeholder="elastic"
              value={auth.basic.user}
              error={auth.basic.user.trim() ? undefined : "Required"}
              onChange={(event) => {
                auth.basic.user = event.currentTarget.value;
                update();
              }}
            />
            <SecretField
              required
              label="Password"
              description="Secret holding the password."
              value={secretName(auth.basic.password)}
              onChange={(name) => {
                auth.basic.password = toSecret(name);
                update();
              }}
            />
          </Grid>
        )}
      </AuthSection>

      <FormSection
        icon={Route}
        title="Indexes"
        description="Leave empty to use the default data streams."
      >
        <Grid cols={3}>
          <TextInput
            label="Logs index"
            placeholder="logs-generic-default"
            value={value.logsIndex}
            error={indexError(value.logsIndex)}
            onChange={(event) => {
              value.logsIndex = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            label="Metrics index"
            placeholder="metrics-generic-default"
            value={value.metricsIndex}
            error={indexError(value.metricsIndex)}
            onChange={(event) => {
              value.metricsIndex = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            label="Ingest pipeline"
            placeholder="None"
            value={value.pipeline}
            onChange={(event) => {
              value.pipeline = event.currentTarget.value.trim();
              update();
            }}
          />
        </Grid>
      </FormSection>

      <Advanced
        summary="Headers, compression, timeout and certificate verification."
        count={countSet(
          value.headers,
          value.compression,
          value.timeout,
          value.tls?.insecureSkipVerify,
        )}
      >
        <AdvancedGroup title="Headers">
          <KeyValueList
            label="Request headers"
            description="Sent with every bulk request."
            addLabel="Add header"
            keyPlaceholder="X-Elastic-Product-Origin"
            valuePlaceholder="value"
            items={value.headers}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_Elasticsearch_Header.create()
            }
            keyError={headerKeyError(value.headers)}
            onUpdate={update}
          />
        </AdvancedGroup>
        <AdvancedGroup title="Requests">
          <Grid>
            <EnumField
              label="Compression"
              description="Gzip when not set."
              value={value.compression}
              options={[
                { label: "Gzip", value: C.GZIP },
                { label: "None", value: C.NONE },
              ]}
              onChange={(next) => {
                value.compression = next;
                update();
              }}
            />
            <DurationField
              label="Timeout"
              description="Per bulk request."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Skip certificate verification"
              description="Accept any certificate presented by the cluster."
              checked={value.tls?.insecureSkipVerify ?? false}
              onChange={(checked) => {
                value.tls = checked
                  ? EnterpriseP.CollectorExporter_Spec_Elasticsearch_TLS.create(
                      { insecureSkipVerify: true },
                    )
                  : undefined;
                update();
              }}
            />
          </ToggleList>
          {value.tls?.insecureSkipVerify && <SkipVerifyWarning />}
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const KafkaForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Kafka;
  update: Update;
}) => {
  const { value, update } = props;
  const Enc = EnterpriseP.CollectorExporter_Spec_Kafka_Encoding;
  const PC = EnterpriseP.CollectorExporter_Spec_Kafka_ProducerCompression;
  const M = EnterpriseP.CollectorExporter_Spec_Kafka_Auth_SASL_Mechanism;
  const saslStash = React.useRef<
    EnterpriseP.CollectorExporter_Spec_Kafka_Auth | undefined
  >(undefined);
  const mode = transportMode(value.tls, true);
  const sasl =
    value.auth?.type.oneofKind === "sasl" ? value.auth.type.sasl : undefined;

  const signal = (key: "logs" | "metrics") => {
    if (!value[key])
      value[key] = EnterpriseP.CollectorExporter_Spec_Kafka_Signal.create();
    return value[key]!;
  };
  const pruneSignal = (key: "logs" | "metrics") => {
    const s = value[key];
    if (s && !s.topic && !s.encoding) value[key] = undefined;
  };
  const producer = () => {
    if (!value.producer) {
      value.producer = EnterpriseP.CollectorExporter_Spec_Kafka_Producer.create(
        { requiredAcks: 1 },
      );
    }
    return value.producer;
  };
  const pruneProducer = () => {
    const p = value.producer;
    if (
      p &&
      !p.maxMessageBytes &&
      !p.flushMaxMessages &&
      !p.compression &&
      !p.allowAutoTopicCreation &&
      !p.linger &&
      p.requiredAcks === 1
    ) {
      value.producer = undefined;
    }
  };

  const signalFields = (
    key: "logs" | "metrics",
    label: string,
    placeholder: string,
  ) => (
    <Grid>
      <TextInput
        label={`${label} topic`}
        description={`${placeholder} when not set.`}
        placeholder={placeholder}
        value={value[key]?.topic ?? ""}
        onChange={(event) => {
          signal(key).topic = event.currentTarget.value.trim();
          pruneSignal(key);
          update();
        }}
      />
      <EnumField
        label={`${label} encoding`}
        description="OTLP Protobuf when not set."
        value={value[key]?.encoding ?? 0}
        options={[
          { label: "OTLP Protobuf", value: Enc.OTLP_PROTO },
          { label: "OTLP JSON", value: Enc.OTLP_JSON },
          ...(key === "logs"
            ? [{ label: "Raw log body", value: Enc.RAW }]
            : []),
        ]}
        onChange={(next) => {
          signal(key).encoding = next;
          pruneSignal(key);
          update();
        }}
      />
    </Grid>
  );

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The bootstrap brokers of the Kafka cluster."
      >
        <TagsInput
          required
          label="Brokers"
          description="host:port of one or more brokers. Press Enter or comma after each."
          placeholder={value.brokers.length ? "" : "kafka-0.example.com:9092"}
          splitChars={[",", " "]}
          acceptValueOnBlur
          maxTags={64}
          value={value.brokers}
          error={
            value.brokers.length === 0 ? "Add at least one broker" : undefined
          }
          onChange={(next) => {
            value.brokers = next;
            update();
          }}
        />
        <Grid>
          <TransportSelect
            value={mode}
            defaultMode="plaintext"
            description="Most managed Kafka services require TLS."
            onChange={(next) => {
              value.tls =
                next === "plaintext"
                  ? undefined
                  : EnterpriseP.CollectorExporter_Spec_Kafka_TLS.create({
                      insecureSkipVerify: next === "skip",
                    });
              update();
            }}
          />
          <TextInput
            label="Protocol version"
            description="Kafka version the brokers speak. 2.1.0 when not set."
            placeholder="2.1.0"
            value={value.protocolVersion}
            onChange={(event) => {
              value.protocolVersion = event.currentTarget.value.trim();
              update();
            }}
          />
        </Grid>
        {mode === "skip" && <SkipVerifyWarning />}
      </FormSection>

      <AuthSection>
        <ChoiceGroup
          label="Method"
          value={sasl ? "sasl" : "none"}
          onChange={(next) => {
            if (next === "none") {
              saslStash.current = value.auth;
              value.auth = undefined;
            } else {
              value.auth =
                saslStash.current ??
                EnterpriseP.CollectorExporter_Spec_Kafka_Auth.create({
                  type: {
                    oneofKind: "sasl",
                    sasl: {
                      username: "",
                      password: toSecret(""),
                      mechanism: M.SCRAM_SHA_512,
                    },
                  },
                });
            }
            update();
          }}
          options={[
            { value: "none", label: "None", hint: "Unauthenticated listener" },
            { value: "sasl", label: "SASL", hint: "PLAIN or SCRAM" },
          ]}
        />
        {sasl && (
          <Grid cols={3}>
            <TextInput
              required
              label="Username"
              description="The SASL user of the Collector."
              value={sasl.username}
              error={sasl.username.trim() ? undefined : "Required"}
              onChange={(event) => {
                sasl.username = event.currentTarget.value;
                update();
              }}
            />
            <SecretField
              required
              label="Password"
              description="Secret holding the password."
              value={secretName(sasl.password)}
              onChange={(name) => {
                sasl.password = toSecret(name);
                update();
              }}
            />
            <EnumField
              label="Mechanism"
              description="PLAIN when not set."
              value={sasl.mechanism}
              options={[
                { label: "SCRAM-SHA-512", value: M.SCRAM_SHA_512 },
                { label: "SCRAM-SHA-256", value: M.SCRAM_SHA_256 },
                { label: "PLAIN", value: M.PLAIN },
              ]}
              onChange={(next) => {
                sasl.mechanism = next;
                update();
              }}
            />
          </Grid>
        )}
        {sasl?.mechanism === M.PLAIN && mode === "plaintext" && (
          <Callout title="The password is sent in clear text">
            SASL/PLAIN over a plaintext connection exposes the password. Enable
            TLS or use SCRAM.
          </Callout>
        )}
      </AuthSection>

      <FormSection
        icon={Route}
        title="Topics"
        description="Where each signal is produced and how its records are encoded."
      >
        {signalFields("logs", "Logs", "otlp_logs")}
        {signalFields("metrics", "Metrics", "otlp_metrics")}
      </FormSection>

      <Advanced
        summary="Producer, record headers, partitioning, client ID and timeouts."
        count={countSet(
          value.producer,
          value.recordHeaders,
          value.partitionLogsByResourceAttributes,
          value.partitionMetricsByResourceAttributes,
          value.clientID,
          value.timeout,
          value.connIdleTimeout,
        )}
      >
        <AdvancedGroup title="Producer">
          <Grid cols={3}>
            <Select
              label="Acknowledgements"
              description="Broker confirmations required per write."
              allowDeselect={false}
              data={[
                { value: "1", label: "Leader only" },
                { value: "-1", label: "All in-sync replicas" },
                { value: "0", label: "None" },
              ]}
              value={String(value.producer?.requiredAcks ?? 1)}
              onChange={(next) => {
                producer().requiredAcks = Number(next ?? 1);
                pruneProducer();
                update();
              }}
            />
            <EnumField
              label="Compression"
              description="None when not set."
              value={value.producer?.compression ?? 0}
              options={[
                { label: "None", value: PC.NONE },
                { label: "Gzip", value: PC.GZIP },
                { label: "Snappy", value: PC.SNAPPY },
                { label: "LZ4", value: PC.LZ4 },
                { label: "Zstandard", value: PC.ZSTD },
              ]}
              onChange={(next) => {
                producer().compression = next;
                pruneProducer();
                update();
              }}
            />
            <NumberField
              label="Maximum message size"
              description="Up to 100 MiB."
              suffix=" bytes"
              max={100 * 1024 * 1024}
              value={value.producer?.maxMessageBytes ?? 0}
              onChange={(next) => {
                producer().maxMessageBytes = next;
                pruneProducer();
                update();
              }}
            />
            <NumberField
              label="Flush after messages"
              description="Records buffered before a flush."
              value={value.producer?.flushMaxMessages ?? 0}
              onChange={(next) => {
                producer().flushMaxMessages = next;
                pruneProducer();
                update();
              }}
            />
            <DurationField
              label="Linger"
              description="Wait to batch records, up to 1 minute."
              value={value.producer?.linger}
              onChange={(next) => {
                producer().linger = next;
                pruneProducer();
                update();
              }}
            />
          </Grid>
          {value.producer?.requiredAcks === 0 && (
            <Callout title="Writes are not confirmed">
              Records are lost silently when a broker fails.
            </Callout>
          )}
          <ToggleList>
            <ToggleRow
              label="Create missing topics"
              description="Let the brokers create topics that do not exist yet."
              checked={value.producer?.allowAutoTopicCreation ?? false}
              onChange={(checked) => {
                producer().allowAutoTopicCreation = checked;
                pruneProducer();
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Partitioning">
          <ToggleList>
            <ToggleRow
              label="Partition logs by resource"
              description="Logs of the same resource land on the same partition."
              checked={value.partitionLogsByResourceAttributes}
              onChange={(checked) => {
                value.partitionLogsByResourceAttributes = checked;
                update();
              }}
            />
            <ToggleRow
              label="Partition metrics by resource"
              description="Metrics of the same resource land on the same partition."
              checked={value.partitionMetricsByResourceAttributes}
              onChange={(checked) => {
                value.partitionMetricsByResourceAttributes = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Record headers">
          <KeyValueList
            label="Headers"
            description="Added to every produced record."
            addLabel="Add header"
            keyPlaceholder="source"
            valuePlaceholder="octelium"
            items={value.recordHeaders}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_Kafka_Header.create()
            }
            keyError={headerKeyError(value.recordHeaders, {
              allowAuthorization: true,
            })}
            onUpdate={update}
          />
        </AdvancedGroup>
        <AdvancedGroup title="Client">
          <Grid cols={3}>
            <TextInput
              label="Client ID"
              description="Reported to the brokers."
              placeholder="otel-collector"
              value={value.clientID}
              onChange={(event) => {
                value.clientID = event.currentTarget.value.trim();
                update();
              }}
            />
            <DurationField
              label="Timeout"
              description="Per export."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
            <DurationField
              label="Idle connection timeout"
              description="Close idle broker connections after this."
              value={value.connIdleTimeout}
              onChange={(next) => {
                value.connIdleTimeout = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const DATADOG_SITES = [
  { value: "datadoghq.com", label: "US1 · datadoghq.com" },
  { value: "us3.datadoghq.com", label: "US3 · us3.datadoghq.com" },
  { value: "us5.datadoghq.com", label: "US5 · us5.datadoghq.com" },
  { value: "datadoghq.eu", label: "EU1 · datadoghq.eu" },
  { value: "ap1.datadoghq.com", label: "AP1 · ap1.datadoghq.com" },
  { value: "ap2.datadoghq.com", label: "AP2 · ap2.datadoghq.com" },
  { value: "ddog-gov.com", label: "US1-FED · ddog-gov.com" },
];

const withCurrent = (
  options: { value: string; label: string }[],
  current: string,
) =>
  current && !options.some((option) => option.value === current)
    ? [...options, { value: current, label: current }]
    : options;

const DatadogForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Datadog;
  update: Update;
}) => {
  const { value, update } = props;
  const api = () => {
    if (!value.api)
      value.api = EnterpriseP.CollectorExporter_Spec_Datadog_API.create();
    return value.api;
  };
  const metrics = () => {
    if (!value.metrics)
      value.metrics =
        EnterpriseP.CollectorExporter_Spec_Datadog_Metrics.create();
    return value.metrics;
  };
  const logs = () => {
    if (!value.logs) {
      value.logs = EnterpriseP.CollectorExporter_Spec_Datadog_Logs.create({
        useCompression: true,
      });
    }
    return value.logs;
  };
  const prune = () => {
    const m = value.metrics;
    if (
      m &&
      !m.endpoint &&
      !m.resourceAttributesAsTags &&
      !m.instrumentationScopeMetadataAsTags
    ) {
      value.metrics = undefined;
    }
    const l = value.logs;
    if (
      l &&
      !l.endpoint &&
      l.useCompression &&
      !l.compressionLevel &&
      !l.batchWait
    ) {
      value.logs = undefined;
    }
  };
  const hostMetadataEnabled = value.hostMetadata?.enabled ?? true;

  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Account"
        description="The Datadog site of your organization and an API key of it."
      >
        <Grid>
          <Select
            label="Site"
            description="Shown in the URL you use to sign in to Datadog."
            placeholder="US1 · datadoghq.com"
            clearable
            data={withCurrent(DATADOG_SITES, value.api?.site ?? "")}
            value={value.api?.site || null}
            onChange={(next) => {
              api().site = next ?? "";
              update();
            }}
          />
          <SecretField
            required
            label="API key"
            description="Secret holding a Datadog API key, not an application key."
            value={secretName(value.api?.key)}
            onChange={(name) => {
              api().key = toSecret(name);
              update();
            }}
          />
        </Grid>
        <SecretsHint />
      </FormSection>

      <Advanced
        summary="Hostname, intake URLs, tags, log batching and host metadata."
        count={countSet(
          value.hostname,
          value.api?.failOnInvalidKey,
          value.hostnameDetectionTimeout,
          value.metrics,
          value.logs,
          value.hostMetadata,
        )}
      >
        <AdvancedGroup title="Host">
          <Grid>
            <TextInput
              label="Hostname"
              description="Attributed to the telemetry. Detected when not set."
              placeholder="Detected automatically"
              value={value.hostname}
              onChange={(event) => {
                value.hostname = event.currentTarget.value.trim();
                update();
              }}
            />
            <DurationField
              label="Hostname detection timeout"
              description="How long detection may take."
              value={value.hostnameDetectionTimeout}
              onChange={(next) => {
                value.hostnameDetectionTimeout = next;
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Fail on an invalid API key"
              description="Stop the Collector instead of only logging a warning."
              checked={value.api?.failOnInvalidKey ?? false}
              onChange={(checked) => {
                api().failOnInvalidKey = checked;
                update();
              }}
            />
            <ToggleRow
              label="Report host metadata"
              description="Send the host tags and metadata shown in the infrastructure list."
              checked={hostMetadataEnabled}
              onChange={(checked) => {
                value.hostMetadata = checked
                  ? undefined
                  : EnterpriseP.CollectorExporter_Spec_Datadog_HostMetadata.create(
                      { enabled: false },
                    );
                update();
              }}
            >
              <Grid>
                <DurationField
                  label="Reporting period"
                  description="Between 5 minutes and 24 hours. 30 minutes by default."
                  value={value.hostMetadata?.reporterPeriod}
                  onChange={(next) => {
                    value.hostMetadata = next
                      ? EnterpriseP.CollectorExporter_Spec_Datadog_HostMetadata.create(
                          {
                            enabled: true,
                            reporterPeriod: next,
                          },
                        )
                      : undefined;
                    update();
                  }}
                />
              </Grid>
            </ToggleRow>
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Metrics">
          <Grid>
            <TextInput
              type="url"
              label="Metrics intake URL"
              description="Derived from the site when not set."
              placeholder="https://api.datadoghq.com"
              value={value.metrics?.endpoint ?? ""}
              error={urlError(value.metrics?.endpoint ?? "")}
              onChange={(event) => {
                metrics().endpoint = event.currentTarget.value.trim();
                prune();
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Resource attributes as tags"
              checked={value.metrics?.resourceAttributesAsTags ?? false}
              onChange={(checked) => {
                metrics().resourceAttributesAsTags = checked;
                prune();
                update();
              }}
            />
            <ToggleRow
              label="Instrumentation scope as tags"
              description="Tag metrics with the scope name and version."
              checked={
                value.metrics?.instrumentationScopeMetadataAsTags ?? false
              }
              onChange={(checked) => {
                metrics().instrumentationScopeMetadataAsTags = checked;
                prune();
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Logs">
          <Grid>
            <TextInput
              type="url"
              label="Logs intake URL"
              description="Derived from the site when not set."
              placeholder="https://http-intake.logs.datadoghq.com"
              value={value.logs?.endpoint ?? ""}
              error={urlError(value.logs?.endpoint ?? "")}
              onChange={(event) => {
                logs().endpoint = event.currentTarget.value.trim();
                prune();
                update();
              }}
            />
            <DurationField
              label="Batch wait"
              description="Maximum wait before a batch is sent."
              value={value.logs?.batchWait}
              onChange={(next) => {
                logs().batchWait = next;
                prune();
                update();
              }}
            />
          </Grid>
          <ToggleList>
            <ToggleRow
              label="Compress log payloads"
              checked={value.logs?.useCompression ?? true}
              onChange={(checked) => {
                logs().useCompression = checked;
                prune();
                update();
              }}
            >
              <Grid>
                <NumberField
                  label="Compression level"
                  description="1 (fastest) to 9 (smallest)."
                  max={9}
                  value={value.logs?.compressionLevel ?? 0}
                  onChange={(next) => {
                    logs().compressionLevel = next;
                    prune();
                    update();
                  }}
                />
              </Grid>
            </ToggleRow>
          </ToggleList>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const SplunkForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Splunk;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <FormSection
        icon={Network}
        title="HTTP Event Collector"
        description="The HEC endpoint and a token allowed to write to the index."
      >
        <TextInput
          required
          type="url"
          label="Endpoint"
          placeholder="https://splunk.example.com:8088/services/collector"
          value={value.endpoint}
          error={urlError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <SecretField
            required
            label="HEC token"
            description="Secret holding the HTTP Event Collector token."
            value={secretName(value.token)}
            onChange={(name) => {
              value.token = toSecret(name);
              update();
            }}
          />
        </Grid>
      </FormSection>

      <FormSection
        icon={Route}
        title="Events"
        description="Leave empty to use the defaults of the HEC token."
      >
        <Grid cols={3}>
          <TextInput
            label="Index"
            placeholder="Token default"
            value={value.index}
            onChange={(event) => {
              value.index = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            label="Source"
            placeholder="Token default"
            value={value.source}
            onChange={(event) => {
              value.source = event.currentTarget.value;
              update();
            }}
          />
          <TextInput
            label="Source type"
            placeholder="Token default"
            value={value.sourceType}
            onChange={(event) => {
              value.sourceType = event.currentTarget.value;
              update();
            }}
          />
        </Grid>
      </FormSection>

      <Advanced
        summary="Metric format, compression, payload limits, timeout and TLS."
        count={countSet(
          value.useMultiMetricFormat,
          value.disableCompression,
          value.appName,
          value.appVersion,
          value.maxContentLengthLogs,
          value.maxContentLengthMetrics,
          value.maxIdleConns,
          value.timeout,
          value.tls?.insecureSkipVerify,
        )}
      >
        <AdvancedGroup title="Payloads">
          <ToggleList>
            <ToggleRow
              label="Multi-metric format"
              description="Combine metrics that share dimensions into a single event."
              checked={value.useMultiMetricFormat}
              onChange={(checked) => {
                value.useMultiMetricFormat = checked;
                update();
              }}
            />
            <ToggleRow
              label="Compress payloads"
              checked={!value.disableCompression}
              onChange={(checked) => {
                value.disableCompression = !checked;
                update();
              }}
            />
          </ToggleList>
          <Grid>
            <NumberField
              label="Maximum log payload"
              description="Per request."
              suffix=" bytes"
              value={value.maxContentLengthLogs}
              onChange={(next) => {
                value.maxContentLengthLogs = next;
                update();
              }}
            />
            <NumberField
              label="Maximum metric payload"
              description="Per request."
              suffix=" bytes"
              value={value.maxContentLengthMetrics}
              onChange={(next) => {
                value.maxContentLengthMetrics = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup title="Application">
          <Grid>
            <TextInput
              label="App name"
              description="Reported to Splunk."
              placeholder="OpenTelemetry Collector"
              value={value.appName}
              onChange={(event) => {
                value.appName = event.currentTarget.value;
                update();
              }}
            />
            <TextInput
              label="App version"
              description="Reported to Splunk."
              placeholder="Collector version"
              value={value.appVersion}
              onChange={(event) => {
                value.appVersion = event.currentTarget.value;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup title="Connection">
          <Grid>
            <DurationField
              label="Timeout"
              description="Per request, up to 5 minutes."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
            <NumberField
              label="Idle connections"
              description="Kept open to the endpoint, up to 10,000."
              max={10000}
              value={value.maxIdleConns}
              onChange={(next) => {
                value.maxIdleConns = next;
                update();
              }}
            />
          </Grid>
          {value.endpoint.startsWith("https://") && (
            <ToggleList>
              <ToggleRow
                label="Skip certificate verification"
                description="Accept any certificate presented by the endpoint."
                checked={value.tls?.insecureSkipVerify ?? false}
                onChange={(checked) => {
                  value.tls = checked
                    ? EnterpriseP.CollectorExporter_Spec_Splunk_TLS.create({
                        insecureSkipVerify: true,
                      })
                    : undefined;
                  update();
                }}
              />
            </ToggleList>
          )}
          {value.tls?.insecureSkipVerify && <SkipVerifyWarning />}
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const InfluxDBForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_InfluxDB;
  update: Update;
}) => {
  const { value, update } = props;
  const S = EnterpriseP.CollectorExporter_Spec_InfluxDB_MetricsSchema;
  const P = EnterpriseP.CollectorExporter_Spec_InfluxDB_Precision;
  const v1 = value.v1Compatibility;

  return (
    <>
      <FormSection
        icon={Network}
        title="Connection"
        description="The InfluxDB server and the bucket that receives the telemetry."
      >
        <TextInput
          required
          type="url"
          label="Endpoint"
          placeholder="https://influxdb.example.com:8086"
          value={value.endpoint}
          error={urlError(value.endpoint)}
          onChange={(event) => {
            value.endpoint = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid cols={3}>
          <TextInput
            required
            label="Organization"
            description="Name of the organization that owns the bucket."
            placeholder="my-org"
            value={value.org}
            error={value.org.trim() ? undefined : "Required"}
            onChange={(event) => {
              value.org = event.currentTarget.value;
              update();
            }}
          />
          <TextInput
            required
            label="Bucket"
            description="Bucket that receives the points."
            placeholder="telemetry"
            value={value.bucket}
            error={value.bucket.trim() ? undefined : "Required"}
            onChange={(event) => {
              value.bucket = event.currentTarget.value;
              update();
            }}
          />
          <SecretField
            label="API token"
            description="Secret holding a token with write access."
            value={secretName(value.token)}
            onChange={(name) => {
              value.token = optionalSecret(name);
              update();
            }}
          />
        </Grid>
      </FormSection>

      <Advanced
        summary="Metrics schema, precision, log tags, headers, payload limits and InfluxDB 1.x."
        count={countSet(
          value.metricsSchema,
          value.precision,
          value.logRecordDimensions,
          value.headers,
          value.payloadMaxLines,
          value.payloadMaxBytes,
          value.timeout,
          value.v1Compatibility,
        )}
      >
        <AdvancedGroup title="Points">
          <Grid>
            <EnumField
              label="Metrics schema"
              description="How metrics map to measurements."
              value={value.metricsSchema}
              options={[
                {
                  label: "Telegraf Prometheus v1, one measurement per metric",
                  value: S.TELEGRAF_PROMETHEUS_V1,
                },
                {
                  label: "Telegraf Prometheus v2, shared measurement",
                  value: S.TELEGRAF_PROMETHEUS_V2,
                },
              ]}
              onChange={(next) => {
                value.metricsSchema = next;
                update();
              }}
            />
            <EnumField
              label="Timestamp precision"
              description="Nanoseconds when not set."
              value={value.precision}
              options={[
                { label: "Nanoseconds", value: P.NS },
                { label: "Microseconds", value: P.US },
                { label: "Milliseconds", value: P.MS },
                { label: "Seconds", value: P.S },
              ]}
              onChange={(next) => {
                value.precision = next;
                update();
              }}
            />
          </Grid>
          <TagsInput
            label="Log attributes as tags"
            description="Log record attributes written as tags instead of fields. Press Enter after each."
            placeholder={value.logRecordDimensions.length ? "" : "service.name"}
            splitChars={[",", " "]}
            acceptValueOnBlur
            maxTags={64}
            value={value.logRecordDimensions}
            onChange={(next) => {
              value.logRecordDimensions = next;
              update();
            }}
          />
        </AdvancedGroup>
        <AdvancedGroup title="Requests">
          <Grid cols={3}>
            <NumberField
              label="Maximum lines"
              description="Per request, up to 1,000,000."
              max={1_000_000}
              value={value.payloadMaxLines}
              onChange={(next) => {
                value.payloadMaxLines = next;
                update();
              }}
            />
            <NumberField
              label="Maximum size"
              description="Per request, up to 256 MiB."
              suffix=" bytes"
              max={256 * 1024 * 1024}
              value={value.payloadMaxBytes}
              onChange={(next) => {
                value.payloadMaxBytes = next;
                update();
              }}
            />
            <DurationField
              label="Timeout"
              description="Per request."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
          <KeyValueList
            label="Request headers"
            description="Sent with every write request."
            addLabel="Add header"
            keyPlaceholder="X-Custom"
            valuePlaceholder="value"
            items={value.headers}
            make={() =>
              EnterpriseP.CollectorExporter_Spec_InfluxDB_Header.create()
            }
            keyError={headerKeyError(value.headers)}
            onUpdate={update}
          />
        </AdvancedGroup>
        <AdvancedGroup title="InfluxDB 1.x">
          <ToggleList>
            <ToggleRow
              label="Write to InfluxDB 1.x"
              description="Use a 1.x database instead of the organization and bucket."
              checked={v1?.enabled ?? false}
              onChange={(checked) => {
                value.v1Compatibility = checked
                  ? EnterpriseP.CollectorExporter_Spec_InfluxDB_V1Compatibility.create(
                      {
                        ...v1,
                        enabled: true,
                      },
                    )
                  : undefined;
                update();
              }}
            >
              {v1 && (
                <Grid cols={3}>
                  <TextInput
                    required
                    label="Database"
                    description="The 1.x database that receives the points."
                    value={v1.db}
                    error={v1.db.trim() ? undefined : "Required"}
                    onChange={(event) => {
                      v1.db = event.currentTarget.value;
                      update();
                    }}
                  />
                  <TextInput
                    label="Username"
                    description="Leave empty without authentication."
                    value={v1.username}
                    onChange={(event) => {
                      v1.username = event.currentTarget.value;
                      update();
                    }}
                  />
                  <SecretField
                    label="Password"
                    description="Secret holding the password."
                    value={secretName(v1.password)}
                    onChange={(name) => {
                      v1.password = optionalSecret(name);
                      update();
                    }}
                  />
                </Grid>
              )}
            </ToggleRow>
          </ToggleList>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const LOGZIO_REGIONS = [
  { value: "us", label: "United States · us" },
  { value: "eu", label: "Europe · eu" },
  { value: "uk", label: "United Kingdom · uk" },
  { value: "ca", label: "Canada · ca" },
  { value: "au", label: "Australia · au" },
];

const LogzioForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_Logzio;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Account"
        description="The region of your Logz.io account and its log shipping token. Logs only."
      >
        <Grid>
          <SecretField
            required
            label="Shipping token"
            description="Secret holding the logs shipping token."
            value={secretName(value.token)}
            onChange={(name) => {
              value.token = toSecret(name);
              update();
            }}
          />
          <Select
            label="Region"
            description="Region of the account. United States when not set."
            placeholder="United States · us"
            clearable
            data={withCurrent(LOGZIO_REGIONS, value.region)}
            value={value.region || null}
            onChange={(next) => {
              value.region = next ?? "";
              update();
            }}
          />
        </Grid>
        <SecretsHint />
      </FormSection>

      <Advanced
        summary="Listener URL and timeout."
        count={countSet(value.endpoint, value.timeout)}
      >
        <AdvancedGroup title="Delivery">
          <Grid>
            <TextInput
              type="url"
              label="Listener URL"
              description="Replaces the URL derived from the region."
              placeholder="https://listener.logz.io:8053"
              value={value.endpoint}
              error={urlError(value.endpoint)}
              onChange={(event) => {
                value.endpoint = event.currentTarget.value.trim();
                update();
              }}
            />
            <DurationField
              label="Timeout"
              description="Per request, up to 5 minutes."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const AzureMonitorForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_AzureMonitor;
  update: Update;
}) => {
  const { value, update } = props;
  const mode =
    value.instrumentationKey && !value.connectionString ? "key" : "connection";

  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Application Insights resource"
        description="Copy the credential from the Overview page of the resource."
      >
        <ChoiceGroup
          label="Credential"
          value={mode}
          onChange={(next) => {
            if (next === "key") {
              value.instrumentationKey = toSecret(
                secretName(value.instrumentationKey),
              );
              value.connectionString = undefined;
            } else {
              value.connectionString = toSecret(
                secretName(value.connectionString),
              );
              value.instrumentationKey = undefined;
            }
            update();
          }}
          options={[
            {
              value: "connection",
              label: "Connection string",
              hint: "Recommended, includes the endpoints",
            },
            { value: "key", label: "Instrumentation key", hint: "Legacy" },
          ]}
        />
        <Grid>
          {mode === "connection" ? (
            <SecretField
              required
              label="Connection string"
              description="Secret holding the connection string."
              value={secretName(value.connectionString)}
              onChange={(name) => {
                value.connectionString = toSecret(name);
                update();
              }}
            />
          ) : (
            <SecretField
              required
              label="Instrumentation key"
              description="Secret holding the instrumentation key."
              value={secretName(value.instrumentationKey)}
              onChange={(name) => {
                value.instrumentationKey = toSecret(name);
                update();
              }}
            />
          )}
        </Grid>
        <SecretsHint />
      </FormSection>

      <Advanced
        summary="Ingestion URL, batching, events and shutdown."
        count={countSet(
          value.endpoint,
          value.maxBatchSize,
          value.maxBatchInterval,
          value.shutdownTimeout,
          value.customEventsEnabled,
          value.exceptionEventsEnabled,
        )}
      >
        <AdvancedGroup title="Log records">
          <ToggleList>
            <ToggleRow
              label="Custom events"
              description="Send log records with a microsoft.custom_event.name attribute as custom events."
              checked={value.customEventsEnabled}
              onChange={(checked) => {
                value.customEventsEnabled = checked;
                update();
              }}
            />
            <ToggleRow
              label="Exceptions"
              description="Send log records with exception attributes as exception telemetry."
              checked={value.exceptionEventsEnabled}
              onChange={(checked) => {
                value.exceptionEventsEnabled = checked;
                update();
              }}
            />
          </ToggleList>
        </AdvancedGroup>
        <AdvancedGroup title="Delivery">
          <Grid>
            <TextInput
              type="url"
              label="Ingestion URL"
              description="Replaces the one from the connection string."
              placeholder="https://westeurope-5.in.applicationinsights.azure.com/"
              value={value.endpoint}
              error={urlError(value.endpoint)}
              onChange={(event) => {
                value.endpoint = event.currentTarget.value.trim();
                update();
              }}
            />
            <NumberField
              label="Maximum batch size"
              description="Items per batch, up to 100,000."
              max={100000}
              value={value.maxBatchSize}
              onChange={(next) => {
                value.maxBatchSize = next;
                update();
              }}
            />
            <DurationField
              label="Maximum batch interval"
              description="Wait before a partial batch is sent."
              value={value.maxBatchInterval}
              onChange={(next) => {
                value.maxBatchInterval = next;
                update();
              }}
            />
            <DurationField
              label="Shutdown timeout"
              description="Time to flush buffered items on shutdown."
              value={value.shutdownTimeout}
              onChange={(next) => {
                value.shutdownTimeout = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

type ADXAuth =
  EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth["type"];

const AzureDataExplorerForm = (props: {
  value: EnterpriseP.CollectorExporter_Spec_AzureDataExplorer;
  update: Update;
}) => {
  const { value, update } = props;
  const I = EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_IngestionType;
  const stash = React.useRef<Partial<Record<string, ADXAuth>>>({});
  const auth = value.auth?.type;
  const method = auth?.oneofKind ?? "servicePrincipal";

  const changeAuth = (next: string) => {
    if (auth?.oneofKind) stash.current[auth.oneofKind] = structuredClone(auth);
    const type: ADXAuth =
      stash.current[next] ??
      (next === "managedIdentity"
        ? { oneofKind: "managedIdentity", managedIdentity: { id: "system" } }
        : next === "azureDefault"
          ? { oneofKind: "azureDefault", azureDefault: {} }
          : {
              oneofKind: "servicePrincipal",
              servicePrincipal: {
                applicationID: "",
                tenantID: "",
                applicationKey: toSecret(""),
              },
            });
    value.auth =
      EnterpriseP.CollectorExporter_Spec_AzureDataExplorer_Auth.create({
        type,
      });
    update();
  };

  const clusterURIError = (uri: string) => {
    if (!uri) return undefined;
    return urlError(uri, { httpsOnly: true });
  };

  return (
    <>
      <FormSection
        icon={Network}
        title="Cluster"
        description="The cluster URI from the Overview page and the database that receives the telemetry."
      >
        <TextInput
          required
          type="url"
          label="Cluster URI"
          placeholder="https://mycluster.westeurope.kusto.windows.net"
          value={value.clusterURI}
          error={clusterURIError(value.clusterURI)}
          onChange={(event) => {
            value.clusterURI = event.currentTarget.value.trim();
            update();
          }}
        />
        <Grid>
          <TextInput
            label="Database"
            description="Database that contains the tables."
            placeholder="oteldb"
            value={value.database}
            error={identError(value.database)}
            onChange={(event) => {
              value.database = event.currentTarget.value.trim();
              update();
            }}
          />
          <EnumField
            label="Ingestion"
            description="Queued when not set."
            value={value.ingestionType}
            options={[
              { label: "Queued, higher throughput", value: I.QUEUED },
              { label: "Managed streaming, lower latency", value: I.MANAGED },
            ]}
            onChange={(next) => {
              value.ingestionType = next;
              update();
            }}
          />
        </Grid>
      </FormSection>

      <AuthSection description="The identity needs the Ingestor role on the database.">
        <ChoiceGroup
          label="Method"
          value={method}
          onChange={changeAuth}
          options={[
            {
              value: "servicePrincipal",
              label: "Service principal",
              hint: "Entra ID app and secret",
            },
            {
              value: "managedIdentity",
              label: "Managed identity",
              hint: "System or user-assigned",
            },
            {
              value: "azureDefault",
              label: "Azure default",
              hint: "Workload identity, environment…",
            },
          ]}
        />
        {auth?.oneofKind === "servicePrincipal" && (
          <Grid cols={3}>
            <TextInput
              required
              label="Application (client) ID"
              description="From the Overview page of the app registration."
              placeholder="00000000-0000-0000-0000-000000000000"
              value={auth.servicePrincipal.applicationID}
              error={
                auth.servicePrincipal.applicationID &&
                !isUUID(auth.servicePrincipal.applicationID)
                  ? "Must be a UUID"
                  : undefined
              }
              onChange={(event) => {
                auth.servicePrincipal.applicationID =
                  event.currentTarget.value.trim();
                update();
              }}
            />
            <TextInput
              required
              label="Directory (tenant) ID"
              description="From the Overview page of the app registration."
              placeholder="00000000-0000-0000-0000-000000000000"
              value={auth.servicePrincipal.tenantID}
              error={
                auth.servicePrincipal.tenantID &&
                !isUUID(auth.servicePrincipal.tenantID)
                  ? "Must be a UUID"
                  : undefined
              }
              onChange={(event) => {
                auth.servicePrincipal.tenantID =
                  event.currentTarget.value.trim();
                update();
              }}
            />
            <SecretField
              required
              label="Client secret"
              description="Secret holding the client secret value."
              value={secretName(auth.servicePrincipal.applicationKey)}
              onChange={(name) => {
                auth.servicePrincipal.applicationKey = toSecret(name);
                update();
              }}
            />
          </Grid>
        )}
        {auth?.oneofKind === "managedIdentity" && (
          <Grid>
            <TextInput
              required
              label="Identity"
              description='"system" for the system-assigned identity, or the client ID of a user-assigned one.'
              placeholder="system"
              value={auth.managedIdentity.id}
              error={
                !auth.managedIdentity.id.trim()
                  ? "Required"
                  : auth.managedIdentity.id.toLowerCase() !== "system" &&
                      !isUUID(auth.managedIdentity.id)
                    ? 'Use "system" or a UUID'
                    : undefined
              }
              onChange={(event) => {
                auth.managedIdentity.id = event.currentTarget.value.trim();
                update();
              }}
            />
          </Grid>
        )}
        {auth?.oneofKind === "azureDefault" && (
          <Callout tone="info" title="Uses the environment of the Collector">
            Credentials are resolved from environment variables, workload
            identity or the managed identity of the node, in that order.
          </Callout>
        )}
      </AuthSection>

      <FormSection
        icon={Route}
        title="Tables"
        description="Leave empty to use OTELLogs and OTELMetrics."
      >
        <Grid>
          <TextInput
            label="Logs table"
            placeholder="OTELLogs"
            value={value.logsTable}
            error={identError(value.logsTable)}
            onChange={(event) => {
              value.logsTable = event.currentTarget.value.trim();
              update();
            }}
          />
          <TextInput
            label="Metrics table"
            placeholder="OTELMetrics"
            value={value.metricsTable}
            error={identError(value.metricsTable)}
            onChange={(event) => {
              value.metricsTable = event.currentTarget.value.trim();
              update();
            }}
          />
        </Grid>
      </FormSection>

      <Advanced
        summary="Ingestion mappings and timeout."
        count={countSet(
          value.logsTableMapping,
          value.metricsTableMapping,
          value.timeout,
        )}
      >
        <AdvancedGroup
          title="Ingestion mappings"
          description="Named mappings that map the JSON payload to table columns."
        >
          <Grid>
            <TextInput
              label="Logs mapping"
              placeholder="None"
              value={value.logsTableMapping}
              error={identError(value.logsTableMapping, "-.")}
              onChange={(event) => {
                value.logsTableMapping = event.currentTarget.value.trim();
                update();
              }}
            />
            <TextInput
              label="Metrics mapping"
              placeholder="None"
              value={value.metricsTableMapping}
              error={identError(value.metricsTableMapping, "-.")}
              onChange={(event) => {
                value.metricsTableMapping = event.currentTarget.value.trim();
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
        <AdvancedGroup title="Requests">
          <Grid>
            <DurationField
              label="Timeout"
              description="Per ingestion request."
              value={value.timeout}
              onChange={(next) => {
                value.timeout = next;
                update();
              }}
            />
          </Grid>
        </AdvancedGroup>
      </Advanced>
    </>
  );
};

const TypeForm = (props: { type: ExporterType; update: Update }) => {
  const { type, update } = props;
  switch (type.oneofKind) {
    case "otlp":
      return <OTLPForm value={type.otlp} update={update} />;
    case "otlpHTTP":
      return <OTLPHTTPForm value={type.otlpHTTP} update={update} />;
    case "prometheusRemoteWrite":
      return (
        <PrometheusForm value={type.prometheusRemoteWrite} update={update} />
      );
    case "clickhouse":
      return <ClickhouseForm value={type.clickhouse} update={update} />;
    case "elasticsearch":
      return <ElasticsearchForm value={type.elasticsearch} update={update} />;
    case "kafka":
      return <KafkaForm value={type.kafka} update={update} />;
    case "datadog":
      return <DatadogForm value={type.datadog} update={update} />;
    case "splunk":
      return <SplunkForm value={type.splunk} update={update} />;
    case "influxDB":
      return <InfluxDBForm value={type.influxDB} update={update} />;
    case "logzio":
      return <LogzioForm value={type.logzio} update={update} />;
    case "azureMonitor":
      return <AzureMonitorForm value={type.azureMonitor} update={update} />;
    case "azureDataExplorer":
      return (
        <AzureDataExplorerForm value={type.azureDataExplorer} update={update} />
      );
    default:
      return null;
  }
};

const Edit = (props: {
  item: EnterpriseP.CollectorExporter;
  onUpdate: (item: EnterpriseP.CollectorExporter) => void;
}) => {
  const [req, setReq] = React.useState(() => cloneForEdit(props.item));
  const configurations = React.useRef<Partial<Record<string, ExporterType>>>(
    {},
  );
  const itemKey =
    props.item.metadata?.uid || props.item.apiVersion || props.item.kind;
  const initialKind = React.useRef(req.spec?.type.oneofKind);

  React.useEffect(() => {
    const next = cloneForEdit(props.item);
    setReq(next);
    configurations.current = {};
    initialKind.current = next.spec?.type.oneofKind;
  }, [itemKey]);

  const updateReq = () => {
    const next = EnterpriseP.CollectorExporter.clone(req);
    setReq(next);
    props.onUpdate(EnterpriseP.CollectorExporter.clone(next));
  };

  if (!req.spec) return null;
  const spec = req.spec;
  const kind = spec.type.oneofKind;

  const changeType = (next: ExporterKind) => {
    if (next === kind) return;
    if (kind) configurations.current[kind] = structuredClone(spec.type);
    spec.type = structuredClone(
      configurations.current[next] ?? createType(next),
    );
    updateReq();
  };

  return (
    <div className="@container flex min-w-0 flex-col gap-4">
      <DestinationPicker value={kind} onChange={changeType} />

      {!!props.item.metadata?.uid && kind !== initialKind.current && (
        <Callout tone="info" title="The destination changes when you save">
          The pipelines that use this exporter will send their telemetry to the
          new destination instead.
        </Callout>
      )}

      <TypeForm key={kind} type={spec.type} update={updateReq} />

      <FormSection
        icon={spec.isDisabled ? Power : Radio}
        title="Status"
        description="A disabled exporter keeps its configuration, but the Collector skips it in every pipeline."
      >
        <ToggleList>
          <ToggleRow
            label="Disable this exporter"
            checked={spec.isDisabled}
            onChange={(checked) => {
              spec.isDisabled = checked;
              updateReq();
            }}
          />
        </ToggleList>
      </FormSection>
    </div>
  );
};

export default Edit;
