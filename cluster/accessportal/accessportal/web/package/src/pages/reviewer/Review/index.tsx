import * as AccessP from "@/apis/accessv1/accessv1";
import { Pagination, SegmentedControl } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import {
  Check,
  ChevronRight,
  CircleDashed,
  History,
  ListChecks,
  Workflow,
  X,
} from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";

import TimeAgo from "@/components/TimeAgo";
import {
  Card,
  DecisionBadge,
  EmptyState,
  ErrorState,
  Eyebrow,
  IconTile,
  PageHeader,
  RefreshButton,
  SearchInput,
  SkeletonRows,
  Toolbar,
} from "@/ui";
import { decisionMeta, shortName, tsToMillis } from "@/utils";
import { getReviewerClient } from "@/utils/client";

const ITEMS_PER_PAGE = 20;

const ReviewRow = (props: { item: AccessP.Review }) => {
  const { item } = props;
  const decision = decisionMeta(item.spec?.decision);
  const request = shortName(item.status?.requestRef?.name);
  const revisions = item.status?.lastRevisions?.length ?? 0;

  return (
    <Card interactive>
      <Link
        to={`/reviewer/reviews/${item.metadata!.name}`}
        className="flex min-w-0 items-center gap-3.5 px-4 py-3.5"
      >
        <IconTile tone={decision.tone}>
          {item.spec?.decision === AccessP.Review_Spec_Decision.APPROVE ? (
            <Check size={16} strokeWidth={3} />
          ) : item.spec?.decision === AccessP.Review_Spec_Decision.REJECT ? (
            <X size={16} strokeWidth={3} />
          ) : (
            <CircleDashed size={16} strokeWidth={2.4} />
          )}
        </IconTile>

        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate font-mono text-[0.84rem] font-bold text-slate-800">
              {request || item.metadata!.name}
            </span>
          </div>

          <div className="mt-1 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-[0.7rem] font-semibold text-slate-400">
            <span>
              <TimeAgo rfc3339={item.status?.setAt} />
            </span>
            {typeof item.status?.stepIndex === "number" && (
              <>
                <span className="text-slate-200">•</span>
                <span className="inline-flex items-center gap-1">
                  <Workflow size={11} strokeWidth={2.6} />
                  Step {item.status.stepIndex + 1}
                </span>
              </>
            )}
            {revisions > 0 && (
              <>
                <span className="text-slate-200">•</span>
                <span className="inline-flex items-center gap-1">
                  <History size={11} strokeWidth={2.6} />
                  {revisions} revision{revisions === 1 ? "" : "s"}
                </span>
              </>
            )}
          </div>

          {item.spec?.justification && (
            <p className="mt-1.5 line-clamp-2 text-[0.74rem] font-medium leading-relaxed text-slate-500">
              {item.spec.justification}
            </p>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <DecisionBadge decision={item.spec?.decision} />
          <ChevronRight size={16} className="text-slate-300" />
        </div>
      </Link>
    </Card>
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
  const q = search.trim().toLowerCase();
  const items = allItems
    .filter((item) => {
      const request = shortName(item.status?.requestRef?.name).toLowerCase();
      const matchesQuery =
        !q ||
        request.includes(q) ||
        item.metadata!.name.toLowerCase().includes(q) ||
        (item.spec?.justification ?? "").toLowerCase().includes(q);
      const matchesDecision =
        decisionFilter === "all" ||
        (decisionFilter === "approved" &&
          item.spec?.decision === AccessP.Review_Spec_Decision.APPROVE) ||
        (decisionFilter === "rejected" &&
          item.spec?.decision === AccessP.Review_Spec_Decision.REJECT) ||
        (decisionFilter === "reset" &&
          item.spec?.decision === AccessP.Review_Spec_Decision.UNSET);
      return matchesQuery && matchesDecision;
    })
    .sort(
      (a, b) => (tsToMillis(b.status?.setAt) ?? 0) - (tsToMillis(a.status?.setAt) ?? 0),
    );

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

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Reviewer"
        title="My Reviews"
        description="Every decision you have made on an access request."
        actions={<RefreshButton onClick={() => qry.refetch()} loading={qry.isFetching} />}
      />

      <Toolbar>
        <SearchInput
          value={search}
          onChange={(value) => setParam("q", value, "")}
          placeholder="Search by request or justification..."
          ariaLabel="Search reviews"
        />
        <SegmentedControl
          value={decisionFilter}
          onChange={(value) => setParam("decision", value, "all")}
          data={[
            { value: "all", label: "All" },
            { value: "approved", label: "Approved" },
            { value: "rejected", label: "Rejected" },
            { value: "reset", label: "Reset" },
          ]}
        />
      </Toolbar>

      {qry.isLoading ? (
        <SkeletonRows rows={4} />
      ) : qry.isError ? (
        <ErrorState
          title="Could not load your reviews"
          onRetry={() => qry.refetch()}
        />
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ListChecks size={20} strokeWidth={2} />}
            title={allItems.length ? "No matching reviews" : "No reviews yet"}
            description={
              allItems.length
                ? "Try a different search term or decision filter."
                : "Once you approve or reject a request, your decision shows up here."
            }
          />
        </Card>
      ) : (
        <>
          <Eyebrow className="mb-2 block">
            {items.length} review{items.length === 1 ? "" : "s"}
          </Eyebrow>
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <ReviewRow
                key={item.metadata!.uid || item.metadata!.name}
                item={item}
              />
            ))}
          </div>
        </>
      )}

      {totalPages > 1 && (
        <div className="mt-6 flex justify-center">
          <Pagination
            value={page}
            total={totalPages}
            onChange={(value) => setParam("page", String(value), "1")}
            color="dark"
            size="sm"
            radius="md"
          />
        </div>
      )}
    </div>
  );
};

export default Reviews;
