import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Boxes,
  BookKey,
  ClipboardCheck,
  Cpu,
  Building2,
  Folder,
  Inbox,
  LaptopMinimal,
  Layers,
  LucideIcon,
  PanelTop,
  Shield,
  ShieldCheck,
  Telescope,
  Terminal,
  User,
  UserCheck,
} from "lucide-react";
import { Link } from "react-router-dom";
import { Panel } from "./components";
import { useResourceRangeSummary, useResourceSummary } from "./queries";
import { compact, exact } from "./utils";

type Api = "core" | "access" | "enterprise";

type Entry = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  method: string;
  withRange?: boolean;
};

type Group = {
  api: Api;
  label: string;
  icon: LucideIcon;
  to: string;
  entries: Entry[];
};

const GROUPS: Group[] = [
  {
    api: "core",
    label: "Core",
    icon: Cpu,
    to: "/core",
    entries: [
      {
        kind: "User",
        label: "Users",
        to: "/core/users",
        icon: User,
        method: "getUserSummary",
        withRange: true,
      },
      {
        kind: "Session",
        label: "Sessions",
        to: "/core/sessions",
        icon: Terminal,
        method: "getSessionSummary",
        withRange: true,
      },
      {
        kind: "Device",
        label: "Devices",
        to: "/core/devices",
        icon: LaptopMinimal,
        method: "getDeviceSummary",
        withRange: true,
      },
      {
        kind: "Service",
        label: "Services",
        to: "/core/services",
        icon: PanelTop,
        method: "getServiceSummary",
        withRange: true,
      },
      {
        kind: "Namespace",
        label: "Namespaces",
        to: "/core/namespaces",
        icon: Boxes,
        method: "getNamespaceSummary",
      },
      {
        kind: "Policy",
        label: "Policies",
        to: "/core/policies",
        icon: Shield,
        method: "getPolicySummary",
      },
    ],
  },
  {
    api: "access",
    label: "Access",
    icon: UserCheck,
    to: "/access",
    entries: [
      {
        kind: "Request",
        label: "Requests",
        to: "/access/requests",
        icon: Inbox,
        method: "getRequestSummary",
        withRange: true,
      },
      {
        kind: "Review",
        label: "Reviews",
        to: "/access/reviews",
        icon: ClipboardCheck,
        method: "getReviewSummary",
        withRange: true,
      },
      {
        kind: "Policy",
        label: "Policies",
        to: "/access/policies",
        icon: Shield,
        method: "getPolicySummary",
      },
      {
        kind: "Catalog",
        label: "Catalogs",
        to: "/access/catalogs",
        icon: Layers,
        method: "getCatalogSummary",
      },
    ],
  },
  {
    api: "enterprise",
    label: "Enterprise",
    icon: Building2,
    to: "/enterprise",
    entries: [
      {
        kind: "Certificate",
        label: "Certificates",
        to: "/enterprise/certificates",
        icon: ShieldCheck,
        method: "getCertificateSummary",
      },
      {
        kind: "DirectoryProvider",
        label: "Directory Providers",
        to: "/enterprise/directoryproviders",
        icon: Folder,
        method: "getDirectoryProviderSummary",
      },
      {
        kind: "SecretStore",
        label: "Secret Stores",
        to: "/enterprise/secretstores",
        icon: BookKey,
        method: "getSecretStoreSummary",
      },
      {
        kind: "CollectorExporter",
        label: "Collector Exporters",
        to: "/enterprise/collectorexporters",
        icon: Telescope,
        method: "getCollectorExporterSummary",
      },
    ],
  },
];

const Row = (props: { api: Api; entry: Entry; periodMinutes: number }) => {
  const { entry } = props;
  const Icon = entry.icon;

  const total = useResourceSummary<{ totalNumber?: unknown }>({
    api: props.api,
    kind: entry.kind,
    method: entry.method,
    priority: QUERY_PRIORITY.low,
  });

  const created = useResourceRangeSummary<{ totalNumber?: unknown }>({
    api: props.api,
    kind: entry.kind,
    method: entry.method,
    periodMinutes: props.periodMinutes,
    priority: QUERY_PRIORITY.low,
    enabled: !!entry.withRange,
  });

  const createdCount = n(created.data?.totalNumber);

  return (
    <Link
      to={entry.to}
      className="group flex items-center gap-3 rounded-lg border border-transparent px-2.5 py-2 outline-none transition-colors duration-150 hover:border-slate-200 hover:bg-slate-50/70 focus-visible:ring-2 focus-visible:ring-slate-400"
    >
      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
        <Icon size={13} strokeWidth={2.2} />
      </span>

      <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700 group-hover:text-slate-900">
        {entry.label}
      </span>

      {entry.withRange && createdCount > 0 && (
        <span className="shrink-0 rounded-full border border-emerald-200 bg-emerald-50 px-1.5 py-px text-micro font-semibold tabular-nums text-emerald-700">
          +{compact(createdCount)}
        </span>
      )}

      {total.isLoading ? (
        <span className="h-4 w-10 shrink-0 animate-pulse rounded bg-slate-100" />
      ) : (
        <span
          className="shrink-0 text-sm font-bold tabular-nums text-slate-900"
          title={exact(n(total.data?.totalNumber))}
        >
          {compact(n(total.data?.totalNumber))}
        </span>
      )}
    </Link>
  );
};

const Inventory = (props: { periodMinutes: number }) => {
  const rangeLabel = periodLabel(props.periodMinutes);

  return (
    <Panel
      icon={Layers}
      title="Cluster inventory"
      description={`Resources across every API · green badges show what was created in the last ${rangeLabel}`}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {GROUPS.map((group) => {
          const GroupIcon = group.icon;
          return (
            <section
              key={group.api}
              className="flex flex-col overflow-hidden rounded-xl border border-slate-200 bg-slate-50/40"
            >
              <header className="flex items-center justify-between gap-2 border-b border-slate-200 px-3.5 py-2.5">
                <span className="flex min-w-0 items-center gap-2">
                  <GroupIcon
                    size={13}
                    strokeWidth={2.3}
                    className="shrink-0 text-slate-500"
                  />
                  <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-600">
                    {group.label}
                  </span>
                </span>
                <Link
                  to={group.to}
                  className="shrink-0 rounded px-1.5 py-0.5 text-micro font-semibold text-slate-500 outline-none transition-colors duration-150 hover:bg-white hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
                >
                  View all
                </Link>
              </header>

              <div className="flex flex-col gap-0.5 p-1.5">
                {group.entries.map((entry) => (
                  <Row
                    key={`${group.api}-${entry.kind}`}
                    api={group.api}
                    entry={entry}
                    periodMinutes={props.periodMinutes}
                  />
                ))}
              </div>
            </section>
          );
        })}
      </div>
    </Panel>
  );
};

export default Inventory;
