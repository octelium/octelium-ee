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
}) => (
  <div className="flex flex-wrap items-center gap-2 rounded-lg border border-slate-300 bg-slate-900 px-3 py-2 text-white">
    <span className="text-xs font-semibold tabular-nums">
      {props.count} selected on this page
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
    <div className="sticky top-[60px] z-20 -mx-1 mb-4 flex flex-col gap-2.5 border-b border-slate-200 bg-slate-100/95 px-1 pb-3 pt-1 backdrop-blur supports-[backdrop-filter]:bg-slate-100/80">
      <div className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          <h1 className="truncate text-lg font-semibold tracking-tight text-slate-950">
            {formatKindName(props.kindName)} resources
          </h1>
          <p className="mt-0.5 text-xs font-normal text-slate-600">
            {props.showCount === false
              ? "Loading resources…"
              : `${props.totalCount.toLocaleString()} ${props.countLabel}`}
            {props.isFetching && props.showCount !== false && (
              <Loader2
                size={11}
                className="ml-1.5 inline animate-spin align-[-1px] text-slate-500"
                strokeWidth={2.5}
              />
            )}
          </p>
        </div>
        {props.canCreate && (
          <Button
            variant="filled"
            color="dark"
            className="shrink-0 !shadow-[0_8px_20px_-6px_rgba(15,23,42,0.35)]"
            leftSection={<Plus size={14} />}
            onClick={props.onCreate}
          >
            Create {props.kindName}
          </Button>
        )}
      </div>

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
