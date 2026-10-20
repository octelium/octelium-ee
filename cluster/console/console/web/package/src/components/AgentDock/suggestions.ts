import type { ResourceRef } from "@/apis/consoleagent/protocol";
import type { Crumb } from "@/components/Navigation/catalog";
import type { PageCondition } from "@/components/PageHeader/context";
import {
  Activity,
  BookOpenText,
  ChartColumn,
  CopyPlus,
  Eraser,
  FileSearch,
  Gauge,
  KeyRound,
  ListChecks,
  ScanSearch,
  ShieldAlert,
  ShieldCheck,
  ShieldPlus,
  Sparkles,
  Trash2,
  UserX,
  Users,
  Waypoints,
  type LucideIcon,
} from "lucide-react";
import type { PinnedContext } from "./context";

export interface Suggestion {
  title: string;
  prompt: string;
  icon: LucideIcon;
}

const s = (icon: LucideIcon, title: string, prompt: string): Suggestion => ({
  icon,
  title,
  prompt,
});

const itemSuggestions = (pinned: PinnedContext): Suggestion[] => {
  const items = pinned.items ?? [];
  if (items.length > 1) {
    return [
      s(ListChecks, "Summarize", "Summarize these entries"),
      s(
        ScanSearch,
        "Find the common cause",
        "What do these entries have in common?",
      ),
    ];
  }
  switch (items[0]?.kind) {
    case "AccessLog":
      return [
        s(
          ShieldCheck,
          "Explain the decision",
          "Explain this request: why was it allowed or denied, and which Policy decided it?",
        ),
        s(
          FileSearch,
          "Find similar requests",
          "Find similar requests from the same User to this Service in the last 24 hours and summarize them",
        ),
        s(
          ShieldAlert,
          "Is it suspicious?",
          "Is this activity suspicious? Check the User's recent access, devices and locations",
        ),
        s(
          ShieldPlus,
          "Change the access",
          "What Policy change would be needed to change this decision?",
        ),
      ];
    case "AuthenticationLog":
      return [
        s(
          BookOpenText,
          "Explain this sign-in",
          "Explain this authentication: the identity provider, the factors and the resulting Session",
        ),
        s(
          Activity,
          "Recent sign-ins",
          "Summarize the recent sign-ins of this User",
        ),
        s(
          ShieldAlert,
          "Is it suspicious?",
          "Is this sign-in suspicious? Compare it with the User's usual devices and locations",
        ),
      ];
    case "AuditLog":
      return [
        s(
          BookOpenText,
          "Explain this change",
          "Explain what this change did and who made it",
        ),
        s(
          Activity,
          "Related changes",
          "Show the other changes made to the same resource in the last 7 days",
        ),
        s(Eraser, "Undo it", "How can this change be undone?"),
      ];
    case "DashboardSection":
      return [
        s(
          BookOpenText,
          "Explain this section",
          "Explain what this section shows and what stands out in it",
        ),
        s(
          ShieldAlert,
          "What needs attention?",
          "What in this section needs my attention, and what should I address first?",
        ),
        s(
          ChartColumn,
          "Compare with before",
          "How does this compare with the previous period, and what drove the change?",
        ),
        s(
          FileSearch,
          "Show the evidence",
          "Find the log entries and resources behind the most significant number in this section",
        ),
      ];
    case "ComponentLog":
      return [
        s(
          ScanSearch,
          "Diagnose",
          "Explain this log entry, its likely cause and how to fix it",
        ),
        s(
          FileSearch,
          "Related entries",
          "Find related entries from the same component in the last hour",
        ),
      ];
    default:
      return [s(BookOpenText, "Explain", "Explain this entry")];
  }
};

