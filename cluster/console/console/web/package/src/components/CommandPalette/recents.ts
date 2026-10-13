export interface RecentEntry {
  path: string;
  title: string;
  subtitle?: string;
  visits: number;
  lastVisited: number;
}

const storageKey = "octelium-console-recents";
const maxEntries = 40;
const hour = 60 * 60 * 1000;

export const frecency = (entry: RecentEntry, now = Date.now()) => {
  const age = now - entry.lastVisited;
  const weight =
    age < hour ? 4 : age < 24 * hour ? 2 : age < 7 * 24 * hour ? 1 : 0.5;
  return entry.visits * weight;
};

export const readRecents = (): RecentEntry[] => {
  try {
    const parsed = JSON.parse(localStorage.getItem(storageKey) ?? "[]");
    return Array.isArray(parsed)
      ? parsed.filter(
          (itm): itm is RecentEntry =>
            typeof itm?.path === "string" &&
            typeof itm.title === "string" &&
            typeof itm.visits === "number" &&
            typeof itm.lastVisited === "number",
        )
      : [];
  } catch {
    return [];
  }
};

export const recordVisit = (
  visit: Pick<RecentEntry, "path" | "title" | "subtitle">,
  now = Date.now(),
) => {
  const entries = readRecents();
  const existing = entries.find((itm) => itm.path === visit.path);
  const next: RecentEntry = {
    ...visit,
    visits: (existing?.visits ?? 0) + 1,
    lastVisited: now,
  };
  const ret = [next, ...entries.filter((itm) => itm.path !== visit.path)];
  if (ret.length > maxEntries) {
    ret.sort((a, b) => frecency(b, now) - frecency(a, now));
    ret.length = maxEntries;
  }
  try {
    localStorage.setItem(storageKey, JSON.stringify(ret));
  } catch {
    return;
  }
};
