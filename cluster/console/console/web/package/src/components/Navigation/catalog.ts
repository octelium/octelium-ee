import {
  AlertTriangle,
  BadgeCheck,
  BookKey,
  Boxes,
  BrainCircuit,
  Building2,
  CalendarClock,
  ChartNoAxesCombined,
  CirclePlay,
  ClipboardCheck,
  CodeXml,
  Container,
  Cpu,
  Crown,
  DoorClosed,
  Eye,
  Fingerprint,
  Folder,
  GitBranch,
  Globe,
  Hammer,
  Globe2,
  Inbox,
  KeyRound,
  KeySquare,
  LaptopMinimal,
  Layers,
  LayoutDashboard,
  LayoutTemplate,
  Library,
  LockKeyhole,
  LockOpen,
  PanelTop,
  Plug,
  Scroll,
  Settings,
  Settings2,
  Shield,
  ShieldCheck,
  ShieldEllipsis,
  ShieldUser,
  Sparkles,
  SquareTerminal,
  Telescope,
  Terminal,
  Timer,
  User,
  UserCheck,
  Users,
  UsersRound,
  type LucideIcon,
} from "lucide-react";
import { getAPIKindFromPath, type API } from "@/utils/pb/paths";

export interface NavItem {
  title: string;
  url: string;
  icon: LucideIcon;
  keywords?: string[];
}

export interface NavSection {
  label: string;
  prefix: string;
  defaultPath: string;
  icon: LucideIcon;
  items: NavItem[];
  views?: NavItem[];
  keywords?: string[];
}

export const itemsCore: NavItem[] = [
  {
    title: "Services",
    url: "/core/services",
    icon: PanelTop,
    keywords: ["apps", "upstreams", "endpoints"],
  },
  { title: "Namespaces", url: "/core/namespaces", icon: Boxes },
  {
    title: "Users",
    url: "/core/users",
    icon: User,
    keywords: ["people", "identities", "accounts"],
  },
  {
    title: "Sessions",
    url: "/core/sessions",
    icon: Terminal,
    keywords: ["connected", "clients"],
  },
  { title: "Devices", url: "/core/devices", icon: LaptopMinimal },
  { title: "Groups", url: "/core/groups", icon: Users, keywords: ["teams"] },
  {
    title: "Policies",
    url: "/core/policies",
    icon: Shield,
    keywords: ["authorization", "rules", "cel", "opa"],
  },
  {
    title: "Identity Providers",
    url: "/core/identityproviders",
    icon: Fingerprint,
    keywords: ["idp", "sso", "oidc", "saml", "github"],
  },
  { title: "Secrets", url: "/core/secrets", icon: KeyRound },
  {
    title: "Credentials",
    url: "/core/credentials",
    icon: LockOpen,
    keywords: ["tokens", "oauth2", "access tokens"],
  },
  { title: "Gateways", url: "/core/gateways", icon: DoorClosed },
  { title: "Regions", url: "/core/regions", icon: Globe },
  {
    title: "Authenticators",
    url: "/core/authenticators",
    icon: LockKeyhole,
    keywords: ["mfa", "passkeys", "fido", "totp"],
  },
  {
    title: "ClusterConfig",
    url: "/core/clusterconfig",
    icon: Settings2,
    keywords: ["cluster configuration", "settings"],
  },
];

export const itemsEnterprise: NavItem[] = [
  {
    title: "Certificates",
    url: "/enterprise/certificates",
    icon: ShieldCheck,
    keywords: ["tls", "x509"],
  },
  {
    title: "Certificate Issuers",
    url: "/enterprise/certificateissuers",
    icon: Crown,
    keywords: ["acme", "ca"],
  },
  {
    title: "Collector Exporters",
    url: "/enterprise/collectorexporters",
    icon: Telescope,
    keywords: ["opentelemetry", "otel", "siem", "export"],
  },
  {
    title: "Directory Providers",
    url: "/enterprise/directoryproviders",
    icon: Folder,
    keywords: ["scim", "directory sync"],
  },
  { title: "Secrets", url: "/enterprise/secrets", icon: KeyRound },
  {
    title: "DNS Providers",
    url: "/enterprise/dnsproviders",
    icon: Globe2,
  },
  {
    title: "Secret Stores",
    url: "/enterprise/secretstores",
    icon: BookKey,
    keywords: ["vault", "kms"],
  },
  {
    title: "ClusterConfig",
    url: "/enterprise/clusterconfig",
    icon: Settings2,
    keywords: ["cluster configuration", "settings"],
  },
  {
    title: "Policy Tester",
    url: "/enterprise/policytester",
    icon: BadgeCheck,
    keywords: ["simulate", "evaluate", "test policies"],
  },
];

