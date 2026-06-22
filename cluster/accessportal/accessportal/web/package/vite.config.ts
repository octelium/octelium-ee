import react from "@vitejs/plugin-react-swc";
import path from "path";
import { defineConfig } from "vite";
import svgr from "vite-plugin-svgr";

import type { RollupCommonJSOptions } from "@rollup/plugin-commonjs";
import { createRequire } from "module";

import { nodePolyfills } from "vite-plugin-node-polyfills";

const require = createRequire(import.meta.url);
const __dirname = path.resolve();

export default defineConfig({
  plugins: [
    react(),
    svgr(),
    nodePolyfills({
      globals: {
        Buffer: true,
        global: true,
        process: true,
      },
    }),
  ],

  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },

  build: {
    manifest: true,
    commonjsOptions: {
      defaultIsModuleExports(id) {
        try {
          const module = require(id);
          if (module?.default) {
            return false;
          }
          return "auto";
        } catch {
          return "auto";
        }
      },
      transformMixedEsModules: true,
    } as RollupCommonJSOptions,
  },

  server: {
    host: "0.0.0.0",

    proxy: {
      "/octelium.api": {
        target: "http://127.0.0.1:10003",
        changeOrigin: true,
        secure: false,
        ws: false,

        headers: {
          "x-octelium": "octelium",
          "content-type": "application/grpc-web-text+proto",
        },

        configure(proxy) {
          proxy.on("error", (err, req) => {
            console.error("[vite grpc-web proxy error]", {
              url: req.url,
              message: err.message,
              stack: err.stack,
            });
          });

          proxy.on("proxyReq", (proxyReq, req) => {
            console.log("[vite grpc-web proxy request]", {
              method: req.method,
              url: req.url,
              upstreamPath: proxyReq.path,
            });
          });

          proxy.on("proxyRes", (proxyRes, req) => {
            console.log("[vite grpc-web proxy response]", {
              url: req.url,
              statusCode: proxyRes.statusCode,
              contentType: proxyRes.headers["content-type"],
              grpcStatus: proxyRes.headers["grpc-status"],
            });
          });
        },
      },
    },
  },
});
