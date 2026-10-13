import { resolveTrail } from "@/components/Navigation/catalog";
import * as React from "react";
import { useLocation } from "react-router-dom";
import { toast } from "sonner";
import {
  AgentAskContext,
  AgentContext,
  isAgentPath,
  isModKey,
  type AgentActivity,
  type AgentContextValue,
  type AskRequest,
} from "./context";
import { useAgentStatus } from "./status";

const storageKey = "octelium-console-agent-dock";

const readSession = (): { opened?: boolean; conversationId?: string } => {
  try {
    return JSON.parse(sessionStorage.getItem(storageKey) ?? "{}");
  } catch {
    return {};
  }
};

const writeSession = (value: { opened: boolean; conversationId?: string }) => {
  try {
    sessionStorage.setItem(storageKey, JSON.stringify(value));
  } catch {
    return;
  }
};

export const AgentProvider = (props: { children: React.ReactNode }) => {
  const { pathname } = useLocation();
  const { status, url, info } = useAgentStatus();
  const [session] = React.useState(readSession);
  const [opened, setOpened] = React.useState(false);
  const [request, setRequest] = React.useState<AskRequest>();
  const [activity, setActivity] = React.useState<AgentActivity>();
  const [conversationId, setConversationId] = React.useState(
    session.conversationId,
  );
  const [dockElement, setDockElement] = React.useState<HTMLElement | null>(
    null,
  );
  const counterRef = React.useRef(0);
  const restoredRef = React.useRef(false);
  const openedRef = React.useRef(opened);
  const hidden = isAgentPath(pathname);
  const trail = React.useMemo(() => resolveTrail(pathname), [pathname]);
  const usable = status !== "unavailable" && status !== "loading";

  React.useEffect(() => {
    openedRef.current = opened;
  }, [opened]);

  React.useEffect(() => {
    if (restoredRef.current || status === "loading") return;
    restoredRef.current = true;
    if (status === "ready" && session.opened) setOpened(true);
  }, [status, session.opened]);

  React.useEffect(() => {
    if (!restoredRef.current) return;
    writeSession({ opened, conversationId });
  }, [opened, conversationId]);

  React.useEffect(() => {
    const root = document.documentElement;
    if (opened && !hidden) {
      root.setAttribute("data-agent-dock", "open");
    } else {
      root.removeAttribute("data-agent-dock");
    }
    return () => root.removeAttribute("data-agent-dock");
  }, [opened, hidden]);

  const open = React.useCallback(() => setOpened(true), []);
  const close = React.useCallback(() => setOpened(false), []);
  const toggle = React.useCallback(() => setOpened((value) => !value), []);

  const ask = React.useCallback((next: Omit<AskRequest, "id">) => {
    counterRef.current += 1;
    setRequest({ ...next, id: counterRef.current });
    setOpened(true);
  }, []);

  const consume = React.useCallback(
    (id: number) =>
      setRequest((current) => (current?.id === id ? undefined : current)),
    [],
  );

  React.useEffect(() => {
    if (!usable || hidden) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        isModKey(event) &&
        !event.shiftKey &&
        !event.altKey &&
        event.key.toLowerCase() === "j"
      ) {
        event.preventDefault();
        toggle();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [usable, hidden, toggle]);

  const previousActivityRef = React.useRef<AgentActivity | undefined>(
    undefined,
  );
  React.useEffect(() => {
    const previous = previousActivityRef.current;
    previousActivityRef.current = activity;
    if (openedRef.current || hidden || previous === activity) return;
    const action = { label: "Open", onClick: () => setOpened(true) };
    if (activity === "approval") {
      toast("The agent is waiting for your approval", { action });
    } else if (!activity && previous) {
      toast("The agent has answered", { action });
    }
  }, [activity, hidden]);

  const value = React.useMemo<AgentContextValue>(
    () => ({
      status,
      url,
      info,
      opened,
      hidden,
      trail,
      request,
      activity,
      conversationId,
      dockElement,
      open,
      close,
      toggle,
      ask,
      consume,
      setActivity,
      setConversationId,
      setDockElement,
    }),
    [
      status,
      url,
      info,
      opened,
      hidden,
      trail,
      request,
      activity,
      conversationId,
      dockElement,
      open,
      close,
      toggle,
      ask,
      consume,
    ],
  );

  const available = status === "ready";
  const askValue = React.useMemo(() => ({ available, ask }), [available, ask]);

  return (
    <AgentContext.Provider value={value}>
      <AgentAskContext.Provider value={askValue}>
        {props.children}
      </AgentAskContext.Provider>
    </AgentContext.Provider>
  );
};
