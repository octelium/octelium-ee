import { AppShell, Drawer } from "@mantine/core";
import {
  useLocalStorage,
  useMediaQuery,
  useReducedMotion,
} from "@mantine/hooks";
import { Menu as MenuIcon } from "lucide-react";
import * as React from "react";
import { useLocation } from "react-router-dom";
import Sidebar from ".";

export default function ConsoleShell(props: {
  header: React.ReactNode;
  children: React.ReactNode;
}) {
  const location = useLocation();
  const desktop = useMediaQuery("(min-width: 48em)");
  const reducedMotion = useReducedMotion();
  const [storedCollapsed, setCollapsed] = useLocalStorage({
    key: "octelium-console-sidebar-collapsed",
    defaultValue: false,
    getInitialValueInEffect: false,
    deserialize: (value) => value === "true",
  });
  const collapsed = storedCollapsed === true;
  const [drawerRoute, setDrawerRoute] = React.useState<string | null>(null);
  const drawerOpened = !desktop && drawerRoute === location.key;
  const closeDrawer = () => setDrawerRoute(null);

  if (desktop && drawerRoute !== null) setDrawerRoute(null);

  return (
    <AppShell
      className="!bg-transparent"
      header={{ height: 60 }}
      navbar={{
        width: collapsed ? 64 : 264,
        breakpoint: "sm",
        collapsed: { mobile: true },
      }}
      padding="md"
      transitionDuration={reducedMotion ? 0 : 240}
      transitionTimingFunction="cubic-bezier(0.22, 1, 0.36, 1)"
    >
      <AppShell.Header
        className="!bg-canvas border-b border-slate-200"
        style={{ zIndex: 200 }}
      >
        <div className="flex h-full items-center">
          <button
            type="button"
            aria-label="Open navigation"
            aria-haspopup="dialog"
            aria-expanded={drawerOpened}
            onClick={() => setDrawerRoute(location.key)}
            className="ml-2 flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-slate-600 outline-none transition-colors hover:bg-slate-200/60 focus-visible:ring-2 focus-visible:ring-slate-400 min-[48em]:hidden"
          >
            <MenuIcon size={20} aria-hidden />
          </button>
          {props.header}
        </div>
      </AppShell.Header>

      <AppShell.Navbar
        p={0}
        className="!bg-canvas !border-slate-200/80"
        style={{ zIndex: 100 }}
        aria-label="Console sidebar"
        inert={!desktop}
      >
        <Sidebar
          collapsed={collapsed}
          onToggle={() => setCollapsed(!collapsed)}
        />
      </AppShell.Navbar>

      <AppShell.Main
        className="!bg-transparent"
        style={{
          paddingInlineEnd:
            "calc(var(--app-shell-padding) + var(--agent-dock-offset, 0px))",
          transitionProperty: "padding-inline-start",
        }}
      >
        {props.children}
      </AppShell.Main>

      <Drawer.Root
        opened={drawerOpened}
        onClose={closeDrawer}
        position="left"
        size="min(320px, calc(100vw - 24px))"
        padding={0}
        zIndex={220}
        transitionProps={{
          duration: reducedMotion ? 0 : 240,
          timingFunction: "cubic-bezier(0.22, 1, 0.36, 1)",
        }}
      >
        <Drawer.Overlay backgroundOpacity={0.28} />
        <Drawer.Content
          aria-label="Console navigation"
          className="!bg-canvas !overflow-hidden"
        >
          <Drawer.Body className="!h-full !p-0">
            <Sidebar onClose={closeDrawer} />
          </Drawer.Body>
        </Drawer.Content>
      </Drawer.Root>
    </AppShell>
  );
}
