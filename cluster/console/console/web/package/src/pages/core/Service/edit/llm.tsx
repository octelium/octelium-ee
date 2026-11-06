import * as CoreP from "@/apis/corev1/corev1";
import {
  Callout,
  ChoiceGroup,
  FormSection,
  Grid,
  isHTTPURL,
  KeyValueList,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Select, Textarea, TextInput } from "@mantine/core";
import { CELEditor, OPAEditor } from "@/components/Condition/Editor";
import {
  ArrowRightLeft,
  BookText,
  Brain,
  CircleGauge,
  Cpu,
  DatabaseZap,
  Gauge,
  KeyRound,
  Layers,
  ShieldCheck,
  Signpost,
  Sparkles,
  Wrench,
} from "lucide-react";
import * as React from "react";
import { AuthEditor } from "./auth";
import {
  AddButton,
  DurationField,
  EnumField,
  FieldText,
  ItemCard,
  joinSummary,
  ListField,
  NumberField,
  OptionList,
  OptionRow,
  plural,
  RemoveButton,
  Scope,
  Segments,
  SourceField,
  StringList,
  TextField,
  Toggle,
  Update,
} from "./fields";
import { guardrailSummary, GuardrailForm } from "./guardrail";
import {
  CORSRow,
  headerVisibilitySummary,
  HeadersRow,
  PathRow,
  VisibilityHeaders,
  VisibilityRow,
} from "./headers";
import { TransportCard } from "./http";
import {
  PluginDefinition,
  PluginsSection,
  RateLimitKey,
  SHARED_PLUGINS,
} from "./plugins";

type LLM = CoreP.Service_Spec_Config_LLM;
type Embedding = CoreP.Service_Spec_Config_LLM_Embedding;
type Reasoning = CoreP.Service_Spec_Config_LLM_Reasoning;
type Tools = CoreP.Service_Spec_Config_LLM_Plugin_Tools;

const Protocol = CoreP.Service_Spec_Config_LLM_Protocol;
const Level = CoreP.Service_Spec_Config_LLM_Reasoning_Level;
const Decision = CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Decision;
const SystemMode = CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System_Mode;
const MessageRole = CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Role;
const MessagePosition =
  CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Position;
const MessageSelector =
  CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message_Selector;
const ToolChoice = CoreP.Service_Spec_Config_LLM_Plugin_Tools_Choice;
const ToolPosition = CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool_Position;
const TokenScope = CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit_Scope;

const PROTOCOLS = [
  { value: Protocol.OPENAI, label: "OpenAI" },
  { value: Protocol.ANTHROPIC, label: "Anthropic" },
  { value: Protocol.GEMINI, label: "Gemini" },
  { value: Protocol.BEDROCK, label: "Bedrock" },
];

const protocolName = (protocol: number) =>
  PROTOCOLS.find((item) => item.value === protocol)?.label ?? "OpenAI";

const LEVELS = [
  { value: Level.NONE, label: "None" },
  { value: Level.MINIMAL, label: "Minimal" },
  { value: Level.LOW, label: "Low" },
  { value: Level.MEDIUM, label: "Medium" },
  { value: Level.HIGH, label: "High" },
  { value: Level.XHIGH, label: "Extra high" },
  { value: Level.MAX, label: "Max" },
];

const levelName = (level: number) =>
  LEVELS.find((item) => item.value === level)?.label;

const AUTH_NOTES: Partial<Record<number, string>> = {
  [Protocol.ANTHROPIC]:
    "Anthropic reads the API key from the x-api-key header, so use a custom header.",
  [Protocol.GEMINI]:
    "Gemini reads the API key from the x-goog-api-key header, so use a custom header.",
  [Protocol.BEDROCK]: "Bedrock authenticates with AWS SigV4.",
};

const createReasoning = () =>
  CoreP.Service_Spec_Config_LLM_Reasoning.create({
    type: { oneofKind: "level", level: Level.MEDIUM },
  });

const createModel = () =>
  CoreP.Service_Spec_Config_LLM_Model.create({
    type: { oneofKind: "value", value: "" },
  });

const reasoningSummary = (value: Reasoning) => {
  const type = value.type;
  switch (type.oneofKind) {
    case "level":
      return levelName(type.level);
    case "tokenBudget":
      return `${type.tokenBudget.toLocaleString()} tokens`;
    case "effort":
      return `Effort ${type.effort}`;
    case "eval":
      return "CEL expression";
    case "opa":
      return "Rego policy";
    default:
      return undefined;
  }
};

