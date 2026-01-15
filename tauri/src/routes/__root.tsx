import { createRootRoute, Outlet } from "@tanstack/react-router";
import { useState } from "react";
import { Toaster } from "sonner";
import { AppSidebar } from "@/components/AppSidebar";

function RootLayout() {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);

  return (
    <div className="flex h-screen bg-background">
      <AppSidebar
        collapsed={sidebarCollapsed}
        onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
      />
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
      <Toaster position="bottom-right" />
    </div>
  );
}

export const Route = createRootRoute({
  component: RootLayout,
});
