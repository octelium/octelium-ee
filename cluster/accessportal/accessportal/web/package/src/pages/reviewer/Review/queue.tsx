import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { ClipboardCheck, Flame, Hourglass, SearchX } from "lucide-react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { twMerge } from "@/lib/utils";

import { useSubjectUser } from "@/components/Access/hooks";
import { resourceIcon } from "@/components/Access/icons";
import { Countdown, Elapsed, useNow } from "@/components/TimeAgo";
import {
  Avatar,
  Badge,
  Card,
  DocumentTitle,
  EmptyState,
  ErrorState,
  Eyebrow,
  IconTile,
  PageHeader,
  RefreshButton,
  SearchInput,
  SkeletonRows,
  StatTile,
  Toolbar,
  UrgencyBadge,
} from "@/ui";
import {
  currentReviewStepIndex,
  formatDuration,
  grantedDuration,
  isOnBehalfOf,
  requestResourceLabel,
  requestSubjectName,
  requesterName,
  reviewStepLabel,
  reviewStepTimeoutAt,
  reviewSteps,
  shortName,
  tsToDate,
  tsToMillis,
  urgencyMeta,
  waitingMillis,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";

const PAGE_SIZE = 500;
const MAX_PAGES = 100;
const STALE_MILLIS = 24 * 3600 * 1000;

const listAllReviewerRequests = async (
  signal?: AbortSignal,
): Promise<AccessP.Request[]> => {
  const items: AccessP.Request[] = [];

  for (let page = 0; page < MAX_PAGES; page++) {
    const { response } = await getReviewerClient().listRequest(
      AccessP.ListReviewerRequestOptions.create({
        common: { page, itemsPerPage: PAGE_SIZE },
      }),
      { abort: signal },
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore) break;
  }

  return items;
};

const QueueRow = (props: { item: AccessP.Request; returnTo: string; now: number }) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const urgency = urgencyMeta(item.spec?.urgency);
  const requester = requesterName(item);
  const onBehalf = isOnBehalfOf(item);
  const requesterQuery = useSubjectUser(requester);
  const steps = reviewSteps(item);
  const current = currentReviewStepIndex(item);
  const timeoutAt = reviewStepTimeoutAt(item);
  const deadline = tsToDate(item.spec?.deadline);
  const stale = waitingMillis(item, props.now) > STALE_MILLIS;
  const Icon = resourceIcon(resource.kind);
  const requesterLabel =
    requesterQuery.data?.displayName || shortName(requester) || "—";

  return (
    <Card interactive>
      <Link
        to={`/reviewer/requests/${item.metadata!.name}`}
        state={{ returnTo: props.returnTo }}
        className="flex min-w-0 items-start gap-3.5 rounded-xl px-4 py-3.5 outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400"
      >
        <IconTile>
          <Icon size={16} strokeWidth={2} />
        </IconTile>

        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="truncate text-body font-semibold text-slate-800">
              {resource.name || item.metadata!.name}
            </span>
            {resource.kind === "Catalog" && <Badge>Catalog</Badge>}
            {onBehalf && <Badge>On behalf</Badge>}
          </div>

          <div className="mt-1 flex min-w-0 flex-wrap items-center gap-x-1.5 gap-y-1 text-xs font-normal text-slate-500">
            <span className="inline-flex items-center gap-1.5">
              <Avatar
                src={requesterQuery.data?.picURL}
                name={requesterLabel}
                size="xs"
              />
              <span className="font-medium text-slate-700">{requesterLabel}</span>
            </span>
            {onBehalf && (
              <>
                <span aria-label="for">→</span>
                <span className="font-medium text-slate-700">
                  {shortName(requestSubjectName(item))}
                </span>
              </>
            )}
            <span aria-hidden className="text-slate-300">·</span>
            <span>{formatDuration(grantedDuration(item))}</span>
            {steps.length > 0 && (
              <>
                <span aria-hidden className="text-slate-300">·</span>
                <span>
                  Step {Math.min(current + 1, steps.length)} of {steps.length}:{" "}
                  {reviewStepLabel(steps[current], current)}
                </span>
              </>
            )}
          </div>

          {item.spec?.justification ? (
            <p className="mt-1.5 line-clamp-2 text-xs font-normal leading-5 text-slate-700 [overflow-wrap:anywhere]">
              {item.spec.justification}
            </p>
          ) : (
            <p className="mt-1.5 text-xs font-normal italic text-slate-500">
              No justification provided
            </p>
          )}

          <p className="mt-1.5 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-xs font-normal text-slate-500">
            <span className={twMerge(stale && "font-medium text-amber-700")}>
              Waiting{" "}
              <Elapsed rfc3339={item.status?.approvalStartAt ?? item.metadata?.createdAt} />
            </span>
            {timeoutAt && (
              <>
                <span aria-hidden className="text-slate-300">·</span>
                {timeoutAt.getTime() > props.now ? (
                  <span>
                    Step times out in <Countdown date={timeoutAt} suffix="" />
                  </span>
                ) : (
                  <span>Step timed out</span>
                )}
              </>
            )}
            {deadline && (
              <>
                <span aria-hidden className="text-slate-300">·</span>
                {deadline.getTime() > props.now ? (
                  <span>
                    Deadline in <Countdown date={deadline} suffix="" />
                  </span>
                ) : (
                  <span className="font-medium text-red-700">Deadline passed</span>
                )}
              </>
            )}
          </p>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-2">
          {urgency.level !== 3 && <UrgencyBadge urgency={item.spec?.urgency} />}
          <Button size="compact-xs" variant="default" component="span">
            Review
          </Button>
        </div>
      </Link>
    </Card>
  );
};

