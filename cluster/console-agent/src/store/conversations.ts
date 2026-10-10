import * as crypto from "node:crypto";
import * as fs from "node:fs";
import * as path from "node:path";
import type {
  Conversation,
  ConversationSearchMatch,
  ConversationSearchResult,
  Message,
  MessageRole,
} from "../protocol/index.ts";

export const isValidID = (id: string): boolean =>
  /^[A-Za-z0-9_-]{1,64}$/.test(id);

export const newID = (): string => crypto.randomUUID();

export const writeFileAtomic = (filePath: string, data: string) => {
  const tmp = `${filePath}.${process.pid}.${crypto.randomBytes(4).toString("hex")}.tmp`;
  fs.writeFileSync(tmp, data, { mode: 0o600 });
  fs.renameSync(tmp, filePath);
};

interface StoredConversation extends Conversation {
  sessionFile?: string;
}

interface SearchEntry {
  messageId: string;
  role: MessageRole;
  text: string;
  lower: string;
}

const searchMaxMatches = 3;
const snippetBefore = 60;
const snippetLength = 180;

const toConversation = (stored: StoredConversation): Conversation => {
  const { sessionFile: _, ...ret } = stored;
  return ret;
};

export const searchTerms = (query: string): string[] => [
  ...new Set(
    query
      .toLowerCase()
      .split(/\s+/)
      .filter((term) => term !== ""),
  ),
];

