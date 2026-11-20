import { Timestamp } from "@/apis/google/protobuf/timestamp";
import SelectResourceMultiple from "@/components/ResourceLayout/SelectResourceMultiple";
import SelectCountry from "@/components/SelectCountry";
import TimestampPicker from "@/components/TimestampPicker";
import { twMerge } from "@/lib/utils";
import {
  Input,
  MultiSelect,
  NumberInput,
  SegmentedControl,
  Select,
  Switch,
  TagsInput,
  TextInput,
} from "@mantine/core";
import { TimeInput } from "@mantine/dates";
import { Check, LocateFixed } from "lucide-react";
import {
  DAYS,
  MATCH_OPS,
  NUMBER_OPTIONS,
  type FieldDef,
  type MatchOp,
  type MatchValue,
  type NumberOp,
  type NumberValue,
  type Option,
  type Transform,
  type Values,
} from "./blocks";
import { localTimezone } from "./recipes";

const COMMON_TIMEZONES = [
  "UTC",
  "America/Los_Angeles",
  "America/Denver",
  "America/Chicago",
  "America/New_York",
  "America/Sao_Paulo",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Africa/Cairo",
  "Asia/Dubai",
  "Asia/Kolkata",
  "Asia/Singapore",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Australia/Sydney",
];

const TIMEZONES =
  typeof Intl !== "undefined" && typeof Intl.supportedValuesOf === "function"
    ? Array.from(new Set(["UTC", ...Intl.supportedValuesOf("timeZone")]))
    : COMMON_TIMEZONES;

const applyTransform = (value: string, transform?: Transform) => {
  const trimmed = value.trim();
  if (transform === "upper") return trimmed.toUpperCase();
  if (transform === "lower") return trimmed.toLowerCase();
  if (transform === "cidr") {
    return trimmed && !trimmed.includes("/")
      ? `${trimmed}/${trimmed.includes(":") ? 128 : 32}`
      : trimmed;
  }
  return trimmed;
};

const unique = (values: string[]) =>
  Array.from(new Set(values.filter((value) => value !== "")));

const toTimestamp = (value?: string) => {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : Timestamp.fromDate(date);
};

export const OptionChips = (props: {
  label: string;
  options: Option[];
  value: string[];
  onChange: (value: string[]) => void;
}) => (
  <Input.Wrapper label={props.label}>
    <div
      role="group"
      aria-label={props.label}
      className="mt-1 flex flex-wrap gap-1.5"
    >
      {props.options.map((option) => {
        const active = props.value.includes(option.value);
        return (
          <button
            key={option.value}
            type="button"
            role="checkbox"
            aria-checked={active}
            onClick={() =>
              props.onChange(
                active
                  ? props.value.filter((value) => value !== option.value)
                  : [...props.value, option.value],
              )
            }
            className={twMerge(
              "inline-flex h-7 cursor-pointer items-center gap-1 rounded-lg border px-2.5 text-xs font-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
              active
                ? "border-slate-800 bg-slate-800 text-white"
                : "border-slate-200 bg-white text-slate-700 hover:border-slate-300 hover:bg-slate-50",
            )}
          >
            {active && <Check size={11} strokeWidth={3} />}
            {option.label}
          </button>
        );
      })}
    </div>
  </Input.Wrapper>
);

const MatchField = (props: {
  field: Extract<FieldDef, { kind: "match" }>;
  value?: MatchValue;
  onChange: (value: MatchValue) => void;
}) => {
  const value = props.value ?? { op: "is", values: [] };
  const op: MatchOp = value.op === "in" ? "is" : value.op;
  return (
    <div className="grid grid-cols-1 gap-2 @md:grid-cols-[150px_minmax(0,1fr)]">
      <Select
        label={props.field.label}
        aria-label={`${props.field.label} operator`}
        allowDeselect={false}
        data={MATCH_OPS.filter((item) => item.value !== "in")}
        value={op}
        onChange={(next) =>
          next && props.onChange({ ...value, op: next as MatchOp })
        }
      />
      <TagsInput
        label={<span className="@md:invisible">Values</span>}
        aria-label={`${props.field.label} values`}
        placeholder={
          value.values.length === 0
            ? (props.field.placeholder ?? "Type a value, then Enter")
            : undefined
        }
        data={props.field.suggestions}
        value={value.values}
        acceptValueOnBlur
        splitChars={[","]}
        clearable
        onChange={(values) =>
          props.onChange({
            ...value,
            values: unique(
              values.map((item) => applyTransform(item, props.field.transform)),
            ),
          })
        }
      />
    </div>
  );
};