const ReasoningEditor = (props: { value: Reasoning; update: Update }) => {
  const { value, update } = props;
  const type = value.type;
  const memory = React.useRef<Record<string, Reasoning["type"]>>({});
  return (
    <>
      <Segments
        label="Set by"
        value={type.oneofKind}
        options={[
          { value: "level", label: "Level" },
          { value: "tokenBudget", label: "Budget" },
          { value: "effort", label: "Effort" },
          { value: "eval", label: "CEL" },
          { value: "opa", label: "Rego" },
        ]}
        onChange={(next) => {
          if (type.oneofKind) memory.current[type.oneofKind] = type;
          value.type =
            memory.current[next] ??
            (next === "tokenBudget"
              ? { oneofKind: "tokenBudget", tokenBudget: 0 }
              : next === "effort"
                ? { oneofKind: "effort", effort: "" }
                : next === "eval"
                  ? { oneofKind: "eval", eval: "" }
                  : next === "opa"
                    ? { oneofKind: "opa", opa: "" }
                    : { oneofKind: "level", level: Level.MEDIUM });
          update();
        }}
      />
      {type.oneofKind === "level" && (
        <Select
          label="Level"
          description="Portable across models. Each model gets the closest level that does not exceed it."
          allowDeselect={false}
          data={LEVELS.map((item) => ({
            value: String(item.value),
            label: item.label,
          }))}
          value={String(type.level)}
          onChange={(next) => {
            type.level = Number(next);
            update();
          }}
        />
      )}
      {type.oneofKind === "tokenBudget" && (
        <NumberField
          of={type}
          name="tokenBudget"
          update={update}
          label="Token budget"
          description="Only served to models that accept a numeric budget. Use the None level to disable reasoning."
          placeholder="4096"
          required
          max={1048576}
          thousands
        />
      )}
      {type.oneofKind === "effort" && (
        <TextField
          of={type}
          name="effort"
          update={update}
          label="Provider effort"
          description="Sent to the model as it is, for values that no level names."
          placeholder="high"
          required
        />
      )}
      {type.oneofKind === "eval" && (
        <div className="flex min-w-0 flex-col gap-1">
          <FieldText
            label="CEL expression"
            description="Returns a level, a number of tokens or an effort. An empty result keeps the client's setting."
          />
          <CELEditor
            label="Reasoning CEL"
            exp={type.eval}
            onChange={(next) => {
              type.eval = next;
              update();
            }}
          />
        </div>
      )}
      {type.oneofKind === "opa" && (
        <OPAEditor
          label="Reasoning Rego"
          exp={type.opa}
          onChange={(next) => {
            type.opa = next;
            update();
          }}
        />
      )}
    </>
  );
};

const embeddingAvailableOnUpstream = (protocol: number) =>
  protocol !== Protocol.ANTHROPIC && protocol !== Protocol.BEDROCK;

const EmbeddingEditor = (props: {
  value: Embedding;
  protocol: number;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.source?.type;
  const memory = React.useRef<
    CoreP.Service_Spec_Config_LLM_Embedding_Source | undefined
  >(undefined);
  const onUpstream = embeddingAvailableOnUpstream(props.protocol);
  return (
    <>
      <Grid>
        <TextField
          of={value}
          name="model"
          update={update}
          label="Model"
          placeholder="text-embedding-3-small"
          required
        />
        <NumberField
          of={value}
          name="dimensions"
          update={update}
          label="Dimensions"
          description="Smaller vectors are cheaper to store and search."
          placeholder="Model default"
          max={4096}
        />
      </Grid>
      <ChoiceGroup
        label="Generated by"
        value={type?.oneofKind}
        options={[
          {
            value: "currentUpstream",
            label: "This Service's upstream",
            hint: onUpstream
              ? "Uses its credentials and protocol"
              : `${protocolName(props.protocol)} serves no embeddings`,
            disabled: !onUpstream,
          },
          {
            value: "upstream",
            label: "A separate API",
            hint: "An OpenAI or Gemini compatible URL",
          },
        ]}
        onChange={(next) => {
          if (value.source?.type.oneofKind === "upstream") {
            memory.current = value.source;
          }
          value.source =
            next === "upstream"
              ? (memory.current ??
                CoreP.Service_Spec_Config_LLM_Embedding_Source.create({
                  type: {
                    oneofKind: "upstream",
                    upstream:
                      CoreP.Service_Spec_Config_LLM_Embedding_Source_Upstream.create(),
                  },
                }))
              : CoreP.Service_Spec_Config_LLM_Embedding_Source.create({
                  type: { oneofKind: "currentUpstream", currentUpstream: true },
                });
          update();
        }}
      />
      {type?.oneofKind === "upstream" && (
        <>
          <Grid>
            <TextField
              of={type.upstream}
              name="url"
              update={update}
              label="URL"
              placeholder="https://api.openai.com/v1"
              required
              error={(url) =>
                isHTTPURL(url) ? undefined : "Enter a full http or https URL"
              }
            />
            <EnumField
              of={type.upstream}
              name="protocol"
              update={update}
              label="Protocol"
              placeholder="OpenAI"
              options={PROTOCOLS.filter(
                (item) =>
                  item.value === Protocol.OPENAI ||
                  item.value === Protocol.GEMINI,
              )}
            />
          </Grid>
          <div className="flex min-w-0 flex-col gap-1.5">
            <FieldText label="Authentication" />
            <AuthEditor holder={type.upstream} update={update} noSigV4 />
          </div>
        </>
      )}
    </>
  );
};

const EmbeddingRow = (props: {
  holder: { embedding?: Embedding };
  protocol: number;
  title: string;
  description: string;
  update: Update;
}) => {
  const { holder, update } = props;
  const value = holder.embedding;
  return (
    <OptionRow
      icon={Layers}
      title={props.title}
      description={props.description}
      summary={
        value &&
        joinSummary(
          value.model || "No model",
          value.source?.type.oneofKind === "upstream"
            ? "Separate API"
            : "This upstream",
        )
      }
      isSet={!!value}
      onAdd={() => {
        holder.embedding = CoreP.Service_Spec_Config_LLM_Embedding.create({
          source: CoreP.Service_Spec_Config_LLM_Embedding_Source.create({
            type: embeddingAvailableOnUpstream(props.protocol)
              ? { oneofKind: "currentUpstream", currentUpstream: true }
              : {
                  oneofKind: "upstream",
                  upstream:
                    CoreP.Service_Spec_Config_LLM_Embedding_Source_Upstream.create(),
                },
          }),
        });
        update();
      }}
      onRemove={() => {
        holder.embedding = undefined;
        update();
      }}
    >
      {value && (
        <EmbeddingEditor
          value={value}
          protocol={props.protocol}
          update={update}
        />
      )}
    </OptionRow>
  );
};

const createPromptContent = () =>
  CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Content.create({
    type: { oneofKind: "value", value: "" },
  });

const SystemPrompt = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System;
  update: Update;
}) => {
  const { value, update } = props;
  const noContent = value.mode === SystemMode.STRIP;
  const content = value.content ?? createPromptContent();
  const commit = () => {
    const empty =
      content.type.oneofKind === "value" && content.type.value === "";
    value.content =
      empty && value.mode === SystemMode.REJECT ? undefined : content;
    update();
  };
  return (
    <>
      <EnumField
        of={value}
        name="mode"
        update={update}
        label="How to apply it"
        placeholder="Prepend"
        clearable={false}
        description={
          value.mode === SystemMode.REJECT
            ? "Reject requests that carry their own instructions."
            : value.mode === SystemMode.STRIP
              ? "Remove the instructions of the client and add nothing."
              : value.mode === SystemMode.REPLACE
                ? "Discard the instructions of the client."
                : "Keep the instructions of the client."
        }
        options={[
          { value: SystemMode.PREPEND, label: "Prepend" },
          { value: SystemMode.APPEND, label: "Append" },
          { value: SystemMode.REPLACE, label: "Replace" },
          { value: SystemMode.STRIP, label: "Strip" },
          { value: SystemMode.REJECT, label: "Reject" },
        ]}
      />
      {!noContent && (
        <SourceField
          label="Instructions"
          holder={content}
          update={commit}
          placeholder="You are a helpful assistant."
          multiline
        />
      )}
    </>
  );
};

