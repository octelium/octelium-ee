import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, Pagination, SegmentedControl, Select, Tooltip } from "@mantine/core";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import {
  ChevronRight,
  Inbox,
  Layers,
  Plus,
  ServerCog,
  Timer,
  Workflow,
  X,
} from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";

import { resourceIcon } from "@/components/Access/icons";
import TimeAgo, { Countdown, Elapsed, useNow } from "@/components/TimeAgo";
import {
  Badge,
  Card,
  ConfirmDialog,
  EmptyState,
  ErrorState,
  IconTile,
  PageHeader,
  ProgressBar,
  RefreshButton,
  SearchInput,
  SkeletonRows,
  StatusBadge,
  Toolbar,
  UrgencyMeter,
} from "@/ui";
import {
  accessWindow,
  currentReviewStepIndex,
  formatDuration,
  isPendingRequest,
  requestResourceLabel,
  requestSubjectName,
  reviewSteps,
  shortName,
  statusMeta,
  tsToMillis,
  urgencyMeta,
} from "@/utils";
import { getUserClient } from "@/utils/client";

const ITEMS_PER_PAGE = 20;

const RequestRow = (props: {
  item: AccessP.Request;
  onCancel: (name: string) => void;
}) => {
  const { item } = props;
  useNow(30000);
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const subject = requestSubjectName(item);
  const pending = isPendingRequest(item);
  const Icon = resourceIcon(resource.kind);
  const steps = reviewSteps(item);
  const window = accessWindow(item);
  const active = status.group === "active" && !!window.endMs;

  return (
    <Card interactive className="relative">
      <div className="flex items-stretch">
        <Link
          to={`/user/requests/${item.metadata!.name}`}
          className="flex min-w-0 flex-1 items-center gap-3.5 px-4 py-3.5"
        >
          <IconTile tone={resource.kind === "Catalog" ? "violet" : "blue"}>
            <Icon size={16} strokeWidth={2.2} />
          </IconTile>

          <div className="min-w-0 flex-1">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate text-[0.87rem] font-bold text-slate-800">
                {resource.name || item.metadata!.name}
              </span>
              <Badge tone={resource.kind === "Catalog" ? "violet" : "slate"}>
                {resource.kind}
              </Badge>
            </div>

            <div className="mt-1 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-[0.7rem] font-semibold text-slate-400">
              <span className="sm:hidden">
                <StatusBadge status={item.status?.state?.status} />
              </span>
              <span>
                <TimeAgo rfc3339={item.status?.state?.createdAt} />
              </span>
              <span className="text-slate-200">•</span>
              <span>{formatDuration(item.spec?.duration)}</span>
              {subject && (
                <>
                  <span className="text-slate-200">•</span>
                  <span>
                    for{" "}
                    <span className="text-slate-500">{shortName(subject)}</span>
                  </span>
                </>
              )}
              {pending && steps.length > 0 && (
                <>
                  <span className="text-slate-200">•</span>
                  <span className="inline-flex items-center gap-1">
                    <Workflow size={11} strokeWidth={2.6} />
                    Step {Math.min(currentReviewStepIndex(item) + 1, steps.length)} of{" "}
                    {steps.length}
                  </span>
                </>
              )}
            </div>

            {active && (
              <div className="mt-2 flex items-center gap-2">
                <ProgressBar
                  value={window.progress}
                  tone={window.progress > 0.8 ? "amber" : "emerald"}
                  className="max-w-[220px]"
                />
                <span className="text-[0.68rem] font-bold text-emerald-600">
                  <Countdown date={new Date(window.endMs!)} suffix=" of access left" />
                </span>
              </div>
            )}

            {pending && (
              <div className="mt-2 inline-flex items-center gap-1 text-[0.68rem] font-bold text-amber-600">
                <Timer size={11} strokeWidth={2.6} />
                <Elapsed
                  rfc3339={item.status?.approvalStartAt ?? item.metadata?.createdAt}
                  suffix=" awaiting a decision"
                />
              </div>
            )}
          </div>

          <div className="hidden shrink-0 items-center gap-3 sm:flex">
            <UrgencyMeter urgency={item.spec?.urgency} />
            <StatusBadge status={item.status?.state?.status} withHint />
          </div>
        </Link>

        <div className="flex shrink-0 items-center gap-1 pr-3">
          {pending ? (
            <Tooltip label="Cancel this request">
              <Button
                size="compact-xs"
                variant="light"
                color="red"
                leftSection={<X size={11} strokeWidth={3} />}
                onClick={() => props.onCancel(item.metadata!.name)}
              >
                Cancel
              </Button>
            </Tooltip>
          ) : (
            <ChevronRight size={16} className="text-slate-300" />
          )}
        </div>
      </div>
    </Card>
  );
};

