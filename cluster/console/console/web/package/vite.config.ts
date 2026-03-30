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
    // tailwindcss(),
    react(),
    svgr(),
    nodePolyfills({
      // Other module polyfills are enabled by default
      // Explicitly configure globals
      globals: {
        Buffer: true, // This ensures the global Buffer is polyfilled
        global: true,
        process: true,
      },
      // You can also include 'buffer' in the 'include' array if you are using that option
      // include: ['buffer'],
    }),
    /*
    visualizer({
      emitFile: true,
      filename: "tmp/stats.html",
    }),
    */
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
        } catch (error) {
          return "auto";
        }
      },
      transformMixedEsModules: true,
    } as RollupCommonJSOptions,
  },

  server: {
    proxy: {
      "/octelium.api": {
        target: "http://127.0.0.1:10003",
        // changeOrigin: true,
        // secure: false,
        // proxyTimeout: 5000,
        headers: {
          "x-octelium": "octelium",
          "content-type": "application/grpc-web-text+proto",
        },
      },
    },
  },
});
