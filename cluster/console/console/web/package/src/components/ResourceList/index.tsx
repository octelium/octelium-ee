import { User } from "@/apis/corev1/corev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { ResourceComponentInfo } from "@/pages/utils/types";
import {
  getResourcePath,
  getShortName,
  printResourceNameWithDisplay,
  printUserWithEmail,
  Resource,
  ResourceList,
} from "@/utils/pb";
import { HoverCard } from "@mantine/core";
import { Link2 } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import ResourceInfo from "../ResourceLayout/ResourceInfo";
import { useResourceFromRef } from "../ResourceLayout/utils";
import TimeAgo from "../TimeAgo";

export const ResourceListWrapper = (props: { children?: React.ReactNode }) => (
  <div className="flex flex-col w-full gap-3">{props.children}</div>
);

export const ResourceListItem = (props: {
  children?: React.ReactNode;
  path?: string;
  state?: unknown;
}) => {
  const hasPath = !!props.path?.length;
  const navigate = useNavigate();

  return (
    <div
      className={twMerge(
        "w-full bg-white",
        "border border-slate-200 rounded-xl",
        "shadow-[0_1px_4px_rgba(15,23,42,0.06)]",
        "px-5 py-4",
        "transition-[border-color,box-shadow] duration-150",
        "hover:border-slate-300 hover:shadow-[0_2px_12px_rgba(15,23,42,0.09)]",
        hasPath && "cursor-pointer",
      )}
      role={hasPath ? "link" : undefined}
      tabIndex={hasPath ? 0 : undefined}
      onClick={(event) => {
        if (
          (event.target as HTMLElement).closest(
            "a, button, input, select, textarea, [role='button']",
          )
        ) {
          return;
        }
        if (hasPath) {
          navigate(props.path!, {
            state: props.state,
            preventScrollReset: true,
          });
        }
      }}
      onKeyDown={(event) => {
        if (hasPath && (event.key === "Enter" || event.key === " ")) {
          event.preventDefault();
          navigate(props.path!, {
            state: props.state,
            preventScrollReset: true,
          });
        }
      }}
    >
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
        <div className="text-[0.72rem] font-semibold text-slate-400">
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
      <span className="text-[0.72rem] font-semibold text-slate-400 tracking-wide">
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
      "bg-slate-50/80 border-slate-200 text-[0.7rem] font-semibold leading-none text-slate-600",
      "shadow-[0_1px_1px_rgba(15,23,42,0.03)] transition-[background-color,border-color,color,box-shadow] duration-500",
      "[&_svg]:size-3 [&_svg]:shrink-0",
      interactive &&
        "cursor-pointer border-blue-200/80 bg-blue-50/70 text-blue-700 hover:border-blue-300 hover:bg-white hover:text-blue-800 hover:shadow-[0_2px_6px_rgba(37,99,235,0.10)]",
    )}
  >
    {reference && <Link2 aria-hidden="true" />}
    {label && (
      <>
        <span className="shrink-0 font-bold text-slate-400">{label}</span>
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

const ResourceHoverCardWrapper = (props: {
  children: React.ReactNode;
  data: Resource;
}) => (
  <HoverCard
    width={460}
    shadow="md"
    withArrow
    openDelay={200}
    closeDelay={400}
    transitionProps={{ transition: "pop" }}
    zIndex={30}
  >
    <HoverCard.Target>
      <span className="inline-flex max-w-full align-middle">
        {props.children}
      </span>
    </HoverCard.Target>
    <HoverCard.Dropdown>
      <ResourceInfo resource={props.data} />
    </HoverCard.Dropdown>
  </HoverCard>
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

const ResourceListLabelWithItemRef = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef: ObjectReference;
}) => {
  const r = useResourceFromRef(props.itemRef);
  if (!r?.isSuccess || !r.data) {
    const fallback = props.itemRef.name || props.itemRef.uid;
    if (!fallback) return null;

    return (
      <LabelContent label={props.label ?? "Resource"} reference>
        <span className="truncate">{fallback}</span>
      </LabelContent>
    );
  }

  const displayName =
    r.data.apiVersion === "core/v1" && r.data.kind === "User"
      ? printUserWithEmail(r.data as User)
      : printResourceNameWithDisplay(r.data);

  return (
    <ResourceHoverCardWrapper data={r.data}>
      <Link
        to={getResourcePath(r.data)}
        preventScrollReset
        className="inline-flex max-w-full rounded-md align-middle outline-none focus-visible:ring-2 focus-visible:ring-blue-500/40 focus-visible:ring-offset-1"
      >
        <LabelContent
          label={props.label ?? r.data.kind}
          interactive
          reference
        >
          <span className="truncate">{props.children ?? displayName}</span>
        </LabelContent>
      </Link>
    </ResourceHoverCardWrapper>
  );
};

export const ResourceHoverCard = (props: {
  children?: React.ReactNode;
  itemRef: ObjectReference;
}) => {
  const r = useResourceFromRef(props.itemRef);
  if (!r?.isSuccess || !r.data) return null;

  return (
    <ResourceHoverCardWrapper data={r.data}>
      {props.children}
    </ResourceHoverCardWrapper>
  );
};

export const ResourceListLabel = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef?: ObjectReference;
  to?: string;
}) =>
  props.itemRef ? (
    <ResourceListLabelWithItemRef label={props.label} itemRef={props.itemRef}>
      {props.children}
    </ResourceListLabelWithItemRef>
  ) : (
    <ResourceListLabelContent label={props.label} to={props.to}>
      {props.children}
    </ResourceListLabelContent>
  );

export const ResourceListLabelWrap = (props: {
  children?: React.ReactNode;
}) => (
  <div className="mt-2 flex w-full flex-row flex-wrap content-start items-center gap-1.5">
    {props.children}
  </div>
);
