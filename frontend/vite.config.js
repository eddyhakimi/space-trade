import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Requests arrive via nginx, so the Host header is whatever the browser used
    allowedHosts: true,
    // Hot-reload websocket goes through nginx on port 8080
    hmr: { clientPort: 8080 },
  },
});
