import type {
  AgentInfo,
  Conversation,
  ConversationSearchResult,
} from "@/apis/consoleagent/protocol";
import { ActionIcon, Loader, Menu, Tooltip } from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import { useQuery } from "@tanstack/react-query";
import { AnimatePresence, motion } from "framer-motion";
import {
  Ellipsis,
  MessagesSquare,
  PanelLeftClose,
  Pencil,
  Search,
  SearchX,
  SquarePen,
  Trash2,
  X,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";
import { AgentClient } from "./client";
import { Highlight, Kbd, modKey, StatusDot } from "./ui";
import { groupConversations, matchesTerms, searchTerms } from "./utils";

const ConversationItem = React.memo(
  (props: {
    conversation: Conversation;
    selected: boolean;
    terms: string[];
    onSelect: (id: string) => void;
    onRename: (conversation: Conversation, title: string) => Promise<void>;
    onDelete: (conversation: Conversation) => void;
  }) => {
    const { conversation, selected } = props;
    const [editing, setEditing] = React.useState(false);
    const [menuOpened, setMenuOpened] = React.useState(false);
    const [title, setTitle] = React.useState(conversation.title);
    const inputRef = React.useRef<HTMLInputElement>(null);

    React.useEffect(() => {
      if (!editing) return;
      setTitle(conversation.title);
      requestAnimationFrame(() => inputRef.current?.select());
    }, [editing, conversation.title]);

    const commit = async () => {
      setEditing(false);
      const value = title.trim();
      if (value === "" || value === conversation.title) return;
      await props.onRename(conversation, value);
    };

    return (
      <li className="relative">
        {selected && (
          <motion.span
            layoutId="agent-conversation-selected"
            transition={{ type: "spring", bounce: 0.12, duration: 0.35 }}
            className="absolute inset-0 rounded-lg bg-white shadow-card ring-1 ring-slate-200"
          />
        )}
        <div
          className={`text-body ${twMerge(
            "group relative flex h-9 items-center rounded-lg transition-colors duration-150",
            !selected && "hover:bg-slate-200/50",
          )}`}
        >
          {editing ? (
            <input
              ref={inputRef}
              aria-label="Conversation title"
              value={title}
              maxLength={200}
              onChange={(event) => setTitle(event.currentTarget.value)}
              onBlur={() => void commit()}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  void commit();
                } else if (event.key === "Escape") {
                  event.preventDefault();
                  setEditing(false);
                }
              }}
              className="mx-1 h-7 min-w-0 flex-1 rounded-md border border-slate-300 bg-white px-1.5 text-slate-900 outline-none ring-2 ring-slate-900/10"
            />
          ) : (
            <button
              type="button"
              onClick={() => props.onSelect(conversation.id)}
              onDoubleClick={() => setEditing(true)}
              aria-current={selected ? "page" : undefined}
              title={conversation.title}
              className={twMerge(
                "flex h-full min-w-0 flex-1 cursor-pointer items-center gap-2 rounded-lg pl-2.5 pr-8 text-left outline-none focus-visible:ring-2 focus-visible:ring-slate-400",
                selected
                  ? "text-slate-900"
                  : "text-slate-600 group-hover:text-slate-900",
              )}
            >
              <span
                className={twMerge(
                  "min-w-0 flex-1 truncate",
                  selected ? "font-semibold" : "font-normal",
                )}
              >
                <Highlight text={conversation.title} terms={props.terms} />
              </span>
              {conversation.activeRunId && (
                <Tooltip label="Working" withArrow>
                  <span className="flex shrink-0 items-center">
                    <StatusDot tone="blue" pulse />
                  </span>
                </Tooltip>
              )}
            </button>
          )}
          {!editing && (
            <span
              className={twMerge(
                "absolute right-1 flex transition-opacity duration-150 focus-within:opacity-100 [@media(hover:none)]:opacity-100",
                selected || menuOpened
                  ? "opacity-100"
                  : "opacity-0 group-hover:opacity-100",
              )}
            >
              <Menu
                position="bottom-end"
                withinPortal
                opened={menuOpened}
                onChange={setMenuOpened}
              >
                <Menu.Target>
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color="gray"
                    aria-label="Conversation actions"
                  >
                    <Ellipsis size={15} />
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Item
                    leftSection={<Pencil size={13} />}
                    onClick={() => setEditing(true)}
                  >
                    Rename
                  </Menu.Item>
                  <Menu.Item
                    color="red"
                    leftSection={<Trash2 size={13} />}
                    onClick={() => props.onDelete(conversation)}
                  >
                    Delete
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </span>
          )}
        </div>
      </li>
    );
  },
);

