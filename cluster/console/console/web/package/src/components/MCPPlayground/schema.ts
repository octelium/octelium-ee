import { isObject, JSONObject } from "./client";

export type JSONSchema = {
  type?: string | string[];
  title?: string;
  description?: string;
  properties?: Record<string, JSONSchema>;
  required?: string[];
  enum?: unknown[];
  default?: unknown;
  minimum?: number;
  maximum?: number;
};

export type FieldKind =
  "string" | "number" | "integer" | "boolean" | "enum" | "json";

export type Field = {
  name: string;
  title?: string;
  description?: string;
  kind: FieldKind;
  required: boolean;
  options?: string[];
  placeholder?: string;
  minimum?: number;
  maximum?: number;
};

export type FieldValues = Record<string, string>;

const primitive = (value: unknown) =>
  value === null || ["string", "number", "boolean"].includes(typeof value);

const schemaType = (schema: JSONSchema): string | undefined => {
  if (Array.isArray(schema.type)) {
    return schema.type.find((type) => type !== "null");
  }
  return schema.type;
};

const fieldKind = (schema: JSONSchema): FieldKind => {
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    return schema.enum.every(primitive) ? "enum" : "json";
  }
  switch (schemaType(schema)) {
    case "string":
      return "string";
    case "number":
      return "number";
    case "integer":
      return "integer";
    case "boolean":
      return "boolean";
    default:
      return "json";
  }
};

const placeholderOf = (value: unknown): string | undefined => {
  if (value === undefined) return undefined;
  return typeof value === "string" ? value : JSON.stringify(value);
};

export const schemaFields = (schema: unknown): Field[] => {
  if (!isObject(schema) || !isObject(schema.properties)) return [];
  const required = Array.isArray(schema.required)
    ? (schema.required as unknown[]).filter(
        (name): name is string => typeof name === "string",
      )
    : [];

  return Object.entries(schema.properties as Record<string, unknown>).map(
    ([name, value]) => {
      const property: JSONSchema = isObject(value) ? value : {};
      return {
        name,
        title: property.title,
        description: property.description,
        kind: fieldKind(property),
        required: required.includes(name),
        options: Array.isArray(property.enum)
          ? property.enum.map((option) => JSON.stringify(option))
          : undefined,
        placeholder: placeholderOf(property.default),
        minimum: property.minimum,
        maximum: property.maximum,
      };
    },
  );
};

export const promptFields = (args: unknown): Field[] =>
  Array.isArray(args)
    ? args.flatMap((arg) =>
        isObject(arg) && typeof arg.name === "string"
          ? [
              {
                name: arg.name,
                title: typeof arg.title === "string" ? arg.title : undefined,
                description:
                  typeof arg.description === "string"
                    ? arg.description
                    : undefined,
                kind: "string" as const,
                required: arg.required === true,
              },
            ]
          : [],
      )
    : [];

export const optionLabel = (option: string) => {
  const value: unknown = JSON.parse(option);
  return typeof value === "string" ? value : option;
};

const convert = (field: Field, raw: string): unknown => {
  switch (field.kind) {
    case "string":
      return raw;
    case "enum":
      return JSON.parse(raw);
    case "boolean":
      return raw === "true";
    case "number":
    case "integer": {
      const number = Number(raw);
      if (
        !Number.isFinite(number) ||
        (field.kind === "integer" && !Number.isInteger(number))
      ) {
        throw new Error(
          `“${field.name}” must be ${field.kind === "integer" ? "an integer" : "a number"}.`,
        );
      }
      return number;
    }
    case "json":
      try {
        return JSON.parse(raw);
      } catch {
        throw new Error(`“${field.name}” must be valid JSON.`);
      }
  }
};

export const buildArguments = (
  fields: Field[],
  values: FieldValues,
): JSONObject => {
  const ret: JSONObject = {};
  for (const field of fields) {
    const raw = values[field.name] ?? "";
    if ((field.kind === "string" ? raw : raw.trim()) === "") {
      if (field.required) throw new Error(`“${field.name}” is required.`);
      continue;
    }
    ret[field.name] = convert(field, raw);
  }
  return ret;
};

export const toFieldValues = (fields: Field[], args: JSONObject) => {
  const ret: FieldValues = {};
  for (const field of fields) {
    const value = args[field.name];
    if (value === undefined) continue;
    ret[field.name] =
      field.kind === "string" && typeof value === "string"
        ? value
        : field.kind === "boolean"
          ? String(value === true)
          : field.kind === "number" || field.kind === "integer"
            ? String(value)
            : field.kind === "enum"
              ? JSON.stringify(value)
              : JSON.stringify(value, null, 2);
  }
  return ret;
};

export const parseArguments = (value: string): JSONObject => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value.trim() || "{}");
  } catch {
    throw new Error("The arguments must be valid JSON.");
  }
  if (!isObject(parsed)) {
    throw new Error("The arguments must be a JSON object.");
  }
  return parsed;
};