const kindSuggestions = (kind: string): Suggestion[] => {
  switch (kind) {
    case "Service":
      return [
        s(
          BookOpenText,
          "Summarize",
          "Summarize this Service: its mode, upstream, Policies and who can access it",
        ),
        s(
          Users,
          "Who accessed it?",
          "Who accessed this Service in the last 24 hours?",
        ),
        s(
          CopyPlus,
          "Clone it",
          "Clone this Service with a new name and the same configuration",
        ),
        s(
          Trash2,
          "Delete if unused",
          "Delete this Service if it has not been accessed in the last 30 days",
        ),
      ];
    case "User":
      return [
        s(
          BookOpenText,
          "Summarize",
          "Summarize this User's access and recent activity",
        ),
        s(
          Waypoints,
          "What can it reach?",
          "Which Services can this User access?",
        ),
        s(
          ShieldPlus,
          "Read-only Policy",
          "Create a read-only Policy for this User",
        ),
        s(UserX, "Revoke Sessions", "Revoke this User's active Sessions"),
      ];
    case "Group":
      return [
        s(BookOpenText, "Summarize", "Summarize this Group and its members"),
        s(
          ShieldCheck,
          "Applied Policies",
          "Which Policies apply to this Group?",
        ),
        s(
          Waypoints,
          "What can it reach?",
          "Which Services can the members of this Group access?",
        ),
      ];
    case "Session":
      return [
        s(
          BookOpenText,
          "Explain",
          "Explain this Session: its User, device, authentication and posture",
        ),
        s(
          Activity,
          "Recent activity",
          "What has this Session accessed recently?",
        ),
        s(
          UserX,
          "Terminate if idle",
          "Terminate this Session if it has been idle for more than a day",
        ),
      ];
    case "Device":
      return [
        s(BookOpenText, "Summarize", "Summarize this Device and its posture"),
        s(Activity, "Sessions", "Which Sessions use this Device?"),
      ];
    case "Policy":
      return [
        s(BookOpenText, "Explain", "Explain this Policy in plain words"),
        s(
          Users,
          "Who is affected?",
          "Which Users and Services does this Policy affect?",
        ),
        s(
          ChartColumn,
          "Recent decisions",
          "Chart the requests allowed and denied by this Policy over the last 7 days",
        ),
      ];
    case "Namespace":
      return [
        s(
          BookOpenText,
          "Summarize",
          "Summarize the Services in this Namespace",
        ),
        s(
          ChartColumn,
          "Traffic",
          "Chart the traffic to this Namespace over the last 24 hours",
        ),
      ];
    case "Credential":
      return [
        s(
          KeyRound,
          "Explain",
          "Explain this Credential, its scope and when it expires",
        ),
        s(
          Activity,
          "Last used",
          "When was this Credential last used and by whom?",
        ),
      ];
    case "IdentityProvider":
      return [
        s(
          Activity,
          "Sign-ins",
          "Summarize the sign-ins through this Identity Provider in the last 7 days",
        ),
        s(
          Gauge,
          "Health",
          "Is this Identity Provider healthy and correctly configured?",
        ),
      ];
    case "Secret":
      return [
        s(Waypoints, "Where is it used?", "Which resources use this Secret?"),
      ];
    case "Certificate":
      return [
        s(
          KeyRound,
          "Expiry",
          "When does this Certificate expire and how is it renewed?",
        ),
      ];
    case "Request":
      return [
        s(
          ShieldAlert,
          "Assess the risk",
          "Summarize this access Request and its risk",
        ),
      ];
    case "Workspace":
      return [
        s(
          Gauge,
          "Summarize",
          "Summarize this Workspace, its state and its resource usage",
        ),
      ];
    default:
      return [
        s(BookOpenText, "Explain", `Explain this ${kind}`),
        s(
          Activity,
          "Recent changes",
          `Show the recent changes to this ${kind}`,
        ),
      ];
  }
};

