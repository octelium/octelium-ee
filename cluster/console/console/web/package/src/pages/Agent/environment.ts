export const readStorage = (key: string): string | undefined => {
  try {
    return localStorage.getItem(key) ?? undefined;
  } catch {
    return undefined;
  }
};

export const writeStorage = (key: string, value: string) => {
  try {
    localStorage.setItem(key, value);
  } catch {
    return;
  }
};

export const devAgentURL = import.meta.env.VITE_CONSOLE_AGENT_URL as
  string | undefined;

export const agentKey = ["agent", "environment"];

export const workspaceStorageKey = "octelium-console-agent-workspace";

export const workspaceKey = (uid: string) => ["agent", "workspace", uid];

export const infoKey = (url?: string) => ["agent", url, "info"];

export const conversationsKey = (baseUrl: string) => [
  "agent",
  baseUrl,
  "conversations",
];
