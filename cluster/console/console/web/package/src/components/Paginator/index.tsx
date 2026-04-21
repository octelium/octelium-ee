import { ListResponseMeta } from "@/apis/metav1/metav1";
import { Pagination, SegmentedControl } from "@mantine/core";
import { ArrowDownWideNarrow, ArrowUpWideNarrow, X } from "lucide-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { twMerge } from "tailwind-merge";

const REF_PARAM_LABELS: Record<string, string> = {
  "userRef.name": "User",
  "sessionRef.name": "Session",
  "deviceRef.name": "Device",
  "namespaceRef.name": "Namespace",
  "serviceRef.name": "Service",
  "groupRef.name": "Group",
  "identityProviderRef.name": "Identity Provider",
  "regionRef.name": "Region",
  "policyRef.name": "Policy",
  "credentialRef.name": "Credential",
};

const TYPE_LABEL_MAP: Record<string, string> = {
  HUMAN: "Human",
  WORKLOAD: "Workload",
  ACTIVE: "Active",
  REJECTED: "Rejected",
  PENDING: "Pending",
  HTTP: "HTTP",
  TCP: "TCP",
  SSH: "SSH",
  WEB: "Web",
  GRPC: "gRPC",
  POSTGRES: "PostgreSQL",
  MYSQL: "MySQL",
  UDP: "UDP",
  DNS: "DNS",
};

const BOOLEAN_PARAM_LABELS: Record<string, string> = {
  isDisabled: "Disabled",
  isPublic: "Public",
  isTLS: "TLS",
  isAnonymous: "Anonymous",
  isSystem: "System",
  isUserHidden: "Hidden",
};

interface FilterChip {
  key: string;
  label: string;
  value: string;
}

const buildFilterChips = (searchParams: URLSearchParams): FilterChip[] => {
  const chips: FilterChip[] = [];

  for (const [key, value] of searchParams.entries()) {
    if (key.startsWith("common.") || !value) continue;

    if (key in REF_PARAM_LABELS) {
      chips.push({ key, label: REF_PARAM_LABELS[key], value });
      continue;
    }

    if (key === "type" || key === "mode") {
      chips.push({
        key,
        label: key === "type" ? "Type" : "Mode",
        value: TYPE_LABEL_MAP[value] ?? value,
      });
      continue;
    }

    if (value === "true") {
      chips.push({
        key,
        label: BOOLEAN_PARAM_LABELS[key] ?? key,
        value: "Yes",
      });
      continue;
    }
  }

  return chips;
};

const FilterChips = ({
  chips,
  onRemove,
}: {
  chips: FilterChip[];
  onRemove: (key: string) => void;
}) => {
  if (chips.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-[0.65rem] font-bold uppercase tracking-[0.07em] text-slate-400 shrink-0">
        Filters
      </span>
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-md border border-slate-200 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05)] text-[0.7rem] font-semibold text-slate-600"
        >
          <span className="text-slate-400 font-bold">{chip.label}:</span>
          <span>{chip.value}</span>
          <button
            onClick={() => onRemove(chip.key)}
            className="flex items-center justify-center w-4 h-4 rounded ml-0.5 text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors duration-150 cursor-pointer"
            title={`Remove ${chip.label} filter`}
          >
            <X size={10} strokeWidth={2.5} />
          </button>
        </span>
      ))}
    </div>
  );
};

const Paginator = (props: {
  meta?: ListResponseMeta;
  onPageChange?: (page: number) => void;
}) => {
  const { meta } = props;
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const loc = useLocation();

  if (!meta) return null;

  const totalPages = Math.ceil(meta.totalCount / meta.itemsPerPage);
  const hasMultiplePages = totalPages > 1;
  const hasItems = meta.totalCount > 0;

  if (!hasItems) return null;

  const currentOrderBy =
    searchParams.get("common.orderBy.type") ?? "CREATED_AT";
  const currentOrderMode = searchParams.get("common.orderBy.mode") ?? "DESC";

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams.toString());
    next.set(key, value);
    navigate(`${loc.pathname}?${next.toString()}`);
  };

  const removeParam = (key: string) => {
    const next = new URLSearchParams(searchParams.toString());
    next.delete(key);
    next.delete("common.page");
    navigate(`${loc.pathname}?${next.toString()}`);
  };

  const itemCountLabel =
    meta.totalCount === 1
      ? "1 item"
      : `${meta.totalCount.toLocaleString()} items`;

  const filterChips = buildFilterChips(searchParams);

  return (
    <div className="w-full flex flex-col gap-3 my-5">
      {filterChips.length > 0 && (
        <FilterChips chips={filterChips} onRemove={removeParam} />
      )}

      <div className="flex items-center justify-between gap-4">
        {hasMultiplePages ? (
          <Pagination
            total={totalPages}
            value={meta.page + 1}
            withEdges
            radius="md"
            color="dark"
            classNames={{
              control: twMerge(
                "!font-bold !text-[0.78rem]",
                "!border-slate-200 !bg-white !text-slate-700",
                "!shadow-[0_1px_3px_rgba(15,23,42,0.07)]",
                "hover:!bg-slate-50 hover:!border-slate-300",
                "!transition-colors !duration-150",
                "data-[active]:!bg-slate-900 data-[active]:!border-slate-900 data-[active]:!text-white",
              ),
            }}
            onChange={(v) => {
              setParam("common.page", `${v}`);
              props.onPageChange?.(v);
            }}
          />
        ) : (
          <span
            className="text-[0.75rem] font-bold text-slate-500 tracking-wide"
            style={{ textShadow: "0 1px 2px rgba(15,23,42,0.08)" }}
          >
            {itemCountLabel}
          </span>
        )}

        <div className="flex items-center gap-2">
          <SegmentedControl
            value={currentOrderBy}
            onChange={(v) => setParam("common.orderBy.type", v)}
            data={[
              { label: "Name", value: "NAME" },
              { label: "Created", value: "CREATED_AT" },
            ]}
          />

          <SegmentedControl
            value={currentOrderMode}
            onChange={(v) => setParam("common.orderBy.mode", v)}
            data={[
              {
                label: (
                  <span className="flex items-center">
                    <ArrowUpWideNarrow size={13} strokeWidth={2.5} />
                  </span>
                ),
                value: "ASC",
              },
              {
                label: (
                  <span className="flex items-center">
                    <ArrowDownWideNarrow size={13} strokeWidth={2.5} />
                  </span>
                ),
                value: "DESC",
              },
            ]}
          />

          {hasMultiplePages && (
            <span
              className="text-[0.75rem] font-bold text-slate-500 tracking-wide whitespace-nowrap ml-1"
              style={{ textShadow: "0 1px 2px rgba(15,23,42,0.08)" }}
            >
              {itemCountLabel}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

export default Paginator;
