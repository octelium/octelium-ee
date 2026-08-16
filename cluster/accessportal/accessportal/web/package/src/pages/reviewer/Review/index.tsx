import * as AccessP from "@/apis/accessv1/accessv1";
import { Pagination, Select } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ChevronRight, ListChecks, RefreshCw, Search } from "lucide-react";
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
import { decisionMeta, shortName } from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

const ITEMS_PER_PAGE = 20;

const ReviewRow = (props: { item: AccessP.Review }) => {
  const { item } = props;
  const decision = decisionMeta(item.spec?.decision);
  const request = shortName(item.status?.requestRef?.name);

  return (
    <Link to={`/reviewer/reviews/${item.metadata!.name}`}>
      <Card interactive className="px-4 py-3">
        <div className="flex items-center gap-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-[0.85rem] font-bold text-slate-800 truncate font-mono">
                {request || item.metadata!.name}
              </span>
            </div>
            {item.status?.setAt && (
              <div className="mt-1">
                <Eyebrow>
                  <TimeAgo rfc3339={item.status.setAt} />
                </Eyebrow>
              </div>
            )}
          </div>

          <Badge tone={decision.tone}>{decision.label}</Badge>
          <ChevronRight size={16} className="text-slate-300 shrink-0" />
        </div>
      </Card>
    </Link>
  );
};

const Reviews = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const search = searchParams.get("q") ?? "";
  const decisionFilter = searchParams.get("decision") ?? "all";

  const qry = useQuery({
    queryKey: ["reviewer", "listReview", page],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { response } = await getReviewerClient().listReview(
        AccessP.ListReviewerReviewOptions.create({
          common: { page: page - 1, itemsPerPage: ITEMS_PER_PAGE },
        }),
      );
      return response;
    },
  });

  const allItems = qry.data?.items ?? [];
  const items = allItems.filter((item) => {
    const request = shortName(item.status?.requestRef?.name).toLowerCase();
    const q = search.trim().toLowerCase();
    const matchesQuery = !q || request.includes(q) || item.metadata?.name.toLowerCase().includes(q);
    const decision = decisionMeta(item.spec?.decision).label.toLowerCase();
    return matchesQuery && (decisionFilter === "all" || decision === decisionFilter);
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
        eyebrow="Reviewer"
        title="My Reviews"
        description="Decisions you have made on access requests."
      />

      {qry.isLoading ? (
        <Loading label="Loading your reviews..." />
      ) : qry.isError ? (
        <ErrorState title="Could not load your reviews" onRetry={() => qry.refetch()} />
      ) : allItems.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ListChecks size={20} strokeWidth={2} />}
            title="No reviews yet"
            description="Once you approve or reject a request, it will appear here."
          />
        </Card>
      ) : (
        <>
          <div className="flex flex-col sm:flex-row gap-2 mb-4">
            <div className="relative flex-1">
              <Search size={13} strokeWidth={2.5} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none" />
              <input
                value={search}
                onChange={(event) => setFilter("q", event.target.value)}
                placeholder="Search request names..."
                aria-label="Search reviews"
                className="w-full pl-8 pr-3 h-9 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 transition-all placeholder:text-slate-400"
              />
            </div>
            <Select
              value={decisionFilter}
              onChange={(value) => setFilter("decision", value ?? "all")}
              aria-label="Filter reviews by decision"
              allowDeselect={false}
              className="min-w-[160px]"
              comboboxProps={{
                transitionProps: { transition: "pop", duration: 180 },
              }}
              data={[
                { value: "all", label: "All decisions" },
                { value: "approved", label: "Approved" },
                { value: "rejected", label: "Rejected" },
              ]}
            />
            <button
              type="button"
              onClick={() => qry.refetch()}
              className="inline-flex items-center justify-center gap-2 h-9 rounded-md border border-slate-200 bg-white px-3 text-[0.75rem] font-bold text-slate-600 hover:bg-slate-50"
            >
              <RefreshCw size={13} className={qry.isFetching ? "animate-spin" : ""} />
              Refresh
            </button>
          </div>
          {items.length === 0 ? (
            <Card>
              <EmptyState
                icon={<ListChecks size={20} strokeWidth={2} />}
                title="No matching reviews"
                description="Try changing the search or decision filter."
              />
            </Card>
          ) : (
            <div className="flex flex-col gap-2">
              {items.map((item) => (
                <ReviewRow
                  key={item.metadata!.uid || item.metadata!.name}
                  item={item}
                />
              ))}
            </div>
          )}

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

export default Reviews;