const SectionLabel = (props: { children: React.ReactNode }) => (
  <p className="px-2.5 pb-1 pt-4 text-micro font-semibold uppercase tracking-[0.08em] text-slate-400 first:pt-1">
    {props.children}
  </p>
);

const MessageResults = (props: {
  results: ConversationSearchResult[];
  terms: string[];
  onSelect: (id: string, messageId?: string) => void;
}) => (
  <>
    {props.results.flatMap((result) =>
      result.matches.map((match) => (
        <li key={`${result.conversation.id}/${match.messageId}`}>
          <button
            type="button"
            onClick={() =>
              props.onSelect(result.conversation.id, match.messageId)
            }
            className="flex w-full cursor-pointer flex-col gap-0.5 rounded-lg px-2.5 py-2 text-left outline-none transition-colors duration-150 hover:bg-slate-200/50 focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            <span className="flex min-w-0 items-center gap-1.5 text-micro font-semibold text-slate-500">
              <span className="min-w-0 truncate">
                {result.conversation.title}
              </span>
              <span className="shrink-0 font-normal text-slate-400">
                · {match.role === "user" ? "You" : "Agent"}
              </span>
            </span>
            <span className="line-clamp-2 text-xs leading-5 text-slate-700">
              <Highlight text={match.snippet} terms={props.terms} />
            </span>
          </button>
        </li>
      )),
    )}
  </>
);

const ListSkeleton = () => (
  <div className="space-y-1.5 px-1 pt-2 motion-safe:animate-pulse">
    {[72, 56, 84, 64, 48].map((width) => (
      <div key={width} className="flex h-9 items-center px-2.5">
        <div
          className="h-2.5 rounded-full bg-slate-200/80"
          style={{ width: `${width}%` }}
        />
      </div>
    ))}
  </div>
);

