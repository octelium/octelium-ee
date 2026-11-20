import type { CategoryId, Values } from "./blocks";
import {
  blockNode,
  everyNode,
  groupNode,
  type ConditionNode,
  type Negation,
} from "./model";

export type RecipeIcon =
  | "users"
  | "user"
  | "building"
  | "server"
  | "clock"
  | "moon"
  | "hourglass"
  | "eye"
  | "pencil"
  | "lock"
  | "terminal"
  | "shield"
  | "fingerprint"
  | "laptop"
  | "globe"
  | "network"
  | "boxes"
  | "bot"
  | "wrench"
  | "check"
  | "ban";

export type Recipe = {
  id: string;
  title: string;
  description: string;
  icon: RecipeIcon;
  category: CategoryId | "general";
  keywords: string[];
  effect?: "ALLOW" | "DENY";
  name: string;
  build: () => ConditionNode;
};

export type RuleDraft = {
  name: string;
  effect: "ALLOW" | "DENY";
  priority: number;
  node?: ConditionNode;
};

export type EnforcementDraft = {
  effect: "ENFORCE" | "IGNORE";
  node?: ConditionNode;
};

export type Starter = {
  id: string;
  title: string;
  description: string;
  icon: RecipeIcon;
  tone: "allow" | "deny" | "neutral";
  build: () => { rules: RuleDraft[]; enforcementRules: EnforcementDraft[] };
};

export const localTimezone = () => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {
    return "";
  }
};

const block = (id: string, values?: Values, negate?: Negation) =>
  blockNode(id, values, negate);

const businessHours = (negate?: Negation) =>
  block(
    "time.window",
    {
      days: ["1", "2", "3", "4", "5"],
      from: "09:00",
      to: "17:00",
      timezone: localTimezone(),
    },
    negate,
  );

const inDays = (days: number) => {
  const date = new Date(Date.now() + days * 24 * 3600 * 1000);
  date.setMinutes(0, 0, 0);
  return date.toISOString();
};

const healthyDevice = () =>
  groupNode("all", [
    block("device.assurance", { values: ["SIGNED", "HARDWARE"] }),
    block("device.posture", {
      checks: ["threatFree", "compliant"],
      grace: false,
    }),
  ]);

const PRIVATE_RANGES = ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"];

