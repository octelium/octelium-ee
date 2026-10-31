import { CollectorExporter } from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import CopyText from "@/components/CopyText";
import Label from "@/components/Label";
import { ResourceListLabel } from "@/components/ResourceList";
import EditItemWrap from "@/components/ResourceLayout/EditItemWrap";
import { ResourceMainInfo } from "@/pages/utils/types";
import { cloneResource } from "@/utils/pb";
import { twMerge } from "@/lib/utils";
import { getDestination, getType } from "./List";
import { OverviewNotice } from "@/components/ResourceLayout/overview";
import { Ban, KeyRound, Radio, Send, SlidersHorizontal } from "lucide-react";

const secretLabel = (name: string) => <ResourceListLabel itemRef={ObjectReference.create({ apiVersion: "enterprise/v1", kind: "Secret", name })} />;

const getAuthInfo = (item: CollectorExporter) => {
  const type = item.spec?.type;
  if (!type?.oneofKind) return undefined;
  const fromSecret = (value?: { type: { oneofKind?: string; fromSecret?: string } }) => value?.type.oneofKind === "fromSecret" ? value.type.fromSecret : undefined;
  if (type.oneofKind === "otlp" || type.oneofKind === "otlpHTTP" || type.oneofKind === "prometheusRemoteWrite") {
    const auth = type.oneofKind === "otlp" ? type.otlp.auth : type.oneofKind === "otlpHTTP" ? type.otlpHTTP.auth : type.prometheusRemoteWrite.auth;
    if (auth?.type.oneofKind === "bearer") return { method: "Bearer", secret: fromSecret(auth.type.bearer) };
    if (auth?.type.oneofKind === "basic") return { method: "Basic", secret: fromSecret(auth.type.basic.password) };
    if (auth?.type.oneofKind === "custom") return { method: `Custom header (${auth.type.custom.header || "unnamed"})`, secret: fromSecret(auth.type.custom.value) };
  }
  if (type.oneofKind === "clickhouse") return { method: "Username and password", secret: fromSecret(type.clickhouse.password) };
  if (type.oneofKind === "elasticsearch") {
    const auth = type.elasticsearch.auth?.type;
    if (auth?.oneofKind === "apiKey") return { method: "API key", secret: fromSecret(auth.apiKey) };
    if (auth?.oneofKind === "basic") return { method: "Basic", secret: fromSecret(auth.basic.password) };
  }
  if (type.oneofKind === "logzio") return { method: "Token", secret: fromSecret(type.logzio.token) };
  if (type.oneofKind === "influxDB") return { method: "Token", secret: fromSecret(type.influxDB.token) };
  if (type.oneofKind === "kafka" && type.kafka.auth?.type.oneofKind === "sasl") return { method: "SASL", secret: fromSecret(type.kafka.auth.type.sasl.password) };
  if (type.oneofKind === "datadog") return { method: "API key", secret: fromSecret(type.datadog.api?.key) };
  if (type.oneofKind === "splunk") return { method: "Token", secret: fromSecret(type.splunk.token) };
  if (type.oneofKind === "azureMonitor") return { method: "Azure credential", secret: fromSecret(type.azureMonitor.connectionString) || fromSecret(type.azureMonitor.instrumentationKey) };
  if (type.oneofKind === "azureDataExplorer" && type.azureDataExplorer.auth?.type.oneofKind === "servicePrincipal") return { method: "Service principal", secret: fromSecret(type.azureDataExplorer.auth.type.servicePrincipal.applicationKey) };
  return undefined;
};

export default (_props: { item: CollectorExporter }) => <></>;

export const MainInfo = (props: { item: CollectorExporter }): ResourceMainInfo => {
  const { item } = props;
  const destination = getDestination(item);
  const type = item.spec?.type;
  const auth = getAuthInfo(item);
  const items: NonNullable<ResourceMainInfo["items"]> = [
    { label: "Type", primary: true, icon: Radio, value: <Label>{getType(item)}</Label> },
    { label: "Authentication", primary: true, icon: KeyRound, value: auth?.method ?? <span className="text-slate-500">None</span> },
    ...(auth?.secret ? [{ label: "Credential", primary: true, icon: KeyRound, value: secretLabel(auth.secret) }] : []),
    ...(destination ? [{ label: "Destination", group: "Destination", value: <CopyText value={destination} />, span: "full" as const }] : []),
  ];

  const settings: NonNullable<ResourceMainInfo["items"]> = [];
  if (type?.oneofKind === "kafka") settings.push({ label: "Brokers", value: type.kafka.brokers.join(", ") || "Not configured", span: "full" }, { label: "Logs topic", value: type.kafka.logs?.topic || "Not configured" }, { label: "Metrics topic", value: type.kafka.metrics?.topic || "Not configured" });
  if (type?.oneofKind === "elasticsearch") settings.push({ label: "Logs index", value: type.elasticsearch.logsIndex || "Default" }, { label: "Metrics index", value: type.elasticsearch.metricsIndex || "Default" }, { label: "Configured endpoints", value: String(type.elasticsearch.endpoints.length || (type.elasticsearch.endpoint ? 1 : 0)) });
  if (type?.oneofKind === "influxDB") settings.push({ label: "Organization", value: type.influxDB.org || "Not configured" }, { label: "Bucket", value: type.influxDB.bucket || "Not configured" });
  if (type?.oneofKind === "clickhouse") settings.push({ label: "Database", value: type.clickhouse.database || "Default" }, { label: "Logs table", value: type.clickhouse.logsTableName || "Default" });
  if (type?.oneofKind === "datadog") settings.push({ label: "Hostname", value: type.datadog.hostname || "Automatically detected" }, { label: "Site", value: type.datadog.api?.site || "Default" });
  if (type?.oneofKind === "splunk") settings.push({ label: "Index", value: type.splunk.index || "Default" }, { label: "Source type", value: type.splunk.sourceType || "Default" });
  if (type?.oneofKind === "azureDataExplorer") settings.push({ label: "Database", value: type.azureDataExplorer.database || "Not configured" }, { label: "Logs table", value: type.azureDataExplorer.logsTable || "Not configured" }, { label: "Metrics table", value: type.azureDataExplorer.metricsTable || "Not configured" });

  return {
    header: item.spec?.isDisabled ? (
      <div className="border-b border-slate-200 bg-white px-4 py-3">
        <OverviewNotice tone="danger" icon={Ban} title="Disabled.">
          No logs or metrics are exported to this destination.
        </OverviewNotice>
      </div>
    ) : undefined,
    groupOrder: [
      { title: "Destination", icon: Send },
      { title: "Settings", icon: SlidersHorizontal },
    ],
    items: [...items, ...settings.map((entry) => ({ ...entry, group: "Settings" }))],
  };
};
