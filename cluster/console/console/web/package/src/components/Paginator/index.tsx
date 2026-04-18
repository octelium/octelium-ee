import { ListResponseMeta } from "@/apis/metav1/metav1";
import { Button, Pagination } from "@mantine/core";
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
      chips.push({
        key,
        label: REF_PARAM_LABELS[key],
        value,
      });
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

    if (key === "isDisabled") {
      chips.push({ key, label: "Disabled", value: "Yes" });
      continue;
    }

    if (key === "isPublic") {
      chips.push({ key, label: "Public", value: "Yes" });
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

  const btnStyles = (active: boolean) => ({
    root: {
      height: "28px",
      fontSize: "0.72rem",
      fontWeight: 700,
      padding: "0 10px",
      backgroundColor: active ? "#0f172a" : "#ffffff",
      color: active ? "#ffffff" : "#64748b",
      border: "none",
      borderRadius: 0,
      transition: "background-color 150ms, color 150ms",
      "&:hover": {
        backgroundColor: active ? "#1e293b" : "#f8fafc",
        color: active ? "#ffffff" : "#0f172a",
      },
    },
  });

  const iconBtnStyles = (active: boolean) => ({
    root: {
      height: "28px",
      width: "28px",
      padding: 0,
      backgroundColor: active ? "#0f172a" : "#ffffff",
      color: active ? "#ffffff" : "#64748b",
      border: "none",
      borderRadius: 0,
      transition: "background-color 150ms, color 150ms",
      "&:hover": {
        backgroundColor: active ? "#1e293b" : "#f8fafc",
        color: active ? "#ffffff" : "#0f172a",
      },
    },
  });

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

        <div className="flex items-center gap-1.5">
          <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
            {[
              { label: "Name", value: "NAME" },
              { label: "Created", value: "CREATED_AT" },
            ].map((opt) => (
              <Button
                key={opt.value}
                onClick={() => setParam("common.orderBy.type", opt.value)}
                styles={btnStyles(currentOrderBy === opt.value)}
              >
                {opt.label}
              </Button>
            ))}
          </Button.Group>

          <Button.Group className="rounded-md overflow-hidden border border-slate-200 shadow-[0_1px_3px_rgba(15,23,42,0.06)]">
            {(
              [
                { value: "ASC", Icon: ArrowUpWideNarrow },
                { value: "DESC", Icon: ArrowDownWideNarrow },
              ] as const
            ).map(({ value, Icon }) => (
              <Button
                key={value}
                onClick={() => setParam("common.orderBy.mode", value)}
                styles={iconBtnStyles(currentOrderMode === value)}
                title={value === "ASC" ? "Ascending" : "Descending"}
              >
                <Icon size={13} />
              </Button>
            ))}
          </Button.Group>

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
