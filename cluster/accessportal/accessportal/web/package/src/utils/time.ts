import * as React from "react";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import utc from "dayjs/plugin/utc";

dayjs.extend(relativeTime);
dayjs.extend(utc);

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

