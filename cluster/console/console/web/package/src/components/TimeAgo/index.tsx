import React from "react";

import { Timestamp } from "@/apis/google/protobuf/timestamp";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

const TICK_MS = 30000;

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

const TimeAgo = (props: { rfc3339?: Timestamp }) => {
  const currentTime = React.useSyncExternalStore(
    subscribe,
    getSnapshot,
    getSnapshot,
  );

  if (!props.rfc3339) {
    return <></>;
  }

  const date = dayjs(Timestamp.toDate(props.rfc3339));

  return (
    <span title={date.local().format("hh:mm:ss A, ddd MMM D, YYYY")}>
      {date.from(currentTime)}
    </span>
  );
};

export default TimeAgo;
