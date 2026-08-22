import { API, Resource, ResourceName } from "@/utils/pb";
import { Drawer } from "@mantine/core";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import ResourceCreatePage from "./ResourceCreate";

type CreateRouteState = {
  createInDrawer?: boolean;
  returnTo?: string;
  returnState?: unknown;
};

type Props = {
  api: API;
  kind: ResourceName;
  specComponent: (props: {
    item: Resource;
    onUpdate: (item: Resource) => void;
  }) => React.ReactNode;
  createResource?: () => Resource;
};

const ResourceCreateRoute = (props: Props) => {
  const location = useLocation();
  const navigate = useNavigate();
  const state = (location.state as CreateRouteState | null) ?? undefined;
  const inDrawer = state?.createInDrawer === true;
  const [opened, setOpened] = React.useState(false);
  const createdName = React.useRef<string | undefined>(undefined);

  React.useEffect(() => {
    if (!inDrawer) return;
    let firstFrame = 0;
    let secondFrame = 0;
    firstFrame = requestAnimationFrame(() => {
      secondFrame = requestAnimationFrame(() => setOpened(true));
    });
    return () => {
      cancelAnimationFrame(firstFrame);
      cancelAnimationFrame(secondFrame);
    };
  }, [inDrawer, location.key]);

  if (!inDrawer) {
    return (
      <ResourceCreatePage
        specComponent={props.specComponent}
        createResource={props.createResource}
      />
    );
  }

  const returnTo = state?.returnTo ?? `/${props.api}`;
  const close = () => setOpened(false);
  const handleExited = () => {
    const returnState =
      state?.returnState && typeof state.returnState === "object"
        ? { ...(state.returnState as Record<string, unknown>) }
        : {};
    if (createdName.current) {
      returnState.createdResourceName = createdName.current;
    }
    navigate(returnTo, {
      replace: true,
      preventScrollReset: true,
      state: Object.keys(returnState).length > 0 ? returnState : undefined,
    });
  };

  return (
    <Drawer
      opened={opened}
      onClose={close}
      position="right"
      size="min(960px, 100vw)"
      transitionProps={{
        transition: "slide-left",
        duration: 500,
        exitDuration: 500,
        onExited: handleExited,
      }}
      title={
        <span className="text-sm font-bold text-slate-800">
          Create {props.kind}
        </span>
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
      <ResourceCreatePage
        specComponent={props.specComponent}
        createResource={props.createResource}
        onCreated={(item) => {
          createdName.current = item.metadata?.name;
          close();
        }}
        onCancel={close}
      />
    </Drawer>
  );
};

export default ResourceCreateRoute;
