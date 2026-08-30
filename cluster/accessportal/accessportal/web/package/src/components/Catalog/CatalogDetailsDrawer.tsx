import * as AccessP from "@/apis/accessv1/accessv1";
import { Drawer, SegmentedControl } from "@mantine/core";
import { useQuery } from "@tanstack/react-query";
import { Globe2, Lock, Radio, Server } from "lucide-react";
import * as React from "react";

import { listAllNamespaces, listAllServices } from "@/components/Access/hooks";
import { serviceTypeIcon } from "@/components/Access/icons";
import {
  Badge,
  EmptyState,
  ErrorState,
  Eyebrow,
  IconTile,
  Loading,
  MonoValue,
  SkeletonRows,
} from "@/ui";
import { namespaceFromName, serviceModeMeta, shortName } from "@/utils";

type CatalogDetailsDrawerProps = {
  catalog?: AccessP.Catalog;
  opened: boolean;
  onClose: () => void;
};

const CatalogDetailsDrawer = (props: CatalogDetailsDrawerProps) => {
  const [view, setView] = React.useState<"services" | "namespaces">("services");
  const scope = props.catalog?.spec?.resourceCollection?.service;
  const serviceNames = new Set(scope?.services ?? []);
  const scopeNamespaces = new Set(scope?.namespaces ?? []);

  const servicesQuery = useQuery({
    queryKey: ["userapi", "catalogServices", props.catalog?.metadata?.name],
    enabled: props.opened && !!props.catalog,
    queryFn: () => listAllServices(),
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

  const serviceCountByNamespace = services.reduce<Record<string, number>>(
    (counts, service) => {
      const namespace =
        service.status?.namespace || namespaceFromName(service.metadata?.name);
      if (namespace) counts[namespace] = (counts[namespace] ?? 0) + 1;
      return counts;
    },
    {},
  );

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="min(560px, 100vw)"
      title={props.catalog?.metadata?.displayName || props.catalog?.metadata?.name}
      overlayProps={{ backgroundOpacity: 0.35, blur: 2 }}
      transitionProps={{ transition: "slide-left", duration: 220 }}
      styles={{
        title: { fontWeight: 700, fontSize: "0.95rem" },
        body: { paddingTop: 0 },
      }}
    >
      <div className="flex flex-col gap-4">
        <div className="rounded-lg border border-slate-200 bg-slate-50/70 px-3 py-2.5">
          <MonoValue className="bg-transparent px-0">
            {props.catalog?.metadata?.name}
          </MonoValue>
          <p className="mt-1.5 text-[0.76rem] font-medium leading-relaxed text-slate-500">
            {props.catalog?.metadata?.description ||
              "Every Service collected by this Catalog is granted at once when the request is approved."}
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
            <SkeletonRows rows={4} />
          ) : servicesQuery.isError ? (
            <ErrorState
              title="Could not load catalog services"
              onRetry={() => {
                void servicesQuery.refetch();
              }}
            />
          ) : services.length === 0 ? (
            <EmptyState
              icon={<Server size={20} strokeWidth={2} />}
              title="No services in this Catalog"
              description="The Catalog does not contain any services available to you."
            />
          ) : (
            <div className="flex flex-col gap-2">
              <Eyebrow>Included services</Eyebrow>
              {services.map((service) => {
                const name = service.metadata?.name ?? "";
                const displayName =
                  service.metadata?.displayName || shortName(name);
                const namespace =
                  service.status?.namespace || namespaceFromName(name);
                const mode = serviceModeMeta(service.spec?.type);
                const Icon = serviceTypeIcon(service.spec?.type);

                return (
                  <div
                    key={service.metadata?.uid || name}
                    className="flex items-start gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
                  >
                    <IconTile tone={mode.tone}>
                      <Icon size={15} strokeWidth={2.2} />
                    </IconTile>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-[0.8rem] font-bold text-slate-800">
                        {displayName}
                      </p>
                      <p className="mt-0.5 truncate font-mono text-[0.68rem] font-semibold text-slate-400">
                        {name}
                      </p>
                      <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                        <Badge tone={mode.tone}>{mode.label}</Badge>
                        {namespace && <Badge tone="slate">{namespace}</Badge>}
                        {(service.spec?.port ?? 0) > 0 && (
                          <Badge tone="slate">Port {service.spec!.port}</Badge>
                        )}
                        {service.spec?.isTLS && (
                          <Badge tone="emerald" icon={<Lock size={9} strokeWidth={3} />}>
                            TLS
                          </Badge>
                        )}
                        {service.spec?.isPublic && (
                          <Badge tone="amber" icon={<Radio size={9} strokeWidth={3} />}>
                            Public
                          </Badge>
                        )}
                      </div>
                      {(service.status?.primaryHostname ||
                        service.status?.addresses.length) && (
                        <p className="mt-1.5 break-all font-mono text-[0.66rem] font-semibold text-slate-400">
                          {service.status.primaryHostname ||
                            service.status.addresses.join(", ")}
                        </p>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )
        ) : namespacesQuery.isLoading ? (
          <Loading label="Loading catalog namespaces..." />
        ) : namespacesQuery.isError ? (
          <ErrorState
            title="Could not load catalog namespaces"
            onRetry={() => {
              void namespacesQuery.refetch();
            }}
          />
        ) : namespaces.length === 0 ? (
          <EmptyState
            icon={<Globe2 size={20} strokeWidth={2} />}
            title="No namespaces in this Catalog"
            description="The Catalog does not contain any namespaces available to you."
          />
        ) : (
          <div className="flex flex-col gap-2">
            <Eyebrow>Included namespaces</Eyebrow>
            {namespaces.map((namespace) => {
              const name = namespace.metadata?.name ?? "";
              const count = serviceCountByNamespace[name] ?? 0;
              return (
                <div
                  key={namespace.metadata?.uid || name}
                  className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
                >
                  <IconTile tone="blue">
                    <Globe2 size={15} strokeWidth={2.2} />
                  </IconTile>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-mono text-[0.8rem] font-bold text-slate-800">
                      {name}
                    </p>
                    <p className="mt-0.5 text-[0.7rem] font-semibold text-slate-400">
                      {count} service{count === 1 ? "" : "s"} available to you
                    </p>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </Drawer>
  );
};

export default CatalogDetailsDrawer;
