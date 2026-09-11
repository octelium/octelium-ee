import { getClientVisibilityEnterprise } from "@/utils/client";
import { useQueries } from "@tanstack/react-query";
import {
  BookKey,
  Crown,
  Folder,
  Globe2,
  KeyRound,
  LucideIcon,
  ShieldCheck,
  Telescope,
  UserRound,
  UsersRound,
} from "lucide-react";
import {
  AttentionItem,
  InventoryRow,
  InventoryTable,
  Segment,
} from "@/components/ResourceInventory/InventoryTable";

const num = (value: unknown) => Number(value ?? 0);

type SummaryEntry = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  fetch: () => Promise<any>;
  derive: (data: any) => { segments: Segment[]; attention: AttentionItem[] };
};

const STATUS = {
  good: "var(--color-emerald-600)",
  warning: "var(--color-amber-600)",
  serious: "var(--color-orange-600)",
  critical: "var(--color-red-600)",
} as const;

const client = () => getClientVisibilityEnterprise();

const ENTRIES: SummaryEntry[] = [
  {
    kind: "Certificate",
    label: "Certificates",
    to: "/enterprise/certificates",
    icon: ShieldCheck,
    fetch: async () => (await client().getCertificateSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Managed", value: num(d?.totalManaged) },
        { label: "Manual", value: num(d?.totalManual) },
      ],
      attention: [
        { label: "Expired", value: num(d?.totalExpired), color: STATUS.critical },
        { label: "Expiring soon", value: num(d?.totalExpiringSoon), color: STATUS.warning },
        { label: "Issuance failed", value: num(d?.totalIssuanceFailed), color: STATUS.critical },
      ],
    }),
  },
  {
    kind: "CertificateIssuer",
    label: "Certificate Issuers",
    to: "/enterprise/certificateissuers",
    icon: Crown,
    fetch: async () => (await client().getCertificateIssuerSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Ready", value: num(d?.totalReady), color: STATUS.good },
        { label: "Preparing", value: num(d?.totalPreparing), color: STATUS.warning },
        { label: "Not ready", value: num(d?.totalNotReady), color: STATUS.critical },
      ],
      attention: [
        { label: "Not ready", value: num(d?.totalNotReady), color: STATUS.critical },
      ],
    }),
  },
  {
    kind: "DirectoryProvider",
    label: "Directory Providers",
    to: "/enterprise/directoryproviders",
    icon: Folder,
    fetch: async () => (await client().getDirectoryProviderSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "SCIM", value: num(d?.totalSCIM) },
        { label: "Google Workspace", value: num(d?.totalGoogleWorkspace) },
        { label: "Keycloak", value: num(d?.totalKeycloak) },
      ],
      attention: [
        { label: "Disabled", value: num(d?.totalDisabled) },
        {
          label: "Sync failed",
          value: num(d?.totalSynchronizationFailed),
          color: STATUS.critical,
        },
      ],
    }),
  },
  {
    kind: "DirectoryProviderUser",
    label: "Directory Users",
    to: "/enterprise/directoryproviders",
    icon: UserRound,
    fetch: async () =>
      (await client().getDirectoryProviderUserSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Linked", value: num(d?.totalUser), color: STATUS.good },
      ],
      attention: [],
    }),
  },
  {
    kind: "DirectoryProviderGroup",
    label: "Directory Groups",
    to: "/enterprise/directoryproviders",
    icon: UsersRound,
    fetch: async () =>
      (await client().getDirectoryProviderGroupSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Linked", value: num(d?.totalGroup), color: STATUS.good },
      ],
      attention: [],
    }),
  },
  {
    kind: "CollectorExporter",
    label: "Collector Exporters",
    to: "/enterprise/collectorexporters",
    icon: Telescope,
    fetch: async () => (await client().getCollectorExporterSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "OTLP", value: num(d?.totalOTLP) },
        { label: "OTLP HTTP", value: num(d?.totalOTLPHTTP) },
        { label: "Clickhouse", value: num(d?.totalClickhouse) },
        { label: "Elasticsearch", value: num(d?.totalElasticsearch) },
        { label: "Logz.io", value: num(d?.totalLogzio) },
        { label: "InfluxDB", value: num(d?.totalInfluxDB) },
        { label: "Kafka", value: num(d?.totalKafka) },
        { label: "Datadog", value: num(d?.totalDatadog) },
        { label: "Splunk", value: num(d?.totalSplunk) },
        { label: "Azure Monitor", value: num(d?.totalAzureMonitor) },
        { label: "Azure Data Explorer", value: num(d?.totalAzureDataExplorer) },
        {
          label: "Prometheus Remote Write",
          value: num(d?.totalPrometheusRemoteWrite),
        },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "DNSProvider",
    label: "DNS Providers",
    to: "/enterprise/dnsproviders",
    icon: Globe2,
    fetch: async () => (await client().getDNSProviderSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Cloudflare", value: num(d?.totalCloudflare) },
        { label: "AWS", value: num(d?.totalAWS) },
        { label: "DigitalOcean", value: num(d?.totalDigitalOcean) },
        { label: "Google", value: num(d?.totalGoogle) },
        { label: "Azure", value: num(d?.totalAzure) },
        { label: "Linode", value: num(d?.totalLinode) },
        { label: "OVH", value: num(d?.totalOVH) },
      ],
      attention: [],
    }),
  },
  {
    kind: "SecretStore",
    label: "Secret Stores",
    to: "/enterprise/secretstores",
    icon: BookKey,
    fetch: async () => (await client().getSecretStoreSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Azure Key Vault", value: num(d?.totalAzureKeyVault) },
        { label: "HashiCorp Vault", value: num(d?.totalHashicorpVault) },
        { label: "GCP KMS", value: num(d?.totalGCPKMS) },
        { label: "AWS KMS", value: num(d?.totalAWSKMS) },
        { label: "Kubernetes", value: num(d?.totalKubernetes) },
      ],
      attention: [
        {
          label: "Sync failed",
          value: num(d?.totalSynchronizationFailed),
          color: STATUS.critical,
        },
      ],
    }),
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/enterprise/secrets",
    icon: KeyRound,
    fetch: async () => (await client().getSecretSummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
];

const EnterpriseResourceInventory = () => {
  const results = useQueries({
    queries: ENTRIES.map((entry) => ({
      queryKey: ["visibility", "enterprise", "summary", entry.kind],
      queryFn: entry.fetch,
    })),
  });

  const rows: InventoryRow[] = ENTRIES.map((entry, index) => {
    const result = results[index];
    const derived = entry.derive(result.data);
    return {
      kind: entry.kind,
      label: entry.label,
      to: entry.to,
      icon: entry.icon,
      total: num((result.data as { totalNumber?: unknown })?.totalNumber),
      segments: derived.segments,
      attention: derived.attention.filter((item) => item.value > 0),
      isLoading: result.isLoading,
    };
  });

  return (
    <InventoryTable title="Enterprise inventory" unitLabel="resources" rows={rows} />
  );
};

export default EnterpriseResourceInventory;