const NumberField = (props: {
  label: string;
  value?: NumberValue;
  onChange: (value: NumberValue) => void;
}) => {
  const value = props.value ?? { op: "<=" };
  return (
    <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-2 @md:grid-cols-[150px_200px]">
      <Select
        label={props.label}
        aria-label={`${props.label} operator`}
        allowDeselect={false}
        data={NUMBER_OPTIONS}
        value={value.op}
        onChange={(op) =>
          op && props.onChange({ ...value, op: op as NumberOp })
        }
      />
      <NumberInput
        label={<span className="invisible">Value</span>}
        aria-label={props.label}
        allowDecimal={false}
        value={value.value ?? ""}
        onChange={(next) =>
          props.onChange({
            ...value,
            value: typeof next === "number" ? next : undefined,
          })
        }
      />
    </div>
  );
};

const DaysField = (props: {
  value: string[];
  onChange: (value: string[]) => void;
}) => {
  const presets = [
    { label: "Weekdays", days: ["1", "2", "3", "4", "5"] },
    { label: "Weekends", days: ["0", "6"] },
    { label: "Every day", days: [] as string[] },
  ];
  const sorted = [...props.value].sort().join();
  const ordered = [...DAYS.slice(1), DAYS[0]];
  return (
    <Input.Wrapper label="Days">
      <div className="mt-1 flex flex-wrap items-center gap-2">
        <div
          role="group"
          aria-label="Days of the week"
          className="flex overflow-hidden rounded-lg border border-slate-200 bg-white"
        >
          {ordered.map((day) => {
            const active =
              props.value.includes(day.value) || props.value.length === 0;
            return (
              <button
                key={day.value}
                type="button"
                role="checkbox"
                aria-checked={props.value.includes(day.value)}
                aria-label={day.label}
                onClick={() => {
                  const current =
                    props.value.length === 0
                      ? DAYS.map((item) => item.value)
                      : props.value;
                  const next = current.includes(day.value)
                    ? current.filter((value) => value !== day.value)
                    : [...current, day.value];
                  props.onChange(next.length === 7 ? [] : next);
                }}
                className={twMerge(
                  "h-8 w-9 cursor-pointer border-l border-slate-200 text-xs font-semibold outline-none transition-colors duration-150 first:border-l-0 focus-visible:z-10 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-slate-400 sm:w-10",
                  active
                    ? "bg-slate-800 text-white"
                    : "bg-white text-slate-500 hover:bg-slate-50 hover:text-slate-800",
                )}
              >
                {day.label.slice(0, 2)}
              </button>
            );
          })}
        </div>
        <div className="flex flex-wrap gap-1">
          {presets.map((preset) => {
            const active = [...preset.days].sort().join() === sorted;
            return (
              <button
                key={preset.label}
                type="button"
                onClick={() => props.onChange(preset.days)}
                className={twMerge(
                  "h-7 cursor-pointer rounded-md px-2 text-micro font-semibold outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
                  active
                    ? "bg-slate-100 text-slate-900"
                    : "text-slate-500 hover:bg-slate-50 hover:text-slate-800",
                )}
              >
                {preset.label}
              </button>
            );
          })}
        </div>
      </div>
    </Input.Wrapper>
  );
};

const HoursField = (props: {
  from?: string;
  to?: string;
  onChange: (from: string, to: string) => void;
}) => {
  const allDay = !props.from && !props.to;
  return (
    <Input.Wrapper label="Hours">
      <div className="mt-1 flex flex-wrap items-center gap-2">
        <SegmentedControl
          size="xs"
          value={allDay ? "all" : "between"}
          data={[
            { value: "all", label: "All day" },
            { value: "between", label: "Between" },
          ]}
          onChange={(mode) =>
            mode === "all"
              ? props.onChange("", "")
              : props.onChange(props.from || "09:00", props.to || "17:00")
          }
        />
        {!allDay && (
          <div className="flex items-center gap-1.5">
            <TimeInput
              aria-label="From"
              className="w-[110px]"
              value={props.from ?? ""}
              onChange={(event) =>
                props.onChange(event.currentTarget.value, props.to ?? "")
              }
            />
            <span className="text-xs text-slate-500">to</span>
            <TimeInput
              aria-label="To"
              className="w-[110px]"
              value={props.to ?? ""}
              onChange={(event) =>
                props.onChange(props.from ?? "", event.currentTarget.value)
              }
            />
          </div>
        )}
      </div>
    </Input.Wrapper>
  );
};

const TimezoneField = (props: {
  label: string;
  value?: string;
  onChange: (value: string) => void;
}) => {
  const local = localTimezone();
  return (
    <div className="flex items-end gap-1.5">
      <Select
        className="min-w-0 flex-1"
        label={props.label}
        placeholder="UTC"
        searchable
        clearable
        data={TIMEZONES}
        value={props.value || null}
        limit={60}
        nothingFoundMessage="No such timezone"
        onChange={(value) => props.onChange(value ?? "")}
      />
      {local && local !== props.value && (
        <button
          type="button"
          onClick={() => props.onChange(local)}
          title={`Use ${local}`}
          className="mb-px inline-flex h-9 shrink-0 cursor-pointer items-center gap-1 rounded-md border border-slate-200 bg-white px-2 text-micro font-semibold text-slate-600 outline-none transition-colors hover:border-slate-300 hover:text-slate-900 focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <LocateFixed size={12} />
          <span className="hidden @md:inline">Mine</span>
        </button>
      )}
    </div>
  );
};

