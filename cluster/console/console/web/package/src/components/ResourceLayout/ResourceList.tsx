import { getDomain } from "@/utils";
import {
  getAPIKindFromPath,
  getClientResourceList,
  getClientResourceListP,
  getListKeyFromPath,
  getPBResourceListFromAPI,
  getRefNameQueryArgStr,
  getResourcePath,
  Resource,
  ResourceList,
} from "@/utils/pb";
import { Badge, Collapse, Text, TextInput, Tooltip } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import {
  Link,
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import ResourceYAML from "../ResourceYAML";

import { ResourceComponentInfo } from "@/pages/utils/types";
import { Button } from "@mantine/core";

import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import { getServicePublicURL } from "@/utils/octelium";
import { Plus } from "lucide-react";
import { FaExternalLinkAlt } from "react-icons/fa";
import { MdModeEdit } from "react-icons/md";
import DeleteResource from "../DeleteResource";
import TimeAgo from "../TimeAgo";
import ResourceInfo, { ResourceVisibilityButtons } from "./ResourceInfo";
import { parseQueryString } from "./queryParse";

const ItemExtra = (props: { item: Resource; info: ResourceComponentInfo }) => {
  const { item } = props;

  /*
  const _hasAccessLog = hasAccessLog(item);
  const _hasAuthenticationLog = hasAuthenticationLog(item);
  const _hasAuditLog = hasAuditLog(item);
  let [tab, setTab] = React.useState<string | undefined>(undefined);
  */

  return (
    <div className="w-full">
      <ResourceInfo resource={item} />

      {props.info.Item.itemInfo && props.info.Item.itemInfo({ item })}
    </div>
  );

  {
    /**
  return (
    <div className="w-full">
      <Tabs
        value={tab}
        defaultValue="main"
        onChange={(v) => {
          setTab(v ?? undefined);
        }}
      >
        <Tabs.List className="mb-6">
          <Tabs.Tab value="main">Main</Tabs.Tab>
          {_hasAccessLog && <Tabs.Tab value="accesslogs">Access Logs</Tabs.Tab>}
          {_hasAuthenticationLog && (
            <Tabs.Tab value="authenticationlogs">Authentication Logs</Tabs.Tab>
          )}
          {_hasAuditLog && <Tabs.Tab value="auditlogs">Audit Logs</Tabs.Tab>}
          <Tabs.Tab value="actions">Actions</Tabs.Tab>
        </Tabs.List>
        <Tabs.Panel value="main">
          <ResourceInfo resource={item} />

          {props.info.Item.itemInfo && props.info.Item.itemInfo({ item })}
        </Tabs.Panel>
        {_hasAccessLog && (
          <Tabs.Panel value="accesslogs">
            {tab === `accesslogs` && (
              <ResourceAccessLogs resource={item} itemsPerPage={20} />
            )}
          </Tabs.Panel>
        )}

        {_hasAuthenticationLog && (
          <Tabs.Panel value="authenticationlogs">
            {tab === `authenticationlogs` && (
              <ResourceAuthenticationLogs resource={item} />
            )}
          </Tabs.Panel>
        )}

        {_hasAuditLog && (
          <Tabs.Panel value="auditlogs">
            {tab === `auditlogs` && <ResourceAuditLogs resource={item} />}
          </Tabs.Panel>
        )}

        <Tabs.Panel value="actions">
          {!props.info.unDeletable && (
            <div className="w-full flex items-center justify-end">
              <DeleteResource
                btnSize={`compact-xs`}
                btnVariant={`outline`}
                item={item}
                doNotNavigateAfter
              />
            </div>
          )}
        </Tabs.Panel>
      </Tabs>
    </div>
  );  
  **/
  }
};

const Item = (props: { item: Resource; info: ResourceComponentInfo }) => {
  const { item } = props;

  const md = item.metadata!;

  let [showDetails, setShowDetails] = React.useState(false);

  const qryNameArg = getRefNameQueryArgStr(item);

  return (
    <div
      className="font-semibold w-full"
      onMouseEnter={() => {
        setShowDetails(true);
      }}
      onMouseLeave={() => {
        setShowDetails(false);
      }}
    >
      <div className="flex">
        <div className="flex flex-col flex-1">
          <div className="flex items-center font-bold">
            {item.metadata?.picURL && (
              <div className="flex flex-col items-center justify-center mr-2">
                <img
                  src={item.metadata.picURL}
                  className="w-10 h-10 rounded-full"
                />
              </div>
            )}
            <div className="font-bold">
              <Link to={getResourcePath(item)}>
                <Text
                  component="span"
                  className="mr-2 flex flex-row"
                  size="sm"
                  fw={"bold"}
                >
                  <CopyText value={item.metadata!.name} />

                  {md.displayName && (
                    <Text component="span" className="ml-3" c="gray.7" inherit>
                      {md.displayName}
                    </Text>
                  )}
                </Text>
              </Link>

              <div className="flex items-center">
                {item.metadata?.isSystem && (
                  <Tooltip
                    className="mr-2"
                    label={
                      <>This is a system Resource created by the Cluster</>
                    }
                  >
                    <Badge size="xs" variant="outline" color="blue">
                      System
                    </Badge>
                  </Tooltip>
                )}
                <Text size="xs" fw={"bold"} c="dimmed">
                  Created <TimeAgo rfc3339={item.metadata!.createdAt} />{" "}
                  {item.metadata!.updatedAt && (
                    <>
                      (Updated <TimeAgo rfc3339={item.metadata!.updatedAt} />)
                    </>
                  )}
                </Text>
              </div>

              {md.description && (
                <Text size="sm" fw={"bold"} c="gray.7" lineClamp={1}>
                  {md.description}
                </Text>
              )}
            </div>
          </div>
          <div className="w-full flex items-center mt-1 mb-3">
            <ResourceYAML item={item} size="xs" />

            {!props.info.unEditable && !item.metadata?.isSystem && (
              <Button
                size={"compact-xs"}
                variant="outline"
                component={Link}
                className="mx-1"
                rightSection={<MdModeEdit />}
                to={`${getResourcePath(item)}/edit`}
              >
                Edit
              </Button>
            )}

            {item.apiVersion === `core/v1` &&
              item.kind === `Service` &&
              (item as Service).spec!.isPublic &&
              (item as Service).spec!.mode === Service_Spec_Mode.WEB && (
                <Button
                  size={"compact-xs"}
                  className="mx-1"
                  // variant="outline"
                  component={Link}
                  to={getServicePublicURL(item as Service, getDomain())}
                  target="_blank"
                  rightSection={<FaExternalLinkAlt />}
                >
                  <span>Visit</span>
                </Button>
              )}

            <ResourceVisibilityButtons item={item} />

            {/**
            {!props.info.unCreatable && (
              <Button
                size={"compact-xs"}
                variant="outline"
                component={Link}
                className="mx-2"
                rightSection={<MdModeEdit />}
                to={`${getResourcePathFromResource(item)}/create?cloneRef.uid=${
                  item.metadata?.uid
                }`}
              >
                Clone
              </Button>
            )} 
             **/}

            <div className="flex-1"></div>
            {!props.info.unDeletable && (
              <DeleteResource
                btnColor={`gray.7`}
                btnSize={`compact-xs`}
                btnVariant={`outline`}
                item={item}
                doNotNavigateAfter
              />
            )}
          </div>
          {props.info.List.labelComponent && (
            <div className="w-full">
              {props.info.List.labelComponent({ item })}
            </div>
          )}

          {
            <Collapse in={showDetails} transitionDuration={500}>
              <div className="mt-6 pt-3 border-t-zinc-300 border-t-[1px]">
                <ItemExtra item={item} info={props.info} />
              </div>
            </Collapse>
          }
        </div>
      </div>
    </div>
  );
};

const ResourceListC = (props: {
  itemsList: ResourceList;
  info: ResourceComponentInfo;
}) => {
  const navigate = useNavigate();

  return (
    <div>
      {!props.info.unCreatable && props.itemsList.items.length > 0 && (
        <div className="flex justify-end mb-6">
          <Button
            size={`lg`}
            className="font-bold text-white transition-all duration-500 rounded-full shadow-lg"
            onClick={() => {
              navigate("create");
            }}
          >
            <Plus />
            Create
          </Button>
        </div>
      )}
      {props.info.List.SummaryComponent !== undefined && (
        <props.info.List.SummaryComponent />
      )}
      {/*
      <ResourceListInfo info={props.info} itemList={props.itemsList} />
      */}
      <ResourceListWrapper>
        {props.itemsList.items.length === 0 && (
          <div className="flex flex-col items-center justify-center">
            <div
              className={twMerge(
                "flex text-center items-center justify-center",
                "font-bold text-4xl text-gray-600",
                "my-16"
              )}
            >
              No {props.itemsList.kind.replace(/List$/, "")} Found
            </div>
            {!props.info.unCreatable && (
              <div className="flex items-center justify-center">
                <Button
                  size={`lg`}
                  className="font-bold shadow-md transition-all duration-500 rounded-full py-7 px-14"
                  onClick={() => {
                    navigate("create");
                  }}
                >
                  <Plus />
                  Create
                </Button>
              </div>
            )}
          </div>
        )}

        <Paginator
          meta={props.itemsList.listResponseMeta}
          // path={getResourceListPath(props.itemsList)}
        />

        {props.itemsList.items.map((item) => (
          <ResourceListItem key={item.metadata!.uid}>
            <Item item={item} info={props.info} />
          </ResourceListItem>
        ))}
      </ResourceListWrapper>
      <Paginator
        meta={props.itemsList.listResponseMeta}
        // path={getResourceListPath(props.itemsList)}
      />
    </div>
  );
};

const SearchInput = (props: {}) => {
  let [searchParams, _] = useSearchParams();
  let [qry, setQry] = React.useState(searchParams.get("common.query") ?? "");
  const navigate = useNavigate();
  const loc = useLocation();

  return (
    <div className="w-full">
      <TextInput
        value={qry}
        onChange={(v) => {
          setQry(v.target.value);
          if (v.target.value.length < 3) {
            return;
          }
          searchParams.set("common.query", v.target.value);
          navigate(`${loc.pathname}?${searchParams.toString()}`);
        }}
      />
    </div>
  );
};

const useListReq = () => {
  // const settings = useAppSelector((state) => state.settings);

  let [searchParams, _] = useSearchParams();

  const searchParamsStr = searchParams.toString();

  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) {
    return undefined;
  }

  // @ts-ignore
  let req = getPBResourceListFromAPI(apiKind.api)![
    `List${apiKind.kind}Options`
  ]["create"]({
    common: CommonListOptions.create({}),
  });

  /*
  if (!!settings.listOptFilter) {
    // @ts-ignore
    getClientResourceListP(apiKind.api)![`List${apiKind.kind}Options`][
      "mergePartial"
    ](req, settings.listOptFilter);
  }
  */

  if (searchParamsStr.length > 0) {
    let parsedQry = parseQueryString<{
      type?: string;
      mode: string;
      common?: {
        page?: number;
        itemsPerPage?: number;
      };
      namespaceRef?: {
        uid?: string;
        name?: string;
      };
      userRef?: {
        uid?: string;
        name?: string;
      };
      deviceRef?: {
        uid?: string;
        name?: string;
      };
    }>(searchParams.toString());

    if (
      parsedQry.common &&
      parsedQry.common.page &&
      parsedQry.common.page > 0
    ) {
      parsedQry.common.page = parsedQry.common.page - 1;
    }

    // @ts-ignore
    const req2 = getClientResourceListP(apiKind.api)![
      `List${apiKind.kind}Options`
    ][`fromJsonString`](JSON.stringify(parsedQry));
    // @ts-ignore
    getClientResourceListP(apiKind.api)![`List${apiKind.kind}Options`][
      "mergePartial"
    ](req, req2);
  }

  return req;
};

const ResourceListPage = (props: { info: ResourceComponentInfo }) => {
  // const settings = useAppSelector((state) => state.settings);

  // let [searchParams, _] = useSearchParams();

  // searchParams.delete("page");

  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) {
    return <></>;
  }

  const req = useListReq();

  const { isLoading, isSuccess, data } = useQuery({
    queryKey: [
      getListKeyFromPath(loc.pathname),
      // @ts-ignore
      getPBResourceListFromAPI(apiKind.api)![`List${apiKind.kind}Options`][
        "toJsonString"
      ](req),
    ],
    queryFn: async () => {
      // @ts-ignore
      return await getClientResourceList(apiKind.api)[`list${apiKind.kind}`](
        req
      );
    },
  });

  if (!data || isLoading) {
    return <></>;
  }

  const itemList = data["response"] as ResourceList | undefined;

  /*
  if (!itemList) {
    return <></>;
  }
  */

  return (
    <div className="w-full">
      {itemList && <ResourceListC itemsList={itemList} info={props.info} />}
    </div>
  );
};

export default ResourceListPage;
