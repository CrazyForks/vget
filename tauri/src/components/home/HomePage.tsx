import { useEffect, useState, useRef } from "react";
import { invoke } from "@tauri-apps/api/core";
import { open } from "@tauri-apps/plugin-dialog";
import { Download, Folder, Link, Loader2, Upload, FileText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  useDownloadsStore,
  setupDownloadListeners,
  startDownload,
} from "@/stores/downloads";
import { MediaInfo, Config } from "./types";
import { DownloadItem } from "./DownloadItem";
import { cn } from "@/lib/utils";
import { useDropZone } from "@/hooks/useDropZone";
import {
  isYouTubeUrl,
  startDockerDownload,
  getDockerJobStatus,
  getDockerServerUrl,
} from "@/services/dockerApi";

export function HomePage() {
  const [url, setUrl] = useState("");
  const [isExtracting, setIsExtracting] = useState(false);
  const [config, setConfig] = useState<Config | null>(null);
  const [bulkProgress, setBulkProgress] = useState<{ current: number; total: number } | null>(null);
  const downloads = useDownloadsStore((state) => state.downloads);
  const clearCompleted = useDownloadsStore((state) => state.clearCompleted);
  const addDownload = useDownloadsStore((state) => state.addDownload);
  const updateDownload = useDownloadsStore((state) => state.updateDownload);
  const { t } = useTranslation();
  const dockerPollingRef = useRef<Map<string, NodeJS.Timeout>>(new Map());

  // Handle bulk file import
  const handleFileImport = async (filePath: string) => {
    try {
      const content = await invoke<string>("read_text_file", { path: filePath });
      const urls = content
        .split("\n")
        .map((line) => line.trim())
        .filter((line) => line && (line.startsWith("http://") || line.startsWith("https://")));

      if (urls.length === 0) {
        toast.error(t("home.noUrlsInFile") || "No valid URLs found in file");
        return;
      }

      toast.success(t("home.foundUrls", { count: urls.length }) || `Found ${urls.length} URLs`);

      // Process URLs one by one
      setBulkProgress({ current: 0, total: urls.length });
      for (let i = 0; i < urls.length; i++) {
        setBulkProgress({ current: i + 1, total: urls.length });
        await processUrl(urls[i]);
      }
      setBulkProgress(null);
    } catch (err) {
      console.error("Failed to read file:", err);
      toast.error(t("home.failedToReadFile") || "Failed to read file");
    }
  };

  // Drop zone for bulk download (.txt files)
  const { ref: dropZoneRef, isDragging } = useDropZone<HTMLDivElement>({
    accept: ["txt"],
    onDrop: (paths) => {
      handleFileImport(paths[0]);
    },
    onInvalidDrop: (_paths, ext) => {
      if (ext === "md" || ext === "markdown") {
        toast.error(t("home.dropMdHint") || "For Markdown files, go to PDF Tools → Markdown to PDF");
      } else {
        toast.error(t("home.dropTxtFile") || "Please drop a .txt file containing URLs");
      }
    },
  });

  useEffect(() => {
    setupDownloadListeners();

    invoke<Config>("get_config")
      .then(setConfig)
      .catch(console.error);

    // Cleanup polling intervals on unmount
    return () => {
      dockerPollingRef.current.forEach((interval) => clearInterval(interval));
      dockerPollingRef.current.clear();
    };
  }, []);

  const handleSelectFile = async () => {
    const selected = await open({
      multiple: false,
      filters: [{ name: "Text", extensions: ["txt"] }],
    });
    if (selected && typeof selected === "string") {
      await handleFileImport(selected);
    }
  };

  // Handle YouTube download via Docker server
  const handleYouTubeDownload = async (inputUrl: string): Promise<boolean> => {
    try {
      // Try to start download on Docker server directly
      const response = await startDockerDownload(inputUrl);
      const jobId = response.data.id;

      // Add to local downloads list with docker prefix to distinguish
      const localId = `docker-${jobId}`;
      addDownload({
        id: localId,
        url: inputUrl,
        title: `YouTube: ${inputUrl}`,
        outputPath: "Docker Server",
        status: "pending",
        progress: null,
        error: null,
      });

      // Poll for status updates
      const pollInterval = setInterval(async () => {
        try {
          const status = await getDockerJobStatus(jobId);
          const job = status.data;

          if (job.status === "downloading") {
            updateDownload(localId, {
              status: "downloading",
              title: job.filename || `YouTube: ${inputUrl}`,
              progress: {
                job_id: localId,
                downloaded: job.downloaded || 0,
                total: job.total || null,
                speed: 0,
                percent: job.progress || 0,
              },
            });
          } else if (job.status === "completed") {
            updateDownload(localId, {
              status: "completed",
              title: job.filename || `YouTube: ${inputUrl}`,
              outputPath: job.filename || "Docker Server",
              progress: null,
            });
            clearInterval(pollInterval);
            dockerPollingRef.current.delete(localId);
            toast.success(t("home.downloadComplete") || "Download complete!");
          } else if (job.status === "failed") {
            updateDownload(localId, {
              status: "failed",
              error: job.error || "Download failed",
              progress: null,
            });
            clearInterval(pollInterval);
            dockerPollingRef.current.delete(localId);
          } else if (job.status === "cancelled") {
            updateDownload(localId, {
              status: "cancelled",
              progress: null,
            });
            clearInterval(pollInterval);
            dockerPollingRef.current.delete(localId);
          }
        } catch (pollError) {
          console.error("Polling error:", pollError);
          // Don't stop polling on transient errors
        }
      }, 1000);

      dockerPollingRef.current.set(localId, pollInterval);

      toast.success(
        t("home.youtubeDownloadStarted") ||
          "YouTube download started via Docker server"
      );
      return true;
    } catch (err) {
      console.error("Docker download error:", err);

      const errorMessage = err instanceof Error ? err.message : String(err);

      // Check if it's an authentication error
      if (errorMessage.includes("Authentication required") || errorMessage.includes("401")) {
        toast.error(
          t("home.dockerAuthRequired") ||
            "Docker server requires authentication. Go to Settings → Sites → Docker Server to add your JWT token.",
          { duration: 8000 }
        );
        return false;
      }

      // Check if server is not reachable (network error)
      if (errorMessage.includes("Failed to fetch") || errorMessage.includes("NetworkError") || errorMessage.includes("fetch")) {
        const serverUrl = getDockerServerUrl();
        toast.error(
          t("home.dockerNotRunning") ||
            `YouTube downloads require vget-server. Please run Docker container or start the server at ${serverUrl}`,
          { duration: 8000 }
        );
        return false;
      }

      // Other errors
      toast.error(errorMessage);
      return false;
    }
  };

  const processUrl = async (inputUrl: string) => {
    if (!inputUrl.trim() || !config) return;

    // Check if it's a YouTube URL
    if (isYouTubeUrl(inputUrl)) {
      await handleYouTubeDownload(inputUrl);
      return;
    }

    try {
      const mediaInfo = await invoke<MediaInfo>("extract_media", { url: inputUrl });

      if (mediaInfo.formats.length === 0) {
        console.warn(`No formats found for: ${inputUrl}`);
        return;
      }

      const format = mediaInfo.formats[0];
      const ext = format.ext || "mp4";
      const sanitizedTitle = mediaInfo.title
        .replace(/[/\\?%*:|"<>]/g, "-")
        .substring(0, 100);
      const outputPath = `${config.output_dir}/${sanitizedTitle}.${ext}`;

      await startDownload(
        format.url,
        mediaInfo.title,
        outputPath,
        format.headers,
        format.audio_url || undefined
      );
    } catch (err) {
      console.error(`Failed to process URL ${inputUrl}:`, err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim() || !config) return;

    setIsExtracting(true);
    try {
      // Check if it's a YouTube URL - handle via Docker
      if (isYouTubeUrl(url)) {
        const success = await handleYouTubeDownload(url);
        if (success) {
          setUrl("");
        }
        return;
      }

      const mediaInfo = await invoke<MediaInfo>("extract_media", { url });

      if (mediaInfo.formats.length === 0) {
        toast.error(t("home.noFormats"));
        return;
      }

      const format = mediaInfo.formats[0];
      const ext = format.ext || "mp4";
      const sanitizedTitle = mediaInfo.title
        .replace(/[/\\?%*:|"<>]/g, "-")
        .substring(0, 100);
      const outputPath = `${config.output_dir}/${sanitizedTitle}.${ext}`;

      await startDownload(
        format.url,
        mediaInfo.title,
        outputPath,
        format.headers,
        format.audio_url || undefined
      );
      setUrl("");
      toast.success(t("home.downloadStarted"));
    } catch (err) {
      console.error("Extraction failed:", err);
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setIsExtracting(false);
    }
  };

  const handleOpenFolder = async () => {
    if (config?.output_dir) {
      try {
        await invoke("open_output_folder", { path: config.output_dir });
      } catch (err) {
        toast.error(t("home.failedToOpenFolder"));
        console.error(err);
      }
    }
  };

  const activeDownloads = downloads.filter(
    (d) => d.status === "downloading" || d.status === "pending"
  );
  const completedDownloads = downloads.filter(
    (d) =>
      d.status === "completed" ||
      d.status === "failed" ||
      d.status === "cancelled"
  );

  return (
    <div className="h-full">
      <header className="h-14 border-b border-border flex items-center px-6">
        <h1 className="text-xl font-semibold">{t("home.title")}</h1>
        {bulkProgress && (
          <span className="ml-4 text-sm text-muted-foreground">
            {t("home.processingBulk", { current: bulkProgress.current, total: bulkProgress.total }) ||
              `Processing ${bulkProgress.current}/${bulkProgress.total}`}
          </span>
        )}
      </header>

      <div className="p-6">
        {/* Single URL input */}
        <form onSubmit={handleSubmit} className="max-w-2xl">
          <div className="relative">
            <Link className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
            <input
              type="text"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder={t("home.urlPlaceholder")}
              className="w-full pl-12 pr-32 py-4 rounded-xl border border-input bg-background text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
            <button
              type="submit"
              disabled={isExtracting || !url.trim()}
              className="absolute right-2 top-1/2 -translate-y-1/2 px-4 py-2 rounded-lg bg-primary text-primary-foreground font-medium disabled:opacity-50 disabled:cursor-not-allowed hover:opacity-90 transition-opacity flex items-center gap-2"
            >
              {isExtracting && <Loader2 className="h-4 w-4 animate-spin" />}
              {isExtracting ? t("home.extracting") : t("home.download")}
            </button>
          </div>
        </form>

        <div className="mt-3 max-w-2xl">
          <p className="text-sm text-muted-foreground">
            {t("home.supportsHint")}
          </p>
        </div>

        {/* Bulk download drop zone */}
        <div className="mt-6 max-w-2xl">
          <div
            ref={dropZoneRef}
            onClick={handleSelectFile}
            className={cn(
              "border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-all",
              isDragging
                ? "border-primary bg-primary/5"
                : "border-muted-foreground/25 hover:border-muted-foreground/50 hover:bg-muted/30"
            )}
          >
            <div className="flex items-center justify-center gap-3">
              {isDragging ? (
                <Upload className="h-8 w-8 text-primary" />
              ) : (
                <FileText className="h-8 w-8 text-muted-foreground" />
              )}
              <div className="text-left">
                <p className={cn(
                  "font-medium",
                  isDragging ? "text-primary" : "text-foreground"
                )}>
                  {t("home.bulkDownloadTitle") || "Bulk Download"}
                </p>
                <p className="text-sm text-muted-foreground">
                  {t("home.bulkDownloadHint") || "Drop a .txt file with URLs or click to select"}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Downloads list */}
        <div className="mt-8 max-w-2xl">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium">{t("home.downloads")}</h2>
            <div className="flex items-center gap-2">
              {completedDownloads.length > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={clearCompleted}
                  className="text-muted-foreground"
                >
                  {t("home.clearCompleted")}
                </Button>
              )}
              <button
                onClick={handleOpenFolder}
                className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                <Folder className="h-4 w-4" />
                {t("home.openFolder")}
              </button>
            </div>
          </div>

          {downloads.length === 0 ? (
            <div className="border border-dashed border-border rounded-xl p-12 text-center">
              <Download className="h-12 w-12 text-muted-foreground/50 mx-auto mb-4" />
              <p className="text-muted-foreground">
                {t("home.noDownloadsYet")}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {activeDownloads.map((download) => (
                <DownloadItem key={download.id} download={download} />
              ))}
              {completedDownloads.map((download) => (
                <DownloadItem key={download.id} download={download} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
