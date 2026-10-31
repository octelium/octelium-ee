import "@/index.css";
import "@mantine/core/styles.layer.css";
import {
  Service,
  Service_Spec_Mode,
  User,
  UserList,
} from "@/apis/corev1/corev1";
import { Timestamp } from "@/apis/google/protobuf/timestamp";
import { GetStatusResponse } from "@/apis/userv1/userv1";
import { ErrorScreen } from "@/components/ErrorBoundary";
import ResourceInfo from "@/components/ResourceLayout/ResourceInfo";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import ResourceListPage from "@/components/ResourceLayout/ResourceList";
import ResourceRow from "@/components/ResourceLayout/resourcePicker/ResourceRow";
import { SelectedChip } from "@/components/ResourceLayout/resourcePicker/TriggerField";
import TopBar from "@/components/TopBar";
import TopList from "@/components/TopList";
import { setStatus } from "@/features/settings/slice";
import Settings from "@/pages/Settings";
import serviceInfo from "@/pages/core/Service/router";
import Leaderboard, { fromResource } from "@/pages/visibility/Main/Leaderboard";
import store from "@/store";
import { getDomain, isDev } from "@/utils";
import {
  getClientVisibilityCore,
  getClientUser,
  getTransport,
} from "@/utils/client";
import {
  buildErrorReport,
  formatErrorReport,
  logAppError,
} from "@/utils/errors";
import { getGetKeyFromPath } from "@/utils/pb";
import { MantineProvider } from "@mantine/core";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Users } from "lucide-react";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { MemoryRouter, Route, Routes } from "react-router-dom";

const mode = new URLSearchParams(window.location.search).get("mode");
const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false, staleTime: Infinity } },
});
const user = (name: string) =>
  User.create({
    apiVersion: "core/v1",
    kind: "User",
    metadata: {
      name,
      uid: name,
      picURL: `https://avatars.example.test/${name}.svg`,
      createdAt: Timestamp.fromDate(new Date("2026-10-12T00:00:00Z")),
    },
    spec: {},
  });

(window as any).security = {
  getDomain,
  isDev,
  transportOptions: () => getTransport().mergeOptions({}),
  requestStatus: () => getClientUser().getStatus({}),
  buildErrorReport,
  formatErrorReport,
  logAppError,
};

let content: React.ReactNode = <div>Security fixture</div>;
let path = "/";
if (mode === "avatars") {
  path = "/core/users/overview";
  queryClient.setQueryData([getGetKeyFromPath(path), "overview"], {
    response: user("overview"),
  });
  store.dispatch(
    setStatus({
      status: GetStatusResponse.create({
        user: { metadata: user("settings").metadata, spec: {} },
        session: {
          metadata: { picURL: "https://avatars.example.test/topbar.svg" },
        },
      }),
    }),
  );
  content = (
    <div className="flex flex-col gap-4 p-4">
      <TopBar />
      <Settings />
      <ResourceItemMainPage unDeletable />
      <ResourceInfo resource={user("info")} />
      <TopList
        title="Users"
        items={[{ resource: user("top-list"), count: 1 }]}
      />
      <Leaderboard
        title="Users"
        icon={Users}
        items={[fromResource(user("leaderboard"), 1)!]}
        emptyLabel="No users"
      />
      <SelectedChip
        name="selected"
        item={user("selected")}
        api="core"
        kind="User"
      />
      <ul>
        <ResourceRow
          item={user("row")}
          api="core"
          kind="User"
          selected={false}
          active={false}
          onPick={() => {}}
          onPeek={() => {}}
        />
      </ul>
    </div>
  );
} else if (mode === "list") {
  path = "/core/users";
  Object.getPrototypeOf(getClientVisibilityCore()).listUser = async () => ({
    response: UserList.create({
      items: [user("list")],
      listResponseMeta: { totalCount: 1 },
    }),
  });
  content = (
    <ResourceListPage
      info={{
        API: "core",
        Kind: "User",
        List: {},
        Item: {},
        unDeletable: true,
      }}
    />
  );
} else if (mode === "service") {
  path = "/core/services/web";
  queryClient.setQueryData([getGetKeyFromPath(path), "web"], {
    response: Service.create({
      apiVersion: "core/v1",
      kind: "Service",
      metadata: { name: "web", uid: "web" },
      spec: { mode: Service_Spec_Mode.WEB, isPublic: true },
      status: {},
    }),
  });
  content = (
    <ResourceItemMainPage
      infoComponent={serviceInfo.infoItemsGetter}
      unDeletable
    />
  );
} else if (mode === "error") {
  content = (
    <ErrorScreen
      report={buildErrorReport(new Error("Fixture failure"), {
        source: "fixture",
      })}
    />
  );
}

createRoot(document.getElementById("root")!).render(
  <MantineProvider>
    <Provider store={store}>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route path="/core/users/:name" element={content} />
            <Route path="/core/services/:name" element={content} />
            <Route path="*" element={content} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    </Provider>
  </MantineProvider>,
);
