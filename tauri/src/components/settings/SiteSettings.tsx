import { useState, useEffect, useRef, useCallback } from "react";
import { QRCodeSVG } from "qrcode.react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Eye,
  EyeOff,
  CheckCircle2,
  Loader2,
  RefreshCw,
  LogOut,
  ExternalLink,
  Server,
} from "lucide-react";
import { toast } from "sonner";
import type { Config } from "./types";
import {
  useAuthStore,
  generateBilibiliQR,
  pollBilibiliQR,
  saveBilibiliCookie,
  openXhsLoginWindow,
  QR_WAITING,
  QR_SCANNED,
  QR_EXPIRED,
  QR_CONFIRMED,
  type QRSession,
} from "@/stores/auth";
import {
  getDockerServerUrl,
  setDockerServerUrl,
  checkDockerHealth,
  getDockerJwtToken,
  setDockerJwtToken,
} from "@/services/dockerApi";

interface SiteSettingsProps {
  config: Config;
  onUpdate: (updates: Partial<Config>) => void;
}

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

export function SiteSettings({ config, onUpdate }: SiteSettingsProps) {
  const { t } = useTranslation();
  const [showTwitterToken, setShowTwitterToken] = useState(false);
  const { bilibili, xiaohongshu, setBilibiliStatus, logout, checkAuthStatus } =
    useAuthStore();

  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  return (
    <div className="space-y-6">
      {/* Twitter */}
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.sites.twitter.title")}</CardTitle>
          <CardDescription>
            {t("settings.sites.twitter.desc")}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="twitter_auth_token">{t("settings.sites.twitter.authToken")}</Label>
            <div className="flex gap-2">
              <Input
                id="twitter_auth_token"
                type={showTwitterToken ? "text" : "password"}
                value={config.twitter?.auth_token || ""}
                onChange={(e) =>
                  onUpdate({
                    twitter: {
                      ...config.twitter,
                      auth_token: e.target.value || null,
                    },
                  })
                }
                placeholder={t("settings.sites.twitter.tokenPlaceholder")}
                className="flex-1"
              />
              <Button
                variant="outline"
                size="icon"
                onClick={() => setShowTwitterToken(!showTwitterToken)}
              >
                {showTwitterToken ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
            </div>
            <p className="text-sm text-muted-foreground">
              {t("settings.sites.twitter.tokenHint")}
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Bilibili */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {t("settings.sites.bilibili.title")}
            {bilibili.status === "logged_in" && (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            )}
          </CardTitle>
          <CardDescription>
            {bilibili.status === "logged_in"
              ? bilibili.username
                ? t("settings.sites.bilibili.loggedInAs", { username: bilibili.username })
                : t("settings.sites.bilibili.loggedIn")
              : t("settings.sites.bilibili.desc")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {bilibili.status === "logged_in" ? (
            <Button
              variant="outline"
              onClick={async () => {
                try {
                  await logout("bilibili");
                  toast.success(t("settings.sites.logoutSuccess"));
                } catch {
                  toast.error(t("settings.sites.logoutFailed"));
                }
              }}
            >
              <LogOut className="h-4 w-4 mr-2" />
              {t("settings.sites.bilibili.logout")}
            </Button>
          ) : bilibili.status === "checking" ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("settings.sites.bilibili.checkingStatus")}
            </div>
          ) : (
            <Tabs defaultValue="qr">
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="qr">{t("settings.sites.bilibili.qrCode")}</TabsTrigger>
                <TabsTrigger value="cookie">{t("settings.sites.bilibili.cookie")}</TabsTrigger>
              </TabsList>
              <TabsContent value="qr" className="mt-4">
                <BilibiliQRLogin
                  onSuccess={(username) => {
                    setBilibiliStatus({ status: "logged_in", username });
                    toast.success(t("settings.sites.welcome", { username: username || "User" }));
                  }}
                />
              </TabsContent>
              <TabsContent value="cookie" className="mt-4">
                <BilibiliCookieLogin
                  onSuccess={(username) => {
                    setBilibiliStatus({ status: "logged_in", username });
                    toast.success(t("settings.sites.loginSuccess"));
                  }}
                />
              </TabsContent>
            </Tabs>
          )}
        </CardContent>
      </Card>

      {/* Xiaohongshu */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {t("settings.sites.xiaohongshu.title")}
            {xiaohongshu.status === "logged_in" && (
              <CheckCircle2 className="h-4 w-4 text-green-500" />
            )}
          </CardTitle>
          <CardDescription>
            {xiaohongshu.status === "logged_in"
              ? xiaohongshu.username
                ? t("settings.sites.xiaohongshu.loggedInAs", { username: xiaohongshu.username })
                : t("settings.sites.xiaohongshu.sessionSaved")
              : t("settings.sites.xiaohongshu.desc")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {xiaohongshu.status === "logged_in" ? (
            <Button
              variant="outline"
              onClick={async () => {
                try {
                  await logout("xiaohongshu");
                  toast.success(t("settings.sites.logoutSuccess"));
                } catch {
                  toast.error(t("settings.sites.logoutFailed"));
                }
              }}
            >
              <LogOut className="h-4 w-4 mr-2" />
              {t("settings.sites.bilibili.logout")}
            </Button>
          ) : xiaohongshu.status === "checking" ? (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("settings.sites.bilibili.checkingStatus")}
            </div>
          ) : (
            <XiaohongshuLogin />
          )}
        </CardContent>
      </Card>

      {/* Docker Server (for YouTube) */}
      <DockerServerSettings />
    </div>
  );
}

function BilibiliQRLogin({
  onSuccess,
}: {
  onSuccess: (username?: string) => void;
}) {
  const { t } = useTranslation();
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
      setError(
        err instanceof Error ? err.message : t("settings.sites.bilibili.failedToGenerateQR")
      );
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

  useEffect(() => {
    generateQR();
  }, [generateQR]);

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
        return t("settings.sites.bilibili.scanWithApp");
      case QR_SCANNED:
        return t("settings.sites.bilibili.confirmLogin");
      case QR_EXPIRED:
        return t("settings.sites.bilibili.qrExpired");
      case QR_CONFIRMED:
        return t("settings.sites.bilibili.loginSuccess");
      default:
        return "";
    }
  };

  return (
    <div className="flex flex-col items-center">
      <div className="mb-4 p-4 bg-white rounded-lg">
        {generating ? (
          <div className="w-40 h-40 flex items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : error ? (
          <div className="w-40 h-40 flex items-center justify-center text-destructive text-center text-sm p-4">
            {error}
          </div>
        ) : qrSession ? (
          <QRCodeSVG
            value={qrSession.url}
            size={160}
            level="L"
            className={qrStatus === QR_EXPIRED ? "opacity-30" : ""}
          />
        ) : (
          <div className="w-40 h-40 flex items-center justify-center text-muted-foreground">
            {t("settings.sites.bilibili.waiting")}
          </div>
        )}
      </div>

      <div className="mb-4 text-center text-sm">
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
        <Button onClick={generateQR} disabled={generating} variant="outline" size="sm">
          <RefreshCw className="h-4 w-4 mr-2" />
          {t("settings.sites.bilibili.refreshQR")}
        </Button>
      )}
    </div>
  );
}

