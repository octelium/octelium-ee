import type {
  PageContext,
  ResourceRef,
  RunContext,
} from "@/apis/consoleagent/protocol";
import type { Crumb } from "@/components/Navigation/catalog";
import type { PinnedContext } from "./context";

export const trailTitle = (trail: Crumb[]): string =>
  trail
    .slice(trail.length > 1 ? 1 : 0)
    .map((crumb) => crumb.label)
    .join(" › ");

export const routeResources = (trail: Crumb[]): ResourceRef[] =>
  trail.flatMap((crumb) =>
    crumb.resource
      ? [
          {
            apiVersion: `${crumb.resource.api}/v1`,
            kind: crumb.resource.kind,
            name: crumb.resource.name,
          },
        ]
      : [],
  );

export const buildPageContext = (args: {
  pathname: string;
  search: string;
  trail: Crumb[];
  details?: Record<string, string>;
}): PageContext => ({
  path: `${args.pathname}${args.search}`,
  title: trailTitle(args.trail),
  details: args.details,
});

const refKey = (ref: ResourceRef) =>
  `${ref.apiVersion}/${ref.kind}/${ref.name ?? ""}/${ref.uid ?? ""}`;

export const buildRunContext = (args: {
  page?: PageContext;
  routeResources: ResourceRef[];
  pinned?: PinnedContext;
}): RunContext | undefined => {
  const seen = new Set<string>();
  const resources = [
    ...(args.page ? args.routeResources : []),
    ...(args.pinned?.resources ?? []),
  ].filter((ref) => {
    const key = refKey(ref);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  const ret: RunContext = {};
  if (args.page) ret.page = args.page;
  if (resources.length > 0) ret.resources = resources;
  if (args.pinned?.items && args.pinned.items.length > 0) {
    ret.items = args.pinned.items;
  }
  return Object.keys(ret).length > 0 ? ret : undefined;
};

export const contextAsText = (ctx: RunContext): string => {
  const lines: string[] = [];
  if (ctx.page) {
    lines.push(`Page: ${ctx.page.title ?? ""} (${ctx.page.path})`.trim());
    for (const [key, value] of Object.entries(ctx.page.details ?? {})) {
      lines.push(`${key}: ${value}`);
    }
  }
  for (const ref of ctx.resources ?? []) {
    lines.push(`${ref.kind}: ${ref.name ?? ref.uid ?? ""}`);
  }
  for (const item of ctx.items ?? []) {
    lines.push(
      `${item.kind}${item.title ? ` (${item.title})` : ""}: ${JSON.stringify(item.data ?? {})}`,
    );
  }
  return `[Console context]\n${lines.join("\n")}`;
};

export const pinnedResource = (resource: {
  apiVersion: string;
  kind: string;
  metadata?: { name: string; uid: string };
}): PinnedContext => ({
  label: `${resource.kind} ${resource.metadata?.name ?? ""}`.trim(),
  resources: [
    {
      apiVersion: resource.apiVersion,
      kind: resource.kind,
      name: resource.metadata?.name,
      uid: resource.metadata?.uid,
    },
  ],
});

const plural = (kind: string) =>
  /[^aeiou]y$/i.test(kind) ? `${kind.slice(0, -1)}ies` : `${kind}s`;

export const pinnedResources = (
  resources: {
    apiVersion: string;
    kind: string;
    metadata?: { name: string; uid: string };
  }[],
): PinnedContext =>
  resources.length === 1
    ? pinnedResource(resources[0])
    : {
        label: `${resources.length} ${plural(resources[0]?.kind ?? "resource")}`,
        resources: resources.map((resource) => ({
          apiVersion: resource.apiVersion,
          kind: resource.kind,
          name: resource.metadata?.name,
          uid: resource.metadata?.uid,
        })),
      };
