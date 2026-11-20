import * as React from "react";

export interface AgentApplyTarget {
  id: string;
  label: string;
  languages: string[];
  apply: (code: string) => string | undefined;
}

let current: AgentApplyTarget | undefined;
const listeners = new Set<() => void>();

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

const emit = () => listeners.forEach((listener) => listener());

export const setAgentApplyTarget = (target: AgentApplyTarget) => {
  current = target;
  emit();
};

export const clearAgentApplyTarget = (id: string) => {
  if (current?.id !== id) return;
  current = undefined;
  emit();
};

export const useAgentApplyTarget = () =>
  React.useSyncExternalStore(
    subscribe,
    () => current,
    () => undefined,
  );
