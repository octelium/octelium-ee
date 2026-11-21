import * as AccessP from "@/apis/accessv1/accessv1";
import DurationPicker from "@/components/DurationPicker";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import { twMerge } from "@/lib/utils";
import { useListKeys } from "@/utils/forms";
import { getResourceRef } from "@/utils/pb";
import {
  ActionIcon,
  Button,
  NumberInput,
  SegmentedControl,
  Switch,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { stepIssues } from "./validate";
import { SurfacesEdit } from "./Surface";

const Requirement = AccessP.Policy_Spec_Rule_Action_Review_Step_ApprovalRequirement;
const OnTimeout = AccessP.Policy_Spec_Rule_Action_Review_Step_OnTimeout;
const ReasonRequirement = AccessP.Policy_Spec_Rule_Action_Review_Step_ReasonRequirement;

export const newReviewStep = (name: string) =>
  AccessP.Policy_Spec_Rule_Action_Review_Step.create({
    name,
    approvalRequirement: Requirement.ANY,
    onTimeout: OnTimeout.REJECT,
  });

export const nextStepName = (steps: AccessP.Policy_Spec_Rule_Action_Review_Step[]) => {
  const names = new Set(steps.map((step) => step.name));
  const base = steps.length === 0 ? "approval" : "step";
  if (!names.has(base)) return base;
  for (let i = 2; ; i++) if (!names.has(`${base}-${i}`)) return `${base}-${i}`;
};

const Field = (props: { label: string; hint?: string; children: React.ReactNode; className?: string }) => (
  <div className={twMerge("flex min-w-0 flex-col gap-1.5", props.className)}>
    <span className="flex flex-col">
      <span className="text-xs font-semibold text-slate-700">{props.label}</span>
      {props.hint && <span className="text-micro leading-4 text-slate-500">{props.hint}</span>}
    </span>
    {props.children}
  </div>
);

type Reviewer = AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer;

export const ReviewersEdit = (props: {
  reviewers: Reviewer[];
  onChange: (reviewers: Reviewer[]) => void;
  allowCatalogOwners?: boolean;
  groupDescription?: string;
  userDescription?: string;
}) => {
  const { reviewers } = props;
  const userNames = reviewers.flatMap((reviewer) =>
    reviewer.type.oneofKind === "user" && reviewer.type.user.userRef?.name
      ? [reviewer.type.user.userRef.name]
      : [],
  );
  const groupNames = reviewers.flatMap((reviewer) =>
    reviewer.type.oneofKind === "group" && reviewer.type.group.groupRef?.name
      ? [reviewer.type.group.groupRef.name]
      : [],
  );
  const hasOwners = reviewers.some((reviewer) => reviewer.type.oneofKind === "catalogOwners");

  const setReviewers = (
    users: Reviewer[] | undefined,
    groups: Reviewer[] | undefined,
    owners: boolean = hasOwners,
  ) => {
    const keepUsers = reviewers.filter((reviewer) => reviewer.type.oneofKind === "user");
    const keepGroups = reviewers.filter((reviewer) => reviewer.type.oneofKind === "group");
    props.onChange([
      ...(users ?? keepUsers),
      ...(groups ?? keepGroups),
      ...(owners
        ? [
            AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer.create({
              type: { oneofKind: "catalogOwners", catalogOwners: {} },
            }),
          ]
        : []),
    ]);
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-1 gap-3 @2xl:grid-cols-2">
        <SelectResourceMultiple
          api="core"
          kind="Group"
          label="Reviewer Groups"
          description={props.groupDescription ?? "Any member of these Groups can review."}
          defaultValue={groupNames}
          clearable
          onChange={(items) =>
            setReviewers(
              undefined,
              (items ?? []).map((item) =>
                AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer.create({
                  type: { oneofKind: "group", group: { groupRef: getResourceRef(item) } },
                }),
              ),
            )
          }
        />
        <SelectResourceMultiple
          api="core"
          kind="User"
          label="Reviewer Users"
          description={props.userDescription ?? "These Users can review."}
          defaultValue={userNames}
          clearable
          onChange={(items) =>
            setReviewers(
              (items ?? []).map((item) =>
                AccessP.Policy_Spec_Rule_Action_Review_Step_Reviewer.create({
                  type: { oneofKind: "user", user: { userRef: getResourceRef(item) } },
                }),
              ),
              undefined,
            )
          }
        />
      </div>
      {props.allowCatalogOwners && (
        <Switch
          size="xs"
          label="The owners of the requested Catalog can review"
          checked={hasOwners}
          onChange={(event) => setReviewers(undefined, undefined, event.target.checked)}
        />
      )}
    </div>
  );
};

const StepEdit = (props: {
  step: AccessP.Policy_Spec_Rule_Action_Review_Step;
  index: number;
  steps: AccessP.Policy_Spec_Rule_Action_Review_Step[];
  onUpdate: () => void;
  onMove: (delta: number) => void;
  onRemove: () => void;
}) => {
  const rows = useListKeys();
  const { step, index, steps, onUpdate } = props;
  const isLast = index === steps.length - 1;
  const issues = stepIssues(step, index, steps);
  const sod = step.separationOfDuties;

  const timeoutValue =
    step.onTimeout === OnTimeout.GOTO_NEXT_STEP
      ? "next"
      : step.onTimeout === OnTimeout.REJECT
        ? "reject"
        : "expire";

  return (
    <li className="relative flex gap-3">
      <div className="flex flex-col items-center">
        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-amber-200 bg-amber-50 text-xs font-bold text-amber-700">
          {index + 1}
        </span>
        {!isLast && <span aria-hidden="true" className="mt-1 w-px flex-1 bg-slate-200" />}
      </div>

      <div className="mb-4 flex min-w-0 flex-1 flex-col gap-4 rounded-xl border border-slate-200 bg-white p-4 shadow-card">
        <div className="flex flex-wrap items-end gap-3">
          <TextInput
            className="min-w-[200px] flex-1"
            label="Step name"
            description="Shown to reviewers and in the request history."
            placeholder="security-review"
            value={step.name}
            error={
              step.name && steps.filter((other) => other.name === step.name).length > 1
                ? "Already used by another step"
                : undefined
            }
            onChange={(event) => {
              step.name = event.target.value
                .toLowerCase()
                .replace(/[^a-z0-9-]/g, "-");
              onUpdate();
            }}
          />
          <div className="flex items-center gap-0.5 pb-1">
            <Tooltip label="Move up" withArrow>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                aria-label="Move step up"
                disabled={index === 0}
                onClick={() => props.onMove(-1)}
              >
                <ArrowUp size={13} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label="Move down" withArrow>
              <ActionIcon
                variant="subtle"
                color="gray"
                size="sm"
                aria-label="Move step down"
                disabled={isLast}
                onClick={() => props.onMove(1)}
              >
                <ArrowDown size={13} />
              </ActionIcon>
            </Tooltip>
            <Tooltip label={steps.length === 1 ? "A review needs at least one step" : "Remove step"} withArrow>
              <ActionIcon
                variant="subtle"
                color="red"
                size="sm"
                aria-label="Remove step"
                disabled={steps.length === 1}
                onClick={props.onRemove}
              >
                <Trash2 size={13} />
              </ActionIcon>
            </Tooltip>
          </div>
        </div>

        <ReviewersEdit
          reviewers={step.reviewers}
          allowCatalogOwners
          onChange={(reviewers) => {
            step.reviewers = reviewers;
            onUpdate();
          }}
        />

        <Field
          label="Approvals needed"
          hint="The step is cleared once enough reviewers approve. A single rejection rejects the request."
        >
          <div className="flex flex-wrap items-center gap-3">
            <SegmentedControl
              size="xs"
              value={
                step.approvalRequirement === Requirement.ALL
                  ? "ALL"
                  : step.approvalRequirement === Requirement.COUNT
                    ? "COUNT"
                    : "ANY"
              }
              data={[
                { label: "Any one reviewer", value: "ANY" },
                { label: "Every reviewer", value: "ALL" },
                { label: "A number of reviewers", value: "COUNT" },
              ]}
              onChange={(value) => {
                step.approvalRequirement =
                  value === "ALL" ? Requirement.ALL : value === "COUNT" ? Requirement.COUNT : Requirement.ANY;
                step.approvalCount = value === "COUNT" ? Math.max(2, step.approvalCount) : 0;
                onUpdate();
              }}
            />
            {step.approvalRequirement === Requirement.COUNT && (
              <NumberInput
                size="xs"
                className="w-24"
                aria-label="Approvals needed"
                min={1}
                max={100}
                value={step.approvalCount}
                onChange={(value) => {
                  step.approvalCount = Math.max(0, Math.floor(Number(value) || 0));
                  onUpdate();
                }}
              />
            )}
          </div>
          {step.approvalRequirement === Requirement.ALL &&
            step.reviewers.some((reviewer) => reviewer.type.oneofKind === "group") && (
              <span className="text-micro leading-4 text-slate-500">
                With a Group, every one of its members must approve.
              </span>
            )}
        </Field>

        <div className="grid grid-cols-1 gap-4 @2xl:grid-cols-2">
          <Field label="Time limit" hint="Leave empty to wait for a decision until the request's own deadline.">
            <DurationPicker
              value={step.timeout}
              placeholder="No time limit"
              onChange={(value) => {
                step.timeout = value;
                onUpdate();
              }}
            />
          </Field>
          <Field
            label="When the time limit passes"
            hint={step.timeout ? undefined : "Applies once a time limit is set."}
          >
            <SegmentedControl
              size="xs"
              disabled={!step.timeout}
              value={isLast && timeoutValue === "next" ? "expire" : timeoutValue}
              data={[
                { label: "Reject", value: "reject" },
                ...(isLast ? [] : [{ label: "Skip to the next step", value: "next" }]),
                { label: "Expire", value: "expire" },
              ]}
              onChange={(value) => {
                step.onTimeout =
                  value === "next"
                    ? OnTimeout.GOTO_NEXT_STEP
                    : value === "reject"
                      ? OnTimeout.REJECT
                      : OnTimeout.UNSET;
                onUpdate();
              }}
            />
            {step.timeout && (
              <span className="text-micro leading-4 text-slate-500">
                {timeoutValue === "reject"
                  ? "The request is rejected."
                  : timeoutValue === "next" && !isLast
                    ? "The step is skipped without any approval and the next step starts."
                    : "The request expires without a decision; the requester can ask again."}
              </span>
            )}
          </Field>
        </div>

        <Field
          label="Reason for a decision"
          hint="Whether reviewers must explain their decision. Integrations hand such decisions off to the access portal."
        >
          <SegmentedControl
            size="xs"
            aria-label="Reason for a decision"
            value={
              step.reasonRequirement === ReasonRequirement.ALWAYS
                ? "always"
                : step.reasonRequirement === ReasonRequirement.REJECT_ONLY
                  ? "reject"
                  : "optional"
            }
            data={[
              { label: "Optional", value: "optional" },
              { label: "Required to reject", value: "reject" },
              { label: "Always required", value: "always" },
            ]}
            onChange={(value) => {
              step.reasonRequirement =
                value === "always"
                  ? ReasonRequirement.ALWAYS
                  : value === "reject"
                    ? ReasonRequirement.REJECT_ONLY
                    : ReasonRequirement.REASON_REQUIREMENT_UNSET;
              onUpdate();
            }}
          />
        </Field>

        <Field
          label="Escalation"
          hint="Bring in backup reviewers when nobody can review the step or when it waits too long."
        >
          <div className="flex flex-col gap-3">
            <Switch
              size="xs"
              label="Escalate to backup reviewers"
              checked={!!step.escalation}
              onChange={(event) => {
                step.escalation = event.target.checked
                  ? AccessP.Policy_Spec_Rule_Action_Review_Step_Escalation.create()
                  : undefined;
                onUpdate();
              }}
            />
            {step.escalation && (
              <div className="flex flex-col gap-3 rounded-lg border border-dashed border-slate-200 p-3">
                <div className="max-w-sm">
                  <DurationPicker
                    title="Escalate after"
                    description="Leave empty to escalate only when nobody can review the step."
                    placeholder="Only when nobody can review"
                    value={step.escalation.after}
                    onChange={(value) => {
                      if (step.escalation) step.escalation.after = value;
                      onUpdate();
                    }}
                  />
                </div>
                <ReviewersEdit
                  reviewers={step.escalation.reviewers}
                  allowCatalogOwners
                  groupDescription="Members of these Groups can also review once the step escalates."
                  userDescription="These Users can also review once the step escalates."
                  onChange={(reviewers) => {
                    if (step.escalation) step.escalation.reviewers = reviewers;
                    onUpdate();
                  }}
                />
              </div>
            )}
          </div>
        </Field>

        <Field
          label="Present it in Slack, Jira or a webhook"
          hint="Reviewers always find the request in the access portal. Add deliveries so they see it where they already work."
        >
          <SurfacesEdit
            surfaces={step.surfaces}
            isReview
            keyAt={(i) => rows.keyAt("surfaces", i)}
            removeKeyAt={(i) => rows.removeAt("surfaces", i)}
            onUpdate={onUpdate}
            addLabel="Add delivery"
            emptyLabel="Only in the access portal."
          />
        </Field>

        <Field label="Separation of duties" hint="By default, nobody can review a request that they created or that gives them access.">
          <div className="flex flex-col gap-2">
            <Switch
              size="xs"
              label="Let the requester review their own request"
              checked={!!sod?.allowRequesterReview}
              onChange={(event) => {
                step.separationOfDuties = AccessP.Policy_Spec_Rule_SeparationOfDuties.create({
                  ...sod,
                  allowRequesterReview: event.target.checked,
                });
                if (!step.separationOfDuties.allowRequesterReview && !step.separationOfDuties.allowSubjectReview)
                  step.separationOfDuties = undefined;
                onUpdate();
              }}
            />
            <Switch
              size="xs"
              label="Let the subject review a request for their own access"
              checked={!!sod?.allowSubjectReview}
              onChange={(event) => {
                step.separationOfDuties = AccessP.Policy_Spec_Rule_SeparationOfDuties.create({
                  ...sod,
                  allowSubjectReview: event.target.checked,
                });
                if (!step.separationOfDuties.allowRequesterReview && !step.separationOfDuties.allowSubjectReview)
                  step.separationOfDuties = undefined;
                onUpdate();
              }}
            />
          </div>
        </Field>

        {issues.length > 0 && (
          <ul className="flex flex-col gap-0.5 rounded-lg bg-amber-50 px-3 py-2 text-micro font-medium text-amber-800">
            {issues.map((issue) => (
              <li key={issue}>{issue.replace(/^[^:]+: /, "")}</li>
            ))}
          </ul>
        )}
      </div>
    </li>
  );
};

export const StepsEdit = (props: {
  review: AccessP.Policy_Spec_Rule_Action_Review;
  onUpdate: () => void;
}) => {
  const rows = useListKeys();
  const { review, onUpdate } = props;
  const steps = review.steps;

  return (
    <div className="flex flex-col">
      <ol className="flex flex-col">
        {steps.map((step, index) => (
          <StepEdit
            key={rows.keyAt("steps", index)}
            step={step}
            index={index}
            steps={steps}
            onUpdate={onUpdate}
            onMove={(delta) => {
              const target = index + delta;
              if (target < 0 || target >= steps.length) return;
              rows.move("steps", index, target);
              const [moved] = steps.splice(index, 1);
              steps.splice(target, 0, moved);
              onUpdate();
            }}
            onRemove={() => {
              rows.removeAt("steps", index);
              steps.splice(index, 1);
              onUpdate();
            }}
          />
        ))}
      </ol>
      <Button
        variant="default"
        size="xs"
        className="ml-10 w-fit"
        leftSection={<Plus size={13} />}
        onClick={() => {
          steps.push(newReviewStep(nextStepName(steps)));
          onUpdate();
        }}
      >
        Add a step after {steps[steps.length - 1]?.name || "the last one"}
      </Button>
      {steps.length > 1 && (
        <span className="ml-10 mt-2 text-micro leading-4 text-slate-500">
          Steps run in order: a request reaches step 2 only once step 1 is
          cleared, and so on.
        </span>
      )}
    </div>
  );
};