const Sidebar = (props: {
  client: AgentClient;
  info: AgentInfo;
  conversations: Conversation[];
  loading: boolean;
  selected?: string;
  onSelect: (id: string, messageId?: string) => void;
  onNew: () => void;
  onRename: (conversation: Conversation, title: string) => Promise<void>;
  onDelete: (conversation: Conversation) => void;
  onClose: () => void;
  closeLabel: string;
  searchRef: React.RefObject<HTMLInputElement | null>;
  footer?: React.ReactNode;
}) => {
  const { client, info } = props;
  const [query, setQuery] = React.useState("");
  const [debounced] = useDebouncedValue(query.trim(), 250);
  const terms = React.useMemo(() => searchTerms(query), [query]);
  const searching = terms.length > 0;

  const searchQuery = useQuery({
    queryKey: ["agent", client.baseUrl, "search", debounced],
    queryFn: ({ signal }) => client.searchConversations(debounced, signal),
    enabled: info.capabilities.search && debounced !== "",
    staleTime: 10000,
    placeholderData: (prev) => prev,
  });

  const groups = React.useMemo(
    () => groupConversations(props.conversations),
    [props.conversations],
  );

  const titleMatches = React.useMemo(
    () =>
      searching
        ? props.conversations.filter((c) => matchesTerms(c.title, terms))
        : [],
    [props.conversations, searching, terms],
  );

  const messageMatches = React.useMemo(
    () =>
      searching && debounced !== ""
        ? (searchQuery.data?.items ?? []).filter((r) => r.matches.length > 0)
        : [],
    [searching, debounced, searchQuery.data],
  );

  const searchPending =
    searching &&
    info.capabilities.search &&
    (debounced !== query.trim() || searchQuery.isFetching);

  const itemProps = {
    terms,
    onSelect: props.onSelect,
    onRename: props.onRename,
    onDelete: props.onDelete,
  };

  return (
    <div className="flex h-full w-full flex-col">
      <div className="flex h-12 shrink-0 items-center gap-2 px-3">
        <span className="min-w-0 flex-1 truncate text-micro font-semibold uppercase tracking-[0.09em] text-slate-500">
          Conversations
        </span>
        <Tooltip label={props.closeLabel} withArrow>
          <ActionIcon
            variant="subtle"
            color="gray"
            aria-label={props.closeLabel}
            onClick={props.onClose}
          >
            <PanelLeftClose size={16} />
          </ActionIcon>
        </Tooltip>
      </div>

      <div className="space-y-2 px-3 pb-2">
        <button
          type="button"
          onClick={props.onNew}
          className="group flex h-9 w-full cursor-pointer items-center gap-2 rounded-lg border border-slate-200 bg-white px-2.5 text-slate-800 shadow-card outline-none transition-[border-color,box-shadow] duration-200 hover:border-slate-300 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <SquarePen size={14} strokeWidth={2.25} className="text-slate-500" />
          <span className="flex-1 text-left text-body font-semibold">
            New chat
          </span>
          <span className="hidden items-center gap-0.5 lg:flex">
            <Kbd>{modKey}</Kbd>
            <Kbd>⇧</Kbd>
            <Kbd>O</Kbd>
          </span>
        </button>

        <div className="relative text-base sm:text-body">
          <Search
            size={14}
            className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400"
          />
          <input
            ref={props.searchRef}
            type="search"
            value={query}
            aria-label="Search conversations"
            placeholder="Search"
            onChange={(event) => setQuery(event.currentTarget.value)}
            onKeyDown={(event) => {
              if (event.key === "Escape") {
                if (query) {
                  event.preventDefault();
                  setQuery("");
                } else {
                  event.currentTarget.blur();
                }
              } else if (event.key === "Enter") {
                const first = titleMatches[0];
                const message = messageMatches[0];
                if (first) {
                  props.onSelect(first.id);
                } else if (message) {
                  props.onSelect(
                    message.conversation.id,
                    message.matches[0]?.messageId,
                  );
                }
              }
            }}
            className="h-9 w-full rounded-lg border border-transparent bg-slate-200/50 pl-8 pr-14 text-slate-900 outline-none transition-[background-color,border-color,box-shadow] duration-150 placeholder:text-slate-500 hover:bg-slate-200/70 focus:border-slate-300 focus:bg-white focus:shadow-[0_0_0_3px_color-mix(in_oklab,var(--color-slate-900)_6%,transparent)] [&::-webkit-search-cancel-button]:hidden"
          />
          <span className="absolute right-1.5 top-1/2 flex -translate-y-1/2 items-center gap-1">
            {searchPending && <Loader size={12} color="gray" />}
            {query ? (
              <ActionIcon
                size="sm"
                variant="subtle"
                color="gray"
                aria-label="Clear the search"
                onClick={() => {
                  setQuery("");
                  props.searchRef.current?.focus();
                }}
              >
                <X size={13} />
              </ActionIcon>
            ) : (
              <span className="hidden items-center gap-0.5 lg:flex">
                <Kbd>{modKey}</Kbd>
                <Kbd>K</Kbd>
              </span>
            )}
          </span>
        </div>
      </div>

      <nav
        aria-label="Conversations"
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-2 pb-3"
      >
        {props.loading ? (
          <ListSkeleton />
        ) : searching ? (
          <AnimatePresence mode="wait" initial={false}>
            <motion.div
              key="search"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.12 }}
            >
              {titleMatches.length > 0 && (
                <>
                  <SectionLabel>Titles</SectionLabel>
                  <ul className="space-y-px">
                    {titleMatches.map((conversation) => (
                      <ConversationItem
                        key={conversation.id}
                        conversation={conversation}
                        selected={conversation.id === props.selected}
                        {...itemProps}
                      />
                    ))}
                  </ul>
                </>
              )}
              {messageMatches.length > 0 && (
                <>
                  <SectionLabel>Messages</SectionLabel>
                  <ul className="space-y-px">
                    <MessageResults
                      results={messageMatches}
                      terms={terms}
                      onSelect={props.onSelect}
                    />
                  </ul>
                </>
              )}
              {titleMatches.length === 0 &&
                messageMatches.length === 0 &&
                !searchPending && (
                  <div className="flex flex-col items-center px-4 py-10 text-center">
                    <SearchX size={20} className="text-slate-400" />
                    <p className="mt-2 text-xs font-semibold text-slate-600">
                      No results
                    </p>
                    <p className="mt-0.5 text-micro text-slate-500">
                      {info.capabilities.search
                        ? "Nothing matches in the titles or the messages"
                        : "Nothing matches in the titles"}
                    </p>
                  </div>
                )}
            </motion.div>
          </AnimatePresence>
        ) : props.conversations.length === 0 ? (
          <div className="flex flex-col items-center px-4 py-10 text-center">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-slate-200 bg-white text-slate-400 shadow-card">
              <MessagesSquare size={17} />
            </span>
            <p className="mt-3 text-xs font-semibold text-slate-600">
              No conversations yet
            </p>
            <p className="mt-0.5 text-micro text-slate-500">
              Your conversations with the agent appear here
            </p>
          </div>
        ) : (
          groups.map((group) => (
            <section key={group.label}>
              <SectionLabel>{group.label}</SectionLabel>
              <ul className="space-y-px">
                {group.items.map((conversation) => (
                  <ConversationItem
                    key={conversation.id}
                    conversation={conversation}
                    selected={conversation.id === props.selected}
                    {...itemProps}
                  />
                ))}
              </ul>
            </section>
          ))
        )}
      </nav>

      {props.footer && (
        <div className="shrink-0 border-t border-slate-200/80 p-2">
          {props.footer}
        </div>
      )}
    </div>
  );
};

export default React.memo(Sidebar);
