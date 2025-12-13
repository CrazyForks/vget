import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(),
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    react(),
  ],
  server: {
    proxy: {
      "/health": "http://localhost:8080",
      "/download": "http://localhost:8080",
      "/status": "http://localhost:8080",
      "/jobs": "http://localhost:8080",
      "/config": "http://localhost:8080",
      "/i18n": "http://localhost:8080",
      "/kuaidi100": "http://localhost:8080",
    },
  },
});
