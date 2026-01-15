import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  CheckCircle2,
  Loader2,
  LogOut,
  ExternalLink,
  RefreshCw,
} from "lucide-react";
import { toast } from "sonner";
import { useAuthStore, openXhsLoginWindow } from "@/stores/auth";

export const Route = createFileRoute("/xiaohongshu")({
  component: XiaohongshuPage,
});

function XiaohongshuPage() {
  const { xiaohongshu, logout, checkAuthStatus } = useAuthStore();
  const [opening, setOpening] = useState(false);

  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  const handleOpenLogin = async () => {
    setOpening(true);
    try {
      await openXhsLoginWindow();
      // After window closes, check auth status
      setTimeout(async () => {
        await checkAuthStatus();
        const state = useAuthStore.getState();
        if (state.xiaohongshu.status === "logged_in") {
          toast.success("Login successful!");
        }
      }, 1000);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to open login window");
    } finally {
      setOpening(false);
    }
  };

  // Logged in view
  if (xiaohongshu.status === "logged_in") {
    return (
      <div className="h-full">
        <header className="h-14 border-b border-border flex items-center px-6">
          <h1 className="text-xl font-semibold">Xiaohongshu</h1>
        </header>

        <div className="p-6 max-w-md">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                Logged In
              </CardTitle>
              <CardDescription>
                {xiaohongshu.username
                  ? `Welcome, ${xiaohongshu.username}`
                  : "Session cookies saved"}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Button
                variant="outline"
                className="w-full"
                onClick={async () => {
                  try {
                    await logout("xiaohongshu");
                    toast.success("Logged out successfully");
                  } catch {
                    toast.error("Failed to logout");
                  }
                }}
              >
                <LogOut className="h-4 w-4 mr-2" />
                Logout
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    );
  }

  // Checking status view
  if (xiaohongshu.status === "checking") {
    return (
      <div className="h-full flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Login view
  return (
    <div className="h-full">
      <header className="h-14 border-b border-border flex items-center px-6">
        <h1 className="text-xl font-semibold">Xiaohongshu</h1>
      </header>

      <div className="p-6 max-w-md">
        <p className="text-sm text-muted-foreground mb-4">
          Login to download videos and images from Xiaohongshu
        </p>
        <Card>
          <CardHeader>
            <CardTitle>Browser Login</CardTitle>
            <CardDescription>
              Login via browser to access Xiaohongshu content
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="p-3 bg-muted rounded-lg text-sm text-muted-foreground">
              <p className="mb-2">
                Click the button below to open a login window:
              </p>
              <ul className="list-disc list-inside space-y-1">
                <li>Scan the QR code with Xiaohongshu app</li>
                <li>Or login with your phone number</li>
                <li>Close the window when done</li>
              </ul>
            </div>

            <Button onClick={handleOpenLogin} disabled={opening} className="w-full">
              {opening ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <ExternalLink className="h-4 w-4 mr-2" />
              )}
              {opening ? "Opening..." : "Open Login Window"}
            </Button>

            <Button
              variant="outline"
              className="w-full"
              onClick={checkAuthStatus}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Check Login Status
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
