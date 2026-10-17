const CHUNK_LOAD_ERROR_PATTERNS = [
  /error loading dynamically imported module/i,
  /failed to fetch dynamically imported module/i,
  /importing a module script failed/i,
  /failed to load module script/i,
  /unable to preload css/i,
];

const RELOAD_KEY = "octelium-accessportal:chunk-reload-at";
const RELOAD_INTERVAL = 30_000;

export const isChunkLoadError = (error: unknown): boolean => {
  const value = error as { name?: unknown; message?: unknown } | null;
  const message =
    typeof error === "string"
      ? error
      : typeof value?.message === "string"
        ? value.message
        : "";
  return (
    value?.name === "ChunkLoadError" ||
    CHUNK_LOAD_ERROR_PATTERNS.some((pattern) => pattern.test(message))
  );
};

export const reloadForNewVersion = (): boolean => {
  if (typeof window === "undefined") return false;

  try {
    const last = Number(window.sessionStorage.getItem(RELOAD_KEY) ?? 0);
    if (Date.now() - last < RELOAD_INTERVAL) return false;
    window.sessionStorage.setItem(RELOAD_KEY, String(Date.now()));
  } catch {
    return false;
  }

  window.location.reload();
  return true;
};

let installed = false;

export const installChunkErrorRecovery = () => {
  if (installed || typeof window === "undefined") return;
  installed = true;

  window.addEventListener("vite:preloadError", () => {
    reloadForNewVersion();
  });
};
