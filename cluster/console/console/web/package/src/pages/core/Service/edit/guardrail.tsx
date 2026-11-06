import * as CoreP from "@/apis/corev1/corev1";
import { Callout } from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Select, TagsInput, TextInput } from "@mantine/core";
import * as React from "react";
import {
  ChipField,
  ListField,
  RemoveButton,
  Segments,
  SourceField,
  TextField,
  Update,
} from "./fields";

type Pattern = CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern;

type Guardrail =
  | CoreP.Service_Spec_Config_LLM_Plugin_Guardrail
  | CoreP.Service_Spec_Config_MCP_Plugin_Guardrail;

const Action = CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Action;
const PatternType = CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Type;

const LEG_REQUEST = 1;
const LEG_RESPONSE = 2;
const LEG_BOTH = 3;

const LLMScope = CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Scope;
const MCPScope = CoreP.Service_Spec_Config_MCP_Plugin_Guardrail_Scope;

const LLM_SCOPES = [
  { value: LLMScope.CONTENT, label: "Content" },
  { value: LLMScope.INSTRUCTIONS, label: "Instructions" },
  { value: LLMScope.TOOL_DEFINITIONS, label: "Tool definitions" },
  { value: LLMScope.TOOL_RESULTS, label: "Tool results" },
  { value: LLMScope.ALL, label: "All" },
];

const MCP_REQUEST_SCOPES = [
  { value: MCPScope.TOOL_ARGUMENTS, label: "Tool arguments" },
];

const MCP_RESPONSE_SCOPES = [
  { value: MCPScope.TOOL_RESULTS, label: "Tool results" },
  { value: MCPScope.RESOURCE_CONTENTS, label: "Resource contents" },
  { value: MCPScope.PROMPT_MESSAGES, label: "Prompt messages" },
  { value: MCPScope.TOOL_DEFINITIONS, label: "Tool definitions" },
];

const createPattern = (rewritable: boolean) =>
  CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern.create({
    match: { oneofKind: "regex", regex: "" },
    action: rewritable ? Action.REDACT : Action.DENY,
  });

const createMatch = (kind: string): Pattern["match"] => {
  switch (kind) {
    case "type":
      return { oneofKind: "type", type: PatternType.EMAIL };
    case "secrets":
      return {
        oneofKind: "secrets",
        secrets:
          CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Secrets.create(),
      };
    default:
      return { oneofKind: "regex", regex: "" };
  }
};

const ACTIONS = [
  { value: Action.DENY, label: "Deny" },
  { value: Action.REDACT, label: "Redact" },
  { value: Action.STRIP, label: "Strip" },
  { value: Action.REPLACE, label: "Replace" },
];

const PatternRow = (props: {
  pattern: Pattern;
  index: number;
  rewritable: boolean;
  update: Update;
  onRemove: () => void;
}) => {
  const { pattern, index, update } = props;
  const match = pattern.match;
  const action = pattern.action || Action.DENY;
  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-lg border border-slate-200 bg-white p-3">
      <div className="flex min-w-0 flex-wrap items-start gap-2">
        <div className="flex h-9 shrink-0 items-center">
          <Segments
            label={`Pattern ${index + 1} match`}
            hideLabel
            value={match.oneofKind}
            options={[
              { value: "regex", label: "Regex" },
              { value: "type", label: "Built-in" },
              { value: "secrets", label: "Secrets" },
            ]}
            onChange={(next) => {
              pattern.match = createMatch(next);
              update();
            }}
          />
        </div>
        <div className="min-w-0 flex-[2_1_12rem]">
          {match.oneofKind === "regex" && (
            <TextInput
              aria-label={`Pattern ${index + 1} regular expression`}
              classNames={{ input: "font-mono" }}
              placeholder="[0-9]{16}"
              value={match.regex}
              onChange={(event) => {
                match.regex = event.currentTarget.value;
                update();
              }}
            />
          )}
          {match.oneofKind === "type" && (
            <Select
              aria-label={`Pattern ${index + 1} detector`}
              allowDeselect={false}
              data={[
                { value: String(PatternType.EMAIL), label: "Email address" },
                { value: String(PatternType.CREDIT_CARD), label: "Credit card" },
                { value: String(PatternType.IBAN), label: "IBAN" },
                { value: String(PatternType.US_SSN), label: "US SSN" },
              ]}
              value={String(match.type)}
              onChange={(next) => {
                match.type = Number(next);
                update();
              }}
            />
          )}
          {match.oneofKind === "secrets" && (
            <TagsInput
              aria-label={`Pattern ${index + 1} excluded rules`}
              placeholder={
                match.secrets.excludeRules.length
                  ? undefined
                  : "Skip rules (optional)"
              }
              splitChars={[",", " "]}
              acceptValueOnBlur
              value={match.secrets.excludeRules}
              onChange={(next) => {
                match.secrets.excludeRules = next;
                update();
              }}
            />
          )}
        </div>
        <Select
          aria-label={`Pattern ${index + 1} action`}
          className="w-[7.5rem] shrink-0"
          allowDeselect={false}
          data={ACTIONS.filter(
            (item) => props.rewritable || item.value === Action.DENY,
          ).map((item) => ({ value: String(item.value), label: item.label }))}
          value={String(action)}
          onChange={(next) => {
            pattern.action = Number(next);
            pattern.replace =
              pattern.action === Action.REPLACE
                ? CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Pattern_Replace.create(
                    { type: { oneofKind: "value", value: "" } },
                  )
                : undefined;
            update();
          }}
        />
        <RemoveButton
          label={`Remove pattern ${index + 1}`}
          onClick={props.onRemove}
        />
      </div>
      {action === Action.REPLACE && pattern.replace && (
        <SourceField
          label="Replacement"
          holder={pattern.replace}
          update={update}
          placeholder="[REDACTED]"
        />
      )}
    </div>
  );
};