export const itemsAccess: NavItem[] = [
  { title: "Catalogs", url: "/access/catalogs", icon: Layers },
  {
    title: "Requests",
    url: "/access/requests",
    icon: Inbox,
    keywords: ["jit", "just-in-time", "access requests"],
  },
  {
    title: "Reviews",
    url: "/access/reviews",
    icon: ClipboardCheck,
    keywords: ["approvals", "access reviews"],
  },
  { title: "Policies", url: "/access/policies", icon: Shield },
  {
    title: "Integrations",
    url: "/access/integrations",
    icon: Plug,
    keywords: ["slack", "teams", "jira"],
  },
  { title: "Secrets", url: "/access/secrets", icon: KeyRound },
  {
    title: "ClusterConfig",
    url: "/access/clusterconfig",
    icon: Settings2,
    keywords: ["access configuration", "settings", "priority"],
  },
];

export const itemsCordium: NavItem[] = [
  {
    title: "Workspaces",
    url: "/cordium/workspaces",
    icon: Container,
    keywords: ["dev environments", "cde"],
  },
  { title: "Templates", url: "/cordium/templates", icon: LayoutTemplate },
  { title: "Spaces", url: "/cordium/spaces", icon: Layers },
  { title: "Memberships", url: "/cordium/memberships", icon: UsersRound },
  {
    title: "Git Providers",
    url: "/cordium/gitproviders",
    icon: GitBranch,
    keywords: ["github", "gitlab"],
  },
  { title: "Secrets", url: "/cordium/secrets", icon: KeyRound },
  { title: "User Secrets", url: "/cordium/usersecrets", icon: KeySquare },
  { title: "Regions", url: "/cordium/regions", icon: Globe },
];

export const itemsVisibility: NavItem[] = [
  {
    title: "Access Logs",
    url: "/visibility/accesslogs",
    icon: ShieldEllipsis,
    keywords: ["requests", "traffic", "denied", "allowed", "decisions"],
  },
  {
    title: "Authentication Logs",
    url: "/visibility/authenticationlogs",
    icon: ShieldUser,
    keywords: ["logins", "sign in", "auth logs"],
  },
  {
    title: "Audit Logs",
    url: "/visibility/auditlogs",
    icon: Library,
    keywords: ["changes", "history", "admin activity"],
  },
  {
    title: "Component Logs",
    url: "/visibility/componentlogs",
    icon: Scroll,
    keywords: ["errors", "runtime", "control plane"],
  },
  {
    title: "SSH Sessions",
    url: "/visibility/ssh",
    icon: SquareTerminal,
    keywords: ["recordings", "terminal", "replay"],
  },
  {
    title: "Metrics",
    url: "/visibility/metrics",
    icon: ChartNoAxesCombined,
    keywords: ["charts", "latency", "throughput"],
  },
  {
    title: "LLM",
    url: "/visibility/llm",
    icon: BrainCircuit,
    keywords: ["ai", "tokens", "models", "inference", "guardrails"],
  },
];

export const navSections: NavSection[] = [
  {
    label: "Core",
    prefix: "/core",
    defaultPath: "/core",
    icon: Cpu,
    items: itemsCore,
  },
  {
    label: "Enterprise",
    prefix: "/enterprise",
    defaultPath: "/enterprise",
    icon: Building2,
    items: itemsEnterprise,
  },
  {
    label: "Access",
    prefix: "/access",
    defaultPath: "/access",
    icon: UserCheck,
    items: itemsAccess,
    views: [
      {
        title: "Pending Requests",
        url: "/access/requests?state=PENDING",
        icon: Timer,
      },
      {
        title: "Requests past their deadline",
        url: "/access/requests?isDeadlinePassed=true",
        icon: CalendarClock,
      },
      {
        title: "Live access",
        url: "/access/requests?isActive=true",
        icon: KeyRound,
      },
      {
        title: "Integrations with problems",
        url: "/access/integrations?state=DEGRADED",
        icon: AlertTriangle,
      },
    ],
    keywords: ["just-in-time"],
  },
  {
    label: "Cordium",
    prefix: "/cordium",
    defaultPath: "/cordium",
    icon: CodeXml,
    items: itemsCordium,
    views: [
      {
        title: "Running Workspaces",
        url: "/cordium/workspaces?state=RUNNING",
        icon: CirclePlay,
      },
      {
        title: "Failed Workspaces",
        url: "/cordium/workspaces?isFailed=true",
        icon: AlertTriangle,
      },
      {
        title: "Building Templates",
        url: "/cordium/templates?isBuilding=true",
        icon: Hammer,
      },
    ],
  },
  {
    label: "Cluster Management",
    prefix: "/clusterman",
    defaultPath: "/clusterman",
    icon: Settings2,
    items: [{ title: "Cluster overview", url: "/clusterman", icon: Settings2 }],
    keywords: ["upgrade", "license", "versions"],
  },
  {
    label: "Visibility",
    prefix: "/visibility",
    defaultPath: "/visibility",
    icon: Eye,
    items: itemsVisibility,
    keywords: ["logs", "observability"],
  },
];

