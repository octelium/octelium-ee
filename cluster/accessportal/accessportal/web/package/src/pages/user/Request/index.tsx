import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, Pagination, SegmentedControl } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { notifications } from "@mantine/notifications";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { ChevronRight, Inbox, Plus, SearchX } from "lucide-react";
import * as React from "react";
import { Link, useLocation, useNavigate, useSearchParams } from "react-router-dom";

import { ResourceIcon } from "@/components/Access/icons";
import TimeAgo, { Countdown, Elapsed } from "@/components/TimeAgo";
import {
  Badge,
  Card,
  ConfirmDialog,
  DocumentTitle,
  EmptyState,
  ErrorState,
  IconTile,
  Note,
  PageHeader,
  ProgressBar,
  RefreshButton,
  SearchInput,
  SkeletonRows,
  StatusBadge,
  SummaryRow,
  Toolbar,
  UrgencyBadge,
} from "@/ui";
import {
  accessWindow,
  currentReviewStepIndex,
  formatDuration,
  isOnBehalfOf,
  isPendingRequest,
  isScheduledRequest,
  openInformationRequest,
  requestCreatedAt,
  requestRefetchInterval,
  requestResourceLabel,
  requestSubjectName,
  reviewSteps,
  statusMeta,
  urgencyMeta,
  userInfoLabel,
} from "@/utils";
import { getUserClient } from "@/utils/client";
import { useNow } from "@/utils/time";

const ITEMS_PER_PAGE = 20;

const MetaLine = (props: { items: React.ReactNode[] }) => (
  <p className="mt-0.5 text-xs font-normal leading-5 text-slate-500">
    {props.items
      .filter((item) => item !== null && item !== false && item !== undefined)
      .map((item, index) => (
        <React.Fragment key={index}>
          {index > 0 && <span aria-hidden className="mx-1.5 text-slate-300">·</span>}
          {item}
        </React.Fragment>
      ))}
  </p>
);

const RequestRow = (props: {
  item: AccessP.Request;
  returnTo: string;
  onCancel: (name: string) => void;
}) => {
  const { item } = props;
  const now = useNow();
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const pending = isPendingRequest(item);
  const steps = reviewSteps(item);
  const grant = accessWindow(item, now);
  const active = status.group === "active" && !!grant.endMs && grant.endMs > now;
  const urgency = urgencyMeta(item.spec?.urgency);

  return (
    <Card interactive>
      <div className="flex items-stretch">
        <Link
          to={`/user/requests/${item.metadata!.name}`}
          state={{ returnTo: props.returnTo }}
          className="flex min-w-0 flex-1 items-center gap-3.5 rounded-xl px-4 py-3.5 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400"
        >
          <IconTile>
            <ResourceIcon kind={resource.kind} size={16} strokeWidth={2} />
          </IconTile>

          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate text-body font-semibold text-slate-800">
                {resource.displayName || resource.name || item.metadata!.name}
              </span>
              {resource.accessLevel ? (
                <Badge>{resource.accessLevel}</Badge>
              ) : (
                resource.kind === "Catalog" && <Badge>Catalog</Badge>
              )}
              {openInformationRequest(item) && (
                <Badge tone="blue">Reply needed</Badge>
              )}
              {isScheduledRequest(item, now) && (
                <Badge tone="blue">Scheduled</Badge>
              )}
              <span className="ml-auto shrink-0 sm:hidden">
                <StatusBadge status={item.status?.state?.status} />
              </span>
            </div>

            <MetaLine
              items={[
                <span key="created">
                  Created <TimeAgo rfc3339={requestCreatedAt(item)} />
                </span>,
                formatDuration(item.spec?.duration),
                isOnBehalfOf(item) && (
                  <span key="subject">
                    for{" "}
                    <span className="font-medium text-slate-700">
                      {userInfoLabel(
                        item.status?.subjectInfo,
                        requestSubjectName(item),
                      )}
                    </span>
                  </span>
                ),
                pending &&
                  steps.length > 0 &&
                  `Step ${Math.min(currentReviewStepIndex(item) + 1, steps.length)} of ${steps.length}`,
                pending && (
                  <span key="waiting">
                    Waiting{" "}
                    <Elapsed
                      rfc3339={item.status?.approvalStartAt ?? item.metadata?.createdAt}
                    />
                  </span>
                ),
              ]}
            />

            {active && (
              <div className="mt-2 flex items-center gap-2.5">
                <ProgressBar
                  value={grant.progress}
                  tone={grant.progress > 0.8 ? "amber" : "emerald"}
                  label="Elapsed share of the access window"
                  className="max-w-[200px]"
                />
                <span className="text-xs font-medium text-slate-700">
                  <Countdown date={new Date(grant.endMs!)} suffix=" of access left" />
                </span>
              </div>
            )}
          </div>

          <div className="hidden shrink-0 items-center gap-2 sm:flex">
            {urgency.level >= 4 && <UrgencyBadge urgency={item.spec?.urgency} />}
            <StatusBadge status={item.status?.state?.status} withHint />
          </div>
        </Link>

        <div className="flex shrink-0 items-center pr-3">
          {pending ? (
            <Button
              size="compact-xs"
              variant="default"
              aria-label={`Cancel the request for ${resource.displayName || resource.name || item.metadata!.name}`}
              onClick={() => props.onCancel(item.metadata!.name)}
            >
              Cancel
            </Button>
          ) : (
            <ChevronRight size={16} aria-hidden className="text-slate-400" />
          )}
        </div>
      </div>
    </Card>
  );
};

