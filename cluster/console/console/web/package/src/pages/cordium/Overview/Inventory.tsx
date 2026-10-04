import { GetClusterSummaryResponse } from "@/apis/visibilityv1/visibilityv1";
import {
  InventoryEntry,
  InventoryRail,
  Panel,
} from "@/components/Dashboard/components";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Container,
  GitBranch,
  Globe,
  KeyRound,
  KeySquare,
  Layers,
  LayoutTemplate,
  LucideIcon,
  UsersRound,
} from "lucide-react";
import { useCordiumCreated, useCordiumTotals } from "./queries";

type Cordium = GetClusterSummaryResponse["cordium"];

type Row = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  pick: (cordium: Cordium) => { totalNumber?: unknown } | undefined;
  attention?: (cordium: Cordium) => InventoryEntry["attention"];
};

const ROWS: Row[] = [
  {
    kind: "Workspace",
    label: "Workspaces",
    to: "/cordium/workspaces",
    icon: Container,
    pick: (cordium) => cordium?.workspace,
    attention: (cordium) => [
      {
        label: "failed ",
        value: n(cordium?.workspace?.totalFailed),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Template",
    label: "Templates",
    to: "/cordium/templates",
    icon: LayoutTemplate,
    pick: (cordium) => cordium?.template,
    attention: (cordium) => [
      {
        label: "never built ",
        value: n(cordium?.template?.totalNeverBuilt),
        tone: "warning",
      },
    ],
  },
  {
    kind: "Space",
    label: "Spaces",
    to: "/cordium/spaces",
    icon: Layers,
    pick: (cordium) => cordium?.space,
  },
  {
    kind: "Membership",
    label: "Memberships",
    to: "/cordium/memberships",
    icon: UsersRound,
    pick: (cordium) => cordium?.membership,
  },
  {
    kind: "GitProvider",
    label: "Git Providers",
    to: "/cordium/gitproviders",
    icon: GitBranch,
    pick: (cordium) => cordium?.gitProvider,
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/cordium/secrets",
    icon: KeyRound,
    pick: (cordium) => cordium?.secret,
  },
  {
    kind: "UserSecret",
    label: "User Secrets",
    to: "/cordium/usersecrets",
    icon: KeySquare,
    pick: (cordium) => cordium?.userSecret,
  },
  {
    kind: "Region",
    label: "Regions",
    to: "/cordium/regions",
    icon: Globe,
    pick: (cordium) => cordium?.region,
  },
];

const Inventory = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useCordiumTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCordiumCreated(periodMinutes, QUERY_PRIORITY.high);

  const entries: InventoryEntry[] = ROWS.map((row) => ({
    kind: row.kind,
    label: row.label,
    to: row.to,
    icon: row.icon,
    total: n(row.pick(totals.data?.cordium)?.totalNumber),
    created: n(row.pick(created.data?.cordium)?.totalNumber),
    attention: row.attention?.(totals.data?.cordium),
  }));

  const unavailable = totals.data?.unavailables ?? [];

  return (
    <Panel
      icon={Layers}
      title="Cordium inventory"
      description={`Every Cordium resource kind · green badges show what was created in the last ${rangeLabel}`}
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
