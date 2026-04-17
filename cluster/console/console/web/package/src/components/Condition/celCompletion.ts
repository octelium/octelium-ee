import $RefParser from "@apidevtools/json-schema-ref-parser";
import { autocompletion, Completion } from "@codemirror/autocomplete";

async function loadResolvedSchema(urlOrObject: string | object) {
  const parser = new $RefParser();
  return await parser.dereference(urlOrObject);
}

function mergeSchemas(schemas: any[]): any {
  return schemas.reduce(
    (acc, cur) => {
      if (cur.properties)
        acc.properties = { ...acc.properties, ...cur.properties };
      if (cur.enum)
        acc.enum = Array.from(new Set([...(acc.enum || []), ...cur.enum]));
      return acc;
    },
    { properties: {}, enum: [] },
  );
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

function getSchemaAtPath(root: any, path: string[]): any | undefined {
  let current: any = root;
  for (const key of path) {
    const norm = normalizeSchema(current);
    current = norm?.properties?.[key];
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

        const options = completionsFromSchema(parentSchema).filter((opt) =>
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
