import { fileURLToPath } from "node:url";
import { createServer } from "vite";

const root = fileURLToPath(new URL("../../", import.meta.url));

export const createConsoleModules = async () => {
  const server = await createServer({
    root,
    configFile: false,
    logLevel: "error",
    resolve: { alias: { "@": `${root}src` } },
    optimizeDeps: { noDiscovery: true, include: [] },
    server: { middlewareMode: true, watch: null, hmr: false, ws: false },
  });
  return {
    load: (name) => server.ssrLoadModule(`/src/${name}`),
    close: () => server.close(),
  };
};
