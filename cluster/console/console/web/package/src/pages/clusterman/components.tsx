import {
  ClusterConfig_Status_LicenseInfo_State,
  ClusterConfig_Status_UpgradeRequest,
  ClusterConfig_Status_UpgradeRequest_State,
  License_Type,
} from "@/apis/enterprisev1/enterprisev1";
import {
  ArrowRight,
  BadgeCheck,
  CheckCircle2,
  CircleHelp,
  Clock,
  Loader2,
  ShieldOff,
  TriangleAlert,
  XCircle,
} from "lucide-react";
import { match } from "ts-pattern";
import { twMerge } from "tailwind-merge";

const BADGE =
  "inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-micro font-semibold";

export const UpgradeStateBadge = (props: {
  state: ClusterConfig_Status_UpgradeRequest_State;
}) =>
  match(props.state)
    .with(ClusterConfig_Status_UpgradeRequest_State.SUCCESS, () => (
      <span
        className={twMerge(
          BADGE,
          "border-emerald-200 bg-emerald-50 text-emerald-700",
        )}
      >
        <CheckCircle2 size={10} strokeWidth={2.5} />
        Succeeded
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.FAILED, () => (
      <span className={twMerge(BADGE, "border-red-200 bg-red-50 text-red-700")}>
        <XCircle size={10} strokeWidth={2.5} />
        Failed
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.UPGRADING, () => (
      <span
        className={twMerge(BADGE, "border-blue-200 bg-blue-50 text-blue-700")}
      >
        <Loader2 size={10} strokeWidth={2.5} className="animate-spin" />
        Upgrading
      </span>
    ))
    .with(ClusterConfig_Status_UpgradeRequest_State.UPGRADE_REQUESTED, () => (
      <span
        className={twMerge(
          BADGE,
          "border-amber-200 bg-amber-50 text-amber-700",
        )}
      >
        <Clock size={10} strokeWidth={2.5} />
        Requested
      </span>
    ))
    .otherwise(() => null);

export const LICENSE_STATE_LABEL: Record<number, string> = {
  [ClusterConfig_Status_LicenseInfo_State.NONE]: "No license",
  [ClusterConfig_Status_LicenseInfo_State.ACTIVE]: "Active",
  [ClusterConfig_Status_LicenseInfo_State.NOT_YET_VALID]: "Not yet valid",
  [ClusterConfig_Status_LicenseInfo_State.EXPIRED]: "Expired",
  [ClusterConfig_Status_LicenseInfo_State.INVALID]: "Invalid",
  [ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN]: "Unknown",
};

const LICENSE_STATE_STYLE: Record<number, string> = {
  [ClusterConfig_Status_LicenseInfo_State.NONE]:
    "border-slate-200 bg-slate-50 text-slate-500",
  [ClusterConfig_Status_LicenseInfo_State.ACTIVE]:
    "border-emerald-200 bg-emerald-50 text-emerald-700",
  [ClusterConfig_Status_LicenseInfo_State.NOT_YET_VALID]:
    "border-amber-200 bg-amber-50 text-amber-700",
  [ClusterConfig_Status_LicenseInfo_State.EXPIRED]:
    "border-red-200 bg-red-50 text-red-700",
  [ClusterConfig_Status_LicenseInfo_State.INVALID]:
    "border-red-200 bg-red-50 text-red-700",
  [ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN]:
    "border-slate-200 bg-slate-50 text-slate-500",
};

export const LicenseStateBadge = (props: { state: number }) => {
  const Icon = match(props.state)
    .with(ClusterConfig_Status_LicenseInfo_State.ACTIVE, () => BadgeCheck)
    .with(ClusterConfig_Status_LicenseInfo_State.NONE, () => ShieldOff)
    .with(
      ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN,
      () => CircleHelp,
    )
    .otherwise(() => TriangleAlert);

  return (
    <span
      className={twMerge(
        BADGE,
        LICENSE_STATE_STYLE[props.state] ??
          LICENSE_STATE_STYLE[
            ClusterConfig_Status_LicenseInfo_State.STATE_UNKNOWN
          ],
      )}
    >
      <Icon size={10} strokeWidth={2.5} />
      {LICENSE_STATE_LABEL[props.state] ?? "Unknown"}
    </span>
  );
};

export const LICENSE_TYPE_LABEL: Record<number, string> = {
  [License_Type.TRIAL]: "Trial",
  [License_Type.SUBSCRIPTION]: "Subscription",
  [License_Type.PERPETUAL]: "Perpetual",
  [License_Type.INTERNAL]: "Internal",
  [License_Type.TYPE_UNKNOWN]: "Unknown",
};

export const VersionDelta = (props: {
  from?: string;
  to?: string;
  highlight?: boolean;
}) => (
  <span className="inline-flex min-w-0 items-center gap-1.5 tabular-nums">
    <span className="truncate text-body font-semibold text-slate-700">
      {props.from || "Unknown"}
    </span>
    <ArrowRight
      size={12}
      strokeWidth={2.4}
      className="shrink-0 text-slate-400"
    />
    <span
      className={twMerge(
        "truncate text-body font-semibold",
        props.highlight ? "text-blue-700" : "text-slate-700",
      )}
    >
      {props.to || "latest"}
    </span>
  </span>
);

const VersionChip = (props: { label: string; version: string }) => (
  <span className="inline-flex items-center gap-1 rounded border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-micro font-normal text-slate-600">
    {props.label}
    <span
      className={
        props.version ? "font-semibold text-slate-800" : "text-slate-500"
      }
    >
      {props.version || "latest"}
    </span>
  </span>
);

export const VersionChips = (props: {
  request?: ClusterConfig_Status_UpgradeRequest["request"];
}) => {
  const { request } = props;
  if (!request) return null;

  return (
    <div className="flex flex-wrap items-center gap-1">
      {request.core && (
        <VersionChip label="Core" version={request.core.version} />
      )}
      {request.packageEnterprise && (
        <VersionChip
          label="Enterprise"
          version={request.packageEnterprise.version}
        />
      )}
      {request.packageCordium && (
        <VersionChip label="Cordium" version={request.packageCordium.version} />
      )}
    </div>
  );
};

export const Field = (props: {
  label: string;
  children: React.ReactNode;
  className?: string;
}) => (
  <div
    className={twMerge(
      "flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200 bg-slate-50/60 px-3 py-2.5",
      props.className,
    )}
  >
    <span className="truncate text-micro font-semibold uppercase tracking-[0.07em] text-slate-500">
      {props.label}
    </span>
    <span className="truncate text-body font-semibold text-slate-800">
      {props.children}
    </span>
  </div>
);
