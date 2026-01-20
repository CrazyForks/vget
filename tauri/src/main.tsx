import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider, createRouter } from "@tanstack/react-router";
import { invoke } from "@tauri-apps/api/core";
import "./index.css";
import { routeTree } from "./routeTree.gen";
import './i18n';
import { changeLanguage } from './i18n';

// Theme management
let currentTheme = "light";
const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");

function applyTheme(theme: string) {
  currentTheme = theme;
  const root = document.documentElement;

  if (theme === "dark") {
    root.classList.add("dark");
  } else if (theme === "system") {
    root.classList.toggle("dark", mediaQuery.matches);
  } else {
    root.classList.remove("dark");
  }
}

// Listen for system theme changes
mediaQuery.addEventListener("change", (e) => {
  if (currentTheme === "system") {
    document.documentElement.classList.toggle("dark", e.matches);
  }
});

// Apply theme and language on startup from config
invoke<{ theme: string; language: string }>("get_config")
  .then((config) => {
    applyTheme(config.theme || "light");
    changeLanguage(config.language || "en");
  })
  .catch(() => {
    // Config not available yet, defaults applied
  });

// Export for use in settings
(window as any).__applyTheme = applyTheme;
(window as any).__changeLanguage = changeLanguage;

const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>
);
