import { useEffect } from "react";
import { open } from "@tauri-apps/plugin-dialog";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Folder } from "lucide-react";
import type { Config } from "./types";

// Use global functions from main.tsx
const applyTheme = (window as any).__applyTheme as (theme: string) => void;
const changeLanguage = (window as any).__changeLanguage as (lang: string) => void;

interface GeneralSettingsProps {
  config: Config;
  onUpdate: (updates: Partial<Config>) => void;
}

export function GeneralSettings({ config, onUpdate }: GeneralSettingsProps) {
  const { t } = useTranslation();
  const theme = config.theme || "light";
  const language = config.language || "en";

  useEffect(() => {
    applyTheme?.(theme);
  }, [theme]);

  useEffect(() => {
    changeLanguage?.(language);
  }, [language]);

  const handleSelectFolder = async () => {
    const selected = await open({
      directory: true,
      multiple: false,
      title: t("settings.general.selectDirectory"),
    });
    if (selected) {
      onUpdate({ output_dir: selected as string });
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t("settings.general.downloads")}</CardTitle>
          <CardDescription>{t("settings.general.downloadsDesc")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label htmlFor="output_dir">{t("settings.general.downloadLocation")}</Label>
            <div className="flex gap-2">
              <Input
                id="output_dir"
                value={config.output_dir}
                onChange={(e) => onUpdate({ output_dir: e.target.value })}
                className="flex-1"
              />
              <Button variant="outline" size="icon" onClick={handleSelectFolder}>
                <Folder className="h-4 w-4" />
              </Button>
            </div>
            <p className="text-sm text-muted-foreground">
              {t("settings.general.downloadLocationHint")}
            </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="format">{t("settings.general.defaultFormat")}</Label>
            <Select
              value={config.format}
              onValueChange={(value) => onUpdate({ format: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("settings.general.selectFormat")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="mp4">MP4</SelectItem>
                <SelectItem value="webm">WebM</SelectItem>
                <SelectItem value="best">{t("settings.general.bestAvailable")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="quality">{t("settings.general.defaultQuality")}</Label>
            <Select
              value={config.quality}
              onValueChange={(value) => onUpdate({ quality: value })}
            >
              <SelectTrigger>
                <SelectValue placeholder={t("settings.general.selectQuality")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="best">{t("settings.general.bestAvailable")}</SelectItem>
                <SelectItem value="1080p">1080p</SelectItem>
                <SelectItem value="720p">720p</SelectItem>
                <SelectItem value="480p">480p</SelectItem>
              </SelectContent>
            </Select>
            <p className="text-sm text-muted-foreground">
              {t("settings.general.qualityHint")}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.general.language")}</CardTitle>
          <CardDescription>{t("settings.general.languageDesc")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Select
            value={config.language}
            onValueChange={(value) => onUpdate({ language: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder={t("settings.general.selectLanguage")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="en">English</SelectItem>
              <SelectItem value="zh">中文</SelectItem>
              <SelectItem value="jp">日本語</SelectItem>
              <SelectItem value="kr">한국어</SelectItem>
              <SelectItem value="es">Español</SelectItem>
              <SelectItem value="fr">Français</SelectItem>
              <SelectItem value="de">Deutsch</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t("settings.general.theme")}</CardTitle>
          <CardDescription>{t("settings.general.themeDesc")}</CardDescription>
        </CardHeader>
        <CardContent>
          <Select value={theme} onValueChange={(value) => onUpdate({ theme: value })}>
            <SelectTrigger>
              <SelectValue placeholder={t("settings.general.selectTheme")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="light">{t("settings.general.light")}</SelectItem>
              <SelectItem value="dark">{t("settings.general.dark")}</SelectItem>
              <SelectItem value="system">{t("settings.general.system")}</SelectItem>
            </SelectContent>
          </Select>
        </CardContent>
      </Card>
    </div>
  );
}
