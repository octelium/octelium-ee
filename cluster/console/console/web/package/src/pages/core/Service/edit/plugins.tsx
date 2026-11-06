import * as CoreP from "@/apis/corev1/corev1";
import Cond from "@/components/Condition";
import {
  Callout,
  FormSection,
  Grid,
  KeyValueList,
  ToggleList,
} from "@/pages/enterprise/CollectorExporter/fields";
import { useListKeys } from "@/utils/forms";
import { Button, Menu, Select, TextInput } from "@mantine/core";
import {
  Blocks,
  DatabaseZap,
  FileJson2,
  Gauge,
  LucideIcon,
  MessageSquareText,
  Plug,
  Plus,
  Route,
  ScrollText,
} from "lucide-react";
import * as React from "react";
import {
  BodyField,
  CodeField,
  DurationField,
  EnumField,
  FieldText,
  ItemCard,
  joinSummary,
  jsonError,
  NumberField,
  Pill,
  plural,
  Scope,
  Segments,
  StatusField,
  TextField,
  Toggle,
  Update,
} from "./fields";
import { hostPortError, nameError, pathPrefixError } from "./model";

export type PluginKind = "http" | "mcp" | "llm";

const Phase = CoreP.Service_Spec_Config_HTTP_Plugin_Phase;

const HeaderSendMode =
  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_HeaderSendMode;
const BodySendMode =
  CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode_BodySendMode;

export type PluginDefinition = {
  type: string;
  label: string;
  description: string;
  icon: LucideIcon;
  group: string;
  phased: boolean;
  summary?: (value: any) => string | undefined;
  create: () => any;
};

const rateLimitKeyLabel = (key?: { type: { oneofKind?: string } }) =>
  key?.type.oneofKind === "perUser"
    ? "per User"
    : key?.type.oneofKind === "eval"
      ? "per CEL key"
      : "per Session";

const durationLabel = (duration?: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit["window"]) => {
  const type = duration?.type;
  if (!type?.oneofKind) return undefined;
  const value = (type as any)[type.oneofKind] as number;
  return `${value} ${type.oneofKind}`;
};

export const SHARED_PLUGINS: PluginDefinition[] = [
  {
    type: "rateLimit",
    label: "Rate limit",
    description: "Limit the number of requests in a window",
    icon: Gauge,
    group: "Traffic",
    phased: true,
    summary: (value) =>
      joinSummary(
        value.limit ? plural(Number(value.limit), "request") : undefined,
        durationLabel(value.window) && `per ${durationLabel(value.window)}`,
        rateLimitKeyLabel(value.key),
      ),
    create: () => ({
      oneofKind: "rateLimit",
      rateLimit: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit.create(),
    }),
  },
  {
    type: "direct",
    label: "Direct response",
    description: "Answer without reaching the upstream",
    icon: MessageSquareText,
    group: "Traffic",
    phased: true,
    summary: (value) =>
      value.statusCode ? `Status ${value.statusCode}` : undefined,
    create: () => ({
      oneofKind: "direct",
      direct: CoreP.Service_Spec_Config_HTTP_Plugin_Direct.create({
        body: { type: { oneofKind: "inline", inline: "" } },
      }),
    }),
  },
  {
    type: "path",
    label: "Path rewrite",
    description: "Add or remove a path prefix",
    icon: Route,
    group: "Traffic",
    phased: true,
    summary: (value) =>
      joinSummary(
        value.removePrefix && `Remove ${value.removePrefix}`,
        value.addPrefix && `Add ${value.addPrefix}`,
      ),
    create: () => ({
      oneofKind: "path",
      path: CoreP.Service_Spec_Config_HTTP_Plugin_Path.create(),
    }),
  },
  {
    type: "jsonSchema",
    label: "JSON Schema",
    description: "Validate the JSON request body",
    icon: FileJson2,
    group: "Validation",
    phased: true,
    summary: (value) =>
      value.statusCode ? `Rejects with ${value.statusCode}` : undefined,
    create: () => ({
      oneofKind: "jsonSchema",
      jsonSchema: CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema.create({
        type: { oneofKind: "inline", inline: "" },
      }),
    }),
  },
  {
    type: "lua",
    label: "Lua",
    description: "Run a script on requests and responses",
    icon: ScrollText,
    group: "Custom logic",
    phased: true,
    create: () => ({
      oneofKind: "lua",
      lua: CoreP.Service_Spec_Config_HTTP_Plugin_Lua.create({
        type: { oneofKind: "inline", inline: "" },
      }),
    }),
  },
  {
    type: "extProc",
    label: "External processor",
    description: "Call an Envoy ext_proc compliant gRPC server",
    icon: Plug,
    group: "Custom logic",
    phased: true,
    summary: (value) =>
      value.type.oneofKind === "address"
        ? value.type.address
        : value.type.oneofKind === "container"
          ? value.type.container.image
          : undefined,
    create: () => ({
      oneofKind: "extProc",
      extProc: CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc.create({
        type: { oneofKind: "address", address: "" },
      }),
    }),
  },
];

