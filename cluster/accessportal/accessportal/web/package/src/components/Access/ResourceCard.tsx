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
  isNotFoundError,
} from "@/ui";
import {
  namespaceFromName,
  requestResourceLabel,
  serviceModeLabel,
  shortName,
} from "@/utils";
import { describeRpcError } from "@/utils/rpcError";

import { serviceTypeIcon } from "./icons";
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
  const Icon = serviceTypeIcon(service?.spec?.type);
  const namespace = service?.status?.namespace || namespaceFromName(props.name);

  return (
    <div className="flex flex-col gap-4">
      <ResourceHeading
        icon={<Icon size={18} strokeWidth={2} />}
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

const CatalogResource = (props: { name: string; reviewer?: boolean }) => {
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const query = useCatalogs();
  const catalog = query.data?.find((item) => item.metadata?.name === props.name);
  const collection = catalog?.spec?.resourceCollection?.service;
  const services = collection?.services ?? [];
  const namespaces = collection?.namespaces ?? [];

  return (
    <div className="flex flex-col gap-4">
      <ResourceHeading
        icon={<Layers size={18} strokeWidth={2} />}
        title={catalog?.metadata?.displayName || props.name}
        name={props.name}
        badges={
          <>
            <Badge>Catalog</Badge>
            {services.length > 0 && (
              <Badge>
                {services.length} listed service{services.length === 1 ? "" : "s"}
              </Badge>
            )}
            {namespaces.length > 0 && (
              <Badge>
                {namespaces.length} namespace{namespaces.length === 1 ? "" : "s"}
              </Badge>
            )}
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
          {namespaces.length > 0 && (
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
              <Eyebrow>Listed services</Eyebrow>
              <div className="flex flex-wrap gap-1.5">
                {services.slice(0, 12).map((item) => (
                  <MonoValue key={item}>{item}</MonoValue>
                ))}
                {services.length > 12 && (
                  <button
                    type="button"
                    onClick={() => setDrawerOpen(true)}
                    className="rounded-md px-1.5 py-0.5 text-xs font-medium text-slate-600 underline underline-offset-2 hover:text-slate-900"
                  >
                    +{services.length - 12} more
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {!catalog && !query.isLoading && (
        <Note>The contents of this Catalog are not visible to you right now.</Note>
      )}

      <Note icon={<Layers size={13} strokeWidth={2.25} />}>
        {props.reviewer
          ? "Approving this request grants access to every Service collected by this Catalog."
          : "An approval grants access to every Service collected by this Catalog."}
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
        <CatalogResource name={resource.name} reviewer={props.reviewer} />
      ) : (
        <Note tone="amber">This request has no resource set.</Note>
      )}
    </SectionCard>
  );
};

export default ResourceCard;
