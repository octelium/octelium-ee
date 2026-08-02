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
  const returnTo = state?.returnTo ?? parentPath;

  useEffect(() => {
    let openFrame = 0;
    const mountFrame = requestAnimationFrame(() => {
      openFrame = requestAnimationFrame(() => setOpened(true));
    });
    return () => {
      cancelAnimationFrame(mountFrame);
      cancelAnimationFrame(openFrame);
    };
  }, []);

  return (
    <Drawer
      opened={opened}
      onClose={() => setOpened(false)}
      position="right"
      size="min(900px, 100vw)"
      transitionProps={{
        transition: "slide-left",
        duration: 500,
        exitDuration: 500,
        onExited: () =>
          navigate(returnTo, { replace: true, preventScrollReset: true }),
      }}
      title={
        <div className="flex min-w-0 items-center gap-2">
          <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
            Resource
          </span>
          <span className="truncate font-mono text-sm font-semibold text-slate-800">
            {name}
          </span>
        </div>
      }
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      styles={{
        header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
        body: {
          minHeight: "calc(100dvh - 56px)",
          padding: "16px",
          backgroundColor: "#f8fafc",
        },
        content: { borderLeft: "1px solid #e2e8f0" },
      }}
    >
      <ResourceItemPage />
    </Drawer>
  );
};

export default ResourceItemDrawer;
