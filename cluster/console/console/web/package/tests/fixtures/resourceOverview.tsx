import "@/index.css";
import "@mantine/core/styles.layer.css";
import "@fontsource/ubuntu/400.css";
import "@fontsource/ubuntu/500.css";
import "@fontsource/ubuntu/700.css";
import ResourceItemDrawer from "@/components/ResourceLayout/ResourceItemDrawer";
import ResourceItemMainPage from "@/components/ResourceLayout/ResourceItemMainPage";
import { getResourceComponentMap } from "@/pages/utils/resourceRegistry";
import store from "@/store";
import * as clients from "@/utils/client";
import { getAPIKindFromPath, getGetKeyFromPath, getResourcePathFromAPIKind } from "@/utils/pb";
import cssVariablesResolver from "@/utils/theme/cssVariables";
import themeMantine from "@/utils/theme/mantine";
import { MantineProvider } from "@mantine/core";
import { Notifications } from "@mantine/notifications";
import { RpcError } from "@protobuf-ts/runtime-rpc";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MotionConfig } from "framer-motion";
import React from "react";
import { createRoot } from "react-dom/client";
import { Provider } from "react-redux";
import { createMemoryRouter, Outlet, RouterProvider, useLocation } from "react-router-dom";
import { errors, resources } from "./resourceOverviewData";

interface Call {
  method: string;
  request: unknown;
}

declare global {
  interface Window {
    overview: {
      calls: Call[];
      errors: string[];
      navigate: (path: string) => void;
    };
  }
}

const params = new URLSearchParams(window.location.search);
const calls: Call[] = [];

const apiOf = (typeName: string) =>
  /\.(core|enterprise|access|cordium)\.v1\./.exec(typeName)?.[1];

const byKind = (kind: string, api?: string) =>
  Object.values(resources).filter(
    (item) =>
      item.kind === kind && (!api || item.apiVersion === `${api}/v1`),
  );

interface Method {
  localName: string;
  O: { typeName: string; create: () => Record<string, unknown> };
}

const stub = (client: { methods: readonly Method[] }) => {
  const prototype = Object.getPrototypeOf(client) as Record<string, unknown>;
  for (const method of client.methods) {
    prototype[method.localName] = (request: Request) => {
      const response = respond(method, request);
      const call = Object.assign(
        response.then((value) => ({ response: value })),
        { response },
      );
      response.catch(() => undefined);
      call.catch(() => undefined);
      return call;
    };
  }
};

type Request = { name?: string; uid?: string } | undefined;

const respond = async (method: Method, request: Request) => {
  calls.push({ method: method.localName, request });
  const api = apiOf(method.O.typeName);
  const get = /^get([A-Z]\w+)$/.exec(method.localName);
  if (get) {
    const item = byKind(get[1], api).find(
      (entry) =>
        entry.metadata?.name === request?.name ||
        (!!request?.uid && entry.metadata?.uid === request.uid),
    );
    if (item) return item;
    if (byKind(get[1], api).length > 0 || method.O.typeName.endsWith(get[1]))
      throw new RpcError("Not found", "NOT_FOUND");
  }
  const list = /^list([A-Z]\w+)$/.exec(method.localName);
  const response = method.O.create();
  if (list && "items" in response) {
    const items = byKind(list[1], api);
    response.items = items;
    response.listResponseMeta = { totalCount: items.length };
  }
  return response;
};

for (const getter of [
  clients.getClientCore,
  clients.getClientEnterprise,
  clients.getClientAccess,
  clients.getClientCordium,
  clients.getClientVisibilityCore,
  clients.getClientVisibilityEnterprise,
  clients.getClientVisibilityAccess,
  clients.getClientVisibilityCordium,
  clients.getClientVisibilityAccessLog,
  clients.getClientVisibilityAuthenticationLog,
  clients.getClientVisibilityAuditLog,
]) {
  stub(getter());
}

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false, staleTime: Infinity } },
});

for (const [path, item] of Object.entries(resources)) {
  queryClient.setQueryData([getGetKeyFromPath(`/${path}`), item.metadata!.name], {
    response: item,
  });
}

const Frame = () => {
  const location = useLocation();
  return (
    <>
      <output data-testid="location" className="sr-only">
        {location.pathname}
      </output>
      <Outlet />
    </>
  );
};

const routes = [...getResourceComponentMap().values()].map((info) => ({
  path: `${info.API}/${getResourcePathFromAPIKind({ api: info.API, kind: info.Kind })}/:name`,
  element: <ResourceItemDrawer />,
  children: [
    {
      path: "",
      element: (
        <ResourceItemMainPage
          infoComponent={info.infoItemsGetter}
          mainAction={info.Item.MainAction}
          unDeletable={info.unDeletable}
          cloneable={info.cloneable}
        />
      ),
    },
    { path: "*", element: <div>Tab</div> },
  ],
}));

const initialPath = params.get("path") ?? "/core/sessions/alice-browser-k3x9";
if (!getAPIKindFromPath(initialPath)) errors.push(`unknown path ${initialPath}`);

const router = createMemoryRouter(
  [
    {
      path: "/",
      element: <Frame />,
      children: [...routes, { path: "*", element: <div /> }],
    },
  ],
  { initialEntries: [initialPath] },
);

window.overview = {
  calls,
  errors,
  navigate: (path: string) => void router.navigate(path),
};

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <MotionConfig reducedMotion="always">
      <Provider store={store}>
        <MantineProvider
          theme={themeMantine}
          cssVariablesResolver={cssVariablesResolver}
          forceColorScheme={params.get("theme") === "dark" ? "dark" : "light"}
        >
          <Notifications />
          <QueryClientProvider client={queryClient}>
            <RouterProvider router={router} />
          </QueryClientProvider>
        </MantineProvider>
      </Provider>
    </MotionConfig>
  </React.StrictMode>,
);
