import { Inbox, ShieldCheck } from "lucide-react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const NAV = [
  { to: "/user/requests", label: "My Requests", icon: Inbox },
  { to: "/user/new", label: "New Request", icon: ShieldCheck },
];

const UserLayout = () => {
  const loc = useLocation();

  return (
    <div className="min-h-screen w-full bg-slate-50">
      <header className="sticky top-0 z-20 w-full border-b border-slate-200 bg-white/80 backdrop-blur-sm">
        <div className="max-w-5xl mx-auto px-6 h-14 flex items-center justify-between">
          <Link to="/user/requests" className="flex items-center gap-2">
            <div className="flex items-center justify-center w-7 h-7 rounded-lg bg-slate-900 text-white">
              <ShieldCheck size={15} strokeWidth={2.5} />
            </div>
            <span className="text-[0.9rem] font-bold text-slate-900 tracking-tight">
              Octelium Access
            </span>
          </Link>

          <nav className="flex items-center gap-1">
            {NAV.map((n) => {
              const active = loc.pathname.startsWith(n.to);
              const Icon = n.icon;
              return (
                <Link
                  key={n.to}
                  to={n.to}
                  className={twMerge(
                    "flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[0.78rem] font-bold transition-colors duration-150",
                    active
                      ? "bg-slate-900 text-white"
                      : "text-slate-500 hover:text-slate-900 hover:bg-slate-100",
                  )}
                >
                  <Icon size={14} strokeWidth={2.5} />
                  {n.label}
                </Link>
              );
            })}
          </nav>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 py-8">
        <Outlet />
      </main>
    </div>
  );
};

export default UserLayout;
