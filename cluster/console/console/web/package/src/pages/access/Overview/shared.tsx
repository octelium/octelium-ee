import { Request } from "@/apis/accessv1/accessv1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ListTopResponse } from "@/apis/visibilityv1/access/vaccessv1";
import { compact, exact } from "@/components/Dashboard/utils";
import { twMerge } from "@/lib/utils";
import { Layers, LucideIcon, Server, Shield } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { Avatar, getRefPath } from "../common";

export const resourceOf = (item: Request): ObjectReference | undefined => {
  const resource = item.spec?.resource?.type;
  if (resource?.oneofKind === "serviceRef")
    return ObjectReference.create({
      ...resource.serviceRef,
      apiVersion: resource.serviceRef.apiVersion || "core/v1",
      kind: resource.serviceRef.kind || "Service",
    });
  if (resource?.oneofKind === "catalog" && resource.catalog.catalogRef)
    return ObjectReference.create({
      ...resource.catalog.catalogRef,
      apiVersion: resource.catalog.catalogRef.apiVersion || "access/v1",
      kind: resource.catalog.catalogRef.kind || "Catalog",
    });
  return undefined;
};

export const resourceIcon = (ref?: ObjectReference): LucideIcon =>
  ref?.kind === "Catalog" ? Layers : ref?.kind === "Policy" ? Shield : Server;

export const subjectOf = (item: Request): ObjectReference | undefined => {
  const subject = item.spec?.subject?.type;
  return subject?.oneofKind === "userRef" ? subject.userRef : undefined;
};

export const TopRefList = (props: {
  title: string;
  items?: ListTopResponse["items"];
  totalCount?: number;
  totalOther?: number;
  unit: string;
  icon: LucideIcon;
  isLoading?: boolean;
  emptyLabel: string;
  avatars?: boolean;
}) => {
  const location = useLocation();
  const items = props.items ?? [];
  const max = Math.max(1, ...items.map((item) => Number(item.count)));
  const Icon = props.icon;

  return (
    <div className="flex min-w-0 flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
      <div className="flex items-center gap-1.5">
        <Icon size={12} strokeWidth={2.3} className="shrink-0 text-slate-500" />
        <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
          {props.title}
        </span>
      </div>

      {props.isLoading ? (
        <div className="flex flex-col gap-1.5">
          {[0, 1, 2].map((index) => (
            <span
              key={index}
              className="block h-8 motion-safe:animate-pulse rounded-md bg-white"
            />
          ))}
        </div>
      ) : items.length === 0 ? (
        <span className="py-3 text-center text-xs font-normal text-slate-500">
          {props.emptyLabel}
        </span>
      ) : (
        <ol className="flex flex-col gap-1">
          {items.map((item) => {
            const ref = item.ref;
            const path = getRefPath(ref);
            const name = item.displayName || ref?.name || ref?.uid || "Unknown";
            const share = (Number(item.count) / max) * 100;
            const ItemIcon = resourceIcon(ref);
            const content = (
              <>
                <span
                  aria-hidden="true"
                  className="absolute inset-y-0 left-0 rounded-md bg-blue-100/60 transition-[width] duration-300"
                  style={{ width: `${share}%` }}
                />
                <span className="relative flex min-w-0 flex-1 items-center gap-2">
                  {props.avatars ? (
                    <Avatar name={name} src={item.picURL} size="xs" />
                  ) : (
                    <ItemIcon
                      size={12}
                      strokeWidth={2.3}
                      className="shrink-0 text-slate-500"
                    />
                  )}
                  <span className="min-w-0 truncate text-xs font-semibold text-slate-700">
                    {name}
                  </span>
                  {item.displayName && ref?.name && (
                    <span className="hidden min-w-0 truncate text-micro font-normal text-slate-500 sm:inline">
                      {ref.name}
                    </span>
                  )}
                </span>
                <span
                  className="relative shrink-0 text-xs font-semibold tabular-nums text-slate-800"
                  title={exact(Number(item.count))}
                >
                  {compact(Number(item.count))}
                </span>
              </>
            );

            const className =
              "relative flex min-h-8 min-w-0 items-center gap-2 overflow-hidden rounded-md border border-slate-200 bg-white px-2.5 py-1.5";

            return (
              <li key={ref?.uid || name}>
                {path ? (
                  <Link
                    to={path}
                    state={{ returnTo: `${location.pathname}${location.search}` }}
                    preventScrollReset
                    className={twMerge(
                      className,
                      "outline-none transition-[border-color,box-shadow] duration-150 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
                    )}
                  >
                    {content}
                  </Link>
                ) : (
                  <div className={className}>{content}</div>
                )}
              </li>
            );
          })}
          {(props.totalOther ?? 0) > 0 && (
            <li className="px-1 text-micro font-normal text-slate-500">
              + {compact(props.totalOther ?? 0)} {props.unit}
              {(props.totalCount ?? 0) > items.length &&
                ` from ${compact((props.totalCount ?? 0) - items.length)} more`}
            </li>
          )}
        </ol>
      )}
    </div>
  );
};
