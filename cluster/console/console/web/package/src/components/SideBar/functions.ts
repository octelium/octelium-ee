import {
  itemsVisibility,
  navSections,
  type NavItem,
} from "@/components/Navigation/catalog";
import {
  Container,
  Eye,
  Globe,
  LaptopMinimal,
  Plug,
  Settings2,
  Shield,
  Users,
  type LucideIcon,
} from "lucide-react";

export interface FunctionGroup {
  id: string;
  label: string;
  icon: LucideIcon;
  items: NavItem[];
}

const items = new Map(
  navSections.flatMap((section) =>
    section.items.map((item) => [item.url, item] as const),
  ),
);

const item = (url: string, title?: string): NavItem => {
  const entry = items.get(url);
  if (!entry) throw new Error(`Unknown navigation path: ${url}`);
  return title ? { ...entry, title } : entry;
};

export const functionGroups: FunctionGroup[] = [
  {
    id: "access",
    label: "Access",
    icon: Shield,
    items: [
      item("/core/services"),
      item("/core/namespaces"),
      item("/core/policies", "Core policies"),
      item("/access/catalogs"),
      item("/access/requests", "Access requests"),
      item("/access/reviews", "Access reviews"),
      item("/access/policies", "Access policies"),
      item("/enterprise/policytester"),
    ],
  },
  {
    id: "identity",
    label: "Identity",
    icon: Users,
    items: [
      item("/core/users"),
      item("/core/groups"),
      item("/core/identityproviders"),
      item("/enterprise/directoryproviders"),
      item("/core/authenticators"),
      item("/core/credentials"),
    ],
  },
  {
    id: "devices",
    label: "Devices & sessions",
    icon: LaptopMinimal,
    items: [item("/core/devices"), item("/core/sessions")],
  },
  {
    id: "network",
    label: "Network",
    icon: Globe,
    items: [
      item("/core/regions"),
      item("/core/gateways"),
      item("/enterprise/dnsproviders"),
      item("/enterprise/certificates"),
      item("/enterprise/certificateissuers"),
    ],
  },
  {
    id: "workspaces",
    label: "Workspaces",
    icon: Container,
    items: [
      item("/cordium/workspaces"),
      item("/cordium/templates"),
      item("/cordium/spaces"),
      item("/cordium/memberships"),
      item("/cordium/gitproviders"),
      item("/cordium/regions", "Workspace regions"),
    ],
  },
  {
    id: "observability",
    label: "Observability",
    icon: Eye,
    items: itemsVisibility,
  },
  {
    id: "integrations",
    label: "Integrations",
    icon: Plug,
    items: [
      item("/access/integrations"),
      item("/access/integrationidentities"),
      item("/access/integrationbindings"),
      item("/enterprise/collectorexporters"),
      item("/enterprise/secretstores"),
    ],
  },
  {
    id: "settings",
    label: "Cluster settings",
    icon: Settings2,
    items: [
      item("/core/clusterconfig", "Core configuration"),
      item("/enterprise/clusterconfig", "Enterprise configuration"),
      item("/core/secrets", "Core secrets"),
      item("/enterprise/secrets", "Enterprise secrets"),
      item("/access/secrets", "Access secrets"),
      item("/cordium/secrets", "Cordium secrets"),
      item("/cordium/usersecrets"),
      item("/clusterman", "Cluster management"),
    ],
  },
];
