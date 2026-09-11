import { getDomain } from "@/utils";
import {
  getAPIKindFromPath,
  deleteResourcePB,
  getListKeyFromPath,
  getListOptionsPB,
  getRefNameQueryArgStr,
  getResourcePath,
  listResourcesPB,
  getResourcePathFromAPIKind,
  hasAccessLog,
  getAuditLogQueryArgStr,
  hasAuditLog,
  hasAuthenticationLog,
  hasSSHSessionLog,
  invalidateResourceList,
  Resource,
  ResourceList,
  ResourceName,
} from "@/utils/pb";
import {
  ActionIcon,
  Button,
  Checkbox,
  Menu,
  Modal,
  Switch,
  Tooltip,
} from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";
import React from "react";
import {
  Link,
  Outlet,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import {
  CompactResourceListLabels,
  ResourceListItem,
  ResourceListWrapper,
} from "../ResourceList";

import { ResourceComponentInfo } from "@/pages/utils/types";
import { Service, Service_Spec_Mode } from "@/apis/corev1/corev1";
import { CommonListOptions } from "@/apis/metav1/metav1";
import { getServicePublicURL } from "@/utils/octelium";
import {
  AlertTriangle,
  Copy,
  ExternalLink,
  FileText,
  FilterX,
  Library,
  Loader2,
  MoreVertical,
  Pencil,
  Plus,
  RefreshCw,
  SearchX,
  ShieldAlert,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Trash2,
  X,
} from "lucide-react";
import TimeAgo from "../TimeAgo";
import { CompactSummary } from "../Summary";
import DeleteResource from "../DeleteResource";
import { parseQueryString } from "./queryParse";
import CloneResource from "./CloneResource";
import ResourceListToolbar, {
  ListDensity,
  useListDensity,
  useListParams,
} from "./ResourceListToolbar";

const LazyResourceYAML = React.lazy(() => import("../ResourceYAML"));

type ResourceItemAction = {
  type: "yaml" | "clone" | "delete";
  item: Resource;
};

const searchFromReturnTo = (returnTo: string) => {
  const queryIndex = returnTo.indexOf("?");
  return queryIndex >= 0 ? returnTo.slice(queryIndex) : "";
};

const ResourceItemActions = (props: {
  item: Resource;
  info: ResourceComponentInfo;
  returnTo: string;
  onAction: (action: ResourceItemAction) => void;
}) => {
  const { item, info, returnTo, onAction } = props;
  const md = item.metadata!;
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
      show: true,
      to: `/visibility/auditlogs?${getAuditLogQueryArgStr(item)}`,
      icon: Library,
      label: hasAuditLog(item) ? "Audit logs" : "Change history",
    },
    {
      show: hasSSHSessionLog(item),
      to: `/visibility/ssh?${query}`,
      icon: SquareTerminal,
      label: "SSH sessions",
    },
  ].filter(({ show }) => show);

  return (
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
          >
            <MoreVertical size={16} strokeWidth={2.25} />
          </ActionIcon>
        </Menu.Target>

        <Menu.Dropdown>
          <Menu.Label className="truncate">{md.name}</Menu.Label>
          <Menu.Item
            leftSection={<FileText size={14} />}
            onClick={() => onAction({ type: "yaml", item })}
          >
            View YAML
          </Menu.Item>
          {!info.unEditable && !md.isSystem && (
            <Menu.Item
              component={Link}
              to={`${getResourcePath(item)}/edit${searchFromReturnTo(returnTo)}`}
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
              onClick={() => onAction({ type: "clone", item })}
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
                onClick={() => onAction({ type: "delete", item })}
              >
                Delete
              </Menu.Item>
            </>
          )}
        </Menu.Dropdown>
    </Menu>
  );
};

const isSelectable = (item: Resource, info: ResourceComponentInfo) =>
  !item.metadata?.isSystem && !info.unDeletable;

const SystemBadge = () => (
  <Tooltip label="This is a system resource created by the cluster" withArrow>
    <span className="inline-flex items-center rounded border border-blue-200 bg-blue-50 px-1.5 py-px text-xs font-medium leading-4 text-blue-700">
      System
    </span>
  </Tooltip>
);

