import "@/index.css";
import "@mantine/core/styles.layer.css";
import { Secret, SecretList, InlinePolicy } from "@/apis/corev1/corev1";
import { SSHSession } from "@/apis/visibilityv1/visibilityv1";
import { GetSecretSummaryResponse } from "@/apis/visibilityv1/core/vcorev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GaugeOperation_Function } from "@/apis/visibilityv1/metrics/vmetricsv1";
import { MantineProvider } from "@mantine/core";
import { QueryClientProvider } from "@tanstack/react-query";
import * as echarts from "echarts/core";
import * as React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import store from "@/store";
import { queryClient } from "@/utils";
import {
  getClientCore,
  getClientVisibilityCore,
  getClientVisibilityMetrics,
} from "@/utils/client";
import themeMantine from "@/utils/theme/mantine";

const params = new URLSearchParams(window.location.search);
const mode = params.get("mode") ?? "lua";
const points = [
  { ts: Timestamp.fromDate(new Date("2026-10-12T13:00:00Z")), value: 1 },
  { ts: Timestamp.fromDate(new Date("2026-10-12T14:00:00Z")), value: 2 },
];
const secret = Secret.create({
  apiVersion: "core/v1",
  kind: "Secret",
  metadata: { name: "perf-secret" },
  spec: {},
  data: { type: { oneofKind: "value", value: "example" } },
});
Object.getPrototypeOf(getClientCore()).listSecret = async () => ({
  response: SecretList.create({ items: [secret] }),
});
Object.getPrototypeOf(getClientCore()).getSecret = async () => ({
  response: secret,
});
Object.getPrototypeOf(getClientVisibilityCore()).getSecretSummary =
  async () => ({ response: GetSecretSummaryResponse.create() });
Object.getPrototypeOf(getClientVisibilityMetrics()).queryMetrics =
  async () => ({
    response: {
      series: [
        {
          labels: [],
          points: {
            oneofKind: "number",
            number: {
              points: points.map((point) => ({
                timestamp: point.ts,
                value: { oneofKind: "asDouble", asDouble: point.value },
              })),
            },
          },
        },
      ],
    },
  });
(window as any).chartOption = () => {
  const element = document.querySelector(".echarts-for-react");
  return element
    ? echarts.getInstanceByDom(element as HTMLElement)?.getOption()
    : undefined;
};

