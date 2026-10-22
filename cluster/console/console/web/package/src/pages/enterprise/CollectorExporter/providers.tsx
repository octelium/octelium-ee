import type { CollectorExporter_Spec } from "@/apis/enterprisev1/enterprisev1";
import { twMerge } from "@/lib/utils";
import * as React from "react";

export type ExporterKind = NonNullable<
  CollectorExporter_Spec["type"]["oneofKind"]
>;

export type ExporterSignal = "logs" | "metrics";

export type ExporterCategory =
  "protocol" | "platform" | "storage" | "streaming";

export const CATEGORY_LABEL: Record<ExporterCategory, string> = {
  protocol: "Open protocols",
  platform: "Observability platforms",
  storage: "Databases & search",
  streaming: "Streaming",
};

export interface ExporterProvider {
  kind: ExporterKind;
  label: string;
  description: string;
  category: ExporterCategory;
  signals: ExporterSignal[];
  keywords: string[];
  tint: string;
}

export const PROVIDERS: ExporterProvider[] = [
  {
    kind: "otlp",
    label: "OTLP gRPC",
    description: "Any OpenTelemetry-compatible backend over gRPC.",
    category: "protocol",
    signals: ["logs", "metrics"],
    keywords: [
      "opentelemetry",
      "otel",
      "collector",
      "grpc",
      "tempo",
      "jaeger",
      "honeycomb",
      "new relic",
      "lightstep",
      "signoz",
    ],
    tint: "#425CC7",
  },
  {
    kind: "otlpHTTP",
    label: "OTLP HTTP",
    description: "Any OpenTelemetry-compatible backend over HTTP.",
    category: "protocol",
    signals: ["logs", "metrics"],
    keywords: [
      "opentelemetry",
      "otel",
      "http",
      "grafana cloud",
      "loki",
      "dynatrace",
      "axiom",
      "uptrace",
      "better stack",
    ],
    tint: "#425CC7",
  },
  {
    kind: "prometheusRemoteWrite",
    label: "Prometheus Remote Write",
    description: "Prometheus, Mimir, Thanos, Cortex or VictoriaMetrics.",
    category: "protocol",
    signals: ["metrics"],
    keywords: [
      "prometheus",
      "remote write",
      "prw",
      "mimir",
      "thanos",
      "cortex",
      "victoriametrics",
      "amp",
    ],
    tint: "#E6522C",
  },
  {
    kind: "datadog",
    label: "Datadog",
    description: "Datadog logs and metrics intake.",
    category: "platform",
    signals: ["logs", "metrics"],
    keywords: ["datadog", "dd", "apm"],
    tint: "#632CA6",
  },
  {
    kind: "splunk",
    label: "Splunk",
    description: "Splunk HTTP Event Collector (HEC).",
    category: "platform",
    signals: ["logs", "metrics"],
    keywords: ["splunk", "hec", "siem", "cisco"],
    tint: "#E2187D",
  },
  {
    kind: "logzio",
    label: "Logz.io",
    description: "Logz.io log management.",
    category: "platform",
    signals: ["logs"],
    keywords: ["logzio", "logz", "elk"],
    tint: "#14A57E",
  },
  {
    kind: "azureMonitor",
    label: "Azure Monitor",
    description: "Application Insights in Azure Monitor.",
    category: "platform",
    signals: ["logs", "metrics"],
    keywords: ["azure", "microsoft", "application insights", "app insights"],
    tint: "#0078D4",
  },
  {
    kind: "clickhouse",
    label: "ClickHouse",
    description: "ClickHouse tables, created automatically if you want.",
    category: "storage",
    signals: ["logs", "metrics"],
    keywords: ["clickhouse", "sql", "olap", "database"],
    tint: "#C9A000",
  },
  {
    kind: "elasticsearch",
    label: "Elasticsearch",
    description: "Elasticsearch or Elastic Cloud indexes and data streams.",
    category: "storage",
    signals: ["logs", "metrics"],
    keywords: ["elastic", "elasticsearch", "elk", "kibana", "search"],
    tint: "#00BFB3",
  },
  {
    kind: "influxDB",
    label: "InfluxDB",
    description: "InfluxDB v2 buckets, or v1 databases.",
    category: "storage",
    signals: ["logs", "metrics"],
    keywords: ["influx", "influxdb", "time series", "tsdb"],
    tint: "#22ADF6",
  },
  {
    kind: "azureDataExplorer",
    label: "Azure Data Explorer",
    description: "Azure Data Explorer (Kusto) tables.",
    category: "storage",
    signals: ["logs", "metrics"],
    keywords: ["azure", "microsoft", "kusto", "adx", "fabric"],
    tint: "#0078D4",
  },
  {
    kind: "kafka",
    label: "Kafka",
    description: "Kafka topics, encoded as OTLP or raw records.",
    category: "streaming",
    signals: ["logs", "metrics"],
    keywords: [
      "kafka",
      "apache",
      "redpanda",
      "msk",
      "confluent",
      "event hubs",
      "stream",
    ],
    tint: "#64748B",
  },
];