const isWide = (field: FieldDef) =>
  [
    "refs",
    "match",
    "strings",
    "days",
    "hours",
    "countries",
    "options",
  ].includes(field.kind);

export const BlockFields = (props: {
  fields: [string, FieldDef][];
  values: Values;
  onChange: (values: Values) => void;
}) => {
  const { values } = props;
  const set = (key: string, value: unknown) =>
    props.onChange({ ...values, [key]: value });

  const render = (key: string, field: FieldDef) => {
    switch (field.kind) {
      case "refs":
        return (
          <SelectResourceMultiple
            api="core"
            kind={field.resource}
            label={field.label}
            defaultValue={values[key] ?? []}
            onChange={(items) =>
              set(
                key,
                (items ?? [])
                  .map((item) => item.metadata?.name ?? "")
                  .filter(Boolean),
              )
            }
          />
        );
      case "options":
        return field.options.length <= 9 ? (
          <OptionChips
            label={field.label}
            options={field.options}
            value={values[key] ?? []}
            onChange={(value) => set(key, value)}
          />
        ) : (
          <MultiSelect
            label={field.label}
            searchable
            clearable
            placeholder={(values[key] ?? []).length ? undefined : "Choose…"}
            data={field.options}
            value={values[key] ?? []}
            onChange={(value) => set(key, value)}
          />
        );
      case "countries":
        return (
          <SelectCountry
            multiple
            label={field.label}
            values={values[key] ?? []}
            onUpdate={(value) => set(key, Array.isArray(value) ? value : [])}
          />
        );
      case "strings":
        return (
          <TagsInput
            label={field.label}
            placeholder={
              (values[key] ?? []).length
                ? undefined
                : (field.placeholder ?? "Type a value, then Enter")
            }
            data={field.suggestions}
            value={values[key] ?? []}
            acceptValueOnBlur
            splitChars={[",", " "]}
            clearable
            onChange={(value) =>
              set(
                key,
                unique(
                  value.map((item) => applyTransform(item, field.transform)),
                ),
              )
            }
          />
        );
      case "text":
        return (
          <TextInput
            label={field.label}
            placeholder={field.placeholder}
            value={values[key] ?? ""}
            onChange={(event) => set(key, event.currentTarget.value)}
          />
        );
      case "match":
        return (
          <MatchField
            field={field}
            value={values[key]}
            onChange={(value) => set(key, value)}
          />
        );
      case "number":
        return (
          <NumberField
            label={field.label}
            value={values[key]}
            onChange={(value) => set(key, value)}
          />
        );
      case "timestamp":
        return (
          <TimestampPicker
            label={field.label}
            placeholder={field.optional ? "Not set" : undefined}
            isFuture={key === "before"}
            disableExcludePast
            value={toTimestamp(values[key])}
            onChange={(value) =>
              set(key, value ? Timestamp.toDate(value).toISOString() : "")
            }
          />
        );
      case "timezone":
        return (
          <TimezoneField
            label={field.label}
            value={values[key]}
            onChange={(value) => set(key, value)}
          />
        );
      case "days":
        return (
          <DaysField
            value={values[key] ?? []}
            onChange={(value) => set(key, value)}
          />
        );
      case "hours":
        return (
          <HoursField
            from={values.from}
            to={values.to}
            onChange={(from, to) => props.onChange({ ...values, from, to })}
          />
        );
      case "switch":
        return (
          <Switch
            className="pt-1"
            label={field.label}
            description={field.description}
            checked={!!values[key]}
            onChange={(event) => set(key, event.currentTarget.checked)}
          />
        );
      case "choice":
        return (
          <Input.Wrapper label={field.label}>
            <div className="mt-1">
              <SegmentedControl
                size="xs"
                data={field.options}
                value={values[key] ?? field.options[0]?.value}
                onChange={(value) => set(key, value)}
              />
            </div>
          </Input.Wrapper>
        );
    }
  };

  if (props.fields.length === 0) return null;

  return (
    <div className="grid grid-cols-1 gap-3 @xl:grid-cols-2">
      {props.fields
        .filter(([, field]) => !field.visible || field.visible(values))
        .map(([key, field]) => (
          <div
            key={key}
            className={twMerge("min-w-0", isWide(field) && "@xl:col-span-2")}
          >
            {render(key, field)}
          </div>
        ))}
    </div>
  );
};
