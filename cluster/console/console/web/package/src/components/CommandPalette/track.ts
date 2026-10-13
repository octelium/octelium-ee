import { resolveTrail } from "@/components/Navigation/catalog";
import * as React from "react";
import { useLocation } from "react-router-dom";
import { recordVisit, type RecentEntry } from "./recents";

export const visitFor = (
  pathname: string,
): Pick<RecentEntry, "path" | "title" | "subtitle"> | undefined => {
  const trail = resolveTrail(pathname);
  const resourceIdx = trail.findIndex((crumb) => crumb.resource);
  const effective = resourceIdx >= 0 ? trail.slice(0, resourceIdx + 1) : trail;
  const last = effective.at(-1);
  if (!last || last.label === "Create" || last.path === "/agent") {
    return undefined;
  }
  const parent = effective
    .slice(1, -1)
    .map((crumb) => crumb.label)
    .join(" › ");
  return {
    path: last.path,
    title: last.label,
    subtitle:
      resourceIdx >= 0
        ? `${trail[resourceIdx].resource!.kind} · ${trail[1].label}`
        : parent || undefined,
  };
};

export const useTrackVisits = () => {
  const { pathname } = useLocation();
  React.useEffect(() => {
    const visit = visitFor(pathname);
    if (!visit) return;
    const timer = window.setTimeout(() => recordVisit(visit), 600);
    return () => window.clearTimeout(timer);
  }, [pathname]);
};
