import { getDomain } from "@/utils";
import {
  getAPIKindFromPath,
  getAPIFromAPIVersion,
  getClientResourceList,
  getClientResourceListP,
  getListKeyFromPath,
  getPBResourceListFromAPI,
  getRefNameQueryArgStr,
  getResourcePath,
  getResourcePathFromAPIKind,
  hasAccessLog,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  Resource,
  ResourceList,
  ResourceName,
} from "@/utils/pb";
import { ActionIcon, Button, Loader, Menu, Tooltip } from "@mantine/core";
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
import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import { getServicePublicURL } from "@/utils/octelium";
import {
  Copy,
  ExternalLink,
  FileText,
  Library,
  MoreVertical,
  Pencil,
  Plus,
  SearchX,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Trash2,
} from "lucide-react";
import DeleteResource from "../DeleteResource";
import TimeAgo from "../TimeAgo";
import CloneResource from "./CloneResource";
import { parseQueryString } from "./queryParse";

const ResourceItemActions = (props: {
  item: Resource;
  info: ResourceComponentInfo;
  returnTo: string;
}) => {
  const { item, info, returnTo } = props;
  const md = item.metadata!;
  const [yamlOpened, setYamlOpened] = React.useState(false);
  const [cloneOpened, setCloneOpened] = React.useState(false);
  const [deleteOpened, setDeleteOpened] = React.useState(false);
  const publicURL =
    item.apiVersion === "core/v1" &&
    item.kind === "Service" &&
    (item as Service).spec?.isPublic &&
    (item as Service).spec?.mode === Service_Spec_Mode.WEB
      ? getServicePublicURL(item as Service, getDomain())
      : undefined;
  const query = getRefNameQueryArgStr(item);
  const visibilityItems = [
    {
      show: hasAccessLog(item),
      to: `/visibility/accesslogs?${query}`,
      icon: ShieldEllipsis,
      label: "Access logs",
    },
    {
      show: hasAuthenticationLog(item),
      to: `/visibility/authenticationlogs?${query}`,
      icon: ShieldUser,
      label: "Authentication logs",
    },
    {
      show: hasAuditLog(item),
      to: `/visibility/auditlogs?${query}`,
      icon: Library,
      label: "Audit logs",
    },
    {
      show: hasSSHSessionLog(item),
      to: `/visibility/ssh?${query}`,
      icon: SquareTerminal,
      label: "SSH sessions",
    },
  ].filter(({ show }) => show);

  return (
    <>
      <Menu
        position="bottom-end"
        width={230}
        shadow="md"
        withinPortal
        transitionProps={{ transition: "pop-top-right", duration: 180 }}
        styles={{ item: { fontWeight: 600 } }}
      >
        <Menu.Target>
          <ActionIcon
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={`Actions for ${md.name}`}
            onClick={(event) => event.stopPropagation()}
          >
            <MoreVertical size={16} strokeWidth={2.25} />
          </ActionIcon>
        </Menu.Target>

        <Menu.Dropdown onClick={(event) => event.stopPropagation()}>
          <Menu.Label className="truncate">{md.name}</Menu.Label>
          <Menu.Item
            leftSection={<FileText size={14} />}
            onClick={() => setYamlOpened(true)}
          >
            View YAML
          </Menu.Item>
          {!info.unEditable && !md.isSystem && (
            <Menu.Item
              component={Link}
              to={`${getResourcePath(item)}/edit`}
              state={{ returnTo }}
              preventScrollReset
              leftSection={<Pencil size={14} />}
            >
              Edit
            </Menu.Item>
          )}
          {info.cloneable && (
            <Menu.Item
              leftSection={<Copy size={14} />}
              onClick={() => setCloneOpened(true)}
            >
              Clone
            </Menu.Item>
          )}
          {publicURL && (
            <Menu.Item
              component="a"
              href={publicURL}
              target="_blank"
              rel="noopener noreferrer"
              leftSection={<ExternalLink size={14} />}
            >
              Visit public service
            </Menu.Item>
          )}

          {visibilityItems.length > 0 && <Menu.Divider />}
          {visibilityItems.map(({ to, icon: Icon, label }) => (
            <Menu.Item
              key={to}
              component={Link}
              to={to}
              preventScrollReset
              leftSection={<Icon size={14} />}
            >
              {label}
            </Menu.Item>
          ))}

          {!info.unDeletable && !md.isSystem && (
            <>
              <Menu.Divider />
              <Menu.Item
                color="red"
                leftSection={<Trash2 size={14} />}
                onClick={() => setDeleteOpened(true)}
              >
                Delete
              </Menu.Item>
            </>
          )}
        </Menu.Dropdown>
      </Menu>

      <ResourceYAML
        item={item}
        hideTrigger
        opened={yamlOpened}
        onClose={() => setYamlOpened(false)}
      />
      {info.cloneable && (
        <CloneResource
          item={item}
          hideTrigger
          opened={cloneOpened}
          onClose={() => setCloneOpened(false)}
        />
      )}
      {!info.unDeletable && !md.isSystem && (
        <DeleteResource
          item={item}
          doNotNavigateAfter
          hideTrigger
          opened={deleteOpened}
          onClose={() => setDeleteOpened(false)}
        />
      )}
    </>
  );
};

