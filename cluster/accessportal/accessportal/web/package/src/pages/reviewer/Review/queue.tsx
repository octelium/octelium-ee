import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, Pagination, SegmentedControl } from "@mantine/core";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import {
  ClipboardCheck,
  Flame,
  Hourglass,
  Timer,
  Workflow,
} from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";

import { useSubjectUser } from "@/components/Access/hooks";
import { resourceIcon } from "@/components/Access/icons";
import { Countdown, Elapsed } from "@/components/TimeAgo";
import {
  Avatar,
  Badge,
  Card,
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
  UrgencyMeter,
} from "@/ui";
import {
  currentReviewStepIndex,
  formatDuration,
  requestResourceLabel,
  requestSubjectName,
  requesterName,
  reviewStepTimeoutAt,
  reviewSteps,
  shortName,
  toneClasses,
  urgencyMeta,
  waitingMillis,
  waitingTone,
} from "@/utils";
import { getReviewerClient } from "@/utils/client";

const ITEMS_PER_PAGE = 20;

const STALE_MILLIS = 24 * 3600 * 1000;

const QueueRow = (props: { item: AccessP.Request }) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const urgency = urgencyMeta(item.spec?.urgency);
  const requester = requesterName(item);
  const subject = requestSubjectName(item);
  const onBehalf = !!subject && subject !== requester;
  const requesterQuery = useSubjectUser(requester);
  const steps = reviewSteps(item);
  const timeoutAt = reviewStepTimeoutAt(item);
  const waited = waitingMillis(item);
  const tone = waitingTone(waited);
  const Icon = resourceIcon(resource.kind);

  return (
    <Card interactive>
      <Link
        to={`/reviewer/requests/${item.metadata!.name}`}
        className="flex min-w-0 items-start gap-3.5 px-4 py-3.5"
      >
        <IconTile tone={resource.kind === "Catalog" ? "violet" : "blue"}>
          <Icon size={16} strokeWidth={2.2} />
        </IconTile>

        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="truncate text-[0.87rem] font-bold text-slate-800">
              {resource.name || item.metadata!.name}
            </span>
            <Badge tone={resource.kind === "Catalog" ? "violet" : "slate"}>
              {resource.kind}
            </Badge>
            {onBehalf && <Badge tone="amber">On behalf</Badge>}
          </div>

          <div className="mt-1.5 flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1 text-[0.7rem] font-semibold text-slate-400">
            <span className="inline-flex items-center gap-1.5">
              <Avatar
                src={requesterQuery.data?.picURL}
                name={requesterQuery.data?.displayName || shortName(requester)}
                size="xs"
              />
              <span className="text-slate-600">
                {requesterQuery.data?.displayName || shortName(requester) || "—"}
              </span>
            </span>
            {onBehalf && (
              <>
                <span className="text-slate-200">→</span>
                <span className="text-slate-600">{shortName(subject)}</span>
              </>
            )}
            <span className="text-slate-200">•</span>
            <span>{formatDuration(item.spec?.duration)}</span>
            {steps.length > 0 && (
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

          {item.spec?.justification && (
            <p className="mt-1.5 line-clamp-2 text-[0.74rem] font-medium leading-relaxed text-slate-500">
              {item.spec.justification}
            </p>
          )}

          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-[0.68rem] font-bold">
            <span className={`inline-flex items-center gap-1 ${toneClasses[tone].text}`}>
              <Hourglass size={11} strokeWidth={2.8} />
              <Elapsed
                rfc3339={item.status?.approvalStartAt ?? item.metadata?.createdAt}
                suffix=" waiting"
              />
            </span>
            {timeoutAt && (
              <span className="inline-flex items-center gap-1 text-amber-600">
                <Timer size={11} strokeWidth={2.8} />
                <Countdown
                  date={timeoutAt}
                  suffix=" until timeout"
                  endedLabel="Timed out"
                />
              </span>
            )}
          </div>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-2">
          <div className="flex items-center gap-2">
            <UrgencyMeter urgency={item.spec?.urgency} />
            <Badge tone={urgency.tone}>{urgency.label}</Badge>
          </div>
          <Button size="compact-xs" variant="default" component="span">
            Review
          </Button>
        </div>
      </Link>
    </Card>
  );
};

const Queue = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = Math.max(1, parseInt(searchParams.get("page") ?? "1", 10) || 1);
  const search = searchParams.get("q") ?? "";
  const sort = searchParams.get("sort") ?? "urgency";
  const focus = searchParams.get("focus") ?? "";

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
    refetchInterval: 15000,
  });

  const allItems = qry.data?.items ?? [];
  const urgentCount = allItems.filter(
    (item) => urgencyMeta(item.spec?.urgency).level >= 4,
  ).length;
  const staleCount = allItems.filter(
    (item) => waitingMillis(item) > STALE_MILLIS,
  ).length;

  const q = search.trim().toLowerCase();
  const items = allItems
    .filter((item) => {
      const resource = requestResourceLabel(item);
      const matchesQuery =
        !q ||
        resource.name.toLowerCase().includes(q) ||
        item.metadata!.name.toLowerCase().includes(q) ||
        shortName(requesterName(item)).toLowerCase().includes(q) ||
        requestSubjectName(item).toLowerCase().includes(q) ||
        (item.spec?.justification ?? "").toLowerCase().includes(q);
      const matchesFocus =
        !focus ||
        (focus === "urgent" && urgencyMeta(item.spec?.urgency).level >= 4) ||
        (focus === "stale" && waitingMillis(item) > STALE_MILLIS);
      return matchesQuery && matchesFocus;
    })
    .sort((a, b) => {
      if (sort === "urgency") {
        const diff =
          urgencyMeta(b.spec?.urgency).level - urgencyMeta(a.spec?.urgency).level;
        if (diff !== 0) return diff;
        return waitingMillis(b) - waitingMillis(a);
      }
      if (sort === "waiting") return waitingMillis(b) - waitingMillis(a);
      return waitingMillis(a) - waitingMillis(b);
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

  const toggleFocus = (value: string) =>
    setParam("focus", focus === value ? "" : value, "");

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Reviewer"
        title="Review Queue"
        description="Access requests that are waiting for your decision right now."
        actions={<RefreshButton onClick={() => qry.refetch()} loading={qry.isFetching} />}
      />

      <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
        <StatTile
          label="In your queue"
          value={allItems.length}
          hint="Pending your review"
          icon={<ClipboardCheck size={13} strokeWidth={2.4} />}
          active={!focus}
          onClick={() => setParam("focus", "", "")}
        />
        <StatTile
          label="High urgency"
          value={urgentCount}
          hint="High and above"
          tone="amber"
          icon={<Flame size={13} strokeWidth={2.4} />}
          active={focus === "urgent"}
          onClick={() => toggleFocus("urgent")}
        />
        <StatTile
          label="Waiting over a day"
          value={staleCount}
          hint="Needs attention"
          tone={staleCount > 0 ? "red" : "slate"}
          icon={<Hourglass size={13} strokeWidth={2.4} />}
          active={focus === "stale"}
          onClick={() => toggleFocus("stale")}
        />
      </div>

      <Toolbar>
        <SearchInput
          value={search}
          onChange={(value) => setParam("q", value, "")}
          placeholder="Search by resource, requester or justification..."
          ariaLabel="Search the review queue"
        />
        <SegmentedControl
          value={sort}
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
      ) : qry.isError ? (
        <ErrorState
          title="Could not load the review queue"
          onRetry={() => qry.refetch()}
        />
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ClipboardCheck size={20} strokeWidth={2} />}
            title={allItems.length ? "No matching requests" : "Your queue is clear"}
            description={
              allItems.length
                ? "Try a different search term or clear the active filter."
                : "Requests assigned to you for review will show up here."
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

export default Queue;
