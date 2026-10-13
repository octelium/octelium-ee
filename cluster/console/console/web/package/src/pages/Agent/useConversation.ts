import {
  isTerminalRunStatus,
  type AgentEvent,
  type ApprovalDecision,
  type Conversation,
  type FileInfo,
  type ListConversationsResponse,
  type Message,
  type RunContext,
  type RunInput,
} from "@/apis/consoleagent/protocol";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import * as React from "react";
import { toast } from "sonner";
import { AgentClient } from "./client";
import { conversationsKey as getConversationsKey } from "./environment";
import { applyEvent, emptyChatState, type ChatState } from "./reducer";
import { messageText } from "./utils";

interface Optimistic {
  text: string;
  attachments: FileInfo[];
  context?: RunContext;
  createdAt: string;
}

export interface SendInput {
  text: string;
  files: FileInfo[];
  context?: RunContext;
}

export const useConversation = (props: {
  client: AgentClient;
  conversationId?: string;
  onCreated: (conversation: Conversation) => void;
}) => {
  const { client, conversationId: selected, onCreated } = props;
  const queryClient = useQueryClient();

  const [chat, setChat] = React.useState<ChatState>(emptyChatState);
  const [loading, setLoading] = React.useState(() => !!selected);
  const [sending, setSending] = React.useState(false);
  const [optimistic, setOptimistic] = React.useState<Optimistic>();
  const loadedRef = React.useRef<string | undefined>(undefined);

  const conversationsKey = React.useMemo(
    () => getConversationsKey(client.baseUrl),
    [client.baseUrl],
  );
  const conversationsQuery = useQuery({
    queryKey: conversationsKey,
    queryFn: () => client.listConversations(),
  });
  const conversations = conversationsQuery.data?.items;
  const conversation = conversations?.find((c) => c.id === selected);

  const onConversationUpdated = React.useCallback(
    (updated: Conversation) => {
      queryClient.setQueryData<ListConversationsResponse>(
        conversationsKey,
        (data) => {
          const items = (data?.items ?? []).filter(
            (itm) => itm.id !== updated.id,
          );
          return { items: [updated, ...items] };
        },
      );
    },
    [queryClient, conversationsKey],
  );

  const load = React.useCallback(
    async (id: string) => {
      setLoading(true);
      try {
        const detail = await client.getConversation(id);
        if (loadedRef.current !== id) return;
        const messages = [...detail.messages];
        const active = detail.activeRun;
        if (
          active?.message &&
          !messages.some((m) => m.id === active.message!.id)
        ) {
          messages.push(active.message);
        }
        setChat({
          messages,
          run: active?.run,
          lastSeq: active?.run.lastSeq ?? 0,
        });
        setOptimistic(undefined);
      } catch (err) {
        if (loadedRef.current === id) {
          toast.error((err as Error).message);
          setChat(emptyChatState);
        }
      } finally {
        if (loadedRef.current === id) setLoading(false);
      }
    },
    [client],
  );

  React.useEffect(() => {
    if (loadedRef.current === selected) return;
    loadedRef.current = selected;
    setOptimistic(undefined);
    setChat(emptyChatState);
    if (selected) {
      void load(selected);
    } else {
      setLoading(false);
    }
  }, [selected, load]);

  const switching = loadedRef.current !== selected;
  const run = switching ? undefined : chat.run;
  const isActive = !!run && !isTerminalRunStatus(run.status);
  const runId = run?.id;
  const runConversationId = run?.conversationId;

  const lastSeqRef = React.useRef(chat.lastSeq);
  React.useEffect(() => {
    lastSeqRef.current = chat.lastSeq;
  }, [chat.lastSeq]);

  React.useEffect(() => {
    if (!isActive || !runId || !runConversationId) return;
    const afterSeq = lastSeqRef.current;
    let queue: AgentEvent[] = [];
    let frame: number | undefined;

    const flush = () => {
      if (frame !== undefined) {
        cancelAnimationFrame(frame);
        frame = undefined;
      }
      if (queue.length === 0) return;
      const events = queue;
      queue = [];
      setChat((state) =>
        events.reduce(
          (current, event) =>
            loadedRef.current === event.conversationId
              ? applyEvent(current, event)
              : current,
          state,
        ),
      );
    };

    const unsubscribe = client.subscribe(
      runId,
      afterSeq,
      (event: AgentEvent) => {
        if (event.type === "conversation.updated") {
          onConversationUpdated(event.conversation);
        }
        queue.push(event);
        frame ??= requestAnimationFrame(flush);
      },
      () => {
        flush();
        void queryClient.invalidateQueries({ queryKey: conversationsKey });
        if (loadedRef.current === runConversationId) {
          void load(runConversationId);
        }
      },
    );

    return () => {
      unsubscribe();
      flush();
    };
  }, [
    client,
    isActive,
    runId,
    runConversationId,
    onConversationUpdated,
    queryClient,
    conversationsKey,
    load,
  ]);

  const userMessageArrived =
    !!run && chat.messages.some((m) => m.id === run.userMessageId);
  const showOptimistic = !!optimistic && !userMessageArrived;

  React.useEffect(() => {
    if (userMessageArrived) setOptimistic(undefined);
  }, [userMessageArrived]);

  const showPendingAssistant =
    (sending || isActive) &&
    !chat.messages.some(
      (m) => m.role === "assistant" && m.id === run?.assistantMessageId,
    );
  const busy = loading || switching;
  const isEmpty =
    chat.messages.length === 0 && !showOptimistic && !busy && !sending;

  const send = async (input: SendInput): Promise<boolean> => {
    const text = input.text.trim();
    if ((!text && input.files.length === 0) || isActive || sending) {
      return false;
    }

    const runInput: RunInput = {
      text,
      attachments:
        input.files.length > 0 ? input.files.map((a) => a.id) : undefined,
      context: input.context,
    };

    setSending(true);
    setOptimistic({
      text,
      attachments: input.files,
      context: input.context,
      createdAt: new Date().toISOString(),
    });

    try {
      if (selected) {
        const next = await client.startRun(selected, runInput);
        if (loadedRef.current === selected) {
          setChat((state) => ({ ...state, run: next, lastSeq: 0 }));
        }
      } else {
        const res = await client.createConversation(runInput);
        loadedRef.current = res.conversation.id;
        setChat({ messages: [], run: res.run, lastSeq: 0 });
        onConversationUpdated(res.conversation);
        onCreated(res.conversation);
      }
      return true;
    } catch (err) {
      toast.error((err as Error).message);
      setOptimistic(undefined);
      return false;
    } finally {
      setSending(false);
    }
  };

  const stop = async () => {
    if (!run) return;
    try {
      await client.cancelRun(run.id);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const decide = React.useCallback(
    async (
      decisionRunId: string,
      approvalId: string,
      decision: ApprovalDecision,
      reason?: string,
    ) => {
      try {
        await client.decideApproval(
          decisionRunId,
          approvalId,
          decision,
          reason,
        );
      } catch (err) {
        toast.error((err as Error).message);
      }
    },
    [client],
  );

  const lastMessage = chat.messages.at(-1);
  const lastAssistantID = [...chat.messages]
    .reverse()
    .find((m) => m.role === "assistant")?.id;

  const retryMessageID =
    !isActive &&
    !sending &&
    lastMessage?.role === "assistant" &&
    lastMessage.status !== "completed" &&
    lastMessage.status !== "streaming"
      ? lastMessage.id
      : undefined;
  const retryInput = React.useMemo<SendInput | undefined>(() => {
    if (!retryMessageID) return undefined;
    const user = [...chat.messages].reverse().find((m) => m.role === "user");
    if (!user) return undefined;
    return {
      text: messageText(user),
      files: user.attachments ?? [],
      context: user.context,
    };
  }, [retryMessageID, chat.messages]);

  const optimisticMessage: Message | undefined = optimistic && {
    id: "optimistic",
    conversationId: selected ?? "",
    role: "user",
    status: "completed",
    createdAt: optimistic.createdAt,
    attachments: optimistic.attachments,
    context: optimistic.context,
    blocks: [
      {
        id: "optimistic",
        type: "markdown",
        text: optimistic.text,
        createdAt: optimistic.createdAt,
      },
    ],
  };

  return {
    chat,
    conversations,
    conversationsLoading: conversationsQuery.isPending,
    conversationsKey,
    conversation,
    onConversationUpdated,
    loading,
    sending,
    switching,
    busy,
    isEmpty,
    run,
    isActive,
    awaitingApproval: run?.status === "awaiting_approval",
    showOptimistic,
    showPendingAssistant,
    optimisticMessage,
    lastMessage,
    lastAssistantID,
    retryMessageID,
    retryInput,
    send,
    stop,
    decide,
  };
};
