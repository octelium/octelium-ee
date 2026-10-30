import assert from "node:assert/strict";
import { after, afterEach, before, describe, it, mock } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Visibility ranges, metrics and setup events", () => {
  let modules;
  let visibility;
  let llm;
  let llmPB;
  let dashboard;
  let queries;
  let visibilityPB;
  let timestamp;
  let meta;
  let failure;
  let setup;

  before(async () => {
    modules = await createConsoleModules();
    [visibility, llm, llmPB, dashboard, queries, visibilityPB, timestamp, meta, failure, setup] = await Promise.all([
      modules.load("utils/visibility/index.ts"), modules.load("components/LLMVisibility/utils.ts"),
      modules.load("apis/visibilityv1/llm/vllmv1.ts"), modules.load("components/Dashboard/utils.ts"),
      modules.load("components/Dashboard/queries.ts"), modules.load("apis/visibilityv1/visibilityv1.ts"),
      modules.load("apis/google/protobuf/timestamp.ts"), modules.load("apis/metav1/metav1.ts"),
      modules.load("components/AuthenticationLogViewer/failure.ts"), modules.load("components/ServiceConnect/setup.ts"),
    ]);
  });
  afterEach(() => mock.restoreAll());
  after(async () => modules?.close());

  for (const [unit, field] of [["millisecond", "milliseconds"], ["second", "seconds"], ["minute", "minutes"], ["hour", "hours"], ["day", "days"], ["week", "weeks"], ["month", "months"], ["unknown", "seconds"]]) {
    it(`creates a duration in ${unit}`, () => assert.deepEqual(visibility.createDuration(3, unit), { type: { oneofKind: field, [field]: 3 } }));
  }
  for (const [minutes, unit, value] of [[5, "seconds", 30], [15, "seconds", 30], [16, "minutes", 1], [60, "minutes", 1], [61, "minutes", 5], [180, "minutes", 5], [181, "minutes", 10], [360, "minutes", 10], [361, "minutes", 15], [720, "minutes", 15], [721, "minutes", 30], [1440, "minutes", 30], [1441, "hours", 1], [4320, "hours", 1], [4321, "hours", 3], [10080, "hours", 3], [10081, "hours", 6]]) {
    it(`selects an automatic interval at the ${minutes}-minute boundary`, () => assert.deepEqual(visibility.getAutoInterval(minutes), { type: { oneofKind: unit, [unit]: value } }));
  }
  it("recognizes every shared period and rejects unknown values", () => {
    assert.equal(new Set(visibility.ALL_PERIODS.map((period) => period.minutes)).size, visibility.ALL_PERIODS.length);
    for (const { minutes, label } of visibility.ALL_PERIODS) {
      assert.equal(visibility.isKnownPeriod(minutes), true);
      assert.equal(visibility.periodLabel(minutes), label);
    }
    for (const value of [0, -1, NaN, Infinity, 61]) {
      assert.equal(visibility.isKnownPeriod(value), false);
      assert.equal(visibility.periodLabel(value), "");
    }
  });
  it("builds equal adjacent comparison windows ending at the current time", () => {
    const before = Date.now();
    const range = visibility.buildTimestamps(60);
    const after = Date.now();
    assert.ok(range.curTo >= before && range.curTo <= after);
    assert.equal(range.curTo - range.curFrom, 3_600_000);
    assert.equal(range.prevTo - range.prevFrom, 3_600_000);
    assert.equal(range.prevTo, range.curFrom);
  });
  it("converts millisecond timestamps and dashboard range boundaries", () => {
    for (const ms of [0, 1, Date.parse("2026-10-30T12:00:00.123Z")]) assert.equal(timestamp.Timestamp.toDate(visibility.toTs(ms)).getTime(), ms);
    const range = dashboard.rangeOptions(1000, 2000);
    assert.equal(timestamp.Timestamp.toDate(range.common.from).getTime(), 1000);
    assert.equal(timestamp.Timestamp.toDate(range.common.to).getTime(), 2000);
  });
  it("handles zero totals and computes rounded percentages and change", () => {
    assert.equal(visibility.pct(1, 3), 33);
    assert.equal(visibility.pct(0, 0), 0);
    assert.equal(visibility.deltaPct(0, 5), -100);
    assert.equal(visibility.deltaPct(3, 2), 50);
    assert.equal(visibility.deltaPct(5, 0), null);
    assert.equal(dashboard.ratio(1, 3), 33.3);
    assert.equal(dashboard.ratio(1, 0), 0);
    assert.equal(visibility.n(undefined), 0);
    assert.equal(visibility.n("12"), 12);
  });
  it("distinguishes scopes, periods and statuses in visibility query keys", () => {
    const first = visibility.visibilityKeys.accessSummary("current", 60, "allowed", { userRef: "alice" });
    const alternatives = [
      visibility.visibilityKeys.accessSummary("previous", 60, "allowed", { userRef: "alice" }),
      visibility.visibilityKeys.accessSummary("current", 30, "allowed", { userRef: "alice" }),
      visibility.visibilityKeys.accessSummary("current", 60, "denied", { userRef: "alice" }),
      visibility.visibilityKeys.accessSummary("current", 60, "allowed", { userRef: "bob" }),
      visibility.visibilityKeys.accessDataPoint(60, "allowed", { userRef: "alice" }),
      visibility.visibilityKeys.authSummary("current", 60, { userRef: "alice" }),
      visibility.visibilityKeys.auditSummary("current", 60, { userRef: "alice" }),
    ];
    for (const alternative of alternatives) assert.notDeepEqual(first, alternative);
    assert.deepEqual(visibility.visibilityKeys.authDataPoint(60, {}), ["authLogDataPoint", 60, {}]);
    assert.deepEqual(visibility.visibilityKeys.auditDataPoint(60, {}), ["auditLogDataPoint", 60, {}]);
    assert.deepEqual(visibility.visibilityKeys.componentSummary("current", 60), ["componentLogSummary", "current", 60]);
    assert.deepEqual(visibility.visibilityKeys.componentDataPoint(60), ["componentLogDataPoint", 60]);
    assert.deepEqual(visibility.visibilityKeys.componentErrorDataPoint(60), ["componentLogDataPointErrors", 60]);
  });
  it("targets dashboard refreshes only at active dashboard observers", () => {
    const predicate = dashboard.dashboardQueryFilters.predicate;
    assert.equal(predicate({ meta: { dashboard: true }, getObserversCount: () => 1 }), true);
    assert.equal(predicate({ meta: { dashboard: true }, getObserversCount: () => 0 }), false);
    assert.equal(predicate({ meta: {}, getObserversCount: () => 1 }), false);
    assert.equal(predicate({ getObserversCount: () => 1 }), false);
  });
  it("derives summary availability without hiding available neighboring kinds", () => {
    const kind = visibilityPB.GetClusterSummaryRequest_Kind;
    const data = visibilityPB.GetClusterSummaryResponse.create({ unavailables: [{ kind: kind.CORE_USER }] });
    assert.equal(queries.isKindAvailable(data, kind.CORE_USER), false);
    assert.equal(queries.isKindAvailable(data, kind.CORE_SERVICE), true);
    assert.deepEqual(queries.summaryTileState({ data, isLoading: false, isError: false }, kind.CORE_USER), { isLoading: false, isError: true, hasData: false });
    assert.deepEqual(queries.summaryTileState({ data, isLoading: false, isError: false }, kind.CORE_SERVICE), { isLoading: false, isError: false, hasData: true });
    assert.deepEqual(queries.summaryTileState({ isLoading: true, isError: false }, kind.CORE_SERVICE), { isLoading: true, isError: false, hasData: false });
  });
  it("maps camel-case resource kinds and acronym kinds to summary enums", () => {
    for (const [api, kind, expected] of [["core", "IdentityProvider", "CORE_IDENTITY_PROVIDER"], ["enterprise", "DNSProvider", "ENTERPRISE_DNSPROVIDER"], ["access", "IntegrationBinding", "ACCESS_INTEGRATION_BINDING"], ["cordium", "UserSecret", "CORDIUM_USER_SECRET"]]) assert.equal(queries.summaryKindOf(api, kind), visibilityPB.GetClusterSummaryRequest_Kind[expected]);
    assert.equal(queries.summaryKindOf("core", "Missing"), undefined);
  });

  for (const [minutes, type] of [[60, { oneofKind: "minutes", minutes: 2 }], [61, { oneofKind: "minutes", minutes: 10 }], [360, { oneofKind: "minutes", minutes: 10 }], [361, { oneofKind: "minutes", minutes: 30 }], [1440, { oneofKind: "minutes", minutes: 30 }], [1441, { oneofKind: "hours", hours: 3 }], [10080, { oneofKind: "hours", hours: 3 }], [10081, { oneofKind: "hours", hours: 12 }]]) {
    it(`selects an LLM interval for ${minutes} minutes`, () => assert.deepEqual(llm.intervalForMinutes(minutes), { type }));
  }
  it("uses now for open LLM ranges and equal preceding comparison windows", () => {
    const now = Date.parse("2026-10-30T12:00:00Z");
    mock.method(Date, "now", () => now);
    const range = llm.rangeFromMinutes(60);
    assert.equal(timestamp.Timestamp.toDate(range.from).getTime(), now - 3_600_000);
    assert.equal(range.to, undefined);
    const comparison = llm.comparisonRange(range);
    assert.equal(timestamp.Timestamp.toDate(comparison.compareFrom).getTime(), now - 7_200_000);
    assert.equal(comparison.compareTo, range.from);
    const closed = llm.comparisonRange({ from: visibility.toTs(3000), to: visibility.toTs(5000) });
    assert.equal(timestamp.Timestamp.toDate(closed.compareFrom).getTime(), 1000);
    assert.deepEqual(llm.comparisonRange({}), {});
  });
  it("combines LLM scope, range, entry scope and explicit extra filters", () => {
    const from = visibility.toTs(1000);
    const filter = llm.buildFilter({ scope: { userRef: { name: "alice", uid: "uid-1" } }, range: { from }, extra: { models: ["model-1"] } });
    assert.equal(filter.userRef.name, "alice");
    assert.deepEqual(filter.from, from);
    assert.equal(filter.entryScope, llmPB.EntryScope.TERMINAL);
    assert.deepEqual(filter.models, ["model-1"]);
    const explicit = llm.buildFilter({ entryScope: llmPB.EntryScope.ALL });
    assert.equal(explicit.entryScope, llmPB.EntryScope.ALL);
  });
  for (const [value, expected] of [[undefined, 0], [0n, 0], [42n, 42], [3.5, 3.5]]) {
    it(`converts the LLM numeric value ${String(value)}`, () => assert.equal(llm.num(value), expected));
  }
  for (const [value, expected] of [[NaN, "—"], [Infinity, "—"], [-1, "—"], [0, "—"], [12.4, "12 ms"], [1000, "1.00 s"], [59999, "60.00 s"], [60000, "1.0 min"]]) {
    it(`formats LLM latency ${value}`, () => assert.equal(llm.formatMs(value), expected));
  }
  it("formats LLM percentages and protects empty cache-hit denominators", () => {
    assert.equal(llm.ratio(2, 5), 40);
    assert.equal(llm.ratio(3, 0), 0);
    assert.equal(llm.ratio(3, -1), 0);
    assert.equal(llm.formatPercent(12.345), "12.3%");
    assert.equal(llm.formatPercent(12.345, 2), "12.35%");
    assert.equal(llm.cacheHitRate(), 0);
    assert.equal(llm.cacheHitRate({ requests: { cacheExactHit: 2n, cacheSemanticHit: 1n, cacheMiss: 3n } }), 50);
    assert.equal(llm.isEmptyStats(), true);
    assert.equal(llm.isEmptyStats({ requests: { total: 1n } }), false);
  });
  it("extracts every selectable LLM metric and defaults absent statistics to zero", () => {
    const metric = llmPB.Metric;
    const stats = { requests: { total: 11n, denied: 2n, failed: 3n, guardrailDenied: 4n, cacheExactHit: 5n, cacheSemanticHit: 6n }, tokens: { total: 100n, input: 60n, output: 40n, reasoningOutput: 7n, cacheReadInput: 8n, discarded: 9n }, latency: { p95Ms: 95, avgMs: 50 }, timeToFirstToken: { p95Ms: 10 }, toolCalls: 12n };
    for (const [name, expected] of [["REQUESTS", 11], ["DENIED_REQUESTS", 2], ["FAILED_REQUESTS", 3], ["GUARDRAIL_DENIED_REQUESTS", 4], ["CACHED_REQUESTS", 11], ["TOTAL_TOKENS", 100], ["INPUT_TOKENS", 60], ["OUTPUT_TOKENS", 40], ["REASONING_OUTPUT_TOKENS", 7], ["CACHE_READ_INPUT_TOKENS", 8], ["DISCARDED_TOKENS", 9], ["LATENCY_P95", 95], ["LATENCY_AVG", 50], ["TIME_TO_FIRST_TOKEN_P95", 10], ["TOOL_CALLS", 12]]) {
      assert.equal(llm.metricAccessor(metric[name]).get(stats), expected, name);
      assert.equal(llm.metricAccessor(metric[name]).get(), 0, name);
    }
    assert.equal(llm.metricAccessor(-1), llm.METRIC_ACCESSORS[0]);
    assert.equal(llm.metricLabel(-1), "Metric");
    assert.equal(llm.dimensionLabel(-1), "Dimension");
    assert.equal(llm.formatMetricValue("duration", 2000), "2.00 s");
    assert.equal(llm.formatMetricValue("tokens", 12), "12");
  });
  for (const [value, expected] of [["", "—"], ["SOURCE_UNSET", "Not set"], ["true", "Yes"], ["false", "No"], ["RATE_LIMITED", "Rate Limited"], ["gpt-model-1", "gpt-model-1"]]) {
    it(`labels the metric key ${JSON.stringify(value)}`, () => assert.equal(llm.prettyKey(value), expected));
  }

  it("normalizes authentication filters and validates known failure reasons", () => {
    for (const [value, expected] of [["ALL", "all"], ["successful", "success"], ["SUCCESS", "success"], ["failed", "failure"], ["Failure", "failure"], ["invalid", undefined], [undefined, undefined]]) assert.equal(failure.parseAuthenticationLogStatus(value), expected);
    for (const reason of failure.FAILURE_REASONS) {
      assert.equal(failure.parseFailureReason(reason.toLowerCase()), reason);
      assert.notEqual(failure.getFailureReasonLabel(reason), "Unknown reason");
    }
    assert.equal(failure.parseFailureReason("unknown"), undefined);
    assert.equal(failure.getFailureReasonLabel(), "Unknown reason");
  });
  it("ranks positive failure reasons and builds matching log paths", () => {
    assert.deepEqual(failure.getFailureReasonSegments({ INVALID_CREDENTIAL: 5, INTERNAL_ERROR: 2, USER_LOCKED: 0, EXPIRED_CREDENTIAL: -1 }, 1), [{ reason: "INVALID_CREDENTIAL", label: "Invalid credential", value: 5 }]);
    assert.deepEqual(failure.getFailureReasonSegments(), []);
    assert.equal(failure.getFailedAuthenticationLogsPath(), "/visibility/authenticationlogs?status=failure");
    assert.equal(failure.getFailedAuthenticationLogsPath("USER_LOCKED"), "/visibility/authenticationlogs?status=failure&reason=USER_LOCKED");
  });

  for (const status of ["run", "ok", "fail", "skip"]) {
    it(`parses a setup step with status ${status}`, () => {
      assert.deepEqual(setup.parseSetupLine(`output @@ocx:nonce:step:install-tool:${status}:detail:with:colons\r\n`, "nonce"), { type: "step", id: "install-tool", status, detail: "detail:with:colons" });
    });
  }
  it("recognizes setup boundaries and facts while ignoring unrelated or malformed output", () => {
    assert.deepEqual(setup.parseSetupLine("@@ocx:nonce:start", "nonce"), { type: "start" });
    assert.deepEqual(setup.parseSetupLine("@@ocx:nonce:done", "nonce"), { type: "done" });
    assert.deepEqual(setup.parseSetupLine("@@ocx:nonce:fact:path=a=b", "nonce"), { type: "fact", key: "path", value: "a=b" });
    for (const line of ["normal output", "@@ocx:other:done", "@@ocx:nonce:fact:=value", "@@ocx:nonce:step:id:unknown", "@@ocx:nonce:unknown"]) assert.equal(setup.parseSetupLine(line, "nonce"), undefined);
    assert.deepEqual(setup.factsFrom([{ type: "start" }, { type: "fact", key: "ssh", value: "ssh" }, { type: "fact", key: "ssh", value: "/usr/bin/ssh" }, { type: "done" }]), { ssh: "/usr/bin/ssh" });
  });
  it("encodes setup scripts without putting their contents in the shell command", () => {
    const script = "printf '%s' \"$SHELL\"\n";
    const command = setup.setupCommand(script, "nonce");
    assert.equal(command, ` echo ${btoa(script)} | base64 -d > /tmp/.octelium-connect-nonce.sh && sh /tmp/.octelium-connect-nonce.sh; exit\r`);
    assert.equal(Buffer.from(command.split(" ")[2], "base64").toString(), script);
  });

  it("CONSOLE-019: distinguishes name-only protobuf references in visibility cache keys", { todo: "CONSOLE-019" }, () => {
    const alice = meta.ObjectReference.create({ name: "alice" });
    const bob = meta.ObjectReference.create({ name: "bob" });
    assert.equal(visibility.refKey(alice), "alice");
    assert.notEqual(visibility.refKey(alice), visibility.refKey(bob));
  });

  it("CONSOLE-023: normalizes negative millisecond timestamps into nonnegative nanos", { todo: "CONSOLE-023" }, () => {
    const value = visibility.toTs(-1);
    assert.equal(value.nanos, 999_000_000);
    assert.equal(timestamp.Timestamp.toDate(value).getTime(), -1);
    assert.equal(timestamp.Timestamp.toJson(value), "1969-12-31T23:59:59.999Z");
  });
});
