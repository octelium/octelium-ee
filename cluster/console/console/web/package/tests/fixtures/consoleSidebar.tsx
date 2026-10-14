import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";
import ConsoleShell from "@/components/SideBar/Shell";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import * as React from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter, useLocation, useNavigate } from "react-router-dom";

const params = new URLSearchParams(window.location.search);

const Content = () => {
  const location = useLocation();
  const navigate = useNavigate();
  React.useEffect(() => {
    window.sidebarNavigate = navigate;
  }, [navigate]);

  return (
    <ConsoleShell
      header={
        <div className="flex min-w-0 flex-1 items-center px-4 text-sm font-semibold text-slate-900">
          Octelium
        </div>
      }
    >
      <div className="mx-auto w-full max-w-[var(--page-max-width)] py-4">
        <h1 className="text-xl font-semibold text-slate-900">Console</h1>
        <output
          data-testid="location"
          className="mt-2 block break-all text-body text-slate-600"
        >
          {location.pathname}
          {location.search}
        </output>
        <div className="mt-6 rounded-xl border border-slate-200 bg-white p-5">
          <label className="flex flex-col gap-2 text-body text-slate-700">
            Page input
            <input className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-slate-900" />
          </label>
        </div>
        <div className="h-[120vh]" />
      </div>
    </ConsoleShell>
  );
};

declare global {
  interface Window {
    sidebarNavigate: ReturnType<typeof useNavigate>;
  }
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider
      theme={themeMantine}
      forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
    >
      <MemoryRouter initialEntries={[params.get("path") ?? "/core/services"]}>
        <Content />
      </MemoryRouter>
    </MantineProvider>
  </React.StrictMode>,
);
