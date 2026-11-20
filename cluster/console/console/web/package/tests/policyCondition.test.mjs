import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { after, before, describe, it } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

const schema = JSON.parse(
  readFileSync(new URL("../src/jsonschema/core/RequestContext.json", import.meta.url)),
);

const resolve = (node) => {
  let current = node;
  while (current?.$ref) current = schema.definitions[current.$ref.split("/").at(-1)];
  return current;
};

const hasPath = (path) => {
  let node = resolve({ $ref: schema.$ref });
  for (const segment of path) {
    if (!node) return false;
    if (node.type === "object" && node.additionalProperties && !node.properties?.[segment]) {
      node = resolve(node.additionalProperties);
      continue;
    }
    node = resolve(node.properties?.[segment]);
  }
  return node !== undefined;
};

describe("Policy conditions", () => {
  let modules;
  let cel;
  let blocks;
  let model;
  let recipes;
  let core;
  let ai;

  const match = (text) => core.Condition.create({ type: { oneofKind: "match", match: text } });
  const not = (text) => core.Condition.create({ type: { oneofKind: "not", not: text } });
  const group = (kind, of) => core.Condition.create({ type: { oneofKind: kind, [kind]: { of } } });

  before(async () => {
    modules = await createConsoleModules();
    [cel, blocks, model, recipes, core, ai] = await Promise.all([
      modules.load("components/Condition/cel.ts"),
      modules.load("components/Condition/blocks.ts"),
      modules.load("components/Condition/model.ts"),
      modules.load("components/Condition/recipes.ts"),
      modules.load("apis/corev1/corev1.ts"),
      modules.load("pages/core/Policy/edit/ai.ts"),
    ]);
  });
  after(async () => modules?.close());

  describe("CEL text helpers", () => {
    it("splits on top-level operators only", () => {
      assert.deepEqual(cel.splitTopLevel('a && (b || c) && "x && y"', "&&"), ["a", "(b || c)", '"x && y"']);
      assert.deepEqual(cel.splitTopLevel("a && b || c", "||"), ["a && b", "c"]);
      assert.equal(cel.splitTopLevel("a && b || c", "&&"), undefined);
      assert.equal(cel.splitTopLevel("a ? b && c : d", "&&"), undefined);
      assert.equal(cel.splitTopLevel("f(a && b)", "&&"), undefined);
      assert.deepEqual(cel.splitTopLevel("(a && b)", "&&"), ["a", "b"]);
      assert.deepEqual(cel.splitTopLevel("a && b // x || y", "&&"), ["a", "b // x || y"]);
      assert.equal(cel.splitTopLevel("r'\\' && b'", "&&"), undefined);
    });

    it("unwraps only fully enclosing parentheses", () => {
      assert.equal(cel.unwrapParens("((a && b))"), "a && b");
      assert.equal(cel.unwrapParens("(a) && (b)"), "(a) && (b)");
      assert.equal(cel.unwrapParens('(")")'), '")"');
    });

    it("detects negations of whole expressions", () => {
      assert.equal(cel.negatedInner("!(a == b)"), "a == b");
      assert.equal(cel.negatedInner("!ctx.session.status.isBrowser"), "ctx.session.status.isBrowser");
      assert.equal(cel.negatedInner("!a == b"), undefined);
      assert.equal(cel.negatedInner("!(a) && b"), undefined);
      assert.equal(cel.negatedInner("a != b"), undefined);
    });

    it("checks that expressions look complete", () => {
      for (const text of ["true", 'a == "("', "f(x) // done", "[1, 2].exists(x, x > 1)"]) {
        assert.equal(cel.looksComplete(text), true, text);
      }
      for (const text of ["", "  ", "a ==", "a &&", "f(", '"open', "a in", "x."]) {
        assert.equal(cel.looksComplete(text), false, text);
      }
    });

    it("compares expressions without whitespace differences", () => {
      assert.equal(cel.sameCEL('a  ==  "x  y"', '(a=="x  y")'), true);
      assert.equal(cel.sameCEL('a == "x y"', 'a == "xy"'), false);
      assert.equal(cel.sameCEL("x in y", "xin y"), false);
    });

    it("parses CEL string literals and lists", () => {
      assert.equal(cel.parseStringLiteral('"a\\"b"'), 'a"b');
      assert.equal(cel.parseStringLiteral("'it\\'s \"x\"'"), 'it\'s "x"');
      assert.deepEqual(cel.parseStringList('["a", \'b\' ,"c",]'), ["a", "b", "c"]);
      assert.deepEqual(cel.parseStringList("[]"), []);
      assert.equal(cel.parseStringList('["a", b]'), undefined);
    });
  });

  describe("Blocks", () => {
    const sample = (block) => {
      const values = block.defaults();
      for (const [key, field] of block.fields) {
        switch (field.kind) {
          case "refs":
            values[key] = ["alpha", "beta"];
            break;
          case "options":
            values[key] = field.options.slice(0, 2).map((option) => option.value);
            break;
          case "countries":
            values[key] = ["DE", "FR"];
            break;
          case "strings":
            values[key] = field.transform === "cidr" ? ["10.0.0.0/8", "192.168.1.0/24"] : ["MainService", "UserService"];
            break;
          case "text":
            values[key] = "x-tenant";
            break;
          case "match":
            values[key] = { op: values[key]?.op ?? "in", values: ["one", "two"] };
            break;
          case "number":
            values[key] = { op: ">=", value: 42 };
            break;
          case "timestamp":
            values[key] = key === "after" ? "2026-01-02T03:04:05.000Z" : "2026-02-02T03:04:05.000Z";
            break;
          case "switch":
            values[key] = true;
            break;
        }
      }
      return values;
    };

    it("have unique identifiers and known categories", () => {
      const ids = blocks.BLOCKS.map((block) => block.id);
      assert.equal(new Set(ids).size, ids.length);
      const categories = new Set(blocks.CATEGORIES.map((category) => category.id));
      for (const block of blocks.BLOCKS) assert.ok(categories.has(block.category), block.id);
    });

    for (const variant of ["defaults", "sample"]) {
      it(`recognize their own CEL with ${variant} values`, () => {
        for (const block of blocks.BLOCKS) {
          const values = variant === "sample" ? sample(block) : block.defaults();
          if (block.issues(values).length > 0) continue;
          const expr = block.toCEL(values);
          const found = blocks.recognize(expr);
          assert.equal(found?.block.id, block.id, `${block.id}: ${expr}`);
          assert.equal(found.block.toCEL(found.values), expr, block.id);
        }
      });
    }

    it("only reference fields of the request context", () => {
      for (const block of blocks.BLOCKS) {
        const expr = block.toCEL(sample(block));
        const literals = expr.replace(/"(?:[^"\\]|\\.)*"/g, '""');
        for (const path of literals.match(/ctx(?:\.\w+)+/g) ?? []) {
          const segments = path.split(".").slice(1);
          const field = segments.at(-1);
          const known = hasPath(segments) || (["startsWith", "endsWith", "contains", "matches", "exists"].includes(field) && hasPath(segments.slice(0, -1)));
          assert.ok(known, `${block.id}: ${path}`);
        }
      }
    });

    it("report missing values", () => {
      assert.deepEqual(blocks.getBlock("identity.group").issues({ groups: [], mode: "any" }), ["Choose at least one Group"]);
      assert.deepEqual(blocks.getBlock("identity.user").issues({ names: [] }), ["Choose at least one user"]);
      assert.deepEqual(blocks.getBlock("http.path").issues({ match: { op: "prefix", values: [""] } }), ["Enter the path"]);
      assert.deepEqual(blocks.getBlock("network.ipRange").issues({ ranges: ["10.0.0.0/33"] }), ["Fix the invalid range 10.0.0.0/33"]);
      assert.deepEqual(blocks.getBlock("time.window").issues({ days: [], from: "", to: "" }), ["Choose the days or the hours of the window"]);
      assert.deepEqual(blocks.getBlock("time.window").issues({ days: ["1"], from: "09:00", to: "09:00" }), ["The window must start and end at different times"]);
      assert.deepEqual(blocks.getBlock("time.range").issues({ after: "2026-02-01T00:00:00Z", before: "2026-01-01T00:00:00Z" }), ["The end date must come after the start date"]);
    });

    it("generate readable CEL", () => {
      const cases = [
        ["identity.group", { groups: ["sre"], mode: "any" }, '"sre" in ctx.user.spec.groups'],
        ["identity.group", { groups: ["sre", "ops"], mode: "all" }, '["sre", "ops"].all(g, g in ctx.user.spec.groups)'],
        ["identity.user", { names: ["alice", "bob"] }, 'ctx.user.metadata.name in ["alice", "bob"]'],
        ["http.method", { values: ["GET", "HEAD", "OPTIONS"] }, 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]'],
        ["http.path", { match: { op: "prefix", values: ["/a", "/b"] } }, '["/a", "/b"].exists(x, ctx.request.http.path.startsWith(x))'],
        ["http.hasHeader", { key: "X-Key" }, '"x-key" in ctx.request.http.headers'],
        ["http.header", { key: "X-Tenant", match: { op: "is", values: ["a"] } }, 'ctx.request.http.headers["x-tenant"] == "a"'],
        ["network.ipRange", { ranges: ["10.0.0.0/8"] }, 'net.isIPInRange(ctx.request.ip, "10.0.0.0/8")'],
        ["time.window", { days: ["5", "1", "2", "3", "4"], from: "09:00", to: "17:00", timezone: "Europe/Berlin" }, 'now().getDayOfWeek("Europe/Berlin") in [1, 2, 3, 4, 5] && now().getHours("Europe/Berlin") >= 9 && now().getHours("Europe/Berlin") < 17'],
        ["time.window", { days: [], from: "22:30", to: "06:00", timezone: "" }, "(now().getHours() * 60 + now().getMinutes() >= 1350 || now().getHours() * 60 + now().getMinutes() < 360)"],
        ["api.octelium", { apis: ["octelium.api.main.core.v1"], services: [], methods: [], readOnly: true }, 'ctx.service.status.namespaceRef.name == "octelium-api" && ctx.service.spec.mode == "GRPC" && ctx.request.grpc.package == "octelium.api.main.core.v1" && ["Get", "List"].exists(x, ctx.request.grpc.method.startsWith(x))'],
        ["mcp.tool", { match: { op: "in", values: ["a", "b"] } }, 'ctx.request.mcp.method == "tools/call" && ctx.request.mcp.name in ["a", "b"]'],
        ["protocol.socks5Port", { number: { op: ">", value: 1024 } }, "ctx.request.socks5.connect.port > uint(1024)"],
        ["device.posture", { checks: ["threatFree"], grace: true }, 'ctx.device.status.posture.threatFree in ["PASS", "GRACE_PERIOD"]'],
      ];
      for (const [id, values, expected] of cases) {
        assert.equal(blocks.getBlock(id).toCEL(values), expected, id);
      }
    });

    it("recognize the expressions of the former server-side builder and built-in Policies", () => {
      const cases = [
        ['ctx.user.metadata.name == "usr"', "identity.user", { names: ["usr"] }],
        ['ctx.session.status.authentication.info.aal == "AAL2"', "session.aal", { values: ["AAL2"] }],
        ['ctx.session.status.authentication.info.geoip.country.code in ["DE", "FR"]', "network.country", { codes: ["DE", "FR"] }],
        ['now() > timestamp("2026-01-02T03:04:05Z")', "time.range", { after: "2026-01-02T03:04:05Z", before: "" }],
        ['time.isWeekendInTZ(now(), "America/Los_Angeles")', "time.dayType", { type: "weekend", timezone: "America/Los_Angeles" }],
        ['ctx.request.http.headers["x-forwarded-for"] == "10.0.0.1"', "http.header", { key: "x-forwarded-for", match: { op: "is", values: ["10.0.0.1"] } }],
        ['ctx.request.ip == "10.0.0.1"', "network.ipRange", { ranges: ["10.0.0.1/32"] }],
        ['(ctx.request.mcp.method == "tools/call") && (ctx.request.mcp.name in ["search", "read_db"])', "mcp.tool", { match: { op: "in", values: ["search", "read_db"] } }],
        ['ctx.request.llm.toolNames.exists(x, x.startsWith("mcp_"))', "llm.toolName", { match: { op: "prefix", values: ["mcp_"] } }],
        ['ctx.namespace.metadata.name == "octelium-api" && ctx.service.spec.mode == "GRPC"', "api.octelium", { apis: [], services: [], methods: [], readOnly: false }],
        ['(ctx.service.status.namespaceRef.name == "octelium-api" && ctx.service.spec.mode == "GRPC") && (ctx.request.grpc.package == "octelium.api.main.enterprise.v1") && (ctx.request.grpc.service == "MainService")', "api.octelium", { apis: ["octelium.api.main.enterprise.v1"], services: ["MainService"], methods: [], readOnly: false }],
        ['ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]', "http.method", { values: ["GET", "HEAD", "OPTIONS"] }],
        ["ctx.request.socks5.connect.port > uint(1024)", "protocol.socks5Port", { number: { op: ">", value: 1024 } }],
        ["'sre' in ctx.user.spec.groups", "identity.group", { groups: ["sre"], mode: "any" }],
      ];
      for (const [expr, id, values] of cases) {
        const found = blocks.recognize(expr);
        assert.equal(found?.block.id, id, expr);
        assert.deepEqual(found.values, values, expr);
      }
    });

    it("leave unknown or partial expressions to CEL", () => {
      for (const expr of [
        'ctx.user.metadata.name == "a" || true',
        'ctx.request.http.headers.exists(k, k == "x-auth")',
        'ctx.request.mcp.name == "x"',
        "now().getHours() >= 9",
        'json.parse(ctx.request.http.headers["x-meta"]).priority > 5.0',
      ]) {
        assert.equal(blocks.recognize(expr), undefined, expr);
      }
    });
  });

  describe("Model", () => {
    const corpus = () => [
      core.Condition.create({ type: { oneofKind: "matchAny", matchAny: true } }),
      core.Condition.create({ type: { oneofKind: "matchAny", matchAny: false } }),
      match('"sre" in ctx.user.spec.groups'),
      match('  "sre"   in ctx.user.spec.groups  '),
      not('ctx.request.http.method in ["GET"]'),
      match("!(ctx.session.status.authentication.info.aal in [\"AAL2\", \"AAL3\"])"),
      match("!ctx.session.status.isBrowser"),
      match('json.parse(ctx.request.http.headers["x-meta"]).strict == true'),
      match(""),
      not("custom(ctx)"),
      group("all", [match("a"), group("any", [match("b"), not("c")]), group("none", [])]),
      group("none", [match('ctx.user.spec.type == "HUMAN"'), match('ctx.device.status.osType == "LINUX"')]),
      core.Condition.create({ type: { oneofKind: "opa", opa: { type: { oneofKind: "inline", inline: "package x\nallow { true }" } } } }),
      core.Condition.create({ type: { oneofKind: undefined } }),
      group("all", [core.Condition.create({ type: { oneofKind: undefined } })]),
    ];

    it("compiles a parsed condition back to exactly the same condition", () => {
      for (const condition of corpus()) {
        const node = model.parseCondition(condition);
        const compiled = model.compileCondition(node);
        if (condition.type.oneofKind === undefined) {
          assert.equal(compiled, undefined);
          continue;
        }
        assert.ok(core.Condition.equals(compiled, condition), JSON.stringify(core.Condition.toJson(condition)));
      }
    });

    it("shows recognized leaves as blocks and keeps the core negation kind", () => {
      const leaf = model.parseCondition(not('"sre" in ctx.user.spec.groups'));
      assert.equal(leaf.kind, "block");
      assert.equal(leaf.negate, "condition");
      const expression = model.parseCondition(match('!("sre" in ctx.user.spec.groups)'));
      assert.equal(expression.kind, "block");
      assert.equal(expression.negate, "expression");
      const edited = model.touch(expression, { values: { groups: ["ops"], mode: "any" } });
      assert.equal(model.compileNode(edited).type.match, '!("ops" in ctx.user.spec.groups)');
      const conditionEdited = model.touch(leaf, { values: { groups: ["ops"], mode: "any" } });
      assert.equal(model.compileNode(conditionEdited).type.not, '"ops" in ctx.user.spec.groups');
    });

    it("regenerates CEL only for edited leaves", () => {
      const node = model.parseCondition(group("all", [match('  "sre" in ctx.user.spec.groups'), match('ctx.user.spec.type  ==  "HUMAN"')]));
      const edited = model.replaceNode(node, node.children[1].key, model.touch(node.children[1], { values: { values: ["WORKLOAD"] } }));
      const compiled = model.compileNode(edited);
      assert.equal(compiled.type.all.of[0].type.match, '  "sre" in ctx.user.spec.groups');
      assert.equal(compiled.type.all.of[1].type.match, 'ctx.user.spec.type == "WORKLOAD"');
    });

    it("renders a whole tree as one CEL expression", () => {
      const node = model.parseCondition(group("all", [
        match('"sre" in ctx.user.spec.groups'),
        group("any", [match("a == 1"), match("b == 2")]),
        not("c"),
        group("none", [match("d"), match("e")]),
      ]));
      assert.equal(model.nodeCEL(node), '"sre" in ctx.user.spec.groups && (a == 1 || b == 2) && !c && !(d || e)');
      assert.equal(model.nodeCEL(node, true), '"sre" in ctx.user.spec.groups\n&& (a == 1 || b == 2)\n&& !c\n&& !(d || e)');
      assert.equal(model.hasCoreNegation(node), true);
      assert.equal(model.nodeCEL(model.parseCondition(group("any", [match("a && b"), match("c")]))), "(a && b) || c");
      assert.equal(model.nodeCEL(model.parseCondition(core.Condition.create({ type: { oneofKind: "opa", opa: { type: { oneofKind: "inline", inline: "x" } } } }))), undefined);
    });

    it("splits a CEL expression into groups and recognized blocks", () => {
      const node = model.splitCEL('"sre" in ctx.user.spec.groups && (ctx.request.http.method == "GET" || ctx.request.http.path.startsWith("/public")) && (x && y)');
      assert.equal(node.kind, "group");
      assert.equal(node.op, "all");
      assert.deepEqual(node.children.map((child) => child.kind), ["block", "group", "cel", "cel"]);
      assert.equal(node.children[0].block, "identity.group");
      assert.deepEqual(node.children[1].children.map((child) => child.block), ["http.method", "http.path"]);
      assert.equal(model.canSplit("a && b"), true);
      assert.equal(model.canSplit("f(a && b)"), false);
    });

    it("lists the issues of incomplete trees", () => {
      const tree = model.groupNode("all", [
        model.blockNode("identity.group"),
        model.celNode(""),
        model.celNode("a =="),
        model.groupNode("any", []),
        model.opaNode(""),
      ]);
      assert.deepEqual(model.conditionIssues(tree), [
        "Choose at least one Group",
        "Write the CEL expression",
        "Finish the CEL expression",
        "Add a condition to the empty group",
        "Write the Rego policy",
      ]);
      assert.deepEqual(model.conditionIssues(undefined), ["Add a condition"]);
      assert.deepEqual(model.conditionIssues(model.parseCondition(match("a =="))), []);
    });

    it("summarizes trees in plain words", () => {
      const tree = model.groupNode("all", [
        model.blockNode("identity.group", { groups: ["sre", "ops"], mode: "any" }),
        model.groupNode("any", [
          model.blockNode("http.method", { values: ["GET"] }),
          model.blockNode("session.aal", { values: ["AAL2"] }, "condition"),
        ]),
      ]);
      assert.equal(model.summarize(tree), "User is in sre or ops and (Method is GET or not (Assurance level is AAL2))");
    });

    it("adds conditions next to the root or inside a group", () => {
      const first = model.blockNode("identity.user", { names: ["a"] });
      let root = model.appendNodes(undefined, [first]);
      assert.equal(root, first);
      root = model.appendNodes(root, [model.everyNode()]);
      assert.equal(root.kind, "group");
      assert.equal(root.children.length, 2);
      const nested = model.groupNode("any", []);
      root = model.appendNodes(root, [nested]);
      root = model.appendNodes(root, [model.celNode("x")], nested.key);
      assert.equal(model.findNode(root, nested.key).children.length, 1);
      root = model.replaceNode(root, first.key, undefined);
      assert.equal(root.children.length, 2);
    });
  });

  describe("Recipes", () => {
    it("build valid trees with unique identifiers", () => {
      const ids = recipes.RECIPES.map((recipe) => recipe.id);
      assert.equal(new Set(ids).size, ids.length);
      for (const recipe of recipes.RECIPES) {
        const node = recipe.build();
        const compiled = model.compileNode(node);
        assert.ok(compiled.type.oneofKind, recipe.id);
        for (const issue of model.nodeIssues(node)) {
          assert.match(issue, /^(Choose|Enter)/, `${recipe.id}: ${issue}`);
        }
      }
    });

    it("build starter Policies whose complete rules compile", () => {
      for (const starter of recipes.STARTERS) {
        const { rules, enforcementRules } = starter.build();
        assert.ok(rules.length > 0, starter.id);
        for (const item of [...rules, ...enforcementRules]) {
          assert.ok(item.node, starter.id);
          if (model.isNodeComplete(item.node)) {
            const text = model.nodeCEL(item.node);
            assert.ok(text === undefined || cel.looksComplete(text), `${starter.id}: ${text}`);
          }
        }
      }
      const http = recipes.STARTERS.find((starter) => starter.id === "http-read-only").build();
      assert.equal(model.compileNode(http.enforcementRules[0].node).type.not, 'ctx.service.spec.mode in ["HTTP", "WEB"]');
      assert.equal(model.compileNode(http.rules[0].node).type.match, 'ctx.request.http.method in ["GET", "HEAD", "OPTIONS"]');
    });

    it("searches across words in any order", () => {
      const text = recipes.searchText("Read-only HTTP", ["safe methods"]);
      assert.equal(recipes.matchesQuery(text, "http read"), true);
      assert.equal(recipes.matchesQuery(text, "kubernetes"), false);
    });
  });

  describe("AI helpers", () => {
    it("parses a Policy spec or a whole Policy proposed by the AI", () => {
      const spec = "rules:\n  - name: a\n    effect: ALLOW\n    condition:\n      match: 'true'\n";
      const parsed = ai.parseSpec(spec);
      assert.equal(parsed.spec.rules[0].effect, core.Policy_Spec_Rule_Effect.ALLOW);
      const wrapped = ai.parseSpec(`kind: Policy\nspec:\n  ${spec.replace(/\n/g, "\n  ")}`);
      assert.equal(wrapped.spec.rules[0].name, "a");
      assert.equal(ai.parseSpec("rules: [").error.startsWith("The YAML cannot be parsed"), true);
      assert.deepEqual(ai.parseSpec("- a"), { error: "The code is not a Policy spec" });
      assert.deepEqual(ai.parseSpec("foo: bar"), { error: "The spec has no rules" });
      assert.deepEqual(ai.parseSpec("rules:\n  - name: a\n    effect: ALLOW\n"), { error: "Rule #1 has no condition or effect" });
      assert.match(ai.parseSpec("rules:\n  - priority: high\n").error, /^The spec is invalid/);
    });

    it("only applies complete CEL expressions", () => {
      assert.equal(ai.extractCEL(' "sre" in ctx.user.spec.groups \n'), '"sre" in ctx.user.spec.groups');
      assert.equal(ai.extractCEL("ctx.user.spec.type =="), undefined);
      assert.equal(ai.extractCEL(""), undefined);
    });

    it("describes the evaluation rules and the expected answer format", () => {
      const rule = ai.rulePrompt({ action: "write", text: "sre on weekdays", kind: "rule", name: "r", index: 0, effect: "ALLOW", priority: -1, cel: "true" });
      assert.match(rule, /^Write the condition of the rule "r" of the Policy draft in the console context \(effect ALLOW, priority -1\)\. It must match: sre on weekdays/);
      assert.match(rule, /exactly one ```cel code block/);
      assert.match(rule, /DENY before ALLOW at equal priority/);
      const review = ai.policyPrompt({ action: "review", text: "", name: "p", yaml: "rules: []\n", empty: false });
      assert.match(review, /IsAuthorized method/);
      assert.match(review, /```yaml\nrules: \[\]\n```/);
      assert.match(ai.specToYAML(core.Policy_Spec.create({ isDisabled: true })), /isDisabled: true/);
    });
  });
});
