import { ObjectReference } from "@/apis/metav1/metav1";
import { ResourceComponentInfo } from "@/pages/utils/types";
import {
  getAPIFromAPIVersion,
  getResourcePathFromAPIKind,
  getShortName,
  Resource,
  ResourceList,
} from "@/utils/pb";
import { Link2 } from "lucide-react";
import { Link, useLocation } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import TimeAgo from "../TimeAgo";

export const ResourceListWrapper = (props: { children?: React.ReactNode }) => (
  <div className="flex flex-col w-full gap-3">{props.children}</div>
);

export const ResourceListItem = (props: {
  children?: React.ReactNode;
  path?: string;
  compact?: boolean;
  overlayLabel?: string;
  state?: unknown;
}) => {
  const location = useLocation();
  const isActive =
    !!props.path?.length &&
    (location.pathname === props.path ||
      location.pathname.startsWith(`${props.path}/`));

  return (
    <div
      className={twMerge(
        "relative w-full bg-white",
        "border border-slate-200 rounded-xl",
        "shadow-[0_1px_3px_rgba(15,23,42,0.04)]",
        props.compact ? "px-3 py-2.5 sm:px-4" : "px-4 py-3.5 sm:px-5 sm:py-4",
        "transition-[background-color,border-color,box-shadow] duration-500 ease-out",
        "hover:border-slate-300 hover:bg-transparent hover:shadow-[0_10px_28px_-8px_rgba(15,23,42,0.12)]",
        isActive && "border-slate-300 shadow-raised",
      )}
      aria-current={isActive ? "page" : undefined}
    >
      {props.path && props.overlayLabel && (
        <Link
          to={props.path}
          state={props.state}
          preventScrollReset
          aria-label={props.overlayLabel}
          className="absolute inset-0 z-[1] rounded-xl outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40"
        />
      )}
      {props.children}
    </div>
  );
};

export const ResourceListItemMetadata = (props: {
  resource: Resource;
  noName?: boolean;
}) => {
  const { resource } = props;
  const md = resource.metadata!;
  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex items-center gap-2">
        {!props.noName && (
          <span className="text-sm font-bold text-slate-800">
            {getShortName(resource)}
          </span>
        )}
        {md.displayName && (
          <span className="text-sm font-semibold text-slate-500">
            {md.displayName}
          </span>
        )}
      </div>
      {md.createdAt && (
        <div className="text-xs font-normal text-slate-500">
          Created <TimeAgo rfc3339={md.createdAt} />
        </div>
      )}
    </div>
  );
};

export const ResourceListInfo = (props: {
  itemList: ResourceList;
  info: ResourceComponentInfo;
}) => {
  const count = props.itemList.listResponseMeta?.totalCount;
  if (!count) return null;
  return (
    <div className="w-full flex justify-end mb-1">
      <span className="text-xs font-semibold text-slate-500 tracking-wide">
        {count.toLocaleString()} items
      </span>
    </div>
  );
};

const LabelContent = ({
  label,
  children,
  interactive,
  reference,
}: {
  label?: string;
  children: React.ReactNode;
  interactive?: boolean;
  reference?: boolean;
}) => (
  <span
    className={twMerge(
      "inline-flex min-h-6 max-w-full items-center gap-1.5 rounded-md border px-2 py-1 align-middle",
      "bg-slate-50/80 border-slate-200 text-xs font-normal leading-none text-slate-600",
      "shadow-card transition-[background-color,border-color,color,box-shadow] duration-200",
      "[&_svg]:size-3 [&_svg]:shrink-0",
      interactive &&
        "cursor-pointer border-blue-200/80 bg-blue-50/70 text-blue-700 hover:border-blue-300 hover:bg-white hover:text-blue-800 hover:shadow-card",
    )}
  >
    {reference && <Link2 aria-hidden="true" />}
    {label && (
      <>
        <span className="shrink-0 font-bold text-slate-600">{label}</span>
        <span
          aria-hidden="true"
          className={twMerge(
            "h-3 w-px shrink-0 bg-slate-200",
            interactive && "bg-blue-200",
          )}
        />
      </>
    )}
    <span className="flex min-w-0 items-center gap-1 [&>span]:min-w-0">
      {children}
    </span>
  </span>
);

const ResourceListLabelContent = (props: {
  children?: React.ReactNode;
  label?: string;
  to?: string;
}) => {
  const content = (
    <LabelContent label={props.label} interactive={!!props.to}>
      {props.children}
    </LabelContent>
  );

  return props.to ? (
    <Link
      to={props.to}
      preventScrollReset
      className="inline-flex max-w-full rounded-md align-middle outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 focus-visible:ring-offset-1"
    >
      {content}
    </Link>
  ) : (
    content
  );
};

const ResourceListLabelReference = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef: ObjectReference;
}) => {
  const location = useLocation();
  const { itemRef } = props;
  const api = getAPIFromAPIVersion(itemRef.apiVersion);
  const kindPath = api
    ? getResourcePathFromAPIKind({ api, kind: itemRef.kind as any })
    : undefined;
  const path =
    api && kindPath && itemRef.name
      ? `/${api}/${kindPath}/${itemRef.name}`
      : undefined;
  const displayName = props.children ?? (itemRef.name || itemRef.uid);

  if (!displayName) return null;

  const content = (
    <LabelContent
      label={props.label || itemRef.kind || "Resource"}
      interactive={!!path}
      reference
    >
      <span className="truncate">{displayName}</span>
    </LabelContent>
  );

  if (!path) return content;

  return (
    <Link
      to={path}
      state={{ returnTo: `${location.pathname}${location.search}` }}
      preventScrollReset
      className="inline-flex max-w-full rounded-md align-middle outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 focus-visible:ring-offset-1"
    >
      {content}
    </Link>
  );
};

export const ResourceListLabel = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef?: ObjectReference;
  to?: string;
}) =>
  props.itemRef ? (
    <ResourceListLabelReference label={props.label} itemRef={props.itemRef}>
      {props.children}
    </ResourceListLabelReference>
  ) : (
    <ResourceListLabelContent label={props.label} to={props.to}>
      {props.children}
    </ResourceListLabelContent>
  );

export const ResourceListLabelWrap = (props: {
  children?: React.ReactNode;
}) => (
  <div className="relative z-10 mt-2 flex w-full flex-row flex-wrap content-start items-center gap-1.5">
    {props.children}
  </div>
);
