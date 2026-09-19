const MAX_IN_FLIGHT = 4;

export const QUERY_PRIORITY = {
  critical: 0,
  high: 1,
  normal: 2,
  low: 3,
} as const;

type Task = {
  priority: number;
  order: number;
  start: () => void;
};

let inFlight = 0;
let counter = 0;
const pending: Task[] = [];

const remove = (task: Task) => {
  const index = pending.indexOf(task);
  if (index >= 0) pending.splice(index, 1);
};

const takeNext = () => {
  if (pending.length === 0) return undefined;
  let best = 0;
  for (let index = 1; index < pending.length; index++) {
    const candidate = pending[index];
    const current = pending[best];
    if (
      candidate.priority < current.priority ||
      (candidate.priority === current.priority &&
        candidate.order < current.order)
    ) {
      best = index;
    }
  }
  return pending.splice(best, 1)[0];
};

const drain = () => {
  while (inFlight < MAX_IN_FLIGHT) {
    const task = takeNext();
    if (!task) return;
    inFlight++;
    task.start();
  }
};

export const queued = <T>(
  priority: number,
  run: (signal?: AbortSignal) => Promise<T>,
  signal?: AbortSignal,
): Promise<T> =>
  new Promise<T>((resolve, reject) => {
    let released = false;
    const release = () => {
      if (released) return;
      released = true;
      inFlight--;
      drain();
    };

    const task: Task = {
      priority,
      order: counter++,
      start: () => {
        run(signal).then(resolve, reject).finally(release);
      },
    };

    const cancel = () => {
      remove(task);
      reject(signal?.reason ?? new Error("The request was cancelled"));
    };

    if (signal?.aborted) {
      cancel();
      return;
    }

    signal?.addEventListener("abort", cancel, { once: true });

    pending.push(task);
    drain();
  });
