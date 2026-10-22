import { Duration } from "@/apis/metav1/metav1";
import DurationPicker from "@/components/DurationPicker";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { twMerge } from "@/lib/utils";
import {
  ActionIcon,
  Button,
  Collapse,
  NumberInput,
  Select,
  Switch,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  AlertTriangle,
  Check,
  ChevronRight,
  LucideIcon,
  Plus,
  SlidersHorizontal,
  Trash2,
} from "lucide-react";
import * as React from "react";

export type SecretRef = {
  type:
    { oneofKind: "fromSecret"; fromSecret: string } | { oneofKind: undefined };
};

export const toSecret = (name: string): SecretRef => ({
  type: { oneofKind: "fromSecret", fromSecret: name },
});

export const secretName = (ref?: SecretRef) =>
  ref?.type.oneofKind === "fromSecret" ? ref.type.fromSecret : "";

export const isHTTPURL = (value: string) => {
  try {
    const url = new URL(value);
    return (
      (url.protocol === "https:" || url.protocol === "http:") && !!url.host
    );
  } catch {
    return false;
  }
};

export const urlError = (value: string, opts?: { httpsOnly?: boolean }) => {
  if (!value) return undefined;
  if (!isHTTPURL(value)) return "Enter a full http:// or https:// URL";
  if (opts?.httpsOnly && !value.startsWith("https://"))
    return "Use an https:// URL";
  return undefined;
};

export const isUUID = (value: string) =>
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
    value.trim(),
  );

export const identError = (value: string, extra = "") => {
  if (!value) return undefined;
  return new RegExp(`^[A-Za-z0-9_${extra}]+$`).test(value.trim())
    ? undefined
    : `Use letters, digits${extra.includes("-") ? ", dashes" : ""}${extra.includes(".") ? ", dots" : ""} and underscores only`;
};

export const Grid = (props: {
  children: React.ReactNode;
  cols?: 2 | 3;
  className?: string;
}) => (
  <div
    className={twMerge(
      "grid grid-cols-1 gap-4",
      (props.cols ?? 2) === 2
        ? "@xl:grid-cols-2"
        : "@xl:grid-cols-2 @3xl:grid-cols-3",
      props.className,
    )}
  >
    {props.children}
  </div>
);

export const FormSection = (props: {
  icon: LucideIcon;
  title: string;
  description?: React.ReactNode;
  aside?: React.ReactNode;
  children: React.ReactNode;
}) => {
  const Icon = props.icon;
  return (
    <section className="min-w-0 rounded-xl border border-slate-200 bg-white">
      <header className="flex items-start gap-3 border-b border-slate-100 px-4 py-3">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
          <Icon size={14} strokeWidth={2.2} />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="text-body font-semibold text-slate-900">
            {props.title}
          </h3>
          {props.description && (
            <p className="text-xs font-normal leading-5 text-slate-500">
              {props.description}
            </p>
          )}
        </div>
        {props.aside}
      </header>
      <div className="flex flex-col gap-4 px-4 py-4">{props.children}</div>
    </section>
  );
};

export const Callout = (props: {
  tone?: "warning" | "info";
  title?: React.ReactNode;
  children?: React.ReactNode;
}) => (
  <div
    role="note"
    className={twMerge(
      "flex items-start gap-2.5 rounded-lg border px-3 py-2.5",
      props.tone === "info"
        ? "border-blue-200 bg-blue-50 text-blue-700"
        : "border-amber-200 bg-amber-50 text-amber-700",
    )}
  >
    <AlertTriangle size={14} strokeWidth={2.3} className="mt-0.5 shrink-0" />
    <div className="min-w-0 text-xs leading-5">
      {props.title && <div className="font-semibold">{props.title}</div>}
      {props.children && (
        <div className="font-normal text-slate-700">{props.children}</div>
      )}
    </div>
  </div>
);

export const ToggleList = (props: { children: React.ReactNode }) => (
  <div className="divide-y divide-slate-100 overflow-hidden rounded-lg border border-slate-200">
    {props.children}
  </div>
);

export const ToggleRow = (props: {
  label: string;
  description?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  children?: React.ReactNode;
}) => (
  <div className="bg-white">
    <Switch
      labelPosition="left"
      label={props.label}
      description={props.description}
      checked={props.checked}
      onChange={(event) => props.onChange(event.currentTarget.checked)}
      className="px-3.5 py-3"
      styles={{
        body: { justifyContent: "space-between", gap: "16px" },
        labelWrapper: { flex: 1, minWidth: 0 },
        label: { paddingInlineEnd: 0, paddingInlineStart: 0 },
        description: { marginBottom: 0, marginTop: "2px" },
      }}
    />
    {props.checked && props.children && (
      <div className="px-3.5 pb-3.5">{props.children}</div>
    )}
  </div>
);