const MessagePrompt = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message;
  update: Update;
}) => {
  const { value, update } = props;
  const isNew =
    value.position === MessagePosition.NEW_BEFORE ||
    value.position === MessagePosition.NEW_AFTER;
  const content = value.content ?? createPromptContent();
  return (
    <>
      <Grid cols={3}>
        <EnumField
          of={value}
          name="role"
          update={update}
          label="Role"
          clearable={false}
          placeholder="Select"
          options={[
            { value: MessageRole.USER, label: "User" },
            { value: MessageRole.ASSISTANT, label: "Assistant" },
          ]}
        />
        <EnumField
          of={value}
          name="position"
          update={update}
          label="Position"
          placeholder="Append"
          clearable={false}
          options={[
            { value: MessagePosition.PREPEND, label: "Prepend to the message" },
            { value: MessagePosition.APPEND, label: "Append to the message" },
            { value: MessagePosition.NEW_BEFORE, label: "New message before" },
            { value: MessagePosition.NEW_AFTER, label: "New message after" },
          ]}
        />
        <EnumField
          of={value}
          name="selector"
          update={update}
          label="Message"
          placeholder="Last"
          clearable={false}
          options={[
            { value: MessageSelector.LAST, label: "Last" },
            { value: MessageSelector.FIRST, label: "First" },
            ...(isNew
              ? []
              : [{ value: MessageSelector.ALL, label: "All" }]),
          ]}
        />
      </Grid>
      <SourceField
        label="Content"
        holder={content}
        update={() => {
          value.content = content;
          update();
        }}
        placeholder="Answer only from the provided context."
        multiline
      />
    </>
  );
};

const PromptForm = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_Prompt;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.type;
  return (
    <>
      <Segments
        label="Target"
        value={type.oneofKind}
        options={[
          { value: "system", label: "System instructions" },
          { value: "message", label: "Conversation message" },
        ]}
        onChange={(next) => {
          value.type =
            next === "message"
              ? {
                  oneofKind: "message",
                  message:
                    CoreP.Service_Spec_Config_LLM_Plugin_Prompt_Message.create({
                      role: MessageRole.USER,
                      position: MessagePosition.APPEND,
                      selector: MessageSelector.LAST,
                      content: createPromptContent(),
                    }),
                }
              : {
                  oneofKind: "system",
                  system:
                    CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System.create({
                      mode: SystemMode.PREPEND,
                      content: createPromptContent(),
                    }),
                };
          update();
        }}
      />
      {type.oneofKind === "system" && (
        <SystemPrompt value={type.system} update={update} />
      )}
      {type.oneofKind === "message" && (
        <MessagePrompt value={type.message} update={update} />
      )}
    </>
  );
};