function BilibiliCookieLogin({
  onSuccess,
}: {
  onSuccess: (username?: string) => void;
}) {
  const { t } = useTranslation();
  const [fields, setFields] = useState<CookieFields>({
    sessdata: "",
    biliJct: "",
    dedeUserId: "",
  });
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    const cookie = buildCookie(fields);
    if (!cookie) {
      toast.error(t("settings.sites.bilibili.fillOneField"));
      return;
    }

    setSaving(true);
    try {
      await saveBilibiliCookie(cookie);
      onSuccess();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.sites.bilibili.failedToSave"));
    } finally {
      setSaving(false);
    }
  };

  const hasAnyInput = fields.sessdata || fields.biliJct || fields.dedeUserId;

  return (
    <div className="space-y-4">
      <div className="p-3 bg-muted rounded-lg text-sm text-muted-foreground">
        <p className="font-medium mb-2">{t("settings.sites.bilibili.cookieInstructions")}</p>
        <ol className="list-decimal list-inside space-y-1">
          <li>{t("settings.sites.bilibili.step1")}</li>
          <li>{t("settings.sites.bilibili.step2")}</li>
          <li>{t("settings.sites.bilibili.step3")}</li>
          <li>{t("settings.sites.bilibili.step4")}</li>
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
            placeholder={t("settings.sites.bilibili.pasteSessdata")}
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
            placeholder={t("settings.sites.bilibili.pasteBiliJct")}
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
            placeholder={t("settings.sites.bilibili.pasteDedeUserId")}
            className="font-mono text-sm"
          />
        </div>
      </div>

      <Button onClick={handleSave} disabled={saving || !hasAnyInput} className="w-full">
        {saving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
        {t("common.save")}
      </Button>
    </div>
  );
}

