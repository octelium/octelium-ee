import {
  BadgeCheck,
  BookKey,
  Boxes,
  BrainCircuit,
  Building2,
  Check,
  ChartNoAxesCombined,
  ChevronDown,
  ClipboardCheck,
  Cpu,
  Crown,
  DoorClosed,
  Eye,
  Fingerprint,
  Folder,
  Globe,
  Globe2,
  Inbox,
  KeyRound,
  LaptopMinimal,
  Layers,
  LayoutDashboard,
  Library,
  Link2,
  LockKeyhole,
  LockOpen,
  PanelTop,
  Plug,
  Send,
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
  UserCheck,
  Users,
} from "lucide-react";

import { AnimatePresence, motion } from "framer-motion";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";

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

const itemsAccess = [
  { title: "Catalogs", url: "/access/catalogs", icon: Layers },
  { title: "Requests", url: "/access/requests", icon: Inbox },
  { title: "Reviews", url: "/access/reviews", icon: ClipboardCheck },
  { title: "Policies", url: "/access/policies", icon: Shield },
  { title: "Integrations", url: "/access/integrations", icon: Plug },
  {
    title: "Integration Identities",
    url: "/access/integrationidentities",
    icon: Link2,
  },
  {
    title: "Integration Bindings",
    url: "/access/integrationbindings",
    icon: Send,
  },
  { title: "Secrets", url: "/access/secrets", icon: KeyRound },
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
  { title: "Metrics", url: "/visibility/metrics", icon: ChartNoAxesCombined },
  { title: "LLM", url: "/visibility/llm", icon: BrainCircuit },
];

export const IconAuthenticationLog = ShieldUser;
export const IconAccessLog = ShieldEllipsis;

const sections = [
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
  },
  {
    label: "Cluster Management",
    prefix: "/clusterman",
    defaultPath: "/clusterman",
    icon: Settings2,
    items: [],
  },
  {
    label: "Visibility",
    prefix: "/visibility",
    defaultPath: "/visibility",
    icon: Eye,
    items: itemsVisibility,
  },
];

const getBarIdx = (pathname: string) => {
  const idx = sections.findIndex(
    (section) =>
      pathname === section.prefix || pathname.startsWith(`${section.prefix}/`),
  );
  return idx >= 0 ? idx : 0;
};

