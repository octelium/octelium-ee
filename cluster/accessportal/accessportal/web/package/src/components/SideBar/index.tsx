import { ClipboardCheck, Inbox, ListChecks, Plus } from "lucide-react";
import * as React from "react";
import { NavLink } from "react-router-dom";
import { twMerge } from "tailwind-merge";

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

const Sidebar = (props: { onNavigate?: () => void }) => (
  <div className="flex flex-col h-full w-full">
    <nav className="min-h-0 flex-1 overflow-y-auto px-3 py-5 flex flex-col gap-6">
      {NAV_GROUPS.map((group) => (
        <div key={group.label} className="flex flex-col gap-1">
          <span className="px-3 mb-1 text-[0.62rem] font-bold uppercase tracking-[0.1em] text-slate-400">
            {group.label}
          </span>
          {group.items.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                onClick={props.onNavigate}
                className={({ isActive }) =>
                  twMerge(
                    "flex items-center gap-2.5 px-3 py-2 rounded-lg text-[0.8rem] font-bold transition-colors duration-150",
                    isActive
                      ? "bg-slate-900 text-white shadow-[0_1px_3px_rgba(15,23,42,0.18)]"
                      : "text-slate-500 hover:text-slate-900 hover:bg-slate-100",
                  )
                }
              >
                <Icon size={15} strokeWidth={2.5} />
                {item.label}
              </NavLink>
            );
          })}
        </div>
      ))}
    </nav>

    <div className="shrink-0 border-t border-slate-100 px-3 py-3">
      <RightSidebar />
    </div>
  </div>
);

export default Sidebar;
