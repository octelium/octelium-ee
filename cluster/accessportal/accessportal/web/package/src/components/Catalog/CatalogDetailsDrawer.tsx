import * as AccessP from "@/apis/accessv1/accessv1";
import * as UserP from "@/apis/userv1/userv1";
import { Drawer, SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { Boxes, Globe2, Server } from "lucide-react";
import * as React from "react";

import {
  Badge,
  Card,
  EmptyState,
  ErrorState,
  Loading,
  SectionTitle,
} from "../../ui";
import { namespaceFromName, serviceModeMeta, shortName } from "../../utils";
import { getUserMainClient } from "../../utils/client";

type CatalogDetailsDrawerProps = {
  catalog?: AccessP.Catalog;
  opened: boolean;
  onClose: () => void;
};

const listAllServices = async (): Promise<UserP.Service[]> => {
  const items: UserP.Service[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listService(
      UserP.ListServiceOptions.create({
        common: { page, itemsPerPage: 500 },
        namespace: "",
      }),
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > 1000) break;
    page += 1;
  }

  return items;
};

const listAllNamespaces = async (): Promise<UserP.Namespace[]> => {
  const items: UserP.Namespace[] = [];
  let page = 0;

  for (;;) {
    const { response } = await getUserMainClient().listNamespace(
      UserP.ListNamespaceOptions.create({
        common: { page, itemsPerPage: 500 },
      }),
    );
    items.push(...response.items);
    if (!response.listResponseMeta?.hasMore || page > 1000) break;
    page += 1;
  }

  return items;
};

const CatalogDetailsDrawer = (props: CatalogDetailsDrawerProps) => {
  const [view, setView] = React.useState<"services" | "namespaces">("services");
  const scope = props.catalog?.spec?.resourceCollection?.service;
  const serviceNames = new Set(scope?.services ?? []);
  const scopeNamespaces = new Set(scope?.namespaces ?? []);

  const servicesQuery = useQuery({
    queryKey: ["userapi", "catalogServices", props.catalog?.metadata?.name],
    enabled: props.opened && !!props.catalog,
    queryFn: listAllServices,
  });
  const namespacesQuery = useQuery({
    queryKey: ["userapi", "catalogNamespaces", props.catalog?.metadata?.name],
    enabled: props.opened && !!props.catalog,
    queryFn: listAllNamespaces,
  });

  const services = (servicesQuery.data ?? []).filter((service) => {
    const name = service.metadata?.name ?? "";
    const namespace = service.status?.namespace || namespaceFromName(name);
    return (
      serviceNames.size === 0 ||
      serviceNames.has(name) ||
      scopeNamespaces.has(namespace)
    );
  });
  const catalogNamespaceNames =
    scopeNamespaces.size > 0
      ? scopeNamespaces
      : serviceNames.size > 0
        ? new Set(
            services.map(
              (service) =>
                service.status?.namespace ||
                namespaceFromName(service.metadata?.name),
            ),
          )
        : undefined;
  const namespaces = (namespacesQuery.data ?? []).filter((namespace) => {
    const name = namespace.metadata?.name ?? "";
    return !catalogNamespaceNames || catalogNamespaceNames.has(name);
  });
  const namespaceNames = new Set(
    services.map(
      (service) =>
        service.status?.namespace ||
        namespaceFromName(service.metadata?.name),
    ),
  );
  const serviceCountByNamespace = services.reduce<Record<string, number>>(
    (counts, service) => {
      const namespace =
        service.status?.namespace || namespaceFromName(service.metadata?.name);
      if (namespace) counts[namespace] = (counts[namespace] ?? 0) + 1;
      return counts;
    },
    {},
  );

  const renderError = (message: string, onRetry: () => void) => (
    <ErrorState title={message} onRetry={onRetry} />
  );

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="min(520px, 100vw)"
      title={props.catalog?.metadata?.displayName || props.catalog?.metadata?.name}
      overlayProps={{ backgroundOpacity: 0.35, blur: 2 }}
      transitionProps={{ transition: "slide-left", duration: 220 }}
      styles={{
        title: { fontWeight: 700, fontSize: "0.95rem" },
        body: { paddingTop: 0 },
      }}
    >
      <div className="flex flex-col gap-4">
        <div>
          <p className="font-mono text-[0.72rem] font-semibold text-slate-400">
            {props.catalog?.metadata?.name}
          </p>
          <p className="mt-2 text-[0.78rem] font-medium leading-relaxed text-slate-500">
            Services and namespaces included in this Catalog and available for
            access requests.
          </p>
        </div>

        <SegmentedControl
          fullWidth
          value={view}
          onChange={(value) => setView(value as "services" | "namespaces")}
          data={[
            {
              value: "services",
              label: `Services${services.length ? ` (${services.length})` : ""}`,
            },
            {
              value: "namespaces",
              label: `Namespaces${namespaces.length ? ` (${namespaces.length})` : ""}`,
            },
          ]}
        />

        {view === "services" ? (
          servicesQuery.isLoading ? (
            <Loading label="Loading catalog services..." />
          ) : servicesQuery.isError ? (
            renderError("Could not load catalog services", () => {
              void servicesQuery.refetch();
            })
          ) : services.length === 0 ? (
            <EmptyState
              icon={<Server size={20} strokeWidth={2} />}
              title="No services in this Catalog"
              description="The Catalog does not contain any services available to you."
            />
          ) : (
            <div className="flex flex-col gap-2">
              <SectionTitle>Included services</SectionTitle>
              {services.map((service) => {
                const name = service.metadata?.name ?? "";
                const displayName = service.metadata?.displayName || shortName(name);
                const namespace =
                  service.status?.namespace || namespaceFromName(name);
                const mode = serviceModeMeta(service.spec?.type);
                const port = service.spec?.port ?? 0;
                const isTLS = service.spec?.isTLS ?? false;
                const isPublic = service.spec?.isPublic ?? false;
                return (
                  <Card key={service.metadata?.uid || name} className="p-3">
                    <div className="flex items-start gap-3">
                      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
                        <Boxes size={15} strokeWidth={2.2} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="truncate text-[0.8rem] font-bold text-slate-800">
                            {displayName}
                          </p>
                          <Badge tone={mode.tone}>{mode.label}</Badge>
                        </div>
                        <p className="mt-0.5 truncate font-mono text-[0.68rem] font-semibold text-slate-400">
                          {name}
                        </p>
                        <div className="mt-2 flex flex-wrap items-center gap-1.5">
                          {namespace && <Badge tone="slate">{namespace}</Badge>}
                          {port > 0 && (
                            <Badge tone="slate">Port {port}</Badge>
                          )}
                          {isTLS && <Badge tone="emerald">TLS</Badge>}
                          {isPublic && <Badge tone="amber">Public</Badge>}
                        </div>
                        {(service.status?.primaryHostname ||
                          service.status?.addresses.length) && (
                          <p className="mt-2 break-all font-mono text-[0.68rem] font-semibold text-slate-400">
                            {service.status.primaryHostname ||
                              service.status.addresses.join(", ")}
                          </p>
                        )}
                      </div>
                    </div>
                  </Card>
                );
              })}
            </div>
          )
        ) : namespacesQuery.isLoading ? (
          <Loading label="Loading catalog namespaces..." />
        ) : namespacesQuery.isError ? (
          renderError("Could not load catalog namespaces", () => {
            void namespacesQuery.refetch();
          })
        ) : namespaces.length === 0 ? (
          <EmptyState
            icon={<Globe2 size={20} strokeWidth={2} />}
            title="No namespaces in this Catalog"
            description="The Catalog does not contain any namespaces available to you."
          />
        ) : (
          <div className="flex flex-col gap-2">
            <SectionTitle>Included namespaces</SectionTitle>
            {namespaces.map((namespace) => {
              const name = namespace.metadata?.name ?? "";
              return (
                <Card key={namespace.metadata?.uid || name} className="p-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100 text-slate-500">
                      <Globe2 size={15} strokeWidth={2.2} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-[0.8rem] font-bold text-slate-800">
                        {name}
                      </p>
                      <p className="mt-0.5 text-[0.7rem] font-semibold text-slate-400">
                        {serviceCountByNamespace[name] ?? 0} services in this namespace
                      </p>
                    </div>
                    <Badge tone={namespaceNames.has(name) ? "blue" : "slate"}>
                      {namespaceNames.has(name) ? "Included" : "Namespace"}
                    </Badge>
                  </div>
                </Card>
              );
            })}
          </div>
        )}
      </div>
    </Drawer>
  );
};

export default CatalogDetailsDrawer;
