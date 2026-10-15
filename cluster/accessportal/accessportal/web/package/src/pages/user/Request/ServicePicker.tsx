import * as UserP from "@/apis/userv1/userv1";
import { Button, Select } from "@mantine/core";
import { Check, Layers, Server } from "lucide-react";
import * as React from "react";
import { twMerge } from "@/lib/utils";

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
import { namespaceFromName, serviceModeLabel, shortName } from "@/utils";

const ServiceCard = (props: {
  service: UserP.Service;
  selected: boolean;
  onClick: () => void;
}) => {
  const s = props.service;
  const name = s.metadata!.name;
  const display = s.metadata!.displayName || shortName(name);
  const ns = s.status?.namespace || namespaceFromName(name);
  const Icon = serviceTypeIcon(s.spec?.type);

  return (
    <button
      type="button"
      onClick={props.onClick}
      aria-pressed={props.selected}
      className={twMerge(
        "flex w-full min-w-0 flex-wrap items-center gap-3 rounded-lg border px-3 py-3 text-left outline-none transition-[border-color,box-shadow,background-color] duration-150 focus-visible:ring-2 focus-visible:ring-slate-400",
        props.selected
          ? "border-slate-400 bg-slate-50 ring-1 ring-slate-300"
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
          <Check size={16} strokeWidth={2.75} />
        ) : (
          <Icon size={16} strokeWidth={2} />
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className="truncate text-body font-semibold text-slate-800">
          {display}
        </div>
        <div className="truncate font-mono text-xs font-normal text-slate-500">
          {name}
        </div>
      </div>

      <div className="flex w-full shrink-0 flex-wrap items-center gap-1.5 sm:w-auto sm:justify-end">
        <Badge>{serviceModeLabel(s.spec?.type)}</Badge>
        {ns && <Badge>{ns}</Badge>}
        {s.spec?.isTLS && <Badge>TLS</Badge>}
        {s.spec?.isPublic && <Badge>Public</Badge>}
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

  const filteredOut = !!q || !!namespace;

  return (
    <div className="flex w-full min-w-0 flex-col gap-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <SearchInput
          value={query}
          onChange={setQuery}
          placeholder="Search services by name"
          ariaLabel="Search services"
        />
        {namespaceNames.length > 0 && (
          <Select
            aria-label="Namespace"
            value={namespace}
            onChange={(value) => setNamespace(value ?? "")}
            data={[
              { value: "", label: "All namespaces" },
              ...namespaceNames.map((name) => ({ value: name, label: name })),
            ]}
            leftSection={<Layers size={14} aria-hidden />}
            searchable
            allowDeselect={false}
            nothingFoundMessage="No namespaces found"
            className="w-full min-w-0 sm:w-48 sm:shrink-0"
          />
        )}
      </div>

      {qry.isLoading ? (
        <SkeletonRows rows={4} />
      ) : qry.isError && !qry.data ? (
        <ErrorState
          title="Could not load services"
          error={qry.error}
          onRetry={() => {
            void qry.refetch();
          }}
        />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Server size={18} strokeWidth={2} />}
          title="No services found"
          description={
            filteredOut
              ? namespace && !q
                ? "There are no Services visible to you in this namespace."
                : "No services match your search."
              : "There are no Services available to you right now."
          }
          action={
            filteredOut ? (
              <Button
                variant="default"
                onClick={() => {
                  setQuery("");
                  setNamespace("");
                }}
              >
                Clear filters
              </Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <Eyebrow>
            {filtered.length} service{filtered.length === 1 ? "" : "s"}
          </Eyebrow>
          <div className="flex max-h-[460px] flex-col gap-2 overflow-y-auto overscroll-contain pr-0.5">
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
