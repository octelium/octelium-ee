import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import ResourceCreatePage from "@/components/ResourceLayout/ResourceCreate";
import ResourceEditPage from "@/components/ResourceLayout/ResourceEdit";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import CollectorExporterEdit from "@/pages/enterprise/CollectorExporter/Edit";
import exporterInfo from "@/pages/enterprise/CollectorExporter/router";
import store from "@/store";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router-dom";

declare global {
  interface Window {
    exporter: {
      calls: { method: string; request: any }[];
      value?: any;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: { method: string; request: any }[] = [];
window.exporter = { calls };

const secret = (fromSecret: string) => ({
  type: { oneofKind: "fromSecret" as const, fromSecret },
});

const exporter = (
  name: string,
  type: EnterpriseP.CollectorExporter_Spec["type"],
  isDisabled = false,
) =>
  EnterpriseP.CollectorExporter.create({
    apiVersion: "enterprise/v1",
    kind: "CollectorExporter",
    metadata: { name, uid: `uid-${name}`, resourceVersion: "1" },
    spec: { isDisabled, type },
    status: {},
  });

const exporters: Record<string, EnterpriseP.CollectorExporter> = {
  "otlp-tempo": exporter("otlp-tempo", {
    oneofKind: "otlp",
    otlp: EnterpriseP.CollectorExporter_Spec_OTLP.create({
      endpoint: "tempo.observability.svc:4317",
      compression: EnterpriseP.CollectorExporter_Spec_OTLP_Compression.GZIP,
      headers: [{ key: "x-scope-orgid", value: "platform" }],
      auth: { type: { oneofKind: "bearer", bearer: secret("tempo-token") } },
      tls: {
        insecure: false,
        insecureSkipVerify: false,
        serverNameOverride: "tempo.internal",
        caPEM: "",
      },
      timeout: { type: { oneofKind: "seconds", seconds: 10 } },
    }),
  }),
  "otlp-plain": exporter("otlp-plain", {
    oneofKind: "otlp",
    otlp: EnterpriseP.CollectorExporter_Spec_OTLP.create({
      endpoint: "collector.monitoring.svc:4317",
    }),
  }),
  "grafana-cloud": exporter("grafana-cloud", {
    oneofKind: "otlpHTTP",
    otlpHTTP: EnterpriseP.CollectorExporter_Spec_OTLPHTTP.create({
      endpoint: "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp",
      auth: {
        type: {
          oneofKind: "basic",
          basic: {
            username: "123456",
            password: secret("grafana-cloud-token"),
          },
        },
      },
      encoding: EnterpriseP.CollectorExporter_Spec_OTLPHTTP_Encoding.PROTO,
    }),
  }),
  mimir: exporter("mimir", {
    oneofKind: "prometheusRemoteWrite",
    prometheusRemoteWrite:
      EnterpriseP.CollectorExporter_Spec_PrometheusRemoteWrite.create({
        endpoint: "https://mimir.example.com/api/v1/push",
        namespace: "octelium",
        externalLabels: [{ key: "cluster", value: "production" }],
        remoteWriteQueue: { enabled: true, queueSize: 10000, numConsumers: 5 },
      }),
  }),
  "clickhouse-logs": exporter("clickhouse-logs", {
    oneofKind: "clickhouse",
    clickhouse: EnterpriseP.CollectorExporter_Spec_Clickhouse.create({
      endpoint: "tcp://clickhouse.data.svc:9000",
      username: "otel",
      password: secret("clickhouse-password"),
      database: "otel",
      createSchema: true,
      ttl: { type: { oneofKind: "days", days: 30 } },
    }),
  }),
  elastic: exporter("elastic", {
    oneofKind: "elasticsearch",
    elasticsearch: EnterpriseP.CollectorExporter_Spec_Elasticsearch.create({
      endpoints: [
        "https://es-1.example.com:9200",
        "https://es-2.example.com:9200",
      ],
      logsIndex: "octelium-logs",
      auth: { type: { oneofKind: "apiKey", apiKey: secret("es-api-key") } },
    }),
  }),
  kafka: exporter("kafka", {
    oneofKind: "kafka",
    kafka: EnterpriseP.CollectorExporter_Spec_Kafka.create({
      brokers: ["kafka-0.kafka:9092", "kafka-1.kafka:9092"],
      logs: {
        topic: "octelium-logs",
        encoding: EnterpriseP.CollectorExporter_Spec_Kafka_Encoding.OTLP_JSON,
      },
    }),
  }),
  datadog: exporter(
    "datadog",
    {
      oneofKind: "datadog",
      datadog: EnterpriseP.CollectorExporter_Spec_Datadog.create({
        api: {
          key: secret("datadog-api-key"),
          site: "datadoghq.eu",
          failOnInvalidKey: false,
        },
      }),
    },
    true,
  ),
  splunk: exporter("splunk", {
    oneofKind: "splunk",
    splunk: EnterpriseP.CollectorExporter_Spec_Splunk.create({
      endpoint: "https://splunk.example.com:8088/services/collector",
      token: secret("splunk-hec-token"),
      index: "octelium",
    }),
  }),
  influxdb: exporter("influxdb", {
    oneofKind: "influxDB",
    influxDB: EnterpriseP.CollectorExporter_Spec_InfluxDB.create({
      endpoint: "https://influx.example.com:8086",
      org: "platform",
      bucket: "octelium",
      token: secret("influx-token"),
    }),
  }),
  logzio: exporter("logzio", {
    oneofKind: "logzio",
    logzio: EnterpriseP.CollectorExporter_Spec_Logzio.create({
      token: secret("logzio-token"),
      region: "eu",
    }),
  }),
  "app-insights": exporter("app-insights", {
    oneofKind: "azureMonitor",
    azureMonitor: EnterpriseP.CollectorExporter_Spec_AzureMonitor.create({
      connectionString: secret("appinsights-connection"),
    }),
  }),
  kusto: exporter("kusto", {
    oneofKind: "azureDataExplorer",
    azureDataExplorer:
      EnterpriseP.CollectorExporter_Spec_AzureDataExplorer.create({
        clusterURI: "https://octelium.westeurope.kusto.windows.net",
        database: "telemetry",
        auth: {
          type: {
            oneofKind: "servicePrincipal",
            servicePrincipal: {
              applicationID: "6f1d2b9c-7a51-4c4e-9a0f-2d8f2f4f8b11",
              tenantID: "0b3f1e7a-24c4-4a42-8e0e-7c1c2f9e5d21",
              applicationKey: secret("adx-app-key"),
            },
          },
        },
      }),
  }),
};

const secrets = [
  "tempo-token",
  "grafana-cloud-token",
  "clickhouse-password",
  "es-api-key",
  "datadog-api-key",
  "splunk-hec-token",
  "influx-token",
  "logzio-token",
  "appinsights-connection",
  "adx-app-key",
  "kafka-sasl-password",
].map((name) =>
  EnterpriseP.Secret.create({
    apiVersion: "enterprise/v1",
    kind: "Secret",
    metadata: { name, uid: `uid-${name}` },
    spec: {},
    status: {},
  }),
);

const handlers: Record<string, (request: any) => unknown> = {
  getCollectorExporter: (request) => exporters[request?.name],
  listSecret: () => EnterpriseP.SecretList.create({ items: secrets }),
  getSecret: (request) =>
    secrets.find((item) => item.metadata?.name === request?.name),
  createCollectorExporter: (request) => request,
  updateCollectorExporter: (request) => request,
};

for (const client of [
  clients.getClientEnterprise(),
  clients.getClientVisibilityEnterprise(),
  clients.getClientCore(),
  clients.getClientVisibilityCore(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = async (request: any) => {
      calls.push({ method: method.localName, request });
      const handler = handlers[method.localName];
      if (handler) {
        const response = handler(request);
        if (!response) throw new RpcError("Not found", "NOT_FOUND");
        return { response };
      }
      return { response: method.O.create() };
    };
  }
}

const EditOnly = () => {
  const name = params.get("name");
  const item = React.useMemo(
    () =>
      name && exporters[name]
        ? exporters[name]
        : exporterInfo.Item.createResource!(),
    [name],
  );
  return (
    <div
      className="mx-auto p-6"
      style={{ maxWidth: Number(params.get("width") ?? 896) }}
    >
      <CollectorExporterEdit
        item={item}
        onUpdate={(next: EnterpriseP.CollectorExporter) => {
          window.exporter.value = EnterpriseP.CollectorExporter.toJson(next);
        }}
      />
    </div>
  );
};

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Outlet />,
      children: [
        { path: "edit-only", element: <EditOnly /> },
        {
          path: "enterprise/collectorexporters",
          element: <Outlet />,
          children: [
            {
              path: "create",
              element: (
                <div className="mx-auto max-w-5xl p-6">
                  <ResourceCreatePage
                    specComponent={exporterInfo.Item.Edit!}
                    createResource={exporterInfo.Item.createResource}
                  />
                </div>
              ),
            },
            {
              path: ":name",
              element: <ResourceItemDrawer />,
              children: [
                {
                  path: "",
                  element: (
                    <ResourceItemMainPage
                      infoComponent={exporterInfo.infoItemsGetter}
                    />
                  ),
                },
                {
                  path: "edit",
                  element: (
                    <ResourceEditPage specComponent={exporterInfo.Item.Edit!} />
                  ),
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  { initialEntries: [params.get("path") ?? "/edit-only"] },
);

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <QueryClientProvider client={queryClient}>
          <React.Suspense fallback={null}>
            <RouterProvider router={router} />
          </React.Suspense>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
