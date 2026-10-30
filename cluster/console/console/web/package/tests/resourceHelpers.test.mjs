import assert from "node:assert/strict";
import { after, afterEach, before, describe, it, mock } from "node:test";
import Yaml from "js-yaml";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Resource serialization, references and API dispatch", () => {
  let modules;
  let pb;
  let utils;
  let clients;
  let picker;
  let core;
  const previousWindow = globalThis.window;
  const previousLocation = globalThis.location;
  const resource = (api = "core", kind = "Service", extra = {}) => pb.newResourcePB(api, kind, {
    apiVersion: `${api}/v1`, kind, metadata: { name: "example.default", uid: "uid-1", displayName: "Example", labels: { team: "platform" } }, ...extra,
  });

  before(async () => {
    globalThis.location = { protocol: "https:" };
    globalThis.window = { location: { host: "console.example.test" } };
    modules = await createConsoleModules();
    [pb, utils, clients, picker, core] = await Promise.all([
      modules.load("utils/pb/index.ts"), modules.load("utils/index.ts"),
      modules.load("utils/client/index.ts"), modules.load("components/ResourceLayout/listResourcesForSelect.ts"),
      modules.load("apis/corev1/corev1.ts"),
    ]);
  });
  afterEach(() => mock.restoreAll());
  after(async () => {
    utils?.queryClient.clear();
    await modules?.close();
    if (previousWindow === undefined) delete globalThis.window;
    else globalThis.window = previousWindow;
    if (previousLocation === undefined) delete globalThis.location;
    else globalThis.location = previousLocation;
  });

  for (const [api, kind] of [["core", "User"], ["enterprise", "CollectorExporter"], ["access", "Catalog"], ["cordium", "Workspace"]]) {
    it(`round-trips ${api}/${kind} through JSON and YAML and clones nested metadata`, () => {
      const value = resource(api, kind);
      assert.deepEqual(pb.resourceFromJSON(pb.resourceToJSON(value)), value);
      assert.deepEqual(pb.resourceFromYAML(pb.resourceToYAML(value)), value);
      assert.deepEqual(pb.resourceToJSONObject(value), JSON.parse(pb.resourceToJSON(value)));
      const clone = pb.cloneResource(value);
      assert.deepEqual(clone, value);
      clone.metadata.labels.team = "changed";
      assert.equal(value.metadata.labels.team, "platform");
      assert.equal(pb.getAPI(value), api);
      assert.equal(pb.getPB(value), pb.getPBFromAPI(api));
    });
    it(`keeps ${api}/${kind} fingerprints stable until an actual edit`, () => {
      const value = resource(api, kind);
      const clone = pb.cloneResource(value);
      assert.equal(pb.resourceFingerprint(value), pb.resourceFingerprint(clone));
      clone.metadata.displayName = "Changed";
      assert.notEqual(pb.resourceFingerprint(value), pb.resourceFingerprint(clone));
    });
  }

  for (const input of ["{", "null", "[]", "42", "{}", '{"kind":"Service"}', '{"apiVersion":"unknown/v1","kind":"Service"}', '{"apiVersion":"core/v1","kind":"Missing"}']) {
    it(`rejects unsupported resource JSON ${input}`, () => {
      mock.method(console, "error", () => {});
      assert.equal(pb.resourceFromJSON(input), undefined);
    });
  }
  it("rejects malformed YAML without throwing to the editor", () => {
    mock.method(console, "error", () => {});
    assert.equal(pb.resourceFromYAML("metadata: [unterminated"), undefined);
    assert.equal(pb.resourceFromYAML(""), undefined);
    assert.equal(pb.resourceFromYAML("null"), undefined);
  });
  it("serializes metadata, spec and status sections independently", () => {
    const value = resource("core", "Service", { spec: { isPublic: true }, status: { namespaceRef: { name: "default" } } });
    const json = pb.resourceToJSONObject(value);
    for (const [jsonName, yamlName, section] of [["resourceMetadataToJSON", "resourceMetadataToYAML", "metadata"], ["resourceSpecToJSON", "resourceSpecToYAML", "spec"], ["resourceStatusToJSON", "resourceStatusToYAML", "status"]]) {
      assert.deepEqual(JSON.parse(pb[jsonName](value)), json[section]);
      assert.deepEqual(Yaml.load(pb[yamlName](value)), json[section]);
    }
  });
  it("returns safe serialization fallbacks for unsupported API versions", () => {
    mock.method(console, "error", () => {});
    const value = { apiVersion: "unknown/v1", kind: "Service" };
    assert.equal(pb.resourceToJSON(value), "");
    assert.equal(pb.resourceToYAML(value), "");
    assert.equal(pb.resourceFingerprint(value), "");
  });
  it("creates complete resource references and short names", () => {
    const value = resource();
    const ref = pb.getResourceRef(value);
    assert.equal(ref.apiVersion, "core/v1");
    assert.equal(ref.kind, "Service");
    assert.equal(ref.name, "example.default");
    assert.equal(ref.uid, "uid-1");
    assert.equal(pb.getShortName(value), "example");
    assert.equal(pb.getShortNameFromRef(ref), "example");
    assert.equal(pb.getShortNameFromStr(""), "");
  });
  it("creates matching detail paths, list paths and invalidation keys", () => {
    const value = resource();
    const list = { apiVersion: "core/v1", kind: "ServiceList" };
    assert.equal(pb.getResourcePath(value), "/core/services/example.default");
    assert.equal(pb.getResourcePathKind(value), "services");
    assert.equal(pb.getResourceListPath(list), "/core/services");
    assert.equal(pb.getResourceListPathFromResource(value), "/core/services");
    assert.equal(pb.getKindFromResourceList(list), "Service");
    assert.equal(pb.getAPIFromResourceList(list), "core");
    assert.equal(pb.getListKey(value), "core.listService");
    assert.equal(pb.getListKeyFromResource(value), "core.listService");
    assert.equal(pb.getListKeyFromPath("/core/services/example.default"), "core.listService");
    assert.equal(pb.getGetKey(value), "core.getService");
    assert.equal(pb.getGetKeyFromRef(pb.getResourceRef(value)), "core.getService");
    assert.equal(pb.getGetKeyFromPath("/core/services/example.default"), "core.getService");
  });
  it("invalidates resource details by both UID and name plus their summaries", () => {
    const invalidate = mock.method(utils.queryClient, "invalidateQueries", () => Promise.resolve());
    pb.invalidateResource(resource());
    assert.deepEqual(invalidate.mock.calls.map((call) => call.arguments[0].queryKey), [
      ["core.getService", "uid-1"], ["core.getService", "example.default"],
      ["visibility", "core", "summary", "Service"], ["visibility", "cluster", "summary"],
    ]);
  });
  it("invalidates resource lists, picker options and summaries", () => {
    const invalidate = mock.method(utils.queryClient, "invalidateQueries", () => Promise.resolve());
    pb.invalidateResourceList(resource());
    pb.invalidateResourceListFromList({ apiVersion: "access/v1", kind: "RequestList" });
    pb.invalidateKey(["custom", "query"]);
    assert.deepEqual(invalidate.mock.calls.map((call) => call.arguments[0].queryKey), [
      ["core.listService"], ["listSelectComponent", "core", "Service"],
      ["visibility", "core", "summary", "Service"], ["visibility", "cluster", "summary"],
      ["access.listRequest"], ["custom", "query"],
    ]);
  });

  for (const [api, kind] of [["core", "Service"], ["enterprise", "CollectorExporter"], ["access", "Catalog"]]) {
    it(`dispatches ${api}/${kind} create, get, update and delete through the resource client`, () => {
      const client = pb.getClient(api);
      const value = resource(api, kind);
      const result = { response: value };
      const create = mock.method(client, `create${kind}`, () => result);
      const get = mock.method(client, `get${kind}`, () => result);
      const update = mock.method(client, `update${kind}`, () => result);
      const remove = mock.method(client, `delete${kind}`, () => result);
      assert.equal(pb.createResourcePB(api, kind, value), result);
      assert.equal(pb.getResourcePB(api, kind, { uid: "uid-1" }), result);
      assert.equal(pb.updateResourcePB(value), result);
      assert.equal(pb.deleteResourcePB(value), result);
      assert.deepEqual(create.mock.calls[0].arguments, [value]);
      assert.deepEqual(get.mock.calls[0].arguments, [{ uid: "uid-1" }]);
      assert.deepEqual(update.mock.calls[0].arguments, [value]);
      assert.deepEqual(remove.mock.calls[0].arguments, [{ uid: "uid-1" }]);
      assert.equal(pb.getResourceClient(value), client);
      assert.equal(pb.getPathClient(pb.getResourcePath(value)), client);
    });
  }
  for (const [api, kind] of [["core", "User"], ["enterprise", "Certificate"], ["access", "Request"], ["cordium", "Workspace"]]) {
    it(`lists ${api}/${kind} through the visibility API with independent cancellation`, async () => {
      const client = pb.getClientResourceList(api);
      const req = pb.getListOptionsPB(api, kind).create();
      const result = { response: { items: [] } };
      const list = mock.method(client, `list${kind}`, () => result);
      const signal = new AbortController().signal;
      assert.equal(pb.listResourcesPB(api, kind, req, { abort: signal }), result);
      assert.deepEqual(list.mock.calls[0].arguments, [req, { abort: signal }]);
      assert.equal(pb.getPBResourceListFromAPI(api), pb.getClientResourceListP(api));
    });
  }
  it("looks up Cordium resources by exact name in the list response", async () => {
    const value = resource("cordium", "Workspace");
    const list = mock.method(clients.getClientVisibilityCordium(), "listWorkspace", () => ({ response: { items: [resource("cordium", "Workspace", { metadata: { name: "similar" } }), value] } }));
    assert.deepEqual(await pb.getResourcePB("cordium", "Workspace", { name: value.metadata.name }), { response: value });
    assert.equal(list.mock.calls[0].arguments[0].common.query, value.metadata.name);
    assert.equal(list.mock.calls[0].arguments[0].common.itemsPerPage, 25);
  });
  it("reports NOT_FOUND for a Cordium resource absent from the list", async () => {
    mock.method(clients.getClientVisibilityCordium(), "listWorkspace", () => ({ response: { items: [] } }));
    await assert.rejects(pb.getResourcePB("cordium", "Workspace", { name: "missing" }), { code: "NOT_FOUND", message: "Workspace missing does not exist" });
  });
  it("loads picker options with trimmed search text, bounded page size and cancellation", async () => {
    const items = [resource()];
    const list = mock.method(clients.getClientVisibilityCore(), "listService", () => ({ response: { items } }));
    const signal = new AbortController().signal;
    assert.equal(await picker.listResourcesForSelect("core", "Service", "  web  ", signal), items);
    const [req, options] = list.mock.calls[0].arguments;
    assert.equal(req.common.query, "web");
    assert.equal(req.common.page, 0);
    assert.equal(req.common.itemsPerPage, 100);
    assert.equal(options.abort, signal);
  });
  it("reports unsupported picker APIs and absent responses", async () => {
    await assert.rejects(picker.listResourcesForSelect("core", "Missing"), /list API is not available/);
    mock.method(clients.getClientVisibilityCore(), "listService", () => undefined);
    await assert.rejects(picker.listResourcesForSelect("core", "Service"), /returned no response/);
  });
  it("does not invent clients for unsupported resource routes", () => {
    assert.equal(pb.getResourceClient({ apiVersion: "unknown/v1", kind: "Service" }), undefined);
    assert.equal(pb.getPathClient("/missing"), undefined);
    for (const name of ["getClient", "getClientResourceList", "getClientResourceListP", "getPBFromAPI", "getPBResourceListFromAPI"]) assert.equal(pb[name]("unknown"), undefined);
  });

  for (const [kind, access, authentication, audit] of [
    ["User", true, true, true], ["Session", true, true, true], ["Device", true, false, true],
    ["Service", true, false, false], ["Namespace", true, false, false], ["Policy", true, false, false],
    ["Region", true, false, false], ["Credential", false, true, false], ["Authenticator", false, true, false],
    ["IdentityProvider", false, true, false], ["Secret", false, false, false],
  ]) {
    it(`offers only supported core/${kind} log tabs`, () => {
      const value = { apiVersion: "core/v1", kind };
      assert.equal(pb.hasAccessLog(value), access);
      assert.equal(pb.hasAuthenticationLog(value), authentication);
      assert.equal(pb.hasAuditLog(value), audit);
      assert.equal(pb.hasAccessLog({ ...value, apiVersion: "access/v1" }), false);
      assert.equal(pb.hasAuthenticationLog({ ...value, apiVersion: "access/v1" }), false);
      assert.equal(pb.hasAuditLog({ ...value, apiVersion: "access/v1" }), false);
    });
  }
  it("exposes Service visibility and recording tabs only for their modes", () => {
    for (const mode of [core.Service_Spec_Mode.LLM, core.Service_Spec_Mode.SSH, core.Service_Spec_Mode.HTTP]) {
      const value = resource("core", "Service", { spec: { mode } });
      assert.equal(pb.hasLLMVisibility(value), mode === core.Service_Spec_Mode.LLM);
      assert.equal(pb.hasSSHSessionLog(value), mode === core.Service_Spec_Mode.SSH);
    }
    assert.equal(pb.hasLLMVisibility(resource("enterprise", "Secret")), false);
    assert.equal(pb.hasSSHSessionLog(resource("access", "Catalog")), false);
  });
  it("formats identity labels using display names and email fallbacks", () => {
    assert.equal(pb.printResourceNameWithDisplay({ metadata: { name: "alice", displayName: "Alice" } }), "alice (Alice)");
    assert.equal(pb.printResourceNameWithDisplay({}), "");
    assert.equal(pb.printUserWithEmail({ spec: { email: "alice@example.test" }, metadata: { name: "alice", displayName: "Alice" } }), "Alice (alice@example.test)");
    assert.equal(pb.printUserWithEmail({ spec: { email: "alice@example.test" } }), "alice@example.test");
    assert.equal(pb.printUserWithEmail({ metadata: { name: "alice" } }), "alice");
    assert.equal(pb.printUserWithEmail({}), "");
  });

  it("builds resource reference filters and falls back to change-history filters", () => {
    assert.equal(pb.getRefName({ kind: "Service" }), "serviceRef");
    assert.deepEqual(pb.getRefNameQueryArg(resource()), { key: "serviceRef", name: "example.default" });
    assert.equal(pb.getRefNameQueryArgStr(resource()), "serviceRef.name=example.default");
    assert.equal(pb.getAuditLogQueryArgStr(resource()), "resourceRef.name=example.default");
    assert.equal(pb.getAuditLogQueryArgStr(resource("core", "User")), "userRef.name=example.default");
  });
});
