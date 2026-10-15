import React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

import { Tooltip } from "@mantine/core";
import { twMerge } from "@/lib/utils";

import { Tone, formatSeconds, toneClasses } from "@/utils";

const TICK_MS = 15000;

const listeners = new Set<() => void>();
let now = Date.now();
let interval: ReturnType<typeof setInterval> | undefined;

const tick = () => {
  now = Date.now();
  listeners.forEach((notify) => notify());
};

const start = () => {
  if (interval || document.hidden) return;
  now = Date.now();
  interval = setInterval(tick, TICK_MS);
};

const stop = () => {
  if (!interval) return;
  clearInterval(interval);
  interval = undefined;
};

const onVisibility = () => {
  if (document.hidden) {
    stop();
    return;
  }
  tick();
  start();
};

const subscribe = (listener: () => void) => {
  listeners.add(listener);
  if (listeners.size === 1) {
    document.addEventListener("visibilitychange", onVisibility);
  }
  start();

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) {
      stop();
      document.removeEventListener("visibilitychange", onVisibility);
    }
  };
};

const getSnapshot = () => now;

export const useNow = () =>
  React.useSyncExternalStore(subscribe, getSnapshot, getSnapshot);

export const absoluteLabel = (date: Date) =>
  dayjs(date).local().format("h:mm A · ddd, MMM D, YYYY");

const TimeAgo = (props: { rfc3339?: Timestamp; className?: string }) => {
  useNow();
  const t = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;

  if (!t) return <span className={props.className}>—</span>;

  return (
    <Tooltip label={absoluteLabel(t)}>
      <time
        dateTime={t.toISOString()}
        className={twMerge("whitespace-nowrap", props.className)}
      >
        {dayjs(t).fromNow()}
      </time>
    </Tooltip>
  );
};

export const AbsoluteTime = (props: {
  rfc3339?: Timestamp;
  className?: string;
}) => {
  const t = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  if (!t) return <span className={props.className}>—</span>;

  return (
    <time
      dateTime={t.toISOString()}
      className={twMerge("whitespace-nowrap", props.className)}
    >
      {absoluteLabel(t)}
    </time>
  );
};

export const Countdown = (props: {
  date?: Date;
  suffix?: string;
  endedLabel?: string;
  tone?: Tone;
  className?: string;
}) => {
  const current = useNow();
  const seconds = props.date
    ? Math.floor((props.date.getTime() - current) / 1000)
    : undefined;

  if (seconds === undefined) return <span className={props.className}>—</span>;
  if (seconds <= 0) {
    return (
      <span className={twMerge("text-slate-500", props.className)}>
        {props.endedLabel ?? "Ended"}
      </span>
    );
  }

  return (
    <Tooltip label={absoluteLabel(props.date!)}>
      <time
        dateTime={props.date!.toISOString()}
        className={twMerge(
          "whitespace-nowrap",
          props.tone ? toneClasses[props.tone].text : undefined,
          props.className,
        )}
      >
        {formatSeconds(seconds)}
        {props.suffix ?? " left"}
      </time>
    </Tooltip>
  );
};

export const Elapsed = (props: {
  rfc3339?: Timestamp;
  suffix?: string;
  className?: string;
}) => {
  const current = useNow();
  const from = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;

  if (!from) return <span className={props.className}>—</span>;

  const seconds = Math.max(0, Math.floor((current - from.getTime()) / 1000));

  return (
    <Tooltip label={absoluteLabel(from)}>
      <time
        dateTime={from.toISOString()}
        className={twMerge("whitespace-nowrap", props.className)}
      >
        {formatSeconds(seconds)}
        {props.suffix ?? ""}
      </time>
    </Tooltip>
  );
};

export default TimeAgo;
