/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolve } from "node:path";

// Output is consumed by the Go server via `internal/web/dist`. Build
// directly into that location so `go build` picks it up.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: resolve(__dirname, "../internal/web/dist"),
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
  },
});