const CACHE_PLUGIN: PluginDefinition = {
  type: "cache",
  label: "Cache",
  description: "Serve cached responses from the Cluster's Redis",
  icon: DatabaseZap,
  group: "Traffic",
  phased: true,
  summary: (value) =>
    durationLabel(value.ttl) ? `TTL ${durationLabel(value.ttl)}` : undefined,
  create: () => ({
    oneofKind: "cache",
    cache: CoreP.Service_Spec_Config_HTTP_Plugin_Cache.create(),
  }),
};

export const HTTP_PLUGINS: PluginDefinition[] = [
  ...SHARED_PLUGINS,
  CACHE_PLUGIN,
];

const HeadersList = (props: {
  description: string;
  headers: { key: string; value: string }[];
  make: () => { key: string; value: string };
  update: Update;
}) => (
  <KeyValueList
    label="Response headers"
    description={props.description}
    items={props.headers}
    make={props.make}
    addLabel="Add header"
    keyPlaceholder="Retry-After"
    valuePlaceholder="60"
    onUpdate={props.update}
  />
);

export const DirectForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_Direct;
  update: Update;
}) => {
  const { value, update } = props;
  return (
    <>
      <StatusField
        of={value}
        name="statusCode"
        update={update}
        placeholder="200"
      />
      <OptionalBody
        value={value}
        title="Body"
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_Direct_Body.create({
            type: { oneofKind: "inline", inline: "" },
          })
        }
        update={update}
        placeholder={'{ "message": "ok" }'}
      />
      <HeadersList
        description="Set on the returned response."
        headers={value.headers}
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_Direct_KeyValue.create()
        }
        update={update}
      />
    </>
  );
};

export const RateLimitKey = (props: {
  holder: { key?: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Key };
  update: Update;
}) => {
  const { holder, update } = props;
  const type = holder.key?.type;
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <Segments
        label="Count requests"
        value={
          type?.oneofKind === "perUser" || type?.oneofKind === "eval"
            ? type.oneofKind
            : "perSession"
        }
        options={[
          { value: "perSession", label: "Per Session" },
          { value: "perUser", label: "Per User" },
          { value: "eval", label: "CEL key" },
        ]}
        onChange={(next) => {
          holder.key = CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Key.create(
            {
              type:
                next === "eval"
                  ? { oneofKind: "eval", eval: "" }
                  : next === "perUser"
                    ? { oneofKind: "perUser", perUser: true }
                    : { oneofKind: "perSession", perSession: true },
            },
          );
          update();
        }}
      />
      {type?.oneofKind === "eval" && (
        <TextField
          of={type}
          name="eval"
          update={update}
          label="CEL expression"
          placeholder="ctx.user.metadata.name"
          required
          mono
        />
      )}
    </div>
  );
};

const RateLimitForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit;
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
          label="Limit"
          placeholder="100"
          required
          thousands
        />
        <DurationField
          of={value}
          name="window"
          update={update}
          label="Window"
          required
        />
        <StatusField of={value} name="statusCode" update={update} placeholder="429" />
      </Grid>
      <RateLimitKey holder={value} update={update} />
      <OptionalBody
        value={value}
        title="Rejection body"
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_Body.create({
            type: { oneofKind: "inline", inline: "" },
          })
        }
        update={update}
        placeholder={'{ "message": "Too Many Requests" }'}
      />
      <HeadersList
        description="Set on the rejection response."
        headers={value.headers}
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_RateLimit_KeyValue.create()
        }
        update={update}
      />
    </>
  );
};