export type Choice<T extends string> = {
  value: T;
  label: string;
  hint?: string;
};

export const ChoiceGroup = <T extends string>(props: {
  label: string;
  value: T;
  options: Choice<T>[];
  onChange: (value: T) => void;
}) => (
  <div className="flex min-w-0 flex-col gap-1.5">
    <span className="text-xs font-semibold text-slate-700">{props.label}</span>
    <div
      role="radiogroup"
      aria-label={props.label}
      className={twMerge(
        "grid grid-cols-1 gap-2",
        props.options.length === 2 && "@md:grid-cols-2",
        props.options.length === 3 && "@xl:grid-cols-3",
        props.options.length >= 4 && "@md:grid-cols-2 @3xl:grid-cols-4",
      )}
    >
      {props.options.map((option) => {
        const active = option.value === props.value;
        return (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => !active && props.onChange(option.value)}
            className={twMerge(
              "flex min-w-0 cursor-pointer items-start gap-2.5 rounded-lg border px-3 py-2.5 text-left outline-none transition-[border-color,box-shadow,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
              active
                ? "border-slate-800 bg-white shadow-sm ring-1 ring-slate-800"
                : "border-slate-200 bg-white hover:border-slate-300",
            )}
          >
            <span
              className={twMerge(
                "mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-full border",
                active
                  ? "border-slate-800 bg-slate-800 text-white"
                  : "border-slate-300",
              )}
            >
              {active && <Check size={10} strokeWidth={3} />}
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="text-xs font-semibold text-slate-900">
                {option.label}
              </span>
              {option.hint && (
                <span className="text-micro leading-4 text-slate-500">
                  {option.hint}
                </span>
              )}
            </span>
          </button>
        );
      })}
    </div>
  </div>
);

export const SecretField = (props: {
  label: string;
  description: string;
  required?: boolean;
  value?: string;
  onChange: (name: string) => void;
}) => (
  <SelectResource
    api="enterprise"
    kind="Secret"
    required={props.required}
    clearable={!props.required}
    label={props.label}
    description={props.description}
    defaultValue={props.value || undefined}
    onChange={(secret) => props.onChange(secret?.metadata?.name ?? "")}
  />
);

export const NumberField = (props: {
  label: string;
  description: string;
  value: number;
  min?: number;
  max?: number;
  suffix?: string;
  onChange: (value: number) => void;
}) => (
  <NumberInput
    label={props.label}
    description={props.description}
    placeholder="Default"
    min={props.min ?? 0}
    max={props.max}
    allowDecimal={false}
    allowNegative={(props.min ?? 0) < 0}
    suffix={props.suffix}
    value={props.value || ""}
    onChange={(value) => props.onChange(Number(value) || 0)}
  />
);

export const DurationField = (props: {
  label: string;
  description: string;
  value?: Duration;
  onChange: (value?: Duration) => void;
}) => (
  <DurationPicker
    title={props.label}
    description={props.description}
    placeholder="Default"
    value={props.value}
    onChange={props.onChange}
  />
);

export const EnumField = (props: {
  label: string;
  description: string;
  value: number;
  options: { label: string; value: number; disabled?: boolean }[];
  onChange: (value: number) => void;
}) => (
  <Select
    label={props.label}
    description={props.description}
    placeholder="Default"
    clearable
    data={props.options.map((option) => ({
      label: option.label,
      value: String(option.value),
      disabled: option.disabled,
    }))}
    value={props.value ? String(props.value) : null}
    onChange={(value) => props.onChange(value ? Number(value) : 0)}
  />
);

type KeyValue = { key: string; value: string };