const tabSuggestions = (tab: string, kind: string): Suggestion[] => {
  switch (tab) {
    case "edit":
      return [
        s(
          BookOpenText,
          "Explain the settings",
          `Explain the configuration of this ${kind}`,
        ),
        s(
          ShieldCheck,
          "Review the configuration",
          `Review the configuration of this ${kind} and suggest safer settings`,
        ),
      ];
    case "accesslogs":
      return [
        s(
          Activity,
          "Summarize the access",
          `Summarize the access to this ${kind} in the last 24 hours`,
        ),
        s(
          ShieldAlert,
          "Explain the denials",
          `Explain the denied requests involving this ${kind}`,
        ),
      ];
    case "authenticationlogs":
      return [
        s(
          Activity,
          "Summarize the sign-ins",
          `Summarize the sign-ins of this ${kind} in the last 7 days`,
        ),
      ];
    case "auditlogs":
      return [
        s(
          Activity,
          "Recent changes",
          `Who changed this ${kind} recently and what changed?`,
        ),
      ];
    case "metrics":
      return [
        s(
          ChartColumn,
          "Explain the traffic",
          `Explain the traffic of this ${kind} and any anomalies`,
        ),
      ];
    case "llm":
      return [
        s(
          ChartColumn,
          "LLM usage",
          `Summarize the LLM usage of this ${kind} and what drives it`,
        ),
      ];
    default:
      return [];
  }
};

const pageSuggestions = (range: string): Record<string, Suggestion[]> => ({
  "/": [
    s(
      ShieldAlert,
      "What needs attention?",
      "What needs my attention in the Cluster right now?",
    ),
    s(
      Activity,
      "Recent activity",
      `Summarize the access activity of the ${range}`,
    ),
    s(
      UserX,
      "Denied Users",
      `Which Users had denied requests in the ${range}?`,
    ),
    s(
      ChartColumn,
      "Traffic trends",
      "Chart the requests per Service over the last 7 days",
    ),
  ],
  "/core": [
    s(
      ShieldAlert,
      "Unprotected Services",
      "Which Services are public or anonymous, and which have no Policies?",
    ),
    s(UserX, "Inactive Users", "Which Users have not signed in for 30 days?"),
    s(
      Activity,
      "Connected now",
      "How many Sessions are connected right now and from where?",
    ),
  ],
  "/enterprise": [
    s(
      KeyRound,
      "Expiring Certificates",
      "Which Certificates expire in the next 30 days?",
    ),
    s(
      Gauge,
      "Integrations",
      "Are the directory providers, secret stores and collector exporters healthy?",
    ),
  ],
  "/access": [
    s(
      ListChecks,
      "Pending Requests",
      "Which access Requests are pending and how old are they?",
    ),
    s(
      Activity,
      "Active grants",
      "Which access grants are active and when do they expire?",
    ),
  ],
  "/cordium": [
    s(Gauge, "Idle Workspaces", "Which Workspaces are running but idle?"),
    s(
      ShieldAlert,
      "Failures",
      "Which Workspaces or Templates failed recently and why?",
    ),
  ],
  "/clusterman": [
    s(
      Gauge,
      "Up to date?",
      "Is the Cluster up to date and is its license valid?",
    ),
  ],
  "/visibility": [
    s(
      ShieldAlert,
      "Investigate denials",
      `Investigate the denied requests of the ${range}`,
    ),
    s(
      ScanSearch,
      "Anomalies",
      `Are there any unusual access or sign-in patterns in the ${range}?`,
    ),
  ],
  "/visibility/accesslogs": [
    s(
      ShieldAlert,
      "Summarize the denials",
      "Summarize the denied requests in this view",
    ),
    s(
      Users,
      "Top denied",
      "Which Users and Services have the most denied requests?",
    ),
    s(
      ChartColumn,
      "Chart",
      "Chart the access logs per Service over the last 7 days",
    ),
  ],
  "/visibility/authenticationlogs": [
    s(Activity, "Summarize", "Summarize the sign-ins of the last 24 hours"),
    s(
      ShieldAlert,
      "Unusual sign-ins",
      "Are there unusual sign-ins, e.g. from new devices or locations?",
    ),
    s(
      ShieldAlert,
      "Failed sign-ins",
      "Summarize the failed sign-ins of the last 24 hours by reason, User and identity provider",
    ),
  ],
  "/visibility/auditlogs": [
    s(
      Activity,
      "Recent changes",
      "Who changed what in the Cluster in the last 24 hours?",
    ),
  ],
  "/visibility/componentlogs": [
    s(
      ScanSearch,
      "Errors",
      "Summarize the component errors of the last hour and their likely causes",
    ),
  ],
  "/visibility/ssh": [
    s(
      Activity,
      "SSH activity",
      "Summarize the SSH sessions of the last 24 hours",
    ),
  ],
  "/visibility/metrics": [
    s(
      ChartColumn,
      "Anomalies",
      "Explain any anomalies in the Cluster traffic and latency",
    ),
  ],
  "/visibility/llm": [
    s(
      ChartColumn,
      "Token usage",
      "What drives the LLM token usage and how did it change?",
    ),
    s(
      ShieldAlert,
      "Guardrails",
      "Which guardrails block requests most, and for whom?",
    ),
  ],
});

