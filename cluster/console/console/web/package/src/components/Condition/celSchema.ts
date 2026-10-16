import type { JSONSchema7 } from "json-schema";
import AccessRequest from "@/jsonschema/access/Request.json";
import RequestContext from "@/jsonschema/core/RequestContext.json";
import User from "@/jsonschema/core/User.json";

export type CELContext = "core" | "access";

const requestContextSchema = RequestContext as JSONSchema7;
const accessRequestSchema = AccessRequest as JSONSchema7;
const userSchema = User as JSONSchema7;

export const celSchemas: Record<CELContext, JSONSchema7> = {
  core: {
    type: "object",
    properties: { ctx: { $ref: requestContextSchema.$ref } },
    definitions: requestContextSchema.definitions,
  },
  access: {
    type: "object",
    properties: {
      ctx: {
        type: "object",
        properties: {
          request: {
            $ref: accessRequestSchema.$ref,
            description:
              "Request is the access Request that is being evaluated.",
          },
          user: {
            $ref: userSchema.$ref,
            description: "User is the User that created the Request.",
          },
        },
      },
    },
    definitions: {
      ...userSchema.definitions,
      ...accessRequestSchema.definitions,
    },
  },
};