const FilterRow = (props: {
  filter: Tools["filters"][number];
  index: number;
  update: Update;
  onRemove: () => void;
}) => {
  const { filter, index, update } = props;
  const match = filter.match;
  const replace = filter.decision === Decision.REPLACE;
  return (
    <div className="flex min-w-0 flex-col gap-3 rounded-lg border border-slate-200 bg-white p-3">
      <div className="flex min-w-0 flex-wrap items-start gap-2">
        <div className="flex h-9 shrink-0 items-center">
          <Segments
            label={`Filter ${index + 1} match`}
            hideLabel
            value={match.oneofKind}
            options={[
              { value: "name", label: "Name" },
              { value: "type", label: "Type" },
            ]}
            onChange={(next) => {
              filter.match =
                next === "type"
                  ? { oneofKind: "type", type: "" }
                  : { oneofKind: "name", name: "" };
              update();
            }}
          />
        </div>
        <TextInput
          aria-label={`Filter ${index + 1} pattern`}
          className="min-w-0 flex-[2_1_10rem]"
          classNames={{ input: "font-mono" }}
          placeholder={match.oneofKind === "type" ? "web_search*" : "read_*"}
          value={
            match.oneofKind === "type"
              ? match.type
              : match.oneofKind === "name"
                ? match.name
                : ""
          }
          onChange={(event) => {
            const value = event.currentTarget.value;
            filter.match =
              match.oneofKind === "type"
                ? { oneofKind: "type", type: value }
                : { oneofKind: "name", name: value };
            update();
          }}
        />
        <Select
          aria-label={`Filter ${index + 1} decision`}
          className="w-[7.5rem] shrink-0"
          allowDeselect={false}
          data={[
            { value: String(Decision.ALLOW), label: "Allow" },
            { value: String(Decision.REMOVE), label: "Remove" },
            { value: String(Decision.DENY), label: "Deny" },
            { value: String(Decision.REPLACE), label: "Replace" },
          ]}
          value={String(filter.decision || Decision.REMOVE)}
          onChange={(next) => {
            filter.decision = Number(next);
            filter.replace =
              filter.decision === Decision.REPLACE
                ? CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter_Replace.create(
                    { type: { oneofKind: "value", value: "" } },
                  )
                : undefined;
            update();
          }}
        />
        <RemoveButton
          label={`Remove filter ${index + 1}`}
          onClick={props.onRemove}
        />
      </div>
      {replace && filter.replace && (
        <SourceField
          label="Replacement tool"
          description="One entry of the tools array, as a JSON object."
          holder={filter.replace}
          update={update}
          placeholder={'{ "name": "lookup" }'}
          multiline
          json
        />
      )}
    </div>
  );
};

const ToolsForm = (props: { value: Tools; update: Update }) => {
  const { value, update } = props;
  const rows = useListKeys();
  const denies = value.filters.some(
    (item) => item.decision === Decision.DENY,
  );
  return (
    <>
      <EnumField
        of={value}
        name="choice"
        update={update}
        label="Tool choice"
        description="How the tool choice of the client is treated."
        placeholder="Keep it"
        options={[
          { value: ToolChoice.PRESERVE, label: "Keep it" },
          { value: ToolChoice.NONE, label: "Force none" },
          { value: ToolChoice.AUTO, label: "Downgrade forced choices to auto" },
        ]}
      />
      <ListField
        label="Filters"
        description="The first matching filter decides. Unmatched provider-hosted tools are removed once a filter exists."
        addLabel="Add filter"
        count={value.filters.length}
        onAdd={() => {
          value.filters.push(
            CoreP.Service_Spec_Config_LLM_Plugin_Tools_Filter.create({
              match: { oneofKind: "name", name: "" },
              decision: Decision.REMOVE,
            }),
          );
          update();
        }}
      >
        {value.filters.map((filter, index) => (
          <FilterRow
            key={rows.keyAt("filters", index)}
            filter={filter}
            index={index}
            update={update}
            onRemove={() => {
              rows.removeAt("filters", index);
              value.filters.splice(index, 1);
              update();
            }}
          />
        ))}
      </ListField>
      {denies && (
        <TextField
          of={value}
          name="denyMessage"
          update={update}
          label="Deny message"
          description="Returned to the client, so keep the policy itself out of it."
          placeholder="This tool is not allowed."
        />
      )}
      <ListField
        label="Injected tools"
        description="Added by the Service. Filters do not apply to them."
        addLabel="Add tool"
        count={value.tools.length}
        onAdd={() => {
          value.tools.push(
            CoreP.Service_Spec_Config_LLM_Plugin_Tools_Tool.create({
              type: { oneofKind: "value", value: "" },
              position: ToolPosition.APPEND,
            }),
          );
          update();
        }}
      >
        {value.tools.map((tool, index) => (
          <div
            key={rows.keyAt("tools", index)}
            className="flex min-w-0 flex-col gap-3 rounded-lg border border-slate-200 bg-white p-3"
          >
            <div className="flex items-center justify-between gap-3">
              <Segments
                label={`Tool ${index + 1} position`}
                hideLabel
                value={
                  tool.position === ToolPosition.PREPEND ? "prepend" : "append"
                }
                options={[
                  { value: "append", label: "After client tools" },
                  { value: "prepend", label: "Before client tools" },
                ]}
                onChange={(next) => {
                  tool.position =
                    next === "prepend"
                      ? ToolPosition.PREPEND
                      : ToolPosition.APPEND;
                  update();
                }}
              />
              <RemoveButton
                label={`Remove tool ${index + 1}`}
                onClick={() => {
                  rows.removeAt("tools", index);
                  value.tools.splice(index, 1);
                  update();
                }}
              />
            </div>
            <SourceField
              label="Definition"
              description="One entry of the tools array, as a JSON object."
              holder={tool}
              update={update}
              placeholder={'{ "name": "lookup" }'}
              multiline
              json
            />
          </div>
        ))}
      </ListField>
    </>
  );
};