const RowTimestamp = (props: {
  item: Resource;
  className?: string;
  mode?: "auto" | "created" | "updated";
}) => {
  const md = props.item.metadata!;
  const mode = props.mode ?? "auto";
  const hasUpdate = !!md.updatedAt && md.updatedAt !== md.createdAt;
  const showUpdated = mode === "updated" || (mode === "auto" && hasUpdate);
  return (
    <span
      className={twMerge(
        "relative z-10 inline-flex items-center gap-1 whitespace-nowrap text-xs font-normal text-slate-600",
        props.className,
      )}
    >
      {showUpdated ? "Updated" : "Created"}
      <TimeAgo rfc3339={showUpdated ? md.updatedAt : md.createdAt} />
    </span>
  );
};

const RowTitleLink = (props: {
  item: Resource;
  returnTo: string;
  stretched?: boolean;
}) => (
  <Link
    to={`${getResourcePath(props.item)}${searchFromReturnTo(props.returnTo)}`}
    state={{ returnTo: props.returnTo }}
    preventScrollReset
    className={twMerge(
      "min-w-0 truncate rounded text-sm font-semibold text-slate-900",
      "outline-none transition-colors hover:text-black focus-visible:ring-2 focus-visible:ring-blue-500/40",
      props.stretched && "after:absolute after:inset-0 after:content-['']",
    )}
  >
    {props.item.metadata!.name}
  </Link>
);

const SelectBox = (props: {
  item: Resource;
  info: ResourceComponentInfo;
  checked: boolean;
  onToggle: (uid: string) => void;
}) => {
  if (!isSelectable(props.item, props.info)) {
    return <span className="inline-block w-[18px]" aria-hidden="true" />;
  }
  return (
    <Checkbox
      size="xs"
      className="relative z-10"
      checked={props.checked}
      aria-label={`Select ${props.item.metadata!.name}`}
      onChange={() => props.onToggle(props.item.metadata!.uid)}
    />
  );
};

const CardItem = (props: {
  item: Resource;
  info: ResourceComponentInfo;
  returnTo: string;
  compact?: boolean;
  checked: boolean;
  onToggle: (uid: string) => void;
  onAction: (action: ResourceItemAction) => void;
}) => {
  const { item, info, compact } = props;
  const md = item.metadata!;
  const Labels = info.List.labelComponent;

  return (
    <div className="w-full">
      <div className="flex items-start gap-3">
        <div className="pt-0.5">
          <SelectBox
            item={item}
            info={info}
            checked={props.checked}
            onToggle={props.onToggle}
          />
        </div>

        {md.picURL && !compact && (
          <img
            src={md.picURL}
            alt=""
            loading="lazy"
            className="mt-0.5 h-10 w-10 shrink-0 rounded-lg border border-slate-200 object-cover"
          />
        )}

        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex min-w-0 items-center gap-2">
            <RowTitleLink item={item} returnTo={props.returnTo} stretched />
            <span className="relative z-10 shrink-0">
              <CopyText value={md.name} hide />
            </span>
            {md.displayName && (
              <span className="truncate text-sm font-normal text-slate-600">
                {md.displayName}
              </span>
            )}
            {md.isSystem && <SystemBadge />}
            {compact && (
              <>
                <div className="flex-1" />
                <RowTimestamp item={item} />
              </>
            )}
          </div>

          {md.description && !compact && (
            <p className="line-clamp-2 max-w-2xl text-sm font-normal leading-5 text-slate-600">
              {md.description}
            </p>
          )}

          {!compact && (
            <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5">
              <RowTimestamp item={item} />
            </div>
          )}
        </div>

        <div className="relative z-10 shrink-0">
          <ResourceItemActions
            item={item}
            info={info}
            returnTo={props.returnTo}
            onAction={props.onAction}
          />
        </div>
      </div>

      {Labels && (
        <div className="relative z-10 min-h-[26px] w-full">
          {compact ? (
            <CompactResourceListLabels>
              <Labels item={item} />
            </CompactResourceListLabels>
          ) : (
            <Labels item={item} />
          )}
        </div>
      )}
    </div>
  );
};

