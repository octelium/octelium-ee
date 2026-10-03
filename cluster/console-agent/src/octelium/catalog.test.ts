import { strict as assert } from "node:assert";
import { describe, it } from "node:test";
import { getCatalog } from "../testutil/app.ts";
import {
  classifyRisk,
  describeWellKnown,
  firstSentences,
  shortTypeName,
} from "./catalog.ts";
import { SearchIndex, stem, tokenize } from "./search.ts";

describe("search", () => {
  it("tokenizes camel case and stems plurals", () => {
    assert.deepEqual(tokenize("ListWorkspaceSnapshots of the Policies"), [
      "list",
      "workspace",
      "snapshot",
      "policy",
    ]);
    assert.equal(stem("addresses"), "address");
    assert.equal(stem("access"), "access");
  });

  it("ranks the documents with BM25, synonyms and prefixes", () => {
    const index = new SearchIndex<string>();
    index.add("delete-user", [
      { text: "DeleteUser", weight: 3, primary: true },
    ]);
    index.add("list-user", [{ text: "ListUser", weight: 3, primary: true }]);
    index.add("list-service", [
      { text: "ListService", weight: 3, primary: true },
    ]);
    assert.equal(index.search("remove user")[0].item, "delete-user");
    assert.equal(index.search("all svc")[0].item, "list-service");
    assert.equal(index.search("serv")[0].item, "list-service");
    assert.deepEqual(index.search("the"), []);
  });
});

