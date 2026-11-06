import { CELEditor, OPAEditor } from "@/components/Condition/Editor";
import { Duration } from "@/apis/metav1/metav1";
import DurationPicker from "@/components/DurationPicker";
import Editor from "@/components/Editor";
import SelectResource from "@/components/ResourceLayout/SelectResource";
import { twMerge } from "@/lib/utils";
import {
  secretName,
  SecretRef,
  ToggleRow,
  toSecret,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import {
  ActionIcon,
  Button,
  NumberInput,
  SegmentedControl,
  Select,
  TagsInput,
  Textarea,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  ArrowDown,
  ArrowUp,
  ChevronRight,
  LucideIcon,
  Plus,
  Trash2,
} from "lucide-react";
import * as React from "react";

export type Update = () => void;

export type Scope = "default" | "named";

type KeysOf<T, V> = {
  [K in keyof T]-?: T[K] extends V ? K : never;
}[keyof T];

const setKey = (target: object, key: PropertyKey, value: unknown) => {
  (target as Record<PropertyKey, unknown>)[key] = value;
};

const pressable = (onPress: () => void) => ({
  role: "button",
  tabIndex: 0,
  onClick: onPress,
  onKeyDown: (event: React.KeyboardEvent) => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    onPress();
  },
});

export const Pill = (props: {
  children: React.ReactNode;
  tone?: "warning";
}) => (
  <span
    className={twMerge(
      "whitespace-nowrap rounded-full border px-1.5 py-px text-micro font-medium",
      props.tone === "warning"
        ? "border-amber-200 bg-amber-50 text-amber-700"
        : "border-slate-200 bg-slate-50 text-slate-600",
    )}
  >
    {props.children}
  </span>
);

export const FieldText = (props: {
  label: string;
  description?: React.ReactNode;
}) => (
  <div className="min-w-0">
    <div className="text-xs font-semibold text-slate-700">{props.label}</div>
    {props.description && (
      <div className="text-xs font-normal leading-5 text-slate-500">
        {props.description}
      </div>
    )}
  </div>
);

export const Note = (props: { children: React.ReactNode }) => (
  <p className="text-xs font-normal leading-5 text-slate-500">
    {props.children}
  </p>
);

export const TextField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, string>;
  update: Update;
  label: string;
  description?: string;
  placeholder?: string;
  required?: boolean;
  mono?: boolean;
  error?: (value: string) => string | undefined;
  className?: string;
}) => {
  const value = props.of[props.name] as string;
  return (
    <TextInput
      label={props.label}
      description={props.description}
      placeholder={props.placeholder}
      required={props.required}
      className={props.className}
      classNames={props.mono ? { input: "font-mono" } : undefined}
      value={value}
      error={value ? props.error?.(value) : undefined}
      onChange={(event) => {
        setKey(props.of, props.name, event.currentTarget.value);
        props.update();
      }}
    />
  );
};

export const NumberField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, number>;
  update: Update;
  label: string;
  description?: string;
  placeholder?: string;
  required?: boolean;
  min?: number;
  max?: number;
  step?: number;
  decimals?: number;
  suffix?: string;
  thousands?: boolean;
  error?: (value: number) => string | undefined;
}) => {
  const value = props.of[props.name] as number;
  return (
    <NumberInput
      label={props.label}
      description={props.description}
      placeholder={props.placeholder ?? "Default"}
      required={props.required}
      min={props.min ?? 0}
      max={props.max}
      step={props.step}
      allowDecimal={props.decimals !== undefined}
      decimalScale={props.decimals}
      allowNegative={false}
      suffix={props.suffix}
      thousandSeparator={props.thousands ? "," : undefined}
      value={value || ""}
      error={value ? props.error?.(value) : undefined}
      onChange={(next) => {
        setKey(props.of, props.name, Number(next) || 0);
        props.update();
      }}
    />
  );
};

export const StatusField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, number>;
  update: Update;
  label?: string;
  description?: string;
  placeholder?: string;
}) => (
  <NumberField
    of={props.of}
    name={props.name}
    update={props.update}
    label={props.label ?? "Status code"}
    description={props.description}
    placeholder={props.placeholder}
    max={599}
    error={(value) =>
      value < 200 ? "Use a status code between 200 and 599" : undefined
    }
  />
);

