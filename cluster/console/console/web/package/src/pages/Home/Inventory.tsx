import { GetClusterSummaryResponse } from "@/apis/visibilityv1/visibilityv1";
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
import { Panel } from "@/components/Dashboard/components";
import { useClusterSummary } from "@/components/Dashboard/queries";
import { compact, exact } from "@/components/Dashboard/utils";

type Api = "core" | "access" | "enterprise";

type Summary = { totalNumber?: unknown } | undefined;

type Entry = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  pick: (summary?: GetClusterSummaryResponse) => Summary;
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
        pick: (summary) => summary?.core?.user,
        withRange: true,
      },
      {
        kind: "Session",
        label: "Sessions",
        to: "/core/sessions",
        icon: Terminal,
        pick: (summary) => summary?.core?.session,
        withRange: true,
      },
      {
        kind: "Device",
        label: "Devices",
        to: "/core/devices",
        icon: LaptopMinimal,
        pick: (summary) => summary?.core?.device,
        withRange: true,
      },
      {
        kind: "Service",
        label: "Services",
        to: "/core/services",
        icon: PanelTop,
        pick: (summary) => summary?.core?.service,
        withRange: true,
      },
      {
        kind: "Namespace",
        label: "Namespaces",
        to: "/core/namespaces",
        icon: Boxes,
        pick: (summary) => summary?.core?.namespace,
      },
      {
        kind: "Policy",
        label: "Policies",
        to: "/core/policies",
        icon: Shield,
        pick: (summary) => summary?.core?.policy,
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
        pick: (summary) => summary?.access?.request,
        withRange: true,
      },
      {
        kind: "Review",
        label: "Reviews",
        to: "/access/reviews",
        icon: ClipboardCheck,
        pick: (summary) => summary?.access?.review,
        withRange: true,
      },
      {
        kind: "Policy",
        label: "Policies",
        to: "/access/policies",
        icon: Shield,
        pick: (summary) => summary?.access?.policy,
      },
      {
        kind: "Catalog",
        label: "Catalogs",
        to: "/access/catalogs",
        icon: Layers,
        pick: (summary) => summary?.access?.catalog,
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
        pick: (summary) => summary?.enterprise?.certificate,
      },
      {
        kind: "DirectoryProvider",
        label: "Directory Providers",
        to: "/enterprise/directoryproviders",
        icon: Folder,
        pick: (summary) => summary?.enterprise?.directoryProvider,
      },
      {
        kind: "SecretStore",
        label: "Secret Stores",
        to: "/enterprise/secretstores",
        icon: BookKey,
        pick: (summary) => summary?.enterprise?.secretStore,
      },
      {
        kind: "CollectorExporter",
        label: "Collector Exporters",
        to: "/enterprise/collectorexporters",
        icon: Telescope,
        pick: (summary) => summary?.enterprise?.collectorExporter,
      },
    ],
  },
];

const Row = (props: {
  entry: Entry;
  total?: GetClusterSummaryResponse;
  created?: GetClusterSummaryResponse;
  isLoading: boolean;
}) => {
  const { entry } = props;
  const Icon = entry.icon;

  const totalCount = n(entry.pick(props.total)?.totalNumber);
  const createdCount = entry.withRange
    ? n(entry.pick(props.created)?.totalNumber)
    : 0;

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

      {createdCount > 0 && (
        <span className="shrink-0 rounded-full border border-emerald-200 bg-emerald-50 px-1.5 py-px text-micro font-semibold tabular-nums text-emerald-700">
          +{compact(createdCount)}
        </span>
      )}

      {props.isLoading ? (
        <span className="h-4 w-10 shrink-0 animate-pulse rounded bg-slate-100" />
      ) : (
        <span
          className="shrink-0 text-sm font-bold tabular-nums text-slate-900"
          title={exact(totalCount)}
        >
          {compact(totalCount)}
        </span>
      )}
    </Link>
  );
};

const Inventory = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const cluster = useClusterSummary(
    "total",
    periodMinutes,
    QUERY_PRIORITY.critical,
  );
  const clusterRange = useClusterSummary(
    "range",
    periodMinutes,
    QUERY_PRIORITY.high,
  );

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
                    entry={entry}
                    total={cluster.data}
                    created={clusterRange.data}
                    isLoading={cluster.isLoading}
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
