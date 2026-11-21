import * as AccessP from "@/apis/accessv1/accessv1";
import { Button } from "@mantine/core";
import { Layers, PanelRightOpen, ServerCog } from "lucide-react";
import * as React from "react";

import CatalogDetailsDrawer from "@/components/Catalog/CatalogDetailsDrawer";
import {
  Badge,
  Eyebrow,
  IconTile,
  InfoGrid,
  KeyValue,
  MonoValue,
  Note,
  SectionCard,
} from "@/ui";
import {
  namespaceFromName,
  requestResourceLabel,
  serviceModeLabel,
  shortName,
} from "@/utils";
import { describeRpcError, isNotFoundError } from "@/utils/rpcError";

import { ServiceTypeIcon } from "./icons";
import { useCatalogs, useService } from "./hooks";

const ResourceHeading = (props: {
  icon: React.ReactNode;
  title: string;
  name: string;
  badges: React.ReactNode;
  action?: React.ReactNode;
}) => (
  <div className="flex items-start gap-3">
    <IconTile size="lg">{props.icon}</IconTile>
    <div className="min-w-0 flex-1">
      <p className="text-body font-semibold text-slate-900 [overflow-wrap:anywhere]">
        {props.title}
      </p>
      <p className="mt-0.5 font-mono text-xs font-normal text-slate-500 [overflow-wrap:anywhere]">
        {props.name}
      </p>
      <div className="mt-2 flex flex-wrap items-center gap-1.5">{props.badges}</div>
    </div>
    {props.action}
  </div>
);

const ServiceResource = (props: { name: string }) => {
  const query = useService(props.name);
  const service = query.data;
  const namespace = service?.status?.namespace || namespaceFromName(props.name);

  return (
    <div className="flex flex-col gap-4">
      <ResourceHeading
        icon={<ServiceTypeIcon serviceType={service?.spec?.type} size={18} strokeWidth={2} />}
        title={service?.metadata?.displayName || shortName(props.name)}
        name={props.name}
        badges={
          <>
            {service && <Badge>{serviceModeLabel(service.spec?.type)}</Badge>}
            {namespace && <Badge>Namespace {namespace}</Badge>}
            {service?.spec?.isTLS && <Badge>TLS</Badge>}
            {service?.spec?.isPublic && <Badge>Public</Badge>}
          </>
        }
      />

      {service ? (
        <InfoGrid className="border-t border-slate-100 pt-4">
          <KeyValue label="Primary hostname" mono>
            {service.status?.primaryHostname || "—"}
          </KeyValue>
          <KeyValue label="Port" mono>
            {service.spec?.port || "—"}
          </KeyValue>
          {service.status?.addresses?.length ? (
            <KeyValue label="Addresses" mono full>
              {service.status.addresses.join(", ")}
            </KeyValue>
          ) : null}
          {service.metadata?.description && (
            <KeyValue label="Description" full>
              <span className="font-normal text-slate-700">
                {service.metadata.description}
              </span>
            </KeyValue>
          )}
        </InfoGrid>
      ) : query.isLoading ? (
        <div aria-hidden className="flex flex-col gap-2 border-t border-slate-100 pt-4">
          <div className="h-3 w-1/3 rounded bg-slate-100 motion-safe:animate-pulse" />
          <div className="h-3 w-1/2 rounded bg-slate-50 motion-safe:animate-pulse" />
        </div>
      ) : isNotFoundError(query.error) ? (
        <Note icon={<ServerCog size={13} strokeWidth={2.25} />}>
          The details of this Service are not visible to you right now.
        </Note>
      ) : (
        <Note tone="amber">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{describeRpcError(query.error).message}</span>
            <Button size="compact-xs" variant="default" onClick={() => query.refetch()}>
              Try again
            </Button>
          </div>
        </Note>
      )}
    </div>
  );
};

