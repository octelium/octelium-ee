import * as CoreP from "@/apis/corev1/corev1";
import {
  Condition_Expression_APIServerAccess_Service as AccessService,
  Condition_Expression_APIServerCordium_Service as CordiumService,
  Condition_Expression_APIServerEnterprise_Service as EnterpriseService,
  Condition_Expression_MCPToolArgument_BoolMatch_Value as MCPBoolValue,
  Condition_Expression_StringSetMatch as StringSetMatch,
  Condition_Expression_StringSetMatch_In as StringSetMatchIn,
  Condition_Expression_TimeDayType_Type as TimeDayType,
  Condition_Expression as Expression,
} from "@/apis/enterprisev1/enterprisev1";
import { ObjectReference } from "@/apis/metav1/metav1";
import { getOSTypeStr } from "@/pages/core/Device/Main";
import {
  getResourceRef,
  printResourceNameWithDisplay,
  printServiceMode,
} from "@/utils/pb";
import {
  Autocomplete,
  Group,
  SegmentedControl,
  Select,
  Switch,
  TagsInput,
  TextInput,
} from "@mantine/core";
import type { ReactNode } from "react";
import { match } from "ts-pattern";
import SelectResource from "../ResourceLayout/SelectResource";
import { useResourceFromRef } from "../ResourceLayout/utils";
import SelectCountry, { CountryFlag } from "../SelectCountry";
import TimeAgo from "../TimeAgo";
import TimestampPicker from "../TimestampPicker";

const ResourceValue = (props: { itemRef: ObjectReference }) => {
  const { data } = useResourceFromRef(props.itemRef);
  return <>{data ? printResourceNameWithDisplay(data) : null}</>;
};

type EditProps = { item?: Expression; onUpdate: (item?: Expression) => void };
type ValueProps = { item: Expression };

export type ItemDef = {
  type: string;
  title: string;
  tags: string[];
  makeDefault: () => Expression;
  components: {
    Value: (props: ValueProps) => ReactNode;
    Edit: (props: EditProps) => ReactNode;
  };
};

const makeResourceItem = (
  type: string,
  title: string,
  tags: string[],
  api: "core" | "enterprise",
  kind: string,
  refKey: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: {} } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      const ref = (item.type as any)[type][refKey] as
        | ObjectReference
        | undefined;
      return ref ? <ResourceValue itemRef={ref} /> : null;
    },
    Edit: ({ item, onUpdate }) => {
      const ref =
        item?.type.oneofKind === type
          ? ((item.type as any)[type][refKey] as ObjectReference | undefined)
          : undefined;
      return (
        <SelectResource
          api={api}
          kind={kind}
          defaultValue={ref?.name}
          onChange={(v) => {
            if (!v) {
              onUpdate();
              return;
            }
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: type,
                  [type]: { [refKey]: getResourceRef(v) },
                } as any,
              }),
            );
          }}
        />
      );
    },
  },
});

const makeMarkerItem = (
  type: string,
  title: string,
  tags: string[],
  note?: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: {} } as any,
    }),
  components: {
    Value: () => null,
    Edit: () => (
      <p className="text-[0.75rem] font-semibold text-slate-500 leading-relaxed">
        {note ??
          "This condition matches on presence. No further configuration is required."}
      </p>
    ),
  },
});

const makeBoolItem = (
  type: string,
  title: string,
  tags: string[],
  boolKey: string,
  switchLabel: string,
  defaultValue = false,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { [boolKey]: defaultValue } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{(item.type as any)[type][boolKey] ? switchLabel : "Any"}</>;
    },
    Edit: ({ item, onUpdate }) => (
      <Switch
        size="md"
        label={switchLabel}
        checked={
          item?.type.oneofKind === type
            ? (item.type as any)[type][boolKey]
            : defaultValue
        }
        onChange={(e) =>
          onUpdate(
            Expression.create({
              type: {
                oneofKind: type,
                [type]: { [boolKey]: e.currentTarget.checked },
              } as any,
            }),
          )
        }
      />
    ),
  },
});

const makeTextItem = (
  type: string,
  title: string,
  tags: string[],
  valueKey: string,
  label: string,
  placeholder: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { [valueKey]: "" } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{(item.type as any)[type][valueKey]}</>;
    },
    Edit: ({ item, onUpdate }) => (
      <TextInput
        label={label}
        placeholder={placeholder}
        value={
          item?.type.oneofKind === type
            ? (item.type as any)[type][valueKey]
            : ""
        }
        onChange={(e) =>
          onUpdate(
            Expression.create({
              type: {
                oneofKind: type,
                [type]: { [valueKey]: e.target.value },
              } as any,
            }),
          )
        }
      />
    ),
  },
});

const STRING_MATCH_OPTIONS = [
  { value: "exact", label: "Exact" },
  { value: "prefix", label: "Prefix" },
  { value: "suffix", label: "Suffix" },
  { value: "contains", label: "Contains" },
  { value: "in", label: "One of" },
];

const MCP_PROTOCOL_VERSIONS = [
  "2024-11-05",
  "2025-03-26",
  "2025-06-18",
  "2025-11-25",
  "2026-07-28",
];

const MCP_METHODS = [
  "server/discover",
  "initialize",
  "ping",
  "tools/list",
  "tools/call",
  "prompts/list",
  "prompts/get",
  "resources/list",
  "resources/read",
  "resources/templates/list",
  "resources/subscribe",
  "resources/unsubscribe",
  "completion/complete",
  "subscriptions/listen",
  "elicitation/create",
  "roots/list",
  "sampling/createMessage",
  "logging/setLevel",
  "tasks/get",
  "tasks/update",
  "tasks/cancel",
  "notifications/initialized",
  "notifications/cancelled",
  "notifications/progress",
  "notifications/message",
  "notifications/roots/list_changed",
  "notifications/tools/list_changed",
  "notifications/prompts/list_changed",
  "notifications/resources/list_changed",
  "notifications/resources/updated",
  "notifications/subscriptions/acknowledged",
  "notifications/tasks",
];

const COMMON_TIMEZONES = [
  "UTC",
  "America/Los_Angeles",
  "America/Denver",
  "America/Chicago",
  "America/New_York",
  "America/Sao_Paulo",
  "America/Toronto",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Europe/Moscow",
  "Africa/Cairo",
  "Asia/Dubai",
  "Asia/Kolkata",
  "Asia/Singapore",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Asia/Seoul",
  "Australia/Sydney",
  "Pacific/Auckland",
];

const TIMEZONES =
  typeof Intl !== "undefined" && typeof Intl.supportedValuesOf === "function"
    ? Array.from(new Set(["UTC", ...Intl.supportedValuesOf("timeZone")]))
    : COMMON_TIMEZONES;

const NUMERIC_MATCH_OPTIONS = [
  { value: "exact", label: "Equals" },
  { value: "lessThan", label: "<" },
  { value: "lessThanOrEqual", label: "≤" },
  { value: "greaterThan", label: ">" },
  { value: "greaterThanOrEqual", label: "≥" },
];

const formatEnumLabel = (enumObj: Record<string, string | number>, value: number) => {
  const key = enumObj[value];
  if (typeof key !== "string") return "Unset";
  return key
    .replace(/^(TYPE|PROTOCOL|OPERATION|ESTIMATE_QUALITY|ADDRESS_TYPE)_/, "")
    .replace(/_UNSET$/, "")
    .replaceAll("_", " ")
    .toLowerCase()
    .replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()) || "Unset";
};