export const DurationField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, Duration | undefined>;
  update: Update;
  label: string;
  description?: string;
  required?: boolean;
}) => (
  <DurationPicker
    title={props.label}
    description={props.description}
    required={props.required}
    placeholder={props.required ? undefined : "Default"}
    value={props.of[props.name] as Duration | undefined}
    onChange={(value) => {
      setKey(props.of, props.name, value);
      props.update();
    }}
  />
);

export const Toggle = <T extends object>(props: {
  of: T;
  name: KeysOf<T, boolean>;
  update: Update;
  label: string;
  description?: string;
  children?: React.ReactNode;
}) => (
  <ToggleRow
    label={props.label}
    description={props.description}
    checked={props.of[props.name] as boolean}
    onChange={(checked) => {
      setKey(props.of, props.name, checked);
      props.update();
    }}
  >
    {props.children}
  </ToggleRow>
);

export const TagsField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, string[]>;
  update: Update;
  label: string;
  description?: string;
  placeholder?: string;
  data?: string[];
  error?: (values: string[]) => string | undefined;
}) => {
  const values = props.of[props.name] as string[];
  return (
    <TagsInput
      label={props.label}
      description={props.description}
      placeholder={values.length ? undefined : props.placeholder}
      data={props.data}
      splitChars={[",", " "]}
      acceptValueOnBlur
      value={values}
      error={props.error?.(values)}
      onChange={(next) => {
        setKey(props.of, props.name, next);
        props.update();
      }}
    />
  );
};

export type EnumOption = { value: number; label: string };

export const EnumField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, number>;
  update: Update;
  label: string;
  description?: string;
  options: EnumOption[];
  placeholder?: string;
  clearable?: boolean;
}) => {
  const value = props.of[props.name] as number;
  return (
    <Select
      label={props.label}
      description={props.description}
      placeholder={props.placeholder ?? "Default"}
      clearable={props.clearable ?? true}
      allowDeselect={false}
      data={props.options.map((option) => ({
        value: String(option.value),
        label: option.label,
      }))}
      value={value ? String(value) : null}
      onChange={(next) => {
        setKey(props.of, props.name, next ? Number(next) : 0);
        props.update();
      }}
    />
  );
};

export const Segments = <V extends string>(props: {
  label: string;
  hideLabel?: boolean;
  value?: V;
  options: { value: V; label: string }[];
  onChange: (value: V) => void;
  fullWidth?: boolean;
}) => (
  <div className="flex min-w-0 flex-col gap-1.5">
    {!props.hideLabel && (
      <span className="text-xs font-semibold text-slate-700">
        {props.label}
      </span>
    )}
    <div className="max-w-full overflow-x-auto">
      <SegmentedControl
        aria-label={props.label}
        size="xs"
        fullWidth={props.fullWidth}
        value={props.value ?? ""}
        data={props.options}
        onChange={(value) => props.onChange(value as V)}
      />
    </div>
  </div>
);

