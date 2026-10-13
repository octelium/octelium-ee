import type { Message } from "@/apis/consoleagent/protocol";
import * as React from "react";

export const useTranscript = (props: {
  mounted: boolean;
  awaitingApproval: boolean;
  messages: Message[];
}) => {
  const scrollRef = React.useRef<HTMLDivElement>(null);
  const contentRef = React.useRef<HTMLDivElement>(null);
  const stickRef = React.useRef(true);
  const [atBottom, setAtBottom] = React.useState(true);
  const [scrolled, setScrolled] = React.useState(false);
  const [approvalVisible, setApprovalVisible] = React.useState(false);

  React.useEffect(() => {
    const el = scrollRef.current;
    const content = contentRef.current;
    if (!el || !content) return;
    const observer = new ResizeObserver(() => {
      if (stickRef.current) {
        el.scrollTop = el.scrollHeight;
      }
    });
    observer.observe(content);
    return () => observer.disconnect();
  }, [props.mounted]);

  React.useEffect(() => {
    const root = scrollRef.current;
    if (!props.awaitingApproval || !root) return;
    const targets = root.querySelectorAll("[data-approval-pending]");
    if (targets.length === 0) {
      setApprovalVisible(false);
      return;
    }
    const visible = new Set<Element>();
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            visible.add(entry.target);
          } else {
            visible.delete(entry.target);
          }
        }
        setApprovalVisible(visible.size > 0);
      },
      { root, threshold: 0.35 },
    );
    targets.forEach((target) => observer.observe(target));
    return () => observer.disconnect();
  }, [props.awaitingApproval, props.messages]);

  const onScroll = React.useCallback((event: React.UIEvent<HTMLDivElement>) => {
    const el = event.currentTarget;
    const distance = el.scrollHeight - el.scrollTop - el.clientHeight;
    stickRef.current = distance < 80;
    setAtBottom(distance < 80);
    setScrolled(el.scrollTop > 4);
  }, []);

  const scrollToBottom = React.useCallback(
    (behavior: ScrollBehavior = "smooth") => {
      const el = scrollRef.current;
      if (!el) return;
      stickRef.current = true;
      el.scrollTo({ top: el.scrollHeight, behavior });
    },
    [],
  );

  const scrollToApproval = React.useCallback(() => {
    const items = scrollRef.current?.querySelectorAll(
      "[data-approval-pending]",
    );
    items?.[items.length - 1]?.scrollIntoView({
      block: "center",
      behavior: "smooth",
    });
  }, []);

  const reset = React.useCallback(() => {
    stickRef.current = true;
    setAtBottom(true);
    setScrolled(false);
  }, []);

  return {
    scrollRef,
    contentRef,
    stickRef,
    atBottom,
    scrolled,
    approvalVisible,
    onScroll,
    scrollToBottom,
    scrollToApproval,
    reset,
  };
};
