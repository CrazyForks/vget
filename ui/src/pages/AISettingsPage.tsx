import { useState, useEffect } from "react";
import { useApp } from "../context/AppContext";
import { AISettings } from "../components/AISettings";
import { LocalSTTSettings } from "../components/LocalSTTSettings";
import { ModelDownloadSettings } from "../components/ModelDownloadSettings";
import {
  fetchLocalASRCapabilities,
  type LocalASRCapabilities,
} from "../utils/apis";

export function AISettingsPage() {
  const { t, isConnected } = useApp();

  const [loading, setLoading] = useState(true);
  const [capabilities, setCapabilities] = useState<LocalASRCapabilities | null>(
    null
  );

  useEffect(() => {
    const loadCapabilities = async () => {
      try {
        const res = await fetchLocalASRCapabilities();
        if (res.code === 200) {
          setCapabilities(res.data);
        }
      } catch {
        // Ignore errors
      } finally {
        setLoading(false);
      }
    };
    loadCapabilities();
  }, []);

  const hasGpu = capabilities?.gpu?.type === "nvidia";

  return (
    <div className="max-w-3xl mx-auto flex flex-col gap-4">
      <h1 className="text-xl font-medium text-zinc-900 dark:text-white">
        {t.ai_settings}
      </h1>
      <LocalSTTSettings
        isConnected={isConnected}
        loading={loading}
        capabilities={capabilities}
      />
      {hasGpu && <ModelDownloadSettings isConnected={isConnected} />}
      <AISettings isConnected={isConnected} />
    </div>
  );
}
