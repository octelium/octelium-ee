import { Breakdown, MiniStat, Panel } from "@/components/Dashboard/components";
import { compact } from "@/components/Dashboard/utils";
import { CompositionBar } from "@/components/ResourceInventory/InventoryTable";
import { useChartColorScheme } from "@/utils/charts/palette";
import { n } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import { Globe2, PowerOff, Telescope } from "lucide-react";
import { useEnterpriseTotals } from "./queries";

const Telemetry = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  useChartColorScheme();

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);

  const exporters = totals.data?.enterprise?.collectorExporter;
  const dns = totals.data?.enterprise?.dnsProvider;

  return (
    <Panel
      icon={Telescope}
      title="Telemetry & DNS"
      description="Where the Cluster ships its logs and metrics, and which DNS zones it can solve challenges in"
      to="/enterprise/collectorexporters"
      toLabel="Collector Exporters"
    >
      <div className="flex flex-col gap-5">
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
          <MiniStat
            label="Exporters"
            value={n(exporters?.totalNumber)}
            icon={Telescope}
            to="/enterprise/collectorexporters"
          />
          <MiniStat
            label="Disabled"
            value={n(exporters?.totalDisabled)}
            tone={n(exporters?.totalDisabled) > 0 ? "warning" : "default"}
            icon={PowerOff}
            to="/enterprise/collectorexporters?isDisabled=true"
          />
          <MiniStat
            label="DNS providers"
            value={n(dns?.totalNumber)}
            icon={Globe2}
            to="/enterprise/dnsproviders"
          />
        </div>

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Breakdown
            title="Exporters by destination"
            note={`${compact(n(exporters?.totalNumber))} configured`}
          >
            <CompositionBar
              segments={[
                { label: "OTLP", value: n(exporters?.totalOTLP) },
                { label: "OTLP HTTP", value: n(exporters?.totalOTLPHTTP) },
                { label: "Clickhouse", value: n(exporters?.totalClickhouse) },
                {
                  label: "Elasticsearch",
                  value: n(exporters?.totalElasticsearch),
                },
                { label: "Logz.io", value: n(exporters?.totalLogzio) },
                { label: "InfluxDB", value: n(exporters?.totalInfluxDB) },
                { label: "Kafka", value: n(exporters?.totalKafka) },
                { label: "Datadog", value: n(exporters?.totalDatadog) },
                { label: "Splunk", value: n(exporters?.totalSplunk) },
                {
                  label: "Azure Monitor",
                  value: n(exporters?.totalAzureMonitor),
                },
                {
                  label: "Azure Data Explorer",
                  value: n(exporters?.totalAzureDataExplorer),
                },
                {
                  label: "Prometheus",
                  value: n(exporters?.totalPrometheusRemoteWrite),
                },
              ]}
              total={n(exporters?.totalNumber)}
            />
          </Breakdown>

          <Breakdown
            title="DNS providers"
            note={`${compact(n(dns?.totalNumber))} configured`}
          >
            <CompositionBar
              segments={[
                { label: "Cloudflare", value: n(dns?.totalCloudflare) },
                { label: "AWS", value: n(dns?.totalAWS) },
                { label: "DigitalOcean", value: n(dns?.totalDigitalOcean) },
                { label: "Google", value: n(dns?.totalGoogle) },
                { label: "Azure", value: n(dns?.totalAzure) },
                { label: "Linode", value: n(dns?.totalLinode) },
                { label: "OVH", value: n(dns?.totalOVH) },
              ]}
              total={n(dns?.totalNumber)}
            />
          </Breakdown>
        </div>
      </div>
    </Panel>
  );
};

export default Telemetry;