export const GuardrailForm = (props: {
  value: Guardrail;
  kind: "llm" | "mcp";
  update: Update;
}) => {
  const { value, kind, update } = props;
  const rows = useListKeys();
  const leg = value.leg || LEG_REQUEST;
  const inspectsResponse = leg === LEG_RESPONSE || leg === LEG_BOTH;
  const scopes = value.scopes as number[];
  const definitions =
    kind === "llm" ? LLMScope.TOOL_DEFINITIONS : MCPScope.TOOL_DEFINITIONS;
  const all = kind === "llm" ? LLMScope.ALL : MCPScope.ALL;
  const rewritable =
    !inspectsResponse && !scopes.includes(definitions) && !scopes.includes(all);
  const scopeOptions =
    kind === "llm"
      ? leg === LEG_RESPONSE
        ? []
        : LLM_SCOPES
      : [
          ...(leg !== LEG_RESPONSE ? MCP_REQUEST_SCOPES : []),
          ...(inspectsResponse ? MCP_RESPONSE_SCOPES : []),
          { value: MCPScope.ALL, label: "All" },
        ];

  const applyRules = (nextScopes: number[]) => {
    const denyOnly =
      leg === LEG_RESPONSE ||
      leg === LEG_BOTH ||
      nextScopes.includes(definitions) ||
      nextScopes.includes(all);
    if (denyOnly) {
      for (const pattern of value.patterns) {
        pattern.action = Action.DENY;
        pattern.replace = undefined;
      }
    }
  };

  return (
    <>
      <Segments
        label="Inspect"
        value={String(leg)}
        options={[
          { value: String(LEG_REQUEST), label: "Requests" },
          { value: String(LEG_RESPONSE), label: "Responses" },
          { value: String(LEG_BOTH), label: "Both" },
        ]}
        onChange={(next) => {
          value.leg = Number(next);
          const nextLeg = value.leg;
          const allowed =
            kind === "llm"
              ? nextLeg === LEG_RESPONSE
                ? []
                : LLM_SCOPES.map((item) => item.value as number)
              : [
                  ...(nextLeg !== LEG_RESPONSE
                    ? MCP_REQUEST_SCOPES.map((item) => item.value as number)
                    : []),
                  ...(nextLeg !== LEG_REQUEST
                    ? MCP_RESPONSE_SCOPES.map((item) => item.value as number)
                    : []),
                  MCPScope.ALL as number,
                ];
          (value as { scopes: number[] }).scopes = scopes.filter((item) =>
            allowed.includes(item),
          );
          if (nextLeg !== LEG_REQUEST) {
            for (const pattern of value.patterns) {
              pattern.action = Action.DENY;
              pattern.replace = undefined;
            }
          }
          update();
        }}
      />
      {scopeOptions.length > 0 && (
        <ChipField
          label="Scopes"
          description="Parts of the exchange to inspect. Without a selection, the default part of each leg is inspected."
          options={scopeOptions}
          selected={scopes}
          onChange={(next) => {
            (value as { scopes: number[] }).scopes = next;
            applyRules(next);
            update();
          }}
        />
      )}
      <ListField
        label="Patterns"
        description="Every pattern is evaluated, and a single Deny match rejects the content."
        addLabel="Add pattern"
        count={value.patterns.length}
        onAdd={() => {
          value.patterns.push(createPattern(rewritable));
          update();
        }}
      >
        {value.patterns.map((pattern, index) => (
          <PatternRow
            key={rows.keyAt("patterns", index)}
            pattern={pattern}
            index={index}
            rewritable={rewritable}
            update={update}
            onRemove={() => {
              rows.removeAt("patterns", index);
              value.patterns.splice(index, 1);
              update();
            }}
          />
        ))}
      </ListField>
      {!rewritable && (
        <Callout tone="info" title="Only the Deny action is available here">
          A Guardrail that inspects responses or tool definitions cannot rewrite
          what it matches.
        </Callout>
      )}
      <TextField
        of={value}
        name="denyMessage"
        update={update}
        label="Deny message"
        description="Returned to the client, so keep the detection logic out of it."
        placeholder="This request was blocked by policy."
      />
    </>
  );
};

export const guardrailSummary = (value: Guardrail) => {
  const leg = value.leg || LEG_REQUEST;
  const count = value.patterns.length;
  return `${leg === LEG_BOTH ? "Requests and responses" : leg === LEG_RESPONSE ? "Responses" : "Requests"}, ${count} ${count === 1 ? "pattern" : "patterns"}`;
};

