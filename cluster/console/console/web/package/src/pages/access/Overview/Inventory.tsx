import { GetClusterSummaryResponse } from "@/apis/visibilityv1/visibilityv1";
import {
  InventoryEntry,
  InventoryRail,
  Panel,
} from "@/components/Dashboard/components";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  ClipboardCheck,
  Inbox,
  KeyRound,
  Layers,
  Link2,
  LucideIcon,
  Plug,
  Send,
  Shield,
} from "lucide-react";
import { useAccessCreated, useAccessTotals } from "./queries";

type Access = GetClusterSummaryResponse["access"];

type Row = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  pick: (access: Access) => { totalNumber?: unknown } | undefined;
  attention?: (access: Access) => InventoryEntry["attention"];
};

const ROWS: Row[] = [
  {
    kind: "Request",
    label: "Requests",
    to: "/access/requests",
    icon: Inbox,
    pick: (access) => access?.request,
    attention: (access) => [
      { label: "pending ", value: n(access?.request?.totalPending) },
      {
        label: "overdue ",
        value: n(access?.request?.totalDeadlinePassed),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Review",
    label: "Reviews",
    to: "/access/reviews",
    icon: ClipboardCheck,
    pick: (access) => access?.review,
    attention: (access) => [
      { label: "pending ", value: n(access?.review?.totalPending) },
    ],
  },
  {
    kind: "Policy",
    label: "Policies",
    to: "/access/policies",
    icon: Shield,
    pick: (access) => access?.policy,
    attention: (access) => [
      { label: "disabled ", value: n(access?.policy?.totalDisabled) },
    ],
  },
  {
    kind: "Catalog",
    label: "Catalogs",
    to: "/access/catalogs",
    icon: Layers,
    pick: (access) => access?.catalog,
  },
  {
    kind: "Integration",
    label: "Integrations",
    to: "/access/integrations",
    icon: Plug,
    pick: (access) => access?.integration,
    attention: (access) => [
      { label: "disabled ", value: n(access?.integration?.totalDisabled) },
      {
        label: "failing ",
        value: n(access?.integration?.totalError),
        tone: "critical",
      },
    ],
  },
  {
    kind: "IntegrationIdentity",
    label: "Integration Identities",
    to: "/access/integrationidentities",
    icon: Link2,
    pick: (access) => access?.integrationIdentity,
  },
  {
    kind: "IntegrationBinding",
    label: "Integration Bindings",
    to: "/access/integrationbindings",
    icon: Send,
    pick: (access) => access?.integrationBinding,
    attention: (access) => [
      {
        label: "out of date ",
        value: n(access?.integrationBinding?.totalOutOfDate),
        tone: "warning",
      },
      {
        label: "failing ",
        value: n(access?.integrationBinding?.totalFailing),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/access/secrets",
    icon: KeyRound,
    pick: (access) => access?.secret,
  },
];

const Inventory = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useAccessTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useAccessCreated(periodMinutes, QUERY_PRIORITY.high);

  const entries: InventoryEntry[] = ROWS.map((row) => ({
    kind: row.kind,
    label: row.label,
    to: row.to,
    icon: row.icon,
    total: n(row.pick(totals.data?.access)?.totalNumber),
    created: n(row.pick(created.data?.access)?.totalNumber),
    attention: row.attention?.(totals.data?.access),
  }));

  const unavailable = totals.data?.unavailables ?? [];

  return (
    <Panel
      icon={Layers}
      title="Access inventory"
      description={`Every access resource kind · green badges show what was created in the last ${rangeLabel}`}
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
