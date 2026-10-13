import type {
  AgentInfo,
  ContextItem,
  ResourceRef,
} from "@/apis/consoleagent/protocol";
import type { Crumb } from "@/components/Navigation/catalog";
import * as React from "react";
import type { AgentStatus } from "./status";

export interface PinnedContext {
  label: string;
  resources?: ResourceRef[];
  items?: ContextItem[];
}

export interface AskRequest {
  id: number;
  text?: string;
  send?: boolean;
  pinned?: PinnedContext;
  fresh?: boolean;
}

export type AgentActivity = "working" | "approval";

export interface AgentContextValue {
  status: AgentStatus;
  url?: string;
  info?: AgentInfo;
  opened: boolean;
  hidden: boolean;
  trail: Crumb[];
  request?: AskRequest;
  activity?: AgentActivity;
  conversationId?: string;
  dockElement: HTMLElement | null;
  open: () => void;
  close: () => void;
  toggle: () => void;
  ask: (request: Omit<AskRequest, "id">) => void;
  consume: (id: number) => void;
  setActivity: (activity?: AgentActivity) => void;
  setConversationId: (id?: string) => void;
  setDockElement: (element: HTMLElement | null) => void;
}

export interface AgentAskValue {
  available: boolean;
  ask: (request: Omit<AskRequest, "id">) => void;
}

export const isAgentPath = (pathname: string) =>
  pathname === "/agent" || pathname.startsWith("/agent/");

export const isModKey = (event: KeyboardEvent) =>
  /Mac|iPhone|iPad|iPod/.test(navigator.userAgent)
    ? event.metaKey && !event.ctrlKey
    : event.ctrlKey && !event.metaKey;

export const AgentContext = React.createContext<AgentContextValue | undefined>(
  undefined,
);

export const AgentAskContext = React.createContext<AgentAskValue>({
  available: false,
  ask: () => {},
});

export const useAgent = (): AgentContextValue => {
  const ctx = React.useContext(AgentContext);
  if (!ctx) {
    throw new Error("useAgent must be used within an AgentProvider");
  }
  return ctx;
};

export const useAgentAsk = () => React.useContext(AgentAskContext);

export const useOptionalAgent = () => React.useContext(AgentContext);
