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
  options: { useBinaryUnits?: boolean; decimals?: number } = {}
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
