import { autocompletion, Completion } from "@codemirror/autocomplete";

function mergeSchemas(schemas: any[]): any {
  const normalized = schemas.map(normalizeSchema);
  const properties = Object.assign(
    {},
    ...normalized.map((schema) => schema.properties ?? {}),
  );
  const enumValues = Array.from(
    new Set(normalized.flatMap((schema) => schema.enum ?? [])),
  );
  const first = normalized.find((schema) => Object.keys(schema).length > 0) ?? {};

  return {
    ...first,
    ...(Object.keys(properties).length > 0 ? { properties } : {}),
    ...(enumValues.length > 0 ? { enum: enumValues } : {}),
  };
}

function normalizeSchema(node: any): any {
  if (!node) return {};
  if (node.oneOf) return mergeSchemas(node.oneOf.map(normalizeSchema));
  if (node.anyOf) return mergeSchemas(node.anyOf.map(normalizeSchema));
  if (node.allOf) return mergeSchemas(node.allOf.map(normalizeSchema));
  return node;
}

function completionsFromSchema(schema: any): Completion[] {
  const completions: Completion[] = [];
  if (schema.properties) {
    for (const key of Object.keys(schema.properties)) {
      const prop = normalizeSchema(schema.properties[key]);
      completions.push({
        label: key,
        type: prop.type === "object" ? "namespace" : (prop.type ?? "property"),
        info: prop.description ?? `Type: ${prop.type ?? "object"}`,
        boost: prop.description ? 1 : 0,
      });
    }
  }
  if (schema.enum) {
    for (const val of schema.enum) {
      completions.push({
        label: JSON.stringify(val),
        type: "constant",
        info: "enum value",
      });
    }
  }
  return completions;
}

const CEL_BUILTINS: Completion[] = [
  "has",
  "exists",
  "size",
  "type",
  "int",
  "uint",
  "double",
  "string",
  "bool",
  "duration",
  "timestamp",
].map((label) => ({
  label,
  type: "function",
  info: "CEL built-in",
}));

function getSchemaAtPath(root: any, path: string[]): any | undefined {
  let current: any = root;
  for (const key of path) {
    const norm = normalizeSchema(current);
    if (norm.type === "array" && norm.items) {
      current = /^\d+$/.test(key)
        ? norm.items
        : normalizeSchema(norm.items)?.properties?.[key];
    } else {
      current = norm?.properties?.[key];
      if (!current && norm?.additionalProperties) {
        current = norm.additionalProperties === true
          ? {}
          : norm.additionalProperties;
      }
    }
    if (!current) return undefined;
  }
  return normalizeSchema(current);
}

export function schemaAutocomplete(schema: any) {
  return autocompletion({
    override: [
      (context) => {
        const before = context.matchBefore(/[A-Za-z0-9_.]*/);
        if (!before || (before.from === before.to && !context.explicit))
          return null;

        const text = before.text;
        const dotIndex = text.lastIndexOf(".");
        const parts =
          dotIndex >= 0
            ? text.slice(0, dotIndex).split(".").filter(Boolean)
            : [];
        const prefix = dotIndex >= 0 ? text.slice(dotIndex + 1) : text;

        const parentSchema = getSchemaAtPath(schema, parts);
        if (!parentSchema) return null;

        const options = [
          ...completionsFromSchema(parentSchema),
          ...(parts.length === 0 ? CEL_BUILTINS : []),
        ].filter((opt) =>
          opt.label.toLowerCase().startsWith(prefix.toLowerCase()),
        );

        if (options.length === 0) return null;

        return {
          from: before.from + (dotIndex >= 0 ? dotIndex + 1 : 0),
          options,
          validFor: /^[A-Za-z0-9_]*$/,
        };
      },
    ],
    activateOnTyping: true,
    defaultKeymap: false,
  });
}
