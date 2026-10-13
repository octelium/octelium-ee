import { AccessLog_Entry_Common_Reason_Type } from "@/apis/corev1/corev1";
import {
  useAccessTop,
  useAuditTop,
  useAuthTop,
} from "@/components/Dashboard/queries";
import { AutoRefreshContext } from "@/components/Dashboard/utils";
import { getPolicyReason } from "@/components/PolicyTester/policyReason";
import {
  getClientVisibilityAccessLog,
  getClientVisibilityAuditLog,
  getClientVisibilityAuthenticationLog,
} from "@/utils/client";
import { RpcOptions } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRoot } from "react-dom/client";

const params = new URLSearchParams(window.location.search);
const periodMinutes = Number(params.get("periodMinutes") ?? 60);
const calls: object[] = [];
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

Object.assign(window, {
  codeHealth: {
    calls,
    getPolicyReason,
    reasons: AccessLog_Entry_Common_Reason_Type,
  },
});

for (const client of [
  getClientVisibilityAccessLog(),
  getClientVisibilityAuthenticationLog(),
  getClientVisibilityAuditLog(),
]) {
  for (const method of client.methods.filter((item) =>
    item.name.includes("Top"),
  )) {
    Object.getPrototypeOf(client)[method.localName] = async (
      request: object,
      options: RpcOptions,
    ) => {
      calls.push({
        name: method.localName,
        type: method.I.typeName,
        valid: method.I.is(request),
        request: method.I.fromBinary(method.I.toBinary(request)),
        abort: options.abort instanceof AbortSignal,
      });
      return {
        response: method.O.fromJson({
          items: [{ count: 3 }],
          totalCount: 4,
          totalOther: 2,
        }),
      };
    };
  }
}

const Output = (props: { query: ReturnType<typeof useAccessTop> }) => (
  <output>
    {props.query.isSuccess
      ? JSON.stringify(props.query.data)
      : props.query.status}
  </output>
);

const AccessTop = () => (
  <Output
    query={useAccessTop(
      (params.get("resource") ?? "user") as Parameters<typeof useAccessTop>[0],
      periodMinutes,
    )}
  />
);

const AuthTop = () => (
  <Output
    query={useAuthTop(
      (params.get("resource") ?? "user") as Parameters<typeof useAuthTop>[0],
      periodMinutes,
    )}
  />
);

const AuditTop = () => (
  <Output
    query={useAuditTop(
      (params.get("resource") ?? "user") as Parameters<typeof useAuditTop>[0],
      periodMinutes,
    )}
  />
);

const App =
  params.get("kind") === "auth"
    ? AuthTop
    : params.get("kind") === "audit"
      ? AuditTop
      : AccessTop;

createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <AutoRefreshContext.Provider value={false}>
      <App />
    </AutoRefreshContext.Provider>
  </QueryClientProvider>,
);