export const getProvider = (kind?: string) =>
  PROVIDERS.find((provider) => provider.kind === kind);

const OpenTelemetryMark = () => (
  <>
    <path
      fill="#425CC7"
      d="M12.6974 13.1173c-1.0224 1.0224-1.0224 2.68 0 3.7024 1.0224 1.0224 2.68 1.0224 3.7024 0 1.0224-1.0223 1.0224-2.68 0-3.7024-1.0223-1.0223-2.68-1.0223-3.7024 0zm2.7677 2.7701c-.5063.5063-1.3267.5063-1.833 0s-.5063-1.3266 0-1.833c.5063-.5062 1.3267-.5062 1.833 0 .5063.504.5063 1.3267 0 1.833z"
    />
    <path
      fill="#F5A800"
      d="M16.356.2355l-1.6041 1.6042c-.314.314-.314.83 0 1.144L21.015 9.247c.314.314.83.314 1.144 0l1.6042-1.6041c.314-.314.314-.83 0-1.144L17.4976.2354c-.314-.314-.8276-.314-1.1416 0z"
    />
    <path
      fill="#425CC7"
      d="M5.1173 20.734c.2848-.2848.2848-.7497 0-1.0345l-.8155-.8155c-.2848-.2848-.7497-.2848-1.0345 0l-1.6845 1.6845-.0024.0024-.4625-.4625c-.2556-.2556-.6718-.2556-.925 0-.2556.2556-.2556.6718 0 .925l2.775 2.775c.2556.2556.6718.2556.925 0 .2532-.2556.2556-.6718 0-.925l-.4625-.4625.0024-.0024z"
    />
    <path
      fill="#F5A800"
      d="M13.6029 4.841l-3.5637 3.5637c-.3164.3164-.3164.8374 0 1.1538l2.2006 2.2005c1.5554-1.1197 3.7365-.981 5.1361.4187l1.7819-1.7818c.3164-.3165.3164-.8374 0-1.1538l-4.401-4.401c-.3165-.319-.8374-.319-1.1539 0z"
    />
    <path
      fill="#425CC7"
      d="M11.3148 12.6865l-1.2999-1.2999c-.3043-.3043-.8033-.3043-1.1076 0l-4.5836 4.586c-.3042.3043-.3042.8033 0 1.1076l2.5973 2.5973c.3043.3043.8033.3043 1.1076 0l2.9478-2.9527c-.6231-1.2877-.5112-2.8431.3384-4.0383z"
    />
  </>
);

