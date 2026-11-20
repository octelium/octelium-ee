import {
  Ban,
  Bot,
  Boxes,
  Braces,
  Building2,
  CalendarRange,
  Check,
  Clock,
  Cpu,
  Eye,
  Fingerprint,
  FolderTree,
  Globe,
  Hourglass,
  KeyRound,
  Laptop,
  Link2,
  Lock,
  Moon,
  Network,
  Pencil,
  Plug,
  Server,
  Shield,
  Terminal,
  UserRound,
  Users,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import type { CategoryId } from "./blocks";
import type { RecipeIcon } from "./recipes";

export type Tone =
  | "blue"
  | "violet"
  | "emerald"
  | "teal"
  | "sky"
  | "amber"
  | "indigo"
  | "slate"
  | "orange"
  | "green"
  | "yellow"
  | "red";

export const TONE_CHIP: Record<Tone, string> = {
  blue: "border-blue-200 bg-blue-50 text-blue-700",
  violet: "border-violet-200 bg-violet-50 text-violet-700",
  emerald: "border-emerald-200 bg-emerald-50 text-emerald-700",
  teal: "border-teal-200 bg-teal-50 text-teal-700",
  sky: "border-sky-200 bg-sky-50 text-sky-700",
  amber: "border-amber-200 bg-amber-50 text-amber-700",
  indigo: "border-indigo-200 bg-indigo-50 text-indigo-700",
  slate: "border-slate-200 bg-slate-100 text-slate-700",
  orange: "border-orange-200 bg-orange-50 text-orange-700",
  green: "border-green-200 bg-green-50 text-green-700",
  yellow: "border-yellow-200 bg-yellow-50 text-yellow-700",
  red: "border-red-200 bg-red-50 text-red-700",
};

export type CategoryKey = CategoryId | "custom" | "general";

export const CATEGORY_META: Record<
  CategoryKey,
  { icon: LucideIcon; tone: Tone }
> = {
  identity: { icon: Users, tone: "blue" },
  resource: { icon: Server, tone: "violet" },
  session: { icon: KeyRound, tone: "emerald" },
  device: { icon: Laptop, tone: "teal" },
  network: { icon: Globe, tone: "sky" },
  time: { icon: Clock, tone: "amber" },
  http: { icon: Link2, tone: "indigo" },
  api: { icon: Boxes, tone: "slate" },
  protocol: { icon: Network, tone: "orange" },
  mcp: { icon: Plug, tone: "green" },
  llm: { icon: Bot, tone: "yellow" },
  custom: { icon: Braces, tone: "slate" },
  general: { icon: Check, tone: "slate" },
};

const BLOCK_ICONS: Record<string, LucideIcon> = {
  "identity.user": UserRound,
  "resource.namespace": FolderTree,
  "session.aal": Fingerprint,
  "session.passkey": Fingerprint,
  "session.hardwareKey": KeyRound,
  "device.posture": Shield,
  "device.os": Cpu,
  "time.range": CalendarRange,
  "time.dayType": CalendarRange,
  "network.ipRange": Network,
  "protocol.ssh": Terminal,
  "protocol.sshUser": Terminal,
  "mcp.tool": Wrench,
};

export const blockIcon = (id: string, category: CategoryKey): LucideIcon =>
  BLOCK_ICONS[id] ?? CATEGORY_META[category].icon;

export const RECIPE_ICONS: Record<RecipeIcon, LucideIcon> = {
  users: Users,
  user: UserRound,
  building: Building2,
  server: Server,
  clock: Clock,
  moon: Moon,
  hourglass: Hourglass,
  eye: Eye,
  pencil: Pencil,
  lock: Lock,
  terminal: Terminal,
  shield: Shield,
  fingerprint: Fingerprint,
  laptop: Laptop,
  globe: Globe,
  network: Network,
  boxes: Boxes,
  bot: Bot,
  wrench: Wrench,
  check: Check,
  ban: Ban,
};
