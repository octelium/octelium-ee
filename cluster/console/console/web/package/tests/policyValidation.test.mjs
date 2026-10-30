import assert from "node:assert/strict";
import { after, before, describe, it } from "node:test";
import { createConsoleModules } from "./helpers/consoleModules.mjs";

describe("Access Policy validation", () => {
  let modules;
  let validation;
  let access;
  const condition = (type) => access.Policy_Spec_Rule_Condition.create({ type });
  const reviewer = (kind = "user", ref = { name: "alice" }) => ({
    type: { oneofKind: kind, [kind]: { [`${kind}Ref`]: ref } },
  });
  const step = (extra = {}) => access.Policy_Spec_Rule_Action_Review_Step.create({
    name: "review", reviewers: [reviewer()],
    approvalRequirement: access.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.ANY,
    ...extra,
  });
  const rule = (extra = {}) => access.Policy_Spec_Rule.create({
    name: "allow", condition: condition({ oneofKind: "matchAny", matchAny: true }),
    effect: access.Policy_Spec_Rule_Effect.AUTO_APPROVE, ...extra,
  });
  const surface = (extra = {}) => access.Policy_Spec_Rule_Surface.create({
    destination: { integrationRef: { name: "slack" }, audience: access.Policy_Spec_Rule_Surface_Destination_Audience.REVIEWERS },
    ...extra,
  });

  before(async () => {
    modules = await createConsoleModules();
    [validation, access] = await Promise.all([
      modules.load("pages/access/Policy/edit/validate.ts"),
      modules.load("apis/accessv1/accessv1.ts"),
    ]);
  });
  after(async () => modules?.close());

  for (const name of ["a", "0", "review-2", "a".repeat(63)]) {
    it(`accepts a valid step name of length ${name.length}: ${name.slice(0, 15)}`, () => assert.equal(validation.isValidName(name), true));
  }
  for (const name of ["", " ", "Upper", "-start", "end-", "under_score", "name.space", "a".repeat(64), "é", "line\n"]) {
    it(`rejects the step name ${JSON.stringify(name)}`, () => assert.equal(validation.isValidName(name), false));
  }

  it("requires a condition type and accepts match-any conditions", () => {
    assert.deepEqual(validation.conditionIssues(), ["Choose what the rule matches"]);
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: undefined })), ["Choose what the rule matches"]);
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: "matchAny", matchAny: true })), []);
  });

  for (const [kind, refKind, error] of [
    ["subject", "groupRef", "Choose the subject's Group"],
    ["subject", "userRef", "Choose the subject User"],
    ["resource", "catalogRef", "Choose the Catalog"],
    ["resource", "serviceRef", "Choose the Service"],
  ]) {
    it(`requires a ${refKind} in a ${kind} condition`, () => {
      const build = (ref) => condition({ oneofKind: kind, [kind]: { type: { oneofKind: refKind, [refKind]: ref } } });
      assert.deepEqual(validation.conditionIssues(build({})), [error]);
      assert.deepEqual(validation.conditionIssues(build({ name: "example" })), []);
      assert.deepEqual(validation.conditionIssues(build({ uid: "uid-1" })), []);
    });
  }

  for (const kind of ["subject", "resource"]) {
    it(`requires a selection inside a ${kind} condition`, () => {
      assert.deepEqual(validation.conditionIssues(condition({ oneofKind: kind, [kind]: { type: { oneofKind: undefined } } })), [`Choose the ${kind}`]);
    });
  }

  it("validates requester references by either name or UID", () => {
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: "userRef", userRef: {} })), ["Choose the requester"]);
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: "userRef", userRef: { uid: "uid-1" } })), []);
  });

  it("requires a nonblank CEL expression", () => {
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: "match", match: " \n " })), ["Write the CEL expression"]);
    assert.deepEqual(validation.conditionIssues(condition({ oneofKind: "match", match: "request.spec != null" })), []);
  });

  for (const kind of ["all", "any"]) {
    it(`validates empty and nested ${kind} condition groups`, () => {
      assert.deepEqual(validation.conditionIssues(condition({ oneofKind: kind, [kind]: { of: [] } })), ["Add at least one condition to the group"]);
      assert.deepEqual(validation.conditionIssues(condition({ oneofKind: kind, [kind]: { of: [
        condition({ oneofKind: "match", match: " " }),
        condition({ oneofKind: "userRef", userRef: {} }),
      ] } })), ["Write the CEL expression", "Choose the requester"]);
    });
  }

  it("requires an Integration and audience for each delivery", () => {
    assert.deepEqual(validation.surfaceIssues(access.Policy_Spec_Rule_Surface.create()), ["Choose the Integration of every delivery", "Choose who every delivery goes to"]);
    assert.deepEqual(validation.surfaceIssues(surface()), []);
    assert.deepEqual(validation.surfaceIssues(surface({ destination: { integrationRef: { uid: "uid-1" }, audience: access.Policy_Spec_Rule_Surface_Destination_Audience.REQUESTER } })), []);
  });

  it("accepts a complete review step", () => {
    const value = step();
    assert.deepEqual(validation.stepIssues(value, 0, [value]), []);
  });

  it("reports missing step name, reviewers and approval requirement together", () => {
    const value = access.Policy_Spec_Rule_Action_Review_Step.create();
    assert.deepEqual(validation.stepIssues(value, 1, [value]), [
      "Step 2: give the step a name", "Step 2: add at least one reviewer", "Step 2: choose how many reviewers must approve",
    ]);
  });

  it("rejects invalid and duplicate step names", () => {
    const invalid = step({ name: "Review" });
    assert.deepEqual(validation.stepIssues(invalid, 0, [invalid]), ["Review: use lowercase letters, digits and dashes in the name"]);
    const first = step();
    assert.deepEqual(validation.stepIssues(first, 0, [first, step()]), ["review: step names must be unique"]);
  });

  for (const kind of ["user", "group"]) {
    it(`requires a reference for a ${kind} reviewer`, () => {
      const missing = step({ reviewers: [reviewer(kind, {})] });
      assert.deepEqual(validation.stepIssues(missing, 0, [missing]), ["review: a reviewer is missing its User or Group"]);
      const present = step({ reviewers: [reviewer(kind, { uid: "uid-1" })] });
      assert.deepEqual(validation.stepIssues(present, 0, [present]), []);
    });
  }

  it("checks explicit approval counts against individually named reviewers", () => {
    const requirement = access.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.COUNT;
    const missing = step({ approvalRequirement: requirement });
    assert.deepEqual(validation.stepIssues(missing, 0, [missing]), ["review: set how many approvals are needed"]);
    const excessive = step({ approvalRequirement: requirement, approvalCount: 2 });
    assert.deepEqual(validation.stepIssues(excessive, 0, [excessive]), ["review: needs 2 approvals but has only 1 reviewers"]);
    const valid = step({ approvalRequirement: requirement, approvalCount: 1 });
    assert.deepEqual(validation.stepIssues(valid, 0, [valid]), []);
  });

  it("allows group reviewers to satisfy a larger approval count", () => {
    const value = step({ reviewers: [reviewer("group")], approvalRequirement: access.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement.COUNT, approvalCount: 3 });
    assert.deepEqual(validation.stepIssues(value, 0, [value]), []);
  });

  it("prefixes and deduplicates delivery problems within a step", () => {
    const value = step({ surfaces: [access.Policy_Spec_Rule_Surface.create(), access.Policy_Spec_Rule_Surface.create()] });
    assert.deepEqual(validation.stepIssues(value, 0, [value]), ["review: choose the integration of every delivery", "review: choose who every delivery goes to"]);
  });

  it("requires rule names and rejects duplicate names", () => {
    const unnamed = rule({ name: " \n " });
    assert.deepEqual(validation.ruleIssues(unnamed, [unnamed]), ["Give the rule a name"]);
    const value = rule();
    assert.deepEqual(validation.ruleIssues(value, [value, rule()]), ["Rule names must be unique within the Policy"]);
  });

  it("reports missing effect, condition and review steps", () => {
    const empty = access.Policy_Spec_Rule.create();
    assert.deepEqual(validation.ruleIssues(empty, [empty]), ["Give the rule a name", "Choose what the rule matches", "Choose what happens to matching requests"]);
    const review = rule({ effect: access.Policy_Spec_Rule_Effect.REVIEW });
    assert.deepEqual(validation.ruleIssues(review, [review]), ["Add at least one review step"]);
  });

  it("validates review steps only when the rule requires review", () => {
    const action = { type: { oneofKind: "review", review: { steps: [step({ reviewers: [] })] } } };
    const review = rule({ effect: access.Policy_Spec_Rule_Effect.REVIEW, action });
    assert.deepEqual(validation.ruleIssues(review, [review]), ["review: add at least one reviewer"]);
    const deny = rule({ effect: access.Policy_Spec_Rule_Effect.DENY, action });
    assert.deepEqual(validation.ruleIssues(deny, [deny]), []);
  });

  it("validates notifications and deduplicates rule issues", () => {
    const value = rule({ notifications: [access.Policy_Spec_Rule_Surface.create(), access.Policy_Spec_Rule_Surface.create()] });
    assert.deepEqual(validation.ruleIssues(value, [value]), ["Notification: choose the integration of every delivery", "Notification: choose who every delivery goes to"]);
  });

  it("CONSOLE-012: rejects a reviewer without a selected type", { todo: "CONSOLE-012" }, () => {
    const value = step({ reviewers: [access.Policy_Spec_Rule_Action_Review_Step_Reviewer.create()] });
    assert.ok(validation.stepIssues(value, 0, [value]).some((issue) => /reviewer|User|Group/.test(issue)));
  });
});