const SplunkMark = () => {
  const id = React.useId();
  return (
    <>
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#ED0080" />
          <stop offset="1" stopColor="#F99D1C" />
        </linearGradient>
      </defs>
      <path
        fill={`url(#${id})`}
        d="M4.2 3.2a1.6 1.6 0 0 1 2.2-.6l13.4 7.7a1.95 1.95 0 0 1 0 3.4L6.4 21.4a1.6 1.6 0 0 1-1.6-2.8L16.2 12 4.8 5.4a1.6 1.6 0 0 1-.6-2.2Z"
      />
    </>
  );
};

const AzureMonitorMark = () => {
  const id = React.useId();
  return (
    <>
      <defs>
        <linearGradient id={id} x1="0" y1="1" x2="1" y2="0">
          <stop offset="0" stopColor="#0062AD" />
          <stop offset="1" stopColor="#3CCBF4" />
        </linearGradient>
      </defs>
      <path
        fill={`url(#${id})`}
        d="M12 3.5A10.5 10.5 0 0 0 1.5 14a10.4 10.4 0 0 0 1.37 5.18 1 1 0 0 0 .87.5h16.52a1 1 0 0 0 .87-.5A10.4 10.4 0 0 0 22.5 14 10.5 10.5 0 0 0 12 3.5Zm0 2.3A8.2 8.2 0 0 1 20.2 14a8.1 8.1 0 0 1-.78 3.4H4.58A8.1 8.1 0 0 1 3.8 14 8.2 8.2 0 0 1 12 5.8Z"
      />
      <path
        fill="#0078D4"
        d="m16.6 8.3-3.36 4.38a1.9 1.9 0 1 1-1.4-.98l4.1-3.95a.42.42 0 0 1 .66.55Z"
      />
      <path
        fill="#50E6FF"
        d="M5.2 14.6h2v1.4h-2zM16.8 14.6h2v1.4h-2zM11.3 7.2h1.4v2h-1.4z"
      />
    </>
  );
};

const AzureDataExplorerMark = () => {
  const id = React.useId();
  return (
    <>
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#3CCBF4" />
          <stop offset="1" stopColor="#0062AD" />
        </linearGradient>
      </defs>
      <path
        fill={`url(#${id})`}
        d="M12 1.5c-5 0-9 1.6-9 3.6v13.8c0 2 4 3.6 9 3.6s9-1.6 9-3.6V5.1c0-2-4-3.6-9-3.6Z"
      />
      <ellipse
        cx="12"
        cy="5.1"
        rx="7.4"
        ry="2.3"
        fill="#50E6FF"
        opacity="0.85"
      />
      <path
        fill="#FFFFFF"
        d="M7.2 11.4h2.2v7.2H7.2zM10.9 14h2.2v4.6h-2.2zM14.6 9.6h2.2v9h-2.2z"
      />
    </>
  );
};

const LogzioMark = () => (
  <>
    <rect x="2.5" y="3" width="19" height="18" rx="4" fill="#14A57E" />
    <path
      fill="#FFFFFF"
      d="M6.5 7.6h1.8v8.8H6.5zM9.9 9.6h7.6v1.6H9.9zM9.9 12.9h5.6v1.6H9.9zM9.9 6.3h4.2v1.6H9.9z"
    />
    <circle cx="17.4" cy="16.6" r="1.4" fill="#FFD23F" />
  </>
);

const ClickHouseMark = () => (
  <>
    <rect x="1" y="1" width="22" height="22" rx="4.5" fill="#151515" />
    <path
      fill="#FFCC01"
      d="M5.2 5.4h1.7v13.2H5.2zM8.6 5.4h1.7v13.2H8.6zM12 5.4h1.7v13.2H12zM15.4 5.4h1.7v13.2h-1.7zM18.8 10.75h1.7v2.5h-1.7z"
    />
  </>
);

