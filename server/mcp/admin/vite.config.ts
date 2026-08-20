import { dirname, resolve } from "path";
import { fileURLToPath } from "url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const root = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: "/",
  build: {
    outDir: resolve(root, "../internal/admin/static"),
    emptyOutDir: true,
    assetsDir: "assets",
  },
  server: {
    port: 5174,
    proxy: {
      "/api": "http://127.0.0.1:8788",
    },
  },
});
