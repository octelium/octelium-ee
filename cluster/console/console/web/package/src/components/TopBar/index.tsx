/// <reference types="vite-plugin-svgr/client" />

import Logo from "@/assets/l03.svg?react";
import { setUseListSearch } from "@/features/settings/slice";
import { useAppDispatch, useAppSelector } from "@/utils/hooks";
import * as React from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import SearchList from "../SearchList";

const TopBar = () => {
  const navigate = useNavigate();
  const settings = useAppSelector((state) => state.settings);
  const location = useLocation();
  const dispatch = useAppDispatch();

  const picURL =
    settings.status?.session?.metadata?.picURL ??
    settings.status?.user?.metadata?.picURL;

  const displayName =
    settings.status?.user?.metadata?.displayName ??
    settings.status?.user?.metadata?.name;

  React.useEffect(() => {
    dispatch(
      setUseListSearch({
        useListSearch:
          location.pathname.startsWith("/core/") &&
          /^\/core\/[^\/]+$/.test(location.pathname),
      }),
    );
  }, [location.pathname, dispatch]);

  return (
    <nav className="w-full h-[60px] flex items-center px-4 gap-4">
      <Link
        to="/"
        className="flex-none flex items-center"
        aria-label="Go to home"
      >
        <Logo className="w-[120px] md:w-[160px] h-auto" />
      </Link>

      <div className="flex-1 flex items-center">
        {settings.useListSearch && (
          <div className="max-w-sm w-full">
            <SearchList />
          </div>
        )}
      </div>

      <button
        onClick={() => navigate("/settings")}
        className="flex-none flex items-center gap-2 group cursor-pointer"
        title={displayName ? `Settings — ${displayName}` : "Settings"}
      >
        {displayName && (
          <span className="text-[0.72rem] font-semibold text-slate-500 group-hover:text-slate-800 transition-colors duration-150 hidden md:block">
            {displayName}
          </span>
        )}

        <div
          className={twMerge(
            "w-8 h-8 rounded-full shrink-0 overflow-hidden",
            "ring-2 ring-white ring-offset-1 ring-offset-slate-100",
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
                    .split(" ")
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
