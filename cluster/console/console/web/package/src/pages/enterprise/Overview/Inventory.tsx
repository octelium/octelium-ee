import { GetClusterSummaryResponse } from "@/apis/visibilityv1/visibilityv1";
import {
  InventoryEntry,
  InventoryRail,
  Panel,
} from "@/components/Dashboard/components";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  BookKey,
  Crown,
  Folder,
  Globe2,
  KeyRound,
  LaptopMinimal,
  Layers,
  LucideIcon,
  ShieldCheck,
  Telescope,
  UserRound,
  UsersRound,
} from "lucide-react";
import { useEnterpriseCreated, useEnterpriseTotals } from "./queries";

type Enterprise = GetClusterSummaryResponse["enterprise"];

type Row = {
  kind: string;
  label: string;
  to?: string;
  icon: LucideIcon;
  pick: (enterprise: Enterprise) => { totalNumber?: unknown } | undefined;
  attention?: (enterprise: Enterprise) => InventoryEntry["attention"];
};

const ROWS: Row[] = [
  {
    kind: "Certificate",
    label: "Certificates",
    to: "/enterprise/certificates",
    icon: ShieldCheck,
    pick: (enterprise) => enterprise?.certificate,
    attention: (enterprise) => [
      {
        label: "expired ",
        value: n(enterprise?.certificate?.totalExpired),
        tone: "critical",
      },
      {
        label: "expiring ",
        value: n(enterprise?.certificate?.totalExpiringSoon),
      },
      {
        label: "failed ",
        value: n(enterprise?.certificate?.totalIssuanceFailed),
        tone: "critical",
      },
    ],
  },
  {
    kind: "CertificateIssuer",
    label: "Certificate issuers",
    to: "/enterprise/certificateissuers",
    icon: Crown,
    pick: (enterprise) => enterprise?.certificateIssuer,
    attention: (enterprise) => [
      {
        label: "not ready ",
        value: n(enterprise?.certificateIssuer?.totalNotReady),
        tone: "critical",
      },
    ],
  },
  {
    kind: "DirectoryProvider",
    label: "Directory providers",
    to: "/enterprise/directoryproviders",
    icon: Folder,
    pick: (enterprise) => enterprise?.directoryProvider,
    attention: (enterprise) => [
      {
        label: "sync failed ",
        value: n(enterprise?.directoryProvider?.totalSynchronizationFailed),
        tone: "critical",
      },
      {
        label: "disabled ",
        value: n(enterprise?.directoryProvider?.totalDisabled),
      },
    ],
  },
  {
    kind: "DirectoryProviderUser",
    label: "Directory users",
    to: "/enterprise/directoryproviders",
    icon: UserRound,
    pick: (enterprise) => enterprise?.directoryProviderUser,
  },
  {
    kind: "DirectoryProviderGroup",
    label: "Directory groups",
    to: "/enterprise/directoryproviders",
    icon: UsersRound,
    pick: (enterprise) => enterprise?.directoryProviderGroup,
  },
  {
    kind: "SecretStore",
    label: "Secret stores",
    to: "/enterprise/secretstores",
    icon: BookKey,
    pick: (enterprise) => enterprise?.secretStore,
    attention: (enterprise) => [
      {
        label: "sync failed ",
        value: n(enterprise?.secretStore?.totalSynchronizationFailed),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/enterprise/secrets",
    icon: KeyRound,
    pick: (enterprise) => enterprise?.secret,
  },
  {
    kind: "DeviceManager",
    label: "Device managers",
    icon: LaptopMinimal,
    pick: (enterprise) => enterprise?.deviceManager,
    attention: (enterprise) => [
      {
        label: "errored ",
        value: n(enterprise?.deviceManager?.totalError),
        tone: "critical",
      },
      {
        label: "degraded ",
        value: n(enterprise?.deviceManager?.totalDegraded),
      },
    ],
  },
  {
    kind: "CollectorExporter",
    label: "Collector exporters",
    to: "/enterprise/collectorexporters",
    icon: Telescope,
    pick: (enterprise) => enterprise?.collectorExporter,
    attention: (enterprise) => [
      {
        label: "disabled ",
        value: n(enterprise?.collectorExporter?.totalDisabled),
      },
    ],
  },
  {
    kind: "DNSProvider",
    label: "DNS providers",
    to: "/enterprise/dnsproviders",
    icon: Globe2,
    pick: (enterprise) => enterprise?.dnsProvider,
  },
];

const Inventory = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useEnterpriseTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useEnterpriseCreated(periodMinutes, QUERY_PRIORITY.high);

  const entries: InventoryEntry[] = ROWS.map((row) => ({
    kind: row.kind,
    label: row.label,
    to: row.to,
    icon: row.icon,
    total: n(row.pick(totals.data?.enterprise)?.totalNumber),
    created: n(row.pick(created.data?.enterprise)?.totalNumber),
    attention: row.attention?.(totals.data?.enterprise),
  }));

  const unavailable = totals.data?.unavailables ?? [];

  return (
    <Panel
      icon={Layers}
      title="Enterprise inventory"
      description={`Every enterprise resource kind · green badges show what was created in the last ${rangeLabel}`}
    >
      <div className="flex flex-col gap-3">
        <InventoryRail
          entries={entries}
          isLoading={totals.isLoading}
          rangeLabel={rangeLabel}
        />

        {unavailable.length > 0 && (
          <p role="alert" className="text-micro font-semibold text-amber-700">
            {unavailable.length} summar
            {unavailable.length === 1 ? "y" : "ies"} could not be loaded, so
            some counts may be missing.
          </p>
        )}
      </div>
    </Panel>
  );
};

export default Inventory;
