/// <reference types="vite-plugin-svgr/client" />

import Logo from "@/assets/l03.svg?react";
import { Avatar } from "@/ui";
import { useAppSelector } from "@/utils/hooks";
import { Link } from "react-router-dom";

const TopBar = () => {
  const status = useAppSelector((state) => state.settings.status);

  const picURL =
    status?.session?.metadata?.picURL ?? status?.user?.metadata?.picURL;

  const displayName =
    status?.user?.metadata?.displayName?.trim() ||
    status?.user?.metadata?.name?.trim();

  return (
    <nav
      aria-label="Account"
      className="flex h-[60px] w-full min-w-0 items-center gap-3 px-2 sm:px-4"
    >
      <Link
        to="/user/requests"
        className="flex flex-none items-center rounded-md outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        aria-label="Go to My Requests"
      >
        <Logo className="h-auto w-[108px] text-slate-900 sm:w-[152px]" />
      </Link>

      <div className="flex-1" />

      <Link
        to="/settings"
        aria-label={displayName ? `Account settings for ${displayName}` : "Account settings"}
        className="group flex h-10 min-w-0 flex-none items-center gap-2 rounded-lg px-2 outline-none transition-colors duration-150 hover:bg-slate-200/60 focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <Avatar
          src={picURL}
          name={displayName}
          size="sm"
          className="ring-2 ring-white ring-offset-1 ring-offset-canvas"
        />
      </Link>
    </nav>
  );
};

export default TopBar;
