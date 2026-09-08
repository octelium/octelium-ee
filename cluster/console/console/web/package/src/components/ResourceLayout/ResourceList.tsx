import { getDomain, onError } from "@/utils";
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
import { keepPreviousData, useMutation, useQuery } from "@tanstack/react-query";
import React from "react";
import {
  Link,
  Outlet,
  useLocation,
  useNavigate,
  useSearchParams,
} from "react-router-dom";
import { toast } from "sonner";
import { twMerge } from "tailwind-merge";
import CopyText from "../CopyText";
import Paginator from "../Paginator";
import { ResourceListItem, ResourceListWrapper } from "../ResourceList";
import ResourceYAML from "../ResourceYAML";

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
  SearchX,
  ShieldAlert,
  ShieldEllipsis,
  ShieldUser,
  SquareTerminal,
  Trash2,
  X,
} from "lucide-react";
import DeleteResource from "../DeleteResource";
import TimeAgo from "../TimeAgo";
import CloneResource from "./CloneResource";
import { parseQueryString } from "./queryParse";
import ResourceListToolbar, {
  ListDensity,
  useListDensity,
  useListParams,
} from "./ResourceListToolbar";

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
          >
            <MoreVertical size={16} strokeWidth={2.25} />
          </ActionIcon>
        </Menu.Target>

        <Menu.Dropdown>
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
    to={getResourcePath(props.item)}
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
              <RowTimestamp item={item} mode="created" />
              {md.updatedAt && md.updatedAt !== md.createdAt && (
                <RowTimestamp item={item} mode="updated" />
              )}
            </div>
          )}
        </div>

        <div className="relative z-10 shrink-0">
          <ResourceItemActions
            item={item}
            info={info}
            returnTo={props.returnTo}
          />
        </div>
      </div>

      {Labels && (
        <div className="relative z-10 min-h-[26px] w-full">
          <Labels item={item} />
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
}) => {
  const { info } = props;
  const Labels = info.List.labelComponent;

  return (
    <div className="w-full overflow-x-auto rounded-xl border border-slate-200 bg-white">
      <table className="w-full min-w-[720px] table-fixed border-collapse text-left">
        <thead>
          <tr className="border-b border-slate-200 bg-slate-50/70">
            <th scope="col" className="w-10 px-3 py-2.5">
              <Checkbox
                size="xs"
                aria-label="Select all"
                checked={props.allSelected}
                indeterminate={props.someSelected && !props.allSelected}
                onChange={props.onToggleAll}
              />
            </th>
            <th
              scope="col"
              className="w-72 px-3 py-2.5 text-xs font-normal text-slate-600"
            >
              Name
            </th>
            {Labels && (
              <th
                scope="col"
                className="px-3 py-2.5 text-xs font-normal text-slate-600"
              >
                Details
              </th>
            )}
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
                  <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-0.5">
                    <RowTimestamp item={item} mode="created" />
                    {md.updatedAt && md.updatedAt !== md.createdAt && (
                      <RowTimestamp item={item} mode="updated" />
                    )}
                  </div>
                </td>
                {Labels && (
                  <td className="px-3 py-2 align-top [&>div]:mt-0">
                    <Labels item={item} />
                  </td>
                )}
                <td className="px-3 py-2.5 align-top">
                  <ResourceItemActions
                    item={item}
                    info={info}
                    returnTo={props.returnTo}
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
          border: "1px solid #e2e8f0",
          borderRadius: "14px",
          boxShadow: "0 24px 64px rgba(15,23,42,0.18)",
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
                  color: "#334155",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                },
                description: {
                  color: "#94a3b8",
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
            {props.isPending ? "Deleting…" : `Delete ${count} ${kindLabel}`}
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

const useListReq = () => {
  const [searchParams] = useSearchParams();
  const searchParamsStr = searchParams.toString();
  const loc = useLocation();

  const apiKind = getAPIKindFromPath(loc.pathname);
  if (!apiKind) return undefined;

  const optionsPB = getListOptionsPB(apiKind.api, apiKind.kind);
  const req = optionsPB["create"]({ common: CommonListOptions.create({}) });

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

    const req2 = optionsPB["fromJsonString"](JSON.stringify(parsedQry));
    optionsPB["mergePartial"](req, req2);
  }

  if (!req.common!.itemsPerPage) {
    req.common!.itemsPerPage = 25;
  }

  return req;
};

const ResourceListContent = (props: { info: ResourceComponentInfo }) => {
  const loc = useLocation();
  const navigate = useNavigate();
  const [density, setDensity] = useListDensity();
  const [selected, setSelected] = React.useState<Set<string>>(new Set());
  const [deleteModalOpened, setDeleteModalOpened] = React.useState(false);
  const { searchParams, activeFilterCount, clearFilters } = useListParams();

  const apiKind = getAPIKindFromPath(loc.pathname);
  const req = useListReq();
  const listKey = getListKeyFromPath(loc.pathname);

  const { isLoading, isFetching, data } = useQuery({
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
  }, [loc.pathname, loc.search]);

  const mutationBulkDelete = useMutation({
    mutationFn: async (targets: Resource[]) => {
      for (const target of targets) {
        await deleteResourcePB(target);
      }
      return targets.length;
    },
    onSuccess: (count, targets) => {
      if (targets[0]) invalidateResourceList(targets[0]);
      setSelected(new Set());
      setDeleteModalOpened(false);
      toast.success(
        `${count} ${count === 1 ? "resource" : "resources"} deleted`,
      );
    },
    onError: (err: unknown) => onError(err as any),
  });

  if (!apiKind) return null;

  const kindName = apiKind.kind as string;
  const collectionName = getResourcePathFromAPIKind({
    api: apiKind.api,
    kind: apiKind.kind as ResourceName,
  });
  const totalCount = itemList?.listResponseMeta?.totalCount ?? 0;
  const itemsPerPage =
    Number(searchParams.get("common.itemsPerPage")) ||
    itemList?.listResponseMeta?.itemsPerPage ||
    25;
  const countLabel =
    totalCount === 1
      ? kindName.toLowerCase()
      : collectionName || `${kindName.toLowerCase()}s`;
  const isPending = isLoading || !itemList;

  const returnTo = `${loc.pathname}${loc.search}`;
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
        density={density}
        onDensityChange={setDensity}
        canCreate={!props.info.unCreatable}
        onCreate={() => navigate("create")}
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
        onConfirm={() => mutationBulkDelete.mutate(selectedItems)}
      />

      {Summary && (
        <div className="mb-6">
          <React.Suspense key={listKey} fallback={null}>
            <Summary />
          </React.Suspense>
        </div>
      )}

      {isPending ? (
        <ListSkeleton density={density} />
      ) : items.length === 0 ? (
        <EmptyState
          kindName={kindName}
          filtered={activeFilterCount > 0}
          canCreate={!props.info.unCreatable}
          onCreate={() => navigate("create")}
          onClearFilters={clearFilters}
        />
      ) : density === "table" ? (
        <React.Suspense key={listKey} fallback={<ListSkeleton density={density} />}>
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
          />
        </React.Suspense>
      ) : (
        <React.Suspense key={listKey} fallback={<ListSkeleton density={density} />}>
          <ResourceListWrapper>
            {items.map((item) => (
              <ResourceListItem
                key={item.metadata!.uid}
                path={getResourcePath(item)}
                compact={density === "compact"}
              >
                <CardItem
                  item={item}
                  info={props.info}
                  returnTo={returnTo}
                  compact={density === "compact"}
                  checked={selected.has(item.metadata!.uid)}
                  onToggle={toggle}
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
