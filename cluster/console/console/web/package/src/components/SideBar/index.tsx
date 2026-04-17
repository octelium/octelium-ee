import {
  BadgeCheck,
  BookKey,
  Boxes,
  ChevronDown,
  Crown,
  DoorClosed,
  Fingerprint,
  Folder,
  Globe,
  Globe2,
  KeyRound,
  LaptopMinimal,
  Library,
  LockKeyhole,
  LockOpen,
  PanelTop,
  Scroll,
  Settings2,
  Shield,
  ShieldCheck,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Telescope,
  Terminal,
  User,
  Users,
} from "lucide-react";

import { AnimatePresence, motion } from "framer-motion";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { match } from "ts-pattern";

const itemsCore = [
  { title: "Services", url: "/core/services", icon: PanelTop },
  { title: "Namespaces", url: "/core/namespaces", icon: Boxes },
  { title: "Users", url: "/core/users", icon: User },
  { title: "Sessions", url: "/core/sessions", icon: Terminal },
  { title: "Devices", url: "/core/devices", icon: LaptopMinimal },
  { title: "Groups", url: "/core/groups", icon: Users },
  { title: "Policies", url: "/core/policies", icon: Shield },
  {
    title: "Identity Providers",
    url: "/core/identityproviders",
    icon: Fingerprint,
  },
  { title: "Secrets", url: "/core/secrets", icon: KeyRound },
  { title: "Credentials", url: "/core/credentials", icon: LockOpen },
  { title: "Gateways", url: "/core/gateways", icon: DoorClosed },
  { title: "Regions", url: "/core/regions", icon: Globe },
  { title: "Authenticators", url: "/core/authenticators", icon: LockKeyhole },
  { title: "ClusterConfig", url: "/core/clusterconfig", icon: Settings2 },
];

const itemsEnterprise = [
  { title: "Certificates", url: "/enterprise/certificates", icon: ShieldCheck },
  {
    title: "Certificate Issuers",
    url: "/enterprise/certificateissuers",
    icon: Crown,
  },
  {
    title: "Collector Exporters",
    url: "/enterprise/collectorexporters",
    icon: Telescope,
  },
  {
    title: "Directory Providers",
    url: "/enterprise/directoryproviders",
    icon: Folder,
  },
  { title: "Secrets", url: "/enterprise/secrets", icon: KeyRound },
  { title: "DNS Providers", url: "/enterprise/dnsproviders", icon: Globe2 },
  { title: "Secret Stores", url: "/enterprise/secretstores", icon: BookKey },
  { title: "ClusterConfig", url: "/enterprise/clusterconfig", icon: Settings2 },
  { title: "Policy Tester", url: "/enterprise/policytester", icon: BadgeCheck },
];

const itemsVisibility = [
  { title: "Access Logs", url: "/visibility/accesslogs", icon: ShieldEllipsis },
  {
    title: "Authentication Logs",
    url: "/visibility/authenticationlogs",
    icon: ShieldUser,
  },
  { title: "Audit Logs", url: "/visibility/auditlogs", icon: Library },
  { title: "Component Logs", url: "/visibility/componentlogs", icon: Scroll },
  { title: "SSH Sessions", url: "/visibility/ssh", icon: SquareTerminal },
];

export const IconAuthenticationLog = ShieldUser;
export const IconAccessLog = ShieldEllipsis;

const sections = [
  { label: "Core", value: 0, prefix: "/core", items: itemsCore },
  {
    label: "Enterprise",
    value: 1,
    prefix: "/enterprise",
    items: itemsEnterprise,
  },
  { label: "Cluster Management", value: 2, prefix: "/clusterman", items: [] },
  {
    label: "Visibility",
    value: 3,
    prefix: "/visibility",
    items: itemsVisibility,
  },
];

