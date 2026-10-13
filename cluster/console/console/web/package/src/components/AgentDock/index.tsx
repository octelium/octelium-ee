import * as React from "react";
import { useAgent } from "./context";

const AgentDock = React.lazy(() => import("./Dock"));

export const AgentDockHost = () => {
  const { opened } = useAgent();
  const [mounted, setMounted] = React.useState(opened);

  React.useEffect(() => {
    if (opened) setMounted(true);
  }, [opened]);

  return mounted ? (
    <React.Suspense fallback={null}>
      <AgentDock />
    </React.Suspense>
  ) : null;
};