export const pageOverview: NavItem = {
  title: "Overview",
  url: "/",
  icon: LayoutDashboard,
  keywords: ["home", "dashboard"],
};

export const pageAgent: NavItem = {
  title: "Agent",
  url: "/agent",
  icon: Sparkles,
  keywords: ["ai", "assistant", "chat", "conversations"],
};

export const pageSettings: NavItem = {
  title: "Settings",
  url: "/settings",
  icon: Settings,
  keywords: ["profile", "appearance", "theme", "account"],
};

export const sectionItems: NavItem[] = navSections.map((section) => ({
  title: section.label,
  url: section.defaultPath,
  icon: section.icon,
  keywords: section.keywords,
}));

const tabLabels: Record<string, string> = {
  edit: "Configure",
  actions: "Actions",
  accesslogs: "Access Logs",
  authenticationlogs: "Authentication Logs",
  auditlogs: "Audit Logs",
  metrics: "Metrics",
  llm: "LLM",
  ssh: "SSH Recordings",
  playground: "Playground",
  connect: "Connect",
};

export interface ResourceCrumb {
  api: API;
  kind: string;
  name: string;
}

export interface Crumb {
  label: string;
  path: string;
  icon?: LucideIcon;
  children?: NavItem[];
  views?: NavItem[];
  resource?: ResourceCrumb;
}

const humanize = (segment: string) =>
  segment.replace(/[-_]+/g, " ").replace(/^./, (value) => value.toUpperCase());

const decode = (segment: string) => {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
};

export const findSection = (pathname: string): NavSection | undefined =>
  navSections.find(
    (section) =>
      pathname === section.prefix || pathname.startsWith(`${section.prefix}/`),
  );

export const findNavItem = (pathname: string): NavItem | undefined => {
  const section = findSection(pathname);
  return section?.items.find(
    (item) => pathname === item.url || pathname.startsWith(`${item.url}/`),
  );
};

export const kindIcon = (pathname: string): LucideIcon | undefined =>
  findNavItem(pathname)?.icon;

const homeCrumb = (): Crumb => ({
  label: pageOverview.title,
  path: pageOverview.url,
  icon: pageOverview.icon,
  children: [...sectionItems, pageAgent],
});

export const resolveTrail = (rawPathname: string): Crumb[] => {
  const pathname = rawPathname.replace(/\/+$/, "") || "/";
  const home = homeCrumb();
  if (pathname === "/") {
    return [home];
  }

  for (const page of [pageAgent, pageSettings]) {
    if (pathname === page.url || pathname.startsWith(`${page.url}/`)) {
      return [home, { label: page.title, path: page.url, icon: page.icon }];
    }
  }

  const section = findSection(pathname);
  const segments = pathname.split("/").filter(Boolean);
  if (!section) {
    return [
      home,
      ...segments.map((segment, idx) => ({
        label: humanize(decode(segment)),
        path: `/${segments.slice(0, idx + 1).join("/")}`,
      })),
    ];
  }

  const ownItems = section.items.filter(
    (item) => item.url !== section.defaultPath,
  );
  const trail: Crumb[] = [
    home,
    {
      label: section.label,
      path: section.defaultPath,
      icon: section.icon,
      children: ownItems.length > 0 ? ownItems : undefined,
      views: section.views,
    },
  ];

  const rest = segments.slice(1);
  if (rest.length === 0) {
    return trail;
  }

  const listPath = `${section.prefix}/${rest[0]}`;
  const item = section.items.find((itm) => itm.url === listPath);
  trail.push({
    label: item?.title ?? humanize(decode(rest[0])),
    path: listPath,
    icon: item?.icon,
  });

  if (rest.length === 1) {
    return trail;
  }

  const resourcePath = `${listPath}/${rest[1]}`;
  if (rest[1] === "create") {
    trail.push({ label: "Create", path: resourcePath });
    return trail;
  }

  const apiKind = getAPIKindFromPath(resourcePath);
  const name = decode(rest[1]);
  trail.push({
    label: name,
    path: resourcePath,
    icon: item?.icon,
    resource: apiKind
      ? { api: apiKind.api, kind: apiKind.kind, name }
      : undefined,
  });

  if (rest.length > 2) {
    trail.push({
      label: tabLabels[rest[2]] ?? humanize(decode(rest[2])),
      path: `${resourcePath}/${rest[2]}`,
    });
  }

  return trail;
};
