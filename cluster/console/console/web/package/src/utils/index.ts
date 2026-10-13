const isDevVal = import.meta.env.MODE === "development";
import type { RpcError } from "@protobuf-ts/runtime-rpc";

import { MutationCache, QueryCache, QueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { onQueryError, queryRetryDelay, retryQuery } from "./queryPolicy";

export function isDev(): boolean {
  return isDevVal;
}

export const onError = (err: RpcError) => {
  toast.error(err.message);
};

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: onQueryError }),
  mutationCache: new MutationCache({ onError: onQueryError }),
  defaultOptions: {
    queries: {
      staleTime: 30000,
      retry: retryQuery,
      retryDelay: queryRetryDelay,
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

  const domain = window.__OCTELIUM_GLOBALS__?.cluster?.domain;
  if (typeof domain !== "string" || domain.length === 0) {
    throw new Error("The cluster domain is unavailable");
  }

  __domain = domain;

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

  if (!Number.isFinite(bytes)) return "—";
  if (bytes === 0) return "0 Bytes";

  const base = useBinaryUnits ? 1024 : 1000;
  const units = useBinaryUnits
    ? ["Bytes", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"]
    : ["Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];

  const i = Math.max(
    0,
    Math.min(
      units.length - 1,
      Math.floor(Math.log(Math.abs(bytes)) / Math.log(base)),
    ),
  );

  return `${(bytes / Math.pow(base, i)).toFixed(i === 0 ? 0 : decimals)} ${units[i]}`;
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
