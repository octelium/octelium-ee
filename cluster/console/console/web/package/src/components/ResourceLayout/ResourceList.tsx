import { getDomain } from "@/utils";
import {
  getAPIKindFromPath,
  getClientResourceList,
  getClientResourceListP,
  getListKeyFromPath,
  getPBResourceListFromAPI,
  getResourcePath,
  Resource,
  ResourceList,
} from "@/utils/pb";
import { Tooltip } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import React from "react";
import {
  Link,
  Outlet,
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import ResourceYAML from "../ResourceYAML";

import { ResourceComponentInfo } from "@/pages/utils/types";
import { Button } from "@mantine/core";

import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import { getServicePublicURL } from "@/utils/octelium";
import { ExternalLink, Pencil, Plus } from "lucide-react";
import DeleteResource from "../DeleteResource";
import TimeAgo from "../TimeAgo";
import CloneResource from "./CloneResource";
import { parseQueryString } from "./queryParse";
import ResourceInfo, { ResourceVisibilityButtons } from "./ResourceInfo";

const ItemExtra = (props: { item: Resource; info: ResourceComponentInfo }) => {
  return (
    <div className="w-full">
      <ResourceInfo resource={props.item} />
      {props.info.Item.itemInfo &&
        props.info.Item.itemInfo({ item: props.item })}
    </div>
  );
};

const Item = (props: { item: Resource; info: ResourceComponentInfo }) => {
  const { item } = props;
  const md = item.metadata!;
  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  return (
    <div className="w-full font-semibold">
      <div className="flex gap-3">
        {md.picURL && (
          <div className="shrink-0 mt-0.5">
            <img
              src={md.picURL}
              className="w-9 h-9 rounded-full border border-slate-200"
            />
          </div>
        )}

        <div className="flex flex-col flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div className="flex flex-col gap-0.5 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <Link
                  to={getResourcePath(item)}
                  state={{ returnTo }}
                  preventScrollReset
                  className="text-[0.92rem] font-bold text-slate-800 hover:text-slate-900 transition-colors duration-150"
                  onClick={(e) => e.stopPropagation()}
                >
                  <CopyText value={md.name} />
                </Link>

                {md.displayName && (
                  <span className="text-[0.85rem] font-semibold text-slate-400">
                    {md.displayName}
                  </span>
                )}

                {md.isSystem && (
                  <Tooltip
                    label="This is a system resource created by the cluster"
                    withArrow
                  >
                    <span className="inline-flex items-center px-1.5 py-px text-[0.65rem] font-bold uppercase tracking-wider rounded border border-blue-200 text-blue-600 bg-blue-50 leading-none">
                      System
                    </span>
                  </Tooltip>
                )}
              </div>

              {md.description && (
                <p className="text-[0.8rem] font-semibold text-slate-500 truncate max-w-xl">
                  {md.description}
                </p>
              )}

              <div className="text-[0.72rem] font-semibold text-slate-500 mt-0.5">
                Created <TimeAgo rfc3339={md.createdAt} />
                {md.updatedAt && (
                  <span className="ml-2 text-slate-400">
                    · Updated <TimeAgo rfc3339={md.updatedAt} />
                  </span>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center flex-wrap gap-1 mt-2.5">
            <ResourceYAML item={item} size="xs" />

            {!props.info.unEditable && !md.isSystem && (
              <Button
                size="compact-xs"
                variant="outline"
                component={Link}
                to={`${getResourcePath(item)}/edit`}
                state={{ returnTo }}
                preventScrollReset
                leftSection={<Pencil size={11} />}
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
              >
                Edit
              </Button>
            )}

            {props.info.cloneable && <CloneResource item={item} />}

            {item.apiVersion === "core/v1" &&
              item.kind === "Service" &&
              (item as Service).spec?.isPublic &&
              (item as Service).spec?.mode === Service_Spec_Mode.WEB && (
                <Button
                  size="compact-xs"
                  variant="outline"
                  color="blue"
                  component={Link}
                  to={getServicePublicURL(item as Service, getDomain())}
                  target="_blank"
                  rel="noopener noreferrer"
                  leftSection={<ExternalLink size={11} />}
                  onClick={(e: React.MouseEvent) => e.stopPropagation()}
                  styles={{
                    root: { fontWeight: 600, fontSize: "0.72rem" },
                  }}
                >
                  Visit
                </Button>
              )}

            <ResourceVisibilityButtons item={item} />

            <div className="flex-1" />

            <div className="flex items-center gap-1">
              {!props.info.unDeletable && (
                <DeleteResource
                  btnColor="gray.6"
                  btnSize="compact-xs"
                  btnVariant="outline"
                  item={item}
                  doNotNavigateAfter
                />
              )}
            </div>
          </div>

          {props.info.List.labelComponent && (
            <div className="w-full mt-1">
              {props.info.List.labelComponent({ item })}
            </div>
          )}

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
  const location = useLocation();
  const kindName = props.itemsList.kind.replace(/List$/, "");
  const totalCount = props.itemsList.listResponseMeta?.totalCount ?? 0;

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        {totalCount > 0 ? (
          <span className="text-[0.72rem] font-semibold text-slate-400 tracking-wide">
            {totalCount.toLocaleString()} {kindName.toLowerCase()}s
          </span>
        ) : (
          <span />
        )}

        {!props.info.unCreatable && props.itemsList.items.length > 0 && (
          <Button
            variant="filled"
            leftSection={<Plus size={14} />}
            onClick={() => navigate("create")}
          >
            Create {kindName}
          </Button>
        )}
      </div>

      {props.info.List.SummaryComponent !== undefined && (
        <div className="mb-6">
          <props.info.List.SummaryComponent />
        </div>
      )}

      {props.itemsList.items.length === 0 && (
        <div className="flex flex-col items-center justify-center py-20 gap-5">
          <div className="text-[0.78rem] font-bold uppercase tracking-[0.08em] text-slate-400">
            No {kindName}s found
          </div>
          {!props.info.unCreatable && (
            <Button
              variant="filled"
              leftSection={<Plus size={14} />}
              onClick={() => navigate("create")}
            >
              Create {kindName}
            </Button>
          )}
        </div>
      )}

      <Paginator meta={props.itemsList.listResponseMeta} />

      <ResourceListWrapper>
        {props.itemsList.items.map((item, i) => (
          <motion.div
            key={item.metadata!.uid}
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.18, delay: i * 0.03, ease: "easeOut" }}
          >
            <ResourceListItem
              path={getResourcePath(item)}
              state={{
                returnTo: `${location.pathname}${location.search}`,
              }}
            >
              <Item item={item} info={props.info} />
            </ResourceListItem>
          </motion.div>
        ))}
      </ResourceListWrapper>

      <Paginator meta={props.itemsList.listResponseMeta} />
    </div>
  );
};

const useListReq = () => {
  const [searchParams] = useSearchParams();
  const searchParamsStr = searchParams.toString();
  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) return undefined;

  // @ts-ignore
  let req = getPBResourceListFromAPI(apiKind.api)![
    // @ts-ignore
    `List${apiKind.kind}Options`
  ]["create"]({
    common: CommonListOptions.create({}),
  });

  if (searchParamsStr.length > 0) {
    let parsedQry = parseQueryString<{
      type?: string;
      mode: string;
      common?: { page?: number; itemsPerPage?: number };
      namespaceRef?: { uid?: string; name?: string };
      userRef?: { uid?: string; name?: string };
      deviceRef?: { uid?: string; name?: string };
    }>(searchParams.toString());

    if (parsedQry.common?.page && parsedQry.common.page > 0) {
      parsedQry.common.page = parsedQry.common.page - 1;
    }

    // @ts-ignore
    const req2 = getClientResourceListP(apiKind.api)![
      // @ts-ignore
      `List${apiKind.kind}Options`
    ]["fromJsonString"](JSON.stringify(parsedQry));
    // @ts-ignore
    getClientResourceListP(apiKind.api)![`List${apiKind.kind}Options`][
      "mergePartial"
    ](req, req2);
  }

  return req;
};

const ResourceListContent = (props: { info: ResourceComponentInfo }) => {
  const loc = useLocation();
  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) return null;

  const req = useListReq();

  const { isLoading, data } = useQuery({
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
        req,
      );
    },
  });

  if (!data || isLoading) return null;

  const itemList = data["response"] as ResourceList | undefined;

  return (
    <div className="w-full">
      {itemList && <ResourceListC itemsList={itemList} info={props.info} />}
    </div>
  );
};

const ResourceListPage = (props: { info: ResourceComponentInfo }) => {
  const location = useLocation();
  const isCreatePage =
    location.pathname.split("/").filter(Boolean).at(-1) === "create";

  if (isCreatePage) return <Outlet />;

  return (
    <>
      <ResourceListContent info={props.info} />
      <Outlet />
    </>
  );
};

export default ResourceListPage;
