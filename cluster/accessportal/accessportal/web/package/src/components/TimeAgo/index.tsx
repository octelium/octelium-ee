import React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

import { Tooltip } from "@mantine/core";
import { twMerge } from "tailwind-merge";

import { Tone, formatSeconds, toneClasses } from "@/utils";

export const useNow = (intervalMs = 10000) => {
  const [now, setNow] = React.useState(() => Date.now());

  React.useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(interval);
  }, [intervalMs]);

  return now;
};

export const absoluteLabel = (date: Date) =>
  dayjs(date).local().format("h:mm A · ddd, MMM D, YYYY");

const TimeAgo = (props: { rfc3339?: Timestamp; className?: string }) => {
  const t = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  useNow(10000);

  if (!t) return <span className={props.className}>—</span>;

  return (
    <Tooltip label={absoluteLabel(t)}>
      <span className={twMerge("whitespace-nowrap", props.className)}>
        {dayjs(t).fromNow()}
      </span>
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
    <Tooltip label={dayjs(t).fromNow()}>
      <span className={twMerge("whitespace-nowrap", props.className)}>
        {absoluteLabel(t)}
      </span>
    </Tooltip>
  );
};

export const remainingSeconds = (target?: Date, now = Date.now()) =>
  target ? Math.floor((target.getTime() - now) / 1000) : undefined;

export const Countdown = (props: {
  date?: Date;
  suffix?: string;
  endedLabel?: string;
  tone?: Tone;
  className?: string;
}) => {
  const now = useNow(1000 * 15);
  const seconds = remainingSeconds(props.date, now);

  if (seconds === undefined) return <span className={props.className}>—</span>;
  if (seconds <= 0) {
    return (
      <span className={twMerge("text-slate-400", props.className)}>
        {props.endedLabel ?? "Ended"}
      </span>
    );
  }

  const label = `${formatSeconds(seconds)}${props.suffix ?? " left"}`;

  return (
    <Tooltip label={absoluteLabel(props.date!)}>
      <span
        className={twMerge(
          "whitespace-nowrap",
          props.tone ? toneClasses[props.tone].text : undefined,
          props.className,
        )}
      >
        {label}
      </span>
    </Tooltip>
  );
};

export const TimeRemaining = (props: {
  rfc3339?: Timestamp;
  className?: string;
}) => {
  const target = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  const now = useNow(1000 * 15);
  const seconds = remainingSeconds(target, now);

  if (seconds === undefined) return <span className={props.className}>—</span>;

  return (
    <Countdown
      date={target}
      tone={seconds <= 3600 ? "amber" : "emerald"}
      className={props.className}
    />
  );
};

export const Elapsed = (props: {
  rfc3339?: Timestamp;
  suffix?: string;
  className?: string;
}) => {
  const from = props.rfc3339 ? Timestamp.toDate(props.rfc3339) : undefined;
  const now = useNow(1000 * 15);

  if (!from) return <span className={props.className}>—</span>;

  const seconds = Math.max(0, Math.floor((now - from.getTime()) / 1000));

  return (
    <Tooltip label={absoluteLabel(from)}>
      <span className={twMerge("whitespace-nowrap", props.className)}>
        {formatSeconds(seconds)}
        {props.suffix ?? ""}
      </span>
    </Tooltip>
  );
};

export default TimeAgo;
