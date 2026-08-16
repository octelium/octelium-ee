import * as AccessP from "@/apis/accessv1/accessv1";
import { Pagination } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ChevronRight, ClipboardCheck, RefreshCw, Search } from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";

import TimeAgo from "@/components/TimeAgo";
import {
  Badge,
  Card,
  EmptyState,
  Eyebrow,
  ErrorState,
  Loading,
  PageHeader,
} from "../../../ui";
import {
  requestResourceLabel,
  requestSubjectName,
  shortName,
  statusMeta,
  urgencyMeta,
} from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

const ITEMS_PER_PAGE = 20;

const QueueRow = (props: { item: AccessP.Request }) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const urgency = urgencyMeta(item.spec?.urgency);
  const requester = shortName(item.status?.userRef?.name);
  const subject = requestSubjectName(item);

  return (
    <Link to={`/reviewer/requests/${item.metadata!.name}`}>
      <Card interactive className="px-4 py-3">
        <div className="flex items-center gap-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-[0.85rem] font-bold text-slate-800 truncate">
                {resource.name || item.metadata!.name}
              </span>
              <Badge tone="slate">{resource.kind}</Badge>
            </div>
            <div className="flex items-center gap-2 mt-1">
              {requester && (
                <Eyebrow>
                  by <span className="text-slate-500">{requester}</span>
                </Eyebrow>
              )}
              {subject && (
                <Eyebrow>
                  for <span className="text-slate-500">{subject}</span>
                </Eyebrow>
              )}
              {item.status?.state?.createdAt && (
                <Eyebrow>
                  <TimeAgo rfc3339={item.status.state.createdAt} />
                </Eyebrow>
              )}
            </div>
          </div>

          <Badge tone={urgency.tone}>{urgency.label}</Badge>
          <Badge tone={status.tone}>{status.label}</Badge>
          <ChevronRight size={16} className="text-slate-300 shrink-0" />
        </div>
      </Card>
    </Link>
  );
};

const Queue = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const search = searchParams.get("q") ?? "";
  const queueFilter = searchParams.get("filter") ?? "pending";

  const qry = useQuery({
    queryKey: ["reviewer", "listRequest", page],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { response } = await getReviewerClient().listRequest(
        AccessP.ListReviewerRequestOptions.create({
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

  const allItems = qry.data?.items ?? [];
  const filteredItems = allItems.filter((item) => {
    const resource = requestResourceLabel(item);
    const subject = requestSubjectName(item);
    const q = search.trim().toLowerCase();
    const matchesQuery =
      !q ||
      resource.name.toLowerCase().includes(q) ||
      item.metadata?.name.toLowerCase().includes(q) ||
      shortName(item.status?.userRef?.name).toLowerCase().includes(q) ||
      subject.toLowerCase().includes(q);
    const isPending =
      item.status?.state?.status === AccessP.Request_Status_State_Status.PENDING;
    return matchesQuery &&
      (queueFilter === "all" || (queueFilter === "pending" ? isPending : !isPending));
  });
  const items = filteredItems;
  const pending = items
    .filter(
    (x) =>
      x.status?.state?.status === AccessP.Request_Status_State_Status.PENDING,
    )
    .sort((a, b) => (b.spec?.urgency ?? 0) - (a.spec?.urgency ?? 0));
  const others = items.filter(
    (x) =>
      x.status?.state?.status !== AccessP.Request_Status_State_Status.PENDING,
  );

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
      if (value && value !== "pending") prev.set(key, value);
      else if (key === "filter") prev.delete(key);
      else prev.delete(key);
      if (key !== "page") prev.delete("page");
      return prev;
    });

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Reviewer"
        title="Review Queue"
        description="Prioritize and decide access requests awaiting your review."
      />

      <div className="flex flex-col sm:flex-row gap-2 mb-4">
        <div className="relative flex-1">
          <Search size={13} strokeWidth={2.5} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
          <input
            value={search}
            onChange={(event) => setFilter("q", event.target.value)}
            placeholder="Search resources or requesters..."
            aria-label="Search review queue"
            className="w-full pl-8 pr-3 h-9 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 transition-all placeholder:text-slate-400"
          />
        </div>
        <select
          value={queueFilter}
          onChange={(event) => setFilter("filter", event.target.value)}
          aria-label="Filter review queue"
          className="h-9 rounded-md border border-slate-200 bg-white px-3 text-[0.75rem] font-bold text-slate-600 outline-none focus:border-slate-400"
        >
          <option value="pending">Awaiting review</option>
          <option value="decided">Recently decided</option>
          <option value="all">All requests</option>
        </select>
        <button
          type="button"
          onClick={() => qry.refetch()}
          className="inline-flex items-center justify-center gap-2 h-9 rounded-md border border-slate-200 bg-white px-3 text-[0.75rem] font-bold text-slate-600 hover:bg-slate-50"
        >
          <RefreshCw size={13} className={qry.isFetching ? "animate-spin" : ""} />
          Refresh
        </button>
      </div>

      {qry.isLoading ? (
        <Loading label="Loading the review queue..." />
      ) : qry.isError ? (
        <ErrorState title="Could not load the review queue" onRetry={() => qry.refetch()} />
      ) : allItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ClipboardCheck size={20} strokeWidth={2} />}
            title="Nothing to review"
            description="When a request needs your review, it will show up here."
          />
        </Card>
      ) : (
        <>
          {items.length === 0 ? (
            <Card>
              <EmptyState
                icon={<ClipboardCheck size={20} strokeWidth={2} />}
                title="No matching requests"
                description="Try changing the search or queue filter."
              />
            </Card>
          ) : <div className="flex flex-col gap-6">
            {pending.length > 0 && (
              <div className="flex flex-col gap-2">
                <Eyebrow>Awaiting review ({pending.length})</Eyebrow>
                {pending.map((item) => (
                  <QueueRow
                    key={item.metadata!.uid || item.metadata!.name}
                    item={item}
                  />
                ))}
              </div>
            )}

            {others.length > 0 && (
              <div className="flex flex-col gap-2">
                <Eyebrow>Recently decided</Eyebrow>
                {others.map((item) => (
                  <QueueRow
                    key={item.metadata!.uid || item.metadata!.name}
                    item={item}
                  />
                ))}
              </div>
            )}
          </div>}

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
    </div>
  );
};

export default Queue;