const TokenRateLimitForm = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <Grid cols={3}>
        <NumberField
          of={value}
          name="limit"
          update={update}
          label="Token limit"
          placeholder="100,000"
          required
          max={1073741824}
          thousands
        />
        <DurationField
          of={value}
          name="window"
          update={update}
          label="Window"
          required
        />
        <EnumField
          of={value}
          name="scope"
          update={update}
          label="Count"
          placeholder="Total tokens"
          clearable={false}
          options={[
            { value: TokenScope.TOTAL, label: "Total tokens" },
            { value: TokenScope.INPUT, label: "Input tokens" },
            { value: TokenScope.OUTPUT, label: "Output tokens" },
          ]}
        />
      </Grid>
      <RateLimitKey holder={value} update={update} />
      {value.scope !== TokenScope.INPUT && (
        <NumberField
          of={value}
          name="defaultOutputTokens"
          update={update}
          label="Reserved output tokens"
          description="Reserved for requests that declare no output limit. Without it their output cannot be bounded in advance."
          placeholder="None"
          max={1073741824}
          thousands
        />
      )}
      <TextField
        of={value}
        name="denyMessage"
        update={update}
        label="Deny message"
        placeholder="Token quota exceeded."
      />
      <KeyValueList
        label="Response headers"
        description="Set on the 429 response."
        items={value.headers}
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_KeyValue.create()
        }
        addLabel="Add header"
        keyPlaceholder="Retry-After"
        valuePlaceholder="60"
        onUpdate={update}
      />
    </>
  );
};

const CacheScope = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.scope?.type;
  const kind =
    type?.oneofKind === "perSession" ||
    type?.oneofKind === "shared" ||
    type?.oneofKind === "eval"
      ? type.oneofKind
      : "perUser";
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Segments
        label="Share cached responses"
        value={kind}
        options={[
          { value: "perUser", label: "Per User" },
          { value: "perSession", label: "Per Session" },
          { value: "eval", label: "CEL key" },
          { value: "shared", label: "Everyone" },
        ]}
        onChange={(next) => {
          value.scope =
            CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache_Scope.create({
              type:
                next === "eval"
                  ? { oneofKind: "eval", eval: "" }
                  : next === "shared"
                    ? { oneofKind: "shared", shared: true }
                    : next === "perSession"
                      ? { oneofKind: "perSession", perSession: true }
                      : { oneofKind: "perUser", perUser: true },
            });
          update();
        }}
      />
      {type?.oneofKind === "eval" && (
        <TextField
          of={type}
          name="eval"
          update={update}
          label="CEL expression"
          description="An empty result rejects the request."
          placeholder="ctx.user.spec.attrs.tenant"
          required
          mono
        />
      )}
      {kind === "shared" && (
        <Callout title="Every authorized caller can get any cached answer">
          A response generated from one caller's prompt is served to another
          caller who asks the same thing.
        </Callout>
      )}
    </div>
  );
};

const SemanticCacheForm = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache;
  protocol: number;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <Grid cols={3}>
        <NumberField
          of={value}
          name="minSimilarity"
          update={update}
          label="Minimum similarity"
          placeholder="Default"
          max={1}
          step={0.01}
          decimals={3}
        />
        <DurationField of={value} name="ttl" update={update} label="TTL" />
        <NumberField
          of={value}
          name="maxSize"
          update={update}
          label="Largest response"
          suffix=" bytes"
          max={1048576}
          thousands
        />
      </Grid>
      <CacheScope value={value} update={update} />
      <ToggleList>
        <Toggle
          of={value}
          name="useXCacheHeader"
          update={update}
          label="Set the X-Cache header"
          description="Tell clients whether the response came from the cache."
        />
      </ToggleList>
      <OptionList>
        <EmbeddingRow
          holder={value}
          protocol={props.protocol}
          title="Embedding override"
          description="Uses the embedding of this Configuration when not set."
          update={update}
        />
      </OptionList>
    </>
  );
};