export const RECIPES: Recipe[] = [
  {
    id: "group-members",
    title: "Members of a Group",
    description: "Role-based access for everyone in one or more Groups",
    icon: "users",
    category: "identity",
    keywords: ["rbac", "role", "team"],
    effect: "ALLOW",
    name: "group-members",
    build: () => block("identity.group"),
  },
  {
    id: "specific-users",
    title: "Specific Users",
    description: "Name the Users this rule is about",
    icon: "user",
    category: "identity",
    keywords: ["user", "person"],
    effect: "ALLOW",
    name: "named-users",
    build: () => block("identity.user"),
  },
  {
    id: "group-namespace",
    title: "Group to a Namespace",
    description: "Members of a Group reach the Services of a Namespace",
    icon: "building",
    category: "resource",
    keywords: ["rbac", "team", "environment", "namespace"],
    effect: "ALLOW",
    name: "team-namespace",
    build: () =>
      groupNode("all", [block("identity.group"), block("resource.namespace")]),
  },
  {
    id: "group-service",
    title: "Group to Services",
    description: "Members of a Group reach specific Services",
    icon: "server",
    category: "resource",
    keywords: ["rbac", "team", "service"],
    effect: "ALLOW",
    name: "team-services",
    build: () =>
      groupNode("all", [block("identity.group"), block("resource.service")]),
  },
  {
    id: "business-hours",
    title: "Business hours",
    description: "Monday to Friday, 09:00 to 17:00 in your timezone",
    icon: "clock",
    category: "time",
    keywords: ["schedule", "working hours", "time window"],
    effect: "ALLOW",
    name: "business-hours",
    build: () => businessHours(),
  },
  {
    id: "outside-hours",
    title: "Outside business hours",
    description: "Nights and weekends, typically with a Deny effect",
    icon: "moon",
    category: "time",
    keywords: ["night", "weekend", "after hours"],
    effect: "DENY",
    name: "outside-business-hours",
    build: () => businessHours("condition"),
  },
  {
    id: "temporary",
    title: "Temporary access",
    description: "Matches until a date, one week from now by default",
    icon: "hourglass",
    category: "time",
    keywords: ["expire", "until", "contractor", "break glass", "jit"],
    effect: "ALLOW",
    name: "temporary-access",
    build: () => block("time.range", { after: "", before: inDays(7) }),
  },
  {
    id: "http-read-only",
    title: "Read-only HTTP",
    description: "GET, HEAD and OPTIONS requests",
    icon: "eye",
    category: "http",
    keywords: ["readonly", "read only", "safe methods"],
    effect: "ALLOW",
    name: "read-only",
    build: () => block("http.method", { values: ["GET", "HEAD", "OPTIONS"] }),
  },
  {
    id: "http-writes",
    title: "HTTP writes",
    description: "POST, PUT, PATCH and DELETE requests",
    icon: "pencil",
    category: "http",
    keywords: ["write", "mutation", "modify"],
    effect: "DENY",
    name: "writes",
    build: () =>
      block("http.method", { values: ["POST", "PUT", "PATCH", "DELETE"] }),
  },
  {
    id: "admin-paths",
    title: "Admin paths",
    description: "HTTP paths starting with /admin",
    icon: "lock",
    category: "http",
    keywords: ["admin", "path", "route"],
    effect: "DENY",
    name: "admin-paths",
    build: () =>
      block("http.path", { match: { op: "prefix", values: ["/admin"] } }),
  },
  {
    id: "api-full",
    title: "Octelium API",
    description: "Every request to the Cluster's APIs (console, octeliumctl)",
    icon: "boxes",
    category: "api",
    keywords: ["admin", "api", "octeliumctl", "full access"],
    effect: "ALLOW",
    name: "octelium-api",
    build: () => block("api.octelium"),
  },
  {
    id: "api-read-only",
    title: "Octelium API, read-only",
    description: "Only the Get and List methods of the Cluster's APIs",
    icon: "eye",
    category: "api",
    keywords: ["auditor", "viewer", "api", "readonly"],
    effect: "ALLOW",
    name: "octelium-api-read-only",
    build: () =>
      block("api.octelium", {
        apis: [],
        services: [],
        methods: [],
        readOnly: true,
      }),
  },
  {
    id: "api-core-read-only",
    title: "Core API, read-only",
    description: "Read Users, Services, Policies… without changing them",
    icon: "eye",
    category: "api",
    keywords: ["auditor", "viewer", "core", "readonly"],
    effect: "ALLOW",
    name: "core-api-read-only",
    build: () =>
      block("api.octelium", {
        apis: ["octelium.api.main.core.v1"],
        services: [],
        methods: [],
        readOnly: true,
      }),
  },
  {
    id: "strong-auth",
    title: "Multi-factor authentication",
    description: "Sessions authenticated at AAL2 or AAL3",
    icon: "fingerprint",
    category: "session",
    keywords: ["mfa", "2fa", "aal", "strong", "step-up"],
    effect: "ALLOW",
    name: "mfa",
    build: () => block("session.aal", { values: ["AAL2", "AAL3"] }),
  },
  {
    id: "no-mfa",
    title: "Without multi-factor",
    description: "Sessions below AAL2, typically with a Deny effect",
    icon: "shield",
    category: "session",
    keywords: ["mfa", "2fa", "weak", "require"],
    effect: "DENY",
    name: "require-mfa",
    build: () =>
      block("session.aal", { values: ["AAL2", "AAL3"] }, "condition"),
  },
  {
    id: "humans",
    title: "Human Users",
    description: "People signing in, not workloads",
    icon: "user",
    category: "identity",
    keywords: ["human", "people"],
    effect: "ALLOW",
    name: "humans",
    build: () => block("identity.userType", { values: ["HUMAN"] }),
  },
  {
    id: "workloads",
    title: "Workloads",
    description: "Services, CI jobs and containers using Credentials",
    icon: "bot",
    category: "identity",
    keywords: ["workload", "machine", "ci", "bot"],
    effect: "ALLOW",
    name: "workloads",
    build: () => block("identity.userType", { values: ["WORKLOAD"] }),
  },
  {
    id: "healthy-device",
    title: "Healthy managed device",
    description: "Strongly bound Devices passing threat and compliance checks",
    icon: "laptop",
    category: "device",
    keywords: ["posture", "managed", "zero trust", "compliance"],
    effect: "ALLOW",
    name: "healthy-device",
    build: healthyDevice,
  },
  {
    id: "countries",
    title: "From allowed countries",
    description: "Sessions authenticated from specific countries",
    icon: "globe",
    category: "network",
    keywords: ["geo", "country", "geofence"],
    effect: "ALLOW",
    name: "allowed-countries",
    build: () => block("network.country"),
  },
  {
    id: "outside-countries",
    title: "Outside allowed countries",
    description: "Every other country, typically with a Deny effect",
    icon: "globe",
    category: "network",
    keywords: ["geo", "country", "geofence", "block"],
    effect: "DENY",
    name: "geofence",
    build: () => block("network.country", { codes: [] }, "condition"),
  },
  {
    id: "private-network",
    title: "Private networks",
    description: "Requests from 10.0.0.0/8, 172.16.0.0/12 or 192.168.0.0/16",
    icon: "network",
    category: "network",
    keywords: ["office", "vpn", "corporate", "internal"],
    effect: "ALLOW",
    name: "private-networks",
    build: () => block("network.ipRange", { ranges: [...PRIVATE_RANGES] }),
  },
  {
    id: "k8s-read-only",
    title: "Kubernetes read-only",
    description: "get, list and watch requests",
    icon: "eye",
    category: "protocol",
    keywords: ["kubernetes", "k8s", "kubectl", "viewer"],
    effect: "ALLOW",
    name: "kubernetes-read-only",
    build: () =>
      block("protocol.k8sVerb", {
        match: { op: "in", values: ["get", "list", "watch"] },
      }),
  },
  {
    id: "ssh-users",
    title: "SSH as specific users",
    description: "Log in only as named Linux users",
    icon: "terminal",
    category: "protocol",
    keywords: ["ssh", "linux", "root"],
    effect: "ALLOW",
    name: "ssh-users",
    build: () => block("protocol.sshUser", { match: { op: "in", values: [] } }),
  },
  {
    id: "mcp-tools",
    title: "Specific MCP tools",
    description: "Calls to named tools, e.g. to deny destructive ones",
    icon: "wrench",
    category: "mcp",
    keywords: ["mcp", "agent", "tool", "ai"],
    effect: "DENY",
    name: "mcp-tools",
    build: () => block("mcp.tool", { match: { op: "in", values: [] } }),
  },
  {
    id: "llm-models",
    title: "Allowed LLM models",
    description: "Requests for specific models",
    icon: "bot",
    category: "llm",
    keywords: ["llm", "model", "ai", "cost"],
    effect: "ALLOW",
    name: "allowed-models",
    build: () => block("llm.model", { match: { op: "in", values: [] } }),
  },
  {
    id: "every",
    title: "Every request",
    description: "Always matches, e.g. for a catch-all Deny",
    icon: "check",
    category: "general",
    keywords: ["all", "everything", "any", "catch-all", "default"],
    name: "everything",
    build: everyNode,
  },
];

