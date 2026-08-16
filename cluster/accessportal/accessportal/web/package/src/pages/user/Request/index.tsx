import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, Pagination, Select } from "@mantine/core";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { ChevronRight, Inbox, Plus, RefreshCw, Search, X } from "lucide-react";
import * as React from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { toast } from "sonner";

import TimeAgo, { TimeRemaining } from "@/components/TimeAgo";
import {
  Badge,
  Card,
  ConfirmDialog,
  EmptyState,
  Eyebrow,
  ErrorState,
  Loading,
  PageHeader,
} from "../../../ui";
import {
  requestResourceLabel,
  requestSubjectName,
  durationToParts,
  statusMeta,
  urgencyMeta,
} from "../../../utils";
import { getUserClient } from "../../../utils/client";

const ITEMS_PER_PAGE = 20;

const RequestRow = (props: {
  item: AccessP.Request;
  onCancel: (name: string) => void;
  canceling: boolean;
}) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const urgency = urgencyMeta(item.spec?.urgency);
  const subject = requestSubjectName(item);
  const duration = durationToParts(item.spec?.duration);
  const isPending =
    item.status?.state?.status === AccessP.Request_Status_State_Status.PENDING;

  return (
    <Card interactive className="px-4 py-3">
      <div className="flex items-center gap-4">
        <Link
          to={`/user/requests/${item.metadata!.name}`}
          className="flex-1 min-w-0 flex items-center gap-4"
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-[0.85rem] font-bold text-slate-800 truncate">
                {resource.name || item.metadata!.name}
              </span>
              <Badge tone="slate">{resource.kind}</Badge>
            </div>
            <div className="flex items-center gap-2 mt-1">
              <Eyebrow>
                {item.status?.state?.createdAt ? (
                  <TimeAgo rfc3339={item.status.state.createdAt} />
                ) : (
                  "—"
                )}
              </Eyebrow>
              {subject && (
                <Eyebrow>
                  for <span className="text-slate-500">{subject}</span>
                </Eyebrow>
              )}
              {item.status?.accessEndsAt && (
                <Eyebrow>
                  <TimeRemaining rfc3339={item.status.accessEndsAt} />
                </Eyebrow>
              )}
              {!item.status?.accessEndsAt && item.spec?.duration && (
                <Eyebrow>
                  {duration.amount} {duration.unit}
                </Eyebrow>
              )}
            </div>
          </div>

          <Badge tone={urgency.tone}>{urgency.label}</Badge>
          <Badge tone={status.tone}>{status.label}</Badge>
        </Link>

        {isPending ? (
          <Button
            size="compact-xs"
            variant="outline"
            color="red"
            leftSection={<X size={11} />}
            loading={props.canceling}
            onClick={() => props.onCancel(item.metadata!.name)}
          >
            Cancel
          </Button>
        ) : (
          <ChevronRight size={16} className="text-slate-300 shrink-0" />
        )}
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
  const statusFilter = searchParams.get("status") ?? "all";
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
      query.state.data?.items.some(
        (item) =>
          item.status?.state?.status ===
          AccessP.Request_Status_State_Status.PENDING,
      )
        ? 15000
        : false,
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

  const visibleItems = (qry.data?.items ?? []).filter((item) => {
    const resource = requestResourceLabel(item);
    const subject = requestSubjectName(item);
    const q = search.trim().toLowerCase();
    const matchesQuery =
      !q ||
      resource.name.toLowerCase().includes(q) ||
      item.metadata?.name.toLowerCase().includes(q) ||
      subject.toLowerCase().includes(q);
    const matchesStatus =
      statusFilter === "all" ||
      statusMeta(item.status?.state?.status).label.toLowerCase() === statusFilter;
    return matchesQuery && matchesStatus;
  });

  const meta = qry.data?.listResponseMeta;
  const perPage = meta?.itemsPerPage || ITEMS_PER_PAGE;
  const totalPages = meta
    ? Math.max(1, Math.ceil(meta.totalCount / perPage))
    : 1;

  const setPage = (v: number) =>
    setSearchParams((prev) => {
      prev.set("page", String(v));
      return prev;
    });

  const setFilter = (key: string, value: string) =>
    setSearchParams((prev) => {
      if (value && value !== "all") prev.set(key, value);
      else prev.delete(key);
      if (key !== "page") prev.delete("page");
      return prev;
    });

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Access"
        title="My Requests"
        description="Track the status of your access requests and create new ones."
        actions={
          <Button
            variant="filled"
            color="dark"
            leftSection={<Plus size={14} strokeWidth={2.5} />}
            onClick={() => navigate("/user/new")}
          >
            New Request
          </Button>
        }
      />

      <div className="flex flex-col sm:flex-row gap-2 mb-4">
        <div className="relative flex-1">
          <Search
            size={13}
            strokeWidth={2.5}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
          />
          <input
            value={search}
            onChange={(event) => setFilter("q", event.target.value)}
            placeholder="Search resources or subjects..."
            aria-label="Search requests"
            className="w-full pl-8 pr-3 h-9 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 transition-all placeholder:text-slate-400"
          />
        </div>
        <Select
          value={statusFilter}
          onChange={(value) => setFilter("status", value ?? "all")}
          aria-label="Filter requests by status"
          allowDeselect={false}
          className="min-w-[160px]"
          comboboxProps={{
            transitionProps: { transition: "pop", duration: 180 },
          }}
          data={[
            { value: "all", label: "All statuses" },
            { value: "pending", label: "Pending" },
            { value: "approved", label: "Approved" },
            { value: "rejected", label: "Rejected" },
            { value: "revoked", label: "Revoked" },
            { value: "expired", label: "Expired" },
            { value: "cancelled", label: "Cancelled" },
          ]}
        />
        <Button
          variant="default"
          size="sm"
          leftSection={<RefreshCw size={13} className={qry.isFetching ? "animate-spin" : ""} />}
          onClick={() => qry.refetch()}
        >
          Refresh
        </Button>
      </div>

      {qry.isLoading ? (
        <Loading label="Loading your requests..." />
      ) : qry.isError ? (
        <ErrorState title="Could not load your requests" onRetry={() => qry.refetch()} />
      ) : visibleItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Inbox size={20} strokeWidth={2} />}
            title={qry.data?.items.length ? "No matching requests" : "No requests yet"}
            description={
              qry.data?.items.length
                ? "Try changing the search or status filter."
                : "When you request access to a Service or Catalog, it will appear here."
            }
            action={
              <Button
                variant="filled"
                color="dark"
                leftSection={<Plus size={14} strokeWidth={2.5} />}
                onClick={() => navigate("/user/new")}
              >
                New Request
              </Button>
            }
          />
        </Card>
      ) : (
        <>
          <div className="flex flex-col gap-2">
            {visibleItems.map((item) => (
              <RequestRow
                key={item.metadata!.uid || item.metadata!.name}
                item={item}
                canceling={
                  cancelMutation.isPending &&
                  cancelMutation.variables === item.metadata!.name
                }
                onCancel={(name) => setCancelName(name)}
              />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex justify-center mt-6">
              <Pagination
                value={page}
                total={totalPages}
                onChange={setPage}
                color="dark"
                size="sm"
                radius="md"
              />
            </div>
          )}
        </>
      )}

      <ConfirmDialog
        opened={!!cancelName}
        onClose={() => setCancelName(undefined)}
        onConfirm={() => {
          if (cancelName) cancelMutation.mutate(cancelName);
        }}
        title="Cancel request?"
        description="This request will stop progressing and cannot be restored."
        confirmLabel="Cancel request"
        loading={cancelMutation.isPending}
      />
    </div>
  );
};

export default Requests;
