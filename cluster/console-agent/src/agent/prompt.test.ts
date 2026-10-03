import { strict as assert } from "node:assert";
import { describe, it } from "node:test";
import { buildSystemPrompt, type PromptContext } from "./prompt.ts";

const base: PromptContext = {
  domain: "example.com",
  user: { name: "alice", displayName: "Alice" },
  workspace: { name: "abc" },
  workDir: "/workspace",
  octeliumMode: "proxy",
  apiIndex: "core.v1 — Core\n  MainService: Service[List|Get]",
  tools: ["bash", "read", "octelium_api_call"],
  approvals: { octeliumAPI: "write", bash: false },
  date: new Date("2026-10-03T10:00:00Z"),
};

describe("system prompt", () => {
  it("describes the environment, the APIs and the presentation rules", () => {
    const prompt = buildSystemPrompt(base);
    assert.match(prompt, /^You are the Octelium Console Agent/);
    assert.match(prompt, /\(domain: example\.com\)/);
    assert.match(prompt, /Cordium Workspace \("abc"\)/);
    assert.match(prompt, /Your working directory is \/workspace/);
    assert.match(prompt, /the Octelium User "alice" \(Alice\)/);
    assert.match(prompt, /<service>\.<namespace>\.local\.example\.com/);
    assert.match(prompt, /Never run `octelium login`/);
    assert.match(prompt, /paused until the User approves them/);
    assert.match(prompt, /octelium:\/\/resource\/<apiVersion>\/<Kind>\/<name>/);
    assert.match(prompt, /# Current date\n2026-10-03/);
    assert.match(
      prompt,
      /# API index\n.*\ncore\.v1 — Core\n  MainService: Service\[List\|Get\]/,
    );
    assert.doesNotMatch(prompt, /NOT available/);
    assert.doesNotMatch(prompt, /Additional instructions/);
  });

  it("adapts to the configuration", () => {
    const prompt = buildSystemPrompt({
      ...base,
      user: undefined,
      tools: ["octelium_api_call"],
      octeliumMode: "disabled",
      octeliumUnavailableReason: "no socket",
      approvals: { octeliumAPI: "never", bash: false },
      append: "  Always answer in French.  ",
    });
    assert.match(prompt, /the signed-in Octelium User/);
    assert.doesNotMatch(prompt, /octeliumctl/);
    assert.match(prompt, /NOT available to you \(no socket\)/);
    assert.match(prompt, /executed directly without an approval step/);
    assert.match(
      prompt,
      /# Additional instructions\nAlways answer in French\.$/,
    );
  });
});
