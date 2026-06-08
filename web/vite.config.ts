import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

// Single build, configured at deploy time. VITE_SCOPE = onboard | shore drives the
// vessel switcher and fleet views. VITE_API_BASE points at the local API.
export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.ico"],
      manifest: {
        name: "Mooring Line Management",
        short_name: "Mooring",
        theme_color: "#0b3d5c",
        background_color: "#0e1726",
        display: "standalone",
        orientation: "any",
        icons: [
          { src: "pwa-192.png", sizes: "192x192", type: "image/png" },
          { src: "pwa-512.png", sizes: "512x512", type: "image/png" },
        ],
      },
      workbox: {
        // The SPA navigation fallback serves index.html for navigations; exclude the
        // server-rendered API reference + spec so they reach nginx/the API instead of
        // being hijacked into the app.
        navigateFallbackDenylist: [/^\/docs(\/|$)/, /^\/openapi\./],
        // Offline read access to line data and status (spec §6). API GETs are cached
        // stale-while-revalidate so deck views and the register work on deck offline.
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith("/api/"),
            handler: "StaleWhileRevalidate",
            options: {
              cacheName: "api-read",
              expiration: { maxEntries: 500, maxAgeSeconds: 60 * 60 * 24 * 7 },
            },
          },
        ],
      },
    }),
  ],
  server: {
    port: 5173,
    allowedHosts: true,
    proxy: {
      "/api": {
        target: process.env.VITE_API_BASE || "http://localhost:8080",
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/api/, ""),
      },
      // Self-hosted API reference + spec, proxied to the API unchanged so
      // http://localhost:5173/docs works in dev too.
      "/docs": { target: process.env.VITE_API_BASE || "http://localhost:8080", changeOrigin: true },
      "/openapi.json": { target: process.env.VITE_API_BASE || "http://localhost:8080", changeOrigin: true },
      "/openapi.yaml": { target: process.env.VITE_API_BASE || "http://localhost:8080", changeOrigin: true },
    },
  },
});
