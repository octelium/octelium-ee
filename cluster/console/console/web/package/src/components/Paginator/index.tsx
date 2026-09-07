import { ListResponseMeta } from "@/apis/metav1/metav1";
import { Pagination } from "@mantine/core";
import { X } from "lucide-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

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
  SOCKS5: "SOCKS5",
  RDP_WEB: "RDP Web",
};

const BOOLEAN_PARAM_LABELS: Record<string, string> = {
  isDisabled: "Disabled",
  isPublic: "Public",
  isTLS: "TLS",
  isAnonymous: "Anonymous",
  isSystem: "System",
  isUserHidden: "Hidden",
};

export interface FilterChip {
  key: string;
  label: string;
  value: string;
}

export const buildFilterChips = (
  searchParams: URLSearchParams,
): FilterChip[] => {
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

export const FilterChips = ({
  chips,
  onRemove,
}: {
  chips: FilterChip[];
  onRemove: (key: string) => void;
}) => {
  if (chips.length === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="shrink-0 text-xs font-normal text-slate-500">
        Filters
      </span>
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white py-0.5 pl-2 pr-1 text-xs font-normal text-slate-700"
        >
          <span className="font-semibold text-slate-500">{chip.label}:</span>
          <span>{chip.value}</span>
          <button
            type="button"
            onClick={() => onRemove(chip.key)}
            aria-label={`Remove ${chip.label} filter`}
            className="ml-0.5 flex h-4 w-4 cursor-pointer items-center justify-center rounded text-slate-500 transition-colors duration-150 hover:bg-slate-100 hover:text-slate-900"
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
  showFilters?: boolean;
}) => {
  const { meta } = props;
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const loc = useLocation();

  if (!meta) return null;

  const itemsPerPage = Math.max(meta.itemsPerPage, 1);
  const totalPages = Math.max(1, Math.ceil(meta.totalCount / itemsPerPage));
  const hasMultiplePages = totalPages > 1;

  if (meta.totalCount <= 0) return null;

  const navigateWithParams = (next: URLSearchParams) => {
    const search = next.toString();
    navigate(
      {
        pathname: loc.pathname,
        search: search ? `?${search}` : "",
        hash: loc.hash,
      },
      { state: loc.state, preventScrollReset: true },
    );
  };

  const removeParam = (key: string) => {
    const next = new URLSearchParams(searchParams.toString());
    next.delete(key);
    next.delete("common.page");
    navigateWithParams(next);
  };

  const filterChips =
    props.showFilters === false ? [] : buildFilterChips(searchParams);

  const first = meta.page * itemsPerPage + 1;
  const last = Math.min(meta.totalCount, (meta.page + 1) * itemsPerPage);

  return (
    <div className="my-5 flex w-full flex-col gap-3">
      {filterChips.length > 0 && (
        <FilterChips chips={filterChips} onRemove={removeParam} />
      )}

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <span className="text-xs font-normal tabular-nums text-slate-600">
          {hasMultiplePages
            ? `${first.toLocaleString()}–${last.toLocaleString()} of ${meta.totalCount.toLocaleString()}`
            : `${meta.totalCount.toLocaleString()} ${meta.totalCount === 1 ? "item" : "items"}`}
        </span>

        {hasMultiplePages && (
          <Pagination
            total={totalPages}
            value={Math.min(Math.max(meta.page + 1, 1), totalPages)}
            withEdges
            radius="md"
            color="dark"
            onChange={(v) => {
              if (props.onPageChange) {
                props.onPageChange(v - 1);
                return;
              }
              const next = new URLSearchParams(searchParams.toString());
              next.set("common.page", `${v}`);
              navigateWithParams(next);
            }}
          />
        )}
      </div>
    </div>
  );
};

export default Paginator;