function XiaohongshuLogin() {
  const { t } = useTranslation();
  const { checkAuthStatus } = useAuthStore();
  const [opening, setOpening] = useState(false);

  const handleOpenLogin = async () => {
    setOpening(true);
    try {
      await openXhsLoginWindow();
      setTimeout(async () => {
        await checkAuthStatus();
        const state = useAuthStore.getState();
        if (state.xiaohongshu.status === "logged_in") {
          toast.success(t("settings.sites.loginSuccess"));
        }
      }, 1000);
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("settings.sites.xiaohongshu.failedToOpenLogin")
      );
    } finally {
      setOpening(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="p-3 bg-muted rounded-lg text-sm text-muted-foreground">
        <p className="mb-2">{t("settings.sites.xiaohongshu.loginInstructions")}</p>
        <ul className="list-disc list-inside space-y-1">
          <li>{t("settings.sites.xiaohongshu.scanWithApp")}</li>
          <li>{t("settings.sites.xiaohongshu.orLoginPhone")}</li>
          <li>{t("settings.sites.xiaohongshu.closeWhenDone")}</li>
        </ul>
      </div>

      <div className="flex gap-2">
        <Button onClick={handleOpenLogin} disabled={opening} className="flex-1">
          {opening ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : (
            <ExternalLink className="h-4 w-4 mr-2" />
          )}
          {opening ? t("settings.sites.xiaohongshu.opening") : t("settings.sites.xiaohongshu.openLoginWindow")}
        </Button>

        <Button variant="outline" onClick={checkAuthStatus}>
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

function DockerServerSettings() {
  const { t } = useTranslation();
  const [serverUrl, setServerUrl] = useState(getDockerServerUrl());
  const [jwtToken, setJwtToken] = useState(getDockerJwtToken());
  const [showToken, setShowToken] = useState(false);
  const [testing, setTesting] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<"unknown" | "connected" | "failed">("unknown");

  const handleTestConnection = async () => {
    setTesting(true);
    setConnectionStatus("unknown");
    try {
      // Save settings first
      setDockerServerUrl(serverUrl);
      if (jwtToken) {
        setDockerJwtToken(jwtToken);
      }

      const isHealthy = await checkDockerHealth();
      setConnectionStatus(isHealthy ? "connected" : "failed");
      if (isHealthy) {
        toast.success(t("settings.sites.docker.connectionSuccess") || "Connected to Docker server!");
      } else {
        toast.error(t("settings.sites.docker.connectionFailed") || "Failed to connect to Docker server");
      }
    } catch {
      setConnectionStatus("failed");
      toast.error(t("settings.sites.docker.connectionFailed") || "Failed to connect to Docker server");
    } finally {
      setTesting(false);
    }
  };

  const handleSave = () => {
    setDockerServerUrl(serverUrl);
    setDockerJwtToken(jwtToken);
    toast.success(t("settings.sites.docker.saved") || "Docker server settings saved");
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          {t("settings.sites.docker.title") || "Docker Server (YouTube)"}
          {connectionStatus === "connected" && (
            <CheckCircle2 className="h-4 w-4 text-green-500" />
          )}
        </CardTitle>
        <CardDescription>
          {t("settings.sites.docker.desc") || "Configure vget-server for YouTube downloads. Run the Docker container or vget-server locally."}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2">
          <Label htmlFor="docker_server_url">
            {t("settings.sites.docker.serverUrl") || "Server URL"}
          </Label>
          <Input
            id="docker_server_url"
            type="text"
            value={serverUrl}
            onChange={(e) => setServerUrl(e.target.value)}
            placeholder="http://localhost:8080"
          />
          <p className="text-sm text-muted-foreground">
            {t("settings.sites.docker.serverUrlHint") || "URL of the vget-server (default: http://localhost:8080)"}
          </p>
        </div>

        <div className="grid gap-2">
          <Label htmlFor="docker_jwt_token">
            {t("settings.sites.docker.jwtToken") || "JWT Token (optional)"}
          </Label>
          <div className="flex gap-2">
            <Input
              id="docker_jwt_token"
              type={showToken ? "text" : "password"}
              value={jwtToken}
              onChange={(e) => setJwtToken(e.target.value)}
              placeholder={t("settings.sites.docker.jwtPlaceholder") || "Paste JWT token if server requires authentication"}
              className="flex-1 font-mono text-sm"
            />
            <Button
              variant="outline"
              size="icon"
              onClick={() => setShowToken(!showToken)}
            >
              {showToken ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            {t("settings.sites.docker.jwtHint") || "Only needed if the server has api_key configured. Get token from: POST /api/auth/token"}
          </p>
        </div>

        <div className="flex gap-2">
          <Button onClick={handleSave} variant="outline">
            {t("common.save")}
          </Button>
          <Button onClick={handleTestConnection} disabled={testing}>
            {testing ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4 mr-2" />
            )}
            {t("settings.sites.docker.testConnection") || "Test Connection"}
          </Button>
        </div>

        {connectionStatus === "failed" && (
          <div className="p-3 bg-destructive/10 border border-destructive/20 rounded-lg text-sm text-destructive">
            {t("settings.sites.docker.notRunningHint") || "Docker server is not running. Start it with: docker run -p 8080:8080 ghcr.io/guiyumin/vget:latest"}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