const Content = React.lazy(async () => {
  switch (mode) {
    case "lua": {
      const [{ default: Editor }, { syntaxTree }, { EditorView }] =
        await Promise.all([
          import("@/components/Editor"),
          import("@codemirror/language"),
          import("@codemirror/view"),
        ]);
      (window as any).editorTokens = () => {
        const element = document.querySelector<HTMLElement>(".cm-editor");
        const view = element ? EditorView.findFromDOM(element) : undefined;
        return view ? syntaxTree(view.state).toString() : "";
      };
      return {
        default: () => (
          <Editor mode="lua" value={'local answer = 42\nreturn "hello"'} />
        ),
      };
    }
    case "schema": {
      const { ResourceSchemaContent } =
        await import("@/components/ResourceSchema");
      const { default: RefParser } =
        await import("@apidevtools/json-schema-ref-parser");
      (window as any).dereference = async () =>
        RefParser.dereference({
          type: "object",
          properties: {
            remote: { $ref: `${window.location.origin}/remote-schema.json` },
          },
        });
      return { default: () => <ResourceSchemaContent item={secret} /> };
    }
    case "cel": {
      const { CELEditor } = await import("@/components/Condition/Editor");
      return { default: () => <CELEditor exp="" onChange={() => {}} /> };
    }
    case "flags": {
      const { CountryFlag } = await import("@/components/SelectCountry");
      return {
        default: () => {
          const [code, setCode] = React.useState("US");
          return (
            <div>
              <CountryFlag code={code} />
              <CountryFlag code="invalid" />
              <CountryFlag code="zz" />
              <button onClick={() => setCode("FR")}>Change flag</button>
            </div>
          );
        },
      };
    }
    case "dates": {
      const { default: TimestampPicker } =
        await import("@/components/TimestampPicker");
      return {
        default: () => <TimestampPicker label="From" onChange={() => {}} />,
      };
    }
    case "terminal": {
      const { XTermSSHReplay } =
        await import("@/components/SSHRecordingPlayer/Player");
      return {
        default: () => <XTermSSHReplay sshSession={SSHSession.create()} />,
      };
    }
    case "yaml": {
      const { default: ResourceYAML } =
        await import("@/components/ResourceYAML");
      return { default: () => <ResourceYAML item={secret} readOnly /> };
    }
    case "inline": {
      const { default: SelectInlinePolicies } =
        await import("@/components/ResourceLayout/SelectInlinePolicies");
      return {
        default: () => {
          const [policies, setPolicies] = React.useState<InlinePolicy[]>([]);
          return (
            <div>
              <button
                onClick={() =>
                  setPolicies([
                    InlinePolicy.create({
                      name: "example",
                      spec: { rules: [] },
                    }),
                  ])
                }
              >
                Open policy
              </button>
              <SelectInlinePolicies
                inlinePolicies={policies}
                onUpdate={setPolicies}
              />
            </div>
          );
        },
      };
    }
    case "routes": {
      const { default: routes } = await import("@/pages/core/router");
      const router = createMemoryRouter([routes()], {
        initialEntries: [params.get("path") ?? "/core/secrets/create"],
      });
      return { default: () => <RouterProvider router={router} /> };
    }
    case "LineChart": {
      const { default: Chart } = await import("@/components/Charts/LineChart");
      return {
        default: () => (
          <Chart
            points={points}
            variant={params.get("variant") === "bar" ? "bar" : "line"}
          />
        ),
      };
    }
    case "SeriesChart": {
      const { default: Chart } =
        await import("@/components/Charts/SeriesChart");
      return {
        default: () => <Chart series={[{ name: "Example", points }]} />,
      };
    }
    case "MetricChart": {
      const { default: Chart, gaugeOp } =
        await import("@/components/Charts/MetricChart");
      return {
        default: () => (
          <Chart
            metric="example"
            operation={gaugeOp(GaugeOperation_Function.AVG)}
          />
        ),
      };
    }
    case "PieChart": {
      const { default: Chart } = await import("@/components/Charts/PieChart");
      return {
        default: () => <Chart data={[{ name: "Example", value: 2 }]} />,
      };
    }
    case "PolarChart": {
      const { default: Chart } = await import("@/components/Charts/PolarChart");
      return {
        default: () => <Chart data={[{ name: "Example", value: 2 }]} />,
      };
    }
    case "GaugeChart": {
      const { default: Chart } = await import("@/components/Charts/GaugeChart");
      return { default: () => <Chart name="Example" num={1} total={2} /> };
    }
    case "CounterChart": {
      const { default: Chart } =
        await import("@/components/Charts/CounterChart");
      return { default: () => <Chart label="Example" value={2} /> };
    }
    case "Agent": {
      const { BlockView } = await import("@/pages/Agent/Blocks");
      return {
        default: () => (
          <BlockView
            ctx={{ streaming: false } as any}
            block={{
              id: "chart",
              createdAt: "2026-10-12T13:00:00Z",
              type: "chart",
              chart: {
                chartType: (params.get("variant") as any) ?? "bar",
                x: { key: "name", type: "category" },
                series: [{ key: "value" }],
                rows: [{ name: "Example", value: 2 }],
              },
            }}
          />
        ),
      };
    }
    default:
      throw new Error(`Unknown mode ${mode}`);
  }
});

createRoot(document.getElementById("root")!).render(
  <Provider store={store}>
    <MantineProvider theme={themeMantine} defaultColorScheme="light">
      <QueryClientProvider client={queryClient}>
        <div style={{ width: 900, padding: 24 }}>
          <React.Suspense fallback={<div>Loading fixture…</div>}>
            <Content />
          </React.Suspense>
        </div>
      </QueryClientProvider>
    </MantineProvider>
  </Provider>,
);