const SemanticRouterForm = (props: {
  value: CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter;
  protocol: number;
  update: Update;
}) => {
  const { value, update } = props;
  const rows = useListKeys();
  return (
    <>
      <Grid>
        <NumberField
          of={value}
          name="minSimilarity"
          update={update}
          label="Minimum similarity"
          description="For the routes that set none of their own."
          placeholder="Default"
          max={1}
          step={0.01}
          decimals={3}
        />
        <TextField
          of={value}
          name="fallbackModel"
          update={update}
          label="Fallback model"
          description="Used when no route matches. Defaults to the requested model."
          placeholder="gpt-5-mini"
        />
      </Grid>
      <div className="flex min-w-0 flex-col gap-2">
        <div className="flex items-start justify-between gap-3">
          <FieldText
            label="Routes"
            description="Each route needs a description or at least one example."
          />
          <AddButton
            label="Add route"
            onClick={() => {
              value.routes.push(
                CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter_Route.create(),
              );
              update();
            }}
          />
        </div>
        {value.routes.map((route, index) => (
          <ItemCard
            key={rows.keyAt("routes", index)}
            title={route.name || `Route ${index + 1}`}
            subtitle={joinSummary(
              route.model,
              route.examples.length > 0 &&
                plural(route.examples.length, "example"),
            )}
            defaultOpen={!route.name}
            removeLabel={`Remove route ${index + 1}`}
            onRemove={() => {
              rows.removeAt("routes", index);
              value.routes.splice(index, 1);
              update();
            }}
          >
            <Grid cols={3}>
              <TextField
                of={route}
                name="name"
                update={update}
                label="Name"
                placeholder="coding"
                required
              />
              <TextField
                of={route}
                name="model"
                update={update}
                label="Model"
                placeholder="gpt-5"
                required
              />
              <NumberField
                of={route}
                name="minSimilarity"
                update={update}
                label="Minimum similarity"
                placeholder="Plugin default"
                max={1}
                step={0.01}
                decimals={3}
              />
            </Grid>
            <Textarea
              label="Description"
              placeholder="Software engineering and code review questions"
              autosize
              minRows={2}
              value={route.description}
              onChange={(event) => {
                route.description = event.currentTarget.value;
                update();
              }}
            />
            <StringList
              label="Examples"
              description="Several short examples define a route better than a long description."
              values={route.examples}
              update={update}
              addLabel="Add example"
              itemLabel="Example"
              placeholder="Why is this Go program deadlocking?"
            />
          </ItemCard>
        ))}
      </div>
      <OptionList>
        <EmbeddingRow
          holder={value}
          protocol={props.protocol}
          title="Embedding override"
          description="Uses the embedding of this Configuration when not set."
          update={update}
        />
      </OptionList>
    </>
  );
};

const LLM_PLUGINS: PluginDefinition[] = [
  {
    type: "guardrail",
    label: "Guardrail",
    description: "Inspect prompts and completions",
    icon: ShieldCheck,
    group: "Content safety",
    phased: false,
    summary: guardrailSummary,
    create: () => ({
      oneofKind: "guardrail",
      guardrail: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail.create({
        leg: CoreP.Service_Spec_Config_LLM_Plugin_Guardrail_Leg.REQUEST,
      }),
    }),
  },
  {
    type: "tools",
    label: "Tools",
    description: "Filter, replace or add the tools a model can call",
    icon: Wrench,
    group: "Content safety",
    phased: false,
    summary: (value) =>
      joinSummary(
        value.filters.length > 0 && plural(value.filters.length, "filter"),
        value.tools.length > 0 && `${value.tools.length} injected`,
      ),
    create: () => ({
      oneofKind: "tools",
      tools: CoreP.Service_Spec_Config_LLM_Plugin_Tools.create(),
    }),
  },
  {
    type: "prompt",
    label: "Prompt",
    description: "Control the system instructions and messages",
    icon: BookText,
    group: "Inference",
    phased: false,
    summary: (value) =>
      value.type.oneofKind === "message" ? "Message" : "System instructions",
    create: () => ({
      oneofKind: "prompt",
      prompt: CoreP.Service_Spec_Config_LLM_Plugin_Prompt.create({
        type: {
          oneofKind: "system",
          system: CoreP.Service_Spec_Config_LLM_Plugin_Prompt_System.create({
            mode: SystemMode.PREPEND,
            content: createPromptContent(),
          }),
        },
      }),
    }),
  },
  {
    type: "model",
    label: "Model",
    description: "Choose the model per request",
    icon: Cpu,
    group: "Inference",
    phased: false,
    summary: (value) =>
      value.type.oneofKind === "value" ? value.type.value : "Computed",
    create: () => ({ oneofKind: "model", model: createModel() }),
  },
  {
    type: "reasoning",
    label: "Reasoning",
    description: "Set how much a model may think per request",
    icon: Brain,
    group: "Inference",
    phased: false,
    summary: reasoningSummary,
    create: () => ({ oneofKind: "reasoning", reasoning: createReasoning() }),
  },
  {
    type: "semanticRouter",
    label: "Semantic router",
    description: "Pick the model from the meaning of the request",
    icon: Signpost,
    group: "Inference",
    phased: false,
    summary: (value) => plural(value.routes.length, "route"),
    create: () => ({
      oneofKind: "semanticRouter",
      semanticRouter:
        CoreP.Service_Spec_Config_LLM_Plugin_SemanticRouter.create(),
    }),
  },
  {
    type: "semanticCache",
    label: "Semantic cache",
    description: "Reuse responses to requests of the same meaning",
    icon: DatabaseZap,
    group: "Cost and capacity",
    phased: false,
    create: () => ({
      oneofKind: "semanticCache",
      semanticCache:
        CoreP.Service_Spec_Config_LLM_Plugin_SemanticCache.create(),
    }),
  },
  {
    type: "tokenRateLimit",
    label: "Token rate limit",
    description: "Set a token quota per identity",
    icon: CircleGauge,
    group: "Cost and capacity",
    phased: false,
    summary: (value) =>
      value.limit ? `${Number(value.limit).toLocaleString()} tokens` : undefined,
    create: () => ({
      oneofKind: "tokenRateLimit",
      tokenRateLimit: CoreP.Service_Spec_Config_LLM_Plugin_TokenRateLimit.create(
        { scope: TokenScope.TOTAL },
      ),
    }),
  },
  ...SHARED_PLUGINS,
];

