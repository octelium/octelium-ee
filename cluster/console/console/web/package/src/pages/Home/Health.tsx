import {
  GetClusterHealthResponse_Status,
  GetClusterHealthResponse_Subsystem_Type,
} from "@/apis/visibilityv1/visibilityv1";
import { n, periodLabel } from "@/utils/visibility";
import { QUERY_PRIORITY } from "@/utils/visibility/queue";
import {
  Activity,
  AlertTriangle,
  BookKey,
  CheckCircle2,
  CircleHelp,
  ClipboardCheck,
  Folder,
  Globe,
  HeartPulse,
  LaptopMinimal,
  LucideIcon,
  ScrollText,
  ShieldCheck,
  UserCheck,
} from "lucide-react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { Panel } from "./components";
import { useClusterHealth } from "./queries";
import { compact } from "./utils";

const STATUS_STYLE: Record<number, string> = {
  [GetClusterHealthResponse_Status.OK]:
    "border-emerald-200 bg-emerald-50/50 text-emerald-700",
  [GetClusterHealthResponse_Status.DEGRADED]:
    "border-amber-200 bg-amber-50/60 text-amber-700",
  [GetClusterHealthResponse_Status.CRITICAL]:
    "border-red-200 bg-red-50/60 text-red-700",
  [GetClusterHealthResponse_Status.UNKNOWN]:
    "border-slate-200 bg-slate-50/60 text-slate-500",
};

const STATUS_LABEL: Record<number, string> = {
  [GetClusterHealthResponse_Status.OK]: "Healthy",
  [GetClusterHealthResponse_Status.DEGRADED]: "Degraded",
  [GetClusterHealthResponse_Status.CRITICAL]: "Critical",
  [GetClusterHealthResponse_Status.UNKNOWN]: "Unknown",
};

const SUBSYSTEM: Record<
  number,
  { label: string; icon: LucideIcon; to?: string }
> = {
  [GetClusterHealthResponse_Subsystem_Type.COMPONENTS]: {
    label: "Components",
    icon: ScrollText,
    to: "/visibility/componentlogs?level=ERROR",
  },
  [GetClusterHealthResponse_Subsystem_Type.AUTHORIZATION]: {
    label: "Authorization",
    icon: Activity,
    to: "/visibility/accesslogs?status=DENIED",
  },
  [GetClusterHealthResponse_Subsystem_Type.CERTIFICATES]: {
    label: "Certificates",
    icon: ShieldCheck,
    to: "/enterprise/certificates",
  },
  [GetClusterHealthResponse_Subsystem_Type.DIRECTORY_PROVIDERS]: {
    label: "Directory sync",
    icon: Folder,
    to: "/enterprise/directoryproviders",
  },
  [GetClusterHealthResponse_Subsystem_Type.SECRET_STORES]: {
    label: "Secret stores",
    icon: BookKey,
    to: "/enterprise/secretstores",
  },
  [GetClusterHealthResponse_Subsystem_Type.DEVICE_MANAGERS]: {
    label: "Device managers",
    icon: LaptopMinimal,
  },
  [GetClusterHealthResponse_Subsystem_Type.ENROLLMENT]: {
    label: "Enrollment",
    icon: UserCheck,
    to: "/core/devices?state=PENDING",
  },
  [GetClusterHealthResponse_Subsystem_Type.ACCESS_REQUESTS]: {
    label: "Access requests",
    icon: ClipboardCheck,
    to: "/access/requests?state=PENDING",
  },
  [GetClusterHealthResponse_Subsystem_Type.DATA_PLANE]: {
    label: "Data plane",
    icon: Globe,
    to: "/core/gateways",
  },
};

const StatusBadge = (props: { status: number; className?: string }) => {
  const Icon =
    props.status === GetClusterHealthResponse_Status.OK
      ? CheckCircle2
      : props.status === GetClusterHealthResponse_Status.UNKNOWN
        ? CircleHelp
        : AlertTriangle;

  return (
    <span
      className={twMerge(
        "inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-micro font-semibold",
        STATUS_STYLE[props.status] ??
          STATUS_STYLE[GetClusterHealthResponse_Status.UNKNOWN],
        props.className,
      )}
    >
      <Icon size={10} strokeWidth={2.5} />
      {STATUS_LABEL[props.status] ?? "Unknown"}
    </span>
  );
};

