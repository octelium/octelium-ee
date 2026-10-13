import { Button, Select, SegmentedControl, Tooltip } from "@mantine/core";
import {
  LayoutGrid,
  Loader2,
  Plus,
  Rows3,
  Table2,
  Trash2,
  X,
} from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import {
  FilterChips,
  buildFilterChips,
  countActiveListFilters,
} from "../Paginator";
import SearchList from "../SearchList";
import RefreshButton from "@/components/RefreshButton";
import { AskAgentButton } from "@/components/AgentDock/AskButtons";
import type { PinnedContext } from "@/components/AgentDock/context";
import {
  PageActions,
  PageTitleAddon,
} from "@/components/PageHeader/context";

export type ListDensity = "comfortable" | "compact" | "table";

const DENSITY_STORAGE_KEY = "octelium.console.listDensity";

const isDensity = (value: unknown): value is ListDensity =>
  value === "comfortable" || value === "compact" || value === "table";

export const useListDensity = (
  scope = "default",
): [ListDensity, (v: ListDensity) => void] => {
  const storageKey = `${DENSITY_STORAGE_KEY}.${scope}`;
  const [density, setDensity] = React.useState<ListDensity>(() => {
    try {
      const stored = window.localStorage.getItem(storageKey);
      return isDensity(stored) ? stored : "compact";
    } catch {
      return "compact";
    }
  });

  React.useEffect(() => {
    try {
      const stored = window.localStorage.getItem(storageKey);
      setDensity(isDensity(stored) ? stored : "compact");
    } catch {
      setDensity("compact");
    }
  }, [storageKey]);

  const update = React.useCallback(
    (value: ListDensity) => {
      setDensity(value);
      try {
        window.localStorage.setItem(storageKey, value);
      } catch {}
    },
    [storageKey],
  );

  return [density, update];
};

const PAGE_SIZES = ["10", "25", "50", "100"];

const formatKindName = (kindName: string) =>
  kindName.replace(/([a-z0-9])([A-Z])/g, "$1 $2");

export const useListParams = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();

  const apply = React.useCallback(
    (mutate: (next: URLSearchParams) => void) => {
      const next = new URLSearchParams(searchParams.toString());
      mutate(next);
      const search = next.toString();
      navigate(
        {
          pathname: location.pathname,
          search: search ? `?${search}` : "",
          hash: location.hash,
        },
        { state: location.state, preventScrollReset: true },
      );
    },
    [location.hash, location.pathname, location.state, navigate, searchParams],
  );

  const setParam = React.useCallback(
    (key: string, value: string) =>
      apply((next) => {
        next.set(key, value);
        if (key !== "common.page") next.delete("common.page");
      }),
    [apply],
  );

  const setParams = React.useCallback(
    (values: Record<string, string>) =>
      apply((next) => {
        for (const [key, value] of Object.entries(values)) {
          next.set(key, value);
        }
        next.delete("common.page");
      }),
    [apply],
  );

  const removeParam = React.useCallback(
    (key: string) =>
      apply((next) => {
        next.delete(key);
        next.delete("common.page");
      }),
    [apply],
  );

  const activeFilterCount = React.useMemo(
    () => countActiveListFilters(searchParams),
    [searchParams],
  );

  const clearFilters = React.useCallback(
    () =>
      apply((next) => {
        for (const key of Array.from(next.keys())) {
          if (key.startsWith("common.orderBy") || key === "common.itemsPerPage")
            continue;
          next.delete(key);
        }
      }),
    [apply],
  );

  return {
    searchParams,
    setParam,
    setParams,
    removeParam,
    activeFilterCount,
    clearFilters,
  };
};

const SelectionBar = (props: {
  count: number;
  isDeleting: boolean;
  onClear: () => void;
  onDelete: () => void;
  onAsk?: () => PinnedContext;
}) => (
  <div className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-300 bg-slate-900 px-3 py-2 text-white">
    <span className="text-xs font-semibold tabular-nums">
      {props.count} selected on this page
    </span>
    <div className="flex-1" />
    {props.onAsk && (
      <AskAgentButton
        variant="white"
        label="Ask agent"
        pinned={props.onAsk}
      />
    )}
    <Button
      size="compact-xs"
      variant="white"
      color="red"
      leftSection={
        props.isDeleting ? (
          <Loader2 size={12} className="motion-safe:animate-spin" strokeWidth={2.5} />
        ) : (
          <Trash2 size={12} strokeWidth={2.5} />
        )
      }
      disabled={props.isDeleting}
      onClick={props.onDelete}
    >
      {props.isDeleting ? "Deleting…" : "Delete"}
    </Button>
    <Button
      size="compact-xs"
      variant="subtle"
      color="gray"
      leftSection={<X size={12} strokeWidth={2.5} />}
      disabled={props.isDeleting}
      onClick={props.onClear}
      styles={{ root: { color: "var(--color-slate-200)" } }}
    >
      Clear
    </Button>
  </div>
);