const OptionalBody = (props: {
  value: { body?: any };
  title: string;
  make: () => any;
  update: Update;
  placeholder?: string;
}) => {
  const { value, update } = props;
  return (
    <ToggleList>
      <div className="px-3.5 py-3">
        <div className="flex items-center justify-between gap-3">
          <FieldText label={props.title} />
          <Button
            type="button"
            variant="default"
            size="compact-xs"
            onClick={() => {
              value.body = value.body ? undefined : props.make();
              update();
            }}
          >
            {value.body ? "Remove" : "Customize"}
          </Button>
        </div>
        {value.body && (
          <div className="mt-3">
            <BodyField
              label="Body"
              holder={value.body}
              update={update}
              placeholder={props.placeholder}
            />
          </div>
        )}
      </div>
    </ToggleList>
  );
};

const CacheForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_Cache;
  update: Update;
}) => {
  const { value, update } = props;
  const key = value.key;
  return (
    <>
      <Grid>
        <DurationField of={value} name="ttl" update={update} label="TTL" />
        <NumberField
          of={value}
          name="maxSize"
          update={update}
          label="Largest cached response"
          suffix=" bytes"
          max={256000000}
          thousands
        />
      </Grid>
      <ToggleList>
        <Toggle
          of={value}
          name="useXCacheHeader"
          update={update}
          label="Set the X-Cache header"
          description="Tell clients whether the response came from the cache."
        />
        <Toggle
          of={value}
          name="allowUnsafeMethods"
          update={update}
          label="Cache unsafe methods"
          description="Also cache methods other than GET and HEAD."
        />
      </ToggleList>
      <TextInput
        label="Cache key"
        description="CEL expression. Defaults to the request path and query."
        classNames={{ input: "font-mono" }}
        placeholder="ctx.request.http.path"
        value={key?.type.oneofKind === "eval" ? key.type.eval : ""}
        onChange={(event) => {
          const eval_ = event.currentTarget.value;
          value.key = eval_
            ? CoreP.Service_Spec_Config_HTTP_Plugin_Cache_Key.create({
                type: { oneofKind: "eval", eval: eval_ },
              })
            : undefined;
          update();
        }}
      />
    </>
  );
};

const LuaForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_Lua;
  update: Update;
}) => (
  <CodeField
    label="Script"
    description="Define onRequest and onResponse. A function that is not defined is skipped."
    mode="lua"
    value={props.value.type.oneofKind === "inline" ? props.value.type.inline : ""}
    onChange={(inline) => {
      props.value.type = { oneofKind: "inline", inline };
      props.update();
    }}
  />
);

const PathForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_Path;
  update: Update;
}) => (
  <Grid>
    <TextField
      of={props.value}
      name="removePrefix"
      update={props.update}
      label="Remove prefix"
      placeholder="/api"
      mono
      error={pathPrefixError}
    />
    <TextField
      of={props.value}
      name="addPrefix"
      update={props.update}
      label="Add prefix"
      placeholder="/v2"
      mono
      error={pathPrefixError}
    />
  </Grid>
);

const JSONSchemaForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema;
  update: Update;
}) => {
  const { value, update } = props;
  const schema = value.type.oneofKind === "inline" ? value.type.inline : "";
  return (
    <>
      <CodeField
        label="Schema"
        description="Requests whose JSON body does not match are rejected."
        required
        mode="json"
        value={schema}
        error={jsonError(schema)}
        onChange={(inline) => {
          value.type = { oneofKind: "inline", inline };
          update();
        }}
      />
      <StatusField of={value} name="statusCode" update={update} placeholder="400" />
      <OptionalBody
        value={value}
        title="Rejection body"
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema_Body.create({
            type: { oneofKind: "inline", inline: "" },
          })
        }
        update={update}
        placeholder="Invalid request body"
      />
      <HeadersList
        description="Set on the rejection response."
        headers={value.headers}
        make={() =>
          CoreP.Service_Spec_Config_HTTP_Plugin_JSONSchema_KeyValue.create()
        }
        update={update}
      />
    </>
  );
};

