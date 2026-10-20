import type { ContextItem } from "@/apis/consoleagent/protocol";

const MAX_DATA = 7500;
const MAX_TEXT = 3000;
const MAX_VALUES = 40;
const MAX_LINKS = 20;

type SnapshotValue = {
  label: string;
  value: string;
  previous?: string;
  to?: string;
};

const collapse = (text: string) =>
  text
    .replace(/[ \t]+/g, " ")
    .replace(/\s*\n\s*/g, "\n")
    .trim();

const hrefOf = (element: Element) =>
  (element.closest("a") ?? element.querySelector("a"))?.getAttribute("href") ??
  undefined;

export const snapshotSection = (
  element: HTMLElement,
  meta: {
    id?: string;
    title: string;
    description?: string;
    path: string;
    details?: Record<string, string>;
  },
): ContextItem => {
  const values: SnapshotValue[] = Array.from(
    element.querySelectorAll("[data-agent-label]"),
  )
    .slice(0, MAX_VALUES)
    .map((item) => ({
      label: item.getAttribute("data-agent-label") ?? "",
      value: item.getAttribute("data-agent-value") ?? "",
      previous: item.getAttribute("data-agent-previous") ?? undefined,
      to: hrefOf(item),
    }));

  const seen = new Set<string>();
  const links = Array.from(element.querySelectorAll("a[href]"))
    .map((item) => ({
      text:
        item.getAttribute("data-agent-label") ??
        collapse((item as HTMLElement).innerText ?? "").split("\n")[0],
      to: item.getAttribute("href") ?? "",
    }))
    .filter((item) => {
      if (!item.to || seen.has(item.to)) return false;
      seen.add(item.to);
      return true;
    })
    .slice(0, MAX_LINKS);

  const data = {
    page: meta.path,
    window: meta.details?.["Time range"],
    description: meta.description,
    capturedAt: new Date().toISOString(),
    values,
    links,
    text: collapse(element.innerText ?? "").slice(0, MAX_TEXT),
  };

  const overflow = JSON.stringify(data).length - MAX_DATA;
  if (overflow > 0) {
    data.text = data.text.slice(0, Math.max(0, data.text.length - overflow));
  }

  return {
    kind: "DashboardSection",
    id: meta.id,
    title: meta.title,
    data,
  };
};
