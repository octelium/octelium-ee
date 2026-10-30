import assert from "node:assert/strict";
import { after, afterEach, before, beforeEach, describe, it, mock } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

let modules;
let query;
let paths;
let match;
let recents;
let layouts;
let settings;
let environment;
let meta;
const storageKey = "octelium-console-recents";
const previousStorage = Object.getOwnPropertyDescriptor(globalThis, "localStorage");

before(async () => {
  modules = await createConsoleModules();
  [query, paths, match, recents, layouts, settings, environment, meta] = await Promise.all([
    modules.load("components/ResourceLayout/queryParse.ts"),
    modules.load("utils/pb/paths.ts"),
    modules.load("components/CommandPalette/match.ts"),
    modules.load("components/CommandPalette/recents.ts"),
    modules.load("features/dashboardLayout/slice.ts"),
    modules.load("features/settings/slice.ts"),
    modules.load("pages/Agent/environment.ts"),
    modules.load("apis/metav1/metav1.ts"),
  ]);
});

beforeEach(() => {
  const values = new Map();
  Object.defineProperty(globalThis, "localStorage", { configurable: true, value: {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: (key) => values.delete(key),
  } });
});
afterEach(() => mock.restoreAll());
after(async () => {
  await modules?.close();
  if (previousStorage) Object.defineProperty(globalThis, "localStorage", previousStorage);
  else delete globalThis.localStorage;
});

describe("Console URL query parsing", () => {
  it("parses empty queries, percent encoding and plus-encoded spaces", () => {
    assert.deepEqual(query.parseQueryString(""), {});
    assert.deepEqual(query.parseQueryString("?common.query=caf%C3%A9+tools&name=a%2Bb"), { common: { query: "café tools" }, name: "a+b" });
  });
  it("builds nested objects from dotted and bracketed keys", () => {
    assert.deepEqual(query.parseQueryString("common.page=2&common[itemsPerPage]=25&common.orderBy.mode=DESC&userRef.uid=uid-1"), {
      common: { page: 2, itemsPerPage: 25, orderBy: { mode: "DESC" } }, userRef: { uid: "uid-1" },
    });
  });
  it("converts ordinary numeric and boolean values while preserving blank values", () => {
    assert.deepEqual(query.parseQueryString("count=0&fraction=-2.5&isActive=true&isDisabled=false&empty=&space=+&text=FALSE"), { count: 0, fraction: -2.5, isActive: true, isDisabled: false, empty: "", space: " ", text: "FALSE" });
  });
  it("supports root array parameters and the last scalar value", () => {
    assert.deepEqual(query.parseQueryString("level[]=ERROR&level[]=FATAL&page=1&page=2"), { level: ["ERROR", "FATAL"], page: 2 });
  });
  it("ignores empty parameter paths", () => {
    assert.deepEqual(query.parseQueryString("=value&[]=ignored&...=ignored&common.page=1"), { common: { page: 1 } });
  });

  for (const attack of ["__proto__.consoleCoveragePolluted=true", "constructor.prototype.consoleCoveragePolluted=true"]) {
    it(`CONSOLE-014: rejects prototype traversal via ${attack.split("=")[0]}`, { todo: "CONSOLE-014" }, () => {
      try {
        query.parseQueryString(attack);
        assert.equal(Object.prototype.consoleCoveragePolluted, undefined);
        assert.equal(({}).consoleCoveragePolluted, undefined);
      } finally {
        delete Object.prototype.consoleCoveragePolluted;
      }
    });
  }
  for (const value of ["common=1&common.page=2", "common=true&common.page=2", "common=text&common.page=2", "level=ERROR&level[]=FATAL"]) {
    it(`CONSOLE-015: handles conflicting query shapes without throwing: ${value}`, { todo: "CONSOLE-015" }, () => {
      assert.doesNotThrow(() => query.parseQueryString(value));
    });
  }
  for (const [key, value] of [["userRef.name", "123"], ["serviceRef.name", "true"], ["common.query", "00123"]]) {
    it(`CONSOLE-016: preserves string-valued ${key}=${value}`, { todo: "CONSOLE-016" }, () => {
      const parsed = query.parseQueryString(`${key}=${value}`);
      const [parent, field] = key.split(".");
      assert.equal(parsed[parent][field], value);
      if (field === "name") assert.doesNotThrow(() => meta.ObjectReference.fromJson(parsed[parent]));
    });
  }
  it("CONSOLE-017: accumulates repeated nested array parameters", { todo: "CONSOLE-017" }, () => {
    assert.deepEqual(query.parseQueryString("filter.level[]=ERROR&filter.level[]=FATAL"), { filter: { level: ["ERROR", "FATAL"] } });
  });
});