const ExtProcForm = (props: {
  value: CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc;
  update: Update;
}) => {
  const { value, update } = props;
  const type = value.type;
  const mode = value.processingMode;
  const seconds = value.messageTimeout?.type;
  const tooLong =
    (seconds?.oneofKind === "seconds" && seconds.seconds > 6) ||
    (seconds?.oneofKind === "milliseconds" && seconds.milliseconds > 6000) ||
    seconds?.oneofKind === "minutes" ||
    seconds?.oneofKind === "hours";
  const headerOptions = [
    { value: HeaderSendMode.SEND, label: "Send" },
    { value: HeaderSendMode.SKIP, label: "Skip" },
  ];
  const bodyOptions = [
    { value: BodySendMode.NONE, label: "Do not send" },
    { value: BodySendMode.BUFFERED, label: "Buffered" },
  ];
  return (
    <>
      <Segments
        label="Server"
        value={type.oneofKind}
        options={[
          { value: "address", label: "Address" },
          { value: "container", label: "Managed container" },
        ]}
        onChange={(next) => {
          value.type =
            next === "container"
              ? {
                  oneofKind: "container",
                  container:
                    CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_Container.create(),
                }
              : { oneofKind: "address", address: "" };
          update();
        }}
      />
      {type.oneofKind === "address" && (
        <TextField
          of={type}
          name="address"
          update={update}
          label="Address"
          placeholder="ext-proc.default.svc:8080"
          required
          error={hostPortError}
        />
      )}
      {type.oneofKind === "container" && (
        <Grid>
          <TextField
            of={type.container}
            name="image"
            update={update}
            label="Image"
            placeholder="ghcr.io/acme/ext-proc:1.0.0"
            required
          />
          <NumberField
            of={type.container}
            name="port"
            update={update}
            label="Port"
            placeholder="8080"
            max={65535}
          />
        </Grid>
      )}
      <DurationField
        of={value}
        name="messageTimeout"
        update={update}
        label="Message timeout"
        description="At most 6 seconds."
      />
      {tooLong && (
        <Callout title="The message timeout is above the 6 second limit" />
      )}
      <ToggleList>
        <div className="px-3.5 py-3">
          <div className="flex items-center justify-between gap-3">
            <FieldText
              label="Processing mode"
              description="What the server receives. Defaults come from ext_proc."
            />
            <Button
              type="button"
              variant="default"
              size="compact-xs"
              onClick={() => {
                value.processingMode = mode
                  ? undefined
                  : CoreP.Service_Spec_Config_HTTP_Plugin_ExtProc_ProcessingMode.create();
                update();
              }}
            >
              {mode ? "Remove" : "Customize"}
            </Button>
          </div>
          {mode && (
            <div className="mt-3">
              <Grid>
                <EnumField
                  of={mode}
                  name="requestHeaderMode"
                  update={update}
                  label="Request headers"
                  options={headerOptions}
                />
                <EnumField
                  of={mode}
                  name="responseHeaderMode"
                  update={update}
                  label="Response headers"
                  options={headerOptions}
                />
                <EnumField
                  of={mode}
                  name="requestBodyMode"
                  update={update}
                  label="Request body"
                  options={bodyOptions}
                />
                <EnumField
                  of={mode}
                  name="responseBodyMode"
                  update={update}
                  label="Response body"
                  options={bodyOptions}
                />
              </Grid>
            </div>
          )}
        </div>
      </ToggleList>
    </>
  );
};

const SharedForm = (props: { type: any; update: Update }) => {
  const { type, update } = props;
  switch (type.oneofKind) {
    case "direct":
      return <DirectForm value={type.direct} update={update} />;
    case "rateLimit":
      return <RateLimitForm value={type.rateLimit} update={update} />;
    case "cache":
      return <CacheForm value={type.cache} update={update} />;
    case "lua":
      return <LuaForm value={type.lua} update={update} />;
    case "path":
      return <PathForm value={type.path} update={update} />;
    case "jsonSchema":
      return <JSONSchemaForm value={type.jsonSchema} update={update} />;
    case "extProc":
      return <ExtProcForm value={type.extProc} update={update} />;
    default:
      return null;
  }
};

