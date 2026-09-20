import {
  GetClusterHealthResponse_Status,
  GetClusterHealthResponse_Subsystem,
  GetClusterHealthResponse_Subsystem_Type,
} from "@/apis/visibilityv1/visibilityv1";
import {
  Activity,
  AlertTriangle,
  BookKey,
  CheckCircle2,
  CircleHelp,
  ClipboardCheck,
  Folder,
  Globe,
  LaptopMinimal,
  LucideIcon,
  ScrollText,
  ShieldCheck,
  UserCheck,
} from "lucide-react";
import { Link } from "react-router-dom";
import { twMerge } from "tailwind-merge";
import { PILL } from "./components";

export const STATUS_STYLE: Record<number, string> = {
  [GetClusterHealthResponse_Status.OK]:
    "border-emerald-200 bg-emerald-50/50 text-emerald-700",
  [GetClusterHealthResponse_Status.DEGRADED]:
    "border-amber-200 bg-amber-50/60 text-amber-700",
  [GetClusterHealthResponse_Status.CRITICAL]:
    "border-red-200 bg-red-50/60 text-red-700",
  [GetClusterHealthResponse_Status.UNKNOWN]:
    "border-slate-200 bg-slate-50/60 text-slate-500",
};

export const STATUS_LABEL: Record<number, string> = {
  [GetClusterHealthResponse_Status.OK]: "Healthy",
  [GetClusterHealthResponse_Status.DEGRADED]: "Degraded",
  [GetClusterHealthResponse_Status.CRITICAL]: "Critical",
  [GetClusterHealthResponse_Status.UNKNOWN]: "Unknown",
};

export const SUBSYSTEM: Record<
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

export const StatusBadge = (props: { status: number; className?: string }) => {
  const Icon =
    props.status === GetClusterHealthResponse_Status.OK
      ? CheckCircle2
      : props.status === GetClusterHealthResponse_Status.UNKNOWN
        ? CircleHelp
        : AlertTriangle;

  return (
    <span
      className={twMerge(
        PILL,
        STATUS_STYLE[props.status] ??
          STATUS_STYLE[GetClusterHealthResponse_Status.UNKNOWN],
        props.className,
      )}
    >
      <Icon size={9} strokeWidth={2.5} />
      {STATUS_LABEL[props.status] ?? "Unknown"}
    </span>
  );
};

export const SubsystemCard = (props: {
  subsystem: GetClusterHealthResponse_Subsystem;
}) => {
  const { subsystem } = props;
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

  if (!meta.to) return <div className={className}>{content}</div>;

  return (
    <Link
      to={meta.to}
      className={twMerge(
        className,
        "outline-none transition-[border-color,box-shadow] duration-150 hover:shadow-raised focus-visible:ring-2 focus-visible:ring-slate-400",
      )}
    >
      {content}
    </Link>
  );
};

export const SubsystemGrid = (props: {
  subsystems: GetClusterHealthResponse_Subsystem[];
  types?: GetClusterHealthResponse_Subsystem_Type[];
}) => {
  const shown = props.types
    ? props.subsystems.filter((item) => props.types!.includes(item.type))
    : props.subsystems;

  if (shown.length === 0) return null;

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {shown.map((subsystem) => (
        <SubsystemCard key={subsystem.type} subsystem={subsystem} />
      ))}
    </div>
  );
};
