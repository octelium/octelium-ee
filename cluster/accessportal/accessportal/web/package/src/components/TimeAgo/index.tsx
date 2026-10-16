import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";

import { Tooltip } from "@mantine/core";
import { twMerge } from "@/lib/utils";

import { Tone, formatSeconds, toneClasses } from "@/utils";
import { absoluteLabel, useNow } from "@/utils/time";

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