describe("catalog", () => {
  const catalog = getCatalog();

  it("loads the Octelium APIs", () => {
    assert.ok(catalog.methods.length > 200);
    const packages = new Set(catalog.methods.map((m) => m.packageName));
    for (const pkg of [
      "core.v1",
      "user.v1",
      "enterprise.v1",
      "cordium.v1",
      "visibility.v1",
    ]) {
      assert.ok(packages.has(pkg), pkg);
    }
    assert.ok(!catalog.methods.some((m) => m.packageName === "auth.v1"));
    assert.equal(
      catalog.resolveMethod("user.v1.MainService/Disconnect"),
      undefined,
    );
    assert.equal(
      catalog.resolveMethod("cordium.v1.WorkspaceService/CreateTerminal"),
      undefined,
    );
  });

  it("resolves the method IDs in their different forms", () => {
    for (const id of [
      "core.v1.MainService/ListService",
      "/octelium.api.main.core.v1.MainService/ListService",
      "octelium.api.main.core.v1.MainService.ListService",
      "core.v1.MainService.ListService",
      "CORE.V1.MAINSERVICE/LISTSERVICE",
    ]) {
      assert.equal(
        catalog.resolveMethod(id)?.grpcMethod,
        "/octelium.api.main.core.v1.MainService/ListService",
        id,
      );
    }
    assert.equal(catalog.resolveMethod("core.v1.MainService/Nope"), undefined);
  });

  it("finds the relevant methods", () => {
    const top = (q: string) => catalog.search(q, 3).map((h) => h.item.id);
    assert.equal(top("list services")[0], "core.v1.MainService/ListService");
    assert.equal(top("delete user")[0], "core.v1.MainService/DeleteUser");
    assert.equal(
      top("start workspace")[0],
      "cordium.v1.MainService/StartWorkspace",
    );
    assert.ok(
      top("snapshot a workspace").includes(
        "cordium.v1.MainService/CreateWorkspaceSnapshot",
      ),
    );
    assert.equal(
      top("cluster health")[0],
      "visibility.v1.ClusterService/GetClusterHealth",
    );
    assert.match(
      catalog.formatSearchResults(catalog.search("list users", 2)),
      /^1\. core\.v1\.MainService\/ListUser — .*\[read\]\n   request: core\.v1\.ListUserOptions/,
    );
  });

  it("classifies the risks", () => {
    const risk = (id: string) => catalog.resolveMethod(id)?.risk;
    assert.equal(risk("core.v1.MainService/ListService"), "read");
    assert.equal(risk("core.v1.MainService/GetUser"), "read");
    assert.equal(risk("core.v1.MainService/CreateService"), "write");
    assert.equal(risk("core.v1.MainService/UpdatePolicy"), "write");
    assert.equal(risk("core.v1.MainService/DeleteService"), "destructive");
    assert.equal(
      risk("core.v1.MainService/GenerateCredentialToken"),
      "sensitive",
    );
    assert.equal(risk("cordium.v1.MainService/StopWorkspace"), "write");
    assert.equal(
      classifyRisk(
        "octelium.api.main.visibility.v1.AccessLogService",
        "DeleteAnything",
      ),
      "read",
    );
    assert.equal(
      classifyRisk("octelium.api.main.core.v1.MainService", "Frobnicate"),
      "write",
    );
  });

  it("describes the methods and the types", () => {
    const method = catalog.resolveMethod("core.v1.MainService/ListService")!;
    const text = catalog.describeMethod(method);
    assert.match(text, /^Method: core\.v1\.MainService\/ListService/);
    assert.match(text, /Risk: read/);
    assert.match(text, /core\.v1\.ListServiceOptions \{/);
    assert.match(text, /common: meta\.v1\.CommonListOptions/);
    assert.match(text, /itemsPerPage: uint32/);
    assert.match(text, /Response summary:\n.*\ncore\.v1\.ServiceList \{/);

    const big = catalog.describeMethod(
      catalog.resolveMethod("core.v1.MainService/CreateService")!,
      { maxChars: 4000 },
    );
    assert.match(big, /Not expanded \(use octelium_api_describe/);

    const type = catalog.resolveType("core.v1.Service.Spec.Config.LLM")!;
    const typeText = catalog.describeType(type);
    assert.match(
      typeText,
      /protocol: enum core\.v1\.Service\.Spec\.Config\.LLM\.Protocol \("PROTOCOL_UNSET" \| "OPENAI" \| "ANTHROPIC"/,
    );

    const enumDesc = catalog.resolveType(
      "octelium.api.main.core.v1.Service.Spec.Mode",
    )!;
    assert.match(
      catalog.describeType(enumDesc),
      /^\/\/ .*\nenum core\.v1\.Service\.Spec\.Mode \{\n  "MODE_UNSET"/,
    );
    assert.equal(catalog.resolveType("core.v1.Nope"), undefined);
  });

  it("keeps the doc comments", () => {
    const method = catalog.resolveMethod(
      "cordium.v1.MainService/StartWorkspace",
    )!;
    assert.match(
      method.description,
      /^StartWorkspace starts a stopped Workspace/,
    );
    assert.equal(
      catalog.getComment(method.desc.parent).startsWith("MainService"),
      true,
    );
  });

  it("formats a compact API index", () => {
    const index = catalog.formatIndex();
    assert.match(index, /^core\.v1 — Core Cluster management/);
    assert.match(
      index,
      /MainService: Policy\[Create\|List\|Update\|Delete\|Get\]/,
    );
    assert.match(index, /GenerateCredentialToken/);
    assert.ok(index.length < 12000);
  });

  it("formats the helpers", () => {
    assert.equal(
      shortTypeName("octelium.api.main.core.v1.Service"),
      "core.v1.Service",
    );
    assert.equal(
      shortTypeName("google.protobuf.Struct"),
      "google.protobuf.Struct",
    );
    assert.match(describeWellKnown("google.protobuf.Timestamp")!, /RFC 3339/);
    assert.equal(describeWellKnown("core.v1.Service"), undefined);
    assert.equal(
      firstSentences(
        "First sentence (i.e. with an abbreviation). Second sentence that is long enough to be cut away from the result.",
        60,
      ),
      "First sentence (i.e. with an abbreviation).",
    );
  });
});
