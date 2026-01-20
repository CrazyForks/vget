import { useState, useEffect } from "react";
import { invoke } from "@tauri-apps/api/core";
import { cn } from "@/lib/utils";
import { Combine, Image, Trash2, Droplets } from "lucide-react";
import { PdfToolId, Config } from "./types";
import { MergePdfPanel, ImagesToPdfPanel, DeletePagesPanel, RemoveWatermarkPanel } from "./panels";

interface Tool {
  id: PdfToolId;
  title: string;
  description: string;
  icon: React.ReactNode;
}

const tools: Tool[] = [
  {
    id: "merge",
    title: "Merge PDFs",
    description: "Combine multiple PDF files into one",
    icon: <Combine className="h-4 w-4" />,
  },
  {
    id: "images-to-pdf",
    title: "Images to PDF",
    description: "Convert images to a single PDF document",
    icon: <Image className="h-4 w-4" />,
  },
  {
    id: "delete-pages",
    title: "Delete Pages",
    description: "Remove specific pages from a PDF",
    icon: <Trash2 className="h-4 w-4" />,
  },
  {
    id: "remove-watermark",
    title: "Remove Watermark",
    description: "Try to remove watermarks from a PDF",
    icon: <Droplets className="h-4 w-4" />,
  },
];

export function PDFToolsPage() {
  const [activeTool, setActiveTool] = useState<PdfToolId>("merge");
  const [loading, setLoading] = useState(false);
  const [config, setConfig] = useState<Config | null>(null);

  useEffect(() => {
    invoke<Config>("get_config")
      .then(setConfig)
      .catch(console.error);
  }, []);

  const handleToolChange = (toolId: PdfToolId) => {
    if (!loading) {
      setActiveTool(toolId);
    }
  };

  const panelProps = {
    outputDir: config?.output_dir || "",
    loading,
    setLoading,
  };

  const activeToolData = tools.find((t) => t.id === activeTool);

  const renderPanel = () => {
    switch (activeTool) {
      case "merge":
        return <MergePdfPanel {...panelProps} />;
      case "images-to-pdf":
        return <ImagesToPdfPanel {...panelProps} />;
      case "delete-pages":
        return <DeletePagesPanel {...panelProps} />;
      case "remove-watermark":
        return <RemoveWatermarkPanel {...panelProps} />;
      default:
        return null;
    }
  };

  return (
    <div className="h-full flex flex-col">
      <header className="h-14 border-b border-border flex items-center px-6 shrink-0">
        <h1 className="text-xl font-semibold">PDF Tools</h1>
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
                <span
                  className={cn(
                    "shrink-0",
                    activeTool === tool.id ? "text-primary" : ""
                  )}
                >
                  {tool.icon}
                </span>
                <span className="text-sm font-medium truncate">{tool.title}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Right pane - Tool content */}
        <div className="flex-1 p-6 overflow-y-auto">
          {activeToolData && (
            <div className="max-w-lg">
              <div className="mb-6">
                <h2 className="text-lg font-semibold">{activeToolData.title}</h2>
                <p className="text-sm text-muted-foreground mt-1">
                  {activeToolData.description}
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
