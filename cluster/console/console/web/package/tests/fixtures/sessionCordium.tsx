import "@mantine/core/styles.css";

import React from "react";
import ReactDOM from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import { JsonObject } from "@protobuf-ts/runtime";
import { QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Route, Routes } from "react-router-dom";

import { Session } from "@/apis/corev1/corev1";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import sessionInfo from "@/pages/core/Session/router";
import { queryClient } from "@/utils";
import themeMantine from "@/utils/theme/mantine";

const cordium = {
  workspaceRef: {
    apiVersion: "cordium/v1",
    kind: "Workspace",
    name: "aoo",
    uid: "69dd3308-1a9c-4cee-ac37-0e6743c0c7bc",
  },
  spaceRef: {
    apiVersion: "cordium/v1",
    kind: "Space",
    name: "default.usr1",
    uid: "03f7617c-6d42-4f8c-b953-038289d542d7",
  },
  templateRef: {
    apiVersion: "cordium/v1",
    kind: "Template",
    name: "default.default.usr1",
    uid: "276aa026-af86-47d2-b416-f6e71e53333a",
  },
  spaceType: "USER",
};

const extensions: Record<string, JsonObject> = {
  workspace: { cordium },
  organization: { cordium: { ...cordium, spaceType: 2, futureField: true } },
  partial: {
    cordium: {
      workspaceRef: { name: "aoo" },
      spaceRef: {},
      templateRef: {},
    },
  },
  uid: {
    cordium: {
      workspaceRef: { uid: cordium.workspaceRef.uid },
    },
  },
  malformed: { cordium: { ...cordium, workspaceRef: "invalid" } },
  empty: { cordium: {} },
  normal: {},
  enterprise: { enterprise: { isEnabled: true } },
};

const scenario = new URLSearchParams(window.location.search).get("scenario");
const item = Session.fromJson({
  apiVersion: "core/v1",
  kind: "Session",
  metadata: {
    name: "usr1-fvhdnl",
    uid: "60eac7f3-dfe9-4294-916d-25b035c23f1d",
  },
  spec: { state: "ACTIVE" },
  ...(scenario === "no-status"
    ? {}
    : {
        status: {
          type: "CLIENT",
          userRef: { apiVersion: "core/v1", kind: "User", name: "usr1" },
          deviceRef: { apiVersion: "core/v1", kind: "Device", name: "dev1" },
          ext: extensions[scenario ?? "workspace"],
        },
      }),
});

queryClient.setQueryData(["core.getSession", item.metadata!.name], {
  response: item,
});
window.history.replaceState(null, "", `/core/sessions/${item.metadata!.name}`);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MantineProvider theme={themeMantine}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Routes>
            <Route
              path="/core/sessions/:name"
              element={
                <ResourceItemMainPage
                  infoComponent={sessionInfo.infoItemsGetter}
                  unDeletable
                />
              }
            />
            <Route path="*" element={<span>Reference</span>} />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </MantineProvider>
  </React.StrictMode>,
);
