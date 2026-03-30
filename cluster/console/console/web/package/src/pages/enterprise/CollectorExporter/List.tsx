import { CollectorExporter } from "@/apis/enterprisev1/enterprisev1";
import { ResourceListLabel } from "@/components/ResourceList";

import { getDomain } from "@/utils";
import { match } from "ts-pattern";

export const getType = (svc: CollectorExporter): string => {
  return match(svc.spec?.type.oneofKind)
    .with("otlp", () => "OTLP")
    .with(`otlpHTTP`, () => "OTLP HTTP")
    .with(`elasticsearch`, () => "Elasticsearch")
    .with(`prometheusRemoteWrite`, () => "Prometheus Remote Write")
    .with(`datadog`, () => "Datadog")
    .with(`splunk`, () => "Splunk")
    .with(`kafka`, () => "Kafka")
    .with(`influxDB`, () => "InfluxDB")
    .with("clickhouse", () => "Clickhouse")
    .with(`logzio`, () => "Logzio")
    .with(`azureMonitor`, () => "Azure Monitor")
    .with(`azureDataExplorer`, () => "Azure Data Explorer")
    .otherwise(() => "");
};

const ItemDetails = (props: { item: CollectorExporter; domain: string }) => {
  const { item } = props;
  const md = item.metadata!;

  return <div></div>;
};

export const LabelComponent = (props: { item: CollectorExporter }) => {
  const { item } = props;

  return (
    <div className="w-full mt-1 flex flex-row">
      <ResourceListLabel label="Type">{getType(item)}</ResourceListLabel>
    </div>
  );
};

export const ExtraComponent = (props: { item: CollectorExporter }) => {
  const { item } = props;
  const domain = getDomain();
  return <ItemDetails item={item} domain={domain} />;
};