const conditionSuggestions = (conditions: PageCondition[]): Suggestion[] =>
  conditions
    .slice(0, 2)
    .map((item) =>
      s(
        ShieldAlert,
        item.label,
        `${item.count} × ${item.label}: what is causing this, and what should I address first?`,
      ),
    );

const listSuggestions = (label: string): Suggestion[] => [
  s(ListChecks, "Summarize", `Summarize the ${label}`),
  s(Activity, "Recent changes", `Which ${label} changed in the last 7 days?`),
];

const listKindSuggestions: Record<string, Suggestion[]> = {
  "/core/services": [
    s(
      Trash2,
      "Unused Services",
      "Which Services were not accessed in the last 30 days?",
    ),
    s(
      ShieldAlert,
      "Public Services",
      "Which Services are public or anonymous?",
    ),
  ],
  "/core/users": [
    s(UserX, "Inactive Users", "Which Users have not signed in for 30 days?"),
    s(ShieldAlert, "Privileged Users", "Which Users have the broadest access?"),
  ],
  "/core/sessions": [
    s(
      Activity,
      "Connected now",
      "Which Sessions are connected right now and from where?",
    ),
    s(
      UserX,
      "Idle Sessions",
      "Which Sessions have been idle for more than a day?",
    ),
  ],
  "/core/policies": [
    s(
      ScanSearch,
      "Unused Policies",
      "Which Policies are not used by any Service, Namespace or Group?",
    ),
    s(
      ShieldPlus,
      "Write a Policy",
      "Help me write a Policy that restricts a Service to a Group",
    ),
  ],
};

export const getSuggestions = (args: {
  trail: Crumb[];
  pathname: string;
  pinned?: PinnedContext;
  details?: Record<string, string>;
  conditions?: PageCondition[];
}): Suggestion[] => {
  const { trail, pinned } = args;
  if (pinned?.items && pinned.items.length > 0) {
    return itemSuggestions(pinned).slice(0, 4);
  }

  const resources: ResourceRef[] = pinned?.resources ?? [];
  if (resources.length > 1) {
    return [
      s(ListChecks, "Summarize", "Summarize these resources"),
      s(
        ScanSearch,
        "Compare",
        "Compare these resources and highlight the differences",
      ),
    ];
  }

  const resourceCrumbIdx = trail.findIndex((crumb) => crumb.resource);
  const kind =
    resources[0]?.kind ??
    (resourceCrumbIdx >= 0
      ? trail[resourceCrumbIdx].resource!.kind
      : undefined);
  if (kind) {
    const tab =
      !resources[0] && resourceCrumbIdx >= 0
        ? trail[resourceCrumbIdx + 1]?.path.split("/").at(-1)
        : undefined;
    return [
      ...(tab ? tabSuggestions(tab, kind) : []),
      ...kindSuggestions(kind),
    ].slice(0, 4);
  }

  const last = trail.at(-1);
  const path = last?.path ?? args.pathname;
  const pages = pageSuggestions(
    args.details?.["Time range"] ?? "last 24 hours",
  );
  const conditions = conditionSuggestions(args.conditions ?? []);
  if (pages[path]) {
    return [...conditions, ...pages[path]].slice(
      0,
      Math.max(4, pages[path].length),
    );
  }
  if (trail.length === 3 && last) {
    return [
      ...(listKindSuggestions[path] ?? []),
      ...listSuggestions(last.label),
    ].slice(0, 4);
  }
  return [
    s(
      Sparkles,
      "Explain this page",
      "Explain what I am looking at on this page",
    ),
    s(
      ScanSearch,
      "What stands out?",
      "What stands out on this page and needs attention?",
    ),
  ];
};
