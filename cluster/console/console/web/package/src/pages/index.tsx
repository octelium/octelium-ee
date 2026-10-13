import { AgentDockHost } from "@/components/AgentDock";
import { AgentProvider } from "@/components/AgentDock/provider";
import { CommandPaletteProvider } from "@/components/CommandPalette";
import { useTrackVisits } from "@/components/CommandPalette/track";
import { AppErrorBoundary } from "@/components/ErrorBoundary";
import Footer from "@/components/Footer";
import PageHeader from "@/components/PageHeader";
import { PageHeaderProvider } from "@/components/PageHeader/context";
import SideBar from "@/components/SideBar";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { setStatus } from "@/features/settings/slice";
import { getClientUser } from "@/utils/client";
import { useAppDispatch } from "@/utils/hooks";
import { getRedirectPath } from "@/utils/redirect";
import { AppShell, Burger } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import { Navigate, Outlet, ScrollRestoration, useLocation } from "react-router-dom";

import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";

const ShellProviders = (props: { children: React.ReactNode }) => (
  <PageHeaderProvider>
    <AgentProvider>
      <CommandPaletteProvider>{props.children}</CommandPaletteProvider>
    </AgentProvider>
  </PageHeaderProvider>
);

export default () => {
  const dispatch = useAppDispatch();
  const [opened, { toggle }] = useDisclosure();
  const { pathname } = useLocation();
  const isAgent = pathname === "/agent" || pathname.startsWith("/agent/");
  useTrackVisits();

  const statusQuery = useQuery({
    queryKey: ["user", "status"],
    queryFn: async () => (await getClientUser().getStatus({})).response,
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  });

  React.useEffect(() => {
    if (statusQuery.data) {
      dispatch(setStatus({ status: statusQuery.data }));
    }
  }, [dispatch, statusQuery.data]);

  const urlSearchParams = new URLSearchParams(window.location.search);
  const redirect = getRedirectPath(urlSearchParams.get("redirect"), window.location.origin);

  if (redirect) {
    urlSearchParams.delete("redirect");
    return <Navigate to={redirect} replace />;
  }

  return (
    <ShellProviders>
      <div className="flex min-h-screen w-full flex-col bg-canvas antialiased">
        <title>Octelium Console</title>
        <ScrollRestoration />

        <AppShell
          className="!bg-transparent"
          header={{ height: 60 }}
          navbar={{
            width: 264,
            breakpoint: "sm",
            collapsed: { mobile: !opened },
          }}
          padding="md"
        >
          <AppShell.Header
            className="!bg-canvas border-b border-slate-200"
            style={{ zIndex: 200 }}
          >
            <div className="flex h-full flex-row items-center">
              <Burger
                opened={opened}
                onClick={toggle}
                hiddenFrom="sm"
                size="sm"
              />
              <TopBar />
            </div>
          </AppShell.Header>

          <AppShell.Navbar p="md" className="!bg-canvas" style={{ zIndex: 10 }}>
            <SideBar />
          </AppShell.Navbar>

          <AppShell.Main
            className="!bg-transparent"
            style={{
              paddingInlineEnd:
                "calc(var(--app-shell-padding) + var(--agent-dock-offset, 0px))",
              transitionDuration: "0ms",
            }}
          >
            <PageHeader />
            <div className="mx-auto w-full max-w-[var(--page-max-width)]">
              <AppErrorBoundary>
                <Outlet />
              </AppErrorBoundary>
              {!isAgent && <Footer />}
            </div>
          </AppShell.Main>
        </AppShell>

        <AgentDockHost />

        <Toaster
          position="bottom-center"
          toastOptions={{ className: "bg-slate-900 font-semibold text-white" }}
        />
      </div>
    </ShellProviders>
  );
};
