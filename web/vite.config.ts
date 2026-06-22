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
        // Never serve the SPA shell (navigateFallback) for /api/* navigations —
        // otherwise full-page navigations like /api/auth/login are intercepted by
        // the service worker and answered with index.html instead of reaching the
        // backend, which breaks the OIDC redirect flow (endless login loop).
        navigateFallbackDenylist: [/^\/api\//],
        // Offline read access to line data and status (spec §6). API GETs are cached
        // stale-while-revalidate so deck views and the register work on deck offline.
        // Auth (/api/auth/*) and admin access-control (/api/access/*) are EXCLUDED —
        // session/login/callback and live permission config must always hit the
        // network, never be served stale from cache.
        runtimeCaching: [
          {
            urlPattern: ({ url }) =>
              url.pathname.startsWith("/api/") &&
              !url.pathname.startsWith("/api/auth/") &&
              !url.pathname.startsWith("/api/access/"),
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
    },
  },
});