export const plainText = (markdown: string): string =>
  markdown
    .replace(/```[^\n]*/g, " ")
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\*\*|__|~~|`/g, "")
    .replace(/^\s{0,3}(#{1,6}|>|[-*+]|\d+\.)\s+/gm, "")
    .replace(/\|/g, " ")
    .replace(/\s+/g, " ")
    .trim();

const messageText = (message: Message): string =>
  plainText(
    message.blocks
      .map((block) => (block.type === "markdown" ? block.text : ""))
      .filter((text) => text !== "")
      .join("\n"),
  );

export const searchSnippet = (
  text: string,
  lower: string,
  terms: string[],
): string | undefined => {
  let first = -1;
  for (const term of terms) {
    const idx = lower.indexOf(term);
    if (idx < 0) {
      return undefined;
    }
    if (first < 0 || idx < first) {
      first = idx;
    }
  }

  let start = Math.max(0, first - snippetBefore);
  if (start > 0) {
    const space = text.indexOf(" ", start);
    if (space >= 0 && space < first) {
      start = space + 1;
    }
  }
  let end = Math.min(text.length, start + snippetLength);
  if (end < text.length) {
    const space = text.lastIndexOf(" ", end);
    if (space > first) {
      end = space;
    }
  }

  return `${start > 0 ? "…" : ""}${text.slice(start, end)}${end < text.length ? "…" : ""}`;
};

export class ConversationStore {
  readonly dir: string;
  private conversations = new Map<string, StoredConversation>();
  private searchCache = new Map<
    string,
    { messageCount: number; entries: SearchEntry[] }
  >();

  constructor(dir: string) {
    this.dir = dir;
  }

  init(): Message[] {
    fs.mkdirSync(this.dir, { recursive: true, mode: 0o700 });
    const recovered: Message[] = [];

    for (const entry of fs.readdirSync(this.dir, { withFileTypes: true })) {
      if (!entry.isDirectory() || !isValidID(entry.name)) {
        continue;
      }
      const metaPath = path.join(this.dir, entry.name, "conversation.json");
      let stored: StoredConversation;
      try {
        stored = JSON.parse(fs.readFileSync(metaPath, "utf8"));
      } catch {
        continue;
      }

      this.conversations.set(stored.id, stored);

      const pending = this.readPending(stored.id);
      if (pending) {
        const message: Message = {
          ...pending,
          status: "interrupted",
          completedAt: pending.completedAt ?? new Date().toISOString(),
        };
        this.appendMessage(stored.id, message);
        this.clearPending(stored.id);
        recovered.push(message);
      }

      if (stored.activeRunId) {
        this.update(stored.id, { activeRunId: undefined });
      }
    }

    return recovered;
  }

  private conversationDir(id: string): string {
    return path.join(this.dir, id);
  }

  sessionDir(id: string): string {
    return path.join(this.conversationDir(id), "session");
  }

  resultsDir(id: string): string {
    return path.join(this.conversationDir(id), "results");
  }

  private save(stored: StoredConversation) {
    writeFileAtomic(
      path.join(this.conversationDir(stored.id), "conversation.json"),
      JSON.stringify(stored, null, 2),
    );
  }

  list(): Conversation[] {
    return [...this.conversations.values()]
      .map(toConversation)
      .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  }

  get(id: string): Conversation | undefined {
    const stored = this.conversations.get(id);
    return stored ? toConversation(stored) : undefined;
  }

  getSessionFile(id: string): string | undefined {
    return this.conversations.get(id)?.sessionFile;
  }

  setSessionFile(id: string, sessionFile: string | undefined) {
    const stored = this.conversations.get(id);
    if (!stored || stored.sessionFile === sessionFile) {
      return;
    }
    stored.sessionFile = sessionFile;
    this.save(stored);
  }

  create(title?: string): Conversation {
    const now = new Date().toISOString();
    const stored: StoredConversation = {
      id: newID(),
      title: title?.trim() || "New conversation",
      createdAt: now,
      updatedAt: now,
      messageCount: 0,
    };
    fs.mkdirSync(this.conversationDir(stored.id), {
      recursive: true,
      mode: 0o700,
    });
    this.save(stored);
    this.conversations.set(stored.id, stored);
    return toConversation(stored);
  }

  update(
    id: string,
    patch: Partial<Pick<Conversation, "title" | "activeRunId">>,
  ): Conversation | undefined {
    const stored = this.conversations.get(id);
    if (!stored) {
      return undefined;
    }
    if (patch.title !== undefined) {
      stored.title = patch.title.trim() || stored.title;
      stored.updatedAt = new Date().toISOString();
    }
    if ("activeRunId" in patch) {
      if (patch.activeRunId) {
        stored.activeRunId = patch.activeRunId;
      } else {
        delete stored.activeRunId;
      }
    }
    this.save(stored);
    return toConversation(stored);
  }

  delete(id: string): boolean {
    if (!this.conversations.has(id)) {
      return false;
    }
    this.conversations.delete(id);
    this.searchCache.delete(id);
    fs.rmSync(this.conversationDir(id), { recursive: true, force: true });
    return true;
  }

  search(query: string, limit: number): ConversationSearchResult[] {
    const terms = searchTerms(query);
    if (terms.length === 0) {
      return [];
    }

    const ret: ConversationSearchResult[] = [];
    for (const conversation of this.list()) {
      const title = conversation.title.toLowerCase();
      const titleMatch = terms.every((term) => title.includes(term));
      const matches: ConversationSearchMatch[] = [];
      for (const entry of this.searchEntries(conversation)) {
        const snippet = searchSnippet(entry.text, entry.lower, terms);
        if (snippet === undefined) {
          continue;
        }
        matches.push({
          messageId: entry.messageId,
          role: entry.role,
          snippet,
        });
        if (matches.length >= searchMaxMatches) {
          break;
        }
      }

      if (titleMatch || matches.length > 0) {
        ret.push({ conversation, titleMatch, matches });
        if (ret.length >= limit) {
          break;
        }
      }
    }
    return ret;
  }

  private searchEntries(conversation: Conversation): SearchEntry[] {
    const cached = this.searchCache.get(conversation.id);
    if (cached && cached.messageCount === conversation.messageCount) {
      return cached.entries;
    }

    const entries: SearchEntry[] = [];
    for (const message of this.getMessages(conversation.id)) {
      const text = messageText(message);
      if (text !== "") {
        entries.push({
          messageId: message.id,
          role: message.role,
          text,
          lower: text.toLowerCase(),
        });
      }
    }
    this.searchCache.set(conversation.id, {
      messageCount: conversation.messageCount,
      entries,
    });
    return entries;
  }

  getMessages(id: string): Message[] {
    let raw: string;
    try {
      raw = fs.readFileSync(
        path.join(this.conversationDir(id), "messages.jsonl"),
        "utf8",
      );
    } catch {
      return [];
    }

    const ret: Message[] = [];
    for (const line of raw.split("\n")) {
      if (line.trim() === "") {
        continue;
      }
      try {
        ret.push(JSON.parse(line));
      } catch {
        continue;
      }
    }
    return ret;
  }

  appendMessage(id: string, message: Message) {
    const stored = this.conversations.get(id);
    if (!stored) {
      return;
    }
    fs.appendFileSync(
      path.join(this.conversationDir(id), "messages.jsonl"),
      `${JSON.stringify(message)}\n`,
      { mode: 0o600 },
    );
    stored.messageCount++;
    stored.updatedAt = new Date().toISOString();
    this.save(stored);
  }

  savePending(id: string, message: Message) {
    if (!this.conversations.has(id)) {
      return;
    }
    writeFileAtomic(
      path.join(this.conversationDir(id), "pending.json"),
      JSON.stringify(message),
    );
  }

  readPending(id: string): Message | undefined {
    try {
      return JSON.parse(
        fs.readFileSync(
          path.join(this.conversationDir(id), "pending.json"),
          "utf8",
        ),
      );
    } catch {
      return undefined;
    }
  }

  clearPending(id: string) {
    fs.rmSync(path.join(this.conversationDir(id), "pending.json"), {
      force: true,
    });
  }
}