const conditionSummary = (condition?: CoreP.Condition) => {
  const type = condition?.type;
  switch (type?.oneofKind) {
    case "matchAny":
      return "Every request";
    case "match":
      return type.match || "Not set";
    case "not":
      return type.not ? `Not ${type.not}` : "Not set";
    case "all":
      return `All of ${plural(type.all.of.length, "condition")}`;
    case "any":
      return `Any of ${plural(type.any.of.length, "condition")}`;
    case "none":
      return `None of ${plural(type.none.of.length, "condition")}`;
    case "opa":
      return "Rego policy";
    default:
      return "Not set";
  }
};

export const createMatchAny = () =>
  CoreP.Condition.create({ type: { oneofKind: "matchAny", matchAny: true } });

export const ConditionField = (props: {
  label: string;
  description: string;
  value?: CoreP.Condition;
  onChange: (condition: CoreP.Condition) => void;
}) => (
  <div className="flex min-w-0 flex-col gap-1">
    <FieldText label={props.label} description={props.description} />
    <Cond item={props.value} onChange={props.onChange} />
  </div>
);

const uniqueName = (plugins: any[], base: string) => {
  const names = new Set(plugins.map((item) => item.name));
  if (!names.has(base)) return base;
  for (let i = 2; ; i++) if (!names.has(`${base}-${i}`)) return `${base}-${i}`;
};

const kebab = (value: string) =>
  value.replace(/[A-Z]/g, (char) => `-${char.toLowerCase()}`);