describe("Resource route mappings", () => {
  const resources = {
    core: { services: "Service", identityproviders: "IdentityProvider", clusterconfig: "ClusterConfig", policies: "Policy", devices: "Device", namespaces: "Namespace", users: "User", groups: "Group", secrets: "Secret", sessions: "Session", credentials: "Credential", configs: "Config", regions: "Region", gateways: "Gateway", authenticators: "Authenticator" },
    enterprise: { clusterconfig: "ClusterConfig", secrets: "Secret", dnsproviders: "DNSProvider", collectorexporters: "CollectorExporter", certificates: "Certificate", certificateissuers: "CertificateIssuer", directoryproviders: "DirectoryProvider", secretstores: "SecretStore" },
    access: { catalogs: "Catalog", policies: "Policy", requests: "Request", reviews: "Review", secrets: "Secret", integrations: "Integration" },
    cordium: { workspaces: "Workspace", templates: "Template", spaces: "Space", memberships: "Membership", gitproviders: "GitProvider", secrets: "Secret", usersecrets: "UserSecret", regions: "Region" },
  };
  for (const [api, kinds] of Object.entries(resources)) {
    for (const [path, kind] of Object.entries(kinds)) {
      it(`round-trips ${api}/${kind} routes including nested item pages`, () => {
        assert.equal(paths.getResourcePathFromAPIKind({ api, kind }), path);
        assert.deepEqual(paths.getAPIKindFromPath(`/${api}/${path}`), { api, kind });
        assert.deepEqual(paths.getAPIKindFromPath(`/${api}/${path}/example/edit`), { api, kind });
        assert.equal(paths.getAPIFromAPIVersion(`${api}/v1`), api);
      });
    }
  }
  it("returns no resource for unknown APIs, kinds and incomplete paths", () => {
    for (const path of ["", "/", "/core", "/core/unknown", "/unknown/services", "/visibility/accesslogs"]) assert.equal(paths.getAPIKindFromPath(path), undefined);
    assert.equal(paths.getAPIFromAPIVersion("unknown/v1"), undefined);
    assert.equal(paths.getAPIFromAPIVersion(""), undefined);
    assert.equal(paths.getResourcePathFromAPIKind({ api: "core", kind: "Unknown" }), "");
    assert.equal(paths.getResourcePathFromAPIKind({ api: "unknown", kind: "User" }), "");
  });
  it("CONSOLE-018: resolves the Access ClusterConfig route in both directions", { todo: "CONSOLE-018" }, () => {
    assert.equal(paths.getResourcePathFromAPIKind({ api: "access", kind: "ClusterConfig" }), "clusterconfig");
    assert.deepEqual(paths.getAPIKindFromPath("/access/clusterconfig"), { api: "access", kind: "ClusterConfig" });
  });
});