const getBarIdx = (pathname: string) =>
  match(pathname)
    .when(
      (v) => v.startsWith("/core"),
      () => 0,
    )
    .when(
      (v) => v.startsWith("/enterprise"),
      () => 1,
    )
    .when(
      (v) => v.startsWith("/clusterman"),
      () => 2,
    )
    .when(
      (v) => v.startsWith("/visibility"),
      () => 3,
    )
    .otherwise(() => 0);

export default function Sidebar() {
  const loc = useLocation();
  const navigate = useNavigate();
  const [barIdx, setBarIdx] = React.useState(() => getBarIdx(loc.pathname));
  const [dropdownOpen, setDropdownOpen] = React.useState(false);
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    setBarIdx(getBarIdx(loc.pathname));
  }, [loc.pathname]);

  React.useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const activeSection = sections[barIdx];
  const items = activeSection?.items ?? [];

  return (
    <div className="min-h-full w-full flex flex-col">
      <div ref={dropdownRef} className="relative mb-8">
        <button
          onClick={() => setDropdownOpen((v) => !v)}
          className={twMerge(
            "w-full flex items-center justify-between gap-2",
            "px-3 h-9 rounded-lg cursor-pointer",
            "bg-white border border-slate-200",
            "shadow-[0_1px_3px_rgba(15,23,42,0.06)]",
            "transition-[border-color,box-shadow] duration-150",
            "hover:border-slate-300",
            dropdownOpen &&
              "border-slate-300 shadow-[0_2px_8px_rgba(15,23,42,0.09)]",
          )}
        >
          <span className="text-[0.78rem] font-bold text-slate-800 truncate">
            {activeSection.label}
          </span>
          <motion.span
            animate={{ rotate: dropdownOpen ? 180 : 0 }}
            transition={{ duration: 0.2, ease: "easeInOut" }}
            className="flex items-center shrink-0"
          >
            <ChevronDown size={13} className="text-slate-400" />
          </motion.span>
        </button>

        <AnimatePresence initial={false}>
          {dropdownOpen && (
            <motion.div
              key="dropdown"
              initial={{ opacity: 0, y: -6, scaleY: 0.95 }}
              animate={{ opacity: 1, y: 0, scaleY: 1 }}
              exit={{ opacity: 0, y: -6, scaleY: 0.95 }}
              transition={{ duration: 0.15, ease: "easeOut" }}
              style={{ transformOrigin: "top" }}
              className="absolute top-[calc(100%+4px)] left-0 right-0 z-50 bg-white border border-slate-200 rounded-lg shadow-[0_8px_24px_rgba(15,23,42,0.10)] overflow-hidden"
            >
              {sections.map((s) => (
                <button
                  key={s.value}
                  onClick={() => {
                    setBarIdx(s.value);
                    setDropdownOpen(false);
                    match(s.value)
                      .with(0, () => navigate("/core"))
                      .with(1, () => navigate("/enterprise/certificates"))
                      .with(2, () => navigate("/clusterman"))
                      .with(3, () => navigate("/visibility"))
                      .otherwise(() => navigate("/core"));
                  }}
                  className={twMerge(
                    "w-full flex items-center px-3 h-8 cursor-pointer text-left",
                    "text-[0.78rem] font-bold transition-colors duration-100",
                    s.value === barIdx
                      ? "bg-slate-900 text-white"
                      : "text-slate-600 hover:bg-slate-50 hover:text-slate-900",
                  )}
                >
                  {s.label}
                </button>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <nav className="flex flex-col gap-0.5">
        {items.map((item) => {
          const isActive =
            loc.pathname === item.url ||
            loc.pathname.startsWith(`${item.url}/`);
          return (
            <Link
              key={item.title}
              viewTransition
              to={item.url}
              className={twMerge(
                "flex w-full items-center gap-2",
                "py-1.5 px-2.5 rounded-md",
                "text-[0.82rem] font-bold",
                "transition-colors duration-150",
                isActive
                  ? "bg-slate-900 text-white shadow-sm"
                  : "text-slate-600 hover:text-slate-900 hover:bg-slate-200/70",
              )}
            >
              <item.icon size={15} className="shrink-0" />
              <span>{item.title}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
