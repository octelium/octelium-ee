import {
  AccessLog,
  AccessLog_Entry_Common_Status,
  ComponentLog,
  ComponentLog_Entry_Level,
} from "@/apis/corev1/corev1";
import {
  AuditLog,
  AuthenticationLog,
  AuthenticationLog_Entry_Status,
} from "@/apis/enterprisev1/enterprisev1";
import { AskAgentButton } from "@/components/AgentDock/AskButtons";
import { logToJSON, type LogItem } from "./Editor";

const refName = (ref?: { name: string; uid: string }) =>
  ref?.name || ref?.uid || undefined;

export const logTitle = (item: LogItem): string => {
  switch (item.kind) {
    case "AccessLog": {
      const common = (item as AccessLog).entry?.common;
      const source =
        refName(common?.userRef) ?? refName(common?.sessionRef) ?? "Anonymous";
      const target = refName(common?.serviceRef) ?? "unknown Service";
      const status =
        common?.status === AccessLog_Entry_Common_Status.ALLOWED
          ? "allowed"
          : common?.status === AccessLog_Entry_Common_Status.DENIED
            ? "denied"
            : undefined;
      return `${source} → ${target}${status ? ` · ${status}` : ""}`;
    }
    case "AuthenticationLog": {
      const entry = (item as AuthenticationLog).entry;
      const failed =
        entry?.status === AuthenticationLog_Entry_Status.FAILURE
          ? "Failed sign-in"
          : "Sign-in";
      return `${failed} of ${refName(entry?.userRef) ?? refName(entry?.sessionRef) ?? "a User"}`;
    }
    case "AuditLog": {
      const entry = (item as AuditLog).entry;
      const resource = entry?.resourceRef;
      return [
        refName(entry?.userRef),
        entry?.operation,
        resource
          ? `${resource.kind} ${refName(resource) ?? ""}`.trim()
          : undefined,
      ]
        .filter(Boolean)
        .join(" · ");
    }
    case "ComponentLog": {
      const entry = (item as ComponentLog).entry;
      const level = entry?.level
        ? ComponentLog_Entry_Level[entry.level]?.toLowerCase()
        : undefined;
      return [entry?.component?.type, level, entry?.message?.slice(0, 80)]
        .filter(Boolean)
        .join(" · ");
    }
    default:
      return item.kind;
  }
};

const AskLog = (props: { item: LogItem }) => (
  <AskAgentButton
    label="Ask agent"
    pinned={() => {
      const title = logTitle(props.item) || props.item.kind;
      return {
        label: title,
        items: [
          {
            kind: props.item.kind,
            id: props.item.metadata?.id,
            title,
            data: logToJSON(props.item),
          },
        ],
      };
    }}
  />
);

export default AskLog;