describe("Command palette matching and recents", () => {
  it("normalizes accents, composed Unicode, casing and outer whitespace", () => {
    assert.equal(match.normalize("  CAFÉ A\u0308  "), "cafe a");
    assert.equal(match.scoreText("cafe", "Café"), 100);
  });
  it("ranks exact matches above prefixes, word matches and substrings", () => {
    const scores = ["user", "users", "Core user", "enduser"].map((text) => match.scoreText("user", text));
    assert.ok(scores.every((score) => score > 0));
    assert.ok(scores.every((score, index) => index === 0 || scores[index - 1] > score));
  });
  it("matches initials and restrained word subsequences", () => {
    assert.ok(match.scoreText("ip", "Identity Provider") > 0);
    assert.ok(match.scoreText("authnt", "authenticator") > 0);
    assert.equal(match.scoreText("authnt", "a very unrelated thing"), 0);
  });
  it("requires every meaningful token and ignores single-letter filler", () => {
    assert.ok(match.scoreText("logs access", "Access logs") > 0);
    assert.equal(match.scoreText("access missing", "Access logs"), 0);
    assert.equal(match.scoreText("a user", "user"), match.scoreText("user", "user"));
    assert.equal(match.scoreText(" ", "anything"), 0);
  });
  it("uses title, keywords and subtitle while favoring direct title matches", () => {
    const direct = match.scoreEntry("postgres", { title: "Postgres" });
    assert.ok(direct > match.scoreEntry("postgres", { title: "Service", keywords: ["postgres"] }));
    assert.ok(match.scoreEntry("core user", { title: "User", subtitle: "Core" }) > 0);
    assert.equal(match.scoreEntry("missing", { title: "User", keywords: [] }), 0);
  });
  for (const value of ["why is this failing", "How", "explain permissions", "requests?", "one two three four"]) {
    it(`offers the agent for question ${JSON.stringify(value)}`, () => assert.equal(match.isQuestion(value), true));
  }
  for (const value of ["", "Users", "Core services", "whatever"]) {
    it(`keeps resource search for ${JSON.stringify(value)}`, () => assert.equal(match.isQuestion(value), false));
  }
  for (const [age, weight] of [[0, 4], [3_599_999, 4], [3_600_000, 2], [86_400_000, 1], [604_800_000, 0.5]]) {
    it(`weights recent visits at age ${age}ms`, () => assert.equal(recents.frecency({ visits: 3, lastVisited: 1_000_000_000 - age }, 1_000_000_000), 3 * weight));
  }
  for (const value of ["invalid", "null", "{}", '[null,{},4,{"path":"/core/users","title":"Users"}]']) {
    it(`ignores malformed recent-visit storage: ${value}`, () => {
      localStorage.setItem(storageKey, value);
      assert.deepEqual(recents.readRecents(), []);
    });
  }
  it("updates an existing visit without duplicates and retains other pages", () => {
    recents.recordVisit({ path: "/core/users", title: "Users" }, 100);
    recents.recordVisit({ path: "/core/services", title: "Services" }, 200);
    recents.recordVisit({ path: "/core/users", title: "People", subtitle: "Core" }, 300);
    assert.deepEqual(recents.readRecents(), [
      { path: "/core/users", title: "People", subtitle: "Core", visits: 2, lastVisited: 300 },
      { path: "/core/services", title: "Services", visits: 1, lastVisited: 200 },
    ]);
  });
  it("caps recents at forty and retains a frequently visited older page", () => {
    localStorage.setItem(storageKey, JSON.stringify([{ path: "/favorite", title: "Favorite", visits: 100, lastVisited: 0 }]));
    for (let index = 0; index < 42; index++) recents.recordVisit({ path: `/page-${index}`, title: `Page ${index}` }, 1_000_000_000);
    assert.equal(recents.readRecents().length, 40);
    assert.ok(recents.readRecents().some((item) => item.path === "/favorite"));
  });
  it("survives unavailable storage and quota errors", () => {
    mock.method(localStorage, "getItem", () => { throw new Error("Denied"); });
    assert.deepEqual(recents.readRecents(), []);
    mock.method(localStorage, "setItem", () => { throw new Error("Quota exceeded"); });
    assert.doesNotThrow(() => recents.recordVisit({ path: "/core/users", title: "Users" }));
  });
});