export const KeyValueList = (props: {
  label: string;
  description: string;
  items: KeyValue[];
  make: () => KeyValue;
  addLabel: string;
  keyPlaceholder: string;
  valuePlaceholder: string;
  keyError?: (key: string, index: number) => string | undefined;
  onUpdate: () => void;
}) => {
  const nextID = React.useRef(0);
  const ids = React.useRef<number[]>([]);
  while (ids.current.length < props.items.length) {
    ids.current.push(nextID.current++);
  }
  if (ids.current.length > props.items.length) {
    ids.current.length = props.items.length;
  }

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs font-semibold text-slate-700">
            {props.label}
          </div>
          <div className="text-xs font-normal leading-5 text-slate-500">
            {props.description}
          </div>
        </div>
        <Button
          type="button"
          variant="default"
          size="compact-xs"
          className="shrink-0"
          leftSection={<Plus size={11} strokeWidth={2.5} />}
          onClick={() => {
            props.items.push(props.make());
            ids.current.push(nextID.current++);
            props.onUpdate();
          }}
        >
          {props.addLabel}
        </Button>
      </div>
      {props.items.length > 0 && (
        <div className="flex flex-col gap-2">
          {props.items.map((item, index) => (
            <div
              key={ids.current[index]}
              className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] items-start gap-2"
            >
              <TextInput
                aria-label={`${props.label} ${index + 1} key`}
                placeholder={props.keyPlaceholder}
                value={item.key}
                error={props.keyError?.(item.key, index)}
                onChange={(event) => {
                  item.key = event.currentTarget.value;
                  props.onUpdate();
                }}
              />
              <TextInput
                aria-label={`${props.label} ${index + 1} value`}
                placeholder={props.valuePlaceholder}
                value={item.value}
                onChange={(event) => {
                  item.value = event.currentTarget.value;
                  props.onUpdate();
                }}
              />
              <Tooltip label="Remove" withArrow>
                <ActionIcon
                  type="button"
                  variant="subtle"
                  color="gray"
                  size={38}
                  aria-label={`Remove ${props.label.toLowerCase()} ${index + 1}`}
                  onClick={() => {
                    props.items.splice(index, 1);
                    ids.current.splice(index, 1);
                    props.onUpdate();
                  }}
                >
                  <Trash2 size={14} strokeWidth={2.1} />
                </ActionIcon>
              </Tooltip>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export const headerKeyError =
  (items: KeyValue[], opts?: { allowAuthorization?: boolean }) =>
  (key: string, index: number) => {
    const trimmed = key.trim();
    if (!trimmed) return "Required";
    if (/[\s()<>@,;:\\"/[\]?={}]/.test(trimmed)) return "Invalid header name";
    if (!opts?.allowAuthorization && trimmed.toLowerCase() === "authorization")
      return "Use Authentication instead";
    if (
      items.findIndex(
        (item) => item.key.trim().toLowerCase() === trimmed.toLowerCase(),
      ) !== index
    )
      return "Duplicate";
    return undefined;
  };

export const Advanced = (props: {
  count: number;
  summary: string;
  children: React.ReactNode;
}) => {
  const [open, setOpen] = React.useState(false);
  const id = React.useId();
  return (
    <section className="min-w-0 rounded-xl border border-slate-200 bg-white">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((value) => !value)}
        className="flex w-full cursor-pointer items-center gap-3 rounded-xl px-4 py-3 text-left outline-none transition-colors duration-150 hover:bg-slate-50/70 focus-visible:ring-2 focus-visible:ring-slate-400"
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-500">
          <SlidersHorizontal size={14} strokeWidth={2.2} />
        </span>
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="flex flex-wrap items-center gap-2">
            <span className="text-body font-semibold text-slate-900">
              Advanced options
            </span>
            {props.count > 0 && (
              <span className="rounded-full border border-slate-200 bg-slate-50 px-1.5 py-px text-micro font-medium text-slate-600">
                {props.count} customized
              </span>
            )}
          </span>
          <span className="text-xs font-normal leading-5 text-slate-500">
            {props.summary}
          </span>
        </span>
        <ChevronRight
          size={15}
          strokeWidth={2.25}
          className={twMerge(
            "shrink-0 text-slate-500 transition-transform duration-150",
            open && "rotate-90",
          )}
        />
      </button>
      <Collapse expanded={open} transitionDuration={150}>
        <div
          id={id}
          className="flex flex-col divide-y divide-slate-100 border-t border-slate-100"
        >
          {props.children}
        </div>
      </Collapse>
    </section>
  );
};

export const AdvancedGroup = (props: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) => (
  <div className="flex min-w-0 flex-col gap-4 px-4 py-4">
    <div>
      <h4 className="text-xs font-semibold uppercase tracking-[0.06em] text-slate-500">
        {props.title}
      </h4>
      {props.description && (
        <p className="mt-0.5 text-xs font-normal leading-5 text-slate-500">
          {props.description}
        </p>
      )}
    </div>
    {props.children}
  </div>
);

export const countSet = (...values: unknown[]) =>
  values.filter((value) => (Array.isArray(value) ? value.length > 0 : !!value))
    .length;