const Health = (props: { periodMinutes: number }) => {
  const { periodMinutes } = props;
  const rangeLabel = periodLabel(periodMinutes);

  const health = useClusterHealth(periodMinutes, QUERY_PRIORITY.high);
  const data = health.data;

  const subsystems = data?.subsystems ?? [];
  const components = (data?.components ?? []).slice(0, 6);
  const regions = data?.regions ?? [];

  return (
    <Panel
      icon={HeartPulse}
      title="Cluster health"
      description={`Subsystem health derived from the Cluster's own signals over the last ${rangeLabel}`}
      actions={data ? <StatusBadge status={data.status} /> : undefined}
    >
      {health.isLoading && !data ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {[0, 1, 2, 3, 4, 5].map((index) => (
            <div
              key={index}
              className="h-[86px] animate-pulse rounded-xl border border-slate-200 bg-slate-50"
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-5">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {subsystems.map((subsystem) => {
              const meta = SUBSYSTEM[subsystem.type];
              if (!meta) return null;
              const Icon = meta.icon;

              const content = (
                <>
                  <div className="flex items-center justify-between gap-2">
                    <span className="inline-flex min-w-0 items-center gap-2">
                      <Icon
                        size={13}
                        strokeWidth={2.3}
                        className="shrink-0 text-slate-500"
                      />
                      <span className="truncate text-body font-semibold text-slate-800">
                        {meta.label}
                      </span>
                    </span>
                    <StatusBadge status={subsystem.status} />
                  </div>
                  <p className="text-micro font-normal leading-4 text-slate-500">
                    {subsystem.message}
                  </p>
                </>
              );

              const className = twMerge(
                "flex min-w-0 flex-col gap-2 rounded-xl border px-3.5 py-3",
                subsystem.status === GetClusterHealthResponse_Status.OK
                  ? "border-slate-200 bg-white"
                  : STATUS_STYLE[subsystem.status],
              );

              if (!meta.to) {
                return (
                  <div key={subsystem.type} className={className}>
                    {content}
                  </div>
                );
              }

              return (
                <Link
                  key={subsystem.type}
                  to={meta.to}
                  className={twMerge(
                    className,
                    "outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
                  )}
                >
                  {content}
                </Link>
              );
            })}
          </div>

          {(components.length > 0 || regions.length > 0) && (
            <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
              {components.length > 0 && (
                <div className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    Noisiest components
                  </span>

                  {components.map((item) => (
                    <Link
                      key={`${item.component?.namespace}/${item.component?.type}`}
                      to={`/visibility/componentlogs?level=ERROR&component.type=${item.component?.type ?? ""}`}
                      className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2 outline-none transition-colors duration-150 hover:border-slate-300 focus-visible:ring-2 focus-visible:ring-slate-400"
                    >
                      <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                        {item.component?.type}
                        <span className="ml-1.5 text-micro font-normal text-slate-500">
                          {item.component?.namespace}
                        </span>
                      </span>
                      {n(item.totalWarn) > 0 && (
                        <span className="shrink-0 text-micro font-semibold tabular-nums text-amber-700">
                          {compact(n(item.totalWarn))} warn
                        </span>
                      )}
                      <span className="shrink-0 text-micro font-semibold tabular-nums text-red-700">
                        {compact(
                          n(item.totalError) +
                            n(item.totalPanic) +
                            n(item.totalFatal),
                        )}{" "}
                        err
                      </span>
                    </Link>
                  ))}
                </div>
              )}

              {regions.length > 0 && (
                <div className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-slate-50/60 px-3.5 py-3">
                  <span className="text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
                    Regions
                  </span>

                  {regions.map((region) => (
                    <div
                      key={region.regionRef?.uid}
                      className="flex items-center gap-3 rounded-lg border border-slate-200 bg-white px-3 py-2"
                    >
                      <span className="min-w-0 flex-1 truncate text-body font-semibold text-slate-700">
                        {region.regionRef?.name}
                        {region.version && (
                          <span className="ml-1.5 text-micro font-normal text-slate-500">
                            {region.version}
                          </span>
                        )}
                      </span>
                      <span className="shrink-0 text-micro font-normal tabular-nums text-slate-500">
                        {n(region.totalGateway)} gateways
                      </span>
                      <StatusBadge status={region.status} />
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </Panel>
  );
};

export default Health;
