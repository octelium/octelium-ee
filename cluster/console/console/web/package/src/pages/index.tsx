import { AgentDockHost } from "@/components/AgentDock";
import { AgentProvider } from "@/components/AgentDock/provider";
import { CommandPaletteProvider } from "@/components/CommandPalette";
import { useTrackVisits } from "@/components/CommandPalette/track";
import { AppErrorBoundary } from "@/components/ErrorBoundary";
import Footer from "@/components/Footer";
import PageHeader from "@/components/PageHeader";
import { PageHeaderProvider } from "@/components/PageHeader/context";
import ConsoleShell from "@/components/SideBar/Shell";
import TopBar from "@/components/TopBar";
import { Toaster } from "@/components/ui/sonner";
import { setStatus } from "@/features/settings/slice";
import { getClientUser } from "@/utils/client";
import { useAppDispatch } from "@/utils/hooks";
import { getRedirectPath } from "@/utils/redirect";
import { useQuery } from "@tanstack/react-query";
import * as React from "react";
import {
  Navigate,
  Outlet,
  ScrollRestoration,
  useLocation,
} from "react-router-dom";

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

export default function Root() {
  const dispatch = useAppDispatch();
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
  const redirect = getRedirectPath(
    urlSearchParams.get("redirect"),
    window.location.origin,
  );

  if (redirect) {
    urlSearchParams.delete("redirect");
    return <Navigate to={redirect} replace />;
  }

  return (
    <ShellProviders>
      <div className="flex min-h-screen w-full flex-col bg-canvas antialiased">
        <title>Octelium Console</title>
        <ScrollRestoration />

        <ConsoleShell header={<TopBar />}>
          <PageHeader />
          <div className="mx-auto w-full max-w-[var(--page-max-width)]">
            <AppErrorBoundary>
              <Outlet />
            </AppErrorBoundary>
            {!isAgent && <Footer />}
          </div>
        </ConsoleShell>

        <AgentDockHost />

        <Toaster
          position="bottom-center"
          toastOptions={{ className: "bg-slate-900 font-semibold text-white" }}
        />
      </div>
    </ShellProviders>
  );
}