const Item = (props: { item: Resource; info: ResourceComponentInfo }) => {
  const { item } = props;
  const md = item.metadata!;
  const location = useLocation();
  const returnTo = `${location.pathname}${location.search}`;

  return (
    <div className="w-full font-semibold">
      <div className="flex items-start gap-3 sm:gap-4">
        {md.picURL && (
          <img
            src={md.picURL}
            alt={md.displayName || md.name}
            loading="lazy"
            className="mt-0.5 h-10 w-10 shrink-0 rounded-lg border border-slate-200 object-cover shadow-sm"
          />
        )}

        <div className="flex min-w-0 flex-1 items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-0.5">
            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
              <span className="min-w-0 text-[0.92rem] font-bold text-slate-800">
                <CopyText value={md.name} />
              </span>

              {md.displayName && (
                <span className="truncate text-[0.82rem] font-semibold text-slate-500">
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
              <p className="mt-0.5 line-clamp-2 max-w-2xl text-[0.78rem] font-medium leading-5 text-slate-500">
                {md.description}
              </p>
            )}

            <div className="mt-1 text-[0.7rem] font-medium text-slate-400">
              Created <TimeAgo rfc3339={md.createdAt} />
              {md.updatedAt && (
                <span className="ml-2">
                  · Updated <TimeAgo rfc3339={md.updatedAt} />
                </span>
              )}
            </div>
          </div>

          <ResourceItemActions
            item={item}
            info={props.info}
            returnTo={returnTo}
          />
        </div>
      </div>

      {props.info.List.labelComponent && (
        <div className="w-full">
          {props.info.List.labelComponent({ item })}
        </div>
      )}
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
  const api = getAPIFromAPIVersion(props.itemsList.apiVersion);
  const collectionName = api
    ? getResourcePathFromAPIKind({
        api,
        kind: kindName as ResourceName,
      })
    : undefined;
  const countLabel =
    totalCount === 1
      ? kindName.toLowerCase()
      : collectionName || `${kindName.toLowerCase()}s`;

  return (
    <div className="w-full">
      <div className="flex items-center justify-between mb-4">
        {totalCount > 0 ? (
          <span className="text-[0.72rem] font-semibold text-slate-400 tracking-wide">
            {totalCount.toLocaleString()} {countLabel}
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
        <div className="flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-slate-200 bg-slate-50/50 px-6 py-14 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-slate-200 bg-white text-slate-400 shadow-sm">
            <SearchX size={20} strokeWidth={1.9} />
          </div>
          <h2 className="mt-4 text-sm font-bold text-slate-700">
            No {kindName} resources found
          </h2>
          <p className="mt-1.5 max-w-sm text-[0.75rem] font-medium leading-5 text-slate-400">
            There are no resources to display with the current filters or page.
          </p>
          {!props.info.unCreatable && (
            <Button
              className="mt-5"
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

const ResourceListLoading = () => (
  <motion.div
    role="status"
    aria-live="polite"
    initial={{ opacity: 0 }}
    animate={{ opacity: 1 }}
    transition={{ duration: 0.5 }}
    className="flex min-h-72 w-full flex-col items-center justify-center gap-3"
  >
    <motion.div
      animate={{ opacity: [0.45, 1, 0.45], scale: [0.96, 1, 0.96] }}
      transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
      className="flex h-12 w-12 items-center justify-center rounded-full border border-slate-200 bg-white shadow-sm"
    >
      <Loader size={22} color="dark" type="oval" />
    </motion.div>
    <span className="text-[0.7rem] font-semibold tracking-wide text-slate-400">
      Loading resources…
    </span>
  </motion.div>
);

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

  if (!data || isLoading) return <ResourceListLoading />;

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
