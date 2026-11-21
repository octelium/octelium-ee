import * as AccessP from "@/apis/accessv1/accessv1";
import { Button, Textarea } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { MessageCircleQuestionMark, Send } from "lucide-react";
import * as React from "react";

import TimeAgo from "@/components/TimeAgo";
import { LengthCounter, SectionCard } from "@/ui";
import {
  MAX_MESSAGE_LENGTH,
  openInformationRequest,
  shortName,
  textLength,
} from "@/utils";
import { getUserClient } from "@/utils/client";

const InformationRequestCard = (props: {
  request: AccessP.Request;
  canRespond?: boolean;
}) => {
  const queryClient = useQueryClient();
  const [response, setResponse] = React.useState("");
  const items = props.request.status?.review?.informationRequests ?? [];
  const open = openInformationRequest(props.request);

  const mutation = useMutation({
    mutationFn: async () => {
      const { response: ret } = await getUserClient().respondToRequest(
        AccessP.RespondToRequestRequest.create({
          requestRef: { name: props.request.metadata!.name },
          response: response.trim(),
        }),
      );
      return ret;
    },
    onSuccess: () => {
      notifications.show({
        color: "green",
        message: "Response sent to the reviewers",
      });
      setResponse("");
      queryClient.invalidateQueries({
        queryKey: ["user", "getRequest", props.request.metadata!.name],
      });
      queryClient.invalidateQueries({ queryKey: ["user", "listRequest"] });
      queryClient.invalidateQueries({ queryKey: ["user", "getInbox"] });
    },
    onError: (e) => {
      notifications.show({
        color: "red",
        message: e instanceof Error ? e.message : "Failed to send the response",
      });
    },
  });

  if (items.length === 0) return null;

  const length = textLength(response);
  const tooLong = length > MAX_MESSAGE_LENGTH;

  return (
    <SectionCard
      title="Questions from reviewers"
      description={
        open
          ? "A reviewer needs more information before deciding"
          : "Questions the reviewers asked about this request"
      }
      icon={<MessageCircleQuestionMark size={14} strokeWidth={2.25} />}
    >
      <div className="flex flex-col gap-4">
        <ol className="flex flex-col gap-3">
          {items.map((item, index) => (
            <li
              key={index}
              className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5"
            >
              <div className="text-xs font-normal text-slate-500">
                <span className="font-medium text-slate-700">
                  {item.userRef?.name
                    ? shortName(item.userRef.name)
                    : "A reviewer"}
                </span>{" "}
                asked <TimeAgo rfc3339={item.requestedAt} />
              </div>
              <p className="text-body font-normal text-slate-800 [overflow-wrap:anywhere]">
                {item.message}
              </p>
              {item.respondedAt ? (
                <div className="border-t border-slate-200 pt-2">
                  <div className="text-xs font-normal text-slate-500">
                    <span className="font-medium text-slate-700">
                      {item.responderRef?.name
                        ? shortName(item.responderRef.name)
                        : "The requester"}
                    </span>{" "}
                    responded <TimeAgo rfc3339={item.respondedAt} />
                  </div>
                  <p className="mt-1 text-body font-normal text-slate-800 [overflow-wrap:anywhere]">
                    {item.response}
                  </p>
                </div>
              ) : (
                <p className="text-xs font-normal text-amber-700">
                  Awaiting a response
                </p>
              )}
            </li>
          ))}
        </ol>

        {open && props.canRespond && (
          <form
            className="flex flex-col gap-3"
            onSubmit={(event) => {
              event.preventDefault();
              if (response.trim() && !tooLong) mutation.mutate();
            }}
          >
            <Textarea
              label={
                <span className="flex w-full items-baseline justify-between gap-3">
                  Your response
                  <LengthCounter length={length} max={MAX_MESSAGE_LENGTH} />
                </span>
              }
              error={tooLong ? "The response is too long" : undefined}
              autosize
              minRows={2}
              maxRows={6}
              value={response}
              onChange={(event) => setResponse(event.currentTarget.value)}
              styles={{ label: { display: "flex", width: "100%" } }}
            />
            <div className="flex justify-end">
              <Button
                type="submit"
                variant="filled"
                color="ink"
                leftSection={<Send size={13} strokeWidth={2.25} />}
                loading={mutation.isPending}
                disabled={!response.trim() || tooLong}
              >
                Send response
              </Button>
            </div>
          </form>
        )}
      </div>
    </SectionCard>
  );
};

export default InformationRequestCard;
