import * as AccessP from "@/apis/accessv1/accessv1";
import { Button } from "@mantine/core";
import { Globe2, Layers, Lock, Radio, ServerCog } from "lucide-react";
import * as React from "react";

import CatalogDetailsDrawer from "@/components/Catalog/CatalogDetailsDrawer";
import {
  Badge,
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
  serviceModeMeta,
  shortName,
} from "@/utils";

import { serviceTypeIcon } from "./icons";
import { useCatalogs, useService } from "./hooks";

const ServiceResource = (props: { name: string }) => {
  const namespace = namespaceFromName(props.name);
  const query = useService(props.name);
  const service = query.data;
  const mode = serviceModeMeta(service?.spec?.type);
  const Icon = serviceTypeIcon(service?.spec?.type);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start gap-3">
        <IconTile size="lg" tone={mode.tone}>
          <Icon size={19} strokeWidth={2.1} />
        </IconTile>
        <div className="min-w-0 flex-1">
          <p className="truncate text-[0.95rem] font-bold text-slate-900">
            {service?.metadata?.displayName || shortName(props.name)}
          </p>
          <p className="mt-0.5 truncate font-mono text-[0.72rem] font-semibold text-slate-400">
            {props.name}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            <Badge tone={mode.tone}>{mode.label}</Badge>
            {(service?.status?.namespace || namespace) && (
              <Badge tone="slate" icon={<Globe2 size={10} strokeWidth={2.6} />}>
                {service?.status?.namespace || namespace}
              </Badge>
            )}
            {service?.spec?.isTLS && (
              <Badge tone="emerald" icon={<Lock size={10} strokeWidth={2.8} />}>
                TLS
              </Badge>
            )}
            {service?.spec?.isPublic && (
              <Badge tone="amber" icon={<Radio size={10} strokeWidth={2.8} />}>
                Public
              </Badge>
            )}
          </div>
        </div>
      </div>

      {service ? (
        <InfoGrid className="border-t border-slate-100 pt-4">
          <KeyValue label="Primary hostname" mono>
            <span className="truncate">
              {service.status?.primaryHostname || "—"}
            </span>
          </KeyValue>
          <KeyValue label="Port" mono>
            {service.spec?.port || "—"}
          </KeyValue>
          {service.status?.addresses?.length ? (
            <KeyValue label="Addresses" mono full>
              <span className="break-all">
                {service.status.addresses.join(", ")}
              </span>
            </KeyValue>
          ) : null}
          {service.metadata?.description && (
            <KeyValue label="Description" full>
              <span className="font-medium text-slate-600">
                {service.metadata.description}
              </span>
            </KeyValue>
          )}
        </InfoGrid>
      ) : query.isLoading ? (
        <div className="flex flex-col gap-2 border-t border-slate-100 pt-4">
          <div className="h-3 w-1/3 animate-pulse rounded bg-slate-100" />
          <div className="h-3 w-1/2 animate-pulse rounded bg-slate-50" />
        </div>
      ) : (
        <Note tone="slate" icon={<ServerCog size={13} strokeWidth={2.4} />}>
          The technical details of this Service are not visible to you right
          now.
        </Note>
      )}
    </div>
  );
};

const CatalogResource = (props: { name: string }) => {
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const query = useCatalogs();
  const catalog = query.data?.find((item) => item.metadata?.name === props.name);
  const collection = catalog?.spec?.resourceCollection?.service;
  const services = collection?.services ?? [];
  const namespaces = collection?.namespaces ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-start gap-3">
        <IconTile size="lg" tone="violet">
          <Layers size={19} strokeWidth={2.1} />
        </IconTile>
        <div className="min-w-0 flex-1">
          <p className="truncate text-[0.95rem] font-bold text-slate-900">
            {catalog?.metadata?.displayName || props.name}
          </p>
          <p className="mt-0.5 truncate font-mono text-[0.72rem] font-semibold text-slate-400">
            {props.name}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-1.5">
            <Badge tone="violet">Catalog</Badge>
            {services.length > 0 && (
              <Badge tone="slate">{services.length} services</Badge>
            )}
            {namespaces.length > 0 && (
              <Badge tone="slate">{namespaces.length} namespaces</Badge>
            )}
          </div>
        </div>
        {catalog && (
          <Button
            variant="default"
            size="compact-sm"
            onClick={() => setDrawerOpen(true)}
          >
            Contents
          </Button>
        )}
      </div>

      {catalog?.metadata?.description && (
        <p className="text-[0.78rem] font-medium leading-relaxed text-slate-600">
          {catalog.metadata.description}
        </p>
      )}

      {(services.length > 0 || namespaces.length > 0) && (
        <div className="flex flex-col gap-3 border-t border-slate-100 pt-4">
          {namespaces.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-[0.62rem] font-bold uppercase tracking-[0.09em] text-slate-400">
                Namespaces
              </span>
              <div className="flex flex-wrap gap-1.5">
                {namespaces.map((item) => (
                  <MonoValue key={item}>{item}</MonoValue>
                ))}
              </div>
            </div>
          )}
          {services.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <span className="text-[0.62rem] font-bold uppercase tracking-[0.09em] text-slate-400">
                Services
              </span>
              <div className="flex flex-wrap gap-1.5">
                {services.slice(0, 12).map((item) => (
                  <MonoValue key={item}>{item}</MonoValue>
                ))}
                {services.length > 12 && (
                  <button
                    type="button"
                    onClick={() => setDrawerOpen(true)}
                    className="rounded bg-slate-900 px-1.5 py-0.5 font-mono text-[0.72rem] font-semibold text-white"
                  >
                    +{services.length - 12} more
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      <Note tone="violet" icon={<Layers size={13} strokeWidth={2.4} />}>
        Approving this request grants access to every Service collected by this
        Catalog.
      </Note>

      <CatalogDetailsDrawer
        catalog={catalog}
        opened={drawerOpen}
        onClose={() => setDrawerOpen(false)}
      />
    </div>
  );
};

const ResourceCard = (props: { request: AccessP.Request }) => {
  const resource = requestResourceLabel(props.request);

  return (
    <SectionCard
      title="Requested resource"
      description="What the access applies to"
      icon={
        resource.kind === "Catalog" ? (
          <Layers size={14} strokeWidth={2.4} />
        ) : (
          <ServerCog size={14} strokeWidth={2.4} />
        )
      }
      tone={resource.kind === "Catalog" ? "violet" : "blue"}
    >
      {resource.kind === "Service" ? (
        <ServiceResource name={resource.name} />
      ) : resource.kind === "Catalog" ? (
        <CatalogResource name={resource.name} />
      ) : (
        <Note tone="amber">This request has no resource set.</Note>
      )}
    </SectionCard>
  );
};

export default ResourceCard;
