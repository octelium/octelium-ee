/// <reference types="vite-plugin-svgr/client" />

import Logo from "@/assets/l03.svg?react";
import { useAppSelector } from "@/utils/hooks";
import { Link, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const TopBar = () => {
  const navigate = useNavigate();
  const settings = useAppSelector((state) => state.settings);

  const picURL =
    settings.status?.session?.metadata?.picURL ??
    settings.status?.user?.metadata?.picURL;

  const displayName =
    settings.status?.user?.metadata?.displayName?.trim() ||
    settings.status?.user?.metadata?.name?.trim();

  return (
    <nav className="flex h-[60px] w-full min-w-0 items-center gap-3 px-2 sm:px-4">
      <Link
        to="/"
        className="flex-none flex items-center"
        aria-label="Go to home"
      >
        <Logo className="h-auto w-[108px] text-slate-900 sm:w-[152px]" />
      </Link>

      <div className="flex-1 flex items-center"></div>

      <button
        type="button"
        onClick={() => navigate("/settings")}
        aria-label={displayName ? `Open settings for ${displayName}` : "Open settings"}
        className="group flex h-10 min-w-0 flex-none items-center gap-2 rounded-lg px-2 outline-none transition-colors duration-150 hover:bg-white/70 focus-visible:ring-2 focus-visible:ring-slate-400"
        title={displayName ? `Settings — ${displayName}` : "Settings"}
      >
        {displayName && (
          <span className="hidden max-w-48 truncate text-xs font-medium text-slate-600 transition-colors duration-150 group-hover:text-slate-900 md:block">
            {displayName}
          </span>
        )}

        <div
          className={twMerge(
            "w-8 h-8 rounded-full shrink-0 overflow-hidden",
            "ring-2 ring-white ring-offset-1 ring-offset-canvas",
            "transition-[ring] duration-150",
            "group-hover:ring-slate-300",
          )}
        >
          {picURL ? (
            <img
              src={picURL}
              alt={displayName ?? "User"}
              className="w-full h-full object-cover"
            />
          ) : (
            <div className="w-full h-full bg-slate-700 flex items-center justify-center">
              {displayName ? (
                <span className="text-[0.6rem] font-bold text-white uppercase">
                  {displayName
                    .split(/\s+/)
                    .slice(0, 2)
                    .map((w) => w.at(0))
                    .join("")}
                </span>
              ) : (
                <div className="w-full h-full bg-sky-700" />
              )}
            </div>
          )}
        </div>
      </button>
    </nav>
  );
};

export default TopBar;
