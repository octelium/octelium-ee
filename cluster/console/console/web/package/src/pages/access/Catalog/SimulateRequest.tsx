import * as AccessPB from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import DurationPicker from "@/components/DurationPicker";
import { ResourceActionButton } from "@/components/ResourceLayout/overview";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { twMerge } from "@/lib/utils";
import { getClientAccess } from "@/utils/client";
import { getResourceRef } from "@/utils/pb";
import { Button, Modal, SegmentedControl } from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangle, FlaskConical, Loader2, X } from "lucide-react";
import * as React from "react";
import { EffectBadge, ToneBadge } from "../common";

const Outcome = AccessPB.RequestPreview_Outcome;
const Result = AccessPB.SimulateRequestResponse_RuleEvaluation_Result;

const OUTCOME_META: Record<
  AccessPB.RequestPreview_Outcome,
  { label: string; tone: "neutral" | "success" | "warning" | "danger" }
> = {
  [Outcome.OUTCOME_UNSET]: { label: "Unknown", tone: "neutral" },
  [Outcome.NOT_REQUESTABLE]: { label: "Not requestable", tone: "danger" },
  [Outcome.INCOMPLETE]: { label: "Incomplete request", tone: "warning" },
  [Outcome.DENIED]: { label: "Denied", tone: "danger" },
  [Outcome.NO_MATCH]: { label: "No matching rule", tone: "danger" },
  [Outcome.AUTO_APPROVED]: { label: "Approved automatically", tone: "success" },
  [Outcome.REVIEW]: { label: "Goes to review", tone: "warning" },
};

const RESULT_LABEL: Record<AccessPB.SimulateRequestResponse_RuleEvaluation_Result, string> = {
  [Result.RESULT_UNSET]: "Not evaluated",
  [Result.MATCHED]: "Matched",
  [Result.NOT_MATCHED]: "Did not match",
  [Result.ERROR]: "Error",
};

