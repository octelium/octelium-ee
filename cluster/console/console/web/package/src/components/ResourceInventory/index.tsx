import { getClientVisibilityCore } from "@/utils/client";
import { useQueries } from "@tanstack/react-query";
import {
  Boxes,
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
import {
  InventoryRow,
  InventoryTable,
  Segment,
} from "./InventoryTable";

const num = (value: unknown) => Number(value ?? 0);

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

const ResourceInventory = () => {
  const results = useQueries({
    queries: ENTRIES.map((entry) => ({
      queryKey: ["visibility", "core", "summary", entry.kind],
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
    <InventoryTable title="Resource inventory" unitLabel="resources" rows={rows} />
  );
};

export default ResourceInventory;
