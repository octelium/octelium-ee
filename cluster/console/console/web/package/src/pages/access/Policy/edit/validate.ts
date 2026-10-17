import * as AccessP from "@/apis/accessv1/accessv1";

const NAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

export const isValidName = (name: string) => NAME_PATTERN.test(name);

const hasRef = (ref?: { name: string; uid: string }) => !!ref?.name || !!ref?.uid;

export const conditionIssues = (
  condition?: AccessP.Policy_Spec_Rule_Condition,
): string[] => {
  const type = condition?.type;
  if (!type?.oneofKind) return ["Choose what the rule matches"];

  switch (type.oneofKind) {
    case "subject":
      if (type.subject.type.oneofKind === "groupRef")
        return hasRef(type.subject.type.groupRef) ? [] : ["Choose the subject's Group"];
      if (type.subject.type.oneofKind === "userRef")
        return hasRef(type.subject.type.userRef) ? [] : ["Choose the subject User"];
      return ["Choose the subject"];
    case "resource":
      if (type.resource.type.oneofKind === "catalogRef")
        return hasRef(type.resource.type.catalogRef) ? [] : ["Choose the Catalog"];
      if (type.resource.type.oneofKind === "serviceRef")
        return hasRef(type.resource.type.serviceRef) ? [] : ["Choose the Service"];
      return ["Choose the resource"];
    case "userRef":
      return hasRef(type.userRef) ? [] : ["Choose the requester"];
    case "match":
      return type.match.trim() ? [] : ["Write the CEL expression"];
    case "all":
    case "any": {
      const children = type.oneofKind === "all" ? type.all.of : type.any.of;
      if (children.length === 0) return ["Add at least one condition to the group"];
      return children.flatMap((child) => conditionIssues(child));
    }
    default:
      return [];
  }
};

export const surfaceIssues = (
  surface: AccessP.Policy_Spec_Rule_Surface,
): string[] => {
  const issues: string[] = [];
  if (!hasRef(surface.destination?.integrationRef))
    issues.push("Choose the Integration of every delivery");
  if (!surface.destination?.audience)
    issues.push("Choose who every delivery goes to");
  return issues;
};

export const stepIssues = (
  step: AccessP.Policy_Spec_Rule_Action_Review_Step,
  index: number,
  steps: AccessP.Policy_Spec_Rule_Action_Review_Step[],
): string[] => {
  const issues: string[] = [];
  const label = step.name || `Step ${index + 1}`;

  if (!step.name) issues.push(`${label}: give the step a name`);
  else if (!isValidName(step.name))
    issues.push(`${label}: use lowercase letters, digits and dashes in the name`);
  else if (steps.filter((other) => other.name === step.name).length > 1)
    issues.push(`${label}: step names must be unique`);

  if (step.reviewers.length === 0) issues.push(`${label}: add at least one reviewer`);
  if (
    step.reviewers.some(
      (reviewer) =>
        (reviewer.type.oneofKind === "user" && !hasRef(reviewer.type.user.userRef)) ||
        (reviewer.type.oneofKind === "group" && !hasRef(reviewer.type.group.groupRef)),
    )
  )
    issues.push(`${label}: a reviewer is missing its User or Group`);

  const requirement = AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement;
  if (step.approvalRequirement === requirement.APPROVAL_REQUIREMENT_UNSET)
    issues.push(`${label}: choose how many reviewers must approve`);
  if (step.approvalRequirement === requirement.COUNT) {
    if (step.approvalCount < 1) issues.push(`${label}: set how many approvals are needed`);
    else if (
      step.reviewers.length > 0 &&
      step.reviewers.every((reviewer) => reviewer.type.oneofKind === "user") &&
      step.approvalCount > step.reviewers.length
    )
      issues.push(
        `${label}: needs ${step.approvalCount} approvals but has only ${step.reviewers.length} reviewers`,
      );
  }

  step.surfaces.forEach((surface) => {
    surfaceIssues(surface).forEach((issue) => issues.push(`${label}: ${issue.toLowerCase()}`));
  });

  return [...new Set(issues)];
};

export const ruleIssues = (
  rule: AccessP.Policy_Spec_Rule,
  rules: AccessP.Policy_Spec_Rule[],
): string[] => {
  const issues: string[] = [];

  if (!rule.name.trim()) issues.push("Give the rule a name");
  else if (rules.filter((other) => other.name === rule.name).length > 1)
    issues.push("Rule names must be unique within the Policy");

  issues.push(...conditionIssues(rule.condition));

  if (rule.effect === AccessP.Policy_Spec_Rule_Effect.EFFECT_UNSET)
    issues.push("Choose what happens to matching requests");

  if (rule.effect === AccessP.Policy_Spec_Rule_Effect.REVIEW) {
    const steps =
      rule.action?.type.oneofKind === "review" ? rule.action.type.review.steps : [];
    if (steps.length === 0) issues.push("Add at least one review step");
    steps.forEach((step, index) => issues.push(...stepIssues(step, index, steps)));
  }

  rule.notifications.forEach((surface) =>
    surfaceIssues(surface).forEach((issue) => issues.push(`Notification: ${issue.toLowerCase()}`)),
  );

  return [...new Set(issues)];
};
