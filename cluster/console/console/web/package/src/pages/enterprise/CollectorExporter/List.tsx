import { CollectorExporter } from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel, ResourceListLabelWrap } from "@/components/ResourceList";
import { match } from "ts-pattern";

export const getType = (item: CollectorExporter): string =>
  match(item.spec?.type.oneofKind)
    .with("otlp", () => "OTLP")
    .with("otlpHTTP", () => "OTLP HTTP")
    .with("elasticsearch", () => "Elasticsearch")
    .with("prometheusRemoteWrite", () => "Prometheus Remote Write")
    .with("datadog", () => "Datadog")
    .with("splunk", () => "Splunk")
    .with("kafka", () => "Kafka")
    .with("influxDB", () => "InfluxDB")
    .with("clickhouse", () => "ClickHouse")
    .with("logzio", () => "Logz.io")
    .with("azureMonitor", () => "Azure Monitor")
    .with("azureDataExplorer", () => "Azure Data Explorer")
    .otherwise(() => "Not configured");

export const getDestination = (item: CollectorExporter): string | undefined =>
  match(item.spec?.type)
    .with({ oneofKind: "otlp" }, ({ otlp }) => otlp.endpoint)
    .with({ oneofKind: "otlpHTTP" }, ({ otlpHTTP }) => otlpHTTP.endpoint)
    .with({ oneofKind: "prometheusRemoteWrite" }, ({ prometheusRemoteWrite }) => prometheusRemoteWrite.endpoint)
    .with({ oneofKind: "clickhouse" }, ({ clickhouse }) => clickhouse.endpoint)
    .with({ oneofKind: "elasticsearch" }, ({ elasticsearch }) => elasticsearch.endpoint || elasticsearch.endpoints[0] || elasticsearch.cloudID)
    .with({ oneofKind: "logzio" }, ({ logzio }) => logzio.endpoint || logzio.region)
    .with({ oneofKind: "influxDB" }, ({ influxDB }) => influxDB.endpoint)
    .with({ oneofKind: "kafka" }, ({ kafka }) => kafka.brokers[0])
    .with({ oneofKind: "datadog" }, ({ datadog }) => datadog.api?.site)
    .with({ oneofKind: "splunk" }, ({ splunk }) => splunk.endpoint)
    .with({ oneofKind: "azureMonitor" }, ({ azureMonitor }) => azureMonitor.endpoint || "Azure Monitor")
    .with({ oneofKind: "azureDataExplorer" }, ({ azureDataExplorer }) => azureDataExplorer.clusterURI)
    .otherwise(() => undefined) || undefined;

export const LabelComponent = (props: { item: CollectorExporter }) => {
  const destination = getDestination(props.item);
  return <ResourceListLabelWrap><ResourceListLabel label="Type">{getType(props.item)}</ResourceListLabel>{props.item.spec?.isDisabled && <ResourceListLabel>Disabled</ResourceListLabel>}{destination && <ResourceListLabel label="Destination">{destination}</ResourceListLabel>}</ResourceListLabelWrap>;
};
