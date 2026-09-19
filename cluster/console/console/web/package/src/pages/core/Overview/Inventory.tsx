import { GetClusterSummaryResponse } from "@/apis/visibilityv1/visibilityv1";
import {
  InventoryEntry,
  InventoryRail,
  Panel,
} from "@/components/Dashboard/components";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Boxes,
  DoorClosed,
  Fingerprint,
  Globe,
  KeyRound,
  LaptopMinimal,
  Layers,
  LockKeyhole,
  LockOpen,
  LucideIcon,
  PanelTop,
  Shield,
  Terminal,
  User,
  Users,
} from "lucide-react";
import { useCoreCreated, useCoreTotals } from "./queries";

type Core = GetClusterSummaryResponse["core"];

type Row = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  pick: (core: Core) => { totalNumber?: unknown } | undefined;
  attention?: (core: Core) => InventoryEntry["attention"];
};

const ROWS: Row[] = [
  {
    kind: "User",
    label: "Users",
    to: "/core/users",
    icon: User,
    pick: (core) => core?.user,
    attention: (core) => [
      { label: "disabled ", value: n(core?.user?.totalDisabled) },
    ],
  },
  {
    kind: "Session",
    label: "Sessions",
    to: "/core/sessions",
    icon: Terminal,
    pick: (core) => core?.session,
    attention: (core) => [
      { label: "pending ", value: n(core?.session?.totalPending) },
      {
        label: "rejected ",
        value: n(core?.session?.totalRejected),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Device",
    label: "Devices",
    to: "/core/devices",
    icon: LaptopMinimal,
    pick: (core) => core?.device,
    attention: (core) => [
      { label: "pending ", value: n(core?.device?.totalPending) },
      {
        label: "rejected ",
        value: n(core?.device?.totalRejected),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Service",
    label: "Services",
    to: "/core/services",
    icon: PanelTop,
    pick: (core) => core?.service,
    attention: (core) => [
      { label: "disabled ", value: n(core?.service?.totalDisabled) },
      {
        label: "anonymous ",
        value: n(core?.service?.totalAnonymous),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Namespace",
    label: "Namespaces",
    to: "/core/namespaces",
    icon: Boxes,
    pick: (core) => core?.namespace,
  },
  {
    kind: "Policy",
    label: "Policies",
    to: "/core/policies",
    icon: Shield,
    pick: (core) => core?.policy,
    attention: (core) => [
      { label: "disabled ", value: n(core?.policy?.totalDisabled) },
    ],
  },
  {
    kind: "Group",
    label: "Groups",
    to: "/core/groups",
    icon: Users,
    pick: (core) => core?.group,
  },
  {
    kind: "Credential",
    label: "Credentials",
    to: "/core/credentials",
    icon: LockOpen,
    pick: (core) => core?.credential,
    attention: (core) => [
      { label: "disabled ", value: n(core?.credential?.totalDisabled) },
    ],
  },
  {
    kind: "IdentityProvider",
    label: "Identity providers",
    to: "/core/identityproviders",
    icon: Fingerprint,
    pick: (core) => core?.identityProvider,
    attention: (core) => [
      { label: "disabled ", value: n(core?.identityProvider?.totalDisabled) },
    ],
  },
  {
    kind: "Authenticator",
    label: "Authenticators",
    to: "/core/authenticators",
    icon: LockKeyhole,
    pick: (core) => core?.authenticator,
    attention: (core) => [
      { label: "pending ", value: n(core?.authenticator?.totalPending) },
      {
        label: "rejected ",
        value: n(core?.authenticator?.totalRejected),
        tone: "critical",
      },
    ],
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/core/secrets",
    icon: KeyRound,
    pick: (core) => core?.secret,
  },
  {
    kind: "Gateway",
    label: "Gateways",
    to: "/core/gateways",
    icon: DoorClosed,
    pick: (core) => core?.gateway,
  },
  {
    kind: "Region",
    label: "Regions",
    to: "/core/regions",
    icon: Globe,
    pick: (core) => core?.region,
  },
];

const Inventory = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const totals = useCoreTotals(periodMinutes, QUERY_PRIORITY.critical);
  const created = useCoreCreated(periodMinutes, QUERY_PRIORITY.high);

  const entries: InventoryEntry[] = ROWS.map((row) => ({
    kind: row.kind,
    label: row.label,
    to: row.to,
    icon: row.icon,
    total: n(row.pick(totals.data?.core)?.totalNumber),
    created: n(row.pick(created.data?.core)?.totalNumber),
    attention: row.attention?.(totals.data?.core),
  }));

  const unavailable = totals.data?.unavailables ?? [];

  return (
    <Panel
      icon={Layers}
      title="Core inventory"
      description={`Every core resource kind · green badges show what was created in the last ${rangeLabel}`}
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
