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
    <nav className="flex h-[60px] min-w-0 w-full items-center gap-2 overflow-hidden px-2 sm:gap-3 sm:px-4 lg:gap-5">
      <Link
        to="/"
        className={twMerge(
          "flex-none items-center",
          settings.useListSearch ? "hidden min-[390px]:flex" : "flex",
        )}
        aria-label="Go to home"
      >
        <Logo className="h-auto w-[92px] sm:w-[120px] lg:w-[152px]" />
      </Link>

      <div className="flex min-w-0 flex-1 items-center justify-center sm:justify-start">
        {settings.useListSearch && (
          <div className="min-w-0 w-full max-w-xl">
            <SearchList />
          </div>
        )}
      </div>

      <button
        type="button"
        onClick={() => navigate("/settings")}
        className="group flex h-10 flex-none cursor-pointer items-center gap-2 rounded-lg px-1 transition-colors duration-500 hover:bg-white/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 sm:px-2"
        title={displayName ? `Settings — ${displayName}` : "Settings"}
        aria-label={displayName ? `Open settings for ${displayName}` : "Open settings"}
      >
        {displayName && (
          <span className="hidden max-w-36 truncate text-[0.72rem] font-semibold text-slate-500 transition-colors duration-500 group-hover:text-slate-800 lg:block">
            {displayName}
          </span>
        )}

        <div
          className={twMerge(
            "w-7 h-7 sm:w-8 sm:h-8 rounded-full shrink-0 overflow-hidden",
            "ring-2 ring-white ring-offset-1 ring-offset-slate-100",
            "transition-[ring] duration-500",
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
