import * as UserP from "@/apis/userv1/userv1";
import { Check, Lock, Radio, Server } from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

import { useNamespaces, useServices } from "@/components/Access/hooks";
import { serviceTypeIcon } from "@/components/Access/icons";
import {
  Badge,
  EmptyState,
  ErrorState,
  Eyebrow,
  SearchInput,
  SkeletonRows,
} from "@/ui";
import { namespaceFromName, serviceModeMeta, shortName } from "@/utils";

const ServiceCard = (props: {
  service: UserP.Service;
  selected: boolean;
  onClick: () => void;
}) => {
  const s = props.service;
  const name = s.metadata!.name;
  const display = s.metadata!.displayName || shortName(name);
  const ns = s.status?.namespace || namespaceFromName(name);
  const mode = serviceModeMeta(s.spec?.type);
  const Icon = serviceTypeIcon(s.spec?.type);

  return (
    <button
      type="button"
      onClick={props.onClick}
      aria-pressed={props.selected}
      className={twMerge(
        "flex w-full min-w-0 flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-[border-color,box-shadow,background-color] duration-150",
        props.selected
          ? "border-slate-900 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
          : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
      )}
    >
      <div
        className={twMerge(
          "flex h-9 w-9 shrink-0 items-center justify-center rounded-lg",
          props.selected ? "bg-slate-900 text-white" : "bg-slate-100 text-slate-500",
        )}
      >
        {props.selected ? (
          <Check size={16} strokeWidth={3} />
        ) : (
          <Icon size={16} strokeWidth={2.2} />
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className="truncate text-[0.84rem] font-bold text-slate-800">
          {display}
        </div>
        <div className="truncate font-mono text-[0.7rem] font-semibold text-slate-400">
          {name}
        </div>
        {(s.spec?.port || s.status?.primaryHostname) && (
          <div className="mt-1 flex min-w-0 items-center gap-2 truncate text-[0.66rem] font-semibold text-slate-400">
            {s.status?.primaryHostname && (
              <span className="truncate font-mono">{s.status.primaryHostname}</span>
            )}
            {s.spec?.port ? <span>:{s.spec.port}</span> : null}
          </div>
        )}
      </div>

      <div className="flex w-full shrink-0 flex-wrap items-center gap-1.5 sm:w-auto sm:justify-end">
        <Badge tone={mode.tone}>{mode.label}</Badge>
        {ns && <Badge tone="slate">{ns}</Badge>}
        {s.spec?.isTLS && (
          <Badge tone="emerald" icon={<Lock size={9} strokeWidth={3} />}>
            TLS
          </Badge>
        )}
        {s.spec?.isPublic && (
          <Badge tone="amber" icon={<Radio size={9} strokeWidth={3} />}>
            Public
          </Badge>
        )}
      </div>
    </button>
  );
};

const ServicePicker = (props: {
  value?: string;
  onChange: (service: UserP.Service) => void;
}) => {
  const [query, setQuery] = React.useState("");
  const [namespace, setNamespace] = React.useState("");

  const qry = useServices(namespace);
  const namespacesQry = useNamespaces();

  const services = qry.data ?? [];
  const q = query.toLowerCase().trim();
  const filtered = services.filter(
    (s) =>
      !q ||
      s.metadata?.name.toLowerCase().includes(q) ||
      s.metadata?.displayName?.toLowerCase().includes(q),
  );

  const namespaceNames =
    namespacesQry.data
      ?.map((item) => item.metadata?.name)
      .filter((name): name is string => !!name) ?? [];
  const namespaceServiceCounts = services.reduce<Record<string, number>>(
    (counts, service) => {
      const name =
        service.status?.namespace || namespaceFromName(service.metadata?.name);
      if (name) counts[name] = (counts[name] ?? 0) + 1;
      return counts;
    },
    {},
  );

  const chipClass = (active: boolean) =>
    twMerge(
      "shrink-0 rounded-md border px-2.5 py-1 text-[0.68rem] font-bold transition-colors duration-150",
      active
        ? "border-slate-900 bg-slate-900 text-white"
        : "border-slate-200 bg-white text-slate-500 hover:border-slate-300 hover:bg-slate-50",
    );

  return (
    <div className="flex w-full min-w-0 flex-col gap-3">
      <SearchInput
        value={query}
        onChange={setQuery}
        placeholder="Search services by name..."
        ariaLabel="Search services"
      />

      {namespaceNames.length > 0 && (
        <div className="flex items-center gap-1.5 overflow-x-auto pb-0.5">
          <button
            type="button"
            onClick={() => setNamespace("")}
            className={chipClass(!namespace)}
          >
            All namespaces
          </button>
          {namespaceNames.map((name) => (
            <button
              type="button"
              key={name}
              onClick={() => setNamespace(name)}
              className={chipClass(namespace === name)}
            >
              <span>{name}</span>
              {namespaceServiceCounts[name] !== undefined && (
                <span className="ml-1 opacity-70">
                  {namespaceServiceCounts[name]}
                </span>
              )}
            </button>
          ))}
        </div>
      )}

      {qry.isLoading || namespacesQry.isLoading ? (
        <SkeletonRows rows={4} />
      ) : qry.isError || namespacesQry.isError ? (
        <ErrorState
          title="Could not load services"
          onRetry={() => {
            void qry.refetch();
            void namespacesQry.refetch();
          }}
        />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Server size={20} strokeWidth={2} />}
          title="No services found"
          description={
            query
              ? "No services match your search."
              : "There are no Services available to you right now."
          }
        />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <Eyebrow>
              {filtered.length} service{filtered.length === 1 ? "" : "s"}
            </Eyebrow>
            {props.value && (
              <Eyebrow className="text-slate-500">1 selected</Eyebrow>
            )}
          </div>
          <div className="flex max-h-[460px] flex-col gap-2 overflow-y-auto pr-0.5">
            {filtered.map((s) => (
              <ServiceCard
                key={s.metadata!.uid || s.metadata!.name}
                service={s}
                selected={props.value === s.metadata!.name}
                onClick={() => props.onChange(s)}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default ServicePicker;
