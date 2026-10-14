import "@/index.css";
import "@mantine/core/styles.layer.css";

import { CompletionContext } from "@codemirror/autocomplete";
import { EditorState } from "@codemirror/state";
import { Drawer, MantineProvider } from "@mantine/core";
import type { JSONSchema7 } from "json-schema";
import * as React from "react";
import { createRoot } from "react-dom/client";
import { CELEditor } from "@/components/Condition/Editor";
import { schemaCompletionSource } from "@/components/Condition/celCompletion";
import RequestContext from "@/jsonschema/core/RequestContext.json";
import themeMantine from "@/utils/theme/mantine";
import cssVariablesResolver from "@/utils/theme/cssVariables";

const params = new URLSearchParams(window.location.search);
const requestSchema = RequestContext as JSONSchema7;
const schema: JSONSchema7 = {
  type: "object",
  properties: { ctx: { $ref: requestSchema.$ref } },
  definitions: requestSchema.definitions,
};

Object.assign(window, {
  celFixture: {
    complete: async (doc: string, customSchema?: JSONSchema7) => {
      const state = EditorState.create({ doc });
      const result = await schemaCompletionSource(customSchema ?? schema)(
        new CompletionContext(state, doc.length, true),
      );
      return result
        ? {
            from: result.from,
            options: result.options.map(({ label, detail }) => ({ label, detail })),
          }
        : null;
    },
  },
});

const Form = () => {
  const [expression, setExpression] = React.useState(
    params.get("expression") ?? "",
  );
  return (
    <>
      <div className="h-[200px] overflow-hidden rounded-xl border border-slate-200 bg-slate-50 p-4">
        <label className="text-xs font-semibold text-slate-700">CEL expression</label>
        <CELEditor
          exp={expression}
          invalid={params.get("invalid") === "true"}
          onChange={setExpression}
        />
      </div>
      <button type="button" className="mt-4">
        Next field
      </button>
      <output data-testid="expression" hidden>
        {expression}
      </output>
    </>
  );
};

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider
      theme={themeMantine}
      cssVariablesResolver={cssVariablesResolver}
      forceColorScheme={params.get("scheme") === "dark" ? "dark" : "light"}
    >
      {params.get("drawer") === "true" ? (
        <Drawer
          opened
          onClose={() => {}}
          position="right"
          size="min(560px, 100vw)"
          title="Edit condition"
        >
          <Form />
        </Drawer>
      ) : (
        <div className="mx-auto w-full max-w-2xl p-6">
          <Form />
        </div>
      )}
    </MantineProvider>
  </React.StrictMode>,
);
