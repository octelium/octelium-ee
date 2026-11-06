import * as CoreP from "@/apis/corev1/corev1";
import { twMerge } from "@/lib/utils";
import {
  AppWindow,
  Boxes,
  Braces,
  BrainCircuit,
  Cable,
  Globe2,
  LucideIcon,
  Monitor,
  MonitorPlay,
  Radio,
  Search,
  Shield,
  SquareTerminal,
  Waypoints,
} from "lucide-react";
import * as React from "react";
import { GrMysql } from "react-icons/gr";
import { SiPostgresql } from "react-icons/si";
import { VscMcp } from "react-icons/vsc";

const Mode = CoreP.Service_Spec_Mode;

export type ModeIcon = LucideIcon | React.ComponentType<any>;

export type ModeName = Exclude<keyof typeof CoreP.Service_Spec_Mode, number>;

export type ConfigKind = CoreP.Service_Spec_Config["type"]["oneofKind"];

export type ModeCategory = "web" | "ai" | "access" | "data" | "network";

export type UpstreamTLS = "full" | "trust" | "none";

export interface ModeInfo {
  kind: ModeName;
  mode: CoreP.Service_Spec_Mode;
  label: string;
  tab?: string;
  description: string;
  category: ModeCategory;
  keywords: string[];
  icon: ModeIcon;
  config: ConfigKind;
  urlPlaceholder: string;
  isPublic: boolean;
  isAnonymous: boolean;
  listenerTLS: boolean;
  upstreamTLS: UpstreamTLS;
}

export const MODE_CATEGORY_LABEL: Record<ModeCategory, string> = {
  web: "Web and APIs",
  ai: "AI gateways",
  access: "Infrastructure access",
  data: "Databases",
  network: "Network",
};

