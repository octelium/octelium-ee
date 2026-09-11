import { Request_Spec_Urgency } from "@/apis/accessv1/accessv1";
import { getUrgencyColor } from "@/pages/access/Request/utils";
import { getClientVisibilityAccess } from "@/utils/client";
import { useQueries } from "@tanstack/react-query";
import { ClipboardCheck, Inbox, Layers, LucideIcon, Shield } from "lucide-react";
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

const STATE_COLORS = {
  approved: "var(--color-emerald-600)",
  rejected: "var(--color-red-600)",
  revoked: "var(--color-red-600)",
  expired: "var(--color-slate-500)",
  cancelled: "var(--color-slate-500)",
  revised: "var(--color-violet-600)",
  pending: "var(--color-amber-600)",
} as const;

const client = () => getClientVisibilityAccess();

const ENTRIES: SummaryEntry[] = [
  {
    kind: "Policy",
    label: "Policies",
    to: "/access/policies",
    icon: Shield,
    fetch: async () => (await client().getPolicySummary({})).response,
    derive: (d) => ({
      segments: [],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "Catalog",
    label: "Catalogs",
    to: "/access/catalogs",
    icon: Layers,
    fetch: async () => (await client().getCatalogSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Service", value: num(d?.totalService) },
        { label: "Namespace", value: num(d?.totalNamespace) },
      ],
      attention: [],
    }),
  },
  {
    kind: "Request",
    label: "Requests",
    to: "/access/requests",
    icon: Inbox,
    fetch: async () => (await client().getRequestSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Approved", value: num(d?.totalApproved), color: STATE_COLORS.approved },
        { label: "Rejected", value: num(d?.totalRejected), color: STATE_COLORS.rejected },
        { label: "Revoked", value: num(d?.totalRevoked), color: STATE_COLORS.revoked },
        { label: "Expired", value: num(d?.totalExpired), color: STATE_COLORS.expired },
        { label: "Cancelled", value: num(d?.totalCancelled), color: STATE_COLORS.cancelled },
      ],
      attention: [
        { label: "Pending", value: num(d?.totalPending), color: STATE_COLORS.pending },
        {
          label: "Highest urgency",
          value: num(d?.totalUrgencyHighest),
          color: getUrgencyColor(Request_Spec_Urgency.HIGHEST),
        },
        {
          label: "Very high urgency",
          value: num(d?.totalUrgencyVeryHigh),
          color: getUrgencyColor(Request_Spec_Urgency.VERY_HIGH),
        },
        {
          label: "High urgency",
          value: num(d?.totalUrgencyHigh),
          color: getUrgencyColor(Request_Spec_Urgency.HIGH),
        },
        { label: "Past deadline", value: num(d?.totalDeadlinePassed), color: "var(--color-red-600)" },
      ],
    }),
  },
  {
    kind: "Review",
    label: "Reviews",
    to: "/access/reviews",
    icon: ClipboardCheck,
    fetch: async () => (await client().getReviewSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Approved", value: num(d?.totalApproved), color: STATE_COLORS.approved },
        { label: "Rejected", value: num(d?.totalRejected), color: STATE_COLORS.rejected },
        { label: "Revised", value: num(d?.totalRevised), color: STATE_COLORS.revised },
      ],
      attention: [
        { label: "Pending", value: num(d?.totalPending), color: STATE_COLORS.pending },
      ],
    }),
  },
];

const AccessResourceInventory = () => {
  const results = useQueries({
    queries: ENTRIES.map((entry) => ({
      queryKey: ["visibility", "access", "summary", entry.kind],
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
    <InventoryTable title="Access inventory" unitLabel="records" rows={rows} />
  );
};

export default AccessResourceInventory;
