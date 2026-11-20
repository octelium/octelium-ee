import assert from "node:assert/strict";
import { once } from "node:events";
import { readFile, readdir } from "node:fs/promises";
import { createServer } from "node:http";
import { after, before, describe, it } from "node:test";
import $RefParser from "@apidevtools/json-schema-ref-parser";

const schemaDirectory = new URL("../src/jsonschema/", import.meta.url);
const schemaFiles = (await readdir(schemaDirectory, { recursive: true }))
  .filter((name) => name.endsWith(".json"))
  .sort();

describe("Resource schema reference parsing", () => {
  let server;
  let origin;

  before(async () => {
    server = createServer((request, response) => {
      if (request.url === "/schema.json") {
        response.setHeader("content-type", "application/json");
        response.end(JSON.stringify({
          type: "object",
          properties: { name: { $ref: "./definition.yaml" } },
        }));
      } else if (request.url === "/definition.yaml") {
        response.setHeader("content-type", "application/yaml");
        response.end("type: string\ndescription: Resource name\n");
      } else {
        response.writeHead(404);
        response.end();
      }
    });
    server.listen(0, "127.0.0.1");
    await once(server, "listening");
    origin = `http://127.0.0.1:${server.address().port}`;
  });

  after(async () => {
    if (server) await new Promise((resolve) => server.close(resolve));
  });

  for (const name of schemaFiles) {
    it(`resolves the bundled ${name} for the schema viewer`, async () => {
      const schema = JSON.parse(await readFile(new URL(name, schemaDirectory), "utf8"));
      const resolved = await $RefParser.dereference(structuredClone(schema), {
        dereference: { circular: "ignore" },
      });
      assert.ok(resolved);
      assert.doesNotThrow(() => JSON.stringify(resolved));
    });
  }

  it("resolves relative references to external YAML schemas", async () => {
    const resolved = await $RefParser.dereference(`${origin}/schema.json`, {
      resolve: { http: { safeUrlResolver: false } },
    });
    assert.deepEqual(resolved.properties.name, {
      type: "string",
      description: "Resource name",
    });
  });

  it("keeps circular references serializable for the schema viewer", async () => {
    const resolved = await $RefParser.dereference({
      type: "object",
      properties: { next: { $ref: "#" } },
    }, { dereference: { circular: "ignore" } });
    assert.deepEqual(resolved.properties.next, { $ref: "#" });
    assert.doesNotThrow(() => JSON.stringify(resolved));
  });

  it("reports missing local references", async () => {
    await assert.rejects(
      $RefParser.dereference({ properties: { name: { $ref: "#/definitions/missing" } } }),
      /does not exist/,
    );
  });

  it("reports failed external references", async () => {
    await assert.rejects(
      $RefParser.dereference({ $ref: `${origin}/missing.json` }, {
        resolve: { http: { safeUrlResolver: false } },
      }),
      /404/,
    );
  });

  it("blocks private network references by default", async () => {
    await assert.rejects(
      $RefParser.dereference({ $ref: `${origin}/schema.json` }),
      /Unable to resolve/,
    );
  });
});