const Requests = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const search = searchParams.get("q") ?? "";
  const group = searchParams.get("group") ?? "all";
  const [query] = useDebouncedValue(search.trim(), 300);
  const [cancelName, setCancelName] = React.useState<string>();

  const serverState =
    group === "pending"
      ? AccessP.Request_Status_State_Status.PENDING
      : group === "active"
        ? AccessP.Request_Status_State_Status.APPROVED
        : undefined;

  const qry = useQuery({
    queryKey: ["user", "listRequest", page, group, query],
    placeholderData: keepPreviousData,
    queryFn: async ({ signal }) => {
      const { response } = await getUserClient().listRequest(
        AccessP.ListUserRequestOptions.create({
          common: { page: page - 1, itemsPerPage: ITEMS_PER_PAGE },
          query,
          ...(group === "pending"
            ? { state: AccessP.Request_Status_State_Status.PENDING }
            : {}),
          ...(group === "active" ? { isActive: true } : {}),
        }),
        { abort: signal },
      );
      return response;
    },
    refetchInterval: (query) => {
      const intervals = (query.state.data?.items ?? [])
        .map(requestRefetchInterval)
        .filter((value): value is number => value !== false);
      return intervals.length ? Math.min(...intervals) : false;
    },
  });

  const cancelMutation = useMutation({
    mutationFn: async (name: string) => {
      const { response } = await getUserClient().cancelRequest(
        AccessP.CancelRequestRequest.create({ requestRef: { name } }),
      );
      return response;
    },
    onSuccess: (_, name) => {
      notifications.show({ color: "green", message: "Request cancelled" });
      setCancelName(undefined);
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
      queryClient.invalidateQueries({ queryKey: ["user", "getRequest", name] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to cancel request",
      });
    },
  });

  const items = qry.data?.items ?? [];
  const meta = qry.data?.listResponseMeta;
  const perPage = meta?.itemsPerPage || ITEMS_PER_PAGE;
  const totalPages = meta ? Math.max(1, Math.ceil(meta.totalCount / perPage)) : 1;
  const singlePage = totalPages <= 1 && page === 1;

  const countOf = (value: string) =>
    value === "all"
      ? items.length
      : items.filter((item) => statusMeta(item.status?.state?.status).group === value)
          .length;

  const searching = search.trim() !== "";
  const visibleItems = items.filter(
    (item) =>
      group === "all" ||
      serverState !== undefined ||
      statusMeta(item.status?.state?.status).group === group,
  );

  const setParam = (key: string, value: string, fallback: string) =>
    setSearchParams(
      (prev) => {
        if (value && value !== fallback) prev.set(key, value);
        else prev.delete(key);
        if (key !== "page") prev.delete("page");
        return prev;
      },
      { replace: key === "q", preventScrollReset: key !== "page" },
    );

  const cancelTarget = items.find((item) => item.metadata!.name === cancelName);
  const filtered = group === "closed";

  return (
    <div className="w-full">
      <DocumentTitle title="My Requests" />
      <PageHeader
        title="My Requests"
        description="Track your just-in-time access requests and the access they granted."
        actions={
          <>
            <RefreshButton onClick={() => qry.refetch()} loading={qry.isFetching} />
            <Button
              variant="filled"
              color="ink"
              leftSection={<Plus size={14} strokeWidth={2.5} />}
              onClick={() => navigate("/user/new")}
            >
              New Request
            </Button>
          </>
        }
      />

      <Toolbar>
        <SearchInput
          value={search}
          onChange={(value) => setParam("q", value, "")}
          placeholder="Search by resource, request ID, recipient or reason"
          ariaLabel="Search requests"
        />
        <SegmentedControl
          value={group}
          onChange={(value) => setParam("group", value, "all")}
          aria-label="Filter by status"
          data={[
            { value: "all", label: "All" },
            { value: "pending", label: "Pending" },
            { value: "active", label: "Active" },
            { value: "closed", label: "Closed" },
          ].map((option) => ({
            ...option,
            label:
              serverState !== undefined
                ? option.value === group && meta
                  ? `${option.label} (${meta.totalCount})`
                  : option.label
                : singlePage && countOf(option.value) > 0
                  ? `${option.label} (${countOf(option.value)})`
                  : option.label,
          }))}
        />
      </Toolbar>

      {!singlePage && filtered && (
        <Note className="mb-3">
          The Closed filter applies to the {items.length} requests on this page.
        </Note>
      )}

      {qry.isLoading ? (
        <SkeletonRows rows={5} />
      ) : qry.isError && !qry.data ? (
        <ErrorState
          title="Could not load your requests"
          error={qry.error}
          onRetry={() => qry.refetch()}
        />
      ) : visibleItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={
              items.length || searching ? (
                <SearchX size={18} strokeWidth={2} />
              ) : (
                <Inbox size={18} strokeWidth={2} />
              )
            }
            title={items.length || searching ? "No matching requests" : "No requests yet"}
            description={
              items.length || searching
                ? "Try a different search term or switch the status filter."
                : "Request access to a Service or a Catalog and it will show up here."
            }
            action={
              items.length || searching ? (
                <Button
                  variant="default"
                  onClick={() =>
                    setSearchParams(
                      (prev) => {
                        prev.delete("q");
                        prev.delete("group");
                        return prev;
                      },
                      { replace: true },
                    )
                  }
                >
                  Clear filters
                </Button>
              ) : (
                <Button
                  variant="filled"
                  color="ink"
                  leftSection={<Plus size={14} strokeWidth={2.5} />}
                  onClick={() => navigate("/user/new")}
                >
                  New Request
                </Button>
              )
            }
          />
        </Card>
      ) : (
        <div className="flex flex-col gap-2">
          {visibleItems.map((item) => (
            <RequestRow
              key={item.metadata!.uid || item.metadata!.name}
              item={item}
              returnTo={`${location.pathname}${location.search}`}
              onCancel={setCancelName}
            />
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="mt-6 flex flex-col items-center gap-2">
          <Pagination
            value={page}
            total={totalPages}
            onChange={(value) => setParam("page", String(value), "1")}
            color="ink"
            size="sm"
            radius="md"
          />
          <span className="text-micro font-normal text-slate-500">
            {meta?.totalCount ?? 0} requests in total
          </span>
        </div>
      )}

      <ConfirmDialog
        opened={!!cancelName}
        onClose={() => setCancelName(undefined)}
        onConfirm={() => {
          if (cancelName) cancelMutation.mutate(cancelName);
        }}
        title="Cancel this request?"
        description="The request stops progressing through its review workflow and cannot be restored. You can always create a new one."
        details={
          cancelTarget ? (
            <SummaryRow label="Resource">
              {requestResourceLabel(cancelTarget).displayName || cancelTarget.metadata!.name}
            </SummaryRow>
          ) : undefined
        }
        confirmLabel="Cancel request"
        danger
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default Requests;