const AddPlugin = (props: {
  definitions: PluginDefinition[];
  onAdd: (definition: PluginDefinition) => void;
}) => {
  const groups = React.useMemo(() => {
    const ret = new Map<string, PluginDefinition[]>();
    for (const item of props.definitions) {
      ret.set(item.group, [...(ret.get(item.group) ?? []), item]);
    }
    return [...ret.entries()];
  }, [props.definitions]);

  return (
    <Menu position="bottom-end" width={300} shadow="md" withinPortal>
      <Menu.Target>
        <Button
          type="button"
          variant="default"
          size="compact-xs"
          className="shrink-0"
          leftSection={<Plus size={11} strokeWidth={2.5} />}
        >
          Add Plugin
        </Button>
      </Menu.Target>
      <Menu.Dropdown
        className="max-h-[min(460px,62dvh)] overflow-y-auto"
        style={{ maxWidth: "calc(100vw - 32px)" }}
      >
        {groups.map(([group, items]) => (
          <React.Fragment key={group}>
            <Menu.Label>{group}</Menu.Label>
            {items.map((item) => {
              const Icon = item.icon;
              return (
                <Menu.Item
                  key={item.type}
                  leftSection={
                    <span className="flex h-7 w-7 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-500">
                      <Icon size={13} strokeWidth={2.2} />
                    </span>
                  }
                  onClick={() => props.onAdd(item)}
                >
                  <span className="flex flex-col">
                    <span className="text-body font-semibold text-slate-900">
                      {item.label}
                    </span>
                    <span className="text-xs font-normal text-slate-500">
                      {item.description}
                    </span>
                  </span>
                </Menu.Item>
              );
            })}
          </React.Fragment>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
};

const PluginForm = (props: {
  plugin: any;
  index: number;
  plugins: any[];
  definitions: PluginDefinition[];
  scope: Scope;
  update: Update;
  renderType?: (plugin: any, update: Update) => React.ReactNode;
}) => {
  const { plugin, definitions, update } = props;
  const definition = definitions.find(
    (item) => item.type === plugin.type.oneofKind,
  );
  const memory = React.useRef<Record<string, any>>({});
  const duplicate = props.plugins.some(
    (item, index) => index !== props.index && item.name === plugin.name,
  );

  return (
    <>
      <div className="grid grid-cols-1 gap-4 @xl:grid-cols-2">
        <TextField
          of={plugin}
          name={"name" as never}
          update={update}
          label="Name"
          placeholder="my-plugin"
          required
          error={(value) =>
            duplicate ? "This name is already used" : nameError(value)
          }
        />
        <Select
          label="Type"
          allowDeselect={false}
          data={definitions.map((item) => ({
            value: item.type,
            label: item.label,
          }))}
          value={plugin.type.oneofKind ?? null}
          onChange={(next) => {
            const nextDefinition = definitions.find(
              (item) => item.type === next,
            );
            if (!nextDefinition) return;
            if (plugin.type.oneofKind) {
              memory.current[plugin.type.oneofKind] = plugin.type;
            }
            plugin.type = memory.current[nextDefinition.type] ?? nextDefinition.create();
            if (!nextDefinition.phased) plugin.phase = Phase.PHASE_UNSET;
            update();
          }}
        />
      </div>

      {definition?.phased && (props.scope === "default" || plugin.phase === Phase.PRE_AUTH) && (
        <Segments
          label="Runs"
          value={plugin.phase === Phase.PRE_AUTH ? "pre" : "post"}
          options={[
            { value: "post", label: "After authorization" },
            { value: "pre", label: "Before authentication" },
          ]}
          onChange={(next) => {
            plugin.phase = next === "pre" ? Phase.PRE_AUTH : Phase.POST_AUTH;
            update();
          }}
        />
      )}

      <ConditionField
        label="Condition"
        description="The Plugin runs only for the requests that match."
        value={plugin.condition}
        onChange={(condition) => {
          plugin.condition = condition;
          update();
        }}
      />

      <SharedForm type={plugin.type} update={update} />
      {props.renderType?.(plugin, update)}

      <ToggleList>
        <Toggle
          of={plugin}
          name={"isDisabled" as never}
          update={update}
          label="Disable this Plugin"
          description="Keep the configuration without running it."
        />
      </ToggleList>
    </>
  );
};

export const PluginsSection = (props: {
  holder: { plugins: any[] };
  kind: PluginKind;
  definitions: PluginDefinition[];
  scope: Scope;
  update: Update;
  description: string;
  renderType?: (plugin: any, update: Update) => React.ReactNode;
}) => {
  const { holder, definitions, update } = props;
  const rows = useListKeys();
  const plugins = holder.plugins;
  const added = React.useRef(new Set<string>());

  return (
    <FormSection
      icon={Blocks}
      title="Plugins"
      description={props.description}
      aside={
        <AddPlugin
          definitions={definitions}
          onAdd={(definition) => {
            const plugin = {
              name: uniqueName(plugins, kebab(definition.type)),
              isDisabled: false,
              phase: Phase.PHASE_UNSET,
              condition: createMatchAny(),
              type: definition.create(),
            };
            plugins.push(plugin);
            added.current.add(rows.keyAt("plugins", plugins.length - 1));
            update();
          }}
        />
      }
    >
      {plugins.length === 0 ? (
        <p className="text-xs font-normal leading-5 text-slate-500">
          No Plugins yet. Every Plugin has a Condition, so it can apply to a
          subset of the requests.
        </p>
      ) : (
        <div className="flex min-w-0 flex-col gap-2">
          {plugins.map((plugin, index) => {
            const definition = definitions.find(
              (item) => item.type === plugin.type.oneofKind,
            );
            const key = rows.keyAt("plugins", index);
            const value = plugin.type.oneofKind
              ? plugin.type[plugin.type.oneofKind]
              : undefined;
            const move = (to: number) => {
              rows.move("plugins", index, to);
              const [item] = plugins.splice(index, 1);
              plugins.splice(to, 0, item);
              update();
            };
            return (
              <ItemCard
                key={key}
                icon={definition?.icon}
                title={plugin.name || `Plugin ${index + 1}`}
                subtitle={joinSummary(
                  definition?.label,
                  value && definition?.summary?.(value),
                  conditionSummary(plugin.condition),
                )}
                badges={
                  <>
                    {plugin.isDisabled && <Pill tone="warning">Disabled</Pill>}
                    {plugin.phase === Phase.PRE_AUTH && (
                      <Pill>Before authentication</Pill>
                    )}
                  </>
                }
                defaultOpen={added.current.has(key)}
                removeLabel={`Remove ${plugin.name || `Plugin ${index + 1}`}`}
                onRemove={() => {
                  rows.removeAt("plugins", index);
                  plugins.splice(index, 1);
                  update();
                }}
                onMoveUp={index > 0 ? () => move(index - 1) : undefined}
                onMoveDown={
                  index < plugins.length - 1 ? () => move(index + 1) : undefined
                }
              >
                <PluginForm
                  plugin={plugin}
                  index={index}
                  plugins={plugins}
                  definitions={definitions}
                  scope={props.scope}
                  update={update}
                  renderType={props.renderType}
                />
              </ItemCard>
            );
          })}
        </div>
      )}
    </FormSection>
  );
};
