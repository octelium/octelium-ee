import { Kbd, modKey } from "@/pages/Agent/ui";
import { NumberInput, SegmentedControl, Select, Textarea } from "@mantine/core";
import { CircleAlert, SlidersHorizontal } from "lucide-react";
import * as React from "react";
import { JSONObject } from "./client";
import {
  buildArguments,
  Field,
  FieldValues,
  optionLabel,
  parseArguments,
  toFieldValues,
} from "./schema";
import { monospace } from "./styles";
import { CardHeader } from "./ui";

export type ArgumentsMode = "form" | "json";

const kindLabels: Record<Field["kind"], string> = {
  string: "string",
  number: "number",
  integer: "integer",
  boolean: "boolean",
  enum: "enum",
  json: "JSON",
};

const lenientArguments = (fields: Field[], values: FieldValues) => {
  const ret: JSONObject = {};
  for (const field of fields) {
    try {
      Object.assign(
        ret,
        buildArguments([{ ...field, required: false }], values),
      );
    } catch {
      ret[field.name] = values[field.name];
    }
  }
  return ret;
};

const FieldInput = (props: {
  field: Field;
  value: string;
  onChange: (value: string) => void;
}) => {
  const { field, value, onChange } = props;
  const common = {
    label: (
      <span className="inline-flex min-w-0 flex-wrap items-baseline gap-x-1.5">
        <span className="font-mono">{field.name}</span>
        {field.title && field.title !== field.name && (
          <span className="font-normal text-slate-500">{field.title}</span>
        )}
        <span className="text-micro font-normal text-slate-400">
          {kindLabels[field.kind]}
        </span>
      </span>
    ),
    description: field.description,
    withAsterisk: field.required,
  };

  switch (field.kind) {
    case "enum":
    case "boolean":
      return (
        <Select
          {...common}
          data={
            field.kind === "boolean"
              ? ["true", "false"]
              : (field.options ?? []).map((option) => ({
                  value: option,
                  label: optionLabel(option),
                }))
          }
          clearable={!field.required}
          placeholder={
            field.placeholder ? `Default: ${field.placeholder}` : "Not set"
          }
          value={value || null}
          onChange={(next) => onChange(next ?? "")}
        />
      );
    case "number":
    case "integer":
      return (
        <NumberInput
          {...common}
          allowDecimal={field.kind === "number"}
          min={field.minimum}
          max={field.maximum}
          placeholder={field.placeholder ? `Default: ${field.placeholder}` : ""}
          value={value}
          onChange={(next) => onChange(String(next))}
        />
      );
    case "string":
      return (
        <Textarea
          {...common}
          autosize
          minRows={1}
          maxRows={8}
          styles={{ input: { resize: "none" } }}
          placeholder={field.placeholder ? `Default: ${field.placeholder}` : ""}
          value={value}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      );
    case "json":
      return (
        <Textarea
          {...common}
          autosize
          minRows={2}
          maxRows={12}
          spellCheck={false}
          styles={monospace}
          placeholder={field.placeholder ?? "JSON value"}
          value={value}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
      );
  }
};

const Arguments = (props: {
  fields: Field[];
  values: FieldValues;
  onValues: (values: FieldValues) => void;
  mode: ArgumentsMode;
  onMode: (mode: ArgumentsMode) => void;
  json: string;
  onJSON: (value: string) => void;
  jsonOnly?: boolean;
  emptyLabel: string;
  error?: string;
  onSubmit: () => void;
  children: React.ReactNode;
}) => {
  const { fields, values, onValues, mode, onMode, json, onJSON } = props;
  const [switchError, setSwitchError] = React.useState<string>();
  const effectiveMode = props.jsonOnly ? "json" : mode;
  const error = switchError ?? props.error;

  const changeMode = (next: ArgumentsMode) => {
    setSwitchError(undefined);
    if (next === "json") {
      onJSON(JSON.stringify(lenientArguments(fields, values), null, 2));
      onMode("json");
      return;
    }
    try {
      onValues(toFieldValues(fields, parseArguments(json)));
      onMode("form");
    } catch (err) {
      setSwitchError((err as Error).message);
    }
  };

  return (
    <section
      className="overflow-hidden rounded-xl border border-slate-200 bg-white"
      onKeyDown={(event) => {
        if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
          event.preventDefault();
          props.onSubmit();
        }
      }}
    >
      <CardHeader icon={SlidersHorizontal} title="Arguments">
        {!props.jsonOnly && (
          <SegmentedControl
            size="xs"
            aria-label="Arguments editor"
            value={effectiveMode}
            onChange={(value) => changeMode(value as ArgumentsMode)}
            data={[
              { label: "Form", value: "form" },
              { label: "JSON", value: "json" },
            ]}
          />
        )}
      </CardHeader>

      <div className="flex flex-col gap-3 p-3">
        {effectiveMode === "json" ? (
          <Textarea
            aria-label="Arguments as JSON"
            autosize
            minRows={4}
            maxRows={16}
            spellCheck={false}
            styles={monospace}
            placeholder="{}"
            value={json}
            onChange={(event) => {
              setSwitchError(undefined);
              onJSON(event.currentTarget.value);
            }}
          />
        ) : fields.length === 0 ? (
          <p className="rounded-lg border border-dashed border-slate-200 bg-slate-50/70 px-3 py-3 text-center text-xs text-slate-500">
            {props.emptyLabel}
          </p>
        ) : (
          fields.map((field) => (
            <FieldInput
              key={field.name}
              field={field}
              value={values[field.name] ?? ""}
              onChange={(value) => {
                setSwitchError(undefined);
                onValues({ ...values, [field.name]: value });
              }}
            />
          ))
        )}

        {error && (
          <p
            role="alert"
            className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-semibold leading-5 text-red-700"
          >
            <CircleAlert size={14} className="mt-0.5 shrink-0" />
            <span className="min-w-0 break-words">{error}</span>
          </p>
        )}
      </div>

      <div className="flex items-center justify-end gap-3 border-t border-slate-100 bg-slate-50/60 px-3 py-2">
        <span className="hidden items-center gap-1 text-micro text-slate-400 @md:flex">
          <Kbd>{modKey}</Kbd>
          <Kbd>↵</Kbd>
        </span>
        {props.children}
      </div>
    </section>
  );
};

export default Arguments;