const LimitsRow = (props: { value: LLM; update: Update }) => {
  const { value, update } = props;
  const limits = value.limits;
  return (
    <OptionRow
      icon={Gauge}
      title="Limits"
      description="Reject requests above these sizes."
      summary={
        limits &&
        (joinSummary(
          !!limits.maxOutputTokens &&
            `${Number(limits.maxOutputTokens).toLocaleString()} output tokens`,
          !!limits.maxEstimatedInputTokens &&
            `${Number(limits.maxEstimatedInputTokens).toLocaleString()} input tokens`,
          !!limits.maxTools && plural(limits.maxTools, "tool"),
          !!limits.maxRequestBytes &&
            `${limits.maxRequestBytes.toLocaleString()} request bytes`,
        ) ||
          "Defaults")
      }
      isSet={!!limits}
      onAdd={() => {
        value.limits = CoreP.Service_Spec_Config_LLM_Limits.create();
        update();
      }}
      onRemove={() => {
        value.limits = undefined;
        update();
      }}
    >
      {limits && (
        <Grid>
          <NumberField
            of={limits}
            name="maxEstimatedInputTokens"
            update={update}
            label="Largest estimated input"
            suffix=" tokens"
            placeholder="No limit"
            thousands
          />
          <NumberField
            of={limits}
            name="maxOutputTokens"
            update={update}
            label="Largest requested output"
            suffix=" tokens"
            placeholder="No limit"
            thousands
          />
          <NumberField
            of={limits}
            name="maxTools"
            update={update}
            label="Most tools"
            placeholder="No limit"
          />
          <NumberField
            of={limits}
            name="maxToolSchemaBytes"
            update={update}
            label="Largest tool schema"
            suffix=" bytes"
            thousands
          />
          <NumberField
            of={limits}
            name="maxRequestBytes"
            update={update}
            label="Largest request"
            suffix=" bytes"
            max={67108864}
            thousands
          />
          <NumberField
            of={limits}
            name="maxStreamEventBytes"
            update={update}
            label="Largest stream event"
            suffix=" bytes"
            max={4194304}
            thousands
          />
        </Grid>
      )}
    </OptionRow>
  );
};

const TranslationRow = (props: { value: LLM; update: Update }) => {
  const { value, update } = props;
  const translation = value.translation;
  const downstream = value.protocol || Protocol.OPENAI;
  return (
    <OptionRow
      icon={ArrowRightLeft}
      title="Protocol translation"
      description="Serve clients over this protocol while the upstream speaks another."
      summary={
        translation &&
        `${protocolName(downstream)} clients to ${protocolName(translation.upstreamProtocol || downstream)} upstream`
      }
      isSet={!!translation}
      onAdd={() => {
        value.translation = CoreP.Service_Spec_Config_LLM_Translation.create();
        update();
      }}
      onRemove={() => {
        value.translation = undefined;
        update();
      }}
    >
      {translation && (
        <>
          <EnumField
            of={translation}
            name="upstreamProtocol"
            update={update}
            label="Upstream protocol"
            description="Only generation requests are translated. Other operations are rejected."
            placeholder="Same as clients"
            options={PROTOCOLS.filter((item) => item.value !== downstream)}
          />
          {translation.upstreamProtocol === Protocol.ANTHROPIC && (
            <NumberField
              of={translation}
              name="defaultMaxOutputTokens"
              update={update}
              label="Default output limit"
              description="Anthropic requires an output limit, so it fills in for requests that set none."
              suffix=" tokens"
              max={1048576}
              thousands
            />
          )}
        </>
      )}
    </OptionRow>
  );
};

