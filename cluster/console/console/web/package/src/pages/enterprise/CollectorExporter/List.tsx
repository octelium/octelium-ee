import { CollectorExporter } from "@/apis/enterprisev1/enterprisev1";
import { GetCollectorExporterSummaryResponse } from "@/apis/visibilityv1/enterprise/venterprisev1";
import { ResourceListLabel, ResourceListLabelWrap } from "@/components/ResourceList";
import { SummaryItemCount, SummaryItemCountWrap, SummaryNoItems } from "@/components/Summary";
import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQuery } from "@tanstack/react-query";
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

const exporterTypes: Array<[keyof GetCollectorExporterSummaryResponse, string, string]> = [
  ["totalOTLP", "OTLP", "OTLP"], ["totalOTLPHTTP", "OTLP HTTP", "OTLP_HTTP"], ["totalClickhouse", "ClickHouse", "CLICKHOUSE"],
  ["totalElasticsearch", "Elasticsearch", "ELASTICSEARCH"], ["totalLogzio", "Logz.io", "LOGZIO"], ["totalInfluxDB", "InfluxDB", "INFLUXDB"],
  ["totalKafka", "Kafka", "KAFKA"], ["totalDatadog", "Datadog", "DATADOG"], ["totalSplunk", "Splunk", "SPLUNK"],
  ["totalAzureMonitor", "Azure Monitor", "AZURE_MONITOR"], ["totalAzureDataExplorer", "Azure Data Explorer", "AZURE_DATA_EXPLORER"],
  ["totalPrometheusRemoteWrite", "Prometheus", "PROMETHEUS_REMOTE_WRITE"],
];

const DoSummary = ({ resp }: { resp: GetCollectorExporterSummaryResponse }) => <SummaryItemCountWrap>
  <SummaryItemCount count={resp.totalNumber} to="/enterprise/collectorexporters">Total</SummaryItemCount>
  <SummaryItemCount count={resp.totalDisabled} to="/enterprise/collectorexporters?isDisabled=true">Disabled</SummaryItemCount>
  {exporterTypes.map(([field, label, type]) => <SummaryItemCount key={field} count={resp[field]} to={`/enterprise/collectorexporters?type=${type}`}>{label}</SummaryItemCount>)}
</SummaryItemCountWrap>;

export const Summary = ({ showNoItems }: { showNoItems?: boolean }) => {
  const query = useQuery({ queryKey: ["visibility", "enterprise", "summary", "CollectorExporter"], queryFn: async () => (await getClientVisibilityEnterprise().getCollectorExporterSummary({})).response });
  if (!query.data) return null;
  return query.data.totalNumber > 0 ? <DoSummary resp={query.data} /> : showNoItems ? <SummaryNoItems /> : null;
};