const ElasticsearchMark = () => (
  <>
    <path
      fill="#FEC514"
      d="M13.394 0C8.683 0 4.609 2.716 2.644 6.667h15.641a4.77 4.77 0 0 0 3.073-1.11c.446-.375.864-.785 1.247-1.243l.001-.002A11.974 11.974 0 0 0 13.394 0z"
    />
    <path
      className="fill-slate-700"
      d="M1.804 8.889a12.009 12.009 0 0 0 0 6.222h14.7a3.111 3.111 0 1 0 0-6.222z"
    />
    <path
      fill="#00BFB3"
      d="M2.644 17.333C4.61 21.283 8.684 24 13.395 24c3.701 0 7.011-1.677 9.212-4.312l-.001-.002a9.958 9.958 0 0 0-1.247-1.243 4.77 4.77 0 0 0-3.073-1.11z"
    />
  </>
);

const MARKS: Record<ExporterKind, () => React.ReactElement> = {
  otlp: OpenTelemetryMark,
  otlpHTTP: OpenTelemetryMark,
  prometheusRemoteWrite: () => (
    <path
      fill="#E6522C"
      d="M12 0C5.373 0 0 5.372 0 12c0 6.627 5.373 12 12 12s12-5.373 12-12c0-6.628-5.373-12-12-12zm0 22.46c-1.885 0-3.414-1.26-3.414-2.814h6.828c0 1.553-1.528 2.813-3.414 2.813zm5.64-3.745H6.36v-2.046h11.28v2.046zm-.04-3.098H6.391c-.037-.043-.075-.086-.111-.13-1.155-1.401-1.427-2.133-1.69-2.879-.005-.025 1.4.287 2.395.511 0 0 .513.119 1.262.255-.72-.843-1.147-1.915-1.147-3.01 0-2.406 1.845-4.508 1.18-6.207.648.053 1.34 1.367 1.387 3.422.689-.951.977-2.69.977-3.755 0-1.103.727-2.385 1.454-2.429-.648 1.069.168 1.984.894 4.256.272.854.237 2.29.447 3.201.07-1.892.395-4.652 1.595-5.605-.529 1.2.079 2.702.494 3.424.671 1.164 1.078 2.047 1.078 3.716a4.642 4.642 0 01-1.11 2.996c.792-.149 1.34-.283 1.34-.283l2.573-.502s-.374 1.538-1.81 3.019z"
    />
  ),
  datadog: () => (
    <path
      fill="#632CA6"
      d="M19.57 17.04l-1.997-1.316-1.665 2.782-1.937-.567-1.706 2.604.087.82 9.274-1.71-.538-5.794zm-8.649-2.498l1.488-.204c.241.108.409.15.697.223.45.117.97.23 1.741-.16.18-.088.553-.43.704-.625l6.096-1.106.622 7.527-10.444 1.882zm11.325-2.712l-.602.115L20.488 0 .789 2.285l2.427 19.693 2.306-.334c-.184-.263-.471-.581-.96-.989-.68-.564-.44-1.522-.039-2.127.53-1.022 3.26-2.322 3.106-3.956-.056-.594-.15-1.368-.702-1.898-.02.22.017.432.017.432s-.227-.289-.34-.683c-.112-.15-.2-.199-.319-.4-.085.233-.073.503-.073.503s-.186-.437-.216-.807c-.11.166-.137.48-.137.48s-.241-.69-.186-1.062c-.11-.323-.436-.965-.343-2.424.6.421 1.924.321 2.44-.439.171-.251.288-.939-.086-2.293-.24-.868-.835-2.16-1.066-2.651l-.028.02c.122.395.374 1.223.47 1.625.293 1.218.372 1.642.234 2.204-.116.488-.397.808-1.107 1.165-.71.358-1.653-.514-1.713-.562-.69-.55-1.224-1.447-1.284-1.883-.062-.477.275-.763.445-1.153-.243.07-.514.192-.514.192s.323-.334.722-.624c.165-.109.262-.178.436-.323a9.762 9.762 0 0 0-.456.003s.42-.227.855-.392c-.318-.014-.623-.003-.623-.003s.937-.419 1.678-.727c.509-.208 1.006-.147 1.286.257.367.53.752.817 1.569.996.501-.223.653-.337 1.284-.509.554-.61.99-.688.99-.688s-.216.198-.274.51c.314-.249.66-.455.66-.455s-.134.164-.259.426l.03.043c.366-.22.797-.394.797-.394s-.123.156-.268.358c.277-.002.838.012 1.056.037 1.285.028 1.552-1.374 2.045-1.55.618-.22.894-.353 1.947.68.903.888 1.609 2.477 1.259 2.833-.294.295-.874-.115-1.516-.916a3.466 3.466 0 0 1-.716-1.562 1.533 1.533 0 0 0-.497-.85s.23.51.23.96c0 .246.03 1.165.424 1.68-.039.076-.057.374-.1.43-.458-.554-1.443-.95-1.604-1.067.544.445 1.793 1.468 2.273 2.449.453.927.186 1.777.416 1.997.065.063.976 1.197 1.15 1.767.306.994.019 2.038-.381 2.685l-1.117.174c-.163-.045-.273-.068-.42-.153.08-.143.241-.5.243-.572l-.063-.111c-.348.492-.93.97-1.414 1.245-.633.359-1.363.304-1.838.156-1.348-.415-2.623-1.327-2.93-1.566 0 0-.01.191.048.234.34.383 1.119 1.077 1.872 1.56l-1.605.177.759 5.908c-.337.048-.39.071-.757.124-.325-1.147-.946-1.895-1.624-2.332-.599-.384-1.424-.47-2.214-.314l-.05.059a2.851 2.851 0 0 1 1.863.444c.654.413 1.181 1.481 1.375 2.124.248.822.42 1.7-.248 2.632-.476.662-1.864 1.028-2.986.237.3.481.705.876 1.25.95.809.11 1.577-.03 2.106-.574.452-.464.69-1.434.628-2.456l.714-.104.258 1.834 11.827-1.424zM15.05 6.848c-.034.075-.085.125-.007.37l.004.014.013.032.032.073c.14.287.295.558.552.696.067-.011.136-.019.207-.023.242-.01.395.028.492.08.009-.048.01-.119.005-.222-.018-.364.072-.982-.626-1.308-.264-.122-.634-.084-.757.068a.302.302 0 0 1 .058.013c.186.066.06.13.027.207m1.958 3.392c-.092-.05-.52-.03-.821.005-.574.068-1.193.267-1.328.372-.247.191-.135.523.047.66.511.382.96.638 1.432.575.29-.038.546-.497.728-.914.124-.288.124-.598-.058-.698m-5.077-2.942c.162-.154-.805-.355-1.556.156-.554.378-.571 1.187-.041 1.646.053.046.096.078.137.104a4.77 4.77 0 0 1 1.396-.412c.113-.125.243-.345.21-.745-.044-.542-.455-.456-.146-.749"
    />
  ),
  splunk: SplunkMark,
  logzio: LogzioMark,
  azureMonitor: AzureMonitorMark,
  clickhouse: ClickHouseMark,
  elasticsearch: ElasticsearchMark,
  influxDB: () => (
    <path
      fill="#22ADF6"
      d="M23.778 14.482l-2.287-9.959c-.13-.545-.624-1.09-1.169-1.248L9.87.051C9.74 0 9.584 0 9.426 0c-.443 0-.909.18-1.222.443L.716 7.412C.3 7.776.092 8.504.222 9.024l2.445 10.662c.13.545.624 1.092 1.169 1.248l9.775 3.015c.13.051.285.051.443.051.443 0 .91-.18 1.223-.443l8.007-7.435c.418-.39.624-1.092.494-1.64zM10.962 2.417l7.175 2.21c.285.08.285.21 0 .286l-3.77.858c-.285.08-.674-.05-.883-.26l-2.626-2.834c-.235-.232-.184-.336.104-.26zm4.47 12.872c.079.286-.105.444-.39.365l-7.748-2.392c-.285-.079-.338-.313-.13-.52l5.93-5.514c.209-.209.443-.13.52.156zM2.667 8.267l6.293-5.85c.21-.209.545-.18.754.025L12.86 5.85c.209.21.18.545-.026.754l-6.293 5.85c-.21.21-.545.181-.754-.025L2.64 9.024a.536.536 0 01.026-.757zm1.536 9.284L2.54 10.244c-.08-.285.05-.34.234-.13L5.4 12.949c.209.209.285.624.209.909L4.462 17.55c-.079.285-.208.285-.26 0zm9.202 4.264l-8.217-2.522a.547.547 0 01-.364-.675l1.378-4.421a.547.547 0 01.675-.365l8.216 2.522c.285.079.443.39.364.675L14.08 21.45a.553.553 0 01-.674.365zm7.279-5.98L15.2 20.93c-.209.209-.31.13-.234-.155l1.144-3.694c.079-.285.39-.573.674-.624l3.77-.858c.288-.076.339.054.13.234zm.598-1.09l-4.523 1.039a.534.534 0 01-.65-.39l-1.922-8.372a.534.534 0 01.39-.65L19.1 5.335a.534.534 0 01.649.39l1.923 8.371c.079.31-.102.596-.39.65Z"
    />
  ),
  azureDataExplorer: AzureDataExplorerMark,
  kafka: () => (
    <path
      className="fill-slate-800"
      d="M9.71 2.136a1.43 1.43 0 0 0-2.047 0h-.007a1.48 1.48 0 0 0-.421 1.042c0 .41.161.777.422 1.039l.007.007c.257.264.616.426 1.019.426.404 0 .766-.162 1.027-.426l.003-.007c.261-.262.421-.629.421-1.039 0-.408-.159-.777-.421-1.042H9.71zM8.683 22.295c.404 0 .766-.167 1.027-.429l.003-.008c.261-.261.421-.631.421-1.036 0-.41-.159-.778-.421-1.044H9.71a1.42 1.42 0 0 0-1.027-.432 1.4 1.4 0 0 0-1.02.432h-.007c-.26.266-.422.634-.422 1.044 0 .406.161.775.422 1.036l.007.008c.258.262.617.429 1.02.429zm7.89-4.462c.359-.096.683-.33.882-.684l.027-.052a1.47 1.47 0 0 0 .114-1.067 1.454 1.454 0 0 0-.675-.896l-.021-.014a1.425 1.425 0 0 0-1.078-.132c-.36.091-.684.335-.881.686-.2.349-.241.75-.146 1.119.099.363.33.691.675.896h.002c.346.203.737.239 1.101.144zm-6.405-7.342a2.083 2.083 0 0 0-1.485-.627c-.58 0-1.103.242-1.482.627-.378.385-.612.916-.612 1.507s.233 1.124.612 1.514a2.08 2.08 0 0 0 2.967 0c.379-.39.612-.923.612-1.514s-.233-1.122-.612-1.507zm-.835-2.51c.843.141 1.6.552 2.178 1.144h.004c.092.093.182.196.265.299l1.446-.851a3.176 3.176 0 0 1-.047-1.808 3.149 3.149 0 0 1 1.456-1.926l.025-.016a3.062 3.062 0 0 1 2.345-.306c.77.21 1.465.721 1.898 1.482v.002c.431.757.518 1.626.313 2.408a3.145 3.145 0 0 1-1.456 1.928l-.198.118h-.02a3.095 3.095 0 0 1-2.154.201 3.127 3.127 0 0 1-1.514-.944l-1.444.848a4.162 4.162 0 0 1 0 2.879l1.444.846c.413-.47.939-.789 1.514-.944a3.041 3.041 0 0 1 2.371.319l.048.023v.002a3.17 3.17 0 0 1 1.408 1.906 3.215 3.215 0 0 1-.313 2.405l-.026.053-.003-.005a3.147 3.147 0 0 1-1.867 1.436 3.096 3.096 0 0 1-2.371-.318v-.006a3.156 3.156 0 0 1-1.456-1.927 3.175 3.175 0 0 1 .047-1.805l-1.446-.848a3.905 3.905 0 0 1-.265.294l-.004.005a3.938 3.938 0 0 1-2.178 1.138v1.699a3.09 3.09 0 0 1 1.56.862l.002.004c.565.572.914 1.368.914 2.243 0 .873-.35 1.664-.914 2.239l-.002.009a3.1 3.1 0 0 1-2.21.931 3.1 3.1 0 0 1-2.206-.93h-.002v-.009a3.186 3.186 0 0 1-.916-2.239c0-.875.35-1.672.916-2.243v-.004h.002a3.1 3.1 0 0 1 1.558-.862v-1.699a3.926 3.926 0 0 1-2.176-1.138l-.006-.005a4.098 4.098 0 0 1-1.173-2.874c0-1.122.452-2.136 1.173-2.872h.006a3.947 3.947 0 0 1 2.176-1.144V6.289a3.137 3.137 0 0 1-1.558-.864h-.002v-.004a3.192 3.192 0 0 1-.916-2.243c0-.871.35-1.669.916-2.243l.002-.002A3.084 3.084 0 0 1 8.683 0c.861 0 1.641.355 2.21.932v.002h.002c.565.574.914 1.372.914 2.243 0 .876-.35 1.667-.914 2.243l-.002.005a3.142 3.142 0 0 1-1.56.864v1.692zm8.121-1.129l-.012-.019a1.452 1.452 0 0 0-.87-.668 1.43 1.43 0 0 0-1.103.146h.002c-.347.2-.58.529-.677.896-.095.365-.054.768.146 1.119l.007.009c.2.347.519.579.874.673.357.103.755.059 1.098-.144l.019-.009a1.47 1.47 0 0 0 .657-.885 1.493 1.493 0 0 0-.141-1.118"
    />
  ),
};

