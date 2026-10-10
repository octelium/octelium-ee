import { strict as assert } from "node:assert";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { Readable } from "node:stream";
import { afterEach, beforeEach, describe, it } from "node:test";
import type { Message } from "../protocol/index.ts";
import {
  ConversationStore,
  isValidID,
  plainText,
  searchSnippet,
  searchTerms,
} from "./conversations.ts";
import {
  FileStore,
  FileTooLargeError,
  guessMimeType,
  sanitizeFileName,
} from "./files.ts";

const message = (conversationId: string, text: string): Message => ({
  id: `m-${text}`,
  conversationId,
  role: "assistant",
  status: "streaming",
  createdAt: new Date().toISOString(),
  blocks: [{ id: "b1", type: "markdown", text, createdAt: "" }],
});

describe("conversation store", () => {
  let dir: string;

  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "console-agent-store-"));
  });

  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("creates, updates, lists and deletes conversations", () => {
    const store = new ConversationStore(dir);
    store.init();
    const a = store.create();
    assert.equal(a.title, "New conversation");
    assert.equal(a.messageCount, 0);
    const b = store.create("  Audit  ");
    assert.equal(b.title, "Audit");

    store.appendMessage(a.id, message(a.id, "hello"));
    assert.equal(store.get(a.id)?.messageCount, 1);
    assert.equal(store.list()[0].id, a.id);
    assert.deepEqual(
      store.getMessages(a.id).map((m) => m.id),
      ["m-hello"],
    );

    store.update(b.id, { title: "Renamed", activeRunId: "r1" });
    assert.equal(store.get(b.id)?.title, "Renamed");
    assert.equal(store.get(b.id)?.activeRunId, "r1");
    store.update(b.id, { activeRunId: undefined });
    assert.equal(store.get(b.id)?.activeRunId, undefined);

    store.setSessionFile(a.id, "/tmp/session.jsonl");
    assert.equal(store.getSessionFile(a.id), "/tmp/session.jsonl");
    assert.equal("sessionFile" in store.get(a.id)!, false);

    assert.equal(store.delete(a.id), true);
    assert.equal(store.get(a.id), undefined);
    assert.equal(store.delete(a.id), false);
  });

  it("reloads the conversations and recovers the interrupted messages", () => {
    const store = new ConversationStore(dir);
    store.init();
    const conv = store.create("Recover");
    store.update(conv.id, { activeRunId: "r1" });
    store.savePending(conv.id, message(conv.id, "partial"));
    store.setSessionFile(conv.id, "/tmp/s.jsonl");

    const reloaded = new ConversationStore(dir);
    const recovered = reloaded.init();
    assert.equal(recovered.length, 1);
    assert.equal(recovered[0].status, "interrupted");
    const loaded = reloaded.get(conv.id)!;
    assert.equal(loaded.title, "Recover");
    assert.equal(loaded.activeRunId, undefined);
    assert.equal(reloaded.getSessionFile(conv.id), "/tmp/s.jsonl");
    assert.deepEqual(
      reloaded.getMessages(conv.id).map((m) => [m.id, m.status]),
      [["m-partial", "interrupted"]],
    );
    assert.equal(reloaded.readPending(conv.id), undefined);
  });

  it("searches the titles and the messages", () => {
    const store = new ConversationStore(dir);
    store.init();
    const audit = store.create("Security audit");
    const policies = store.create("Policies");
    store.appendMessage(policies.id, {
      ...message(policies.id, "List the Policies of the Group admins"),
      id: "m-user",
      role: "user",
    });
    store.appendMessage(
      policies.id,
      message(policies.id, "The Group **admins** has 3 Policies attached"),
    );

    assert.deepEqual(store.search("   ", 10), []);

    const byTitle = store.search("AUDIT", 10);
    assert.deepEqual(
      byTitle.map((r) => [r.conversation.id, r.titleMatch, r.matches.length]),
      [[audit.id, true, 0]],
    );

    const byContent = store.search("admins policies", 10);
    assert.equal(byContent.length, 1);
    assert.equal(byContent[0].conversation.id, policies.id);
    assert.equal(byContent[0].titleMatch, false);
    assert.deepEqual(
      byContent[0].matches.map((m) => [m.messageId, m.role]),
      [
        ["m-user", "user"],
        ["m-The Group **admins** has 3 Policies attached", "assistant"],
      ],
    );
    assert.equal(
      byContent[0].matches[1].snippet,
      "The Group admins has 3 Policies attached",
    );

    assert.deepEqual(store.search("admins audit", 10), []);
    assert.equal(store.search("policies", 1).length, 1);

    store.appendMessage(audit.id, message(audit.id, "No admins were found"));
    assert.deepEqual(
      store.search("admins", 10).map((r) => r.conversation.id),
      [audit.id, policies.id],
    );

    store.delete(policies.id);
    assert.deepEqual(
      store.search("admins", 10).map((r) => r.conversation.id),
      [audit.id],
    );
  });

  it("converts the markdown to plain text", () => {
    assert.equal(
      plainText(
        "## Summary\n\n- `api.prod` is **healthy**\n> see [the docs](https://x.y)\n\n```bash\nls -la\n```\n| a | b |",
      ),
      "Summary api.prod is healthy see the docs ls -la a b",
    );
    assert.equal(plainText("allow_all_prod"), "allow_all_prod");
  });

  it("builds the search snippets", () => {
    assert.deepEqual(searchTerms("  Foo bar   foo "), ["foo", "bar"]);

    const text = `${"lorem ".repeat(30)}needle ${"ipsum ".repeat(60)}`.trim();
    const snippet = searchSnippet(text, text.toLowerCase(), ["needle"])!;
    assert.ok(snippet.startsWith("…lorem"));
    assert.ok(snippet.endsWith("ipsum…"));
    assert.ok(snippet.includes("needle"));
    assert.ok(snippet.length <= 182);

    assert.equal(
      searchSnippet("short text", "short text", ["text"]),
      "short text",
    );
    assert.equal(
      searchSnippet("short text", "short text", ["text", "missing"]),
      undefined,
    );
  });

  it("validates the IDs", () => {
    assert.equal(isValidID("2b3f6a0e-1c2d-4e5f-8a9b-0c1d2e3f4a5b"), true);
    assert.equal(isValidID("../etc"), false);
    assert.equal(isValidID(""), false);
    assert.equal(isValidID("a".repeat(65)), false);
  });
});

