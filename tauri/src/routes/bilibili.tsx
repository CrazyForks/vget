import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect, useRef, useCallback } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { CheckCircle2, Loader2, RefreshCw, LogOut } from "lucide-react";
import { toast } from "sonner";
import {
  useAuthStore,
  generateBilibiliQR,
  pollBilibiliQR,
  saveBilibiliCookie,
  QR_WAITING,
  QR_SCANNED,
  QR_EXPIRED,
  QR_CONFIRMED,
  type QRSession,
} from "@/stores/auth";

export const Route = createFileRoute("/bilibili")({
  component: BilibiliPage,
});

interface CookieFields {
  sessdata: string;
  biliJct: string;
  dedeUserId: string;
}

function buildCookie(fields: CookieFields): string {
  const parts: string[] = [];
  if (fields.sessdata) parts.push(`SESSDATA=${fields.sessdata}`);
  if (fields.biliJct) parts.push(`bili_jct=${fields.biliJct}`);
  if (fields.dedeUserId) parts.push(`DedeUserID=${fields.dedeUserId}`);
  return parts.join("; ");
}

function BilibiliPage() {
  const { bilibili, setBilibiliStatus, logout, checkAuthStatus } = useAuthStore();

  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  // Logged in view
  if (bilibili.status === "logged_in") {
    return (
      <div className="h-full">
        <header className="h-14 border-b border-border flex items-center px-6">
          <h1 className="text-xl font-semibold">Bilibili</h1>
        </header>

        <div className="p-6 max-w-md">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <CheckCircle2 className="h-5 w-5 text-green-500" />
                Logged In
              </CardTitle>
              <CardDescription>
                Welcome, {bilibili.username || "User"}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button
                variant="outline"
                className="w-full"
                onClick={async () => {
                  try {
                    await logout("bilibili");
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
  if (bilibili.status === "checking") {
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
        <h1 className="text-xl font-semibold">Bilibili</h1>
      </header>

      <div className="p-6 max-w-md">
        <p className="text-sm text-muted-foreground mb-4">
          Login to download high-quality videos and member content
        </p>
        <Tabs defaultValue="qr">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="qr">QR Code</TabsTrigger>
            <TabsTrigger value="cookie">Cookie</TabsTrigger>
          </TabsList>
          <TabsContent value="qr" className="mt-4">
            <QRLogin
              onSuccess={(username) => {
                setBilibiliStatus({ status: "logged_in", username });
                toast.success(`Welcome, ${username || "User"}!`);
              }}
            />
          </TabsContent>
          <TabsContent value="cookie" className="mt-4">
            <CookieLogin
              onSuccess={(username) => {
                setBilibiliStatus({ status: "logged_in", username });
                toast.success("Login successful!");
              }}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

function QRLogin({ onSuccess }: { onSuccess: (username?: string) => void }) {
  const [qrSession, setQrSession] = useState<QRSession | null>(null);
  const [qrStatus, setQrStatus] = useState<number | null>(null);
  const [generating, setGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollIntervalRef = useRef<number | null>(null);

  const generateQR = useCallback(async () => {
    setGenerating(true);
    setError(null);
    setQrStatus(null);

    try {
      const session = await generateBilibiliQR();
      setQrSession(session);
      setQrStatus(QR_WAITING);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to generate QR code");
    } finally {
      setGenerating(false);
    }
  }, []);

  const pollStatus = useCallback(async () => {
    if (!qrSession) return;

    try {
      const result = await pollBilibiliQR(qrSession.qrcode_key);
      setQrStatus(result.status);

      if (result.status === QR_CONFIRMED) {
        if (pollIntervalRef.current) {
          clearInterval(pollIntervalRef.current);
          pollIntervalRef.current = null;
        }
        onSuccess(result.username);
      } else if (result.status === QR_EXPIRED) {
        if (pollIntervalRef.current) {
          clearInterval(pollIntervalRef.current);
          pollIntervalRef.current = null;
        }
      }
    } catch (err) {
      console.error("Poll error:", err);
    }
  }, [qrSession, onSuccess]);

  // Generate QR on mount
  useEffect(() => {
    generateQR();
  }, [generateQR]);

  // Poll status
  useEffect(() => {
    const shouldPoll =
      qrSession && (qrStatus === QR_WAITING || qrStatus === QR_SCANNED);

    if (shouldPoll) {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
      pollIntervalRef.current = window.setInterval(pollStatus, 1500);
    }

    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
        pollIntervalRef.current = null;
      }
    };
  }, [qrSession, qrStatus, pollStatus]);

  const getStatusText = () => {
    switch (qrStatus) {
      case QR_WAITING:
        return "Scan with Bilibili app";
      case QR_SCANNED:
        return "Confirm login on your phone";
      case QR_EXPIRED:
        return "QR code expired";
      case QR_CONFIRMED:
        return "Login successful!";
      default:
        return "";
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>QR Code Login</CardTitle>
        <CardDescription>Scan with the Bilibili mobile app</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col items-center">
        <div className="mb-4 p-4 bg-white rounded-lg">
          {generating ? (
            <div className="w-48 h-48 flex items-center justify-center">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <div className="w-48 h-48 flex items-center justify-center text-destructive text-center text-sm p-4">
              {error}
            </div>
          ) : qrSession ? (
            <QRCodeSVG
              value={qrSession.url}
              size={192}
              level="L"
              className={qrStatus === QR_EXPIRED ? "opacity-30" : ""}
            />
          ) : (
            <div className="w-48 h-48 flex items-center justify-center text-muted-foreground">
              Waiting...
            </div>
          )}
        </div>

        <div className="mb-4 text-center">
          {qrStatus === QR_SCANNED ? (
            <span className="text-green-600 font-medium flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              {getStatusText()}
            </span>
          ) : qrStatus === QR_EXPIRED ? (
            <span className="text-destructive">{getStatusText()}</span>
          ) : (
            <span className="text-muted-foreground">{getStatusText()}</span>
          )}
        </div>

        {(qrStatus === QR_EXPIRED || error) && (
          <Button onClick={generateQR} disabled={generating} variant="outline">
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh QR Code
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

function CookieLogin({ onSuccess }: { onSuccess: (username?: string) => void }) {
  const [fields, setFields] = useState<CookieFields>({
    sessdata: "",
    biliJct: "",
    dedeUserId: "",
  });
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    const cookie = buildCookie(fields);
    if (!cookie) {
      toast.error("Please fill in at least one field");
      return;
    }

    setSaving(true);
    try {
      await saveBilibiliCookie(cookie);
      onSuccess();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save cookie");
    } finally {
      setSaving(false);
    }
  };

  const hasAnyInput = fields.sessdata || fields.biliJct || fields.dedeUserId;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Cookie Login</CardTitle>
        <CardDescription>
          Enter cookie values from your browser
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="p-3 bg-muted rounded-lg text-sm text-muted-foreground">
          <p className="font-medium mb-2">How to get cookies:</p>
          <ol className="list-decimal list-inside space-y-1">
            <li>Open bilibili.com and login</li>
            <li>Press F12 to open DevTools</li>
            <li>Go to Application tab</li>
            <li>Find Cookies under Storage</li>
            <li>Copy the values below</li>
          </ol>
        </div>

        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="sessdata">SESSDATA</Label>
            <Input
              id="sessdata"
              value={fields.sessdata}
              onChange={(e) =>
                setFields((f) => ({ ...f, sessdata: e.target.value }))
              }
              placeholder="Paste SESSDATA value"
              className="font-mono text-sm"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="bili_jct">bili_jct</Label>
            <Input
              id="bili_jct"
              value={fields.biliJct}
              onChange={(e) =>
                setFields((f) => ({ ...f, biliJct: e.target.value }))
              }
              placeholder="Paste bili_jct value"
              className="font-mono text-sm"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dedeUserId">DedeUserID</Label>
            <Input
              id="dedeUserId"
              value={fields.dedeUserId}
              onChange={(e) =>
                setFields((f) => ({ ...f, dedeUserId: e.target.value }))
              }
              placeholder="Paste DedeUserID value"
              className="font-mono text-sm"
            />
          </div>
        </div>

        <Button
          onClick={handleSave}
          disabled={saving || !hasAnyInput}
          className="w-full"
        >
          {saving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
          Save
        </Button>
      </CardContent>
    </Card>
  );
}