const ResourceListToolbar = (props: {
  kindName: string;
  countLabel: string;
  totalCount: number;
  showCount?: boolean;
  itemsPerPage: number;
  isFetching?: boolean;
  onRefresh?: () => void;
  density: ListDensity;
  onDensityChange: (value: ListDensity) => void;
  canCreate?: boolean;
  onCreate: () => void;
  selectedCount?: number;
  isDeleting?: boolean;
  onClearSelection?: () => void;
  onDeleteSelection?: () => void;
  onAskSelection?: () => PinnedContext;
}) => {
  const {
    searchParams,
    setParam,
    setParams,
    removeParam,
    activeFilterCount,
    clearFilters,
  } = useListParams();
  const orderBy = searchParams.get("common.orderBy.type") ?? "CREATED_AT";
  const orderMode = searchParams.get("common.orderBy.mode") ?? "DESC";
  const orderValue = `${orderBy}.${orderMode}`;
  const chips = buildFilterChips(searchParams);
  const hasSelection = (props.selectedCount ?? 0) > 0;

  return (
    <div className="sticky top-[60px] z-20 -mx-1 mb-4 flex flex-col gap-2.5 border-b border-slate-200 bg-canvas/95 px-1 pb-3 pt-3 backdrop-blur supports-[backdrop-filter]:bg-canvas/80 sm:top-[calc(60px+var(--page-header-height,0px))]">
      <PageTitleAddon>
        {props.showCount !== false && (
          <Tooltip
            label={`${props.totalCount.toLocaleString()} ${props.countLabel}`}
            withArrow
            openDelay={300}
          >
            <span className="text-micro rounded-full border border-slate-200 bg-white px-2 py-0.5 font-semibold tabular-nums text-slate-600">
              {props.totalCount.toLocaleString()}
            </span>
          </Tooltip>
        )}
        {props.isFetching && (
          <Loader2
            size={12}
            className="motion-safe:animate-spin text-slate-400"
            strokeWidth={2.5}
            aria-label="Loading resources"
          />
        )}
      </PageTitleAddon>
      <PageActions>
        {props.onRefresh && (
          <RefreshButton
            label="Refresh resources"
            loading={props.isFetching}
            onClick={props.onRefresh}
          />
        )}
        {props.canCreate && (
          <Button
            variant="filled"
            color="ink"
            size="xs"
            className="shrink-0"
            leftSection={<Plus size={13} />}
            onClick={props.onCreate}
          >
            Create {props.kindName}
          </Button>
        )}
      </PageActions>

      <div className="flex flex-wrap items-center gap-2">
        <div className="min-w-56 flex-1">
          <SearchList
            placeholder={`Search ${formatKindName(props.kindName).toLowerCase()} resources…`}
          />
        </div>

        <Select
          w={156}
          allowDeselect={false}
          aria-label="Sort resources"
          value={orderValue}
          data={[
            { label: "Newest first", value: "CREATED_AT.DESC" },
            { label: "Oldest first", value: "CREATED_AT.ASC" },
            { label: "Name A–Z", value: "NAME.ASC" },
            { label: "Name Z–A", value: "NAME.DESC" },
          ]}
          onChange={(value) => {
            if (!value) return;
            const [type, mode] = value.split(".");
            setParams({
              "common.orderBy.type": type,
              "common.orderBy.mode": mode,
            });
          }}
        />

        <SegmentedControl
          value={props.density}
          onChange={(v) => props.onDensityChange(v as ListDensity)}
          aria-label="List density"
          data={[
            {
              label: (
                <Tooltip label="Comfortable" withArrow>
                  <span className="flex items-center">
                    <LayoutGrid size={13} strokeWidth={2.5} />
                  </span>
                </Tooltip>
              ),
              value: "comfortable",
            },
            {
              label: (
                <Tooltip label="Compact" withArrow>
                  <span className="flex items-center">
                    <Rows3 size={13} strokeWidth={2.5} />
                  </span>
                </Tooltip>
              ),
              value: "compact",
            },
            {
              label: (
                <Tooltip label="Table" withArrow>
                  <span className="flex items-center">
                    <Table2 size={13} strokeWidth={2.5} />
                  </span>
                </Tooltip>
              ),
              value: "table",
            },
          ]}
        />

      </div>

      {hasSelection ? (
        <SelectionBar
          count={props.selectedCount!}
          isDeleting={!!props.isDeleting}
          onClear={props.onClearSelection ?? (() => {})}
          onDelete={props.onDeleteSelection ?? (() => {})}
          onAsk={props.onAskSelection}
        />
      ) : (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <FilterChips chips={chips} onRemove={removeParam} />

          {activeFilterCount > 0 && (
            <button
              type="button"
              onClick={clearFilters}
              className="text-xs font-semibold text-slate-600 hover:text-slate-950"
            >
              Clear filters
            </button>
          )}

          <div className="ml-auto flex items-center gap-1.5">
            <span className="text-xs font-normal text-slate-600">Per page</span>
            <Select
              size="xs"
              w={78}
              allowDeselect={false}
              aria-label="Items per page"
              data={PAGE_SIZES}
              value={`${props.itemsPerPage}`}
              onChange={(v) => v && setParam("common.itemsPerPage", v)}
            />
          </div>
        </div>
      )}
    </div>
  );
};

export default ResourceListToolbar;
