import * as AccessP from "@/apis/accessv1/accessv1";
import { Drawer, SegmentedControl } from "@mantine/core";
import { Globe2, Server } from "lucide-react";
import * as React from "react";

import { useCatalogServices } from "@/components/Access/hooks";
import { ServiceTypeIcon } from "@/components/Access/icons";
import {
  Badge,
  EmptyState,
  ErrorState,
  Eyebrow,
  IconTile,
  MonoValue,
  Note,
  SkeletonRows,
} from "@/ui";
import {
  formatDuration,
  namespaceFromName,
  serviceModeLabel,
  shortName,
} from "@/utils";

type CatalogDetailsDrawerProps = {
  catalog?: AccessP.UserCatalog;
  opened: boolean;
  onClose: () => void;
};

export const AccessLevelList = (props: { catalog?: AccessP.UserCatalog }) => {
  const levels = props.catalog?.accessLevels ?? [];
  if (levels.length === 0) return null;

  return (
    <div className="flex flex-col gap-2">
      <Eyebrow>Access levels</Eyebrow>
      {levels.map((level) => (
        <div
          key={level.name}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2.5"
        >
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-body font-semibold text-slate-800">
              {level.displayName || level.name}
            </span>
            <Badge>
              {level.type === AccessP.UserCatalog_AccessLevel_Type.RESTRICTED
                ? "Restricted"
                : "Full access"}
            </Badge>
            {level.overridesGuardrails && (
              <Badge tone="amber">Overrides guardrails</Badge>
            )}
            {level.maxDuration && (
              <Badge>Up to {formatDuration(level.maxDuration)}</Badge>
            )}
          </div>
          {level.description && (
            <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
              {level.description}
            </p>
          )}
        </div>
      ))}
    </div>
  );
};

const CatalogDetailsDrawer = (props: CatalogDetailsDrawerProps) => {
  const [view, setView] = React.useState<"services" | "namespaces">("services");
  const servicesQuery = useCatalogServices(
    props.opened ? props.catalog?.metadata?.name : undefined,
  );
  const isDynamic =
    props.catalog?.membership === AccessP.Catalog_Spec_Membership.DYNAMIC;

  const namespaceOf = (service: { metadata?: { name: string }; status?: { namespace: string } }) =>
    service.status?.namespace || namespaceFromName(service.metadata?.name);

  const services = servicesQuery.data ?? [];
  const hiddenCount = Math.max(
    0,
    (props.catalog?.totalService ?? 0) - services.length,
  );

  const serviceCountByNamespace = services.reduce<Record<string, number>>(
    (counts, service) => {
      const namespace = namespaceOf(service);
      if (namespace) counts[namespace] = (counts[namespace] ?? 0) + 1;
      return counts;
    },
    {},
  );

  const namespaces = Object.keys(serviceCountByNamespace).sort();

  return (
    <Drawer
      opened={props.opened}
      onClose={props.onClose}
      position="right"
      size="min(560px, 100vw)"
      title={props.catalog?.metadata?.displayName || props.catalog?.metadata?.name}
      overlayProps={{ backgroundOpacity: 0.3 }}
      transitionProps={{ transition: "slide-left", duration: 200 }}
      styles={{
        title: {
          fontWeight: 600,
          fontSize: "0.9375rem",
          color: "var(--color-slate-900)",
        },
        body: { paddingTop: 0 },
      }}
    >
      <div className="flex flex-col gap-4">
        <div className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5">
          <MonoValue className="border-0 bg-transparent px-0">
            {props.catalog?.metadata?.name}
          </MonoValue>
          <p className="mt-1 text-xs font-normal leading-5 text-slate-600">
            {props.catalog?.metadata?.description ||
              (props.catalog?.allowPartialRequests
                ? "You can request the whole Catalog or only some of its Services."
                : "Every Service collected by this Catalog is granted at once when the request is approved.")}
          </p>
        </div>

        <AccessLevelList catalog={props.catalog} />

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

        {servicesQuery.isLoading ? (
          <SkeletonRows rows={4} />
        ) : servicesQuery.isError && !servicesQuery.data ? (
          <ErrorState
            title="Could not load the contents of this Catalog"
            error={servicesQuery.error}
            onRetry={() => {
              void servicesQuery.refetch();
            }}
          />
        ) : view === "services" ? (
          services.length === 0 ? (
            <EmptyState
              icon={<Server size={18} strokeWidth={2} />}
              title="No services visible to you"
              description="None of the Services in this Catalog are currently visible to you."
            />
          ) : (
            <div className="flex flex-col gap-2">
              <Eyebrow>Included services</Eyebrow>
              {hiddenCount > 0 && (
                <Note>
                  {hiddenCount} Service{hiddenCount === 1 ? " is" : "s are"} not
                  shown.
                </Note>
              )}
              {services.map((service) => {
                const name = service.metadata?.name ?? "";
                const namespace = namespaceOf(service);
                const location =
                  service.status?.primaryHostname ||
                  service.status?.addresses?.join(", ");

                return (
                  <div
                    key={service.metadata?.uid || name}
                    className="flex items-start gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
                  >
                    <IconTile>
                      <ServiceTypeIcon serviceType={service.spec?.type} size={15} strokeWidth={2} />
                    </IconTile>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-body font-semibold text-slate-800">
                        {service.metadata?.displayName || shortName(name)}
                      </p>
                      <p className="truncate font-mono text-xs font-normal text-slate-500">
                        {name}
                      </p>
                      <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
                        <Badge>{serviceModeLabel(service.spec?.type)}</Badge>
                        {namespace && <Badge>Namespace {namespace}</Badge>}
                        {(service.spec?.port ?? 0) > 0 && (
                          <Badge>Port {service.spec!.port}</Badge>
                        )}
                        {service.spec?.isTLS && <Badge>TLS</Badge>}
                        {service.spec?.isPublic && <Badge>Public</Badge>}
                      </div>
                      {location ? (
                        <p className="mt-1.5 font-mono text-xs font-normal text-slate-500 [overflow-wrap:anywhere]">
                          {location}
                        </p>
                      ) : null}
                    </div>
                  </div>
                );
              })}
            </div>
          )
        ) : namespaces.length === 0 ? (
          <EmptyState
            icon={<Globe2 size={18} strokeWidth={2} />}
            title="No namespaces in this Catalog"
            description="The Catalog does not include any namespace visible to you."
          />
        ) : (
          <div className="flex flex-col gap-2">
            <Eyebrow>Included namespaces</Eyebrow>
            {namespaces.map((name) => {
              const count = serviceCountByNamespace[name] ?? 0;
              return (
                <div
                  key={name}
                  className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2.5"
                >
                  <IconTile>
                    <Globe2 size={15} strokeWidth={2} />
                  </IconTile>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-mono text-body font-semibold text-slate-800">
                      {name}
                    </p>
                    <p className="mt-0.5 text-xs font-normal text-slate-500">
                      {count} Service{count === 1 ? "" : "s"}
                      {isDynamic &&
                        " · New Services are included automatically"}
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
