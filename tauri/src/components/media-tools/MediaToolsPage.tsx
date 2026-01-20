import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/plugin-dialog";
import { cn } from "@/lib/utils";
import {
  FileVideo,
  FileAudio,
  Scissors,
  Minimize2,
  Image,
  FileType,
} from "lucide-react";
import { toast } from "sonner";
import { MediaInfo, ToolId, Config } from "./types";
import {
  ConvertPanel,
  CompressPanel,
  TrimPanel,
  ExtractAudioPanel,
  ExtractFramesPanel,
  AudioConvertPanel,
} from "./panels";

interface Tool {
  id: ToolId;
  title: string;
  description: string;
  icon: React.ReactNode;
}

const tools: Tool[] = [
  {
    id: "convert",
    title: "Convert Video",
    description: "Convert between video formats (MP4, WebM, MKV, MOV)",
    icon: <FileVideo className="h-4 w-4" />,
  },
  {
    id: "compress",
    title: "Compress Video",
    description: "Reduce file size while maintaining quality",
    icon: <Minimize2 className="h-4 w-4" />,
  },
  {
    id: "trim",
    title: "Trim Video",
    description: "Cut clips from videos with start and end times",
    icon: <Scissors className="h-4 w-4" />,
  },
  {
    id: "extract-audio",
    title: "Extract Audio",
    description: "Extract audio track from video files",
    icon: <FileAudio className="h-4 w-4" />,
  },
  {
    id: "extract-frames",
    title: "Extract Frames",
    description: "Extract images or thumbnails from video",
    icon: <Image className="h-4 w-4" />,
  },
  {
    id: "audio-convert",
    title: "Convert Audio",
    description: "Convert between audio formats (MP3, AAC, FLAC, WAV)",
    icon: <FileType className="h-4 w-4" />,
  },
];

export function MediaToolsPage() {
  const [activeTool, setActiveTool] = useState<ToolId>("convert");
  const [inputFile, setInputFile] = useState("");
  const [loading, setLoading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [mediaInfo, setMediaInfo] = useState<MediaInfo | null>(null);
  const [jobId, setJobId] = useState<string | null>(null);
  const [config, setConfig] = useState<Config | null>(null);

  useEffect(() => {
    invoke<Config>("get_config")
      .then(setConfig)
      .catch(console.error);
  }, []);

  useEffect(() => {
    const unlistenProgress = listen<{ jobId: string; progress: number }>(
      "ffmpeg-progress",
      (event) => {
        if (event.payload.jobId === jobId && mediaInfo?.duration) {
          const percent = Math.min(
            100,
            (event.payload.progress / mediaInfo.duration) * 100
          );
          setProgress(percent);
        }
      }
    );

    const unlistenComplete = listen<{ jobId: string; outputPath: string }>(
      "ffmpeg-complete",
      (event) => {
        if (event.payload.jobId === jobId) {
          setLoading(false);
          setProgress(100);
          toast.success("Operation completed successfully!");
          setTimeout(() => {
            resetState();
          }, 1500);
        }
      }
    );

    const unlistenError = listen<{ jobId: string; error: string }>(
      "ffmpeg-error",
      (event) => {
        if (event.payload.jobId === jobId) {
          setLoading(false);
          toast.error(event.payload.error);
        }
      }
    );

    return () => {
      unlistenProgress.then((fn) => fn());
      unlistenComplete.then((fn) => fn());
      unlistenError.then((fn) => fn());
    };
  }, [jobId, mediaInfo]);

  const resetState = () => {
    setInputFile("");
    setMediaInfo(null);
    setProgress(0);
    setJobId(null);
  };

  const selectInputFile = async () => {
    const file = await open({
      multiple: false,
      filters: [
        { name: "Media", extensions: ["mp4", "mkv", "webm", "mov", "avi", "mp3", "aac", "flac", "wav", "ogg"] },
      ],
    });
    if (file) {
      setInputFile(file);
      try {
        const info = await invoke<MediaInfo>("ffmpeg_get_media_info", {
          inputPath: file,
        });
        setMediaInfo(info);
      } catch (e) {
        console.error("Failed to get media info:", e);
      }
    }
  };

  const handleToolChange = (toolId: ToolId) => {
    if (!loading) {
      setActiveTool(toolId);
      resetState();
    }
  };

  const panelProps = {
    inputFile,
    outputDir: config?.output_dir || "",
    loading,
    progress,
    mediaInfo,
    onSelectInput: selectInputFile,
    setLoading,
    setProgress,
    setJobId,
  };

  const activeTolData = tools.find((t) => t.id === activeTool);

  const renderPanel = () => {
    switch (activeTool) {
      case "convert":
        return <ConvertPanel {...panelProps} />;
      case "compress":
        return <CompressPanel {...panelProps} />;
      case "trim":
        return <TrimPanel {...panelProps} />;
      case "extract-audio":
        return <ExtractAudioPanel {...panelProps} />;
      case "extract-frames":
        return <ExtractFramesPanel {...panelProps} />;
      case "audio-convert":
        return <AudioConvertPanel {...panelProps} />;
      default:
        return null;
    }
  };

  return (
    <div className="h-full flex flex-col">
      <header className="h-14 border-b border-border flex items-center px-6 shrink-0">
        <h1 className="text-xl font-semibold">Media Tools</h1>
      </header>

      <div className="flex-1 flex min-h-0">
        {/* Left pane - Tool list */}
        <div className="w-56 border-r border-border p-2 overflow-y-auto shrink-0">
          <div className="space-y-1">
            {tools.map((tool) => (
              <button
                key={tool.id}
                onClick={() => handleToolChange(tool.id)}
                disabled={loading}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2 rounded-md text-left transition-colors",
                  "hover:bg-accent disabled:opacity-50 disabled:cursor-not-allowed",
                  activeTool === tool.id
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                <span className={cn(
                  "shrink-0",
                  activeTool === tool.id ? "text-primary" : ""
                )}>
                  {tool.icon}
                </span>
                <span className="text-sm font-medium truncate">{tool.title}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Right pane - Tool content */}
        <div className="flex-1 p-6 overflow-y-auto">
          {activeTolData && (
            <div className="max-w-lg">
              <div className="mb-6">
                <h2 className="text-lg font-semibold">{activeTolData.title}</h2>
                <p className="text-sm text-muted-foreground mt-1">
                  {activeTolData.description}
                </p>
              </div>
              {renderPanel()}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