describe("Persisted layouts and settings reducers", () => {
  const layout = { order: ["inventory", "activity"], hidden: ["activity"], collapsed: ["inventory"] };
  it("accepts valid layouts while discarding invalid neighboring entries", () => {
    assert.deepEqual(layouts.parseLayouts(JSON.stringify({ home: layout, invalid: { order: [], hidden: false, collapsed: [] }, nil: null })), { home: layout });
  });
  for (const raw of [null, "", "{", "[]", "null", "true", "42"]) {
    it(`ignores invalid layout storage ${JSON.stringify(raw)}`, () => assert.deepEqual(layouts.parseLayouts(raw), {}));
  }
  for (const ids of [[""], [1], ["x".repeat(65)], Array.from({ length: 65 }, (_, index) => `id-${index}`)]) {
    it(`rejects invalid layout IDs ${JSON.stringify(ids).slice(0, 45)}`, () => {
      assert.deepEqual(layouts.parseLayouts(JSON.stringify({ home: { ...layout, order: ids } })), {});
    });
  }
  it("accepts layout identifier count and length boundaries", () => {
    const value = { order: Array.from({ length: 64 }, (_, index) => `id-${index}`), hidden: ["x".repeat(64)], collapsed: [] };
    assert.deepEqual(layouts.parseLayouts(JSON.stringify({ home: value })), { home: value });
  });
  it("keeps stored layouts separate by domain and user", () => {
    const first = layouts.layoutStorageKey("cluster.example", "alice");
    const second = layouts.layoutStorageKey("cluster.example", "bob");
    layouts.saveLayouts(first, { home: layout });
    assert.deepEqual(layouts.loadLayouts(first), { home: layout });
    assert.deepEqual(layouts.loadLayouts(second), {});
    assert.notEqual(first, layouts.layoutStorageKey("other.example", "alice"));
    layouts.saveLayouts(first, {});
    assert.equal(localStorage.getItem(first), null);
  });
  it("ignores unavailable layout storage during both reads and writes", () => {
    for (const method of ["getItem", "setItem", "removeItem"]) mock.method(localStorage, method, () => { throw new Error("Storage denied"); });
    assert.deepEqual(layouts.loadLayouts("layout"), {});
    assert.doesNotThrow(() => layouts.saveLayouts("layout", { home: layout }));
    assert.doesNotThrow(() => layouts.saveLayouts("layout", {}));
  });
  it("hydrates, updates and resets layouts without modifying previous state", () => {
    const hydrated = layouts.default(undefined, layouts.hydrateLayouts({ key: "alice", layouts: { home: layout } }));
    const updated = layouts.default(hydrated, layouts.setLayout({ page: "core", layout }));
    assert.equal(hydrated.layouts.core, undefined);
    assert.deepEqual(updated.layouts, { home: layout, core: layout });
    const reset = layouts.default(updated, layouts.setLayout({ page: "home" }));
    assert.deepEqual(reset.layouts, { core: layout });
    assert.equal(reset.key, "alice");
    assert.equal(layouts.default(reset, { type: "unknown" }), reset);
  });
  it("updates settings fields independently and supports clearing list filters", () => {
    const initial = settings.default(undefined, { type: "unknown" });
    let next = settings.default(initial, settings.setItemsPerPage({ itemsPerPage: 50 }));
    next = settings.default(next, settings.setOrderByType({ type: meta.CommonListOptions_OrderBy_Type.CREATED_AT }));
    next = settings.default(next, settings.setOrderByMode({ mode: meta.CommonListOptions_OrderBy_Mode.DESC }));
    next = settings.default(next, settings.setPersonalSpaceUID({ uid: "space-1" }));
    next = settings.default(next, settings.setStatus({ status: { domain: "example.test" } }));
    next = settings.default(next, settings.setListOptFilter({ listOptFilter: { isDisabled: true } }));
    assert.equal(initial.itemsPerPage, 10);
    assert.equal(next.itemsPerPage, 50);
    assert.equal(next.personalSpaceUID, "space-1");
    assert.deepEqual(next.status, { domain: "example.test" });
    assert.deepEqual(next.orderBy, { type: meta.CommonListOptions_OrderBy_Type.CREATED_AT, mode: meta.CommonListOptions_OrderBy_Mode.DESC });
    assert.deepEqual(next.listOptFilter, { isDisabled: true });
    assert.equal(settings.default(next, settings.setListOptFilter({})).listOptFilter, undefined);
    assert.equal(next.wideTerminal, true);
  });
  it("reads and writes optional agent preferences even when storage is unavailable", () => {
    assert.equal(environment.readStorage("missing"), undefined);
    environment.writeStorage("model", "example");
    assert.equal(environment.readStorage("model"), "example");
    mock.method(localStorage, "getItem", () => { throw new Error("Denied"); });
    mock.method(localStorage, "setItem", () => { throw new Error("Denied"); });
    assert.equal(environment.readStorage("model"), undefined);
    assert.doesNotThrow(() => environment.writeStorage("model", "new"));
  });
});
