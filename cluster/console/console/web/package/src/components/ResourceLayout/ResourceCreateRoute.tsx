import { API, Resource, ResourceName } from "@/utils/pb";
import { Drawer } from "@mantine/core";
import * as React from "react";
import { approveNextNavigation, useHasDirtyForm } from "@/utils/forms";
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
  specComponent: React.ComponentType<{
    item: Resource;
    onUpdate: (item: Resource) => void;
  }>;
  createResource?: () => Resource;
};

const ResourceCreateRoute = (props: Props) => {
  const location = useLocation();
  const navigate = useNavigate();
  const state = (location.state as CreateRouteState | null) ?? undefined;
  const inDrawer = state?.createInDrawer === true;
  const [opened, setOpened] = React.useState(false);
  const hasDirtyForm = useHasDirtyForm();
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
  const close = () => {
    approveNextNavigation();
    setOpened(false);
  };
  const requestClose = () => {
    if (hasDirtyForm && !window.confirm("Discard unsaved changes?")) {
      return;
    }
    close();
  };
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
      onClose={requestClose}
      position="right"
      size="min(960px, 100vw)"
      transitionProps={{
        transition: "slide-left",
        duration: 250,
        exitDuration: 250,
        onExited: handleExited,
      }}
      title={
        <span className="text-sm font-semibold text-slate-900">
          Create {props.kind}
        </span>
      }
      overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
      styles={{
        header: {
          borderBottomWidth: "1px",
          borderBottomStyle: "solid",
          borderBottomColor: "var(--color-slate-200)",
          minHeight: "56px",
        },
        body: {
          minHeight: "calc(100dvh - 56px)",
          padding: "16px",
          backgroundColor: "var(--color-slate-50)",
        },
        content: {
          borderLeftWidth: "1px",
          borderLeftStyle: "solid",
          borderLeftColor: "var(--color-slate-200)",
        },
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
