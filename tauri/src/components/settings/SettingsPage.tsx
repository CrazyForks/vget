import { useEffect, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { ArrowLeft, Settings, Globe, Info } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { GeneralSettings } from "./GeneralSettings";
import { SiteSettings } from "./SiteSettings";
import { AboutSettings } from "./AboutSettings";
import type { Config } from "./types";
import { cn } from "@/lib/utils";

type SettingsSection = "general" | "sites" | "about";

const sectionIcons: Record<SettingsSection, React.ComponentType<{ className?: string }>> = {
  general: Settings,
  sites: Globe,
  about: Info,
};

export function SettingsPage() {
  const { t } = useTranslation();
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [activeSection, setActiveSection] =
    useState<SettingsSection>("general");

  useEffect(() => {
    invoke<Config>("get_config")
      .then((configData) => {
        // Ensure nested objects have defaults
        setConfig({
          ...configData,
          twitter: configData.twitter ?? { auth_token: null },
          bilibili: configData.bilibili ?? { cookie: null },
          server: configData.server ?? { max_concurrent: 10 },
          webdav_servers: configData.webdav_servers ?? {},
          express: configData.express ?? { kuaidi100: null },
        });
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const updateConfig = (updates: Partial<Config>) => {
    if (!config) return;
    setConfig({ ...config, ...updates });
    setDirty(true);
  };

  const saveConfig = async () => {
    if (!config) return;
    setSaving(true);
    try {
      await invoke("save_config", { config });
      setDirty(false);
    } catch (err) {
      console.error("Failed to save config:", err);
    } finally {
      setSaving(false);
    }
  };

  const sections: { id: SettingsSection; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
    { id: "general", label: t("settings.sections.general"), icon: sectionIcons.general },
    { id: "sites", label: t("settings.sections.sites"), icon: sectionIcons.sites },
    { id: "about", label: t("settings.sections.about"), icon: sectionIcons.about },
  ];

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <p className="text-muted-foreground">{t("settings.loading")}</p>
      </div>
    );
  }

  if (!config) {
    return (
      <div className="flex h-screen items-center justify-center">
        <p className="text-destructive">{t("settings.loadFailed")}</p>
      </div>
    );
  }

  const renderSection = () => {
    switch (activeSection) {
      case "general":
        return <GeneralSettings config={config} onUpdate={updateConfig} />;
      case "sites":
        return <SiteSettings config={config} onUpdate={updateConfig} />;
      case "about":
        return <AboutSettings />;
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Sidebar */}
      <aside className="w-56 border-r bg-muted/30 flex flex-col shrink-0">
        <div className="h-14 px-4 border-b flex items-center">
          <Link
            to="/"
            className="flex items-center gap-2 text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            <span className="text-sm font-medium">{t("settings.back")}</span>
          </Link>
        </div>

        <nav className="flex-1 p-2">
          <ul className="space-y-1">
            {sections.map((section) => {
              const Icon = section.icon;
              return (
                <li key={section.id}>
                  <button
                    onClick={() => setActiveSection(section.id)}
                    className={cn(
                      "w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md transition-colors",
                      activeSection === section.id
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground"
                    )}
                  >
                    <Icon className="h-4 w-4" />
                    {section.label}
                  </button>
                </li>
              );
            })}
          </ul>
        </nav>

        <div className="mt-auto p-4 border-t">
          <p className="text-xs text-muted-foreground">{t("nav.vgetDesktop")}</p>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <header className="h-14 border-b flex items-center justify-between px-6 shrink-0">
          <h1 className="text-lg font-semibold">
            {sections.find((s) => s.id === activeSection)?.label}
          </h1>
          <div className="flex items-center gap-3">
            {dirty && (
              <span className="text-sm text-muted-foreground">
                {t("settings.unsavedChanges")}
              </span>
            )}
            <Button onClick={saveConfig} disabled={!dirty || saving} size="sm">
              {saving ? t("settings.saving") : t("settings.save")}
            </Button>
          </div>
        </header>

        <div className="flex-1 overflow-auto">
          <div className="max-w-2xl p-6">{renderSection()}</div>
        </div>
      </main>
    </div>
  );
}
