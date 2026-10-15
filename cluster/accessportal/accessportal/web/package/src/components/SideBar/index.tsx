import {
  ClipboardCheck,
  Inbox,
  ListChecks,
  Plus,
  Settings,
  ShieldCheck,
  X,
} from "lucide-react";
import * as React from "react";
import { NavLink } from "react-router-dom";
import { twMerge } from "@/lib/utils";

import RightSidebar from "./RightSidebar";

type NavItem = {
  to: string;
  label: string;
  icon: React.ComponentType<{ size?: number; strokeWidth?: number }>;
  end?: boolean;
};

const NAV_GROUPS: { label: string; items: NavItem[] }[] = [
  {
    label: "Requests",
    items: [
      { to: "/user/requests", label: "My Requests", icon: Inbox },
      { to: "/user/new", label: "New Request", icon: Plus, end: true },
    ],
  },
  {
    label: "Reviews",
    items: [
      { to: "/reviewer/requests", label: "Review Queue", icon: ClipboardCheck },
      { to: "/reviewer/reviews", label: "My Reviews", icon: ListChecks },
    ],
  },
];

const NavigationLink = (props: {
  item: NavItem;
  onNavigate?: () => void;
}) => (
  <NavLink
    to={props.item.to}
    end={props.item.end}
    onClick={props.onNavigate}
    className={({ isActive }) =>
      twMerge(
        "flex min-h-9 items-center gap-2.5 rounded-lg px-2.5 text-body outline-none transition-[background-color,color,box-shadow] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400 [@media(pointer:coarse)]:min-h-11",
        isActive
          ? "bg-white font-semibold text-slate-900 shadow-card ring-1 ring-slate-200"
          : "font-medium text-slate-600 hover:bg-slate-200/50 hover:text-slate-900",
      )
    }
  >
    <props.item.icon size={16} strokeWidth={1.8} aria-hidden />
    {props.item.label}
  </NavLink>
);

const Sidebar = (props: { onNavigate?: () => void; onClose?: () => void }) => (
  <div className="flex h-full min-h-0 w-full flex-col">
    <div className="flex h-16 shrink-0 items-center gap-2.5 px-4">
      <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-700 shadow-card">
        <ShieldCheck size={17} strokeWidth={1.8} aria-hidden />
      </div>
      <span className="text-body font-semibold text-slate-800">
        Access portal
      </span>
      {props.onClose && (
        <button
          type="button"
          aria-label="Close navigation"
          onClick={props.onClose}
          className="ml-auto flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 outline-none transition-colors hover:bg-slate-200/60 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <X size={17} aria-hidden />
        </button>
      )}
    </div>
    <nav
      aria-label="Access portal"
      className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overscroll-contain px-2 pb-4 pt-2 [scrollbar-width:thin]"
    >
      {NAV_GROUPS.map((group) => (
        <div key={group.label} className="flex flex-col gap-1">
          <span className="mb-1 px-2.5 text-xs font-medium text-slate-500">
            {group.label}
          </span>
          {group.items.map((item) => (
            <NavigationLink
              key={item.to}
              item={item}
              onNavigate={props.onNavigate}
            />
          ))}
        </div>
      ))}
    </nav>

    <div className="shrink-0 space-y-1 border-t border-slate-200/80 p-2">
      <NavigationLink
        item={{ to: "/settings", label: "Settings", icon: Settings }}
        onNavigate={props.onNavigate}
      />
      <RightSidebar />
    </div>
  </div>
);

export default Sidebar;