export const STARTERS: Starter[] = [
  {
    id: "group-namespace",
    title: "Team access to a Namespace",
    description: "Members of a Group reach every Service of a Namespace",
    icon: "building",
    tone: "allow",
    build: () => ({
      rules: [
        {
          name: "team-namespace",
          effect: "ALLOW",
          priority: 0,
          node: groupNode("all", [
            block("identity.group"),
            block("resource.namespace"),
          ]),
        },
      ],
      enforcementRules: [],
    }),
  },
  {
    id: "http-read-only",
    title: "Read-only HTTP",
    description: "Allow GET, HEAD and OPTIONS, deny every other method",
    icon: "eye",
    tone: "allow",
    build: () => ({
      enforcementRules: [
        {
          effect: "IGNORE",
          node: block(
            "resource.serviceMode",
            { values: ["HTTP", "WEB"] },
            "condition",
          ),
        },
      ],
      rules: [
        {
          name: "allowed-methods",
          effect: "ALLOW",
          priority: -1,
          node: block("http.method", { values: ["GET", "HEAD", "OPTIONS"] }),
        },
        {
          name: "deny-otherwise",
          effect: "DENY",
          priority: 0,
          node: everyNode(),
        },
      ],
    }),
  },
  {
    id: "api-read-only",
    title: "Octelium API read-only",
    description: "Auditors read every resource but change nothing",
    icon: "boxes",
    tone: "allow",
    build: () => ({
      enforcementRules: [
        {
          effect: "ENFORCE",
          node: groupNode("all", [
            block("api.octelium"),
            block(
              "protocol.grpcService",
              {
                match: {
                  op: "in",
                  values: [
                    "octelium.api.main.auth.v1.MainService",
                    "octelium.api.main.user.v1.MainService",
                  ],
                },
              },
              "condition",
            ),
          ]),
        },
        { effect: "IGNORE", node: everyNode() },
      ],
      rules: [
        {
          name: "allowed-methods",
          effect: "ALLOW",
          priority: -1,
          node: block("api.octelium", {
            apis: [],
            services: [],
            methods: [],
            readOnly: true,
          }),
        },
        {
          name: "deny-otherwise",
          effect: "DENY",
          priority: 0,
          node: everyNode(),
        },
      ],
    }),
  },
  {
    id: "business-hours",
    title: "Business hours only",
    description: "A guardrail denying access on nights and weekends",
    icon: "clock",
    tone: "deny",
    build: () => ({
      enforcementRules: [],
      rules: [
        {
          name: "outside-business-hours",
          effect: "DENY",
          priority: -2,
          node: businessHours("condition"),
        },
      ],
    }),
  },
  {
    id: "require-mfa",
    title: "Require MFA",
    description: "Deny Sessions that did not authenticate with a second factor",
    icon: "fingerprint",
    tone: "deny",
    build: () => ({
      enforcementRules: [],
      rules: [
        {
          name: "require-mfa",
          effect: "DENY",
          priority: -2,
          node: block("session.aal", { values: ["AAL2", "AAL3"] }, "condition"),
        },
      ],
    }),
  },
  {
    id: "trusted-devices",
    title: "Trusted devices only",
    description: "Deny Devices that are not bound and healthy",
    icon: "laptop",
    tone: "deny",
    build: () => {
      const node = healthyDevice();
      return {
        enforcementRules: [],
        rules: [
          {
            name: "untrusted-device",
            effect: "DENY",
            priority: -2,
            node: groupNode("none", [node]),
          },
        ],
      };
    },
  },
  {
    id: "geofence",
    title: "Geo-fence",
    description: "Deny Sessions authenticated outside allowed countries",
    icon: "globe",
    tone: "deny",
    build: () => ({
      enforcementRules: [],
      rules: [
        {
          name: "geofence",
          effect: "DENY",
          priority: -3,
          node: block("network.country", { codes: [] }, "condition"),
        },
      ],
    }),
  },
  {
    id: "temporary",
    title: "Temporary access",
    description: "Allow a Group until a date, then stop matching",
    icon: "hourglass",
    tone: "allow",
    build: () => ({
      enforcementRules: [],
      rules: [
        {
          name: "temporary-access",
          effect: "ALLOW",
          priority: 0,
          node: groupNode("all", [
            block("identity.group"),
            block("time.range", { after: "", before: inDays(7) }),
          ]),
        },
      ],
    }),
  },
  {
    id: "allow-all",
    title: "Allow everything",
    description: "A single rule that allows every request",
    icon: "check",
    tone: "allow",
    build: () => ({
      enforcementRules: [],
      rules: [
        { name: "main", effect: "ALLOW", priority: 0, node: everyNode() },
      ],
    }),
  },
  {
    id: "deny-all",
    title: "Deny everything",
    description: "A single rule that denies every request",
    icon: "ban",
    tone: "deny",
    build: () => ({
      enforcementRules: [],
      rules: [{ name: "main", effect: "DENY", priority: 0, node: everyNode() }],
    }),
  },
];

export const featuredRecipes = (ids: string[]) =>
  ids
    .map((id) => RECIPES.find((recipe) => recipe.id === id))
    .filter((recipe): recipe is Recipe => recipe !== undefined);

export const searchText = (...parts: (string | string[] | undefined)[]) =>
  parts.flat().filter(Boolean).join(" ").toLowerCase();

export const matchesQuery = (text: string, query: string) => {
  const words = query.toLowerCase().trim().split(/\s+/).filter(Boolean);
  return words.every((word) => text.includes(word));
};
