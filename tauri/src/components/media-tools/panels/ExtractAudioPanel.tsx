import { useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Progress } from "@/components/ui/progress";
import { FolderOpen, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { PanelProps, generateOutputPath } from "../types";

export function ExtractAudioPanel({
  inputFile,
  outputDir,
  loading,
  progress,
  onSelectInput,
  setLoading,
  setProgress,
  setJobId,
}: PanelProps) {
  const [audioFormat, setAudioFormat] = useState("mp3");

  const outputPath = inputFile ? generateOutputPath(outputDir, inputFile, audioFormat) : "";

  const handleExtractAudio = async () => {
    if (!inputFile || !outputDir) return;
    setLoading(true);
    setProgress(0);
    try {
      const id = await invoke<string>("ffmpeg_extract_audio", {
        inputPath: inputFile,
        outputPath,
        format: audioFormat,
      });
      setJobId(id);
    } catch (e) {
      setLoading(false);
      toast.error(String(e));
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label>Input Video</Label>
        <div className="flex gap-2">
          <Input value={inputFile} readOnly placeholder="Select a video..." className="min-w-0 flex-1" />
          <Button variant="outline" onClick={onSelectInput} className="shrink-0">
            <FolderOpen className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <div className="space-y-2">
        <Label>Audio Format</Label>
        <Select value={audioFormat} onValueChange={setAudioFormat}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="mp3">MP3</SelectItem>
            <SelectItem value="aac">AAC</SelectItem>
            <SelectItem value="flac">FLAC</SelectItem>
            <SelectItem value="wav">WAV</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {inputFile && (
        <div className="space-y-2">
          <Label className="text-muted-foreground">Output</Label>
          <p className="text-sm text-muted-foreground break-all" title={outputPath}>{outputPath}</p>
        </div>
      )}
      {loading && <Progress value={progress} />}
      <div className="pt-2">
        <Button onClick={handleExtractAudio} disabled={!inputFile || !outputDir || loading}>
          {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
          Extract
        </Button>
      </div>
    </div>
  );
}