export default function Sidebar() {
  const loc = useLocation();
  const navigate = useNavigate();
  const [dropdownOpen, setDropdownOpen] = React.useState(false);
  const dropdownRef = React.useRef<HTMLDivElement>(null);
  const triggerRef = React.useRef<HTMLButtonElement>(null);
  const optionRefs = React.useRef<Array<HTMLButtonElement | null>>([]);
  const dropdownID = React.useId();
  const barIdx = getBarIdx(loc.pathname);

  React.useEffect(() => {
    const handler = (event: PointerEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setDropdownOpen(false);
      }
    };
    document.addEventListener("pointerdown", handler);
    return () => document.removeEventListener("pointerdown", handler);
  }, []);

  React.useEffect(() => {
    setDropdownOpen(false);
  }, [loc.pathname]);

  const focusOption = (index: number) => {
    optionRefs.current[index]?.focus();
  };

  const openAndFocus = (index: number) => {
    setDropdownOpen(true);
    requestAnimationFrame(() => focusOption(index));
  };

  const handleMenuKeyDown = (event: React.KeyboardEvent) => {
    const focusedIndex = optionRefs.current.findIndex(
      (option) => option === document.activeElement,
    );

    switch (event.key) {
      case "Escape":
        event.preventDefault();
        setDropdownOpen(false);
        triggerRef.current?.focus();
        break;
      case "ArrowDown":
        event.preventDefault();
        focusOption((focusedIndex + 1 + sections.length) % sections.length);
        break;
      case "ArrowUp":
        event.preventDefault();
        focusOption((focusedIndex - 1 + sections.length) % sections.length);
        break;
      case "Home":
        event.preventDefault();
        focusOption(0);
        break;
      case "End":
        event.preventDefault();
        focusOption(sections.length - 1);
        break;
    }
  };

  const activeSection = sections[barIdx] ?? sections[0];
  const ActiveIcon = activeSection.icon;
  const items = activeSection?.items ?? [];

  const isOverview = loc.pathname === "/";

  return (
    <div className="min-h-full w-full flex flex-col">
      <Link
        to="/"
        viewTransition
        className={`text-body font-semibold ${twMerge(
          "mb-3 flex w-full items-center gap-2",
          "py-2 px-2.5 rounded-lg",
          "transition-colors duration-150",
          isOverview
            ? "bg-slate-900 text-white shadow-sm"
            : "text-slate-600 hover:text-slate-900 hover:bg-slate-200/70",
        )}`}
      >
        <LayoutDashboard
          size={15}
          className="shrink-0"
          strokeWidth={isOverview ? 2.5 : 2}
        />
        <span>Overview</span>
      </Link>

      <div ref={dropdownRef} className="relative mb-7">
        <button
          ref={triggerRef}
          type="button"
          onClick={() => setDropdownOpen((v) => !v)}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown" || event.key === "ArrowUp") {
              event.preventDefault();
              openAndFocus(
                event.key === "ArrowDown" ? barIdx : sections.length - 1,
              );
            }
          }}
          aria-haspopup="menu"
          aria-expanded={dropdownOpen}
          aria-controls={dropdownOpen ? dropdownID : undefined}
          className={twMerge(
            "group flex h-14 w-full cursor-pointer items-center gap-3 rounded-xl border border-slate-200 bg-white px-2.5 text-left",
            "shadow-card",
            "transition-[border-color,box-shadow,background-color] duration-200",
            "hover:border-slate-300 hover:bg-slate-50/70 hover:shadow-raised",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-offset-2",
            dropdownOpen &&
              "border-slate-300 bg-slate-50/80 shadow-raised",
          )}
        >
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white shadow-sm transition-transform duration-200 group-hover:scale-[1.03]">
            <ActiveIcon size={15} strokeWidth={2.25} />
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-micro font-semibold uppercase tracking-[0.09em] text-slate-500">
              API workspace
            </span>
            <span className="mt-0.5 block truncate text-body font-semibold text-slate-800">
              {activeSection.label}
            </span>
          </span>
          <motion.span
            animate={{ rotate: dropdownOpen ? 180 : 0 }}
            transition={{ duration: 0.25, ease: "easeInOut" }}
            className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-slate-500 transition-colors duration-200 group-hover:bg-white group-hover:text-slate-600"
          >
            <ChevronDown size={14} strokeWidth={2.5} />
          </motion.span>
        </button>

        <AnimatePresence initial={false}>
          {dropdownOpen && (
            <motion.div
              key="dropdown"
              id={dropdownID}
              role="menu"
              aria-label="Choose API workspace"
              onKeyDown={handleMenuKeyDown}
              initial={{ opacity: 0, y: -7, scale: 0.97 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -5, scale: 0.98 }}
              transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
              style={{ transformOrigin: "top" }}
              className="absolute left-0 right-0 top-[calc(100%+7px)] z-50 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-overlay"
            >
              <div className="border-b border-slate-100 bg-slate-50/70 px-3.5 py-3">
                <p className="text-xs font-semibold text-slate-700">
                  Choose workspace
                </p>
                <p className="mt-0.5 text-micro font-normal text-slate-500">
                  Switch management API context
                </p>
              </div>

              <div className="space-y-1 p-1.5">
                {sections.map((s, idx) => {
                  const Icon = s.icon;
                  const isActive = idx === barIdx;
                  return (
                    <button
                      key={s.prefix}
                      ref={(element) => {
                        optionRefs.current[idx] = element;
                      }}
                      type="button"
                      role="menuitemradio"
                      aria-checked={isActive}
                      tabIndex={isActive ? 0 : -1}
                      onClick={() => {
                        setDropdownOpen(false);
                        if (!isActive) navigate(s.defaultPath);
                      }}
                      className={twMerge(
                        "group/item flex min-h-12 w-full cursor-pointer items-center gap-3 rounded-lg px-2.5 py-2 text-left",
                        "transition-[background-color,color] duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400 focus-visible:ring-inset",
                        isActive
                          ? "bg-slate-900 text-white"
                          : "text-slate-700 hover:bg-slate-100 hover:text-slate-900",
                      )}
                    >
                      <span
                        className={twMerge(
                          "flex h-8 w-8 shrink-0 items-center justify-center rounded-lg transition-colors duration-200",
                          isActive
                            ? "bg-white/10 text-white"
                            : "border border-slate-200 bg-white text-slate-500 group-hover/item:border-slate-300 group-hover/item:text-slate-700",
                        )}
                      >
                        <Icon size={14} strokeWidth={2.25} />
                      </span>
                      <span className="min-w-0 flex-1 truncate text-body font-semibold">
                        {s.label}
                      </span>
                      {isActive && (
                        <Check
                          size={14}
                          strokeWidth={2.75}
                          className="shrink-0 text-slate-200"
                        />
                      )}
                    </button>
                  );
                })}
              </div>
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
              className={`text-body font-semibold ${twMerge(
                "flex w-full items-center gap-2",
                "py-1.5 px-2.5 rounded-md",
                "transition-colors duration-150",
                isActive
                  ? "bg-slate-900 text-white shadow-sm"
                  : "text-slate-600 hover:text-slate-900 hover:bg-slate-200/70",
              )}`}
            >
              <item.icon
                size={15}
                className="shrink-0"
                strokeWidth={isActive ? 2.5 : 2}
              />
              <span>{item.title}</span>
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
