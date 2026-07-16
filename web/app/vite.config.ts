import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The SPA is served by the Go binary at the root. The dev server proxies /api
// to a locally running vmt instance.
export default defineConfig({
  base: "/",
  // Build stamp: versions the service-worker URL/cache so each deploy rolls
  // clients forward (paired with the update toast).
  define: { __BUILD_ID__: JSON.stringify(Date.now().toString(36)) },
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
