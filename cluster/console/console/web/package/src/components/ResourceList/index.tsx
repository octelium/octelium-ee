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
import { Link, useNavigate } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import Label from "../Label";
import ResourceInfo from "../ResourceLayout/ResourceInfo";
import { useResourceFromRef } from "../ResourceLayout/utils";
import TimeAgo from "../TimeAgo";

export const ResourceListWrapper = (props: { children?: React.ReactNode }) => (
  <div className="flex flex-col w-full gap-3">{props.children}</div>
);

export const ResourceListItem = (props: {
  children?: React.ReactNode;
  path?: string;
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
      onClick={() => {
        if (hasPath) navigate(props.path!);
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
}: {
  label?: string;
  children: React.ReactNode;
}) => (
  <Label isLink>
    {label && <span className="text-blue-300 font-bold mr-0.5">{label}</span>}
    <span className="flex items-center">{children}</span>
  </Label>
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
      <span>{props.children}</span>
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
    <LabelContent label={props.label}>{props.children}</LabelContent>
  );

  return props.to ? <Link to={props.to}>{content}</Link> : content;
};

const ResourceListLabelWithItemRef = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef: ObjectReference;
}) => {
  const r = useResourceFromRef(props.itemRef);
  if (!r?.isSuccess || !r.data) return null;

  const displayName =
    r.data.apiVersion === "core/v1" && r.data.kind === "User"
      ? printUserWithEmail(r.data as User)
      : printResourceNameWithDisplay(r.data);

  return (
    <ResourceHoverCardWrapper data={r.data}>
      <Link to={getResourcePath(r.data)}>
        <LabelContent label={props.label ?? r.data.kind}>
          {props.children ?? displayName}
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
  <div className="w-full mt-1.5 flex flex-row flex-wrap gap-1">
    {props.children}
  </div>
);
