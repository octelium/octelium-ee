import resourceJSONSchema from "@/jsonschema";
import { Resource } from "@/utils/pb";
import $RefParser from "@apidevtools/json-schema-ref-parser";
import {
  Badge,
  Button,
  Drawer,
  Loader,
  TextInput,
} from "@mantine/core";
import {
  Braces,
  Check,
  ChevronRight,
  CircleHelp,
  FileJson2,
  Search,
} from "lucide-react";
import * as React from "react";

type SchemaNode = {
  $ref?: string;
  type?: string | string[];
  title?: string;
  description?: string;
  format?: string;
  enum?: unknown[];
  default?: unknown;
  properties?: Record<string, SchemaNode>;
  required?: string[];
  items?: SchemaNode;
  oneOf?: SchemaNode[];
  anyOf?: SchemaNode[];
  allOf?: SchemaNode[];
  definitions?: Record<string, SchemaNode>;
  [key: string]: unknown;
};

const normalizeRootSchema = (schema: SchemaNode): SchemaNode => {
  const prefix = "#/definitions/";
  if (!schema.$ref?.startsWith(prefix) || !schema.definitions) return schema;

  const definitionName = decodeURIComponent(schema.$ref.slice(prefix.length));
  return schema.definitions[definitionName] ?? schema;
};

const resolveSchemaTree = (
  schema: SchemaNode,
  definitions: Record<string, SchemaNode>,
  resolving = new Set<string>(),
): SchemaNode => {
  const prefix = "#/definitions/";
  const ref = schema.$ref;

  if (ref?.startsWith(prefix)) {
    const definitionName = decodeURIComponent(ref.slice(prefix.length));
    const target = definitions[definitionName];
    if (target) {
      const merged = { ...target, ...schema, $ref: undefined };
      if (resolving.has(ref)) return merged;

      resolving.add(ref);
      const resolved = resolveSchemaTree(merged, definitions, resolving);
      resolving.delete(ref);
      return resolved;
    }
  }

  const resolved: SchemaNode = { ...schema };
  if (resolved.properties) {
    resolved.properties = Object.fromEntries(
      Object.entries(resolved.properties).map(([name, child]) => [
        name,
        resolveSchemaTree(child, definitions, resolving),
      ]),
    );
  }
  if (resolved.items) {
    resolved.items = resolveSchemaTree(resolved.items, definitions, resolving);
  }
  for (const key of ["oneOf", "anyOf", "allOf"] as const) {
    if (resolved[key]) {
      resolved[key] = resolved[key]!.map((variant) =>
        resolveSchemaTree(variant, definitions, resolving),
      );
    }
  }
  return resolved;
};

const getTypeLabel = (schema: SchemaNode) => {
  if (schema.enum?.length) return "enum";
  if (Array.isArray(schema.type)) return schema.type.join(" | ");
  if (schema.type) return schema.type;
  if (schema.properties) return "object";
  if (schema.items) return "array";
  return "value";
};

const getDefaultLabel = (value: unknown) => {
  if (value === undefined) return undefined;
  if (typeof value === "string") return value || '""';
  if (typeof value === "boolean" || typeof value === "number") {
    return String(value);
  }
  return JSON.stringify(value);
};

const getChildren = (schema: SchemaNode): Array<[string, SchemaNode]> => {
  const children: Array<[string, SchemaNode]> = Object.entries(
    schema.properties ?? {},
  );
  if (schema.items) children.push(["items", schema.items]);
  (schema.oneOf ?? []).forEach((variant, index) =>
    children.push([`option ${index + 1}`, variant]),
  );
  (schema.anyOf ?? []).forEach((variant, index) =>
    children.push([`option ${index + 1}`, variant]),
  );
  (schema.allOf ?? []).forEach((variant, index) =>
    children.push([`part ${index + 1}`, variant]),
  );
  return children;
};

