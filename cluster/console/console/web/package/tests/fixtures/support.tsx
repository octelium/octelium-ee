import "@/index.css";
import "@mantine/core/styles.layer.css";
import * as CoreP from "@/apis/corev1/corev1";
import * as EnterpriseP from "@/apis/enterprisev1/enterprisev1";
import { Struct } from "@/apis/google/protobuf/struct";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetStatusResponse } from "@/apis/userv1/userv1";
import * as VisibilityP from "@/apis/visibilityv1/visibilityv1";
import Notifications from "@/components/Notifications";
import PageHeader from "@/components/PageHeader";
import { PageHeaderProvider } from "@/components/PageHeader/context";
import { SupportAvailabilityContext } from "@/components/Support/context";
import { setStatus } from "@/features/settings/slice";
import routerSupport from "@/pages/support/router";
import store from "@/store";
import { queryClient } from "@/utils";
import * as clients from "@/utils/client";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Outlet, useLocation } from "react-router";
import { RouterProvider } from "react-router/dom";

declare global {
  interface Window {
    support: {
      calls: { method: string; request: any }[];
      uploads: { name: string; purpose: number; data: number[] }[];
      location: string;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const scenario = params.get("state") ?? "available";
const calls: { method: string; request: any }[] = [];
const uploads: { name: string; purpose: number; data: number[] }[] = [];
window.support = { calls, uploads, location: "" };

const ago = (minutes: number) =>
  Timestamp.fromDate(new Date(Date.now() - minutes * 60_000));

const ORG = "6f1b2c4d-8e9f-4a0b-9c1d-2e3f4a5b6c7d";
const PORTAL = "https://portal.octelium.com";
const WAITING = "0b1c2d3e-4f5a-4b6c-8d7e-9f0a1b2c3d4e";
const RESOLVED = "1c2d3e4f-5a6b-4c7d-8e9f-0a1b2c3d4e5f";
const LOGS = "2d3e4f5a-6b7c-4d8e-9f0a-1b2c3d4e5f6a";

const {
  SupportInfo_State: State,
  SupportRequest_Status: Status,
  SupportRequest_Priority: Priority,
  SupportRequest_Actor_Type: Actor,
  SupportRequest_Event_Type: Event,
  SupportRequest_Channel: Channel,
  SupportRequest_Attachment_Purpose: Purpose,
} = EnterpriseP;

const states: Record<string, EnterpriseP.SupportInfo_State> = {
  available: State.AVAILABLE,
  "no-license": State.NO_LICENSE,
  "not-entitled": State.NOT_ENTITLED,
  unreachable: State.UNREACHABLE,
  unauthorized: State.UNAUTHORIZED,
};

const messages: Record<string, string> = {
  "no-license": "The Cluster does not have a License",
  "not-entitled":
    "The subscription of the Cluster's License does not include technical support",
  unreachable: "Could not reach the Octelium customer portal from the Cluster",
  unauthorized: "The License has been revoked",
};

const environment = {
  "version.core": "0.48.0",
  "version.enterprise": "0.29.1",
  "version.cordium": "0.15.0",
  regions: "2 (default, us-west)",
  gateways: "6",
  "license.uid": "8b3d4e5f-0a1b-4c2d-9e3f-4a5b6c7d8e9f",
  "license.type": "SUBSCRIPTION",
  "license.plan": "ENTERPRISE E2",
};

const info = EnterpriseP.SupportInfo.create({
  state: states[scenario] ?? State.AVAILABLE,
  message: messages[scenario] ?? "",
  organization:
    scenario === "available" || scenario === "not-entitled"
      ? { uid: ORG, displayName: "Acme Inc." }
      : undefined,
  clusterDomain: "cluster.acme.example",
  supportLevel:
    scenario === "not-entitled"
      ? EnterpriseP.SupportRequest_SupportLevel.NONE
      : EnterpriseP.SupportRequest_SupportLevel.STANDARD,
  limits: {
    maxSubjectLength: 200,
    maxBodyLength: 16000,
    maxAttachmentSize: 3 * 1048576,
    maxAttachmentsPerRequest: 100,
    maxAttachmentsPerMessage: 10,
    attachmentExtensions: ["txt", "log", "gz", "png", "json"],
    autoCloseDays: 7,
  },
  environment,
  portalURL: PORTAL,
  supportEmail: "support@octelium.com",
  installationID: scenario === "available" ? "3f9a1c0d7e2b4a65" : "",
});

const actor = (type: EnterpriseP.SupportRequest_Actor_Type, name = "") =>
  EnterpriseP.SupportRequest_Actor.create({ type, name });

const requests = new Map<string, EnterpriseP.SupportRequest>();

requests.set(
  WAITING,
  EnterpriseP.SupportRequest.create({
    id: WAITING,
    number: 1042,
    subject: "Gateways restart every few minutes after the upgrade to 0.48",
    status: Status.WAITING_ON_CUSTOMER,
    priority: Priority.P2,
    requestedPriority: Priority.P1,
    supportLevel: EnterpriseP.SupportRequest_SupportLevel.STANDARD,
    channel: Channel.CLUSTER,
    reporter: { name: "Jordan Lee", isVerified: false },
    assignee: "Sam Rivera",
    sla: {
      coverage: "BUSINESS",
      firstResponseDueAt: ago(-300),
      firstRespondedAt: ago(90),
      isPaused: true,
    },
    clusterDomain: "cluster.acme.example",
    environment,
    createdAt: ago(240),
    updatedAt: ago(60),
    lastActivityAt: ago(60),
    messageCount: 2,
    attachmentCount: 1,
    events: [
      {
        id: "e1",
        sequence: 1,
        type: Event.CREATED,
        actor: actor(Actor.CLUSTER, "Jordan Lee"),
        channel: Channel.CLUSTER,
        createdAt: ago(240),
        body: "Every gateway restarts with a panic since the upgrade.\n\n<b>Not HTML</b>",
      },
      {
        id: "e2",
        sequence: 2,
        type: Event.ATTACHMENT,
        actor: actor(Actor.CLUSTER, "Jordan Lee"),
        channel: Channel.CLUSTER,
        createdAt: ago(239),
        body: "component-logs.ndjson.gz",
        attachmentIDs: [LOGS],
      },
      {
        id: "e3",
        sequence: 3,
        type: Event.PRIORITY,
        actor: actor(Actor.AGENT, "Sam Rivera"),
        channel: Channel.AGENT,
        createdAt: ago(100),
        from: "P1",
        to: "P2",
        reason: "One Region of two is affected and traffic fails over.",
      },
      {
        id: "e4",
        sequence: 4,
        type: Event.ASSIGNMENT,
        actor: actor(Actor.AGENT, "Sam Rivera"),
        channel: Channel.AGENT,
        createdAt: ago(90),
        to: "Sam Rivera",
      },
      {
        id: "e5",
        sequence: 4,
        type: Event.MESSAGE,
        actor: actor(Actor.AGENT, "Sam Rivera"),
        channel: Channel.AGENT,
        createdAt: ago(90),
        body: "Thanks for the logs. Could you share the output of the Gateway readiness probe?",
      },
      {
        id: "e6",
        sequence: 5,
        type: Event.STATUS,
        actor: actor(Actor.AGENT, "Sam Rivera"),
        channel: Channel.AGENT,
        createdAt: ago(60),
        from: "OPEN",
        to: "WAITING_ON_CUSTOMER",
      },
    ],
    eventsCursor: "earlier",
    attachments: [
      {
        id: LOGS,
        name: "component-logs.ndjson.gz",
        contentType: "application/gzip",
        size: 48213,
        purpose: Purpose.DIAGNOSTIC,
        uploadedBy: actor(Actor.CLUSTER, "Jordan Lee"),
        channel: Channel.CLUSTER,
        createdAt: ago(239),
        isAvailable: true,
      },
    ],
    actions: { canReply: true, canResolve: true, canEscalate: true },
    portalURL: `${PORTAL}/organizations/${ORG}/support/${WAITING}`,
  }),
);

requests.set(
  RESOLVED,
  EnterpriseP.SupportRequest.create({
    id: RESOLVED,
    number: 1038,
    subject: "How to rotate the Cluster certificate",
    status: Status.RESOLVED,
    priority: Priority.P4,
    requestedPriority: Priority.P4,
    supportLevel: EnterpriseP.SupportRequest_SupportLevel.STANDARD,
    channel: Channel.CLUSTER,
    reporter: { name: "Jordan Lee", isVerified: false },
    assignee: "Ana Ruiz",
    clusterDomain: "cluster.acme.example",
    createdAt: ago(60 * 24 * 6),
    updatedAt: ago(60 * 24 * 2),
    lastActivityAt: ago(60 * 24 * 2),
    resolvedAt: ago(60 * 24 * 2),
    messageCount: 3,
    events: [
      {
        id: "r1",
        sequence: 1,
        type: Event.CREATED,
        actor: actor(Actor.CLUSTER, "Jordan Lee"),
        channel: Channel.CLUSTER,
        createdAt: ago(60 * 24 * 6),
        body: "Which command rotates the certificate?",
      },
    ],
    actions: { canReply: true, canResolve: false, canEscalate: false },
    portalURL: `${PORTAL}/organizations/${ORG}/support/${RESOLVED}`,
  }),
);

const summary = (request: EnterpriseP.SupportRequest) =>
  EnterpriseP.SupportRequest.create({
    ...request,
    events: [],
    attachments: [],
    actions: undefined,
    eventsCursor: "",
  });

const logEntry = (
  minutes: number,
  level: CoreP.ComponentLog_Entry_Level,
  message: string,
  fields: Record<string, unknown> = {},
) =>
  CoreP.ComponentLog.create({
    apiVersion: "core/v1",
    kind: "ComponentLog",
    metadata: { id: `log-${minutes}-${level}` },
    entry: {
      time: ago(minutes),
      level,
      message,
      component: { namespace: "octelium", type: "gateway", uid: "gw-1" },
      function: "vigil.(*Server).Run",
      file: "server.go",
      line: 120,
      fields: Struct.fromJson(fields as any),
    },
  });

const logs = [
  logEntry(
    3,
    CoreP.ComponentLog_Entry_Level.ERROR,
    "upstream call failed: Authorization: Bearer abcdefghijklmnop0123",
    {
      password: "hunter2-secret",
      url: "https://api.acme.example/v1/items?api_key=sk_live_123&debug=1",
      owner: "alice@acme.example",
    },
  ),
  logEntry(
    5,
    CoreP.ComponentLog_Entry_Level.PANIC,
    "runtime error: invalid memory address",
  ),
  logEntry(
    8,
    CoreP.ComponentLog_Entry_Level.ERROR,
    "could not refresh eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiIxMjM0In0.c2lnbmF0dXJl",
  ),
];

let created = 0;

const handlers: Record<string, (request: any) => unknown> = {
  getStatus: () => GetStatusResponse.create({}),
  getSupportInfo: () => info,
  listSupportRequest: (request: EnterpriseP.ListSupportRequestOptions) => {
    const items = [...requests.values()]
      .filter((item) =>
        request.filter === EnterpriseP.ListSupportRequestOptions_Filter.OPEN
          ? item.status !== Status.RESOLVED && item.status !== Status.CLOSED
          : request.filter === EnterpriseP.ListSupportRequestOptions_Filter.CLOSED
            ? item.status === Status.RESOLVED || item.status === Status.CLOSED
            : true,
      )
      .map(summary);
    return EnterpriseP.SupportRequestList.create({ items });
  },
  getSupportRequest: (request: EnterpriseP.GetSupportRequestRequest) =>
    requests.get(request.id),
  listSupportRequestEvent: () =>
    EnterpriseP.SupportRequestEventList.create({
      items: [
        {
          id: "e0",
          sequence: 0,
          type: Event.MESSAGE,
          actor: actor(Actor.CLUSTER, "Jordan Lee"),
          channel: Channel.CLUSTER,
          createdAt: ago(300),
          body: "An earlier message",
        },
      ],
    }),
  createSupportRequest: (request: EnterpriseP.CreateSupportRequestRequest) => {
    created += 1;
    const id = `3e4f5a6b-7c8d-4e9f-8a0b-${String(created).padStart(12, "0")}`;
    const ret = EnterpriseP.SupportRequest.create({
      id,
      number: 1042 + created,
      subject: request.subject,
      status: Status.NEW,
      priority: request.priority,
      requestedPriority: request.priority,
      supportLevel: EnterpriseP.SupportRequest_SupportLevel.STANDARD,
      channel: Channel.CLUSTER,
      reporter: { name: "Jordan Lee" },
      clusterDomain: "cluster.acme.example",
      environment,
      createdAt: ago(0),
      lastActivityAt: ago(0),
      messageCount: 1,
      events: [
        {
          id: `${id}-1`,
          sequence: 1,
          type: Event.CREATED,
          actor: actor(Actor.CLUSTER, "Jordan Lee"),
          channel: Channel.CLUSTER,
          createdAt: ago(0),
          body: request.description,
        },
      ],
      actions: { canReply: true, canResolve: true, canEscalate: true },
      portalURL: `${PORTAL}/organizations/${ORG}/support/${id}`,
    });
    requests.set(id, ret);
    return ret;
  },
  addSupportRequestAttachment: (
    request: EnterpriseP.AddSupportRequestAttachmentRequest,
  ) => {
    uploads.push({
      name: request.name,
      purpose: request.purpose,
      data: Array.from(request.data),
    });
    const attachment = EnterpriseP.SupportRequest_Attachment.create({
      id: `4f5a6b7c-8d9e-4f0a-9b1c-${String(uploads.length).padStart(12, "0")}`,
      name: request.name,
      size: request.data.length,
      purpose: request.purpose,
      uploadedBy: actor(Actor.CLUSTER, "Jordan Lee"),
      channel: Channel.CLUSTER,
      createdAt: ago(0),
      isAvailable: true,
    });
    requests.get(request.id)?.attachments.push(attachment);
    return attachment;
  },
  addSupportRequestMessage: (
    request: EnterpriseP.AddSupportRequestMessageRequest,
  ) => {
    const ret = requests.get(request.id)!;
    ret.events.push(
      EnterpriseP.SupportRequest_Event.create({
        id: `m-${ret.events.length}`,
        sequence: ret.events.length + 10,
        type: Event.MESSAGE,
        actor: actor(Actor.CLUSTER, "Jordan Lee"),
        channel: Channel.CLUSTER,
        createdAt: ago(0),
        body: request.body,
        attachmentIDs: request.attachmentIDs,
      }),
    );
    ret.status = Status.OPEN;
    return EnterpriseP.SupportRequest.clone(ret);
  },
  escalateSupportRequest: (
    request: EnterpriseP.EscalateSupportRequestRequest,
  ) => {
    const ret = requests.get(request.id)!;
    ret.isEscalated = true;
    ret.actions = { ...ret.actions!, canEscalate: false };
    return EnterpriseP.SupportRequest.clone(ret);
  },
  resolveSupportRequest: (request: EnterpriseP.ResolveSupportRequestRequest) => {
    const ret = requests.get(request.id)!;
    ret.status = Status.RESOLVED;
    ret.actions = { canReply: true, canResolve: false, canEscalate: false };
    return EnterpriseP.SupportRequest.clone(ret);
  },
  listComponentLog: (request: VisibilityP.ListComponentLogRequest) =>
    VisibilityP.ListComponentLogResponse.create({
      items: logs.filter(
        (item) =>
          request.levels.length === 0 ||
          request.levels.includes(item.entry!.level),
      ),
      listResponseMeta: { totalCount: logs.length, hasMore: false },
    }),
  listComponentLogTopComponent: () =>
    VisibilityP.ListComponentLogTopComponentResponse.create({
      items: [
        { component: { namespace: "octelium", type: "gateway" }, count: 3 },
        { component: { namespace: "octelium", type: "apiserver" }, count: 1 },
      ],
    }),
};

for (const client of [
  clients.getClientSupport(),
  clients.getClientVisibilityComponentLog(),
  clients.getClientUser(),
]) {
  for (const method of client.methods) {
    Object.getPrototypeOf(client)[method.localName] = (
      request: any,
      options?: { abort?: AbortSignal },
    ) => {
      calls.push({ method: method.localName, request });
      if (method.localName === "downloadSupportRequestAttachment") {
        return {
          responses: (async function* () {
            yield EnterpriseP.DownloadSupportRequestAttachmentResponse.create({
              type: {
                oneofKind: "header",
                header: { name: "component-logs.ndjson.gz", size: 4 },
              },
            });
            yield EnterpriseP.DownloadSupportRequestAttachmentResponse.create({
              type: {
                oneofKind: "chunk",
                chunk: new TextEncoder().encode("logs"),
              },
            });
          })(),
        };
      }
      return (async () => {
        if (options?.abort?.aborted) throw new Error("aborted");
        const handler = handlers[method.localName];
        if (!handler) return { response: method.O.create() };
        const response = handler(request);
        if (!response) throw new RpcError("The support request could not be found", "NOT_FOUND");
        return { response };
      })();
    };
  }
}

store.dispatch(
  setStatus({
    status: GetStatusResponse.create({
      user: {
        metadata: { name: "jordan", displayName: "Jordan Lee" },
        spec: { email: "jordan@acme.example" },
      },
    }),
  }),
);

const Location = () => {
  const location = useLocation();
  window.support.location = `${location.pathname}${location.search}`;
  return null;
};

const route = routerSupport();

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: (
        <div
          className="mx-auto p-4"
          style={{ maxWidth: Number(params.get("width") ?? 1280) }}
        >
          <Location />
          <PageHeader />
          <Outlet />
        </div>
      ),
      children: [route],
    },
  ],
  { initialEntries: [params.get("path") ?? "/support"] },
);

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <Provider store={store}>
      <MantineProvider
        theme={themeMantine}
        cssVariablesResolver={cssVariablesResolver}
        forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
      >
        <QueryClientProvider client={queryClient}>
          <SupportAvailabilityContext.Provider value={true}>
            <PageHeaderProvider>
              <Notifications />
              <React.Suspense fallback={null}>
                <RouterProvider router={router} />
              </React.Suspense>
            </PageHeaderProvider>
          </SupportAvailabilityContext.Provider>
        </QueryClientProvider>
      </MantineProvider>
    </Provider>
  </React.StrictMode>,
);
