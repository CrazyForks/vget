import { useState } from "react";
import { check } from "@tauri-apps/plugin-updater";
import { relaunch } from "@tauri-apps/plugin-process";
import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ExternalLink, RefreshCw } from "lucide-react";
import logo from "@/assets/logo.png";

export function AboutSettings() {
  const { t } = useTranslation();
  const [checking, setChecking] = useState(false);
  const [updateAvailable, setUpdateAvailable] = useState<string | null>(null);
  const [downloading, setDownloading] = useState(false);

  const checkForUpdates = async () => {
    setChecking(true);
    try {
      const update = await check();
      if (update) {
        setUpdateAvailable(update.version);
      } else {
        setUpdateAvailable(null);
      }
    } catch (err) {
      console.error("Update check failed:", err);
    } finally {
      setChecking(false);
    }
  };

  const downloadAndInstall = async () => {
    setDownloading(true);
    try {
      const update = await check();
      if (update) {
        await update.downloadAndInstall();
        await relaunch();
      }
    } catch (err) {
      console.error("Update failed:", err);
    } finally {
      setDownloading(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.about.title")}</CardTitle>
          <CardDescription>{t("settings.about.desc")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center gap-4">
            <img src={logo} alt="VGet" className="h-16 w-16" />
            <div>
              <h3 className="text-lg font-semibold">{t("nav.vgetDesktop")}</h3>
              <p className="text-sm text-muted-foreground">{t("settings.about.version")} 0.1.0</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              onClick={checkForUpdates}
              disabled={checking || downloading}
            >
              <RefreshCw
                className={`h-4 w-4 mr-2 ${checking ? "animate-spin" : ""}`}
              />
              {checking ? t("settings.about.checking") : t("settings.about.checkForUpdates")}
            </Button>

            {updateAvailable && (
              <Button onClick={downloadAndInstall} disabled={downloading}>
                {downloading
                  ? t("settings.about.downloading")
                  : t("settings.about.updateTo", { version: updateAvailable })}
              </Button>
            )}
          </div>

          {updateAvailable === null && !checking && (
            <p className="text-sm text-muted-foreground">
              {t("settings.about.latestVersion")}
            </p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.about.links")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <a
            href="https://github.com/guiyumin/vget"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ExternalLink className="h-4 w-4" />
            {t("settings.about.githubRepo")}
          </a>
          <a
            href="https://github.com/guiyumin/vget/issues"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
          >
            <ExternalLink className="h-4 w-4" />
            {t("settings.about.reportIssue")}
          </a>
        </CardContent>
      </Card>
    </div>
  );
}