const makeStringMatch = (kind: string, value: string | string[]): any => ({
  type: {
    oneofKind: kind,
    [kind]: kind === "in" ? { values: value as string[] } : value as string,
  },
});

const stringMatchText = (match?: any): string => {
  const kind = match?.type?.oneofKind;
  if (!kind) return "Any string";
  if (kind === "in") return (match.type.in.values ?? []).join(", ") || "One of (none)";
  return `${kind === "exact" ? "" : `${kind} `}${match.type[kind] ?? ""}`;
};

const StringMatchEditor = (props: {
  value?: any;
  label?: string;
  placeholder?: string;
  data?: string[];
  options?: { value: string; label: string }[];
  onChange: (value: any) => void;
}) => {
  const kind = props.value?.type?.oneofKind ?? "exact";
  const options = props.options ?? STRING_MATCH_OPTIONS;
  const current = kind === "in"
    ? (props.value?.type?.in?.values ?? [])
    : (props.value?.type?.[kind] ?? "");
  return (
    <div className="space-y-2">
      <SegmentedControl
        fullWidth
        size="xs"
        value={kind}
        data={options}
        onChange={(next) =>
          props.onChange(makeStringMatch(next, next === "in" ? [] : ""))
        }
      />
      {kind === "in" ? (
        <TagsInput
          label={props.label ?? "Values"}
          placeholder="Add an allowed value"
          value={current}
          onChange={(values) => props.onChange(makeStringMatch(kind, values))}
        />
      ) : (
        props.data?.length && kind === "exact" ? (
          <Autocomplete
            label={props.label ?? "Value"}
            description="Choose a suggested value or enter a custom value"
            placeholder={props.placeholder ?? "Enter a value"}
            data={props.data}
            value={current}
            onChange={(value) => props.onChange(makeStringMatch(kind, value))}
          />
        ) : (
          <TextInput
            label={props.label ?? "Value"}
            placeholder={props.placeholder ?? "Enter a value"}
            value={current}
            onChange={(event) =>
              props.onChange(makeStringMatch(kind, event.currentTarget.value))
            }
          />
        )
      )}
    </div>
  );
};

const makeStringMatchItem = (
  type: string,
  title: string,
  tags: string[],
  label = "Match value",
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { match: makeStringMatch("exact", "") } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{stringMatchText((item.type as any)[type].match)}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const matchValue =
        item?.type.oneofKind === type
          ? (item.type as any)[type].match
          : undefined;
      return (
        <StringMatchEditor
          label={label}
          value={matchValue}
          onChange={(match) =>
            onUpdate(
              Expression.create({
                type: { oneofKind: type, [type]: { match } } as any,
              }),
            )
          }
        />
      );
    },
  },
});

const makeMCPStringMatchItem = (
  type: string,
  title: string,
  tags: string[],
  label: string,
  defaultValue: string,
  suggestions: string[],
  placeholder: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: {
        oneofKind: type,
        [type]: { match: makeStringMatch("exact", defaultValue) },
      } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{stringMatchText((item.type as any)[type].match)}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const matchValue =
        item?.type.oneofKind === type
          ? (item.type as any)[type].match
          : makeStringMatch("exact", defaultValue);
      return (
        <StringMatchEditor
          label={label}
          placeholder={placeholder}
          data={suggestions}
          value={matchValue}
          onChange={(match) =>
            onUpdate(
              Expression.create({
                type: { oneofKind: type, [type]: { match } } as any,
              }),
            )
          }
        />
      );
    },
  },
});

const makeNumericMatch = (kind: string, value: number) => ({
  type: { oneofKind: kind, [kind]: value },
});

const numericMatchText = (match?: any): string => {
  const kind = match?.type?.oneofKind;
  if (!kind) return "Any number";
  const symbol: Record<string, string> = {
    exact: "=",
    lessThan: "<",
    lessThanOrEqual: "≤",
    greaterThan: ">",
    greaterThanOrEqual: "≥",
  };
  return `${symbol[kind] ?? kind} ${match.type[kind] ?? 0}`;
};

const NumericMatchEditor = (props: {
  value?: any;
  label?: string;
  onChange: (value: any) => void;
}) => {
  const kind = props.value?.type?.oneofKind ?? "exact";
  const current = props.value?.type?.[kind] ?? 0;
  return (
    <div className="space-y-2">
      <SegmentedControl
        fullWidth
        size="xs"
        value={kind}
        data={NUMERIC_MATCH_OPTIONS}
        onChange={(next) => props.onChange(makeNumericMatch(next, 0))}
      />
      <TextInput
        label={props.label ?? "Value"}
        type="number"
        inputMode="numeric"
        value={String(current)}
        onChange={(event) =>
          props.onChange(
            makeNumericMatch(kind, Number(event.currentTarget.value) || 0),
          )
        }
      />
    </div>
  );
};

const makeNumericMatchItem = (
  type: string,
  title: string,
  tags: string[],
  label = "Value",
  extraKey?: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: {
        oneofKind: type,
        [type]: { match: makeNumericMatch("exact", 0), ...(extraKey ? { [extraKey]: false } : {}) },
      } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      const value = (item.type as any)[type];
      return <>{numericMatchText(value.match)}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const value =
        item?.type.oneofKind === type ? (item.type as any)[type] : undefined;
      const emit = (match: any, extra = value?.[extraKey ?? ""]) =>
        onUpdate(
          Expression.create({
            type: {
              oneofKind: type,
              [type]: { match, ...(extraKey ? { [extraKey]: extra } : {}) },
            } as any,
          }),
        );
      return (
        <div className="space-y-3">
          <NumericMatchEditor
            label={label}
            value={value?.match}
            onChange={(match) => emit(match)}
          />
          {extraKey === "requireComplete" && (
            <Switch
              label="Require a complete estimate"
              checked={Boolean(value?.requireComplete)}
              onChange={(event) => emit(value?.match ?? makeNumericMatch("exact", 0), event.currentTarget.checked)}
            />
          )}
        </div>
      );
    },
  },
});

const MCP_ARGUMENT_MATCH_OPTIONS = [
  { value: "stringMatch", label: "String" },
  { value: "doubleMatch", label: "Number" },
  { value: "boolMatch", label: "Boolean" },
  { value: "isNull", label: "Null" },
  { value: "exists", label: "Exists" },
];

const DOUBLE_MATCH_OPTIONS = [
  { value: "lessThan", label: "<" },
  { value: "lessThanOrEqual", label: "≤" },
  { value: "greaterThan", label: ">" },
  { value: "greaterThanOrEqual", label: "≥" },
];

const makeDoubleMatch = (kind: string, value: number) => ({
  type: { oneofKind: kind, [kind]: value },
});

const mcpArgumentMatchText = (match?: any): string => {
  switch (match?.oneofKind) {
    case "stringMatch":
      return `string ${stringMatchText(match.stringMatch)}`;
    case "doubleMatch": {
      const type = match.doubleMatch?.type;
      if (!type?.oneofKind) return "number (any)";
      return `number ${type.oneofKind.replace(/([A-Z])/g, " $1").toLowerCase()} ${type[type.oneofKind]}`;
    }
    case "boolMatch":
      return match.boolMatch?.value === MCPBoolValue.TRUE
        ? "boolean true"
        : match.boolMatch?.value === MCPBoolValue.FALSE
          ? "boolean false"
          : "boolean";
    case "isNull":
      return "is null";
    case "exists":
      return "exists";
    default:
      return "any value";
  }
};

