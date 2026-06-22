import * as AccessP from "@/apis/accessv1/accessv1";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, ListChecks } from "lucide-react";
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
import { decisionMeta, shortName } from "../../../utils";
import { getReviewerClient } from "../../../utils/client";

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
  const qry = useQuery({
    queryKey: ["reviewer", "listReview"],
    queryFn: async () => {
      const { response } = await getReviewerClient().listReview(
        AccessP.ListReviewerReviewOptions.create({}),
      );
      return response;
    },
  });

  const items = qry.data?.items ?? [];

  return (
    <div className="w-full">
      <PageHeader
        eyebrow="Reviewer"
        title="My Reviews"
        description="Decisions you have made on access requests."
      />

      {qry.isLoading ? (
        <Loading label="Loading your reviews..." />
      ) : items.length === 0 ? (
        <Card>
          <EmptyState
            icon={<ListChecks size={20} strokeWidth={2} />}
            title="No reviews yet"
            description="Once you approve or reject a request, it will appear here."
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
    </div>
  );
};

export default Reviews;
