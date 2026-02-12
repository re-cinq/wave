import { defineConfig } from "vite";
import preact from "@preact/preset-vite";
import { resolve } from "path";

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: resolve(__dirname, "../internal/dashboard/static"),
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: "assets/index.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/[name][extname]",
      },
    },
    // Target small bundle size — use esbuild (built-in, no extra dep)
    minify: "esbuild",
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
