import { getClientVisibilityCore } from "@/utils/client";
import { seriesColor, STATUS_COLORS } from "@/utils/charts/palette";
import { SegmentedControl } from "@mantine/core";
import { useQueries } from "@tanstack/react-query";
import {
  Boxes,
  ChevronRight,
  DoorClosed,
  Fingerprint,
  Globe,
  KeyRound,
  LaptopMinimal,
  LockKeyhole,
  LockOpen,
  LucideIcon,
  PanelTop,
  Shield,
  Terminal,
  User,
  Users,
} from "lucide-react";
import * as React from "react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const num = (value: unknown) => Number(value ?? 0);

type Segment = { label: string; value: number };

type Row = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  total: number;
  segments: Segment[];
  attention: Segment[];
  isLoading: boolean;
};

type SummaryEntry = {
  kind: string;
  label: string;
  to: string;
  icon: LucideIcon;
  fetch: () => Promise<any>;
  derive: (data: any) => { segments: Segment[]; attention: Segment[] };
};

const client = () => getClientVisibilityCore();

const ENTRIES: SummaryEntry[] = [
  {
    kind: "User",
    label: "Users",
    to: "/core/users",
    icon: User,
    fetch: async () => (await client().getUserSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Human", value: num(d?.totalHuman) },
        { label: "Workload", value: num(d?.totalWorkload) },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "Session",
    label: "Sessions",
    to: "/core/sessions",
    icon: Terminal,
    fetch: async () => (await client().getSessionSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Client", value: num(d?.totalClient) },
        { label: "Clientless", value: num(d?.totalClientless) },
      ],
      attention: [
        { label: "Pending", value: num(d?.totalPending) },
        { label: "Rejected", value: num(d?.totalRejected) },
      ],
    }),
  },
  {
    kind: "Device",
    label: "Devices",
    to: "/core/devices",
    icon: LaptopMinimal,
    fetch: async () => (await client().getDeviceSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Linux", value: num(d?.totalLinux) },
        { label: "Windows", value: num(d?.totalWindows) },
        { label: "macOS", value: num(d?.totalMac) },
        { label: "Android", value: num(d?.totalAndroid) },
        { label: "iOS", value: num(d?.totalIOS) },
      ],
      attention: [
        { label: "Pending", value: num(d?.totalPending) },
        { label: "Rejected", value: num(d?.totalRejected) },
      ],
    }),
  },
  {
    kind: "Service",
    label: "Services",
    to: "/core/services",
    icon: PanelTop,
    fetch: async () => (await client().getServiceSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "HTTP", value: num(d?.totalHTTP) },
        { label: "Web", value: num(d?.totalWeb) },
        { label: "TCP", value: num(d?.totalTCP) },
        { label: "SSH", value: num(d?.totalSSH) },
        { label: "LLM", value: num(d?.totalLLM) },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "Namespace",
    label: "Namespaces",
    to: "/core/namespaces",
    icon: Boxes,
    fetch: async () => (await client().getNamespaceSummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
  {
    kind: "Policy",
    label: "Policies",
    to: "/core/policies",
    icon: Shield,
    fetch: async () => (await client().getPolicySummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Allow rules", value: num(d?.totalRuleAllow) },
        { label: "Deny rules", value: num(d?.totalRuleDenied) },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "Credential",
    label: "Credentials",
    to: "/core/credentials",
    icon: LockOpen,
    fetch: async () => (await client().getCredentialSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "Auth token", value: num(d?.totalAuthenticationToken) },
        { label: "Access token", value: num(d?.totalAccessToken) },
        { label: "OAuth2", value: num(d?.totalOAuth2) },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "IdentityProvider",
    label: "Identity providers",
    to: "/core/identityproviders",
    icon: Fingerprint,
    fetch: async () => (await client().getIdentityProviderSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "OIDC", value: num(d?.totalOIDC) },
        { label: "SAML", value: num(d?.totalSAML) },
        { label: "GitHub", value: num(d?.totalGithub) },
      ],
      attention: [{ label: "Disabled", value: num(d?.totalDisabled) }],
    }),
  },
  {
    kind: "Authenticator",
    label: "Authenticators",
    to: "/core/authenticators",
    icon: LockKeyhole,
    fetch: async () => (await client().getAuthenticatorSummary({})).response,
    derive: (d) => ({
      segments: [
        { label: "FIDO", value: num(d?.totalFIDO) },
        { label: "TOTP", value: num(d?.totalTOTP) },
        { label: "TPM", value: num(d?.totalTPM) },
      ],
      attention: [
        { label: "Pending", value: num(d?.totalPending) },
        { label: "Rejected", value: num(d?.totalRejected) },
      ],
    }),
  },
  {
    kind: "Group",
    label: "Groups",
    to: "/core/groups",
    icon: Users,
    fetch: async () => (await client().getGroupSummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
  {
    kind: "Secret",
    label: "Secrets",
    to: "/core/secrets",
    icon: KeyRound,
    fetch: async () => (await client().getSecretSummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
  {
    kind: "Gateway",
    label: "Gateways",
    to: "/core/gateways",
    icon: DoorClosed,
    fetch: async () => (await client().getGatewaySummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
  {
    kind: "Region",
    label: "Regions",
    to: "/core/regions",
    icon: Globe,
    fetch: async () => (await client().getRegionSummary({})).response,
    derive: () => ({ segments: [], attention: [] }),
  },
];

const CompositionBar = (props: { segments: Segment[]; total: number }) => {
  const shown = props.segments.filter((segment) => segment.value > 0);
  if (shown.length === 0 || props.total === 0) {
    return <span className="text-micro font-normal text-slate-500">—</span>;
  }

  const covered = shown.reduce((sum, segment) => sum + segment.value, 0);
  const rest = Math.max(0, props.total - covered);

  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="flex h-1.5 w-full gap-0.5 overflow-hidden rounded-full">
        {shown.map((segment, index) => (
          <span
            key={segment.label}
            className="h-full rounded-full"
            style={{
              width: `${(segment.value / props.total) * 100}%`,
              backgroundColor: seriesColor(index),
            }}
          />
        ))}
        {rest > 0 && (
          <span
            className="h-full rounded-full bg-slate-200"
            style={{ width: `${(rest / props.total) * 100}%` }}
          />
        )}
      </div>

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        {shown.map((segment, index) => (
          <span
            key={segment.label}
            className="inline-flex items-center gap-1.5 text-micro font-normal text-slate-600"
          >
            <span
              className="h-2 w-2 shrink-0 rounded-full"
              style={{ backgroundColor: seriesColor(index) }}
            />
            {segment.label}
            <span className="font-semibold tabular-nums text-slate-800">
              {segment.value.toLocaleString()}
            </span>
          </span>
        ))}
        {rest > 0 && (
          <span className="inline-flex items-center gap-1.5 text-micro font-normal text-slate-500">
            <span className="h-2 w-2 shrink-0 rounded-full bg-slate-300" />
            Other
            <span className="font-semibold tabular-nums text-slate-700">
              {rest.toLocaleString()}
            </span>
          </span>
        )}
      </div>
    </div>
  );
};

type SortMode = "name" | "total";

const ResourceInventory = () => {
  const [sort, setSort] = React.useState<SortMode>("total");

  const results = useQueries({
    queries: ENTRIES.map((entry) => ({
      queryKey: ["visibility", "core", "summary", entry.kind],
      queryFn: entry.fetch,
    })),
  });

  const rows: Row[] = ENTRIES.map((entry, index) => {
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

  const sorted = [...rows].sort((a, b) =>
    sort === "total" ? b.total - a.total : a.label.localeCompare(b.label),
  );

  const grandTotal = rows.reduce((sum, row) => sum + row.total, 0);

  return (
    <section className="overflow-hidden rounded-xl border border-slate-200 bg-white">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200 px-4 py-3">
        <div className="min-w-0">
          <h2 className="text-body font-semibold text-slate-800">
            Resource inventory
          </h2>
          <p className="mt-0.5 text-micro font-normal text-slate-500">
            {grandTotal.toLocaleString()} resources across {rows.length} kinds
          </p>
        </div>

        <SegmentedControl
          size="xs"
          value={sort}
          onChange={(value) => setSort(value as SortMode)}
          data={[
            { value: "total", label: "By count" },
            { value: "name", label: "By name" },
          ]}
        />
      </header>

      <div className="w-full overflow-x-auto">
        <table className="w-full min-w-[720px] border-collapse">
          <thead>
            <tr className="border-b border-slate-100 bg-slate-50/60">
              <th className="px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Kind
              </th>
              <th className="w-20 px-4 py-2 text-right text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Total
              </th>
              <th className="w-1/2 px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Composition
              </th>
              <th className="px-4 py-2 text-left text-micro font-semibold uppercase tracking-[0.06em] text-slate-500">
                Needs attention
              </th>
              <th className="w-8 px-4 py-2" />
            </tr>
          </thead>
          <tbody>
            {sorted.map((row) => {
              const Icon = row.icon;
              return (
                <tr
                  key={row.kind}
                  className="group border-b border-slate-100 last:border-b-0 hover:bg-slate-50/70"
                >
                  <td className="px-4 py-3 align-middle">
                    <Link
                      to={row.to}
                      className="inline-flex items-center gap-2.5 outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
                    >
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
                        <Icon size={13} strokeWidth={2.2} />
                      </span>
                      <span className="text-body font-semibold text-slate-800 group-hover:text-slate-900">
                        {row.label}
                      </span>
                    </Link>
                  </td>

                  <td className="px-4 py-3 text-right align-middle">
                    {row.isLoading ? (
                      <span className="ml-auto block h-4 w-10 animate-pulse rounded bg-slate-100" />
                    ) : (
                      <span
                        className={twMerge(
                          "text-sm font-semibold tabular-nums",
                          row.total === 0 ? "text-slate-500" : "text-slate-900",
                        )}
                      >
                        {row.total.toLocaleString()}
                      </span>
                    )}
                  </td>

                  <td className="w-1/2 px-4 py-3 align-middle">
                    <CompositionBar segments={row.segments} total={row.total} />
                  </td>

                  <td className="px-4 py-3 align-middle">
                    {row.attention.length === 0 ? (
                      <span className="text-micro font-normal text-slate-500">
                        None
                      </span>
                    ) : (
                      <span className="flex flex-wrap gap-1.5">
                        {row.attention.map((item) => (
                          <span
                            key={item.label}
                            className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-micro font-semibold"
                            style={{
                              borderColor: `${STATUS_COLORS.warning}55`,
                              color: STATUS_COLORS.critical,
                            }}
                          >
                            {item.label}
                            <span className="tabular-nums">
                              {item.value.toLocaleString()}
                            </span>
                          </span>
                        ))}
                      </span>
                    )}
                  </td>

                  <td className="px-4 py-3 align-middle">
                    <ChevronRight
                      size={13}
                      strokeWidth={2.4}
                      className="text-slate-500 transition-colors duration-150 group-hover:text-slate-800"
                    />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default ResourceInventory;