export const LLMForm = (props: {
  value: LLM;
  scope: Scope;
  update: Update;
}) => {
  const { value, scope, update } = props;
  const global = scope === "default";
  const protocol = value.protocol || Protocol.OPENAI;
  const upstreamProtocol = value.translation?.upstreamProtocol || protocol;
  const visibility = value.visibility;
  const model = value.model;
  const reasoning = value.reasoning;

  return (
    <>
      <FormSection
        icon={KeyRound}
        title="Provider"
        description="The inference API and the credentials the Service injects, so that Users never see them."
      >
        {global && (
          <ChoiceGroup
            label="API protocol"
            value={String(protocol)}
            options={[
              {
                value: String(Protocol.OPENAI),
                label: "OpenAI",
                hint: "Also vLLM, Ollama and compatible servers",
              },
              {
                value: String(Protocol.ANTHROPIC),
                label: "Anthropic",
                hint: "Messages API",
              },
              {
                value: String(Protocol.GEMINI),
                label: "Gemini",
                hint: "Developer API",
              },
              {
                value: String(Protocol.BEDROCK),
                label: "Bedrock",
                hint: "AWS Bedrock Runtime",
              },
            ]}
            onChange={(next) => {
              value.protocol = Number(next);
              if (value.translation?.upstreamProtocol === value.protocol) {
                value.translation.upstreamProtocol =
                  Protocol.PROTOCOL_UNSET;
              }
              update();
            }}
          />
        )}
        <div className="flex min-w-0 flex-col gap-1.5">
          <FieldText label="Upstream authentication" />
          <AuthEditor
            holder={value}
            update={update}
            note={AUTH_NOTES[upstreamProtocol]}
          />
        </div>
      </FormSection>

      <OptionList title="Inference defaults">
        <OptionRow
          icon={Sparkles}
          title="Model override"
          description="Replace the model that clients ask for."
          summary={
            model &&
            (model.type.oneofKind === "value"
              ? model.type.value || "No model"
              : model.type.oneofKind === "eval"
                ? "CEL expression"
                : "Rego policy")
          }
          isSet={!!model}
          onAdd={() => {
            value.model = createModel();
            update();
          }}
          onRemove={() => {
            value.model = undefined;
            update();
          }}
        >
          {model && (
            <SourceField
              label="Model"
              description="Policies still see the model that the client asked for."
              holder={model}
              update={update}
              placeholder="gpt-5-mini"
            />
          )}
        </OptionRow>
        <OptionRow
          icon={Brain}
          title="Reasoning"
          description="Decide how much models may think, replacing the client's setting."
          summary={reasoning && reasoningSummary(reasoning)}
          isSet={!!reasoning}
          onAdd={() => {
            value.reasoning = createReasoning();
            update();
          }}
          onRemove={() => {
            value.reasoning = undefined;
            update();
          }}
        >
          {reasoning && <ReasoningEditor value={reasoning} update={update} />}
        </OptionRow>
        {global && <LimitsRow value={value} update={update} />}
        {global && <TranslationRow value={value} update={update} />}
        <EmbeddingRow
          holder={value}
          protocol={protocol}
          title="Embedding"
          description="Needed by the Semantic cache and Semantic router Plugins."
          update={update}
        />
      </OptionList>

      <PluginsSection
        holder={value}
        kind="llm"
        definitions={LLM_PLUGINS}
        scope={scope}
        update={update}
        description="Guardrails, quotas, routing and prompts per Condition. Inference Plugins run in a fixed order."
        renderType={(plugin, updatePlugin) => {
          const type = plugin.type;
          switch (type.oneofKind) {
            case "guardrail":
              return (
                <GuardrailForm
                  value={type.guardrail}
                  kind="llm"
                  update={updatePlugin}
                />
              );
            case "prompt":
              return <PromptForm value={type.prompt} update={updatePlugin} />;
            case "tools":
              return <ToolsForm value={type.tools} update={updatePlugin} />;
            case "model":
              return (
                <SourceField
                  label="Model"
                  description="Overrides the model of this Configuration for the matching requests."
                  holder={type.model}
                  update={updatePlugin}
                  placeholder="gpt-5-mini"
                />
              );
            case "reasoning":
              return (
                <ReasoningEditor value={type.reasoning} update={updatePlugin} />
              );
            case "tokenRateLimit":
              return (
                <TokenRateLimitForm
                  value={type.tokenRateLimit}
                  update={updatePlugin}
                />
              );
            case "semanticCache":
              return (
                <SemanticCacheForm
                  value={type.semanticCache}
                  protocol={protocol}
                  update={updatePlugin}
                />
              );
            case "semanticRouter":
              return (
                <SemanticRouterForm
                  value={type.semanticRouter}
                  protocol={protocol}
                  update={updatePlugin}
                />
              );
            default:
              return null;
          }
        }}
      />

      <OptionList title="Traffic and logging">
        <HeadersRow holder={value} update={update} />
        <PathRow holder={value} update={update} />
        {global && <CORSRow holder={value} update={update} />}
        <VisibilityRow
          isSet={!!visibility}
          summary={
            visibility &&
            (joinSummary(
              visibility.enableRequestBody && "Prompts",
              visibility.enableRequestBodyMap && "Prompts as JSON",
              visibility.enableResponseBody && "Completions",
              visibility.enableResponseBodyMap && "Completions as JSON",
              headerVisibilitySummary(visibility),
            ) ||
              "Nothing extra recorded")
          }
          onAdd={() => {
            value.visibility = CoreP.Service_Spec_Config_LLM_Visibility.create();
            update();
          }}
          onRemove={() => {
            value.visibility = undefined;
            update();
          }}
        >
          {visibility && (
            <>
              <ToggleList>
                <Toggle
                  of={visibility}
                  name="enableRequestBody"
                  update={update}
                  label="Record prompts"
                  description="Prompts routinely contain secrets, personal data and source code."
                />
                <Toggle
                  of={visibility}
                  name="enableRequestBodyMap"
                  update={update}
                  label="Record prompts as JSON"
                />
                <Toggle
                  of={visibility}
                  name="enableResponseBody"
                  update={update}
                  label="Record completions"
                  description="Only for responses that are not streamed."
                />
                <Toggle
                  of={visibility}
                  name="enableResponseBodyMap"
                  update={update}
                  label="Record completions as JSON"
                />
              </ToggleList>
              <VisibilityHeaders value={visibility} update={update} />
            </>
          )}
        </VisibilityRow>
      </OptionList>

      <TransportCard holder={value} scope={scope} update={update} />
    </>
  );
};

