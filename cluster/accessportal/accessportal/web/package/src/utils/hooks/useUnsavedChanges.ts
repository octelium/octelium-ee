import * as React from "react";
import { useBlocker } from "react-router-dom";

export const useUnsavedChanges = (dirty: boolean) => {
  const bypass = React.useRef(false);
  const [discard, setDiscard] = React.useState<(() => void) | null>(null);
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      dirty &&
      !bypass.current &&
      currentLocation.pathname !== nextLocation.pathname,
  );

  React.useEffect(() => {
    if (!dirty) return;

    const beforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", beforeUnload);
    return () => window.removeEventListener("beforeunload", beforeUnload);
  }, [dirty]);

  const confirmDiscard = (action: () => void) => {
    if (dirty) setDiscard(() => action);
    else action();
  };

  const allowNavigation = (action: () => void) => {
    bypass.current = true;
    try {
      action();
    } finally {
      bypass.current = false;
    }
  };

  return {
    opened: blocker.state === "blocked" || discard !== null,
    onClose: () => {
      if (blocker.state === "blocked") blocker.reset();
      setDiscard(null);
    },
    onConfirm: () => {
      if (blocker.state === "blocked") blocker.proceed();
      else discard?.();
      setDiscard(null);
    },
    confirmDiscard,
    allowNavigation,
  };
};