const rankDeadline = (item: AccessP.Request) =>
  tsToMillis(item.spec?.deadline) ?? Number.MAX_SAFE_INTEGER;

const Queue = () => {
  const location = useLocation();
  const now = useNow();
  const [searchParams, setSearchParams] = useSearchParams();
  const search = searchParams.get("q") ?? "";
  const sort = searchParams.get("sort") ?? "urgency";
  const focus = searchParams.get("focus") ?? "";

  const qry = useQuery({
    queryKey: ["reviewer", "listRequest"],
    queryFn: ({ signal }) => listAllReviewerRequests(signal),
    refetchInterval: 15000,
  });

  const allItems = qry.data ?? [];
  const urgentCount = allItems.filter(
    (item) => urgencyMeta(item.spec?.urgency).level >= 4,
  ).length;
  const staleCount = allItems.filter(
    (item) => waitingMillis(item, now) > STALE_MILLIS,
  ).length;

  const q = search.trim().toLowerCase();
  const items = allItems
    .filter((item) => {
      const resource = requestResourceLabel(item);
      const matchesQuery =
        !q ||
        resource.name.toLowerCase().includes(q) ||
        item.metadata!.name.toLowerCase().includes(q) ||
        requesterName(item).toLowerCase().includes(q) ||
        requestSubjectName(item).toLowerCase().includes(q) ||
        (item.spec?.justification ?? "").toLowerCase().includes(q);
      const matchesFocus =
        !focus ||
        (focus === "urgent" && urgencyMeta(item.spec?.urgency).level >= 4) ||
        (focus === "stale" && waitingMillis(item, now) > STALE_MILLIS);
      return matchesQuery && matchesFocus;
    })
    .sort((a, b) => {
      if (sort === "urgency") {
        const diff =
          urgencyMeta(b.spec?.urgency).level - urgencyMeta(a.spec?.urgency).level;
        if (diff !== 0) return diff;
        const deadlineDiff = rankDeadline(a) - rankDeadline(b);
        if (deadlineDiff !== 0) return deadlineDiff;
        return waitingMillis(b, now) - waitingMillis(a, now);
      }
      if (sort === "waiting") return waitingMillis(b, now) - waitingMillis(a, now);
      return waitingMillis(a, now) - waitingMillis(b, now);
    });

  const setParam = (key: string, value: string, fallback: string) =>
    setSearchParams(
      (prev) => {
        if (value && value !== fallback) prev.set(key, value);
        else prev.delete(key);
        return prev;
      },
      { replace: key === "q", preventScrollReset: true },
    );

  const toggleFocus = (value: string) =>
    setParam("focus", focus === value ? "" : value, "");

  const returnTo = `${location.pathname}${location.search}`;

  return (
    <div className="w-full">
      <DocumentTitle title={allItems.length ? `Review Queue (${allItems.length})` : "Review Queue"} />
      <PageHeader
        title="Review Queue"
        description="Access requests that are waiting for your decision right now."
        actions={<RefreshButton onClick={() => qry.refetch()} loading={qry.isFetching} />}
      />

      <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
        <StatTile
          label="In your queue"
          value={allItems.length}
          hint="Pending your review"
          icon={<ClipboardCheck size={13} strokeWidth={2.25} aria-hidden />}
          active={!focus}
          onClick={() => setParam("focus", "", "")}
        />
        <StatTile
          label="High urgency"
          value={urgentCount}
          hint="High and above"
          tone={urgentCount > 0 ? "amber" : undefined}
          icon={<Flame size={13} strokeWidth={2.25} aria-hidden />}
          active={focus === "urgent"}
          onClick={() => toggleFocus("urgent")}
        />
        <StatTile
          label="Waiting over a day"
          value={staleCount}
          hint="Needs attention"
          tone={staleCount > 0 ? "amber" : undefined}
          icon={<Hourglass size={13} strokeWidth={2.25} aria-hidden />}
          active={focus === "stale"}
          onClick={() => toggleFocus("stale")}
        />
      </div>

      <Toolbar>
        <SearchInput
          value={search}
          onChange={(value) => setParam("q", value, "")}
          placeholder="Search by resource, requester or justification"
          ariaLabel="Search the review queue"
        />
        <SegmentedControl
          value={sort}
          aria-label="Sort the review queue"
          onChange={(value) => setParam("sort", value, "urgency")}
          data={[
            { value: "urgency", label: "Most urgent" },
            { value: "waiting", label: "Longest waiting" },
            { value: "newest", label: "Newest" },
          ]}
        />
      </Toolbar>

      {qry.isLoading ? (
        <SkeletonRows rows={4} />
      ) : qry.isError && !qry.data ? (
        <ErrorState
          title="Could not load the review queue"
          error={qry.error}
          onRetry={() => qry.refetch()}
        />
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={
              allItems.length ? (
                <SearchX size={18} strokeWidth={2} />
              ) : (
                <ClipboardCheck size={18} strokeWidth={2} />
              )
            }
            title={allItems.length ? "No matching requests" : "Your queue is clear"}
            description={
              allItems.length
                ? "Try a different search term or clear the active filter."
                : "Requests assigned to you for review will show up here."
            }
            action={
              allItems.length ? (
                <Button
                  variant="default"
                  onClick={() =>
                    setSearchParams(
                      (prev) => {
                        prev.delete("q");
                        prev.delete("focus");
                        return prev;
                      },
                      { replace: true },
                    )
                  }
                >
                  Clear filters
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <>
          <Eyebrow className="mb-2 block">
            {items.length} request{items.length === 1 ? "" : "s"} awaiting you
          </Eyebrow>
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <QueueRow
                key={item.metadata!.uid || item.metadata!.name}
                item={item}
                returnTo={returnTo}
                now={now}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default Queue;
