const isDevVal = import.meta.env.MODE === "development";
import type { RpcError } from "@protobuf-ts/runtime-rpc";

import { QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

export function isDev(): boolean {
  return isDevVal;
}

const isWebgl2SupportedFn = (() => {
  let isSupported = window.WebGL2RenderingContext ? undefined : false;
  return () => {
    if (isSupported === undefined) {
      const canvas = document.createElement("canvas");
      const gl = canvas.getContext("webgl2", {
        depth: false,
        antialias: false,
      });
      isSupported = gl instanceof window.WebGL2RenderingContext;
    }
    return isSupported;
  };
})();

export const isWebgl2Supported = isWebgl2SupportedFn();

export const onError = (err: RpcError) => {
  toast.error(err.message);
};

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000,
    },
  },
});

export const toNumOrZero = (arg: string | null | undefined): number => {
  if (!arg) {
    return 0;
  }

  try {
    return parseInt(arg, 10);
  } catch {
    return 0;
  }
};

let __domain: string | undefined;

export const getDomain = (): string => {
  if (isDev()) {
    return window.location.host;
  }

  if (__domain) {
    return __domain;
  }

  __domain =
    ("; " + window.document.cookie)
      .split("; octelium_domain=")
      .pop()
      ?.split(";")
      .shift() ?? "";

  return __domain;
};

export function randomStringLowerCase(n: number): string {
  const characters = "abcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  const charactersLength = characters.length;

  for (let i = 0; i < n; i++) {
    const randomIndex = Math.floor(Math.random() * charactersLength);
    result += characters.charAt(randomIndex);
  }

  return result;
}

export function formatNumber(num: number): string {
  if (num >= 1_000_000_000) {
    return (num / 1_000_000_000).toFixed(2) + "B";
  } else if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(2) + "M";
  } else if (num >= 1_000) {
    return (num / 1_000).toFixed(2) + "K";
  } else {
    return num.toString();
  }
}

export function formatBytes(
  bytes: number,
  options: { useBinaryUnits?: boolean; decimals?: number } = {},
): string {
  const { useBinaryUnits = false, decimals = 2 } = options;

  if (decimals < 0) {
    throw new Error(`Invalid decimals ${decimals}`);
  }

  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.floor(Math.log(bytes) / Math.log(base));

  return `${(bytes / Math.pow(base, i)).toFixed(decimals)} ${units[i]}`;
}

export function slugify(input: string): string {
  return input
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/-{2,}/g, "-")
    .replace(/^-+|-+$/g, "");
}

import * as AccessP from "@/apis/accessv1/accessv1";
import * as MetaP from "@/apis/metav1/metav1";
import { match } from "ts-pattern";

export type Tone = "amber" | "emerald" | "red" | "slate" | "blue" | "violet";

export const toneClasses: Record<Tone, { text: string; badge: string }> = {
  amber: {
    text: "text-amber-600",
    badge: "bg-amber-50 text-amber-700 border-amber-200",
  },
  emerald: {
    text: "text-emerald-600",
    badge: "bg-emerald-50 text-emerald-700 border-emerald-200",
  },
  red: {
    text: "text-red-600",
    badge: "bg-red-50 text-red-700 border-red-200",
  },
  slate: {
    text: "text-slate-600",
    badge: "bg-slate-100 text-slate-600 border-slate-200",
  },
  blue: {
    text: "text-blue-600",
    badge: "bg-blue-50 text-blue-700 border-blue-200",
  },
  violet: {
    text: "text-violet-600",
    badge: "bg-violet-50 text-violet-700 border-violet-200",
  },
};

