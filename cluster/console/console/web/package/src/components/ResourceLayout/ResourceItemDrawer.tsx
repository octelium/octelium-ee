import { useHasDirtyForm } from "@/utils/forms";
import { getAPIKindFromPath } from "@/utils/pb";
import { Drawer } from "@mantine/core";
import { useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import ResourceItemPage from "./ResourceItemPage";

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

  return (
    <Drawer
      opened={opened}
      onClose={() => setOpened(false)}
      position="right"
      size="min(900px, 100vw)"
      closeOnClickOutside={!hasDirtyForm}
      closeOnEscape={!hasDirtyForm}
      transitionProps={{
        transition: "slide-left",
        duration: 250,
        exitDuration: 250,
        onExited: () =>
          navigate(returnTo, { replace: true, preventScrollReset: true }),
      }}
      title={
        <div className="flex min-w-0 flex-col">
          <span className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
            {kind ?? "Resource"}
          </span>
          <span className="truncate text-sm font-bold text-slate-900">
            {name}
          </span>
        </div>
      }
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      styles={{
        header: {
          borderBottomWidth: "1px",
          borderBottomStyle: "solid",
          borderBottomColor: "#e2e8f0",
          minHeight: "56px",
        },
        body: {
          minHeight: "calc(100dvh - 56px)",
          padding: "16px",
          backgroundColor: "#f8fafc",
        },
        content: {
          borderLeftWidth: "1px",
          borderLeftStyle: "solid",
          borderLeftColor: "#e2e8f0",
        },
      }}
    >
      <ResourceItemPage />
    </Drawer>
  );
};

export default ResourceItemDrawer;
