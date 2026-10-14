import {
  autocompletion,
  snippetCompletion,
  type Completion,
  type CompletionContext,
  type CompletionResult,
  type CompletionSource,
} from "@codemirror/autocomplete";
import type { JSONSchema7, JSONSchema7Definition } from "json-schema";

interface SchemaCompletion extends Completion {
  description?: string;
}

const memberPattern = /[A-Za-z_]\w*(?:\s*(?:\.\s*\w*|\[\s*(?:\d+|[A-Za-z_]\w*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')\s*\]))*/;

const functions: SchemaCompletion[] = [
  ["has", "value", "bool", "Check whether a field is present."],
  ["size", "value", "int", "Count the elements in a string, list, or map."],
  ["type", "value", "type", "Return the type of a value."],
  ["int", "value", "int", "Convert a value to an integer."],
  ["uint", "value", "uint", "Convert a value to an unsigned integer."],
  ["double", "value", "double", "Convert a value to a floating-point number."],
  ["string", "value", "string", "Convert a value to a string."],
  ["bool", "value", "bool", "Convert a value to a boolean."],
  ["duration", "string", "duration", "Parse a duration such as \"300s\"."],
  ["timestamp", "string", "timestamp", "Parse an RFC 3339 timestamp."],
].map(([label, argument, result, description]) => ({
  ...snippetCompletion(`${label}(\${${argument}})`, {
    label,
    type: "function",
    detail: `(${argument}) → ${result}`,
  }),
  description,
}));

const methods = (schema: JSONSchema7): SchemaCompletion[] => {
  const type = Array.isArray(schema.type) ? schema.type[0] : schema.type;
  const entries =
    type === "array"
      ? [
          ["all", "item, predicate", "bool", "Check whether every item matches."],
          ["exists", "item, predicate", "bool", "Check whether any item matches."],
          ["exists_one", "item, predicate", "bool", "Check whether exactly one item matches."],
          ["filter", "item, predicate", "list", "Keep the items that match."],
          ["map", "item, expression", "list", "Transform each item."],
          ["size", "", "int", "Count the items in the list."],
        ]
      : type === "string"
        ? [
            ["contains", "string", "bool", "Check whether this string contains another string."],
            ["startsWith", "string", "bool", "Check the start of this string."],
            ["endsWith", "string", "bool", "Check the end of this string."],
            ["matches", "pattern", "bool", "Match this string against a regular expression."],
            ["size", "", "int", "Count the characters in this string."],
          ]
        : schema.additionalProperties
          ? [["size", "", "int", "Count the entries in the map."]]
          : [];

  return entries.map(([label, argument, result, description]) => ({
    ...snippetCompletion(argument ? `${label}(\${${argument}})` : `${label}()`, {
      label,
      type: "function",
      detail: `(${argument}) → ${result}`,
    }),
    description,
  }));
};

function resolveReference(root: JSONSchema7, reference: string) {
  if (reference === "#") return root;
  if (!reference.startsWith("#/")) return undefined;
  let node: unknown = root;
  for (const part of reference.slice(2).split("/")) {
    const key = part.replace(/~1/g, "/").replace(/~0/g, "~");
    if (!node || typeof node !== "object" || !Object.hasOwn(node, key)) {
      return undefined;
    }
    node = (node as Record<string, unknown>)[key];
  }
  return typeof node === "object" && node !== null
    ? (node as JSONSchema7)
    : undefined;
}

function normalizeSchema(
  node: JSONSchema7Definition | undefined,
  root: JSONSchema7,
  seen = new Set<JSONSchema7>(),
): JSONSchema7 {
  if (!node || typeof node === "boolean" || seen.has(node)) return {};
  const visited = new Set(seen).add(node);
  const variants = [
    ...(node.$ref ? [resolveReference(root, node.$ref)] : []),
    ...(node.allOf ?? []),
    ...(node.anyOf ?? []),
    ...(node.oneOf ?? []),
  ].map((part) => normalizeSchema(part, root, visited));
  const properties = Object.assign(
    {},
    ...variants.map((part) => part.properties ?? {}),
    node.properties ?? {},
  ) as NonNullable<JSONSchema7["properties"]>;
  const values = [
    ...new Set(
      variants.flatMap((part) => part.enum ?? []).concat(node.enum ?? []),
    ),
  ];
  return {
    ...Object.assign({}, ...variants),
    ...node,
    ...(Object.keys(properties).length > 0 ? { properties } : {}),
    ...(values.length > 0 ? { enum: values } : {}),
  };
}

function pathParts(text: string) {
  return [...text.matchAll(/([A-Za-z_]\w*)|\[\s*(\d+|[A-Za-z_]\w*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')\s*\]/g)].map(
    (match) => {
      const key = match[1] ?? match[2];
      return {
        key: /^["']/.test(key)
          ? key.slice(1, -1).replace(/\\(["'\\])/g, "$1")
          : key,
        indexed: match[2] !== undefined,
      };
    },
  );
}

function schemaAtPath(root: JSONSchema7, path: string) {
  let current: JSONSchema7Definition | undefined = root;
  for (const { key, indexed } of pathParts(path)) {
    const node = normalizeSchema(current, root);
    if (node.type === "array" && indexed) {
      current = Array.isArray(node.items) ? node.items[0] : node.items;
    } else {
      current = node.properties?.[key];
      if (!current && typeof node.additionalProperties === "object") {
        current = node.additionalProperties;
      }
    }
    if (current === undefined) return undefined;
  }
  return normalizeSchema(current, root);
}

function expressionContext(text: string) {
  const code = text.split("");
  let quote = "";
  let raw = false;
  let comment = false;
  for (let index = 0; index < text.length; index++) {
    if (comment) {
      if (text[index] === "\n") comment = false;
      else code[index] = " ";
    } else if (quote) {
      if (text.startsWith(quote, index)) {
        code.fill(" ", index, index + quote.length);
        index += quote.length - 1;
        quote = "";
      } else {
        code[index] = " ";
        if (!raw && text[index] === "\\") code[++index] = " ";
      }
    } else if (text.startsWith("//", index)) {
      comment = true;
      code[index] = " ";
    } else if (text[index] === '"' || text[index] === "'") {
      quote = text.startsWith(text[index].repeat(3), index)
        ? text[index].repeat(3)
        : text[index];
      raw = /[rR]$/.test(text.slice(0, index));
      code.fill(" ", index, index + quote.length);
      index += quote.length - 1;
    }
  }
  return { code: code.join(""), literal: quote !== "", comment };
}

function localSchema(root: JSONSchema7, code: string) {
  const properties = { ...root.properties };
  const bindings = code.matchAll(/([A-Za-z_]\w*(?:\s*\.\s*\w+|\[\s*\d+\s*\])*)\s*\.\s*(?:all|exists|exists_one|filter|map)\(\s*([A-Za-z_]\w*)\s*,/g);
  for (const binding of bindings) {
    let depth = 1;
    for (const character of code.slice(binding.index + binding[0].length)) {
      if (character === "(") depth++;
      if (character === ")") depth--;
      if (depth === 0) break;
    }
    if (depth === 0) continue;
    const parent = schemaAtPath({ ...root, properties }, binding[1]);
    if (parent?.type === "array" && parent.items && !Array.isArray(parent.items)) {
      properties[binding[2]] = parent.items;
    }
  }
  return { ...root, properties };
}

function schemaType(schema: JSONSchema7) {
  if (schema.type === "array") return "list";
  if (schema.type === "integer") return "int";
  if (schema.type === "number") return "number";
  if (schema.type === "boolean") return "bool";
  return Array.isArray(schema.type)
    ? schema.type.join(" | ")
    : schema.type ?? (schema.properties ? "object" : "value");
}

function enumCompletion(
  context: CompletionContext,
  root: JSONSchema7,
  before: string,
  code: string,
): CompletionResult | null {
  const match = before.match(
    new RegExp(
      `(${memberPattern.source})\\s*(?:==|!=)\\s*(?:(["'])([^"'\\\\\\n]*)|([\\w-]*))$`,
    ),
  );
  if (
    !match ||
    code.slice(match.index, match.index! + match[1].length).trim() !== match[1].trim()
  ) return null;
  const node = schemaAtPath(root, match[1]);
  if (!node?.enum) return null;
  const quote = match[2];
  const prefix = match[3] ?? match[4] ?? "";
  const options = node.enum
    .filter((value) => !quote || typeof value === "string")
    .filter((value) => context.explicit || (quote ? String(value) : JSON.stringify(value)) !== prefix)
    .map((value): SchemaCompletion => ({
      label: quote ? String(value) : JSON.stringify(value),
      type: "constant",
      detail: "enum",
      apply: quote
        ? (view, completion, from, to) => {
            const escaped = completion.label
              .replace(/\\/g, "\\\\")
              .replace(new RegExp(quote, "g"), `\\${quote}`);
            const insert =
              escaped + (view.state.sliceDoc(to, to + 1) === quote ? "" : quote);
            view.dispatch({
              changes: { from, to, insert },
              selection: { anchor: from + escaped.length },
              userEvent: "input.complete",
            });
          }
        : undefined,
    }));
  return options.length > 0
    ? {
        from: context.pos - prefix.length,
        options,
        validFor: (text: string) =>
          /^[\w-]*$/.test(text) && !options.some((option) => option.label === text),
      }
    : null;
}

export function schemaCompletionSource(schema: JSONSchema7): CompletionSource {
  return (context) => {
    const before = context.state.sliceDoc(0, context.pos);
    const scanned = expressionContext(before);
    if (scanned.comment) return null;
    const root = localSchema(schema, scanned.code);
    const enumeration = enumCompletion(context, root, before, scanned.code);
    if (enumeration) return enumeration;
    if (scanned.literal) return null;
    const member = context.matchBefore(memberPattern);
    if (!member && (!context.explicit || /[\w\]"']\s*$/.test(before))) return null;
    const text = member?.text ?? "";
    const prefix = text.match(/[A-Za-z_]\w*$/)?.[0] ?? "";
    const parentText = text.slice(0, text.length - prefix.length).trim();
    if (parentText && !parentText.endsWith(".")) return null;
    const parent = schemaAtPath(root, parentText.replace(/\.\s*$/, ""));
    if (!parent) return null;
    const options: SchemaCompletion[] = Object.entries(parent.properties ?? {})
      .filter(([, definition]) => definition !== false)
      .map(([label, definition]) => {
        const node = normalizeSchema(definition, root);
        return {
          label,
          type:
            node.type === "object" || node.properties ? "namespace" : "property",
          detail: schemaType(node),
          description: node.description,
        };
      });
    options.push(...(parentText ? methods(parent) : functions));
    if (!parentText) {
      options.push(
        ...["true", "false", "null"].map((label) => ({
          label,
          type: "constant",
          detail: label === "null" ? "null" : "bool",
        })),
      );
    }
    const available = context.explicit
      ? options
      : options.filter((option) => option.label !== prefix || typeof option.apply === "function");
    if (available.length === 0) return null;
    return {
      from: context.pos - prefix.length,
      options: available,
      validFor: (text: string) =>
        /^[A-Za-z_]\w*$|^$/.test(text) &&
        !available.some((option) => option.label === text && typeof option.apply !== "function"),
    };
  };
}

export function schemaAutocomplete(schema: JSONSchema7) {
  return autocompletion({
    override: [schemaCompletionSource(schema)],
    activateOnTyping: true,
    activateOnTypingDelay: 75,
    interactionDelay: 0,
    defaultKeymap: false,
    icons: false,
    maxRenderedOptions: 60,
    addToOptions: [
      {
        position: 90,
        render: (completion) => {
          const description = (completion as SchemaCompletion).description;
          if (!description) return null;
          const element = document.createElement("span");
          element.className = "cm-completionDescription";
          element.textContent = description;
          element.title = description;
          return element;
        },
      },
    ],
  });
}
