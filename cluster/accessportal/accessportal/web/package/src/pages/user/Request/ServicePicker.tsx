import * as UserP from "@/apis/userv1/userv1";
import { useQuery } from "@tanstack/react-query";
import {
  Boxes,
  Database,
  Globe,
  Monitor,
  Network,
  Search,
  Server,
  ShieldCheck,
  Terminal,
} from "lucide-react";
import * as React from "react";
import { twMerge } from "tailwind-merge";

import { Badge, EmptyState, ErrorState, Loading } from "../../../ui";
import { namespaceFromName, serviceModeMeta, shortName } from "../../../utils";
import { getUserMainClient } from "../../../utils/client";

const modeIcon = (mode?: UserP.Service_Spec_Type) => {
  switch (mode) {
    case UserP.Service_Spec_Type.HTTP:
    case UserP.Service_Spec_Type.WEB:
    case UserP.Service_Spec_Type.DNS:
      return Globe;
    case UserP.Service_Spec_Type.SSH:
      return Terminal;
    case UserP.Service_Spec_Type.POSTGRES:
    case UserP.Service_Spec_Type.MYSQL:
      return Database;
    case UserP.Service_Spec_Type.KUBERNETES:
      return Boxes;
    case UserP.Service_Spec_Type.TCP:
    case UserP.Service_Spec_Type.UDP:
    case UserP.Service_Spec_Type.GRPC:
    case UserP.Service_Spec_Type.MCP:
    case UserP.Service_Spec_Type.LLM:
      return Network;
    case UserP.Service_Spec_Type.SOCKS5:
      return ShieldCheck;
    case UserP.Service_Spec_Type.RDP_WEB:
      return Monitor;
    default:
      return Server;
  }
};

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
  const Icon = modeIcon(s.spec?.type);

  return (
    <button
      onClick={props.onClick}
      className={twMerge(
        "w-full min-w-0 flex items-center gap-3 text-left rounded-lg border px-3 py-2.5 transition-[border-color,box-shadow,background-color] duration-150",
        props.selected
          ? "border-slate-900 bg-slate-50 shadow-[0_2px_8px_rgba(15,23,42,0.10)]"
          : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50",
      )}
    >
      <div
        className={twMerge(
          "flex items-center justify-center w-9 h-9 rounded-lg shrink-0",
          props.selected
            ? "bg-slate-900 text-white"
            : "bg-slate-100 text-slate-500",
        )}
      >
        <Icon size={16} strokeWidth={2.2} />
      </div>

      <div className="flex-1 min-w-0">
        <div className="text-[0.84rem] font-bold text-slate-800 truncate">
          {display}
        </div>
        <div className="text-[0.7rem] font-semibold text-slate-400 font-mono truncate">
          {name}
        </div>
        <div className="flex items-center gap-2 mt-1 text-[0.66rem] font-semibold text-slate-400 truncate">
          {s.spec?.port ? <span>Port {s.spec.port}</span> : null}
          {s.status?.primaryHostname ? (
            <span className="font-mono truncate">{s.status.primaryHostname}</span>
          ) : null}
        </div>
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
        {ns && <Badge tone="slate">{ns}</Badge>}
        <Badge tone={mode.tone}>{mode.label}</Badge>
        {s.spec?.isTLS && <Badge tone="emerald">TLS</Badge>}
        {s.spec?.isPublic && <Badge tone="amber">Public</Badge>}
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

  const qry = useQuery({
    queryKey: ["userapi", "listService", namespace],
    queryFn: async () => {
      const items: UserP.Service[] = [];
      let page = 0;
      for (;;) {
        const { response } = await getUserMainClient().listService(
          UserP.ListServiceOptions.create({
            common: { page, itemsPerPage: 500 },
            namespace,
          }),
        );
        items.push(...response.items);
        if (!response.listResponseMeta?.hasMore) break;
        page += 1;
        if (page > 1000) break;
      }
      return { items };
    },
  });

  const namespacesQry = useQuery({
    queryKey: ["userapi", "listNamespace"],
    queryFn: async () => {
      const items: UserP.Namespace[] = [];
      let page = 0;
      for (;;) {
        const { response } = await getUserMainClient().listNamespace(
          UserP.ListNamespaceOptions.create({
            common: { page, itemsPerPage: 500 },
          }),
        );
        items.push(...response.items);
        if (!response.listResponseMeta?.hasMore) break;
        page += 1;
        if (page > 1000) break;
      }
      return items;
    },
  });

  const services = (qry.data?.items ?? []) as UserP.Service[];
  const q = query.toLowerCase().trim();
  const filtered = services.filter(
    (s) =>
      !q ||
      s.metadata?.name.toLowerCase().includes(q) ||
      s.metadata?.displayName?.toLowerCase().includes(q),
  );

  const namespaceNames = namespacesQry.data
    ?.map((item) => item.metadata?.name)
    .filter((name): name is string => !!name) ?? [];
  const namespaceServiceCounts = services.reduce<Record<string, number>>((counts, service) => {
    const name = service.status?.namespace || namespaceFromName(service.metadata?.name);
    if (name) counts[name] = (counts[name] ?? 0) + 1;
    return counts;
  }, {});

  return (
    <div className="w-full min-w-0 flex flex-col gap-3">
      <div className="relative">
        <Search
          size={13}
          strokeWidth={2.5}
          className="absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
        />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search services by name or namespace..."
          className="w-full pl-8 pr-3 h-8 text-[0.78rem] font-semibold text-slate-700 bg-white border border-slate-200 rounded-md shadow-[0_1px_3px_rgba(15,23,42,0.05)] outline-none focus:border-slate-400 focus:shadow-[0_0_0_2px_rgba(148,163,184,0.2)] transition-all duration-150 placeholder:text-slate-400 placeholder:font-semibold"
        />
      </div>

      {namespaceNames.length > 0 && (
        <div className="flex items-center gap-1.5 overflow-x-auto pb-0.5">
          <button
            type="button"
            onClick={() => setNamespace("")}
            className={twMerge(
              "shrink-0 px-2.5 py-1 rounded-md text-[0.68rem] font-bold border transition-colors",
              !namespace
                ? "bg-slate-900 text-white border-slate-900"
                : "bg-white text-slate-500 border-slate-200 hover:bg-slate-50",
            )}
          >
            All namespaces
          </button>
          {namespaceNames.map((name) => (
            <button
              type="button"
              key={name}
              onClick={() => setNamespace(name)}
              className={twMerge(
                "shrink-0 px-2.5 py-1 rounded-md text-[0.68rem] font-bold border transition-colors",
                namespace === name
                  ? "bg-slate-900 text-white border-slate-900"
                  : "bg-white text-slate-500 border-slate-200 hover:bg-slate-50",
              )}
            >
              <span>{name}</span>
              {namespaceServiceCounts[name] !== undefined && (
                <span className="ml-1 opacity-70">{namespaceServiceCounts[name]}</span>
              )}
            </button>
          ))}
        </div>
      )}

      {qry.isLoading || namespacesQry.isLoading ? (
        <Loading label="Loading services..." />
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
        <div className="flex flex-col gap-2 max-h-[460px] overflow-y-auto pr-0.5">
          {filtered.map((s) => (
            <ServiceCard
              key={s.metadata!.uid || s.metadata!.name}
              service={s}
              selected={props.value === s.metadata!.name}
              onClick={() => props.onChange(s)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

export default ServicePicker;
