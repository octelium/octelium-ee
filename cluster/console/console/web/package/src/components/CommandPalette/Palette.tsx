import type { Conversation } from "@/apis/consoleagent/protocol";
import { useOptionalAgent } from "@/components/AgentDock/context";
import type { NavItem } from "@/components/Navigation/catalog";
import { AgentClient } from "@/pages/Agent/client";
import { conversationsKey } from "@/pages/Agent/environment";
import {
  getListOptionsPB,
  getResourcePathFromAPIKind,
  listResourcesPB,
  type API,
  type Resource,
  type ResourceList,
  type ResourceName,
} from "@/utils/pb";
import {
  Loader,
  Modal,
  useComputedColorScheme,
  useMantineColorScheme,
} from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { useQueries, useQuery } from "@tanstack/react-query";
import {
  Box,
  CornerDownLeft,
  History,
  MessagesSquare,
  Moon,
  Plus,
  Search,
  Sparkles,
  Sun,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { twMerge } from "@/lib/utils";
import { isQuestion, normalize, scoreEntry } from "./match";
import { frecency, readRecents } from "./recents";

interface PaletteItem {
  id: string;
  group: string;
  title: string;
  subtitle?: string;
  hint?: string;
  icon: LucideIcon;
  to?: string;
  run?: () => void;
  accent?: boolean;
}

interface SearchKind {
  api: API;
  kind: string;
  aliases: string[];
  standard?: boolean;
}

const searchKinds: SearchKind[] = [
  {
    api: "core",
    kind: "Service",
    aliases: ["service", "services", "svc"],
    standard: true,
  },
  { api: "core", kind: "User", aliases: ["user", "users"], standard: true },
  { api: "core", kind: "Group", aliases: ["group", "groups"], standard: true },
  {
    api: "core",
    kind: "Namespace",
    aliases: ["namespace", "namespaces", "ns"],
    standard: true,
  },
  {
    api: "core",
    kind: "Policy",
    aliases: ["policy", "policies"],
    standard: true,
  },
  { api: "core", kind: "Session", aliases: ["session", "sessions"] },
  { api: "core", kind: "Device", aliases: ["device", "devices"] },
  { api: "core", kind: "Credential", aliases: ["credential", "credentials"] },
  {
    api: "core",
    kind: "IdentityProvider",
    aliases: ["idp", "idps", "identityprovider"],
  },
  { api: "core", kind: "Secret", aliases: ["secret", "secrets"] },
  { api: "core", kind: "Gateway", aliases: ["gateway", "gateways"] },
  { api: "core", kind: "Region", aliases: ["region", "regions"] },
  {
    api: "core",
    kind: "Authenticator",
    aliases: ["authenticator", "authenticators"],
  },
  {
    api: "enterprise",
    kind: "Certificate",
    aliases: ["certificate", "certificates", "cert"],
  },
  { api: "access", kind: "Request", aliases: ["request", "requests"] },
  { api: "access", kind: "Catalog", aliases: ["catalog", "catalogs"] },
  { api: "cordium", kind: "Workspace", aliases: ["workspace", "workspaces"] },
];

const createKinds: { api: API; kind: string }[] = [
  { api: "core", kind: "Service" },
  { api: "core", kind: "Namespace" },
  { api: "core", kind: "User" },
  { api: "core", kind: "Group" },
  { api: "core", kind: "Policy" },
  { api: "core", kind: "Secret" },
  { api: "core", kind: "Credential" },
];

const parseSearch = (value: string) => {
  const tokens = value.trim().split(/\s+/).filter(Boolean);
  const scoped =
    tokens.length > 1
      ? searchKinds.find((itm) => itm.aliases.includes(normalize(tokens[0])))
      : undefined;
  return scoped
    ? { kinds: [scoped], term: tokens.slice(1).join(" "), scope: scoped }
    : {
        kinds: searchKinds.filter((itm) => itm.standard),
        term: tokens.join(" "),
        scope: undefined,
      };
};

export interface PalettePage extends NavItem {
  subtitle?: string;
}

const listPath = (api: API, kind: string) =>
  `/${api}/${getResourcePathFromAPIKind({ api, kind: kind as ResourceName })}`;

const ItemRow = (props: {
  item: PaletteItem;
  id: string;
  active: boolean;
  onHover: () => void;
  onSelect: (event: React.MouseEvent) => void;
}) => {
  const { item } = props;
  return (
    <div
      id={props.id}
      role="option"
      aria-selected={props.active}
      data-active={props.active || undefined}
      onMouseMove={props.onHover}
      onMouseDown={(event) => event.preventDefault()}
      onClick={props.onSelect}
      className={twMerge(
        "flex h-11 cursor-pointer items-center gap-3 rounded-lg px-2.5 transition-colors duration-75",
        props.active ? "bg-slate-100" : "bg-transparent",
      )}
    >
      <span
        className={twMerge(
          "flex h-7 w-7 shrink-0 items-center justify-center rounded-md",
          item.accent
            ? "bg-slate-900 text-white"
            : "border border-slate-200 bg-white text-slate-500",
        )}
      >
        <item.icon size={14} strokeWidth={2.2} />
      </span>
      <span className="flex min-w-0 flex-1 items-baseline gap-2">
        <span className="truncate text-body font-semibold text-slate-800">
          {item.title}
        </span>
        {item.subtitle && (
          <span className="truncate text-xs text-slate-500">
            {item.subtitle}
          </span>
        )}
      </span>
      {item.hint && (
        <span className="text-micro shrink-0 font-semibold text-slate-400">
          {item.hint}
        </span>
      )}
      {props.active && (
        <CornerDownLeft
          size={13}
          strokeWidth={2.25}
          className="shrink-0 text-slate-400"
        />
      )}
    </div>
  );
};

const Kbd = (props: { children: React.ReactNode }) => (
  <kbd className="text-micro inline-flex h-5 min-w-5 items-center justify-center rounded-md border border-slate-200 bg-white px-1 font-semibold text-slate-500">
    {props.children}
  </kbd>
);

const Palette = (props: {
  opened: boolean;
  onClose: () => void;
  pages: PalettePage[];
  shortcuts: NavItem[];
  iconFor: (path: string) => LucideIcon | undefined;
}) => {
  const { iconFor } = props;
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const agent = useOptionalAgent();
  const { setColorScheme } = useMantineColorScheme();
  const scheme = useComputedColorScheme("light");
  const [query, setQuery] = React.useState("");
  const [active, setActive] = React.useState(0);
  const [debounced] = useDebouncedValue(query, 200);
  const listRef = React.useRef<HTMLDivElement>(null);
  const listID = React.useId();

  React.useEffect(() => {
    if (props.opened) {
      setQuery("");
      setActive(0);
    }
  }, [props.opened]);

  const recents = React.useMemo(
    () => (props.opened ? readRecents() : []),
    [props.opened],
  );

  const search = React.useMemo(() => parseSearch(debounced), [debounced]);
  const searchable = props.opened && search.term.length >= 2;
  const resourceResults = useQueries({
    queries: search.kinds.map((itm) => ({
      queryKey: ["palette", itm.api, itm.kind, search.term],
      enabled: searchable,
      staleTime: 30_000,
      retry: false,
      refetchOnWindowFocus: false,
      queryFn: async ({ signal }: { signal: AbortSignal }) => {
        const { response } = await listResourcesPB(
          itm.api,
          itm.kind,
          getListOptionsPB(itm.api, itm.kind)["create"]({
            common: { query: search.term, itemsPerPage: 5 },
          }),
          { abort: signal },
        );
        return ((response as ResourceList).items ?? []) as Resource[];
      },
    })),
  });
  const searching =
    searchable && resourceResults.some((result) => result.isFetching);

  const ready = agent?.status === "ready";
  const usable =
    !!agent && agent.status !== "unavailable" && agent.status !== "loading";
  const conversationsQuery = useQuery({
    queryKey: conversationsKey(agent?.url ?? ""),
    queryFn: () => new AgentClient(agent!.url!).listConversations(),
    enabled: props.opened && ready && !!agent?.url,
    staleTime: 30_000,
  });

  const items = React.useMemo<PaletteItem[]>(() => {
    const q = query.trim();
    const ret: PaletteItem[] = [];
    const seen = new Set<string>();
    const push = (item: PaletteItem) => {
      const key = item.to ?? item.id;
      if (seen.has(key)) return;
      seen.add(key);
      ret.push(item);
    };

    const askItem = (): PaletteItem => ({
      id: "ask",
      group: "Agent",
      title: q ? `Ask the agent: “${q}”` : "Ask the agent about this page",
      icon: Sparkles,
      accent: true,
      hint: ready ? undefined : "Starts the agent",
      run: () =>
        q ? agent?.ask({ text: q, send: true, fresh: true }) : agent?.open(),
    });

    const conversationItem = (conversation: Conversation): PaletteItem => ({
      id: `conversation-${conversation.id}`,
      group: "Agent conversations",
      title: conversation.title,
      icon: MessagesSquare,
      to: `/agent?c=${encodeURIComponent(conversation.id)}`,
    });

    const themeItem: PaletteItem = {
      id: "theme",
      group: "Actions",
      title:
        scheme === "dark"
          ? "Switch to the light theme"
          : "Switch to the dark theme",
      icon: scheme === "dark" ? Sun : Moon,
      run: () => setColorScheme(scheme === "dark" ? "light" : "dark"),
    };

    const conversations = conversationsQuery.data?.items ?? [];

    if (!q) {
      const recent = [...recents]
        .filter((itm) => itm.path !== pathname)
        .sort((a, b) => b.lastVisited - a.lastVisited)
        .slice(0, 5);
      recent.forEach((itm, idx) =>
        push({
          id: `recent-${itm.path}`,
          group: "Recent",
          title: itm.title,
          subtitle: itm.subtitle,
          hint: idx === 0 ? "Last page" : undefined,
          icon: iconFor(itm.path) ?? History,
          to: itm.path,
        }),
      );
      [...recents]
        .filter((itm) => itm.path !== pathname && itm.visits >= 3)
        .sort((a, b) => frecency(b) - frecency(a))
        .slice(0, 4)
        .forEach((itm) =>
          push({
            id: `frequent-${itm.path}`,
            group: "Frequently visited",
            title: itm.title,
            subtitle: itm.subtitle,
            icon: iconFor(itm.path) ?? History,
            to: itm.path,
          }),
        );
      if (usable) push(askItem());
      conversations.slice(0, 3).forEach((itm) => push(conversationItem(itm)));
      props.shortcuts.forEach((page) =>
        push({
          id: `page-${page.url}`,
          group: "Go to",
          title: page.title,
          icon: page.icon,
          to: page.url,
        }),
      );
      push(themeItem);
      return ret;
    }

    if (usable && isQuestion(q)) push(askItem());

    props.pages
      .map((page) => ({ page, score: scoreEntry(q, page) }))
      .filter((itm) => itm.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 6)
      .forEach(({ page }) =>
        push({
          id: `page-${page.url}`,
          group: "Pages",
          title: page.title,
          subtitle: page.subtitle,
          icon: page.icon,
          to: page.url,
        }),
      );

    recents
      .map((itm) => ({ itm, score: scoreEntry(q, itm) }))
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score || frecency(b.itm) - frecency(a.itm))
      .slice(0, 4)
      .forEach(({ itm }) =>
        push({
          id: `recent-${itm.path}`,
          group: "Recent",
          title: itm.title,
          subtitle: itm.subtitle,
          icon: iconFor(itm.path) ?? History,
          to: itm.path,
        }),
      );

    if (searchable && normalize(query) === normalize(debounced)) {
      search.kinds.forEach((kind, idx) => {
        const path = listPath(kind.api, kind.kind);
        for (const resource of resourceResults[idx]?.data ?? []) {
          const name = resource.metadata?.name;
          if (!name) continue;
          push({
            id: `resource-${kind.kind}-${resource.metadata?.uid}`,
            group: "Resources",
            title: name,
            subtitle: resource.metadata?.displayName || undefined,
            hint: kind.kind,
            icon: iconFor(path) ?? Box,
            to: `${path}/${encodeURIComponent(name)}`,
          });
        }
      });
    }

    conversations
      .map((itm) => ({ itm, score: scoreEntry(q, { title: itm.title }) }))
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 4)
      .forEach(({ itm }) => push(conversationItem(itm)));

    const actions: PaletteItem[] = [
      ...createKinds.map((itm) => ({
        id: `create-${itm.kind}`,
        group: "Actions",
        title: `Create ${itm.kind}`,
        icon: Plus,
        to: `${listPath(itm.api, itm.kind)}/create`,
      })),
      themeItem,
    ];
    actions
      .map((itm) => ({ itm, score: scoreEntry(q, itm) }))
      .filter(({ score }) => score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 3)
      .forEach(({ itm }) => push(itm));

    if (usable && !isQuestion(q)) push(askItem());
    return ret;
  }, [
    query,
    debounced,
    recents,
    pathname,
    usable,
    ready,
    agent,
    scheme,
    setColorScheme,
    conversationsQuery.data,
    searchable,
    search,
    resourceResults,
    props.pages,
    props.shortcuts,
    iconFor,
  ]);

  React.useEffect(() => {
    setActive((value) => Math.min(value, Math.max(0, items.length - 1)));
  }, [items.length]);

  React.useEffect(() => {
    listRef.current
      ?.querySelector("[data-active]")
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const select = (item: PaletteItem | undefined, newTab = false) => {
    if (!item) return;
    if (item.to && newTab) {
      window.open(item.to, "_blank", "noopener");
      return;
    }
    props.onClose();
    if (item.to) navigate(item.to);
    item.run?.();
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.nativeEvent.isComposing) return;
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        setActive((value) => (items.length ? (value + 1) % items.length : 0));
        break;
      case "ArrowUp":
        event.preventDefault();
        setActive((value) =>
          items.length ? (value - 1 + items.length) % items.length : 0,
        );
        break;
      case "Enter":
        event.preventDefault();
        select(items[active], event.metaKey || event.ctrlKey);
        break;
    }
  };

  let lastGroup = "";

  return (
    <Modal
      opened={props.opened}
      onClose={props.onClose}
      withCloseButton={false}
      padding={0}
      size={640}
      yOffset="12vh"
      overlayProps={{ backgroundOpacity: 0.3, blur: 2 }}
      transitionProps={{ transition: "pop", duration: 160 }}
      styles={{
        content: {
          border: "1px solid var(--color-slate-200)",
          borderRadius: "14px",
          boxShadow: "var(--shadow-modal)",
          overflow: "hidden",
        },
      }}
    >
      <div className="flex h-14 items-center gap-3 border-b border-slate-200 px-4">
        <Search
          size={17}
          strokeWidth={2.2}
          className="shrink-0 text-slate-400"
        />
        {search.scope && (
          <span className="text-micro shrink-0 rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 font-semibold text-slate-600">
            {search.scope.kind}
          </span>
        )}
        <input
          data-autofocus
          role="combobox"
          aria-label="Search pages and resources, or ask the agent"
          aria-expanded
          aria-controls={listID}
          aria-activedescendant={
            items[active] ? `${listID}-${active}` : undefined
          }
          aria-autocomplete="list"
          value={query}
          onChange={(event) => {
            setQuery(event.currentTarget.value);
            setActive(0);
          }}
          onKeyDown={onKeyDown}
          placeholder="Search pages and resources, or ask the agent…"
          className="h-full min-w-0 flex-1 bg-transparent text-[15px] text-slate-900 outline-none placeholder:text-slate-400"
        />
        {searching && <Loader size={14} color="gray" />}
        <span className="hidden sm:inline-flex">
          <Kbd>Esc</Kbd>
        </span>
      </div>

      <div
        ref={listRef}
        id={listID}
        role="listbox"
        aria-label="Results"
        className="max-h-[min(60vh,480px)] overflow-y-auto overscroll-contain p-2"
      >
        {items.length === 0 ? (
          <p className="px-3 py-10 text-center text-body text-slate-500">
            {searching ? "Searching…" : `No results for “${query.trim()}”`}
          </p>
        ) : (
          items.map((item, idx) => {
            const header = item.group !== lastGroup ? item.group : undefined;
            lastGroup = item.group;
            return (
              <React.Fragment key={item.id}>
                {header && (
                  <p
                    className="text-micro px-2.5 pb-1 pt-2.5 font-semibold text-slate-500 first:pt-1"
                    role="presentation"
                  >
                    {header}
                  </p>
                )}
                <ItemRow
                  item={item}
                  id={`${listID}-${idx}`}
                  active={idx === active}
                  onHover={() => setActive(idx)}
                  onSelect={(event) =>
                    select(item, event.metaKey || event.ctrlKey)
                  }
                />
              </React.Fragment>
            );
          })
        )}
      </div>

      <div className="text-micro flex items-center gap-3 border-t border-slate-200 bg-slate-50/70 px-4 py-2 text-slate-500">
        <span className="flex items-center gap-1">
          <Kbd>↑</Kbd>
          <Kbd>↓</Kbd>
          to navigate
        </span>
        <span className="flex items-center gap-1">
          <Kbd>↵</Kbd>
          to open
        </span>
        <span className="ml-auto hidden items-center gap-1 sm:flex">
          Start with a kind, e.g.
          <span className="rounded bg-slate-200/60 px-1 font-mono text-slate-600">
            user alice
          </span>
        </span>
      </div>
    </Modal>
  );
};

export default Palette;