describe("file store", () => {
  let dir: string;

  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), "console-agent-files-"));
  });

  afterEach(() => {
    fs.rmSync(dir, { recursive: true, force: true });
  });

  it("saves the uploads and enforces the size limit", async () => {
    const store = new FileStore(dir);
    store.init();
    const info = await store.saveUpload(
      "../../evil name.json",
      undefined,
      Readable.from([Buffer.from('{"a":1}')]),
      100,
    );
    assert.equal(info.name, "evil name.json");
    assert.equal(info.mimeType, "application/json");
    assert.equal(info.size, 7);
    assert.ok(info.path.startsWith(path.join(dir, "uploads")));
    assert.deepEqual(store.getUpload(info.id), info);
    assert.equal(store.getUpload("../x"), undefined);

    await assert.rejects(
      store.saveUpload(
        "big.bin",
        undefined,
        Readable.from([Buffer.alloc(200)]),
        100,
      ),
      FileTooLargeError,
    );
    assert.equal(fs.readdirSync(path.join(dir, "uploads")).length, 1);
  });

  it("publishes the artifacts as copies", () => {
    const store = new FileStore(dir);
    store.init();
    const source = path.join(dir, "report.csv");
    fs.writeFileSync(source, "a,b\n1,2\n");
    const info = store.publishArtifact(source, { title: "Report" });
    assert.equal(info.name, "report.csv");
    assert.equal(info.mimeType, "text/csv");
    assert.equal(info.size, 8);
    assert.equal(info.url, `/v1/artifacts/${info.id}/content`);

    fs.writeFileSync(source, "changed");
    const stored = store.getArtifact(info.id)!;
    assert.equal(fs.readFileSync(stored.filePath, "utf8"), "a,b\n1,2\n");
    assert.throws(() => store.publishArtifact(dir));
    assert.equal(store.getArtifact("nope"), undefined);
  });

  it("sanitizes the names and guesses the MIME types", () => {
    assert.equal(sanitizeFileName("a/b/c.txt"), "c.txt");
    assert.equal(sanitizeFileName("..\\..\\x.txt"), "x.txt");
    assert.equal(sanitizeFileName(".hidden"), "_hidden");
    assert.equal(sanitizeFileName("a\u0000b?.txt"), "a_b_.txt");
    assert.equal(sanitizeFileName(""), "file");
    assert.equal(guessMimeType("x.PNG"), "image/png");
    assert.equal(guessMimeType("x.unknown"), "application/octet-stream");
  });
});