export const statusMeta = (
  status?: AccessP.Request_Status_State_Status,
): { label: string; tone: Tone } =>
  match(status)
    .with(AccessP.Request_Status_State_Status.PENDING, () => ({
      label: "Pending",
      tone: "amber" as Tone,
    }))
    .with(AccessP.Request_Status_State_Status.APPROVED, () => ({
      label: "Approved",
      tone: "emerald" as Tone,
    }))
    .with(AccessP.Request_Status_State_Status.REJECTED, () => ({
      label: "Rejected",
      tone: "red" as Tone,
    }))
    .with(AccessP.Request_Status_State_Status.REVOKED, () => ({
      label: "Revoked",
      tone: "red" as Tone,
    }))
    .with(AccessP.Request_Status_State_Status.EXPIRED, () => ({
      label: "Expired",
      tone: "slate" as Tone,
    }))
    .with(AccessP.Request_Status_State_Status.CANCELLED, () => ({
      label: "Cancelled",
      tone: "slate" as Tone,
    }))
    .otherwise(() => ({ label: "Unknown", tone: "slate" as Tone }));

export const urgencyMeta = (
  urgency?: AccessP.Request_Spec_Urgency,
): { label: string; tone: Tone } =>
  match(urgency)
    .with(AccessP.Request_Spec_Urgency.VERY_LOW, () => ({
      label: "Very Low",
      tone: "slate" as Tone,
    }))
    .with(AccessP.Request_Spec_Urgency.LOW, () => ({
      label: "Low",
      tone: "slate" as Tone,
    }))
    .with(AccessP.Request_Spec_Urgency.NORMAL, () => ({
      label: "Normal",
      tone: "blue" as Tone,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGH, () => ({
      label: "High",
      tone: "amber" as Tone,
    }))
    .with(AccessP.Request_Spec_Urgency.VERY_HIGH, () => ({
      label: "Very High",
      tone: "red" as Tone,
    }))
    .with(AccessP.Request_Spec_Urgency.HIGHEST, () => ({
      label: "Highest",
      tone: "red" as Tone,
    }))
    .otherwise(() => ({ label: "Normal", tone: "blue" as Tone }));

export const decisionMeta = (
  decision?: AccessP.Review_Spec_Decision,
): { label: string; tone: Tone } =>
  match(decision)
    .with(AccessP.Review_Spec_Decision.APPROVE, () => ({
      label: "Approved",
      tone: "emerald" as Tone,
    }))
    .with(AccessP.Review_Spec_Decision.REJECT, () => ({
      label: "Rejected",
      tone: "red" as Tone,
    }))
    .otherwise(() => ({ label: "Pending", tone: "amber" as Tone }));

export const requestResourceLabel = (
  item: AccessP.Request,
): { kind: "Service" | "Catalog" | "Unknown"; name: string } => {
  const r = item.spec?.resource?.type;
  if (!r) return { kind: "Unknown", name: "" };
  if (r.oneofKind === "serviceRef")
    return { kind: "Service", name: r.serviceRef.name };
  if (r.oneofKind === "catalog")
    return { kind: "Catalog", name: r.catalog.catalogRef?.name ?? "" };
  return { kind: "Unknown", name: "" };
};

export const shortName = (name?: string): string =>
  name?.split(".").at(0) ?? "";

const DURATION_UNITS = ["minutes", "hours", "days"] as const;
export type DurationUnit = (typeof DURATION_UNITS)[number];

export const DURATION_UNIT_OPTIONS = [
  { value: "minutes", label: "Minutes" },
  { value: "hours", label: "Hours" },
  { value: "days", label: "Days" },
];

export const durationToParts = (
  d?: MetaP.Duration,
): { unit: DurationUnit; amount: number } => {
  const kind = d?.type.oneofKind;
  if (kind && (DURATION_UNITS as readonly string[]).includes(kind)) {
    return {
      unit: kind as DurationUnit,
      amount: Number((d!.type as any)[kind]) || 0,
    };
  }
  return { unit: "hours", amount: 1 };
};

export const partsToDuration = (
  unit: DurationUnit,
  amount: number,
): MetaP.Duration =>
  MetaP.Duration.create({
    type: { oneofKind: unit, [unit]: amount } as any,
  });