const MCPToolArgumentEditor = (props: {
  value?: any;
  onChange: (value: any) => void;
}) => {
  const path = props.value?.path ?? [];
  const match = props.value?.match;
  const matchType = match?.oneofKind ?? "stringMatch";

  const updateMatch = (next: any) =>
    props.onChange({ path, match: next });

  return (
    <div className="space-y-4">
      <TagsInput
        label="Argument path"
        description="Keys relative to params.arguments. Add one key per path segment."
        placeholder={path.length ? "Add another nested key" : "e.g. query"}
        value={path}
        onChange={(next) => props.onChange({ path: next, match })}
      />
      <div className="space-y-2">
        <p className="text-[0.72rem] font-bold text-slate-700">Value rule</p>
        <SegmentedControl
          fullWidth
          size="sm"
          value={matchType}
          data={MCP_ARGUMENT_MATCH_OPTIONS}
          onChange={(next) => {
            if (next === "stringMatch") updateMatch({
              oneofKind: next,
              stringMatch: { type: { oneofKind: "exact", exact: "" } },
            });
            else if (next === "doubleMatch") updateMatch({
              oneofKind: next,
              doubleMatch: makeDoubleMatch("lessThan", 0),
            });
            else if (next === "boolMatch") updateMatch({
              oneofKind: next,
              boolMatch: { value: MCPBoolValue.TRUE },
            });
            else updateMatch({ oneofKind: next, [next]: {} });
          }}
        />
      </div>

      {matchType === "stringMatch" && (
        <StringMatchEditor
          label="String value"
          placeholder="e.g. weather"
          value={match?.stringMatch}
          onChange={(next) => updateMatch({ oneofKind: "stringMatch", stringMatch: next })}
        />
      )}
      {matchType === "doubleMatch" && (() => {
        const current = match?.doubleMatch?.type?.oneofKind ?? "lessThan";
        const value = match?.doubleMatch?.type?.[current] ?? 0;
        return (
          <div className="space-y-2">
            <SegmentedControl
              fullWidth
              size="xs"
              value={current}
              data={DOUBLE_MATCH_OPTIONS}
              onChange={(next) => updateMatch({
                oneofKind: "doubleMatch",
                doubleMatch: makeDoubleMatch(next, 0),
              })}
            />
            <TextInput
              label="Number"
              type="number"
              step="any"
              value={String(value)}
              onChange={(event) => updateMatch({
                oneofKind: "doubleMatch",
                doubleMatch: makeDoubleMatch(current, Number(event.currentTarget.value) || 0),
              })}
            />
          </div>
        );
      })()}
      {matchType === "boolMatch" && (
        <SegmentedControl
          fullWidth
          size="sm"
          value={String(match?.boolMatch?.value ?? MCPBoolValue.TRUE)}
          data={[
            { value: String(MCPBoolValue.TRUE), label: "True" },
            { value: String(MCPBoolValue.FALSE), label: "False" },
          ]}
          onChange={(next) => updateMatch({
            oneofKind: "boolMatch",
            boolMatch: { value: Number(next) },
          })}
        />
      )}
      {(matchType === "isNull" || matchType === "exists") && (
        <p className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-[0.7rem] font-semibold text-slate-500">
          {matchType === "isNull"
            ? "Matches only when the argument is explicitly null."
            : "Matches when the argument path exists, including when its value is null."}
        </p>
      )}
    </div>
  );
};

const makeMCPToolArgumentItem = (): ItemDef => ({
  type: "requestMCPToolArgument",
  title: "MCP tool argument",
  tags: ["mcp", "request", "tool", "argument", "json"],
  makeDefault: () =>
    Expression.create({
      type: {
        oneofKind: "requestMCPToolArgument",
        requestMCPToolArgument: {
          path: [],
          match: {
            oneofKind: "stringMatch",
            stringMatch: { type: { oneofKind: "exact", exact: "" } },
          },
        },
      } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== "requestMCPToolArgument") return null;
      const value = (item.type as any).requestMCPToolArgument;
      const path = value.path?.length ? value.path.join(".") : "(argument path)";
      return <>{path} · {mcpArgumentMatchText(value.match)}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const value =
        item?.type.oneofKind === "requestMCPToolArgument"
          ? (item.type as any).requestMCPToolArgument
          : undefined;
      return (
        <MCPToolArgumentEditor
          value={value}
          onChange={(next) =>
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "requestMCPToolArgument",
                  requestMCPToolArgument: next,
                } as any,
              }),
            )
          }
        />
      );
    },
  },
});

const makeEnumItem = (
  type: string,
  title: string,
  tags: string[],
  field: string,
  enumObj: Record<string, number> & Record<number, string>,
  values: number[],
  label: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { [field]: values[0] } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      return <>{formatEnumLabel(enumObj, (item.type as any)[type][field])}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const current =
        item?.type.oneofKind === type
          ? (item.type as any)[type][field]
          : values[0];
      return (
        <Select
          label={label}
          data={values.map((value) => ({
            value: String(value),
            label: formatEnumLabel(enumObj, value),
          }))}
          value={String(current)}
          onChange={(next) =>
            next &&
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: type,
                  [type]: { [field]: Number(next) },
                } as any,
              }),
            )
          }
        />
      );
    },
  },
});

const makeStringListItem = (
  type: string,
  title: string,
  tags: string[],
  listKey: string,
  label: string,
  placeholder: string,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { [listKey]: [] } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      const arr = ((item.type as any)[type][listKey] as string[]) ?? [];
      return <>{arr.join(", ")}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const arr =
        item?.type.oneofKind === type
          ? ((item.type as any)[type][listKey] as string[])
          : [];
      return (
        <TagsInput
          label={label}
          placeholder={placeholder}
          value={arr}
          onChange={(vals) =>
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: type,
                  [type]: { [listKey]: vals },
                } as any,
              }),
            )
          }
        />
      );
    },
  },
});

const makeApiServiceItem = (
  type: string,
  title: string,
  tags: string[],
  enumObj: any,
  values: number[],
  labels: Record<number, string>,
  defaultValue: number,
): ItemDef => ({
  type,
  title,
  tags,
  makeDefault: () =>
    Expression.create({
      type: { oneofKind: type, [type]: { service: defaultValue } } as any,
    }),
  components: {
    Value: ({ item }) => {
      if (item.type.oneofKind !== type) return null;
      const s = (item.type as any)[type].service as number;
      return <>{labels[s] ?? ""}</>;
    },
    Edit: ({ item, onUpdate }) => {
      const cur =
        item?.type.oneofKind === type
          ? ((item.type as any)[type].service as number)
          : defaultValue;
      return (
        <Select
          label="API service"
          data={values.map((v) => ({ value: enumObj[v], label: labels[v] }))}
          value={enumObj[cur]}
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: type,
                  [type]: { service: enumObj[v] },
                } as any,
              }),
            );
          }}
        />
      );
    },
  },
});

