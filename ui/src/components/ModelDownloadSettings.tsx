import { useState, useEffect } from "react";
import { useApp } from "../context/AppContext";
import {
  fetchVmirrorModels,
  fetchVmirrorAuth,
  requestVmirrorDownloadURL,
  type VmirrorModel,
} from "../utils/apis";
import {
  FaDownload,
  FaCheck,
  FaSpinner,
  FaCircleExclamation,
} from "react-icons/fa6";

interface ModelDownloadSettingsProps {
  isConnected: boolean;
}

interface DownloadState {
  status: "idle" | "downloading" | "completed" | "error";
  error?: string;
}

export function ModelDownloadSettings({
  isConnected,
}: ModelDownloadSettingsProps) {
  const { t, showToast } = useApp();

  const [loading, setLoading] = useState(true);
  const [models, setModels] = useState<VmirrorModel[]>([]);
  const [email, setEmail] = useState("");
  const [emailInput, setEmailInput] = useState("");
  const [showEmailForm, setShowEmailForm] = useState(false);
  const [savingEmail, setSavingEmail] = useState(false);
  const [downloadStates, setDownloadStates] = useState<
    Record<string, DownloadState>
  >({});

  const loadData = async () => {
    try {
      const [modelsRes, authRes] = await Promise.all([
        fetchVmirrorModels(),
        fetchVmirrorAuth(),
      ]);

      if (modelsRes.code === 200) {
        setModels(modelsRes.data.models || []);
      }

      if (authRes.code === 200 && authRes.data.registered) {
        setEmail(authRes.data.email || "");
        setEmailInput(authRes.data.email || "");
      }
    } catch {
      // Ignore errors
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleSaveEmail = async () => {
    if (!emailInput.trim()) {
      showToast("error", t.model_download_email_required);
      return;
    }

    // Basic email validation
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(emailInput)) {
      showToast("error", t.model_download_invalid_email);
      return;
    }

    setSavingEmail(true);
    try {
      setEmail(emailInput);
      setShowEmailForm(false);
      showToast("success", t.model_download_email_saved);
    } finally {
      setSavingEmail(false);
    }
  };

  const handleDownload = async (modelName: string) => {
    // Check if email is set
    if (!email) {
      setShowEmailForm(true);
      return;
    }

    setDownloadStates((prev) => ({
      ...prev,
      [modelName]: { status: "downloading" },
    }));

    try {
      // Server downloads the model directly
      const res = await requestVmirrorDownloadURL(modelName, email);

      if (res.code !== 200) {
        const errorCode = res.data?.error_code;
        let errorMsg = res.message;
        if (errorCode === "RATE_LIMIT") {
          errorMsg = t.model_download_rate_limit;
        } else if (errorCode === "AUTH_SERVER_DOWN") {
          errorMsg = t.model_download_server_down;
        }
        setDownloadStates((prev) => ({
          ...prev,
          [modelName]: { status: "error", error: errorMsg },
        }));
        showToast("error", errorMsg);
        return;
      }

      // Mark as completed and refresh models list
      setDownloadStates((prev) => ({
        ...prev,
        [modelName]: { status: "completed" },
      }));

      showToast("success", t.model_download_success);

      // Refresh models list after a short delay
      setTimeout(() => {
        loadData();
        // Clear completed state after refresh
        setDownloadStates((prev) => {
          const newStates = { ...prev };
          delete newStates[modelName];
          return newStates;
        });
      }, 2000);
    } catch (err) {
      const errorMsg =
        err instanceof Error ? err.message : t.model_download_failed;
      setDownloadStates((prev) => ({
        ...prev,
        [modelName]: { status: "error", error: errorMsg },
      }));
      showToast("error", errorMsg);
    }
  };

  if (loading) {
    return (
      <div className="bg-white dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-700 rounded-lg p-4">
        <div className="text-sm text-zinc-500">{t.loading}</div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-700 rounded-lg p-4">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-white flex items-center gap-2">
          <FaDownload className="text-blue-500" />
          {t.model_download_title}
        </h2>
      </div>

      {/* Email Section */}
      <div className="mb-4 p-3 bg-zinc-50 dark:bg-zinc-800 rounded-lg">
        {email && !showEmailForm ? (
          <div className="flex items-center justify-between">
            <div className="text-sm text-zinc-700 dark:text-zinc-300">
              <span className="text-zinc-500">{t.model_download_email}:</span>{" "}
              {email}
            </div>
            <button
              onClick={() => setShowEmailForm(true)}
              className="text-xs text-blue-500 hover:text-blue-600"
            >
              {t.edit}
            </button>
          </div>
        ) : (
          <div className="space-y-2">
            <p className="text-xs text-zinc-500 dark:text-zinc-400">
              {t.model_download_email_hint}
            </p>
            <div className="flex gap-2">
              <input
                type="email"
                value={emailInput}
                onChange={(e) => setEmailInput(e.target.value)}
                placeholder={t.model_download_email_placeholder}
                className="flex-1 px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded bg-white dark:bg-zinc-950 text-zinc-900 dark:text-white text-sm focus:outline-none focus:border-blue-500"
              />
              <button
                onClick={handleSaveEmail}
                disabled={savingEmail || !emailInput.trim()}
                className="px-4 py-2 bg-blue-500 text-white text-sm rounded hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {savingEmail ? <FaSpinner className="animate-spin" /> : t.save}
              </button>
              {showEmailForm && email && (
                <button
                  onClick={() => {
                    setShowEmailForm(false);
                    setEmailInput(email);
                  }}
                  className="px-4 py-2 bg-zinc-200 dark:bg-zinc-700 text-zinc-700 dark:text-zinc-300 text-sm rounded hover:bg-zinc-300 dark:hover:bg-zinc-600"
                >
                  {t.cancel}
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Models List */}
      <div className="space-y-2">
        {models.map((model) => {
          const state = downloadStates[model.name];
          const isDownloading = state?.status === "downloading";
          const hasError = state?.status === "error";

          return (
            <div
              key={model.name}
              className="flex items-center justify-between p-3 bg-zinc-50 dark:bg-zinc-800 rounded-lg"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm text-zinc-900 dark:text-white">
                    {model.name}
                  </span>
                  <span className="text-xs text-zinc-500">{model.size}</span>
                  {model.downloaded && (
                    <span className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                      <FaCheck className="text-xs" />
                    </span>
                  )}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
                  {model.description}
                </p>
                {isDownloading && (
                  <p className="text-xs text-blue-500 mt-1">
                    {t.downloading}...
                  </p>
                )}
                {hasError && state?.error && (
                  <p className="text-xs text-red-500 mt-1 flex items-center gap-1">
                    <FaCircleExclamation />
                    {state.error}
                  </p>
                )}
              </div>
              <button
                onClick={() => handleDownload(model.name)}
                disabled={!isConnected || isDownloading || model.downloaded}
                className={`ml-3 px-3 py-1.5 text-xs rounded flex items-center gap-1.5 ${
                  model.downloaded
                    ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300 cursor-default"
                    : isDownloading
                      ? "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300"
                      : "bg-blue-500 text-white hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed"
                }`}
              >
                {model.downloaded ? (
                  <>
                    <FaCheck />
                    {t.model_download_downloaded}
                  </>
                ) : isDownloading ? (
                  <>
                    <FaSpinner className="animate-spin" />
                    {t.downloading}
                  </>
                ) : (
                  <>
                    <FaDownload />
                    {t.download}
                  </>
                )}
              </button>
            </div>
          );
        })}
      </div>

      {/* Info */}
      <div className="mt-4 text-xs text-zinc-400 dark:text-zinc-500">
        {t.model_download_info}
      </div>
    </div>
  );
}
