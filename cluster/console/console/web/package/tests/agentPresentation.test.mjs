import assert from "node:assert/strict";
import { after, before, describe, it } from "node:test";
import { File, FileArchive, FileCode, FileImage, FileSpreadsheet, FileText } from "lucide-react";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Agent exports, formatting and console context", () => {
  let modules;
  let utils;
  let context;
  let files;
  let cordium;
  const ref = (extra = {}) => ({ apiVersion: "core/v1", kind: "Service", name: "web.default", ...extra });
  const message = (blocks, extra = {}) => ({ id: "m-1", role: "assistant", status: "completed", blocks, ...extra });

  before(async () => {
    modules = await createConsoleModules();
    [utils, context, files, cordium] = await Promise.all([
      modules.load("pages/Agent/utils.ts"), modules.load("components/AgentDock/pageContext.ts"),
      modules.load("pages/Agent/files.ts"), modules.load("apis/cordiumv1/cordiumv1.ts"),
    ]);
  });
  after(async () => modules?.close());

  for (const [value, expected] of [[-1, ""], [NaN, ""], [Infinity, ""], [0, "0ms"], [999, "999ms"], [1000, "1.0s"], [9999, "10.0s"], [10000, "10s"], [59999, "59s"], [60000, "1m 0s"], [125000, "2m 5s"]]) {
    it(`formats request duration ${value}`, () => assert.equal(utils.formatDuration(value), expected));
  }
  it("formats elapsed time, bytes and compact counts at display boundaries", () => {
    for (const [value, expected] of [[-1, "0s"], [0, "0s"], [59999, "59s"], [60000, "1m 00s"], [125000, "2m 05s"]]) assert.equal(utils.formatElapsed(value), expected);
    for (const [value, expected] of [[0, "0 B"], [1023, "1023 B"], [1024, "1.0 KB"], [1048576, "1.0 MB"], [1073741824, "1.0 GB"]]) assert.equal(utils.formatBytes(value), expected);
    for (const [value, expected] of [[999, "999"], [1000, "1.0k"], [10000, "10k"], [1000000, "1.0M"]]) assert.equal(utils.formatCount(value), expected);
  });
  for (const [mime, name, icon] of [
    ["image/png", "data.csv", FileImage], ["text/csv", "data", FileSpreadsheet],
    ["application/octet-stream", "report.XLSX", FileSpreadsheet], ["application/gzip", "archive", FileArchive],
    ["application/json", "data", FileCode], ["application/octet-stream", "script.go", FileCode],
    ["text/plain", "notes.txt", FileText], ["application/pdf", "report.pdf", FileText],
    ["application/octet-stream", "binary", File],
  ]) {
    it(`chooses the attachment icon for ${mime} ${name}`, () => assert.equal(files.fileIcon(mime, name), icon));
  }
  it("creates bounded, filesystem-safe export names", () => {
    assert.equal(files.toFileName("  My / conversation? ", "md"), "My_conversation_.md");
    assert.equal(files.toFileName("", "json"), "conversation.json");
    assert.equal(files.toFileName("x".repeat(100), "csv"), `${"x".repeat(80)}.csv`);
  });

  it("formats primitive, absent and structured export values", () => {
    assert.equal(utils.formatValue(undefined), "");
    assert.equal(utils.formatValue(null), "");
    assert.equal(utils.formatValue(false), "false");
    assert.equal(utils.formatValue(0), "0");
    assert.equal(utils.formatValue("text"), "text");
    assert.equal(utils.formatValue({ id: 1 }), '{"id":1}');
    assert.equal(utils.toJSON(undefined), "");
    assert.equal(utils.toJSON("already text"), "already text");
    assert.equal(utils.toJSON({ id: 1 }), '{\n  "id": 1\n}');
  });
  it("escapes CSV delimiters, quotes and multiline cells with explicit column order", () => {
    assert.equal(utils.toCSV([{ key: "name", label: "Full name" }, { key: "data" }, { key: "missing" }], [
      { name: 'Alice, "A"', data: "line1\nline2", ignored: "private" },
      { name: "Bob", data: false },
    ]), 'Full name,data,missing\n"Alice, ""A""","line1\nline2",\nBob,false,');
    assert.equal(utils.toCSV([{ key: "name" }], []), "name");
  });
  it("exports the visible conversation while omitting private thinking", () => {
    const output = utils.conversationMarkdown("Example", [
      message([{ type: "markdown", text: " Question " }], { role: "user", attachments: [{ name: "notes.txt" }] }),
      message([
        { type: "thinking", text: "private chain of thought" },
        { type: "markdown", text: " Answer " },
        { type: "tool", name: "search", title: "Find records", status: "completed" },
        { type: "approval", title: "Delete record", status: "rejected" },
        { type: "table", title: "Results", columns: [{ key: "value", label: "Value | detail" }], rows: [{ value: "line1\nline2" }] },
        { type: "chart", chart: { chartType: "line", title: "Trend" } },
        { type: "resources", resources: [{ ref: ref() }] },
        { type: "artifact", artifact: { name: "report.csv" } },
        { type: "notice", text: "Read carefully" }, { type: "error", message: "Failed request" },
      ]),
    ]);
    assert.equal(output, "# Example\n\n## You\n\nAttachments: notes.txt\n\nQuestion\n\n## Agent\n\nAnswer\n\n- Find records (completed)\n\n> Approval rejected: Delete record\n\n**Results**\n\n| Value \\| detail |\n| --- |\n| line1 line2 |\n\n_Chart: Trend_\n\n- Service web.default\n\n_Artifact: report.csv_\n\n> Read carefully\n\n> Error: Failed request\n");
    assert.doesNotMatch(output, /private chain/);
    assert.equal(utils.messageText(message([{ type: "thinking", text: "private" }, { type: "markdown", text: " First " }, { type: "markdown", text: " " }, { type: "markdown", text: "Second" }])), "First\n\nSecond");
  });
  it("normalizes and deduplicates conversation search terms", () => {
    assert.deepEqual(utils.searchTerms("  USER user \n logs  "), ["user", "logs"]);
    assert.equal(utils.matchesTerms("User access LOGS", ["user", "logs"]), true);
    assert.equal(utils.matchesTerms("User access", ["user", "logs"]), false);
    assert.equal(utils.matchesTerms("Anything", []), true);
  });
  it("groups sorted conversations into calendar buckets", () => {
    const now = new Date("2026-10-30T12:00:00Z");
    const conversations = [
      { id: "today-1", updatedAt: "2026-10-30T10:00:00Z" }, { id: "today-2", updatedAt: "2026-10-30T09:00:00Z" },
      { id: "yesterday", updatedAt: "2026-10-29T09:00:00Z" }, { id: "week", updatedAt: "2026-10-25T09:00:00Z" },
      { id: "month", updatedAt: "2026-10-05T09:00:00Z" },
    ];
    const groups = utils.groupConversations(conversations, now);
    assert.deepEqual(groups.map((group) => [group.label, group.items.map((item) => item.id)]), [
      ["Today", ["today-1", "today-2"]], ["Yesterday", ["yesterday"]], ["Previous 7 days", ["week"]], ["Previous 30 days", ["month"]],
    ]);
    assert.deepEqual(utils.groupConversations([], now), []);
  });
  it("computes tool durations only when both timestamps exist", () => {
    assert.equal(utils.toolDuration({ startedAt: "2026-10-30T12:00:00Z", completedAt: "2026-10-30T12:00:01.250Z" }), 1250);
    assert.equal(utils.toolDuration({ startedAt: "2026-10-30T12:00:00Z" }), undefined);
  });
  for (const [hour, expected] of [[0, "Good evening"], [4, "Good evening"], [5, "Good morning"], [11, "Good morning"], [12, "Good afternoon"], [17, "Good afternoon"], [18, "Good evening"]]) {
    it(`chooses the greeting at hour ${hour}`, () => assert.equal(utils.greeting(new Date(2026, 9, 30, hour)), expected));
  }
  it("builds safe routes for supported resource APIs and rejects unknown references", () => {
    assert.equal(utils.getResourceRoute(ref({ name: "a/b?c" })), "/core/services/a%2Fb%3Fc");
    assert.equal(utils.getResourceURIRoute("octelium://resource/access/v1/Catalog/tools"), "/access/catalogs/tools");
    assert.equal(utils.getResourceRoute(ref({ apiVersion: "unknown/v1" })), undefined);
    assert.equal(utils.getResourceRoute(ref({ kind: "Missing" })), undefined);
    assert.equal(utils.getResourceRoute(ref({ name: undefined, uid: "uid-1" })), undefined);
    assert.equal(utils.getResourceURIRoute("malformed"), undefined);
    assert.equal(utils.urlTransform("javascript:alert(1)"), "");
    assert.equal(utils.urlTransform("octelium://resource/core/v1/Service/web"), "octelium://resource/core/v1/Service/web");
  });
  it("maps workspace transition states and display-name fallbacks", () => {
    const state = cordium.Workspace_Status_State;
    for (const name of ["INIT_REQUEST", "INITIALIZING", "PULLING_IMAGE", "BUILDING_IMAGE", "STARTING_RUNTIME", "PREPARING"]) assert.equal(utils.isWorkspaceStarting({ status: { state: state[name] } }), true);
    for (const name of ["STOPPING_REQUEST", "STOPPING"]) assert.equal(utils.isWorkspaceStopping({ status: { state: state[name] } }), true);
    assert.equal(utils.isWorkspaceStarting({ status: { state: state.RUNNING } }), false);
    assert.equal(utils.isWorkspaceStopping(), false);
    assert.equal(utils.workspaceStateLabel(state.RUNNING), "Running");
    assert.equal(utils.workspaceStateLabel(state.STOPPED), "Stopped");
    assert.equal(utils.workspaceStateLabel(-1), "Unknown");
    assert.equal(utils.getWorkspaceDisplayName(), "Workspace");
    assert.equal(utils.getWorkspaceDisplayName({ metadata: { name: "workspace", displayName: "My workspace" } }), "My workspace");
  });
  it("maps API operation risk to its presentation color", () => {
    for (const [risk, color] of [["read", "green"], ["write", "blue"], ["destructive", "red"], ["sensitive", "orange"]]) assert.equal(utils.riskColor(risk), color);
  });

  it("builds breadcrumb titles and resource references without the console root", () => {
    const trail = [{ label: "Console" }, { label: "Core" }, { label: "Services" }, { label: "Web", resource: { api: "core", kind: "Service", name: "web.default" } }];
    assert.equal(context.trailTitle(trail), "Core › Services › Web");
    assert.equal(context.trailTitle([{ label: "Console" }]), "Console");
    assert.equal(context.trailTitle([]), "");
    assert.deepEqual(context.routeResources(trail), [ref()]);
    assert.deepEqual(context.buildPageContext({ pathname: "/core/services", search: "?common.page=2", trail, details: { Filter: "Web" } }), { path: "/core/services?common.page=2", title: "Core › Services › Web", details: { Filter: "Web" } });
  });
  it("bounds attention details while retaining unrelated page details", () => {
    const details = { "Time range": "1h" };
    assert.equal(context.withConditions(details, []), details);
    assert.deepEqual(context.withConditions(details, [{ count: 3, label: "Denied requests" }]), { ...details, "Needs attention": "3 × Denied requests" });
    const value = context.withConditions(details, [{ count: 3, label: "x".repeat(300) }])["Needs attention"];
    assert.equal(value.length, 250);
    assert.ok(value.endsWith("…"));
  });
  it("deduplicates pinned and routed resources while preserving distinct kinds", () => {
    const page = { path: "/core/services/web.default", title: "Web" };
    const pinned = { resources: [ref(), ref({ kind: "Namespace", name: "default" })], items: [{ kind: "DashboardSection", title: "Inventory", data: { count: 3 } }] };
    assert.deepEqual(context.buildRunContext({ page, routeResources: [ref()], pinned }), { page, resources: [ref(), ref({ kind: "Namespace", name: "default" })], items: pinned.items });
    assert.equal(context.buildRunContext({ routeResources: [ref()] }), undefined);
    assert.deepEqual(context.buildRunContext({ routeResources: [], pinned: { resources: [ref()] } }), { resources: [ref()] });
  });
  it("describes pinned resources and serializes page context for the agent", () => {
    const value = { apiVersion: "core/v1", kind: "Policy", metadata: { name: "allow", uid: "uid-1" } };
    const pinned = context.pinnedResource(value);
    assert.equal(pinned.label, "Policy allow");
    assert.equal(context.pinnedResources([value]).label, "Policy allow");
    assert.equal(context.pinnedResources([value, value]).label, "2 Policies");
    assert.equal(context.pinnedResources([]).label, "0 resources");
    assert.equal(context.contextAsText({ page: { title: "Policies", path: "/core/policies", details: { Filter: "Enabled" } }, resources: pinned.resources, items: [{ kind: "Inventory", title: "Counts", data: { total: 2 } }] }), '[Console context]\nPage: Policies (/core/policies)\nFilter: Enabled\nPolicy: allow\nInventory (Counts): {"total":2}');
  });

  it("CONSOLE-024: links Cordium resources returned by the agent", { todo: "CONSOLE-024" }, () => {
    assert.equal(utils.getResourceRoute({ apiVersion: "cordium/v1", kind: "Workspace", name: "workspace-1" }), "/cordium/workspaces/workspace-1");
  });
  it("CONSOLE-025: neutralizes formula text in CSV exports", { todo: "CONSOLE-025" }, () => {
    const cell = utils.toCSV([{ key: "value" }], [{ value: "=1+1" }]).split("\n")[1];
    assert.ok(cell?.includes("=1+1"));
    assert.ok(cell.startsWith("'") || cell.startsWith("\"'") || cell.startsWith("\"\t"));
  });
  it("CONSOLE-026: groups conversations by local calendar days across spring DST", { todo: "CONSOLE-026" }, () => {
    const previousTZ = process.env.TZ;
    try {
      process.env.TZ = "America/New_York";
      assert.equal(utils.conversationGroupLabel(new Date("2026-03-07T23:30:00-05:00"), new Date("2026-03-09T12:00:00-04:00")), "Previous 7 days");
    } finally {
      if (previousTZ === undefined) delete process.env.TZ;
      else process.env.TZ = previousTZ;
    }
  });
});
