import { AskAgentButton } from "@/components/AgentDock/AskButtons";
import { useOptionalAgent } from "@/components/AgentDock/context";
import { approveNextNavigation, useHasDirtyForm } from "@/utils/forms";
import { getAPIKindFromPath } from "@/utils/pb";
import { Drawer } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import * as React from "react";
import { useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import ResourceItemPage from "./ResourceItemPage";

const drawerID = "resource-item-drawer";

const isDrawerEscape = (event: KeyboardEvent) => {
  const target = event.target;
  if (
    event.key !== "Escape" ||
    event.defaultPrevented ||
    event.isComposing ||
    !(target instanceof Element) ||
    target.getAttribute("data-mantine-stop-propagation") === "true"
  ) {
    return false;
  }
  return (
    target === document.body ||
    !!document.getElementById(drawerID)?.contains(target)
  );
};

const confirmClose = (dirty: boolean) => {
  if (!dirty) return true;
  if (!window.confirm("Discard unsaved changes?")) return false;
  approveNextNavigation();
  return true;
};

const ResourceItemDrawer = () => {
  const [opened, setOpened] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const { name } = useParams();
  const state = location.state as { returnTo?: string } | null;
  const segments = location.pathname.split("/").filter(Boolean);
  const parentPath = `/${segments.slice(0, 2).join("/")}`;
  const resourcePath = `/${segments.slice(0, 3).join("/")}`;
  const returnTo = state?.returnTo ?? parentPath;
  const kind = getAPIKindFromPath(location.pathname)?.kind;
  const hasDirtyForm = useHasDirtyForm();
  const agent = useOptionalAgent();
  const docked = useMediaQuery("(min-width: 80em)");
  const dockOpen = !!agent?.opened && !agent.hidden;

  useEffect(() => {
    let openFrame = 0;
    const mountFrame = requestAnimationFrame(() => {
      openFrame = requestAnimationFrame(() => setOpened(true));
    });
    return () => {
      cancelAnimationFrame(mountFrame);
      cancelAnimationFrame(openFrame);
    };
  }, [resourcePath]);

  const requestClose = () => {
    if (confirmClose(hasDirtyForm)) setOpened(false);
  };

  useEffect(() => {
    if (!opened) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (isDrawerEscape(event) && confirmClose(hasDirtyForm)) {
        setOpened(false);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [opened, hasDirtyForm]);

  return (
    <Drawer
      id={drawerID}
      opened={opened}
      onClose={requestClose}
      closeOnEscape={false}
      position="right"
      size="min(960px, 100vw)"
      zIndex={200}
      trapFocus={!dockOpen}
      closeButtonProps={
        { "data-autofocus": true } as React.ComponentProps<"button">
      }
      removeScrollProps={{
        shards: agent?.dockElement ? [agent.dockElement] : [],
      }}
      transitionProps={{
        transition: "slide-left",
        duration: 250,
        exitDuration: 250,
        onExited: () =>
          navigate(returnTo, { replace: true, preventScrollReset: true }),
      }}
      title={
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex min-w-0 flex-1 flex-col">
            <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
              {kind ?? "Resource"}
            </span>
            <span className="truncate text-sm font-bold text-slate-900">
              {name}
            </span>
          </div>
          <AskAgentButton
            variant="icon"
            label={`Ask the agent about ${name}`}
          />
        </div>
      }
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      styles={{
        inner:
          dockOpen && docked
            ? { width: "calc(100% - var(--agent-dock-width))" }
            : undefined,
        title: { flex: 1, minWidth: 0 },
        header: {
          borderBottomWidth: "1px",
          borderBottomStyle: "solid",
          borderBottomColor: "var(--color-slate-200)",
          minHeight: "56px",
        },
        body: {
          flex: "1 0 auto",
          padding: "16px",
          backgroundColor: "var(--color-slate-50)",
        },
        content: {
          display: "flex",
          flexDirection: "column",
          borderLeftWidth: "1px",
          borderLeftStyle: "solid",
          borderLeftColor: "var(--color-slate-200)",
        },
      }}
    >
      <ResourceItemPage />
    </Drawer>
  );
};

export default ResourceItemDrawer;
