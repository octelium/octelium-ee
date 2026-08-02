import React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

import { Tooltip } from "@mantine/core";

const listeners = new Set<() => void>();
let now = Date.now();
let interval: ReturnType<typeof setInterval> | undefined;

const subscribe = (listener: () => void) => {
  listeners.add(listener);

  if (!interval) {
    now = Date.now();
    interval = setInterval(() => {
      now = Date.now();
      listeners.forEach((notify) => notify());
    }, 10000);
  }

  return () => {
    listeners.delete(listener);
    if (listeners.size === 0 && interval) {
      clearInterval(interval);
      interval = undefined;
    }
  };
};

const getSnapshot = () => now;

const TimeAgo = (props: { rfc3339?: Timestamp }) => {
  const currentTime = React.useSyncExternalStore(
    subscribe,
    getSnapshot,
    getSnapshot,
  );

  if (!props.rfc3339) {
    return <></>;
  }

  const t = Timestamp.toDate(props.rfc3339);
  const time = dayjs(t).from(currentTime);
  return (
    <Tooltip
      label={
        <p className="font-bold shadow-md text-xs rounded-sm">
          {dayjs(t).local().format("hh:mm:ss A, ddd MMM D, YYYY")}
        </p>
      }
      transitionProps={{
        transition: "fade",
        duration: 340,
      }}
    >
      <span>{time}</span>
    </Tooltip>
  );
};

export default TimeAgo;
