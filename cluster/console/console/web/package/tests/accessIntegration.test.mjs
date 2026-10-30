import assert from "node:assert/strict";
import { after, before, describe, it } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Access Integration presentation and workflow labels", () => {
  let modules;
  let access;
  let integration;
  let request;
  let review;
  let filters;
  const item = (kind, config = {}, status) => access.Integration.create({
    spec: kind ? { type: { oneofKind: kind, [kind]: config } } : undefined,
    status,
  });
  const secret = (name) => ({ type: { oneofKind: "fromSecret", fromSecret: name } });

  before(async () => {
    modules = await createConsoleModules();
    [access, integration, request, review, filters] = await Promise.all([
      modules.load("apis/accessv1/accessv1.ts"), modules.load("pages/access/Integration/utils.ts"),
      modules.load("pages/access/Request/utils.ts"), modules.load("pages/access/Review/utils.ts"),
      modules.load("pages/access/listFilters.ts"),
    ]);
  });
  after(async () => modules?.close());

  for (const [kind, title, filter] of [["slack", "Slack", "SLACK"], ["jira", "Jira", "JIRA"], ["webhook", "Webhook", "WEBHOOK"]]) {
    it(`identifies ${kind} Integrations and their filter values`, () => {
      const value = item(kind);
      assert.equal(integration.getType(value), title);
      assert.equal(integration.getTypeFilterValue(value), filter);
      assert.equal(integration.getProviderSegment(value), kind);
      assert.equal(integration.getIntegrationPresentation(value).isConfigured, true);
    });
  }
  it("uses explicit empty states for an unconfigured Integration", () => {
    const value = item();
    assert.equal(integration.getType(value), "Not configured");
    assert.equal(integration.getTypeFilterValue(value), undefined);
    assert.equal(integration.getProviderSegment(value), undefined);
    assert.equal(integration.getSharedDestination(value), undefined);
    assert.deepEqual(integration.integrationSecretNames(value), []);
    assert.equal(integration.getIntegrationPresentation(value).isConfigured, false);
  });
  for (const [kind, config, expected] of [
    ["slack", { channelID: "channel-1" }, { label: "Channel ID", value: "channel-1" }],
    ["jira", { projectKey: "OPS" }, { label: "Project key", value: "OPS" }],
    ["webhook", { url: "https://hooks.example.test" }, { label: "Delivery URL", value: "https://hooks.example.test" }],
  ]) {
    it(`shows the ${kind} shared destination only when configured`, () => {
      assert.deepEqual(integration.getSharedDestination(item(kind, config)), expected);
      assert.equal(integration.getSharedDestination(item(kind)), undefined);
    });
  }
  for (const [kind, config] of [
    ["slack", () => ({ botToken: secret("bot"), signingSecret: secret("signing") })],
    ["jira", () => ({ apiToken: secret("bot"), webhookSecret: secret("signing") })],
    ["webhook", () => ({ signingSecret: secret("bot"), inboundSecret: secret("signing") })],
  ]) {
    it(`lists the referenced Secrets for ${kind}`, () => {
      assert.deepEqual(integration.integrationSecretNames(item(kind, config())), ["bot", "signing"]);
      assert.deepEqual(integration.integrationSecretNames(item(kind)), []);
    });
  }
  it("deduplicates referenced Secrets and omits empty Secret names", () => {
    assert.deepEqual(integration.integrationSecretNames(item("slack", { botToken: secret("shared"), signingSecret: secret("shared") })), ["shared"]);
    assert.deepEqual(integration.integrationSecretNames(item("slack", { botToken: secret("") })), []);
  });
  for (const [kind, keys] of [["slack", ["interactions", "commands", "events"]], ["jira", ["webhook"]], ["webhook", ["inbound"]]]) {
    it(`builds ${kind} callback URLs from the cluster domain and Integration ID`, () => {
      const endpoints = integration.getInboundEndpoints(item(kind, {}, { id: "integration-1" }), "cluster.example.test");
      assert.deepEqual(endpoints.map((endpoint) => endpoint.key), keys);
      assert.ok(endpoints.every((endpoint) => endpoint.url.startsWith(`https://public.octelium.cluster.example.test/integration/v1/callback/integration-1/${kind}`)));
      assert.deepEqual(integration.getInboundEndpoints(item(kind), "cluster.example.test"), []);
      if (kind === "slack") assert.deepEqual(endpoints.map((endpoint) => endpoint.isOptional === true), [false, true, true]);
    });
  }
  it("does not publish callbacks for unknown Integration providers", () => {
    assert.deepEqual(integration.getInboundEndpoints(item(undefined, {}, { id: "integration-1" }), "cluster.example.test"), []);
  });
  it("checks declared capabilities without inventing absent capabilities", () => {
    const capability = access.Integration_Status_Capability;
    const value = item("slack", {}, { capabilities: [capability.NOTIFICATION] });
    assert.equal(integration.hasCapability(value, capability.NOTIFICATION), true);
    assert.equal(integration.hasCapability(value, capability.INTERACTIVE_REVIEW), false);
    assert.equal(integration.hasCapability(item(), capability.NOTIFICATION), false);
    assert.equal(integration.getCapabilityLabel(capability.NOTIFICATION), "Shared notification");
    assert.equal(integration.getCapabilityLabel(capability.DIRECT_USER_DELIVERY), "Direct user delivery");
    assert.equal(integration.getCapabilityLabel(capability.INTERACTIVE_REVIEW), "Interactive review");
    assert.equal(integration.getCapabilityLabel(capability.REQUEST_CREATION), "Request creation");
    assert.equal(integration.getCapabilityLabel(capability.IDENTITY_RESOLUTION), "Identity resolution");
    assert.equal(integration.getCapabilityLabel(capability.PRESENTATION_UPDATE), "Presentation update");
    assert.equal(integration.getCapabilityLabel(-1), "Unknown");
  });
  for (const [name, label, tone] of [["READY", "Ready", "success"], ["DEGRADED", "Degraded", "warning"], ["ERROR", "Error", "danger"]]) {
    it(`labels Integration state ${name}`, () => assert.deepEqual(integration.getStateMeta(access.Integration_Status_State[name]), { label, tone }));
  }
  for (const [name, label, tone] of [["SYNC_REQUESTED", "Sync requested", "warning"], ["SYNCING", "Synchronizing", "info"], ["SUCCESS", "Successful", "success"], ["FAILED", "Failed", "danger"]]) {
    it(`labels synchronization state ${name}`, () => assert.deepEqual(integration.getSyncStateMeta(access.Integration_Status_Synchronization_State[name]), { label, tone }));
  }
  it("selects the last successful synchronization without sorting stored history", () => {
    const state = access.Integration_Status_Synchronization_State;
    const value = item("slack", {}, {
      state: access.Integration_Status_State.READY,
      synchronization: { state: state.FAILED, completedAt: { seconds: 40 } },
      lastSynchronizations: [
        { state: state.SUCCESS, createdAt: { seconds: 10 } },
        { state: state.SUCCESS, completedAt: { seconds: 30 } },
        { state: state.FAILED, completedAt: { seconds: 35 } },
      ],
    });
    const previous = [...value.status.lastSynchronizations];
    const presentation = integration.getIntegrationPresentation(value);
    assert.equal(presentation.allSynchronizations.length, 4);
    assert.equal(presentation.lastSuccessfulSync, value.status.lastSynchronizations[1]);
    assert.equal(presentation.currentSync, value.status.synchronization);
    assert.deepEqual(value.status.lastSynchronizations, previous);
    assert.equal(presentation.isSynchronizing, false);
  });
  for (const name of ["SYNC_REQUESTED", "SYNCING", "SUCCESS", "FAILED"]) {
    it(`detects active synchronization state ${name}`, () => {
      assert.equal(integration.getIntegrationPresentation(item("slack", {}, { synchronization: { state: access.Integration_Status_Synchronization_State[name] } })).isSynchronizing, ["SYNC_REQUESTED", "SYNCING"].includes(name));
    });
  }
  for (const [name, label, tone] of [["PENDING", "Pending", "warning"], ["READY", "Delivered", "success"], ["DEGRADED", "Failing", "danger"], ["CLOSED", "Closed", "neutral"]]) {
    it(`labels delivery state ${name}`, () => assert.deepEqual(integration.getDeliveryStateMeta(access.IntegrationBinding_Status_State[name]), { label, tone }));
  }
  it("detects pending delivery revisions and consecutive delivery failures", () => {
    assert.equal(integration.isDeliveryOutOfDate({}), false);
    assert.equal(integration.isDeliveryOutOfDate({ status: { desiredRevision: "new", appliedRevision: "old" } }), true);
    assert.equal(integration.isDeliveryOutOfDate({ status: { desiredRevision: "same", appliedRevision: "same" } }), false);
    assert.equal(integration.isDeliveryFailing({}), false);
    assert.equal(integration.isDeliveryFailing({ status: { attempts: 0 } }), false);
    assert.equal(integration.isDeliveryFailing({ status: { attempts: 2 } }), true);
  });
  it("labels delivery purpose, identity sources and unknown states", () => {
    assert.equal(integration.getDeliveryPurposeLabel(access.IntegrationBinding_Status_Purpose.REVIEW_SURFACE), "Review request");
    assert.equal(integration.getDeliveryPurposeLabel(access.IntegrationBinding_Status_Purpose.NOTIFICATION), "Outcome notification");
    assert.equal(integration.getDeliveryPurposeLabel(), "Delivery");
    assert.equal(integration.getIdentitySourceLabel(access.IntegrationIdentity_Status_Source.EMAIL_DISCOVERY), "Matched by email");
    assert.equal(integration.getIdentitySourceLabel(), "Unknown");
    assert.deepEqual(integration.getStateMeta(-1), { label: "Not checked", tone: "neutral" });
    assert.deepEqual(integration.getSyncStateMeta(-1), { label: "Unknown", tone: "neutral" });
    assert.deepEqual(integration.getDeliveryStateMeta(-1), { label: "Unknown", tone: "neutral" });
  });
  for (const [name, label] of [["SHARED", "Shared destination"], ["REVIEWERS", "Reviewers"], ["REQUESTER", "Requester"], ["SUBJECT", "Subject"]]) {
    it(`labels delivery audience ${name}`, () => assert.equal(integration.getAudienceLabel(access.Policy_Spec_Rule_Surface_Destination_Audience[name]), label));
  }
  it("defaults unknown audiences and interaction modes", () => {
    assert.equal(integration.getAudienceLabel(), "Unset");
    assert.equal(integration.getInteractionModeLabel(), "Deep link only");
    assert.equal(integration.getInteractionModeLabel(access.Policy_Spec_Rule_Surface_InteractionMode.INTERACTIVE), "Interactive");
  });

  for (const [name, label, tone] of [["PENDING", "Pending", "warning"], ["APPROVED", "Approved", "success"], ["REJECTED", "Rejected", "danger"], ["REVOKED", "Revoked", "danger"], ["EXPIRED", "Expired", "neutral"], ["CANCELLED", "Cancelled", "neutral"]]) {
    it(`labels Request state ${name}`, () => {
      assert.equal(request.getStatusMeta(access.Request_Status_State_Status[name]).label, label);
      assert.equal(request.getStatusTone(access.Request_Status_State_Status[name]), tone);
    });
  }
  it("uses neutral fallbacks for unknown Request and Review states", () => {
    assert.equal(request.getStatusMeta(-1).label, "Unknown");
    assert.equal(request.getStatusTone(-1), "neutral");
    assert.equal(request.getUrgencyLabel(-1), "Unset");
    assert.equal(review.getDecisionMeta(-1).label, "Pending");
  });
  it("formats explicit approval counts and last-step timeout behavior", () => {
    const step = access.Policy_Spec_Rule_Action_Review_Step.create({ approvalRequirement: access.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.COUNT, approvalCount: 1, onTimeout: access.Policy_Spec_Rule_Action_Review_Step_OnTimeout.GOTO_NEXT_STEP });
    assert.equal(request.getApprovalRequirementLabel(step), "1 approval needed");
    step.approvalCount = 2;
    assert.equal(request.getApprovalRequirementLabel(step), "2 approvals needed");
    assert.equal(request.getOnTimeoutLabel(step, false), "then moves to the next step");
    assert.equal(request.getOnTimeoutLabel(step, true), "then the request expires");
    step.onTimeout = access.Policy_Spec_Rule_Action_Review_Step_OnTimeout.REJECT;
    assert.equal(request.getOnTimeoutLabel(step), "then the request is rejected");
  });
  it("offers unique list-filter keys and excludes unset enum choices", () => {
    for (const kind of ["Request", "Review", "Policy", "IntegrationIdentity", "IntegrationBinding"]) {
      const values = filters.getAccessListFilters(kind);
      assert.equal(new Set(values.map((filter) => filter.key)).size, values.length);
      for (const filter of values) {
        assert.ok(filter.label);
        for (const option of filter.options ?? []) {
          assert.ok(option.label);
          assert.doesNotMatch(option.value, /UNSET$/);
        }
      }
    }
    assert.deepEqual(filters.getAccessListFilters("Missing"), []);
  });

  it("CONSOLE-022: selects the latest successful sync using subsecond timestamp precision", { todo: "CONSOLE-022" }, () => {
    const state = access.Integration_Status_Synchronization_State.SUCCESS;
    const value = item("slack", {}, { lastSynchronizations: [
      { state, completedAt: { seconds: 100, nanos: 100_000_000 } },
      { state, completedAt: { seconds: 100, nanos: 900_000_000 } },
    ] });
    assert.equal(integration.getIntegrationPresentation(value).lastSuccessfulSync, value.status.lastSynchronizations[1]);
  });
});