export const ChipField = (props: {
  label: string;
  description?: string;
  options: EnumOption[];
  selected: number[];
  onChange: (selected: number[]) => void;
}) => (
  <div className="flex min-w-0 flex-col gap-1.5">
    <FieldText label={props.label} description={props.description} />
    <div role="group" aria-label={props.label} className="flex flex-wrap gap-1.5">
      {props.options.map((option) => {
        const checked = props.selected.includes(option.value);
        return (
          <button
            key={option.value}
            type="button"
            role="checkbox"
            aria-checked={checked}
            onClick={() =>
              props.onChange(
                checked
                  ? props.selected.filter((item) => item !== option.value)
                  : [...props.selected, option.value],
              )
            }
            className={twMerge(
              "cursor-pointer rounded-lg border px-2.5 py-1.5 text-xs font-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
              checked
                ? "border-slate-800 bg-slate-800 text-white"
                : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:text-slate-800",
            )}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  </div>
);

export const SecretField = (props: {
  label?: string;
  description?: string;
  required?: boolean;
  value?: string;
  onChange: (name: string) => void;
}) => (
  <SelectResource
    api="core"
    kind="Secret"
    required={props.required}
    clearable={!props.required}
    label={props.label}
    description={props.description}
    defaultValue={props.value || undefined}
    onChange={(secret) => props.onChange(secret?.metadata?.name ?? "")}
  />
);

export const SecretRefField = <T extends object>(props: {
  of: T;
  name: KeysOf<T, SecretRef | undefined>;
  update: Update;
  label: string;
  description?: string;
  required?: boolean;
}) => (
  <SecretField
    label={props.label}
    description={props.description}
    required={props.required}
    value={secretName(props.of[props.name] as SecretRef | undefined)}
    onChange={(name) => {
      setKey(
        props.of,
        props.name,
        name || props.required ? toSecret(name) : undefined,
      );
      props.update();
    }}
  />
);

export const SecretSourceField = (props: {
  value: SecretRef;
  update: Update;
  label: string;
  description?: string;
}) => (
  <SecretField
    label={props.label}
    description={props.description}
    required
    value={secretName(props.value)}
    onChange={(name) => {
      props.value.type = toSecret(name).type;
      props.update();
    }}
  />
);

export const UserField = (props: {
  label?: string;
  description?: string;
  value?: string;
  onChange: (name: string) => void;
}) => (
  <SelectResource
    api="core"
    kind="User"
    clearable
    label={props.label}
    description={props.description}
    defaultValue={props.value || undefined}
    onChange={(user) => props.onChange(user?.metadata?.name ?? "")}
  />
);

export const RemoveButton = (props: { label: string; onClick: () => void }) => (
  <Tooltip label="Remove" withArrow>
    <ActionIcon
      type="button"
      variant="subtle"
      color="gray"
      size={36}
      className="shrink-0"
      aria-label={props.label}
      onClick={props.onClick}
    >
      <Trash2 size={14} strokeWidth={2.1} />
    </ActionIcon>
  </Tooltip>
);

export const AddButton = (props: {
  label: string;
  ariaLabel?: string;
  onClick: () => void;
}) => (
  <Button
    type="button"
    variant="default"
    size="compact-xs"
    className="shrink-0"
    aria-label={props.ariaLabel}
    leftSection={<Plus size={11} strokeWidth={2.5} />}
    onClick={props.onClick}
  >
    {props.label}
  </Button>
);

export const ListField = (props: {
  label: string;
  description?: React.ReactNode;
  addLabel: string;
  count: number;
  onAdd: () => void;
  children?: React.ReactNode;
}) => (
  <div className="flex min-w-0 flex-col gap-2">
    <div className="flex items-start justify-between gap-3">
      <FieldText label={props.label} description={props.description} />
      <AddButton
        label={props.addLabel}
        ariaLabel={`${props.addLabel} to ${props.label}`}
        onClick={props.onAdd}
      />
    </div>
    {props.count > 0 && <div className="flex flex-col gap-2">{props.children}</div>}
  </div>
);

export const StringList = (props: {
  label: string;
  description?: React.ReactNode;
  values: string[];
  update: Update;
  addLabel: string;
  itemLabel: string;
  placeholder?: string;
  mono?: boolean;
  numbered?: boolean;
  error?: (value: string) => string | undefined;
}) => {
  const rows = useListKeys();
  const inputs = React.useRef<Record<string, HTMLInputElement | null>>({});
  const focusNext = React.useRef(false);
  const lastKey = props.values.length
    ? rows.keyAt("rows", props.values.length - 1)
    : undefined;

  React.useEffect(() => {
    if (!focusNext.current || !lastKey) return;
    focusNext.current = false;
    inputs.current[lastKey]?.focus();
  }, [lastKey]);

  const add = () => {
    focusNext.current = true;
    props.values.push("");
    props.update();
  };

  return (
    <ListField
      label={props.label}
      description={props.description}
      addLabel={props.addLabel}
      count={props.values.length}
      onAdd={add}
    >
      {props.values.map((value, index) => {
        const key = rows.keyAt("rows", index);
        return (
          <div key={key} className="flex min-w-0 items-start gap-2">
            {props.numbered && (
              <span className="flex h-9 w-5 shrink-0 items-center justify-end font-mono text-micro text-slate-400">
                {index}
              </span>
            )}
            <TextInput
              ref={(element) => {
                inputs.current[key] = element;
              }}
              aria-label={`${props.itemLabel} ${index + 1}`}
              className="min-w-0 flex-1"
              classNames={props.mono ? { input: "font-mono" } : undefined}
              placeholder={props.placeholder}
              value={value}
              error={value ? props.error?.(value) : undefined}
              onChange={(event) => {
                props.values[index] = event.currentTarget.value;
                props.update();
              }}
              onKeyDown={(event) => {
                if (event.key !== "Enter") return;
                event.preventDefault();
                add();
              }}
            />
            <RemoveButton
              label={`Remove ${props.itemLabel} ${index + 1}`}
              onClick={() => {
                rows.removeAt("rows", index);
                props.values.splice(index, 1);
                props.update();
              }}
            />
          </div>
        );
      })}
    </ListField>
  );
};

export const PEMList = (props: {
  label: string;
  description?: string;
  values: string[];
  update: Update;
}) => {
  const rows = useListKeys();
  return (
    <ListField
      label={props.label}
      description={props.description}
      addLabel="Add certificate"
      count={props.values.length}
      onAdd={() => {
        props.values.push("");
        props.update();
      }}
    >
      {props.values.map((value, index) => (
        <div key={rows.keyAt("rows", index)} className="flex items-start gap-2">
          <Textarea
            aria-label={`${props.label} ${index + 1}`}
            className="min-w-0 flex-1"
            classNames={{ input: "font-mono !text-xs" }}
            placeholder="-----BEGIN CERTIFICATE-----"
            autosize
            minRows={3}
            maxRows={8}
            value={value}
            error={
              value && !value.includes("-----BEGIN CERTIFICATE-----")
                ? "Paste a PEM-encoded certificate"
                : undefined
            }
            onChange={(event) => {
              props.values[index] = event.currentTarget.value;
              props.update();
            }}
          />
          <RemoveButton
            label={`Remove ${props.label} ${index + 1}`}
            onClick={() => {
              rows.removeAt("rows", index);
              props.values.splice(index, 1);
              props.update();
            }}
          />
        </div>
      ))}
    </ListField>
  );
};

export const CodeField = (props: {
  label: string;
  description?: React.ReactNode;
  required?: boolean;
  mode: "json" | "lua" | "yaml";
  value: string;
  onChange: (value: string) => void;
  error?: string;
}) => (
  <div className="flex min-w-0 flex-col gap-1.5">
    <FieldText label={props.label} description={props.description} />
    <div className="min-w-0 overflow-hidden rounded-lg border border-slate-200">
      <Editor
        mode={props.mode}
        value={props.value}
        onChange={(value) => props.onChange(value ?? "")}
        minHeight="140px"
        maxHeight="420px"
      />
    </div>
    {props.error && (
      <span className="text-xs font-normal text-red-600">{props.error}</span>
    )}
  </div>
);

export const jsonError = (value: string, opts?: { object?: boolean }) => {
  if (!value.trim()) return undefined;
  try {
    const parsed = JSON.parse(value);
    if (
      opts?.object &&
      (parsed === null || typeof parsed !== "object" || Array.isArray(parsed))
    ) {
      return "Enter a JSON object";
    }
    return undefined;
  } catch {
    return "Enter valid JSON";
  }
};

export type ValueSource =
  | { oneofKind: "value"; value: string }
  | { oneofKind: "eval"; eval: string }
  | { oneofKind: "opa"; opa: string }
  | { oneofKind: undefined };

type SourceKind = "value" | "eval" | "opa";

const sourceText = (type: ValueSource) => {
  switch (type.oneofKind) {
    case "value":
      return type.value;
    case "eval":
      return type.eval;
    case "opa":
      return type.opa;
    default:
      return "";
  }
};

const makeSource = (kind: SourceKind, text = ""): ValueSource => {
  switch (kind) {
    case "eval":
      return { oneofKind: "eval", eval: text };
    case "opa":
      return { oneofKind: "opa", opa: text };
    default:
      return { oneofKind: "value", value: text };
  }
};

export const SourceField = (props: {
  label: string;
  description?: React.ReactNode;
  holder: { type: ValueSource };
  update: Update;
  placeholder?: string;
  multiline?: boolean;
  json?: boolean;
}) => {
  const kind = (props.holder.type.oneofKind ?? "value") as SourceKind;
  const text = sourceText(props.holder.type);
  const set = (next: string) => {
    props.holder.type = makeSource(kind, next);
    props.update();
  };
  const memory = React.useRef<Partial<Record<SourceKind, string>>>({});

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex flex-wrap items-end justify-between gap-x-3 gap-y-2">
        <FieldText label={props.label} description={props.description} />
        <Segments
          label={`${props.label} source`}
          hideLabel
          value={kind}
          options={[
            { value: "value", label: "Value" },
            { value: "eval", label: "CEL" },
            { value: "opa", label: "Rego" },
          ]}
          onChange={(next) => {
            memory.current[kind] = text;
            props.holder.type = makeSource(next, memory.current[next] ?? "");
            props.update();
          }}
        />
      </div>
      {kind === "eval" && (
        <CELEditor label={`${props.label} CEL`} exp={text} onChange={set} />
      )}
      {kind === "opa" && (
        <OPAEditor label={`${props.label} Rego`} exp={text} onChange={set} />
      )}
      {kind === "value" &&
        (props.multiline ? (
          <Textarea
            aria-label={props.label}
            classNames={props.json ? { input: "font-mono !text-xs" } : undefined}
            autosize
            minRows={3}
            maxRows={14}
            placeholder={props.placeholder}
            value={text}
            error={props.json ? jsonError(text, { object: true }) : undefined}
            onChange={(event) => set(event.currentTarget.value)}
          />
        ) : (
          <TextInput
            aria-label={props.label}
            placeholder={props.placeholder}
            value={text}
            onChange={(event) => set(event.currentTarget.value)}
          />
        ))}
    </div>
  );
};

const toBase64 = (bytes: Uint8Array) => btoa(String.fromCharCode(...bytes));

const fromBase64 = (value: string) => {
  try {
    return Uint8Array.from(atob(value.replace(/\s/g, "")), (char) =>
      char.charCodeAt(0),
    );
  } catch {
    return undefined;
  }
};

type BodyType =
  | { oneofKind: "inline"; inline: string }
  | { oneofKind: "inlineBytes"; inlineBytes: Uint8Array }
  | { oneofKind: undefined };

export const BodyField = (props: {
  label: string;
  description?: string;
  holder: { type: BodyType };
  update: Update;
  placeholder?: string;
}) => {
  const type = props.holder.type;
  const [draft, setDraft] = React.useState(() =>
    type.oneofKind === "inlineBytes" ? toBase64(type.inlineBytes) : "",
  );
  const bytes = type.oneofKind === "inlineBytes";

  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex flex-wrap items-end justify-between gap-x-3 gap-y-2">
        <FieldText label={props.label} description={props.description} />
        <Segments
          label={`${props.label} format`}
          hideLabel
          value={bytes ? "inlineBytes" : "inline"}
          options={[
            { value: "inline", label: "Text" },
            { value: "inlineBytes", label: "Base64" },
          ]}
          onChange={(next) => {
            props.holder.type =
              next === "inlineBytes"
                ? { oneofKind: "inlineBytes", inlineBytes: new Uint8Array() }
                : { oneofKind: "inline", inline: "" };
            setDraft("");
            props.update();
          }}
        />
      </div>
      <Textarea
        aria-label={props.label}
        classNames={{ input: "font-mono !text-xs" }}
        autosize
        minRows={3}
        maxRows={14}
        placeholder={bytes ? "SGVsbG8gd29ybGQ=" : props.placeholder}
        value={
          bytes
            ? draft
            : type.oneofKind === "inline"
              ? type.inline
              : ""
        }
        error={bytes && draft && !fromBase64(draft) ? "Enter valid Base64" : undefined}
        onChange={(event) => {
          const value = event.currentTarget.value;
          if (bytes) {
            setDraft(value);
            const decoded = fromBase64(value);
            if (!decoded) return;
            props.holder.type = { oneofKind: "inlineBytes", inlineBytes: decoded };
          } else {
            props.holder.type = { oneofKind: "inline", inline: value };
          }
          props.update();
        }}
      />
    </div>
  );
};

export const OptionList = (props: {
  title?: string;
  description?: string;
  children: React.ReactNode;
}) => (
  <section className="flex min-w-0 flex-col gap-2">
    {props.title && (
      <div>
        <h3 className="text-body font-semibold text-slate-900">
          {props.title}
        </h3>
        {props.description && (
          <p className="text-xs font-normal leading-5 text-slate-500">
            {props.description}
          </p>
        )}
      </div>
    )}
    <div className="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 bg-white">
      {props.children}
    </div>
  </section>
);

const RowIcon = (props: { icon: LucideIcon }) => {
  const Icon = props.icon;
  return (
    <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-500">
      <Icon size={13} strokeWidth={2.2} />
    </span>
  );
};

export const OptionRow = (props: {
  icon: LucideIcon;
  title: string;
  description?: string;
  summary?: React.ReactNode;
  isSet: boolean;
  pinned?: boolean;
  defaultOpen?: boolean;
  onAdd?: () => void;
  onRemove?: () => void;
  children?: React.ReactNode;
}) => {
  const [open, setOpen] = React.useState(!!props.defaultOpen);
  const text = (
    <span className="flex min-w-0 flex-1 flex-col">
      <span className="text-body font-semibold text-slate-900">
        {props.title}
      </span>
      <span className="truncate text-xs font-normal leading-5 text-slate-500">
        {props.isSet ? (props.summary ?? props.description) : props.description}
      </span>
    </span>
  );

  return (
    <div className="min-w-0 bg-white" data-option-row={props.title}>
      <div className="flex min-w-0 items-center gap-2 px-3.5 py-2.5">
        {props.isSet ? (
          <div
            {...pressable(() => setOpen((value) => !value))}
            aria-expanded={open}
            aria-label={props.title}
            className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
          >
            <RowIcon icon={props.icon} />
            {text}
            <ChevronRight
              size={15}
              strokeWidth={2.25}
              className={twMerge(
                "shrink-0 text-slate-500 transition-transform duration-150",
                open && "rotate-90",
              )}
            />
          </div>
        ) : (
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <RowIcon icon={props.icon} />
            {text}
          </div>
        )}
        {!props.isSet && props.onAdd && (
          <AddButton
            label="Add"
            ariaLabel={`Add ${props.title}`}
            onClick={() => {
              props.onAdd?.();
              setOpen(true);
            }}
          />
        )}
        {props.isSet && props.onRemove && (
          <Tooltip label="Remove" withArrow>
            <ActionIcon
              type="button"
              variant="subtle"
              color="red"
              size="sm"
              aria-label={`Remove ${props.title}`}
              onClick={() => {
                props.onRemove?.();
                setOpen(false);
              }}
            >
              <Trash2 size={13} strokeWidth={2.1} />
            </ActionIcon>
          </Tooltip>
        )}
      </div>
      {props.isSet && open && (
        <div className="flex min-w-0 flex-col gap-4 border-t border-slate-100 px-3.5 py-4">
          {props.children}
        </div>
      )}
    </div>
  );
};

export const ItemCard = (props: {
  icon?: LucideIcon;
  title: string;
  subtitle?: React.ReactNode;
  badges?: React.ReactNode;
  defaultOpen?: boolean;
  removeLabel: string;
  onRemove: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  children: React.ReactNode;
}) => {
  const [open, setOpen] = React.useState(!!props.defaultOpen);
  return (
    <div className="min-w-0 rounded-lg border border-slate-200 bg-white">
      <div className="flex min-w-0 items-center gap-1.5 px-3 py-2">
        <div
          {...pressable(() => setOpen((value) => !value))}
          aria-expanded={open}
          aria-label={props.title}
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-2.5 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <ChevronRight
            size={14}
            strokeWidth={2.25}
            className={twMerge(
              "shrink-0 text-slate-500 transition-transform duration-150",
              open && "rotate-90",
            )}
          />
          {props.icon && <RowIcon icon={props.icon} />}
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
              <span className="truncate text-body font-semibold text-slate-900">
                {props.title}
              </span>
              {props.badges}
            </span>
            {props.subtitle && (
              <span className="truncate text-xs font-normal leading-5 text-slate-500">
                {props.subtitle}
              </span>
            )}
          </span>
        </div>
        {props.onMoveUp && (
          <ActionIcon
            type="button"
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={`Move ${props.title} up`}
            onClick={props.onMoveUp}
          >
            <ArrowUp size={13} strokeWidth={2.2} />
          </ActionIcon>
        )}
        {props.onMoveDown && (
          <ActionIcon
            type="button"
            variant="subtle"
            color="gray"
            size="sm"
            aria-label={`Move ${props.title} down`}
            onClick={props.onMoveDown}
          >
            <ArrowDown size={13} strokeWidth={2.2} />
          </ActionIcon>
        )}
        <Tooltip label="Remove" withArrow>
          <ActionIcon
            type="button"
            variant="subtle"
            color="red"
            size="sm"
            aria-label={props.removeLabel}
            onClick={props.onRemove}
          >
            <Trash2 size={13} strokeWidth={2.1} />
          </ActionIcon>
        </Tooltip>
      </div>
      {open && (
        <div className="flex min-w-0 flex-col gap-4 border-t border-slate-100 px-3 py-4">
          {props.children}
        </div>
      )}
    </div>
  );
};

export const plural = (count: number, noun: string, many?: string) =>
  `${count} ${count === 1 ? noun : (many ?? `${noun}s`)}`;

export const listSummary = (values: string[], max = 3) => {
  const items = values.filter(Boolean);
  if (items.length <= max) return items.join(", ");
  return `${items.slice(0, max).join(", ")} +${items.length - max}`;
};

export const joinSummary = (...values: (string | false | undefined)[]) =>
  values.filter(Boolean).join(" · ");
