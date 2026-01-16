import { useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { open } from "@tauri-apps/plugin-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FolderOpen, Printer, ExternalLink, Loader2, FileText, User, Hash } from "lucide-react";
import { toast } from "sonner";
import { PdfPanelProps, PdfInfo } from "../types";

export function PdfInfoPanel({ loading, setLoading }: PdfPanelProps) {
  const [inputFile, setInputFile] = useState("");
  const [pdfInfo, setPdfInfo] = useState<PdfInfo | null>(null);
  const [printing, setPrinting] = useState(false);

  const selectFile = async () => {
    const selected = await open({
      multiple: false,
      filters: [{ name: "PDF", extensions: ["pdf"] }],
    });
    if (selected) {
      setInputFile(selected);
      setLoading(true);
      try {
        const info = await invoke<PdfInfo>("pdf_get_info", { inputPath: selected });
        setPdfInfo(info);
      } catch (e) {
        toast.error(String(e));
        setPdfInfo(null);
      } finally {
        setLoading(false);
      }
    }
  };

  const handlePrint = async () => {
    if (!inputFile) return;
    setPrinting(true);
    try {
      await invoke("pdf_print", { inputPath: inputFile });
      toast.success("Print job sent to printer");
    } catch (e) {
      toast.error(String(e));
    } finally {
      setPrinting(false);
    }
  };

  const handleOpenExternal = async () => {
    if (!inputFile) return;
    try {
      await invoke("pdf_open_external", { inputPath: inputFile });
    } catch (e) {
      toast.error(String(e));
    }
  };

  const getFileName = (path: string) => path.split(/[/\\]/).pop() || path;

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>Select PDF File</Label>
        <div className="flex gap-2">
          <Input
            value={inputFile}
            readOnly
            placeholder="Select a PDF to view info..."
            className="min-w-0 flex-1"
          />
          <Button variant="outline" onClick={selectFile} className="shrink-0">
            <FolderOpen className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {loading && (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      )}

      {pdfInfo && !loading && (
        <>
          <div className="p-4 bg-muted rounded-lg space-y-3">
            <div className="flex items-start gap-3">
              <FileText className="h-5 w-5 text-muted-foreground mt-0.5" />
              <div className="flex-1 min-w-0">
                <p className="text-xs text-muted-foreground">Filename</p>
                <p className="text-sm font-medium truncate" title={inputFile}>
                  {getFileName(inputFile)}
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <Hash className="h-5 w-5 text-muted-foreground mt-0.5" />
              <div>
                <p className="text-xs text-muted-foreground">Pages</p>
                <p className="text-sm font-medium">{pdfInfo.pages}</p>
              </div>
            </div>

            {pdfInfo.title && (
              <div className="flex items-start gap-3">
                <FileText className="h-5 w-5 text-muted-foreground mt-0.5" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-muted-foreground">Title</p>
                  <p className="text-sm font-medium truncate" title={pdfInfo.title}>
                    {pdfInfo.title}
                  </p>
                </div>
              </div>
            )}

            {pdfInfo.author && (
              <div className="flex items-start gap-3">
                <User className="h-5 w-5 text-muted-foreground mt-0.5" />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-muted-foreground">Author</p>
                  <p className="text-sm font-medium truncate" title={pdfInfo.author}>
                    {pdfInfo.author}
                  </p>
                </div>
              </div>
            )}
          </div>

          <div className="flex gap-2 pt-2">
            <Button onClick={handlePrint} disabled={printing}>
              {printing ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <Printer className="h-4 w-4 mr-2" />
              )}
              Print
            </Button>
            <Button variant="outline" onClick={handleOpenExternal}>
              <ExternalLink className="h-4 w-4 mr-2" />
              Open in Viewer
            </Button>
          </div>
        </>
      )}

      {!pdfInfo && !loading && inputFile && (
        <p className="text-sm text-muted-foreground">Failed to load PDF info</p>
      )}
    </div>
  );
}