export const itemList: ItemDef[] = [
  makeResourceItem(
    "user",
    "User",
    ["identity", "user"],
    "core",
    "User",
    "userRef",
  ),
  makeResourceItem(
    "session",
    "Session",
    ["session", "access"],
    "core",
    "Session",
    "sessionRef",
  ),
  makeResourceItem(
    "device",
    "Device",
    ["device", "endpoint"],
    "core",
    "Device",
    "deviceRef",
  ),
  makeResourceItem(
    "service",
    "Service",
    ["service", "resource"],
    "core",
    "Service",
    "serviceRef",
  ),
  makeResourceItem(
    "group",
    "User belongs to group",
    ["identity", "group", "user"],
    "core",
    "Group",
    "groupRef",
  ),
  makeResourceItem(
    "namespace",
    "Namespace",
    ["resource", "namespace"],
    "core",
    "Namespace",
    "namespaceRef",
  ),
  makeResourceItem(
    "sessionAuthenticationCredential",
    "Session credential",
    ["session", "authentication", "credential"],
    "core",
    "Credential",
    "credentialRef",
  ),
  makeResourceItem(
    "sessionAuthenticationIdentityProvider",
    "Session identity provider",
    ["session", "authentication", "identity provider", "sso"],
    "core",
    "IdentityProvider",
    "identityProviderRef",
  ),

  {
    type: "userType",
    title: "User type",
    tags: ["identity", "user", "type"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "userType",
          userType: { type: CoreP.User_Spec_Type.HUMAN },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "userType") return null;
        return (
          <>
            {match(item.type.userType.type)
              .with(CoreP.User_Spec_Type.HUMAN, () => "Human")
              .with(CoreP.User_Spec_Type.WORKLOAD, () => "Workload")
              .otherwise(() => "")}
          </>
        );
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="User type"
          data={[
            {
              label: "Human",
              value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.HUMAN],
            },
            {
              label: "Workload",
              value: CoreP.User_Spec_Type[CoreP.User_Spec_Type.WORKLOAD],
            },
          ]}
          value={
            item?.type.oneofKind === "userType"
              ? CoreP.User_Spec_Type[item.type.userType.type]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "userType",
                  userType: { type: CoreP.User_Spec_Type[v as "HUMAN"] },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  {
    type: "deviceOSType",
    title: "Device OS type",
    tags: ["device", "os", "platform", "endpoint"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "deviceOSType",
          deviceOSType: { osType: CoreP.Device_Status_OSType.LINUX },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "deviceOSType") return null;
        return <>{getOSTypeStr(item.type.deviceOSType.osType)}</>;
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Device OS type"
          data={[
            CoreP.Device_Status_OSType.LINUX,
            CoreP.Device_Status_OSType.WINDOWS,
            CoreP.Device_Status_OSType.MAC,
            CoreP.Device_Status_OSType.ANDROID,
          ].map((v) => ({
            label: getOSTypeStr(v),
            value: CoreP.Device_Status_OSType[v],
          }))}
          value={
            item?.type.oneofKind === "deviceOSType"
              ? CoreP.Device_Status_OSType[item.type.deviceOSType.osType]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "deviceOSType",
                  deviceOSType: {
                    osType: CoreP.Device_Status_OSType[v as "LINUX"],
                  },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  {
    type: "serviceMode",
    title: "Service mode",
    tags: ["service", "protocol", "mode", "http", "tcp", "ssh"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "serviceMode",
          serviceMode: { mode: CoreP.Service_Spec_Mode.HTTP },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "serviceMode") return null;
        return <>{printServiceMode(item.type.serviceMode.mode)}</>;
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Service mode"
          data={[
            CoreP.Service_Spec_Mode.HTTP,
            CoreP.Service_Spec_Mode.TCP,
            CoreP.Service_Spec_Mode.SSH,
            CoreP.Service_Spec_Mode.WEB,
            CoreP.Service_Spec_Mode.KUBERNETES,
            CoreP.Service_Spec_Mode.POSTGRES,
            CoreP.Service_Spec_Mode.MYSQL,
            CoreP.Service_Spec_Mode.UDP,
            CoreP.Service_Spec_Mode.GRPC,
            CoreP.Service_Spec_Mode.DNS,
          ].map((v) => ({
            label: printServiceMode(v),
            value: CoreP.Service_Spec_Mode[v],
          }))}
          value={
            item?.type.oneofKind === "serviceMode"
              ? CoreP.Service_Spec_Mode[item.type.serviceMode.mode]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "serviceMode",
                  serviceMode: { mode: CoreP.Service_Spec_Mode[v as "HTTP"] },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  makeMarkerItem(
    "servicePublic",
    "Public Service",
    ["service", "public", "beyondcorp"],
    "Matches when the target Service is public. No further configuration is required.",
  ),

  {
    type: "sessionType",
    title: "Session type",
    tags: ["session", "type", "client", "clientless"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionType",
          sessionType: { type: CoreP.Session_Status_Type.CLIENT },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionType") return null;
        return (
          <>
            {match(item.type.sessionType.type)
              .with(CoreP.Session_Status_Type.CLIENT, () => "Client")
              .with(CoreP.Session_Status_Type.CLIENTLESS, () => "Clientless")
              .otherwise(() => "")}
          </>
        );
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Session type"
          data={[
            {
              label: "Client",
              value:
                CoreP.Session_Status_Type[CoreP.Session_Status_Type.CLIENT],
            },
            {
              label: "Clientless",
              value:
                CoreP.Session_Status_Type[CoreP.Session_Status_Type.CLIENTLESS],
            },
          ]}
          value={
            item?.type.oneofKind === "sessionType"
              ? CoreP.Session_Status_Type[item.type.sessionType.type]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "sessionType",
                  sessionType: {
                    type: CoreP.Session_Status_Type[v as "CLIENT"],
                  },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  makeMarkerItem(
    "sessionBrowser",
    "Browser session",
    ["session", "browser", "web"],
    "Matches when the session originates from a browser. No further configuration is required.",
  ),

  {
    type: "sessionAuthenticationAAL",
    title: "Session authentication assurance level (AAL)",
    tags: ["session", "authentication", "aal", "assurance", "mfa"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionAuthenticationAAL",
          sessionAuthenticationAAL: {
            aal: CoreP.Session_Status_Authentication_Info_AAL.AAL1,
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationAAL") return null;
        return (
          <>
            {match(item.type.sessionAuthenticationAAL.aal)
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL1,
                () => "AAL1",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL2,
                () => "AAL2",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_AAL.AAL3,
                () => "AAL3",
              )
              .otherwise(() => "")}
          </>
        );
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Assurance level"
          data={["AAL1", "AAL2", "AAL3"].map((v) => ({ label: v, value: v }))}
          value={
            item?.type.oneofKind === "sessionAuthenticationAAL"
              ? CoreP.Session_Status_Authentication_Info_AAL[
                  item.type.sessionAuthenticationAAL.aal
                ]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "sessionAuthenticationAAL",
                  sessionAuthenticationAAL: {
                    aal: CoreP.Session_Status_Authentication_Info_AAL[
                      v as "AAL1"
                    ],
                  },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationType",
    title: "Session authentication type",
    tags: ["session", "authentication", "type", "credential", "oauth", "idp"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionAuthenticationType",
          sessionAuthenticationType: {
            type: CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR,
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationType") return null;
        return (
          <>
            {match(item.type.sessionAuthenticationType.type)
              .with(
                CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR,
                () => "Authenticator",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL,
                () => "Credential",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.IDENTITY_PROVIDER,
                () => "Identity Provider",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.REFRESH_TOKEN,
                () => "Refresh Token",
              )
              .with(
                CoreP.Session_Status_Authentication_Info_Type.INTERNAL,
                () => "Internal",
              )
              .otherwise(() => "")}
          </>
        );
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Authentication type"
          data={[
            {
              label: "Authenticator",
              value:
                CoreP.Session_Status_Authentication_Info_Type[
                  CoreP.Session_Status_Authentication_Info_Type.AUTHENTICATOR
                ],
            },
            {
              label: "Credential",
              value:
                CoreP.Session_Status_Authentication_Info_Type[
                  CoreP.Session_Status_Authentication_Info_Type.CREDENTIAL
                ],
            },
            {
              label: "Identity Provider",
              value:
                CoreP.Session_Status_Authentication_Info_Type[
                  CoreP.Session_Status_Authentication_Info_Type
                    .IDENTITY_PROVIDER
                ],
            },
            {
              label: "Internal",
              value:
                CoreP.Session_Status_Authentication_Info_Type[
                  CoreP.Session_Status_Authentication_Info_Type.INTERNAL
                ],
            },
            {
              label: "Refresh Token",
              value:
                CoreP.Session_Status_Authentication_Info_Type[
                  CoreP.Session_Status_Authentication_Info_Type.REFRESH_TOKEN
                ],
            },
          ]}
          value={
            item?.type.oneofKind === "sessionAuthenticationType"
              ? CoreP.Session_Status_Authentication_Info_Type[
                  item.type.sessionAuthenticationType.type
                ]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "sessionAuthenticationType",
                  sessionAuthenticationType: {
                    type: CoreP.Session_Status_Authentication_Info_Type[
                      v as "CREDENTIAL"
                    ],
                  },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationCredentialType",
    title: "Session credential type",
    tags: ["session", "authentication", "credential", "oauth2", "token"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionAuthenticationCredentialType",
          sessionAuthenticationCredentialType: {
            type: CoreP.Credential_Spec_Type.ACCESS_TOKEN,
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationCredentialType")
          return null;
        return (
          <>
            {match(item.type.sessionAuthenticationCredentialType.type)
              .with(
                CoreP.Credential_Spec_Type.ACCESS_TOKEN,
                () => "Access Token",
              )
              .with(
                CoreP.Credential_Spec_Type.AUTH_TOKEN,
                () => "Authentication Token",
              )
              .with(
                CoreP.Credential_Spec_Type.OAUTH2,
                () => "OAuth2 Client Credentials",
              )
              .otherwise(() => "")}
          </>
        );
      },
      Edit: ({ item, onUpdate }) => (
        <Select
          label="Credential type"
          data={[
            {
              label: "Access Token",
              value:
                CoreP.Credential_Spec_Type[
                  CoreP.Credential_Spec_Type.ACCESS_TOKEN
                ],
            },
            {
              label: "Authentication Token",
              value:
                CoreP.Credential_Spec_Type[
                  CoreP.Credential_Spec_Type.AUTH_TOKEN
                ],
            },
            {
              label: "OAuth2 Client Credentials",
              value:
                CoreP.Credential_Spec_Type[CoreP.Credential_Spec_Type.OAUTH2],
            },
          ]}
          value={
            item?.type.oneofKind === "sessionAuthenticationCredentialType"
              ? CoreP.Credential_Spec_Type[
                  item.type.sessionAuthenticationCredentialType.type
                ]
              : null
          }
          onChange={(v) => {
            if (!v) return;
            onUpdate(
              Expression.create({
                type: {
                  oneofKind: "sessionAuthenticationCredentialType",
                  sessionAuthenticationCredentialType: {
                    type: CoreP.Credential_Spec_Type[v as "OAUTH2"],
                  },
                },
              }),
            );
          }}
        />
      ),
    },
  },

  {
    type: "sessionAuthenticationGeoipCountryCode",
    title: "Session country",
    tags: ["session", "geo", "country", "location", "ip"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionAuthenticationGeoipCountryCode",
          sessionAuthenticationGeoipCountryCode: {
            match: StringSetMatch.create({
              type: { oneofKind: "exact", exact: "" },
            }),
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationGeoipCountryCode")
          return null;
        const countryMatch =
          item.type.sessionAuthenticationGeoipCountryCode.match;
        if (countryMatch?.type.oneofKind === "in") {
          return (
            <span className="flex flex-wrap items-center gap-1.5">
              {countryMatch.type.in.values.map((code) => (
                <span
                  key={code}
                  className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-[0.67rem] font-bold text-slate-600"
                >
                  <CountryFlag code={code} />
                  {code}
                </span>
              ))}
            </span>
          );
        }

        const code =
          countryMatch?.type.oneofKind === "exact"
            ? countryMatch.type.exact
            : "";
        return code ? (
          <span className="inline-flex items-center gap-1.5">
            <CountryFlag code={code} />
            {code}
          </span>
        ) : (
          <>Any country</>
        );
      },
      Edit: ({ item, onUpdate }) => {
        const countryMatch =
          item?.type.oneofKind === "sessionAuthenticationGeoipCountryCode"
            ? item.type.sessionAuthenticationGeoipCountryCode.match
            : undefined;
        const matchType =
          countryMatch?.type.oneofKind === "in" ? "in" : "exact";
        const exactValue =
          countryMatch?.type.oneofKind === "exact"
            ? countryMatch.type.exact
            : "";
        const allowedValues =
          countryMatch?.type.oneofKind === "in"
            ? countryMatch.type.in.values
            : [];

        const updateMatch = (nextType: "exact" | "in", value: string | string[]) => {
          const nextMatch =
            nextType === "in"
              ? StringSetMatch.create({
                  type: {
                    oneofKind: "in",
                    in: StringSetMatchIn.create({
                      values: Array.isArray(value) ? value : value ? [value] : [],
                    }),
                  },
                })
              : StringSetMatch.create({
                  type: {
                    oneofKind: "exact",
                    exact: Array.isArray(value) ? value[0] ?? "" : value,
                  },
                });

          onUpdate(
            Expression.create({
              type: {
                oneofKind: "sessionAuthenticationGeoipCountryCode",
                sessionAuthenticationGeoipCountryCode: { match: nextMatch },
              },
            }),
          );
        };

        return (
          <div className="space-y-2">
            <SegmentedControl
              fullWidth
              size="xs"
              value={matchType}
              data={[
                { value: "exact", label: "Exact country" },
                { value: "in", label: "In list" },
              ]}
              onChange={(next) =>
                updateMatch(
                  next as "exact" | "in",
                  next === "in"
                    ? exactValue
                      ? [exactValue]
                      : allowedValues
                    : allowedValues[0] ?? exactValue,
                )
              }
            />
            {matchType === "in" ? (
              <SelectCountry
                multiple
                values={allowedValues}
                onUpdate={(next) =>
                  updateMatch("in", Array.isArray(next) ? next : [])
                }
              />
            ) : (
              <SelectCountry
                val={exactValue}
                onUpdate={(next) =>
                  updateMatch("exact", typeof next === "string" ? next : "")
                }
              />
            )}
          </div>
        );
      },
    },
  },

  {
    type: "sessionAuthenticationGeoipContinentCode",
    title: "Session continent",
    tags: ["session", "geo", "continent", "location", "ip"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "sessionAuthenticationGeoipContinentCode",
          sessionAuthenticationGeoipContinentCode: {
            match: makeStringMatch("exact", ""),
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "sessionAuthenticationGeoipContinentCode")
          return null;
        return (
          <>{stringMatchText(item.type.sessionAuthenticationGeoipContinentCode.match)}</>
        );
      },
      Edit: ({ item, onUpdate }) => {
        const continentMatch =
          item?.type.oneofKind === "sessionAuthenticationGeoipContinentCode"
            ? item.type.sessionAuthenticationGeoipContinentCode.match
            : undefined;
        return (
          <StringMatchEditor
            label="Continent code"
            placeholder="e.g. EU"
            options={[
              { value: "exact", label: "Exact" },
              { value: "in", label: "In list" },
            ]}
            data={[
              "AF",
              "AN",
              "AS",
              "EU",
              "NA",
              "OC",
              "SA",
            ]}
            value={continentMatch}
            onChange={(next) =>
              onUpdate(
                Expression.create({
                  type: {
                    oneofKind: "sessionAuthenticationGeoipContinentCode",
                    sessionAuthenticationGeoipContinentCode: { match: next },
                  },
                }),
              )
            }
          />
        );
      },
    },
  },

  {
    type: "timeDayType",
    title: "Weekday or weekend",
    tags: ["time", "schedule", "weekday", "weekend", "timezone"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "timeDayType",
          timeDayType: { type: TimeDayType.WEEKDAY, timezone: "" },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "timeDayType") return null;
        const day = formatEnumLabel(
          TimeDayType as any,
          item.type.timeDayType.type,
        );
        return <>{day} · {item.type.timeDayType.timezone || "UTC"}</>;
      },
      Edit: ({ item, onUpdate }) => {
        const value =
          item?.type.oneofKind === "timeDayType"
            ? item.type.timeDayType
            : { type: TimeDayType.WEEKDAY, timezone: "" };
        const emit = (next: Partial<typeof value>) =>
          onUpdate(
            Expression.create({
              type: {
                oneofKind: "timeDayType",
                timeDayType: { ...value, ...next },
              },
            }),
          );
        return (
          <div className="space-y-3">
            <SegmentedControl
              fullWidth
              size="sm"
              value={String(value.type)}
              data={[
                { value: String(TimeDayType.WEEKDAY), label: "Weekday" },
                { value: String(TimeDayType.WEEKEND), label: "Weekend" },
              ]}
              onChange={(next) => emit({ type: Number(next) })}
            />
            <Autocomplete
              label="Timezone"
              description="Use a canonical IANA timezone or enter a custom value. Empty means UTC."
              placeholder="e.g. America/New_York"
              data={TIMEZONES}
              value={value.timezone}
              clearable
              onChange={(timezone) => emit({ timezone })}
            />
          </div>
        );
      },
    },
  },

  {
    type: "timeBefore",
    title: "Time before",
    tags: ["time", "schedule", "temporal"],
    makeDefault: () =>
      Expression.create({
        type: { oneofKind: "timeBefore", timeBefore: { timestamp: undefined } },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "timeBefore") return null;
        return <TimeAgo rfc3339={item.type.timeBefore.timestamp} />;
      },
      Edit: ({ item, onUpdate }) => (
        <TimestampPicker
          value={
            item?.type.oneofKind === "timeBefore"
              ? item.type.timeBefore.timestamp
              : undefined
          }
          onChange={(v) =>
            onUpdate(
              Expression.create({
                type: { oneofKind: "timeBefore", timeBefore: { timestamp: v } },
              }),
            )
          }
        />
      ),
    },
  },

  {
    type: "timeAfter",
    title: "Time after",
    tags: ["time", "schedule", "temporal"],
    makeDefault: () =>
      Expression.create({
        type: { oneofKind: "timeAfter", timeAfter: { timestamp: undefined } },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "timeAfter") return null;
        return <TimeAgo rfc3339={item.type.timeAfter.timestamp} />;
      },
      Edit: ({ item, onUpdate }) => (
        <TimestampPicker
          isFuture
          value={
            item?.type.oneofKind === "timeAfter"
              ? item.type.timeAfter.timestamp
              : undefined
          }
          onChange={(v) =>
            onUpdate(
              Expression.create({
                type: { oneofKind: "timeAfter", timeAfter: { timestamp: v } },
              }),
            )
          }
        />
      ),
    },
  },

  makeMarkerItem(
    "sessionAuthenticationCredAuthenticatorFIDOPasskey",
    "FIDO passkey",
    ["fido", "passkey", "authenticator", "webauthn"],
    "Matches when the authenticator is a FIDO passkey. No further configuration is required.",
  ),
  makeMarkerItem(
    "sessionAuthenticationCredAuthenticatorFIDOHardware",
    "Hardware-based FIDO",
    ["fido", "hardware", "authenticator", "webauthn"],
    "Matches when the FIDO authenticator is hardware-based. No further configuration is required.",
  ),
  makeMarkerItem(
    "sessionAuthenticationCredAuthenticatorFIDOAttestationVerified",
    "FIDO verified attestation",
    ["fido", "attestation", "webauthn", "security"],
    "Matches when FIDO attestation is verified. No further configuration is required.",
  ),
  makeMarkerItem(
    "sessionAuthenticationCredAuthenticatorFIDOUserVerified",
    "FIDO user verified",
    ["fido", "user", "verification", "webauthn"],
    "Matches when the FIDO user was verified. No further configuration is required.",
  ),
  makeMarkerItem(
    "sessionAuthenticationCredAuthenticatorFIDOUserPresent",
    "FIDO user present",
    ["fido", "user", "presence", "webauthn"],
    "Matches when the FIDO user was present. No further configuration is required.",
  ),
  makeTextItem(
    "sessionAuthenticationCredAuthenticatorAAGUID",
    "FIDO authenticator AAGUID",
    ["fido", "aaguid", "authenticator", "webauthn"],
    "aaguid",
    "AAGUID",
    "00000000-0000-0000-0000-000000000000",
  ),

  makeStringMatchItem(
    "requestHTTPPath",
    "Request HTTP path",
    ["http", "request", "path", "url"],
    "Path",
  ),
  makeStringMatchItem(
    "requestHTTPMethod",
    "Request HTTP method",
    ["http", "request", "method", "verb"],
    "HTTP method",
  ),
  makeStringMatchItem(
    "requestHTTPHasHeader",
    "Request HTTP header exists",
    ["http", "request", "header"],
    "Header name",
  ),
  makeTextItem(
    "requestIP",
    "Request IP address",
    ["network", "ip", "request"],
    "value",
    "IP address",
    "1.2.3.4",
  ),
  makeTextItem(
    "requestIPInRange",
    "Request IP address range",
    ["network", "ip", "cidr", "range"],
    "value",
    "CIDR range",
    "1.2.3.0/24",
  ),

  makeStringMatchItem(
    "requestHTTPHost",
    "Request HTTP host",
    ["http", "request", "host"],
    "Host",
  ),
  makeStringMatchItem(
    "requestHTTPProtocol",
    "Request HTTP protocol",
    ["http", "request", "protocol"],
    "Protocol",
  ),
  makeStringMatchItem(
    "requestHTTPScheme",
    "Request HTTP scheme",
    ["http", "request", "scheme"],
    "Scheme",
  ),
  makeStringMatchItem(
    "requestHTTPURI",
    "Request HTTP URI",
    ["http", "request", "uri", "url"],
    "URI",
  ),
  makeNumericMatchItem(
    "requestHTTPSize",
    "Request HTTP size",
    ["http", "request", "size", "bytes"],
    "Size in bytes",
  ),
  makeTextItem(
    "requestHTTPHasQueryParam",
    "Request HTTP query parameter exists",
    ["http", "request", "query", "parameter"],
    "name",
    "Parameter name",
    "tenant",
  ),

  {
    type: "requestHTTPHeaderValue",
    title: "Request HTTP header value",
    tags: ["http", "request", "header", "value"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "requestHTTPHeaderValue",
          requestHTTPHeaderValue: {
            header: makeStringMatch("exact", ""),
            value: makeStringMatch("exact", ""),
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "requestHTTPHeaderValue") return null;
        const { header, value } = item.type.requestHTTPHeaderValue;
        return <>{`${stringMatchText(header)} = ${stringMatchText(value)}`}</>;
      },
      Edit: ({ item, onUpdate }) => {
        const cur =
          item?.type.oneofKind === "requestHTTPHeaderValue"
            ? item.type.requestHTTPHeaderValue
            : undefined;
        const emit = (header: any, value: any) =>
          onUpdate(
            Expression.create({
              type: {
                oneofKind: "requestHTTPHeaderValue",
                requestHTTPHeaderValue: { header, value },
              },
            }),
          );

        return (
          <div className="space-y-4">
            <StringMatchEditor
              label="Header name"
              value={cur?.header}
              onChange={(header) => emit(header, cur?.value ?? makeStringMatch("exact", ""))}
            />
            <StringMatchEditor
              label="Header value"
              value={cur?.value}
              onChange={(value) => emit(cur?.header ?? makeStringMatch("exact", ""), value)}
            />
          </div>
        );
      },
    },
  },

  {
    type: "requestHTTPQueryParamValue",
    title: "Request HTTP query parameter value",
    tags: ["http", "request", "query", "parameter", "value"],
    makeDefault: () =>
      Expression.create({
        type: {
          oneofKind: "requestHTTPQueryParamValue",
          requestHTTPQueryParamValue: {
            name: "",
            match: makeStringMatch("exact", ""),
          },
        },
      }),
    components: {
      Value: ({ item }) => {
        if (item.type.oneofKind !== "requestHTTPQueryParamValue") return null;
        const value = item.type.requestHTTPQueryParamValue;
        return <>{`${value.name}: ${stringMatchText(value.match)}`}</>;
      },
      Edit: ({ item, onUpdate }) => {
        const value =
          item?.type.oneofKind === "requestHTTPQueryParamValue"
            ? item.type.requestHTTPQueryParamValue
            : undefined;
        const emit = (name: string, match: any) =>
          onUpdate(
            Expression.create({
              type: {
                oneofKind: "requestHTTPQueryParamValue",
                requestHTTPQueryParamValue: { name, match },
              },
            }),
          );
        return (
          <div className="space-y-4">
            <TextInput
              label="Parameter name"
              placeholder="tenant"
              value={value?.name ?? ""}
              onChange={(event) =>
                emit(event.currentTarget.value, value?.match ?? makeStringMatch("exact", ""))
              }
            />
            <StringMatchEditor
              label="Parameter value"
              value={value?.match}
              onChange={(match) => emit(value?.name ?? "", match)}
            />
          </div>
        );
      },
    },
  },

  makeMarkerItem("requestSSH", "SSH request", ["ssh", "request"]),
  makeStringMatchItem(
    "requestSSHUser",
    "Request SSH user",
    ["ssh", "request", "user"],
    "SSH user",
  ),

  makeMarkerItem(
    "requestKubernetes",
    "Kubernetes request",
    ["kubernetes", "api", "request"],
  ),
  ...([
    ["requestKubernetesVerb", "Kubernetes verb", "verb"],
    ["requestKubernetesAPIPrefix", "Kubernetes API prefix", "API prefix"],
    ["requestKubernetesAPIGroup", "Kubernetes API group", "API group"],
    ["requestKubernetesAPIVersion", "Kubernetes API version", "API version"],
    ["requestKubernetesNamespace", "Kubernetes namespace", "Namespace"],
    ["requestKubernetesResource", "Kubernetes resource", "Resource"],
    ["requestKubernetesSubresource", "Kubernetes subresource", "Subresource"],
    ["requestKubernetesName", "Kubernetes resource name", "Resource name"],
  ] as [string, string, string][]).map(([type, title, label]) =>
    makeStringMatchItem(type, title, ["kubernetes", "api", "request"], label),
  ),

  makeMarkerItem("requestGRPC", "gRPC request", ["grpc", "request"]),
  ...([
    ["requestGRPCMethod", "gRPC method", "Method"],
    ["requestGRPCService", "gRPC service", "Service"],
    ["requestGRPCServiceFullName", "gRPC service full name", "Full service name"],
    ["requestGRPCPackage", "gRPC package", "Package"],
  ] as [string, string, string][]).map(([type, title, label]) =>
    makeStringMatchItem(type, title, ["grpc", "request"], label),
  ),

  makeMarkerItem(
    "requestPostgresConnect",
    "PostgreSQL connection",
    ["postgres", "postgresql", "request", "connect"],
  ),
  ...([
    ["requestPostgresConnectUser", "PostgreSQL connection user", "User"],
    ["requestPostgresConnectDatabase", "PostgreSQL connection database", "Database"],
    ["requestPostgresConnectApplicationName", "PostgreSQL application name", "Application name"],
  ] as [string, string, string][]).map(([type, title, label]) =>
    makeStringMatchItem(type, title, ["postgres", "postgresql", "request"], label),
  ),
  makeMarkerItem("requestPostgresQuery", "PostgreSQL query", ["postgres", "postgresql", "request", "query"]),
  makeStringMatchItem(
    "requestPostgresQueryText",
    "PostgreSQL query text",
    ["postgres", "postgresql", "request", "query"],
    "Query text",
  ),
  makeMarkerItem("requestPostgresParse", "PostgreSQL parse", ["postgres", "postgresql", "request", "parse"]),
  makeStringMatchItem("requestPostgresParseName", "PostgreSQL parse name", ["postgres", "postgresql", "request", "parse"], "Statement name"),
  makeStringMatchItem("requestPostgresParseQuery", "PostgreSQL parse query", ["postgres", "postgresql", "request", "parse"], "Query"),

  makeMarkerItem("requestDNS", "DNS request", ["dns", "request"]),
  makeStringMatchItem("requestDNSName", "DNS name", ["dns", "request", "name"], "Name"),
  makeNumericMatchItem("requestDNSTypeID", "DNS type ID", ["dns", "request", "type"], "Type ID"),

  makeMarkerItem("requestSOCKS5", "SOCKS5 request", ["socks5", "request"]),
  makeStringMatchItem("requestSOCKS5Host", "SOCKS5 host", ["socks5", "request", "host"], "Host"),
  makeNumericMatchItem("requestSOCKS5Port", "SOCKS5 port", ["socks5", "request", "port"], "Port"),
  makeEnumItem(
    "requestSOCKS5AddressType",
    "SOCKS5 address type",
    ["socks5", "request", "address"],
    "addressType",
    CoreP.RequestContext_Request_SOCKS5_Connect_AddressType as any,
    [
      CoreP.RequestContext_Request_SOCKS5_Connect_AddressType.IPV4,
      CoreP.RequestContext_Request_SOCKS5_Connect_AddressType.DOMAIN,
      CoreP.RequestContext_Request_SOCKS5_Connect_AddressType.IPV6,
    ],
    "Address type",
  ),

  makeMCPStringMatchItem(
    "requestMCPProtocolVersion",
    "MCP protocol version",
    ["mcp", "request", "protocol", "version"],
    "Protocol version",
    "",
    MCP_PROTOCOL_VERSIONS,
    "e.g. 2026-07-28",
  ),
  makeMCPStringMatchItem(
    "requestMCPMethod",
    "MCP method",
    ["mcp", "request", "method", "json-rpc"],
    "Method",
    "",
    MCP_METHODS,
    "e.g. tools/call",
  ),
  makeMCPStringMatchItem(
    "requestMCPToolName",
    "MCP tool name",
    ["mcp", "request", "tool"],
    "Tool name",
    "",
    [],
    "e.g. get_weather",
  ),
  makeMCPToolArgumentItem(),
  makeMCPStringMatchItem(
    "requestMCPPromptName",
    "MCP prompt name",
    ["mcp", "request", "prompt"],
    "Prompt name",
    "",
    ["review", "summarize", "code_review"],
    "e.g. review",
  ),
  makeMCPStringMatchItem(
    "requestMCPResourceURI",
    "MCP resource URI",
    ["mcp", "request", "resource", "uri"],
    "Resource URI",
    "",
    ["file:///example.txt", "https://example.com/resource"],
    "e.g. file:///example.txt",
  ),
  makeMarkerItem("requestMCPIsNotification", "MCP notification", ["mcp", "request", "notification"]),

  makeEnumItem(
    "requestLLMProtocol",
    "LLM protocol",
    ["llm", "ai", "request", "protocol"],
    "protocol",
    CoreP.Service_Spec_Config_LLM_Protocol as any,
    [
      CoreP.Service_Spec_Config_LLM_Protocol.OPENAI,
      CoreP.Service_Spec_Config_LLM_Protocol.ANTHROPIC,
      CoreP.Service_Spec_Config_LLM_Protocol.GEMINI,
      CoreP.Service_Spec_Config_LLM_Protocol.BEDROCK,
    ],
    "Protocol",
  ),
  makeEnumItem(
    "requestLLMOperation",
    "LLM operation",
    ["llm", "ai", "request", "operation"],
    "operation",
    CoreP.Service_Spec_Config_LLM_Operation as any,
    [1, 2, 3, 4, 5, 6, 7],
    "Operation",
  ),
  makeStringMatchItem("requestLLMModel", "LLM model", ["llm", "ai", "request", "model"], "Model"),
  makeMarkerItem("requestLLMStream", "LLM streaming request", ["llm", "ai", "request", "stream"]),
  makeNumericMatchItem("requestLLMEstimatedInputTokens", "LLM estimated input tokens", ["llm", "ai", "request", "tokens"], "Token count", "requireComplete"),
  makeEnumItem(
    "requestLLMEstimateQuality",
    "LLM estimate quality",
    ["llm", "ai", "request", "tokens"],
    "quality",
    CoreP.RequestContext_Request_LLM_EstimateQuality as any,
    [
      CoreP.RequestContext_Request_LLM_EstimateQuality.COMPLETE,
      CoreP.RequestContext_Request_LLM_EstimateQuality.PARTIAL,
      CoreP.RequestContext_Request_LLM_EstimateQuality.UNAVAILABLE,
    ],
    "Estimate quality",
  ),
  makeNumericMatchItem("requestLLMMaxOutputTokens", "LLM maximum output tokens", ["llm", "ai", "request", "tokens"], "Token count"),
  makeMarkerItem("requestLLMHasTools", "LLM request has tools", ["llm", "ai", "request", "tools"]),
  makeNumericMatchItem("requestLLMToolCount", "LLM tool count", ["llm", "ai", "request", "tools"], "Tool count"),
  makeStringMatchItem("requestLLMToolName", "LLM tool name", ["llm", "ai", "request", "tools"], "Tool name"),
  makeNumericMatchItem("requestLLMInputItemCount", "LLM input item count", ["llm", "ai", "request", "input"], "Item count"),
  makeMarkerItem("requestLLMHasImageInput", "LLM request has image input", ["llm", "ai", "request", "input"]),
  makeMarkerItem("requestLLMHasAudioInput", "LLM request has audio input", ["llm", "ai", "request", "input"]),

  makeMarkerItem(
    "apiServerReadOnlyMethods",
    "Request to read-only API methods",
    ["api", "server", "readonly", "methods"],
    "Matches requests to read-only API methods. No further configuration is required.",
  ),
  makeStringListItem(
    "apiServerMethods",
    "Request to API methods",
    ["api", "server", "methods"],
    "methods",
    "Full methods",
    "/octelium.api.main.core.v1.MainService/CreateService",
  ),
  makeStringListItem(
    "apiServerServices",
    "Request to API services",
    ["api", "server", "services"],
    "services",
    "Services",
    "MainService",
  ),
  makeBoolItem(
    "apiServerCore",
    "Request to core API",
    ["api", "server", "core"],
    "readOnlyMethods",
    "Read-only methods only",
  ),
  makeBoolItem(
    "apiServerUser",
    "Request to user API",
    ["api", "server", "user"],
    "readOnlyMethods",
    "Read-only methods only",
  ),
  makeApiServiceItem(
    "apiServerEnterprise",
    "Request to enterprise API",
    ["api", "server", "enterprise"],
    EnterpriseService,
    [
      EnterpriseService.ANY,
      EnterpriseService.MAIN,
      EnterpriseService.CLUSTER,
      EnterpriseService.POLICY_PORTAL,
    ],
    {
      [EnterpriseService.ANY]: "Any",
      [EnterpriseService.MAIN]: "Main",
      [EnterpriseService.CLUSTER]: "Cluster",
      [EnterpriseService.POLICY_PORTAL]: "Policy Portal",
    },
    EnterpriseService.ANY,
  ),
  makeApiServiceItem(
    "apiServerCordium",
    "Request to Cordium API",
    ["api", "server", "cordium"],
    CordiumService,
    [CordiumService.MAIN, CordiumService.MANAGEMENT, CordiumService.WORKSPACE],
    {
      [CordiumService.MAIN]: "Main",
      [CordiumService.MANAGEMENT]: "Management",
      [CordiumService.WORKSPACE]: "Workspace",
    },
    CordiumService.MAIN,
  ),
  makeApiServiceItem(
    "apiServerAccess",
    "Request to access API",
    ["api", "server", "access"],
    AccessService,
    [
      AccessService.ANY,
      AccessService.MAIN,
      AccessService.USER,
      AccessService.REVIEWER,
    ],
    {
      [AccessService.ANY]: "Any",
      [AccessService.MAIN]: "Main",
      [AccessService.USER]: "User",
      [AccessService.REVIEWER]: "Reviewer",
    },
    AccessService.ANY,
  ),
];

export default itemList;