export const ExporterMark = (props: {
  kind?: ExporterKind;
  size?: number;
  className?: string;
}) => {
  const Mark = props.kind ? MARKS[props.kind] : undefined;
  const size = props.size ?? 18;
  if (!Mark) {
    return (
      <span
        aria-hidden="true"
        style={{ width: size, height: size }}
        className={twMerge("rounded-md bg-slate-200", props.className)}
      />
    );
  }
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      aria-hidden="true"
      className={twMerge("shrink-0", props.className)}
    >
      <Mark />
    </svg>
  );
};

export const ExporterTile = (props: {
  kind?: ExporterKind;
  size?: "sm" | "md" | "lg";
  className?: string;
}) => {
  const size = props.size ?? "md";
  const tint = getProvider(props.kind)?.tint ?? "#64748B";
  return (
    <span
      aria-hidden="true"
      style={{
        backgroundColor: `color-mix(in srgb, ${tint} 9%, var(--color-white))`,
        borderColor: `color-mix(in srgb, ${tint} 22%, var(--color-slate-200))`,
      }}
      className={twMerge(
        "flex shrink-0 items-center justify-center border",
        size === "sm" && "h-7 w-7 rounded-md",
        size === "md" && "h-9 w-9 rounded-lg",
        size === "lg" && "h-11 w-11 rounded-xl",
        props.className,
      )}
    >
      <ExporterMark
        kind={props.kind}
        size={size === "sm" ? 15 : size === "md" ? 19 : 23}
      />
    </span>
  );
};

export const SignalBadges = (props: {
  signals: ExporterSignal[];
  className?: string;
}) => (
  <span className={twMerge("flex flex-wrap gap-1", props.className)}>
    {props.signals.map((signal) => (
      <span
        key={signal}
        className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600"
      >
        {signal === "logs" ? "Logs" : "Metrics"}
      </span>
    ))}
  </span>
);