const Requests = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const search = searchParams.get("q") ?? "";
  const group = searchParams.get("group") ?? "all";
  const sort = searchParams.get("sort") ?? "recent";
  const [cancelName, setCancelName] = React.useState<string>();

  const qry = useQuery({
    queryKey: ["user", "listRequest", page],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { response } = await getUserClient().listRequest(
        AccessP.ListUserRequestOptions.create({
          common: { page: page - 1, itemsPerPage: ITEMS_PER_PAGE },
        }),
      );
      return response;
    },
    refetchInterval: (query) =>
      query.state.data?.items.some((item) => isPendingRequest(item)) ? 15000 : false,
  });

  const cancelMutation = useMutation({
    mutationFn: async (name: string) => {
      const { response } = await getUserClient().cancelRequest(
        AccessP.CancelRequestRequest.create({ requestRef: { name } }),
      );
      return response;
    },
    onSuccess: () => {
      toast.success("Request cancelled");
      setCancelName(undefined);
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to cancel request");
    },
  });

  const items = qry.data?.items ?? [];
  const counts = {
    all: items.length,
    pending: items.filter((item) => statusMeta(item.status?.state?.status).group === "pending").length,
    active: items.filter((item) => statusMeta(item.status?.state?.status).group === "active").length,
    closed: items.filter((item) => statusMeta(item.status?.state?.status).group === "closed").length,
  };

  const q = search.trim().toLowerCase();
  const visibleItems = items
    .filter((item) => {
      const resource = requestResourceLabel(item);
      const subject = requestSubjectName(item);
      const matchesQuery =
        !q ||
        resource.name.toLowerCase().includes(q) ||
        item.metadata!.name.toLowerCase().includes(q) ||
        subject.toLowerCase().includes(q);
      const matchesGroup =
        group === "all" || statusMeta(item.status?.state?.status).group === group;
      return matchesQuery && matchesGroup;
    })
    .sort((a, b) => {
      if (sort === "urgency") {
        const diff = urgencyMeta(b.spec?.urgency).level - urgencyMeta(a.spec?.urgency).level;
        if (diff !== 0) return diff;
      }
      const aAt = tsToMillis(a.status?.state?.createdAt ?? a.metadata?.createdAt) ?? 0;
      const bAt = tsToMillis(b.status?.state?.createdAt ?? b.metadata?.createdAt) ?? 0;
      return sort === "oldest" ? aAt - bAt : bAt - aAt;
    });

  const meta = qry.data?.listResponseMeta;
  const perPage = meta?.itemsPerPage || ITEMS_PER_PAGE;
  const totalPages = meta ? Math.max(1, Math.ceil(meta.totalCount / perPage)) : 1;

  const setParam = (key: string, value: string, fallback: string) =>
    setSearchParams((prev) => {
      if (value && value !== fallback) prev.set(key, value);
      else prev.delete(key);
      if (key !== "page") prev.delete("page");
      return prev;
    });

  const cancelTarget = items.find((item) => item.metadata!.name === cancelName);

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Access"
        title="My Requests"
        description="Track your just-in-time access requests and the access they granted."
        actions={
          <>
            <RefreshButton onClick={() => qry.refetch()} loading={qry.isFetching} />
            <Button
              variant="filled"
              color="dark"
              leftSection={<Plus size={14} strokeWidth={2.8} />}
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
          placeholder="Search by resource, request ID or subject..."
          ariaLabel="Search requests"
        />
        <SegmentedControl
          value={group}
          onChange={(value) => setParam("group", value, "all")}
          data={[
            { value: "all", label: `All ${counts.all ? `(${counts.all})` : ""}`.trim() },
            {
              value: "pending",
              label: `Pending ${counts.pending ? `(${counts.pending})` : ""}`.trim(),
            },
            {
              value: "active",
              label: `Active ${counts.active ? `(${counts.active})` : ""}`.trim(),
            },
            {
              value: "closed",
              label: `Closed ${counts.closed ? `(${counts.closed})` : ""}`.trim(),
            },
          ]}
        />
        <Select
          value={sort}
          onChange={(value) => setParam("sort", value ?? "recent", "recent")}
          aria-label="Sort requests"
          allowDeselect={false}
          className="sm:w-[150px]"
          data={[
            { value: "recent", label: "Newest first" },
            { value: "oldest", label: "Oldest first" },
            { value: "urgency", label: "Most urgent" },
          ]}
        />
      </Toolbar>

      {qry.isLoading ? (
        <SkeletonRows rows={5} />
      ) : qry.isError ? (
        <ErrorState
          title="Could not load your requests"
          onRetry={() => qry.refetch()}
        />
      ) : visibleItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={items.length ? <ServerCog size={20} strokeWidth={2} /> : <Inbox size={20} strokeWidth={2} />}
            title={items.length ? "No matching requests" : "No requests yet"}
            description={
              items.length
                ? "Try a different search term or switch the status filter."
                : "Request access to a Service or a Catalog and it will show up here."
            }
            action={
              !items.length ? (
                <Button
                  variant="filled"
                  color="dark"
                  leftSection={<Plus size={14} strokeWidth={2.8} />}
                  onClick={() => navigate("/user/new")}
                >
                  New Request
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <div className="flex flex-col gap-2">
          {visibleItems.map((item) => (
            <RequestRow
              key={item.metadata!.uid || item.metadata!.name}
              item={item}
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
            color="dark"
            size="sm"
            radius="md"
          />
          <span className="text-[0.68rem] font-semibold text-slate-400">
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
            <div className="flex items-center gap-2">
              <Layers size={13} className="text-slate-400" />
              <span className="truncate font-mono text-[0.74rem] font-semibold text-slate-600">
                {requestResourceLabel(cancelTarget).name || cancelTarget.metadata!.name}
              </span>
            </div>
          ) : undefined
        }
        confirmLabel="Cancel request"
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default Requests;