const SimulationResult = (props: { result: AccessPB.SimulateRequestResponse }) => {
  const preview = props.result.preview;
  const outcome = OUTCOME_META[preview?.outcome ?? Outcome.OUTCOME_UNSET];
  const scope = preview?.resolvedScope;

  return (
    <div className="flex flex-col gap-3" aria-label="Simulation result">
      <div className="flex flex-wrap items-center gap-2">
        <ToneBadge tone={outcome.tone}>{outcome.label}</ToneBadge>
        {props.result.rule && (
          <span className="text-xs text-slate-600">
            decided by <span className="font-semibold">{props.result.rule.name}</span>
            {props.result.policyRef?.name && ` of ${props.result.policyRef.name}`}
          </span>
        )}
      </div>
      {preview?.message && <p className="text-xs leading-5 text-slate-700">{preview.message}</p>}
      {(preview?.problems.length ?? 0) > 0 && (
        <ul className="flex flex-col gap-1">
          {preview!.problems.map((problem, index) => (
            <li key={index} className="flex items-start gap-1.5 text-xs text-amber-800">
              <AlertTriangle size={12} className="mt-0.5 shrink-0" />
              {problem.message}
            </li>
          ))}
        </ul>
      )}
      {scope && (
        <p className="text-xs text-slate-600">
          {scope.serviceRefs.length} Service{scope.serviceRefs.length === 1 ? "" : "s"} at{" "}
          <span className="font-semibold">
            {scope.accessLevel?.displayName || scope.accessLevel?.name}
          </span>
          {scope.overridesGuardrails && " · overrides guardrails"}
          {scope.totalExcludedService > 0 &&
            ` · ${scope.totalExcludedService} system Service${scope.totalExcludedService === 1 ? "" : "s"} excluded`}
        </p>
      )}
      {props.result.reviewers.length > 0 && (
        <ul className="flex flex-col gap-1 text-xs text-slate-600">
          {props.result.reviewers.map((step) => (
            <li key={step.index}>
              <span className="font-semibold text-slate-800">{step.name}</span>:{" "}
              {step.totalEligibleReviewer} eligible reviewer
              {step.totalEligibleReviewer === 1 ? "" : "s"}, {step.requiredApprovals} approval
              {step.requiredApprovals === 1 ? "" : "s"} needed
              {step.isUnroutable && (
                <span className="font-semibold text-red-700"> · not enough reviewers</span>
              )}
            </li>
          ))}
        </ul>
      )}
      {props.result.rules.length > 0 && (
        <ol className="flex flex-col divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
          {props.result.rules.map((rule, index) => (
            <li
              key={`${rule.policyRef?.uid}-${rule.ruleName}-${index}`}
              className={twMerge(
                "flex flex-wrap items-center gap-2 px-3 py-1.5 text-xs",
                rule.result === Result.MATCHED && "bg-emerald-50/60",
                rule.result === Result.ERROR && "bg-red-50/60",
              )}
            >
              <span className="font-semibold text-slate-800">{rule.ruleName}</span>
              {rule.policyRef?.name && (
                <span className="text-micro text-slate-500">{rule.policyRef.name}</span>
              )}
              <EffectBadge effect={rule.effect} />
              <span className="ml-auto text-micro text-slate-600">
                {RESULT_LABEL[rule.result]}
                {rule.error && `: ${rule.error}`}
              </span>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
};

const SimulateRequest = (props: { item: AccessPB.Catalog }) => {
  const [opened, { open, close }] = useDisclosure(false);
  const levels = props.item.spec?.accessLevels ?? [];
  const [requester, setRequester] = React.useState<MetaP.ObjectReference>();
  const [level, setLevel] = React.useState(levels[0]?.name ?? "");
  const [duration, setDuration] = React.useState<MetaP.Duration>();

  const mutation = useMutation({
    mutationFn: async () =>
      (
        await getClientAccess().simulateRequest(
          AccessPB.SimulateRequestRequest.create({
            requesterRef: requester,
            spec: AccessPB.Request_Spec.create({
              subject: requester
                ? AccessPB.Request_Spec_Subject.create({
                    type: { oneofKind: "userRef", userRef: requester },
                  })
                : undefined,
              duration,
              resource: {
                type: {
                  oneofKind: "catalog",
                  catalog: AccessPB.Request_Spec_Resource_Catalog.create({
                    catalogRef: getResourceRef(props.item),
                    accessLevel: level,
                  }),
                },
              },
            }),
          }),
        )
      ).response,
  });

  return (
    <>
      <ResourceActionButton icon={FlaskConical} onClick={open}>
        Simulate
      </ResourceActionButton>

      <Modal
        opened={opened}
        onClose={close}
        centered
        size="lg"
        withCloseButton={false}
        padding={0}
        overlayProps={{ backgroundOpacity: 0.25, blur: 1 }}
        transitionProps={{ transition: "pop", duration: 250 }}
        styles={{
          content: {
            border: "1px solid var(--color-slate-200)",
            borderRadius: "14px",
            boxShadow: "var(--shadow-modal)",
            overflow: "hidden",
          },
        }}
      >
        <header className="flex items-center justify-between gap-3 border-b border-slate-200 bg-slate-50/70 px-4 py-3.5 sm:px-5">
          <div className="flex min-w-0 items-center gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-900 text-white shadow-sm">
              <FlaskConical size={16} strokeWidth={2.25} />
            </span>
            <div className="min-w-0">
              <h2 className="truncate text-sm font-bold text-slate-900">Simulate a request</h2>
              <p className="mt-0.5 truncate text-micro font-normal text-slate-500">
                Nothing is created. The Catalog and every enabled Policy are evaluated as they are now.
              </p>
            </div>
          </div>
          <Button
            type="button"
            variant="subtle"
            color="gray"
            size="compact-xs"
            leftSection={<X size={12} strokeWidth={2.5} />}
            onClick={close}
          >
            Close
          </Button>
        </header>

        <div className="flex flex-col gap-4 px-4 py-4 sm:px-5">
          <SelectResource
            api="core"
            kind="User"
            label="Requester"
            description="The User who would request this Catalog for themselves."
            defaultValue={requester?.name}
            onChange={(resource) => {
              setRequester(resource ? getResourceRef(resource) : undefined);
              mutation.reset();
            }}
          />
          {levels.length > 1 && (
            <SegmentedControl
              aria-label="Access level"
              value={level}
              onChange={(value) => {
                setLevel(value);
                mutation.reset();
              }}
              data={levels.map((item) => ({
                value: item.name,
                label: item.displayName || item.name,
              }))}
            />
          )}
          <div className="max-w-sm">
            <DurationPicker
              title="Duration"
              description="Leave empty to use the default duration of the access level."
              placeholder="Access level default"
              value={duration}
              onChange={(value) => {
                setDuration(value);
                mutation.reset();
              }}
            />
          </div>

          <div className="flex justify-end">
            <Button
              type="button"
              size="xs"
              color="dark"
              disabled={!requester || mutation.isPending}
              leftSection={
                mutation.isPending ? (
                  <Loader2 className="motion-safe:animate-spin" size={13} />
                ) : (
                  <FlaskConical size={13} strokeWidth={2.25} />
                )
              }
              onClick={() => mutation.mutate()}
            >
              Run simulation
            </Button>
          </div>

          {mutation.isError && (
            <div className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-3 text-xs font-semibold text-red-700">
              {mutation.error instanceof Error
                ? mutation.error.message
                : "The simulation failed."}
            </div>
          )}
          {mutation.data && <SimulationResult result={mutation.data} />}
        </div>
      </Modal>
    </>
  );
};

export default SimulateRequest;
