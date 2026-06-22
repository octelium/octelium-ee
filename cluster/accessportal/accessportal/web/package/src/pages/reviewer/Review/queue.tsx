import * as AccessP from "@/apis/accessv1/accessv1";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, ClipboardCheck } from "lucide-react";
import { Link } from "react-router-dom";

import TimeAgo from "@/components/TimeAgo";
import {
  Badge,
  Card,
  EmptyState,
  Eyebrow,
  Loading,
  PageHeader,
} from "../../../ui";
import {
  requestResourceLabel,
  shortName,
  statusMeta,
  urgencyMeta,
} from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

const QueueRow = (props: { item: AccessP.Request }) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const urgency = urgencyMeta(item.spec?.urgency);
  const requester = shortName(item.status?.userRef?.name);

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
  const qry = useQuery({
    queryKey: ["reviewer", "listRequest"],
    queryFn: async () => {
      const { response } = await getReviewerClient().listRequest(
        AccessP.ListReviewerRequestOptions.create({}),
      );
      return response;
    },
  });

  const items = qry.data?.items ?? [];
  const pending = items.filter(
    (x) =>
      x.status?.state?.status === AccessP.Request_Status_State_Status.PENDING,
  );
  const others = items.filter(
    (x) =>
      x.status?.state?.status !== AccessP.Request_Status_State_Status.PENDING,
  );

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Reviewer"
        title="Review Queue"
        description="Requests awaiting your decision."
      />

      {qry.isLoading ? (
        <Loading label="Loading the review queue..." />
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ClipboardCheck size={20} strokeWidth={2} />}
            title="Nothing to review"
            description="When a request needs your review, it will show up here."
          />
        </Card>
      ) : (
        <div className="flex flex-col gap-6">
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
        </div>
      )}
    </div>
  );
};

export default Queue;
