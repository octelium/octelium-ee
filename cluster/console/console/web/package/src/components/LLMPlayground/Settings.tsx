import {
  ActionIcon,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";
import { RotateCcw, ShieldCheck, SlidersHorizontal, X } from "lucide-react";
import * as React from "react";
import { supportsStreaming } from "./protocols";
import { Protocol, PROTOCOL_MODEL_PLACEHOLDER, RequestOptions } from "./types";

const monospace = {
  input: {
    fontFamily: "var(--mantine-font-family-monospace)",
    fontSize: "0.75rem",
  },
};

const Section = (props: { title: string; children: React.ReactNode }) => (
  <section className="flex flex-col gap-3">
    <h3 className="text-micro font-semibold uppercase tracking-[0.08em] text-slate-500">
      {props.title}
    </h3>
    {props.children}
  </section>
);

const Settings = (props: {
  protocol: Protocol;
  options: RequestOptions;
  onOption: <K extends keyof RequestOptions>(
    key: K,
    value: RequestOptions[K],
  ) => void;
  endpoint: string;
  defaultEndpoint: string;
  onEndpoint: (value: string) => void;
  configuredModel: string;
  modelError?: string;
  modelRef: React.RefObject<HTMLInputElement | null>;
  onReset: () => void;
  onClose: () => void;
}) => {
  const {
    protocol,
    options,
    onOption,
    endpoint,
    defaultEndpoint,
    onEndpoint,
    configuredModel,
    modelError,
    modelRef,
    onReset,
    onClose,
  } = props;
  const canStream = supportsStreaming(protocol);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b border-slate-200 pl-4 pr-1.5">
        <SlidersHorizontal size={14} className="shrink-0 text-slate-500" />
        <h2 className="min-w-0 flex-1 truncate text-body font-semibold text-slate-800">
          Request settings
        </h2>
        <Tooltip label="Reset to the defaults" withArrow>
          <ActionIcon
            variant="subtle"
            color="gray"
            aria-label="Reset to the defaults"
            onClick={onReset}
          >
            <RotateCcw size={14} />
          </ActionIcon>
        </Tooltip>
        <Tooltip label="Close" withArrow>
          <ActionIcon
            variant="subtle"
            color="gray"
            aria-label="Close the request settings"
            onClick={onClose}
          >
            <X size={16} />
          </ActionIcon>
        </Tooltip>
      </header>

      <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto overscroll-contain px-4 py-4">
        <Section title="Model">
          <TextInput
            ref={modelRef}
            label="Model"
            required
            placeholder={PROTOCOL_MODEL_PLACEHOLDER[protocol]}
            description={
              configuredModel
                ? `The Service is configured with ${configuredModel}`
                : "Sent as the model of every request"
            }
            error={modelError}
            value={options.model}
            onChange={(event) => onOption("model", event.currentTarget.value)}
          />
          {protocol === "openai" && (
            <div className="flex flex-col gap-1">
              <span className="text-xs font-semibold text-slate-700">API</span>
              <SegmentedControl
                fullWidth
                size="xs"
                aria-label="OpenAI API"
                value={options.apiShape}
                onChange={(value) =>
                  onOption("apiShape", value as RequestOptions["apiShape"])
                }
                data={[
                  { label: "Chat Completions", value: "chat" },
                  { label: "Responses", value: "responses" },
                ]}
              />
            </div>
          )}
        </Section>

        <Section title="Instructions">
          <Textarea
            label="System prompt"
            placeholder="You are a concise assistant."
            autosize
            minRows={3}
            maxRows={10}
            value={options.systemPrompt}
            onChange={(event) =>
              onOption("systemPrompt", event.currentTarget.value)
            }
          />
        </Section>

        <Section title="Sampling">
          <div className="grid grid-cols-2 gap-3">
            <NumberInput
              label="Temperature"
              placeholder="Default"
              min={0}
              max={2}
              step={0.1}
              decimalScale={2}
              value={options.temperature}
              onChange={(value) => onOption("temperature", String(value ?? ""))}
            />
            <NumberInput
              label="Top P"
              placeholder="Default"
              min={0}
              max={1}
              step={0.05}
              decimalScale={2}
              value={options.topP}
              onChange={(value) => onOption("topP", String(value ?? ""))}
            />
          </div>
          <NumberInput
            label="Max output tokens"
            placeholder={protocol === "anthropic" ? "4096" : "Default"}
            min={1}
            allowDecimal={false}
            thousandSeparator=","
            value={options.maxTokens}
            onChange={(value) => onOption("maxTokens", String(value ?? ""))}
          />
        </Section>

        <Section title="Reasoning">
          {protocol === "openai" ? (
            <Select
              label="Reasoning effort"
              placeholder="Model default"
              clearable
              data={["minimal", "low", "medium", "high"]}
              value={options.reasoning || null}
              onChange={(value) => onOption("reasoning", value ?? "")}
            />
          ) : protocol === "bedrock" ? (
            <p className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs leading-5 text-slate-500">
              The Bedrock Converse route does not take reasoning settings.
            </p>
          ) : (
            <NumberInput
              label="Thinking budget"
              description="Tokens the model may spend thinking before it answers"
              placeholder="Off"
              min={0}
              step={1024}
              allowDecimal={false}
              thousandSeparator=","
              value={options.reasoning}
              onChange={(value) => onOption("reasoning", String(value ?? ""))}
            />
          )}
        </Section>

        <Section title="Response">
          <Switch
            label="Stream the response"
            description={
              canStream
                ? "Show the tokens as they arrive"
                : "The Bedrock Converse route answers in one piece"
            }
            disabled={!canStream}
            checked={options.stream && canStream}
            onChange={(event) =>
              onOption("stream", event.currentTarget.checked)
            }
          />
        </Section>

        <Section title="Connection">
          <TextInput
            label="Endpoint"
            description="Base URL of the requests, sent with your session"
            spellCheck={false}
            styles={monospace}
            value={endpoint}
            onChange={(event) => onEndpoint(event.currentTarget.value)}
            rightSection={
              endpoint !== defaultEndpoint ? (
                <Tooltip label="Use the Service endpoint" withArrow>
                  <ActionIcon
                    size="sm"
                    variant="subtle"
                    color="gray"
                    aria-label="Use the Service endpoint"
                    onClick={() => onEndpoint(defaultEndpoint)}
                  >
                    <RotateCcw size={12} />
                  </ActionIcon>
                </Tooltip>
              ) : undefined
            }
          />
        </Section>
      </div>

      <p className="flex shrink-0 items-start gap-2 border-t border-slate-200 bg-slate-50/70 px-4 py-2.5 text-micro leading-4 text-slate-500">
        <ShieldCheck size={13} className="mt-px shrink-0" />
        The Policies, plugins and limits of this Service apply to every request.
      </p>
    </div>
  );
};

export default Settings;