const TableView = (props: {
  items: Resource[];
  info: ResourceComponentInfo;
  returnTo: string;
  selected: Set<string>;
  onToggle: (uid: string) => void;
  onToggleAll: () => void;
  allSelected: boolean;
  someSelected: boolean;
  onAction: (action: ResourceItemAction) => void;
}) => {
  const { info } = props;
  const Labels = info.List.labelComponent;

  return (
    <div className="w-full overflow-x-auto rounded-xl border border-slate-200 bg-white">
      <table className="w-full min-w-[760px] table-fixed border-collapse text-left">
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50/70">
            <th scope="col" className="w-10 px-3 py-2.5">
              <Checkbox
                size="xs"
                aria-label="Select all deletable resources on this page"
                checked={props.allSelected}
                indeterminate={props.someSelected && !props.allSelected}
                onChange={props.onToggleAll}
              />
            </th>
            <th
              scope="col"
              className="w-72 px-3 py-2.5 text-xs font-semibold text-slate-600"
            >
              Resource
            </th>
            {Labels && (
              <th
                scope="col"
                className="px-3 py-2.5 text-xs font-semibold text-slate-600"
              >
                Properties
              </th>
            )}
            <th
              scope="col"
              className="w-36 px-3 py-2.5 text-xs font-semibold text-slate-600"
            >
              Last modified
            </th>
            <th scope="col" className="w-12 px-3 py-2.5">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {props.items.map((item) => {
            const md = item.metadata!;
            return (
              <tr
                key={md.uid}
                className="border-b border-slate-100 last:border-b-0 hover:bg-slate-50/70"
              >
                <td className="px-3 py-2.5 align-top">
                  <SelectBox
                    item={item}
                    info={info}
                    checked={props.selected.has(md.uid)}
                    onToggle={props.onToggle}
                  />
                </td>
                <td className="px-3 py-2.5 align-top">
                  <div className="flex min-w-0 items-center gap-2">
                    <RowTitleLink item={item} returnTo={props.returnTo} />
                    <CopyText value={md.name} hide />
                    {md.isSystem && <SystemBadge />}
                  </div>
                  {md.displayName && (
                    <div className="truncate text-xs font-normal text-slate-600">
                      {md.displayName}
                    </div>
                  )}
                </td>
                {Labels && (
                  <td className="px-3 py-2 align-top [&>div]:mt-0">
                    <CompactResourceListLabels>
                      <Labels item={item} />
                    </CompactResourceListLabels>
                  </td>
                )}
                <td className="px-3 py-2.5 align-top">
                  <RowTimestamp item={item} />
                </td>
                <td className="px-3 py-2.5 align-top">
                  <ResourceItemActions
                    item={item}
                    info={info}
                    returnTo={props.returnTo}
                    onAction={props.onAction}
                  />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

const EmptyState = (props: {
  kindName: string;
  filtered: boolean;
  canCreate: boolean;
  onCreate: () => void;
  onClearFilters: () => void;
}) => (
  <div className="flex min-h-72 flex-col items-center justify-center rounded-2xl border border-dashed border-slate-300 bg-white/60 px-6 py-14 text-center">
    <div className="flex h-12 w-12 items-center justify-center rounded-2xl border border-slate-200 bg-white text-slate-500">
      {props.filtered ? (
        <FilterX size={20} strokeWidth={1.9} />
      ) : (
        <SearchX size={20} strokeWidth={1.9} />
      )}
    </div>
    <h2 className="mt-4 text-sm font-semibold text-slate-800">
      {props.filtered
        ? "No results match your filters"
        : `No ${props.kindName} resources yet`}
    </h2>
    <p className="mt-1.5 max-w-sm text-sm font-normal leading-5 text-slate-600">
      {props.filtered
        ? "Try a different search term, or clear the active filters to see everything."
        : `Create the first ${props.kindName} to get started.`}
    </p>
    {props.filtered ? (
      <button
        type="button"
        onClick={props.onClearFilters}
        className="mt-5 inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm font-semibold text-slate-700 hover:border-slate-300 hover:bg-slate-50"
      >
        <FilterX size={14} strokeWidth={2.2} />
        Clear filters
      </button>
    ) : (
      props.canCreate && (
        <button
          type="button"
          onClick={props.onCreate}
          className="mt-5 inline-flex items-center gap-1.5 rounded-lg bg-slate-900 px-3 py-1.5 text-sm !font-bold text-white shadow-[0_8px_20px_-6px_rgba(15,23,42,0.35)] hover:bg-slate-800"
        >
          <Plus size={14} strokeWidth={2.2} />
          Create {props.kindName}
        </button>
      )
    )}
  </div>
);

const BulkDeleteModal = (props: {
  opened: boolean;
  onClose: () => void;
  items: Resource[];
  kindName: string;
  isPending: boolean;
  progress?: { completed: number; total: number };
  onConfirm: () => void;
}) => {
  const [isConfirmed, setIsConfirmed] = React.useState(false);
  const count = props.items.length;
  const kindLabel = `${props.kindName}${count === 1 ? "" : "s"}`;

  React.useEffect(() => {
    if (!props.opened) setIsConfirmed(false);
  }, [props.opened]);

  const handleClose = () => {
    if (props.isPending) return;
    setIsConfirmed(false);
    props.onClose();
  };

  return (
    <Modal
      opened={props.opened}
      onClose={handleClose}
      centered
      size="md"
      withCloseButton={false}
      padding={0}
      closeOnClickOutside={!props.isPending}
      closeOnEscape={!props.isPending}
      overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
      transitionProps={{ transition: "pop", duration: 250 }}
      styles={{
        content: {
          border: "1px solid var(--color-slate-200)",
          borderRadius: "14px",
          boxShadow: "var(--shadow-modal)",
          overflow: "hidden",
        },
      }}
    >
      <div className="flex flex-col">
        <header className="flex items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-red-600 text-white shadow-sm">
              <Trash2 size={16} strokeWidth={2.25} />
            </span>
            <div className="min-w-0">
              <h2 className="truncate text-sm font-bold text-slate-900">
                Delete {count} {kindLabel}
              </h2>
              <p className="mt-0.5 text-micro font-normal text-slate-500">
                Permanent destructive action
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-xs"
            disabled={props.isPending}
            leftSection={<X size={12} strokeWidth={2.5} />}
            onClick={handleClose}
          >
            Close
          </Button>
        </header>

        <div className="space-y-4 px-4 py-4 sm:px-5">
          <div className="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50/70 px-3.5 py-3">
            <AlertTriangle
              size={16}
              className="mt-0.5 shrink-0 text-red-600"
              strokeWidth={2.25}
            />
            <div>
              <p className="text-body font-semibold text-red-800">
                This action cannot be undone
              </p>
              <p className="mt-1 text-xs font-semibold leading-relaxed text-red-700/80">
                {count} {count === 1 ? "resource" : "resources"} will be
                permanently removed from the cluster. Review the list below
                carefully before continuing.
              </p>
            </div>
          </div>

          <section className="max-h-48 overflow-y-auto rounded-xl border border-slate-200 bg-white">
            {props.items.map((item) => (
              <div
                key={item.metadata!.uid}
                className="flex items-center gap-2 border-b border-slate-100 px-3.5 py-2 last:border-b-0"
              >
                <ShieldAlert
                  size={13}
                  className="shrink-0 text-slate-400"
                  strokeWidth={2.25}
                />
                <span className="min-w-0 truncate text-xs font-semibold text-slate-700">
                  {item.metadata!.name}
                </span>
              </div>
            ))}
          </section>

          <div className="rounded-xl border border-slate-200 bg-slate-50/60 px-3.5 py-3">
            <Switch
              autoFocus
              checked={isConfirmed}
              disabled={props.isPending}
              color="red.8"
              size="sm"
              label="I understand that this action is permanent"
              description={`Confirm deletion of ${count} ${kindLabel.toLowerCase()}`}
              onChange={(event) => setIsConfirmed(event.currentTarget.checked)}
              styles={{
                label: {
                  color: "var(--color-slate-700)",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                },
                description: {
                  color: "var(--color-slate-400)",
                  fontSize: "0.67rem",
                  fontWeight: 600,
                  marginTop: 2,
                },
              }}
            />
          </div>
        </div>

        <footer className="flex items-center justify-end gap-2 border-t border-slate-200 bg-slate-50/70 px-4 py-3 sm:px-5">
          <Button
            type="button"
            variant="default"
            disabled={props.isPending}
            onClick={handleClose}
          >
            Cancel
          </Button>
          <Button
            type="button"
            color="red.8"
            disabled={!isConfirmed || props.isPending}
            loading={props.isPending}
            leftSection={
              props.isPending ? (
                <Loader2 size={13} className="animate-spin" />
              ) : (
                <Trash2 size={13} strokeWidth={2.25} />
              )
            }
            onClick={props.onConfirm}
          >
            {props.isPending
              ? `Deleting ${props.progress?.completed ?? 0}/${props.progress?.total ?? count}…`
              : `Delete ${count} ${kindLabel}`}
          </Button>
        </footer>
      </div>
    </Modal>
  );
};

const ListSkeleton = (props: { density: ListDensity }) => {
  const rows = Array.from({ length: 6 });

  if (props.density === "table") {
    return (
      <div className="w-full animate-pulse overflow-hidden rounded-xl border border-slate-200 bg-white">
        <div className="h-9 border-b border-slate-200 bg-slate-50/70" />
        {rows.map((_, index) => (
          <div
            key={index}
            className="flex items-center gap-4 border-b border-slate-100 px-3 py-3 last:border-b-0"
          >
            <div className="h-3.5 w-3.5 rounded bg-slate-200" />
            <div className="h-3 w-44 rounded bg-slate-200" />
            <div className="h-3 flex-1 rounded bg-slate-100" />
            <div className="h-3 w-24 rounded bg-slate-100" />
          </div>
        ))}
      </div>
    );
  }

  return (
    <div
      className="flex w-full animate-pulse flex-col gap-3"
      aria-hidden="true"
    >
      {rows.map((_, index) => (
        <div
          key={index}
          className={twMerge(
            "w-full rounded-xl border border-slate-200 bg-white px-4 py-3.5 sm:px-5 sm:py-4",
            props.density === "compact" ? "h-[52px]" : "h-[104px]",
          )}
        >
          <div className="h-3 w-48 rounded bg-slate-200" />
          {props.density !== "compact" && (
            <>
              <div className="mt-3 h-3 w-full max-w-md rounded bg-slate-100" />
              <div className="mt-3 h-3 w-32 rounded bg-slate-100" />
            </>
          )}
        </div>
      ))}
    </div>
  );
};

const buildListReq = (pathname: string, search: string) => {
  const apiKind = getAPIKindFromPath(pathname);
  if (!apiKind) return { apiKind: undefined, req: undefined };

  const optionsPB = getListOptionsPB(apiKind.api, apiKind.kind);
  const req = optionsPB["create"]({ common: CommonListOptions.create({}) });
  const searchParams = new URLSearchParams(search);
  let requestWarning: string | undefined;

  const page = Number(searchParams.get("common.page"));
  if (searchParams.has("common.page")) {
    if (!Number.isFinite(page) || page < 1) {
      searchParams.delete("common.page");
      requestWarning = "The page parameter was invalid and has been ignored.";
    } else {
      searchParams.set("common.page", `${Math.min(Math.floor(page), 100000)}`);
    }
  }

  const pageSize = Number(searchParams.get("common.itemsPerPage"));
  if (
    searchParams.has("common.itemsPerPage") &&
    ![10, 25, 50, 100].includes(pageSize)
  ) {
    searchParams.set("common.itemsPerPage", "25");
    requestWarning = "The page size was invalid and has been reset to 25.";
  }

  const orderType = searchParams.get("common.orderBy.type");
  if (orderType && orderType !== "NAME" && orderType !== "CREATED_AT") {
    searchParams.delete("common.orderBy.type");
    requestWarning = "The sort field was invalid and has been ignored.";
  }
  const orderMode = searchParams.get("common.orderBy.mode");
  if (orderMode && orderMode !== "ASC" && orderMode !== "DESC") {
    searchParams.delete("common.orderBy.mode");
    requestWarning = "The sort direction was invalid and has been ignored.";
  }

  try {
    if (searchParams.size > 0) {
      const parsedQry = parseQueryString<{
        type?: string;
        mode?: string;
        common?: { page?: number; itemsPerPage?: number };
        namespaceRef?: { uid?: string; name?: string };
        userRef?: { uid?: string; name?: string };
        deviceRef?: { uid?: string; name?: string };
      }>(searchParams.toString());

      if (parsedQry.common?.page && parsedQry.common.page > 0) {
        parsedQry.common.page -= 1;
      }

      const parsedReq = optionsPB["fromJsonString"](
        JSON.stringify(parsedQry),
      );
      optionsPB["mergePartial"](req, parsedReq);
    }
  } catch {
    requestWarning =
      "Some URL filters are not valid for this resource type and were ignored.";
  }

  if (!req.common!.itemsPerPage) req.common!.itemsPerPage = 25;
  return { apiKind, req, requestWarning };
};

const ListErrorState = (props: {
  message?: string;
  onRetry: () => void;
}) => (
  <div
    role="alert"
    className="flex min-h-64 flex-col items-center justify-center rounded-2xl border border-red-200 bg-red-50/60 px-6 py-12 text-center"
  >
    <div className="flex h-11 w-11 items-center justify-center rounded-xl border border-red-200 bg-white text-red-600">
      <AlertTriangle size={19} strokeWidth={2} />
    </div>
    <h2 className="mt-4 text-sm font-semibold text-slate-900">
      Resources could not be loaded
    </h2>
    <p className="mt-1.5 max-w-md text-sm text-slate-600">
      {props.message ?? "Check your connection and try again."}
    </p>
    <Button
      className="mt-5"
      variant="default"
      leftSection={<RefreshCw size={14} />}
      onClick={props.onRetry}
    >
      Try again
    </Button>
  </div>
);

const ResourceListContent = (props: { info: ResourceComponentInfo }) => {
  const loc = useLocation();
  const navigate = useNavigate();
  const [selected, setSelected] = React.useState<Set<string>>(new Set());
  const [deleteModalOpened, setDeleteModalOpened] = React.useState(false);
  const [activeAction, setActiveAction] =
    React.useState<ResourceItemAction | null>(null);
  const [deleteProgress, setDeleteProgress] = React.useState({
    completed: 0,
    total: 0,
  });
  const { activeFilterCount, clearFilters } = useListParams();

  const pathSegments = loc.pathname.split("/").filter(Boolean);
  const listPathname = `/${pathSegments.slice(0, 2).join("/")}`;
  const isDrawerOpen = pathSegments.length > 2;
  const stateReturnTo = (loc.state as { returnTo?: unknown } | null)?.returnTo;
  let listSearch = loc.search;
  if (isDrawerOpen && typeof stateReturnTo === "string") {
    try {
      const returnURL = new URL(stateReturnTo, window.location.origin);
      if (returnURL.pathname === listPathname) listSearch = returnURL.search;
    } catch {}
  }

  const listKey = getListKeyFromPath(listPathname);
  const [density, setDensity] = useListDensity(listKey || listPathname);
  const isNarrow = useMediaQuery("(max-width: 48em)");
  const effectiveDensity =
    isNarrow && density === "table" ? "compact" : density;
  const { apiKind, req, requestWarning } = React.useMemo(
    () => buildListReq(listPathname, listSearch),
    [listPathname, listSearch],
  );

  const { isLoading, isFetching, isError, error, data, refetch } = useQuery({
    queryKey: [
      listKey,
      apiKind && req
        ? getListOptionsPB(apiKind.api, apiKind.kind)["toJsonString"](req)
        : "",
    ],
    enabled: !!apiKind && !!req,
    placeholderData: (previousData, previousQuery) =>
      previousQuery?.queryKey[0] === listKey
        ? keepPreviousData(previousData)
        : undefined,
    queryFn: async () =>
      await listResourcesPB(apiKind!.api, apiKind!.kind, req),
  });

  const itemList = data?.["response"] as ResourceList | undefined;
  const items = itemList?.items ?? [];

  React.useEffect(() => {
    setSelected(new Set());
  }, [listPathname, listSearch]);

  const mutationBulkDelete = useMutation({
    mutationFn: async (targets: Resource[]) => {
      setDeleteProgress({ completed: 0, total: targets.length });
      const failed: Resource[] = [];
      const succeeded: Resource[] = [];
      let nextIndex = 0;
      const worker = async () => {
        while (nextIndex < targets.length) {
          const target = targets[nextIndex++];
          try {
            await deleteResourcePB(target);
            succeeded.push(target);
          } catch {
            failed.push(target);
          } finally {
            setDeleteProgress((current) => ({
              ...current,
              completed: current.completed + 1,
            }));
          }
        }
      };
      await Promise.all(
        Array.from({ length: Math.min(4, targets.length) }, () => worker()),
      );
      return { succeeded, failed };
    },
    onSuccess: ({ succeeded, failed }) => {
      if (succeeded[0]) invalidateResourceList(succeeded[0]);
      setSelected(new Set(failed.map((item) => item.metadata!.uid)));
      setDeleteModalOpened(false);
      if (succeeded.length > 0) {
        toast.success(
          `${succeeded.length} ${succeeded.length === 1 ? "resource" : "resources"} deleted`,
        );
      }
      if (failed.length > 0) {
        toast.error(
          `${failed.length} ${failed.length === 1 ? "resource" : "resources"} could not be deleted and remain selected`,
        );
      }
    },
  });

  if (!apiKind) return null;

  const kindName = apiKind.kind as string;
  const collectionName = getResourcePathFromAPIKind({
    api: apiKind.api,
    kind: apiKind.kind as ResourceName,
  });
  const totalCount = itemList?.listResponseMeta?.totalCount ?? 0;
  const effectiveSearchParams = new URLSearchParams(listSearch);
  const itemsPerPage =
    Number(effectiveSearchParams.get("common.itemsPerPage")) ||
    itemList?.listResponseMeta?.itemsPerPage ||
    25;
  const countLabel =
    totalCount === 1
      ? kindName.toLowerCase()
      : collectionName || `${kindName.toLowerCase()}s`;
  const isPending = isLoading || (!itemList && !isError);

  const returnTo = `${listPathname}${listSearch}`;
  const Summary = props.info.List.SummaryComponent;
  const selectableItems = items.filter((item) =>
    isSelectable(item, props.info),
  );
  const toggle = (uid: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(uid)) next.delete(uid);
      else next.add(uid);
      return next;
    });
  const toggleAll = () =>
    setSelected((prev) =>
      prev.size === selectableItems.length
        ? new Set()
        : new Set(selectableItems.map((item) => item.metadata!.uid)),
    );
  const selectedItems = items.filter((item) => selected.has(item.metadata!.uid));

  return (
    <div className="w-full">
      <ResourceListToolbar
        kindName={kindName}
        countLabel={countLabel}
        totalCount={totalCount}
        showCount={!isPending}
        itemsPerPage={itemsPerPage}
        isFetching={isFetching}
        density={effectiveDensity}
        onDensityChange={setDensity}
        canCreate={!props.info.unCreatable}
        onCreate={() =>
          navigate(`${listPathname}/create${listSearch}`, {
            state: { returnTo },
          })
        }
        selectedCount={selected.size}
        isDeleting={mutationBulkDelete.isPending}
        onClearSelection={() => setSelected(new Set())}
        onDeleteSelection={() => setDeleteModalOpened(true)}
      />

      <BulkDeleteModal
        opened={deleteModalOpened}
        onClose={() => setDeleteModalOpened(false)}
        items={selectedItems}
        kindName={kindName}
        isPending={mutationBulkDelete.isPending}
        progress={deleteProgress}
        onConfirm={() => mutationBulkDelete.mutate(selectedItems)}
      />

      {activeAction && (
        <React.Suspense fallback={null}>
          {activeAction.type === "yaml" && (
            <LazyResourceYAML
              item={activeAction.item}
              hideTrigger
              opened
              onClose={() => setActiveAction(null)}
            />
          )}
          {activeAction.type === "clone" && (
            <CloneResource
              item={activeAction.item}
              hideTrigger
              opened
              onClose={() => setActiveAction(null)}
            />
          )}
          {activeAction.type === "delete" && (
            <DeleteResource
              item={activeAction.item}
              doNotNavigateAfter
              hideTrigger
              opened
              onClose={() => setActiveAction(null)}
            />
          )}
        </React.Suspense>
      )}

      {Summary && !isDrawerOpen && (
        <section
          aria-label="Resource summary"
          className="mb-4 min-h-[64px] rounded-xl border border-slate-200 bg-white/60 p-2"
        >
          <React.Suspense
            key={listKey}
            fallback={<div className="h-12 animate-pulse rounded-lg bg-slate-100" />}
          >
            <CompactSummary>
              <Summary />
            </CompactSummary>
          </React.Suspense>
        </section>
      )}

      {requestWarning && (
        <div
          role="status"
          className="mb-3 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900"
        >
          <AlertTriangle size={14} className="mt-px shrink-0" />
          <span>{requestWarning}</span>
        </div>
      )}

      {isError && itemList && (
        <div
          role="alert"
          className="mb-3 flex items-center gap-2 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900"
        >
          <AlertTriangle size={14} className="shrink-0" />
          <span>Refresh failed. Showing the last successfully loaded data.</span>
          <button className="ml-auto font-semibold" onClick={() => refetch()}>
            Retry
          </button>
        </div>
      )}

      {isError && !itemList ? (
        <ListErrorState
          message={error instanceof Error ? error.message : undefined}
          onRetry={() => refetch()}
        />
      ) : isPending ? (
        <ListSkeleton density={effectiveDensity} />
      ) : items.length === 0 ? (
        <EmptyState
          kindName={kindName}
          filtered={
            isDrawerOpen
              ? Array.from(effectiveSearchParams.entries()).some(
                  ([key, value]) =>
                    !!value &&
                    key !== "common.page" &&
                    key !== "common.itemsPerPage" &&
                    !key.startsWith("common.orderBy"),
                )
              : activeFilterCount > 0
          }
          canCreate={!props.info.unCreatable}
          onCreate={() =>
            navigate(`${listPathname}/create${listSearch}`, {
              state: { returnTo },
            })
          }
          onClearFilters={clearFilters}
        />
      ) : effectiveDensity === "table" ? (
        <React.Suspense key={listKey} fallback={<ListSkeleton density={effectiveDensity} />}>
          <TableView
            items={items}
            info={props.info}
            returnTo={returnTo}
            selected={selected}
            onToggle={toggle}
            onToggleAll={toggleAll}
            allSelected={
              selectableItems.length > 0 &&
              selected.size === selectableItems.length
            }
            someSelected={selected.size > 0}
            onAction={setActiveAction}
          />
        </React.Suspense>
      ) : (
        <React.Suspense key={listKey} fallback={<ListSkeleton density={effectiveDensity} />}>
          <ResourceListWrapper>
            {items.map((item) => (
              <ResourceListItem
                key={item.metadata!.uid}
                path={getResourcePath(item)}
                compact={effectiveDensity === "compact"}
              >
                <CardItem
                  item={item}
                  info={props.info}
                  returnTo={returnTo}
                  compact={effectiveDensity === "compact"}
                  checked={selected.has(item.metadata!.uid)}
                  onToggle={toggle}
                  onAction={setActiveAction}
                />
              </ResourceListItem>
            ))}
          </ResourceListWrapper>
        </React.Suspense>
      )}

      <Paginator meta={itemList?.listResponseMeta} showFilters={false} />
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
