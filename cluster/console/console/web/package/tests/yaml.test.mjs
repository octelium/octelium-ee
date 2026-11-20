import assert from "node:assert/strict";
import { after, before, describe, it } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Resource YAML compatibility", () => {
  let modules;
  let loadYAML;

  before(async () => {
    modules = await createConsoleModules();
    ({ loadYAML } = await modules.load("utils/pb/yaml.ts"));
  });

  after(async () => {
    await modules?.close();
  });

  for (const source of ["", " \n\t", "# resource", " \r# draft\r\n "]) {
    it(`keeps empty resource input undefined for ${JSON.stringify(source)}`, () => {
      assert.equal(loadYAML(source), undefined);
    });
  }

  for (const [source, expected] of [
    ["yes", "yes"],
    ["NO", "NO"],
    ["on", "on"],
    ["Off", "Off"],
    ["true", true],
    ["FALSE", false],
    ["null", null],
    ["~", null],
    ["012", 12],
    ["0b101", 5],
    ["-0b101", -5],
    ["+0b101", 5],
    ["0o12", 10],
    ["-0o12", -10],
    ["0x10", 16],
    ["+0x10", 16],
    ["1_000", "1_000"],
    ["1:30", "1:30"],
    ["1e3", 1000],
    [".5", 0.5],
    ["-.5", "-.5"],
    ["+.5", "+.5"],
    ["-0.5", -0.5],
    ["1e999", "1e999"],
    ["!!int 0b101", 5],
    ["!!float 1e3", 1000],
  ]) {
    it(`preserves the existing interpretation of ${source}`, () => {
      assert.deepEqual(loadYAML(`value: ${source}`), { value: expected });
    });
  }

  it("preserves timestamp and binary values", () => {
    assert.deepEqual(loadYAML("value: 2026-11-20"), {
      value: new Date("2026-11-20T00:00:00.000Z"),
    });
    assert.deepEqual(loadYAML("value: !!binary SGVsbG8="), {
      value: new Uint8Array([72, 101, 108, 108, 111]),
    });
  });

  it("keeps aliases and merge precedence", () => {
    const value = loadYAML(`
first: &first {team: platform, owner: first}
second: &second {owner: second, region: west}
alias: *first
merged:
  <<: [*first, *second]
  team: console
`);
    assert.equal(value.alias, value.first);
    assert.deepEqual(value.merged, {
      team: "console",
      owner: "first",
      region: "west",
    });
  });

  it("keeps legacy collection representations", () => {
    assert.deepEqual(loadYAML("!!set {one, two}"), {
      one: null,
      two: null,
    });
    assert.deepEqual(loadYAML("!!omap [{one: 1}, {two: 2}]"), [
      { one: 1 },
      { two: 2 },
    ]);
    assert.deepEqual(loadYAML("!!pairs [{one: 1}, {one: 2}]"), [
      ["one", 1],
      ["one", 2],
    ]);
    assert.deepEqual(loadYAML("[one, two]: value"), {
      "one,two": "value",
    });
  });

  it("treats prototype keys as data", () => {
    const value = loadYAML("__proto__: {polluted: true}");
    assert.deepEqual(value, JSON.parse('{"__proto__":{"polluted":true}}'));
    assert.equal(Object.getPrototypeOf(value), Object.prototype);
    assert.equal({}.polluted, undefined);
  });

  for (const source of [
    "value: [unterminated",
    "value: 1\nvalue: 2",
    "value: !!unknown test",
    "!!set {one: 1}",
    "value: 1\n---\nvalue: 2",
  ]) {
    it(`rejects invalid or multiple documents ${JSON.stringify(source)}`, () => {
      assert.throws(() => loadYAML(source));
    });
  }
});