const schemaMatches = (name: string, schema: SchemaNode, query: string): boolean => {
  if (!query) return true;
  const haystack = [
    name,
    schema.title,
    schema.description,
    getTypeLabel(schema),
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
  return (
    haystack.includes(query) ||
    getChildren(schema).some(([childName, child]) =>
      schemaMatches(childName, child, query),
    )
  );
};

const SchemaProperty = (props: {
  name: string;
  schema: SchemaNode;
  required?: boolean;
  depth?: number;
  query: string;
}) => {
  const depth = props.depth ?? 0;
  const children = getChildren(props.schema);
  const hasChildren = children.length > 0;
  const [expanded, setExpanded] = React.useState(false);
  const queryMatches = schemaMatches(props.name, props.schema, props.query);

  if (!queryMatches) return null;

  const isOpen = expanded || !!props.query;
  const defaultValue = getDefaultLabel(props.schema.default);

  return (
    <div className={depth > 0 ? "ml-3 border-l border-slate-200 pl-3" : ""}>
      <button
        type="button"
        className="flex w-full min-w-0 items-center gap-2 rounded-lg px-2.5 py-2 text-left transition-colors duration-500 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        onClick={() => hasChildren && setExpanded((value) => !value)}
        aria-expanded={hasChildren ? isOpen : undefined}
      >
        <span className="flex h-5 w-5 shrink-0 items-center justify-center text-slate-400">
          {hasChildren ? (
            <ChevronRight
              size={14}
              className={isOpen ? "rotate-90 transition-transform" : "transition-transform"}
            />
          ) : (
            <span className="h-1.5 w-1.5 rounded-full bg-slate-300" />
          )}
        </span>
        <span className="min-w-0 flex-1 truncate text-[0.73rem] font-bold text-slate-700">
          {props.name}
        </span>
        {props.required && (
          <Badge size="xs" variant="light" color="red">
            required
          </Badge>
        )}
        <Badge size="xs" variant="light" color="gray">
          {getTypeLabel(props.schema)}
        </Badge>
      </button>

      {(props.schema.description || props.schema.format || defaultValue) && (
        <div className="ml-10 flex flex-wrap items-center gap-x-2 gap-y-1 pb-2 text-[0.65rem] font-medium text-slate-500">
          {props.schema.description && <span>{props.schema.description}</span>}
          {props.schema.format && (
            <span className="rounded bg-slate-100 px-1.5 py-0.5 text-slate-500">
              format: {props.schema.format}
            </span>
          )}
          {defaultValue !== undefined && (
            <span className="rounded bg-slate-100 px-1.5 py-0.5 text-slate-500">
              default: {defaultValue}
            </span>
          )}
        </div>
      )}

      {props.schema.enum && props.schema.enum.length > 0 && (
        <div className="ml-10 mb-2 flex flex-wrap gap-1">
          {props.schema.enum.slice(0, 12).map((value) => (
            <span
              key={String(value)}
              className="rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-[0.61rem] font-semibold text-slate-500"
            >
              {String(value)}
            </span>
          ))}
          {props.schema.enum.length > 12 && (
            <span className="px-1 py-0.5 text-[0.61rem] font-semibold text-slate-400">
              +{props.schema.enum.length - 12} more
            </span>
          )}
        </div>
      )}

      {hasChildren && isOpen && (
        <div className="space-y-0.5 pb-1">
          {children.map(([childName, child], index) => (
            <SchemaProperty
              key={`${childName}-${index}`}
              name={childName}
              schema={child}
              required={props.schema.required?.includes(childName)}
              depth={depth + 1}
              query={props.query}
            />
          ))}
        </div>
      )}
    </div>
  );
};

const ResourceSchema = (props: { item: Resource }) => {
  const [opened, setOpened] = React.useState(false);
  const [loading, setLoading] = React.useState(false);
  const [schema, setSchema] = React.useState<SchemaNode>();
  const [error, setError] = React.useState<string>();
  const [query, setQuery] = React.useState("");
  const resourceKey = `${props.item.apiVersion}:${props.item.kind}`;

  const loadSchema = React.useCallback(async () => {
    const rawSchema = resourceJSONSchema(props.item) as SchemaNode | undefined;
    if (!rawSchema) {
      setError("A schema is not available for this resource type yet.");
      return;
    }

    setLoading(true);
    setError(undefined);
    try {
      const resolved = await $RefParser.dereference(structuredClone(rawSchema), {
        dereference: { circular: "ignore" },
      });
      const resolvedDocument = resolved as SchemaNode;
      const root = normalizeRootSchema(resolvedDocument);
      setSchema(
        resolveSchemaTree(root, resolvedDocument.definitions ?? {}),
      );
    } catch {
      setError("The resource schema could not be loaded.");
    } finally {
      setLoading(false);
    }
  }, [resourceKey]);

  React.useEffect(() => {
    setSchema(undefined);
    setError(undefined);
    setQuery("");
    if (opened) void loadSchema();
  }, [loadSchema, opened, resourceKey]);

  const open = () => {
    setOpened(true);
  };

  const rootProperties = Object.entries(schema?.properties ?? {});
  const required = new Set(schema?.required ?? []);

  return (
    <>
      <Button
        type="button"
        variant="outline"
        size="compact-xs"
        leftSection={<Braces size={11} strokeWidth={2.5} />}
        onClick={open}
      >
        Schema
      </Button>

      <Drawer
        opened={opened}
        onClose={() => setOpened(false)}
        position="right"
        size="min(620px, 100vw)"
        title={
          <div className="flex min-w-0 items-center gap-2">
            <FileJson2 size={15} className="shrink-0 text-slate-400" />
            <span className="text-xs font-bold uppercase tracking-[0.06em] text-slate-500">
              Resource schema
            </span>
            <span className="truncate text-sm font-semibold text-slate-800">
              {props.item.kind}
            </span>
          </div>
        }
        overlayProps={{ backgroundOpacity: 0.2, blur: 1 }}
        transitionProps={{
          transition: "slide-left",
          duration: 500,
          exitDuration: 500,
        }}
        styles={{
          header: { borderBottom: "1px solid #e2e8f0", minHeight: "56px" },
          body: {
            minHeight: "calc(100dvh - 56px)",
            padding: "16px",
            backgroundColor: "#f8fafc",
          },
          content: { borderLeft: "1px solid #e2e8f0" },
        }}
      >
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-3.5 shadow-sm">
            <div className="flex items-start gap-3">
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white">
                <Braces size={15} />
              </span>
              <div className="min-w-0">
                <p className="text-[0.72rem] font-bold text-slate-800">
                  {props.item.kind} fields
                </p>
                <p className="mt-0.5 text-[0.65rem] font-semibold leading-5 text-slate-500">
                  Expand a field to inspect its type, description, defaults, and nested values.
                </p>
              </div>
            </div>
            {schema && (
              <div className="mt-3 flex flex-wrap gap-1.5">
                <Badge variant="light" color="gray">
                  {rootProperties.length} top-level fields
                </Badge>
                {required.size > 0 && (
                  <Badge variant="light" color="red">
                    {required.size} required
                  </Badge>
                )}
              </div>
            )}
          </div>

          {schema && rootProperties.length > 0 && (
            <TextInput
              placeholder="Search fields or descriptions"
              leftSection={<Search size={14} />}
              value={query}
              onChange={(event) => setQuery(event.currentTarget.value.toLowerCase())}
              rightSection={query ? <Check size={13} className="text-emerald-500" /> : undefined}
            />
          )}

          {loading && (
            <div className="flex items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-8 text-sm font-semibold text-slate-500">
              <Loader size={16} />
              Loading schema…
            </div>
          )}

          {!loading && error && (
            <div className="flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3.5 py-3 text-sm font-semibold text-amber-800">
              <CircleHelp size={16} className="mt-0.5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {!loading && !error && schema && rootProperties.length > 0 && (
            <div className="rounded-xl border border-slate-200 bg-white p-2 shadow-sm">
              {rootProperties.map(([name, child]) => (
                <SchemaProperty
                  key={name}
                  name={name}
                  schema={child}
                  required={required.has(name)}
                  query={query}
                />
              ))}
            </div>
          )}

          {!loading && !error && schema && rootProperties.length === 0 && (
            <div className="rounded-xl border border-slate-200 bg-white px-4 py-8 text-center text-sm font-semibold text-slate-500">
              This schema does not expose any object fields.
            </div>
          )}
        </div>
      </Drawer>
    </>
  );
};

export default ResourceSchema;
