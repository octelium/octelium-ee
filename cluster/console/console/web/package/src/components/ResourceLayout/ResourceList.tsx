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
import { AnimatePresence, motion } from "framer-motion";
import React from "react";
import {
  Link,
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
import { ChevronDown, ExternalLink, Pencil, Plus } from "lucide-react";
import DeleteResource from "../DeleteResource";
import TimeAgo from "../TimeAgo";
import ResourceInfo, { ResourceVisibilityButtons } from "./ResourceInfo";
import { parseQueryString } from "./queryParse";

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

  const [showDetails, setShowDetails] = React.useState(false);
  const hoverTimerRef = React.useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );

  const handleMouseEnter = () => {
    hoverTimerRef.current = setTimeout(() => setShowDetails(true), 120);
  };

  const handleMouseLeave = () => {
    if (hoverTimerRef.current) {
      clearTimeout(hoverTimerRef.current);
      hoverTimerRef.current = null;
    }
    setShowDetails(false);
  };

  return (
    <div
      className="w-full font-semibold"
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
    >
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
                leftSection={<Pencil size={11} />}
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
              >
                Edit
              </Button>
            )}

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

            <div className="flex flex-col items-center gap-1">
              <button
                className="flex items-center justify-center w-6 h-6 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150"
                onClick={(e) => {
                  e.stopPropagation();
                  setShowDetails((v) => !v);
                }}
                title={showDetails ? "Collapse" : "Expand"}
              >
                <motion.span
                  animate={{ rotate: showDetails ? 180 : 0 }}
                  transition={{ duration: 0.2 }}
                  className="flex items-center"
                >
                  <ChevronDown size={14} />
                </motion.span>
              </button>

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

          <AnimatePresence initial={false}>
            {showDetails && (
              <motion.div
                key="details"
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.22, ease: "easeInOut" }}
                className="overflow-hidden"
              >
                <div className="mt-4 pt-3 border-t border-slate-100">
                  <ItemExtra item={item} info={props.info} />
                </div>
              </motion.div>
            )}
          </AnimatePresence>
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
            <ResourceListItem>
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

const ResourceListPage = (props: { info: ResourceComponentInfo }) => {
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

export default ResourceListPage;