const CatalogResource = (props: {
  request: AccessP.Request;
  reviewer?: boolean;
}) => {
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const resource = requestResourceLabel(props.request);
  const scope = props.request.status?.resolvedScope;
  const query = useCatalogs();
  const catalog = query.data?.find(
    (item) => item.metadata?.name === resource.name,
  );
  const services =
    scope?.serviceRefs.map((ref) => ref.name) ?? resource.services;
  const namespaces = scope?.namespaceRefs.map((ref) => ref.name) ?? [];
  const isPartial = resource.services.length > 0;
  const isDynamic =
    scope?.membership === AccessP.Catalog_Spec_Membership.DYNAMIC;

  return (
    <div className="flex flex-col gap-4">
      <ResourceHeading
        icon={<Layers size={18} strokeWidth={2} />}
        title={
          resource.displayName ||
          catalog?.metadata?.displayName ||
          resource.name
        }
        name={resource.name}
        badges={
          <>
            <Badge>Catalog</Badge>
            {resource.accessLevel && <Badge>{resource.accessLevel}</Badge>}
            {services.length > 0 && (
              <Badge>
                {services.length} service{services.length === 1 ? "" : "s"}
              </Badge>
            )}
            {isPartial && <Badge tone="blue">Selected services only</Badge>}
          </>
        }
        action={
          catalog && (
            <Button
              variant="default"
              size="compact-sm"
              leftSection={<PanelRightOpen size={13} strokeWidth={2.25} />}
              onClick={() => setDrawerOpen(true)}
            >
              Contents
            </Button>
          )
        }
      />

      {catalog?.metadata?.description && (
        <p className="text-body font-normal leading-relaxed text-slate-700">
          {catalog.metadata.description}
        </p>
      )}

      {(services.length > 0 || namespaces.length > 0) && (
        <div className="flex flex-col gap-3 border-t border-slate-100 pt-4">
          {namespaces.length > 0 && isDynamic && (
            <div className="flex flex-col gap-1.5">
              <Eyebrow>Every Service in these namespaces</Eyebrow>
              <div className="flex flex-wrap gap-1.5">
                {namespaces.map((item) => (
                  <MonoValue key={item}>{item}</MonoValue>
                ))}
              </div>
            </div>
          )}
          {services.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <Eyebrow>Services</Eyebrow>
              <div className="flex flex-wrap gap-1.5">
                {services.slice(0, 12).map((item) => (
                  <MonoValue key={item}>{item}</MonoValue>
                ))}
                {services.length > 12 && (
                  <span className="rounded-md px-1.5 py-0.5 text-xs font-medium text-slate-600">
                    +{services.length - 12} more
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {!scope && !catalog && !query.isLoading && (
        <Note>
          The contents of this Catalog are not visible to you right now.
        </Note>
      )}

      <Note icon={<Layers size={13} strokeWidth={2.25} />}>
        {isPartial
          ? props.reviewer
            ? "Approving this request grants access to the selected Services only."
            : "An approval grants access to the selected Services only."
          : props.reviewer
            ? "Approving this request grants access to every Service listed above."
            : "An approval grants access to every Service listed above."}
      </Note>

      <CatalogDetailsDrawer
        catalog={catalog}
        opened={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
};

const ResourceCard = (props: { request: AccessP.Request; reviewer?: boolean }) => {
  const resource = requestResourceLabel(props.request);

  return (
    <SectionCard
      title="Requested resource"
      description="What the access applies to"
      icon={
        resource.kind === "Catalog" ? (
          <Layers size={14} strokeWidth={2.25} />
        ) : (
          <ServerCog size={14} strokeWidth={2.25} />
        )
      }
    >
      {resource.kind === "Service" ? (
        <ServiceResource name={resource.name} />
      ) : resource.kind === "Catalog" ? (
        <CatalogResource request={props.request} reviewer={props.reviewer} />
      ) : (
        <Note tone="amber">This request has no resource set.</Note>
      )}
    </SectionCard>
  );
};

export default ResourceCard;
