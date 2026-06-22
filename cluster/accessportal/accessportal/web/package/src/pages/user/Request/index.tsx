import * as AccessP from "@/apis/accessv1/accessv1";
import { Button } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, Inbox, Plus, X } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";

import TimeAgo from "@/components/TimeAgo";
import {
  Badge,
  Card,
  EmptyState,
  Eyebrow,
  Loading,
  PageHeader,
} from "../../../ui";
import { requestResourceLabel, statusMeta, urgencyMeta } from "../../../utils";
import { getUserClient } from "../../../utils/client";

const RequestRow = (props: {
  item: AccessP.Request;
  onCancel: (name: string) => void;
  canceling: boolean;
}) => {
  const { item } = props;
  const resource = requestResourceLabel(item);
  const status = statusMeta(item.status?.state?.status);
  const urgency = urgencyMeta(item.spec?.urgency);
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

  const qry = useQuery({
    queryKey: ["user", "listRequest"],
    queryFn: async () => {
      const { response } = await getUserClient().listRequest(
        AccessP.ListUserRequestOptions.create({}),
      );
      return response;
    },
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
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
    },
    onError: (e) => {
      toast.error(e instanceof Error ? e.message : "Failed to cancel request");
    },
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

      {qry.isLoading ? (
        <Loading label="Loading your requests..." />
      ) : !qry.data || qry.data.items.length === 0 ? (
        <Card>
          <EmptyState
            icon={<Inbox size={20} strokeWidth={2} />}
            title="No requests yet"
            description="When you request access to a Service or Catalog, it will appear here."
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
        <div className="flex flex-col gap-2">
          {qry.data.items.map((item) => (
            <RequestRow
              key={item.metadata!.uid || item.metadata!.name}
              item={item}
              canceling={
                cancelMutation.isPending &&
                cancelMutation.variables === item.metadata!.name
              }
              onCancel={(name) => cancelMutation.mutate(name)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default Requests;
