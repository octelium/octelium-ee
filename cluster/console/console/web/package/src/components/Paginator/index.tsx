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
  "authenticatorRef.name": "Authenticator",
  "resourceRef.name": "Resource",
  "clusterRef.name": "Cluster",
  "catalogRef.name": "Catalog",
  "requestRef.name": "Request",
  "reviewerRef.name": "Reviewer",
  "subjectUserRef.name": "Subject User",
  "policyTriggerRef.name": "Policy Trigger",
  "integrationRef.name": "Integration",
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
  RDP: "RDP",
  SLACK: "Slack",
  JIRA: "Jira",
  WEBHOOK: "Webhook",
  READY: "Ready",
  DEGRADED: "Degraded",
  ERROR: "Error",
  CLOSED: "Closed",
  SYNC_REQUESTED: "Sync requested",
  SYNCING: "Synchronizing",
  SUCCESS: "Successful",
  FAILED: "Failed",
  NOTIFICATION: "Notification",
  DIRECT_USER_DELIVERY: "Direct user delivery",
  INTERACTIVE_REVIEW: "Interactive review",
  REQUEST_CREATION: "Request creation",
  IDENTITY_RESOLUTION: "Identity resolution",
  PRESENTATION_UPDATE: "Presentation update",
  EMAIL_DISCOVERY: "Email discovery",
  REVIEW_SURFACE: "Review surface",
  SHARED: "Shared",
  REVIEWERS: "Reviewers",
  REQUESTER: "Requester",
  SUBJECT: "Subject",
  DEEP_LINK_ONLY: "Deep link only",
  INTERACTIVE: "Interactive",
};

const BOOLEAN_PARAM_LABELS: Record<string, string> = {
  isDisabled: "Disabled",
  isPublic: "Public",
  isTLS: "TLS",
  isAnonymous: "Anonymous",
  isSystem: "System",
  isUserHidden: "Hidden",
  isActive: "Active",
  isDecided: "Decided",
  isOutOfDate: "Out of date",
  isFailing: "Failing",
  hasDeadline: "With deadline",
  isDeadlinePassed: "Past deadline",
};

export interface FilterChip {
  key: string;
  label: string;
  value: string;
}

const formatParamLabel = (key: string) => {
  const leaf = key.split(".").at(-1) ?? key;
  const spaced = leaf.replace(/([a-z0-9])([A-Z])/g, "$1 $2");
  return spaced.charAt(0).toUpperCase() + spaced.slice(1);
};

export const isListFilterParam = (key: string, value: string) =>
  !!value &&
  key !== "common.page" &&
  key !== "common.itemsPerPage" &&
  !key.startsWith("common.orderBy");

export const countActiveListFilters = (searchParams: URLSearchParams) => {
  let count = 0;
  for (const [key, value] of searchParams.entries()) {
    if (isListFilterParam(key, value)) count += 1;
  }
  return count;
};

export const buildFilterChips = (
  searchParams: URLSearchParams,
): FilterChip[] => {
  const chips: FilterChip[] = [];

  for (const [key, value] of searchParams.entries()) {
    if (!isListFilterParam(key, value) || key === "common.query") continue;

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

    if (value === "true" || value === "false") {
      chips.push({
        key,
        label: BOOLEAN_PARAM_LABELS[key] ?? formatParamLabel(key),
        value: value === "true" ? "Yes" : "No",
      });
      continue;
    }

    chips.push({
      key,
      label: REF_PARAM_LABELS[key] ?? formatParamLabel(key),
      value: TYPE_LABEL_MAP[value] ?? value,
    });
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
      <span className="shrink-0 text-[10px] font-semibold uppercase tracking-[0.06em] text-slate-500">
        Filters
      </span>
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="inline-flex items-center gap-1 rounded border border-slate-200 bg-slate-50 py-px pl-1.5 pr-1 text-[10px] font-medium leading-4 text-slate-600"
        >
          <span className="font-semibold text-slate-500">{chip.label}:</span>
          <span>{chip.value}</span>
          <button
            type="button"
            onClick={() => onRemove(chip.key)}
            aria-label={`Remove ${chip.label} filter`}
            className="ml-0.5 flex h-3.5 w-3.5 cursor-pointer items-center justify-center rounded text-slate-400 transition-colors duration-150 hover:bg-slate-200 hover:text-slate-800"
            title={`Remove ${chip.label} filter`}
          >
            <X size={9} strokeWidth={2.5} />
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
            color="ink"
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
