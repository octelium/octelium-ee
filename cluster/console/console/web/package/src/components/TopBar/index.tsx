/// <reference types="vite-plugin-svgr/client" />

import Logo from "@/assets/l03.svg?react";
import { getDomain } from "@/utils";
import { getClientAuth } from "@/utils/client";
import { useAppSelector } from "@/utils/hooks";
import { Button, Menu, Modal } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import {
  ChevronDown,
  Loader2,
  LockKeyhole,
  LogOut,
  Settings,
  X,
} from "lucide-react";
import { Link, useNavigate } from "react-router-dom";

const initials = (name: string) =>
  name
    .split(" ")
    .slice(0, 2)
    .map((part) => part.at(0))
    .join("");

const TopBar = () => {
  const navigate = useNavigate();
  const settings = useAppSelector((state) => state.settings);
  const [logoutOpened, { open: openLogout, close: closeLogout }] =
    useDisclosure(false);

  const mutationLogout = useMutation({
    mutationFn: async () => {
      await getClientAuth().logout({});
    },
    onSuccess: () => {
      window.location.reload();
    },
  });

  const picURL =
    settings.status?.session?.metadata?.picURL ??
    settings.status?.user?.metadata?.picURL;

  const displayName =
    settings.status?.user?.metadata?.displayName ??
    settings.status?.user?.metadata?.name;

  return (
    <nav className="flex h-[60px] w-full min-w-0 items-center gap-2 overflow-hidden px-2 sm:gap-3 sm:px-4 lg:gap-5">
      <Link
        to="/"
        className="flex flex-none items-center"
        aria-label="Go to home"
      >
        <Logo className="h-auto w-[92px] sm:w-[120px] lg:w-[152px]" />
      </Link>

      <div className="flex-1" />

      <Menu position="bottom-end" width={224} withinPortal>
        <Menu.Target>
          <button
            type="button"
            className="group flex h-10 flex-none cursor-pointer items-center gap-2 rounded-lg px-1 transition-colors duration-150 hover:bg-white/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-500 sm:px-2"
            aria-label="Account menu"
          >
            {displayName && (
              <span className="hidden max-w-36 truncate text-body font-normal text-slate-600 transition-colors duration-150 group-hover:text-slate-900 lg:block">
                {displayName}
              </span>
            )}

            <div className="h-7 w-7 shrink-0 overflow-hidden rounded-full ring-2 ring-white ring-offset-1 ring-offset-slate-100 transition-[box-shadow] duration-150 group-hover:ring-slate-300 sm:h-8 sm:w-8">
              {picURL ? (
                <img
                  src={picURL}
                  alt=""
                  className="h-full w-full object-cover"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center bg-slate-700">
                  {displayName ? (
                    <span className="text-micro font-semibold uppercase text-white">
                      {initials(displayName)}
                    </span>
                  ) : (
                    <div className="h-full w-full bg-sky-700" />
                  )}
                </div>
              )}
            </div>

            <ChevronDown
              size={13}
              strokeWidth={2.25}
              className="hidden shrink-0 text-slate-500 sm:block"
            />
          </button>
        </Menu.Target>

        <Menu.Dropdown>
          {displayName && <Menu.Label>{displayName}</Menu.Label>}
          <Menu.Item
            leftSection={<Settings size={14} />}
            onClick={() => navigate("/settings")}
          >
            Settings
          </Menu.Item>
          <Menu.Item
            component="a"
            href={`https://${getDomain()}/authenticators`}
            leftSection={<LockKeyhole size={14} />}
          >
            Authenticators
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            color="red"
            leftSection={<LogOut size={14} />}
            onClick={openLogout}
          >
            Log out
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>

      <Modal
        opened={logoutOpened}
        onClose={closeLogout}
        centered
        withCloseButton={false}
        padding={0}
        size="sm"
      >
        <div className="flex flex-col">
          <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50/70 px-5 py-3.5">
            <LogOut
              size={14}
              className="shrink-0 text-red-500"
              strokeWidth={2.25}
            />
            <span className="text-body font-semibold text-slate-900">
              Log out
            </span>
          </div>

          <p className="px-5 py-4 text-body font-normal text-slate-600">
            Are you sure you want to log out of this session?
          </p>

          <div className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-5 py-3">
            <Button
              variant="default"
              size="sm"
              leftSection={<X size={13} strokeWidth={2.25} />}
              onClick={closeLogout}
            >
              Cancel
            </Button>
            <Button
              variant="filled"
              color="red"
              size="sm"
              leftSection={
                mutationLogout.isPending ? (
                  <Loader2
                    size={13}
                    className="animate-spin"
                    strokeWidth={2.25}
                  />
                ) : (
                  <LogOut size={13} strokeWidth={2.25} />
                )
              }
              loading={mutationLogout.isPending}
              onClick={() => mutationLogout.mutate()}
            >
              {mutationLogout.isPending ? "Logging out…" : "Log out"}
            </Button>
          </div>
        </div>
      </Modal>
    </nav>
  );
};

export default TopBar;
