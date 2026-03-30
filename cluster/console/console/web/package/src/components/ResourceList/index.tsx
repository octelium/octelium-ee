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

export const ResourceListWrapper = (props: { children?: React.ReactNode }) => {
  return <div className="flex flex-col w-full">{props.children}</div>;
};

export const ResourceListItem = (props: {
  children?: React.ReactNode;
  path?: string;
}) => {
  const hasPath = props.path !== undefined && props.path.length > 0;
  const navigate = useNavigate();
  return (
    <div
      className={twMerge(
        "w-full",
        hasPath ? "cursor-pointer" : undefined,
        "transition-all duration-300",
        "bg-white",
        // "hover:bg-transparent",
        "py-4 px-2",
        "font-semibold",
        "rounded-xl",
        "shadow-sm shadow-slate-200",
        "border-[2px] border-slate-300",
        "mb-4",
      )}
      onClick={() => {
        if (hasPath) {
          navigate(props.path!);
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
    <div className="flex flex-col flex-1">
      <div className="flex items-center font-bold">
        {!props.noName && (
          <span className="text-gray-800 mr-2">{getShortName(resource)}</span>
        )}
        {md.displayName && (
          <span className="text-gray-600">{md.displayName}</span>
        )}
      </div>

      {md.createdAt && (
        <div className="flex items-center text-xs text-gray-500 font-bold">
          <span className="mr-1">Created</span>
          <span>
            <TimeAgo rfc3339={md.createdAt} />
          </span>
        </div>
      )}
    </div>
  );
};

export const ResourceListInfo = (props: {
  itemList: ResourceList;
  info: ResourceComponentInfo;
}) => {
  const { itemList } = props;
  return (
    <div className="w-full font-bold text-zinc-700 flex flex-col items-center justify-center my-6">
      {itemList.listResponseMeta &&
        itemList.listResponseMeta.totalCount > 0 && (
          <span>{itemList.listResponseMeta.totalCount} Items</span>
        )}
      <div></div>
    </div>
  );
};

const ResourceListLabelContent = (props: {
  children?: React.ReactNode;
  label?: string;
  to?: string;
}) => {
  return props.to ? (
    <Link to={props.to}>
      <Label isLink={!!props.to}>
        {props.label && (
          <span className="text-blue-300 mr-1">{props.label}</span>
        )}
        <span className="flex items-center">{props.children}</span>
      </Label>
    </Link>
  ) : (
    <Label isLink={!!props.to}>
      {props.label && <span className="text-blue-300 mr-1">{props.label}</span>}
      <span className="flex items-center">{props.children}</span>
    </Label>
  );
};

const ResourceListLabelWithItemRef = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef: ObjectReference;
  to?: string;
}) => {
  const r = useResourceFromRef(props.itemRef);

  if (!r || !r.isSuccess || !r.data) {
    return <></>;
  }

  return (
    <HoverCard
      width={460}
      shadow="md"
      withArrow
      openDelay={200}
      closeDelay={400}
      transitionProps={{
        transition: "pop",
      }}
      zIndex={30}
    >
      <HoverCard.Target>
        <span>
          {props.children ? (
            <ResourceListLabelContent
              label={props.label ? props.label : r.data?.kind}
              to={getResourcePath(r.data)}
            >
              {props.children}
            </ResourceListLabelContent>
          ) : (
            <ResourceListLabelContent
              label={props.label ? props.label : r.data.kind}
              to={getResourcePath(r.data)}
            >
              {r.data.apiVersion === `core/v1` && r.data.kind === `User`
                ? printUserWithEmail(r.data as User)
                : printResourceNameWithDisplay(r.data)}
            </ResourceListLabelContent>
          )}
        </span>
      </HoverCard.Target>
      <HoverCard.Dropdown className="shadow-md">
        <div className="w-full">
          <ResourceInfo resource={r.data} />
        </div>
      </HoverCard.Dropdown>
    </HoverCard>
  );
};

export const ResourceListLabel = (props: {
  children?: React.ReactNode;
  label?: string;
  itemRef?: ObjectReference;
  to?: string;
}) => {
  return props.itemRef ? (
    <ResourceListLabelWithItemRef label={props.label} itemRef={props.itemRef}>
      {props.children}
    </ResourceListLabelWithItemRef>
  ) : (
    <ResourceListLabelContent label={props.label} to={props.to}>
      {props.children}
    </ResourceListLabelContent>
  );
};

export const ResourceListLabelWrap = (props: {
  children?: React.ReactNode;
}) => {
  return (
    <div className="w-full mt-1 flex flex-row flex-wrap">{props.children}</div>
  );
};