export const MODES: ModeInfo[] = [
  {
    kind: "HTTP",
    mode: Mode.HTTP,
    label: "HTTP",
    tab: "HTTP",
    description: "HTTP APIs and services",
    category: "web",
    keywords: ["rest", "api", "https", "proxy"],
    icon: Globe2,
    config: "http",
    urlPlaceholder: "https://api.example.com",
    isPublic: true,
    isAnonymous: true,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "WEB",
    mode: Mode.WEB,
    label: "Web App",
    tab: "Web",
    description: "Web applications opened in a browser",
    category: "web",
    keywords: ["browser", "site", "human", "http"],
    icon: AppWindow,
    config: "http",
    urlPlaceholder: "http://app.internal:3000",
    isPublic: true,
    isAnonymous: true,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "GRPC",
    mode: Mode.GRPC,
    label: "gRPC",
    tab: "gRPC",
    description: "gRPC services over HTTP/2",
    category: "web",
    keywords: ["http2", "protobuf", "rpc"],
    icon: Braces,
    config: "http",
    urlPlaceholder: "grpc://api.internal:50051",
    isPublic: true,
    isAnonymous: true,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "LLM",
    mode: Mode.LLM,
    label: "LLM / AI Gateway",
    tab: "LLM",
    description: "OpenAI, Anthropic, Gemini and Bedrock inference APIs",
    category: "ai",
    keywords: ["openai", "anthropic", "claude", "gemini", "bedrock", "model"],
    icon: BrainCircuit,
    config: "llm",
    urlPlaceholder: "https://api.openai.com/v1",
    isPublic: true,
    isAnonymous: true,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "MCP",
    mode: Mode.MCP,
    label: "MCP Gateway",
    tab: "MCP",
    description: "Model Context Protocol servers",
    category: "ai",
    keywords: ["model context protocol", "tools", "agent"],
    icon: VscMcp,
    config: "mcp",
    urlPlaceholder: "https://mcp.example.com/mcp",
    isPublic: true,
    isAnonymous: true,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "SSH",
    mode: Mode.SSH,
    label: "SSH",
    tab: "SSH",
    description: "SSH servers and embedded SSH on connected clients",
    category: "access",
    keywords: ["shell", "terminal", "sftp", "esshd"],
    icon: SquareTerminal,
    config: "ssh",
    urlPlaceholder: "ssh://10.0.0.4",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: false,
    upstreamTLS: "none",
  },
  {
    kind: "KUBERNETES",
    mode: Mode.KUBERNETES,
    label: "Kubernetes",
    tab: "Kubernetes",
    description: "Kubernetes API servers",
    category: "access",
    keywords: ["k8s", "kubectl", "cluster"],
    icon: Boxes,
    config: "kubernetes",
    urlPlaceholder: "https://10.0.0.1:6443",
    isPublic: true,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "none",
  },
  {
    kind: "RDP_WEB",
    mode: Mode.RDP_WEB,
    label: "RDP Web",
    tab: "RDP",
    description: "Remote desktops opened in a browser",
    category: "access",
    keywords: ["windows", "remote desktop", "clientless"],
    icon: MonitorPlay,
    config: "rdp",
    urlPlaceholder: "rdp://10.0.0.8",
    isPublic: true,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "none",
  },
  {
    kind: "RDP",
    mode: Mode.RDP,
    label: "RDP",
    tab: "RDP",
    description: "Remote desktops for native RDP clients",
    category: "access",
    keywords: ["windows", "remote desktop"],
    icon: Monitor,
    config: "rdp",
    urlPlaceholder: "rdp://10.0.0.8",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: false,
    upstreamTLS: "none",
  },
  {
    kind: "POSTGRES",
    mode: Mode.POSTGRES,
    label: "PostgreSQL",
    tab: "PostgreSQL",
    description: "PostgreSQL databases",
    category: "data",
    keywords: ["postgres", "pg", "sql", "database"],
    icon: SiPostgresql,
    config: "postgres",
    urlPlaceholder: "postgres://db.internal",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "trust",
  },
  {
    kind: "MYSQL",
    mode: Mode.MYSQL,
    label: "MySQL",
    tab: "MySQL",
    description: "MySQL databases",
    category: "data",
    keywords: ["mariadb", "sql", "database"],
    icon: GrMysql,
    config: "mysql",
    urlPlaceholder: "mysql://db.internal",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "none",
  },
  {
    kind: "TCP",
    mode: Mode.TCP,
    label: "TCP",
    description: "Any TCP-based protocol",
    category: "network",
    keywords: ["socket", "stream", "l4"],
    icon: Cable,
    config: undefined,
    urlPlaceholder: "tcp://10.0.0.20:9000",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "full",
  },
  {
    kind: "UDP",
    mode: Mode.UDP,
    label: "UDP",
    description: "Any UDP-based protocol",
    category: "network",
    keywords: ["datagram", "l4"],
    icon: Radio,
    config: undefined,
    urlPlaceholder: "udp://10.0.0.21:9000",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: false,
    upstreamTLS: "none",
  },
  {
    kind: "DNS",
    mode: Mode.DNS,
    label: "DNS",
    description: "DNS resolvers",
    category: "network",
    keywords: ["resolver", "name server"],
    icon: Search,
    config: undefined,
    urlPlaceholder: "dns://10.0.0.53",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: false,
    upstreamTLS: "none",
  },
  {
    kind: "SOCKS5",
    mode: Mode.SOCKS5,
    label: "SOCKS5",
    tab: "SOCKS5",
    description: "SOCKS5 proxies and egress through connected clients",
    category: "network",
    keywords: ["proxy", "egress"],
    icon: Shield,
    config: "socks5",
    urlPlaceholder: "socks5://proxy.internal",
    isPublic: false,
    isAnonymous: false,
    listenerTLS: true,
    upstreamTLS: "none",
  },
];

export const getModeInfo = (mode?: CoreP.Service_Spec_Mode) =>
  MODES.find((item) => item.mode === mode);

export const getModeIcon = (mode?: CoreP.Service_Spec_Mode): ModeIcon =>
  getModeInfo(mode)?.icon ?? Waypoints;

export const ModeTile = (props: {
  mode?: CoreP.Service_Spec_Mode;
  size?: "sm" | "lg";
}) => {
  const Icon = getModeIcon(props.mode);
  const large = props.size === "lg";
  return (
    <span
      aria-hidden="true"
      className={twMerge(
        "flex shrink-0 items-center justify-center border border-slate-200 bg-slate-50 text-slate-600",
        large ? "h-11 w-11 rounded-xl" : "h-7 w-7 rounded-md",
      )}
    >
      <Icon size={large ? 20 : 14} strokeWidth={2.1} />
    </span>
  );
};
