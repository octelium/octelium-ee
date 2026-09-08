import { Button, Select, SegmentedControl, Tooltip } from "@mantine/core";
import {
  ArrowDownWideNarrow,
  ArrowUpWideNarrow,
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
import { FilterChips, buildFilterChips } from "../Paginator";
import SearchList from "../SearchList";

export type ListDensity = "comfortable" | "compact" | "table";

const DENSITY_STORAGE_KEY = "octelium.console.listDensity";

const isDensity = (value: unknown): value is ListDensity =>
  value === "comfortable" || value === "compact" || value === "table";

export const useListDensity = (): [ListDensity, (v: ListDensity) => void] => {
  const [density, setDensity] = React.useState<ListDensity>(() => {
    try {
      const stored = window.localStorage.getItem(DENSITY_STORAGE_KEY);
      return isDensity(stored) ? stored : "comfortable";
    } catch {
      return "comfortable";
    }
  });

  const update = React.useCallback((value: ListDensity) => {
    setDensity(value);
    try {
      window.localStorage.setItem(DENSITY_STORAGE_KEY, value);
    } catch {}
  }, []);

  return [density, update];
};

const PAGE_SIZES = ["10", "25", "50", "100"];

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

  const removeParam = React.useCallback(
    (key: string) =>
      apply((next) => {
        next.delete(key);
        next.delete("common.page");
      }),
    [apply],
  );

  const activeFilterCount = React.useMemo(() => {
    let count = 0;
    for (const [key, value] of searchParams.entries()) {
      if (!value) continue;
      if (key === "common.page") continue;
      if (key.startsWith("common.orderBy")) continue;
      if (key === "common.itemsPerPage") continue;
      count += 1;
    }
    return count;
  }, [searchParams]);

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
}) => (
  <div className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-300 bg-slate-900 px-3 py-2 text-white">
    <span className="text-xs font-semibold tabular-nums">
      {props.count} selected
    </span>
    <div className="flex-1" />
    <Button
      size="compact-xs"
      variant="white"
      color="red"
      leftSection={
        props.isDeleting ? (
          <Loader2 size={12} className="animate-spin" strokeWidth={2.5} />
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
      styles={{ root: { color: "#e2e8f0" } }}
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
  density: ListDensity;
  onDensityChange: (value: ListDensity) => void;
  canCreate?: boolean;
  onCreate: () => void;
  selectedCount?: number;
  isDeleting?: boolean;
  onClearSelection?: () => void;
  onDeleteSelection?: () => void;
}) => {
  const { searchParams, setParam, removeParam } = useListParams();
  const orderBy = searchParams.get("common.orderBy.type") ?? "CREATED_AT";
  const orderMode = searchParams.get("common.orderBy.mode") ?? "DESC";
  const chips = buildFilterChips(searchParams);
  const hasSelection = (props.selectedCount ?? 0) > 0;

  return (
    <div className="sticky top-[60px] z-20 -mx-1 mb-4 flex flex-col gap-2.5 border-b border-slate-200 bg-slate-100/95 px-1 pb-3 pt-1 backdrop-blur supports-[backdrop-filter]:bg-slate-100/80">
      <div className="flex flex-wrap items-center gap-2">
        <div className="min-w-56 flex-1">
          <SearchList
            placeholder={`Search ${props.countLabel.toLowerCase()}…`}
          />
        </div>

        <SegmentedControl
          value={orderBy}
          onChange={(v) => setParam("common.orderBy.type", v)}
          aria-label="Sort field"
          data={[
            { label: "Name", value: "NAME" },
            { label: "Created", value: "CREATED_AT" },
          ]}
        />

        <SegmentedControl
          value={orderMode}
          onChange={(v) => setParam("common.orderBy.mode", v)}
          aria-label="Sort direction"
          data={[
            {
              label: (
                <Tooltip label="Ascending" withArrow>
                  <span className="flex items-center">
                    <ArrowUpWideNarrow size={13} strokeWidth={2.5} />
                  </span>
                </Tooltip>
              ),
              value: "ASC",
            },
            {
              label: (
                <Tooltip label="Descending" withArrow>
                  <span className="flex items-center">
                    <ArrowDownWideNarrow size={13} strokeWidth={2.5} />
                  </span>
                </Tooltip>
              ),
              value: "DESC",
            },
          ]}
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

        {props.canCreate && (
          <Button
            variant="filled"
            color="dark"
            className="!shadow-[0_8px_20px_-6px_rgba(15,23,42,0.35)]"
            leftSection={<Plus size={14} />}
            onClick={props.onCreate}
          >
            Create {props.kindName}
          </Button>
        )}
      </div>

      {hasSelection ? (
        <SelectionBar
          count={props.selectedCount!}
          isDeleting={!!props.isDeleting}
          onClear={props.onClearSelection ?? (() => {})}
          onDelete={props.onDeleteSelection ?? (() => {})}
        />
      ) : (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <span className="text-xs font-normal tabular-nums text-slate-600">
            {props.showCount === false
              ? "Loading…"
              : `${props.totalCount.toLocaleString()} ${props.countLabel}`}
            {props.isFetching && props.showCount !== false && (
              <Loader2
                size={11}
                className="ml-1.5 inline animate-spin align-[-1px] text-slate-500"
                strokeWidth={2.5}
              />
            )}
          </span>

          <FilterChips chips={chips} onRemove={removeParam} />

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
