package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guiyumin/vget/internal/core/ai"
	aioutput "github.com/guiyumin/vget/internal/core/ai/output"
	"github.com/guiyumin/vget/internal/core/ai/transcriber"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/spf13/cobra"
)

var (
	aiModel    string
	aiLanguage string
	aiFrom     string
	aiRemote   bool
	aiOutput   string
	aiToFormat string
)

// aiCmd is the parent command for all AI features
var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered transcription and more",
	Long: `AI features for vget including speech-to-text transcription.

Models are downloaded on first use and stored in ~/.config/vget/models/

Examples:
  vget ai transcribe audio.mp3 --language zh
  vget ai models
  vget ai models download whisper-large-v3-turbo
  vget ai convert transcript.md --to srt`,
}

// aiTranscribeCmd transcribes audio/video files
var aiTranscribeCmd = &cobra.Command{
	Use:   "transcribe <file>",
	Short: "Transcribe audio/video to markdown",
	Long: `Transcribe audio or video files to markdown with timestamps.

The transcript is saved as <filename>.transcript.md

Language is required. Common language codes:
  zh - Chinese    en - English    ja - Japanese
  ko - Korean     es - Spanish    fr - French
  de - German     ru - Russian    pt - Portuguese

Examples:
  vget ai transcribe podcast.mp3 --language zh
  vget ai transcribe video.mp4 --language en
  vget ai transcribe audio.m4a --language ja --model whisper-small
  vget ai transcribe podcast.mp3 --language zh -o my-transcript.md`,
	Args: cobra.ExactArgs(1),
	Run:  runTranscribe,
}

// aiConvertCmd converts transcript to other formats
var aiConvertCmd = &cobra.Command{
	Use:   "convert <transcript.md>",
	Short: "Convert transcript to SRT/VTT/TXT",
	Long: `Convert a markdown transcript to subtitle or text formats.

Supported output formats:
  srt - SubRip subtitle format
  vtt - WebVTT subtitle format
  txt - Plain text (no timestamps)

Examples:
  vget ai convert podcast.transcript.md --to srt
  vget ai convert podcast.transcript.md --to vtt
  vget ai convert podcast.transcript.md --to txt -o subtitles.txt`,
	Args: cobra.ExactArgs(1),
	Run:  runConvert,
}

// aiModelsCmd is the parent command for model management
var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List and manage transcription models",
	Long: `List downloaded models or available models from remote.

By default, shows locally downloaded models.
Use -r/--remote to show models available for download.

Examples:
  vget ai models              # List downloaded models
  vget ai models -r           # List available models from remote
  vget ai models download whisper-large-v3-turbo
  vget ai models rm whisper-small`,
	Run: runModels,
}

// aiModelsDownloadCmd downloads a model
var aiModelsDownloadCmd = &cobra.Command{
	Use:   "download <model>",
	Short: "Download a transcription model",
	Long: `Download a Whisper model for local transcription.

Available models:
  whisper-tiny            (78MB)  - Fastest, basic quality
  whisper-base           (148MB)  - Good for quick drafts
  whisper-small          (488MB)  - Balanced for most uses
  whisper-medium         (1.5GB)  - Higher accuracy
  whisper-large-v3       (3.1GB)  - Highest accuracy, slowest
  whisper-large-v3-turbo (1.6GB)  - Best quality + fast (recommended)

Download sources:
  huggingface (default) - Official Hugging Face
  vmirror               - vmirror.org (faster in China)

Examples:
  vget ai models download whisper-large-v3-turbo
  vget ai models download whisper-small --from=vmirror`,
	Args: cobra.ExactArgs(1),
	Run:  runModelsDownload,
}

// aiModelsRmCmd removes a downloaded model
var aiModelsRmCmd = &cobra.Command{
	Use:   "rm <model>",
	Short: "Remove a downloaded model",
	Long: `Remove a downloaded model to free up disk space.

Examples:
  vget ai models rm whisper-small
  vget ai models rm whisper-medium`,
	Args: cobra.ExactArgs(1),
	Run:  runModelsRm,
}

// aiDownloadCmd is an alias for models download
var aiDownloadCmd = &cobra.Command{
	Use:   "download <model>",
	Short: "Download a transcription model (alias for 'models download')",
	Long: `Download a Whisper model for local transcription.

This is an alias for 'vget ai models download'.

Examples:
  vget ai download whisper-large-v3-turbo
  vget ai download whisper-small --from=vmirror`,
	Args: cobra.ExactArgs(1),
	Run:  runModelsDownload,
}

func runTranscribe(cmd *cobra.Command, args []string) {
	filePath := args[0]

	// Validate language is provided
	if aiLanguage == "" {
		fmt.Fprintf(os.Stderr, "Error: --language is required\n\n")
		fmt.Fprintln(os.Stderr, "Common language codes:")
		fmt.Fprintln(os.Stderr, "  zh - Chinese    en - English    ja - Japanese")
		fmt.Fprintln(os.Stderr, "  ko - Korean     es - Spanish    fr - French")
		fmt.Fprintln(os.Stderr, "  de - German     ru - Russian    pt - Portuguese")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintf(os.Stderr, "  vget ai transcribe %s --language zh\n", filePath)
		os.Exit(1)
	}

	// Validate file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", filePath)
		os.Exit(1)
	}

	// Get models directory
	modelsDir, err := transcriber.DefaultModelsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Determine model to use
	modelName := aiModel
	if modelName == "" {
		modelName = transcriber.DefaultModel
	}

	// Check if model is downloaded
	mm := transcriber.NewModelManager(modelsDir)
	if !mm.IsModelDownloaded(modelName) {
		fmt.Printf("Model not found. Download it with:\n")
		fmt.Printf("  vget ai models download %s\n\n", modelName)
		fmt.Println("Available models:")
		for _, m := range transcriber.ASRModels {
			fmt.Printf("  %-24s (%s) - %s\n", m.Name, m.Size, m.Description)
		}
		os.Exit(0)
	}

	// Create local ASR config
	localCfg := config.LocalASRConfig{
		Engine:    "whisper",
		Model:     modelName,
		ModelsDir: modelsDir,
		Language:  aiLanguage,
	}

	// Create pipeline with local transcription (no summarization)
	pipeline, err := ai.NewLocalPipeline(localCfg, nil, "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run transcription
	ctx := context.Background()
	opts := ai.Options{
		Transcribe: true,
		Summarize:  false,
	}

	result, err := pipeline.Process(ctx, filePath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Handle custom output path
	outputPath := result.TranscriptPath
	if aiOutput != "" {
		// Copy to custom output path
		data, err := os.ReadFile(result.TranscriptPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading transcript: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(aiOutput, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", aiOutput, err)
			os.Exit(1)
		}
		outputPath = aiOutput
	}

	fmt.Printf("\nTranscript saved: %s\n", outputPath)
}

func runConvert(cmd *cobra.Command, args []string) {
	inputPath := args[0]

	// Validate input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", inputPath)
		os.Exit(1)
	}

	// Validate --to format
	if aiToFormat == "" {
		fmt.Fprintf(os.Stderr, "Error: --to is required\n\n")
		fmt.Fprintln(os.Stderr, "Supported formats:")
		fmt.Fprintln(os.Stderr, "  srt - SubRip subtitle format")
		fmt.Fprintln(os.Stderr, "  vtt - WebVTT subtitle format")
		fmt.Fprintln(os.Stderr, "  txt - Plain text (no timestamps)")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintf(os.Stderr, "  vget ai convert %s --to srt\n", inputPath)
		os.Exit(1)
	}

	// Validate format
	format := strings.ToLower(aiToFormat)
	if format != "srt" && format != "vtt" && format != "txt" {
		fmt.Fprintf(os.Stderr, "Error: unsupported format '%s'\n\n", aiToFormat)
		fmt.Fprintln(os.Stderr, "Supported formats: srt, vtt, txt")
		os.Exit(1)
	}

	// Read input transcript
	content, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse transcript
	segments, err := aioutput.ParseTranscript(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing transcript: %v\n", err)
		os.Exit(1)
	}

	// Determine output path
	outputPath := aiOutput
	if outputPath == "" {
		// Generate from input path
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		// Remove .transcript suffix if present
		base = strings.TrimSuffix(base, ".transcript")
		outputPath = base + "." + format
	}

	// Convert and write
	var outputContent string
	switch format {
	case "srt":
		outputContent = aioutput.ToSRT(segments)
	case "vtt":
		outputContent = aioutput.ToVTT(segments)
	case "txt":
		outputContent = aioutput.ToTXT(segments)
	}

	if err := os.WriteFile(outputPath, []byte(outputContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Converted to %s: %s\n", strings.ToUpper(format), outputPath)
}

func runModels(cmd *cobra.Command, args []string) {
	modelsDir, err := transcriber.DefaultModelsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mm := transcriber.NewModelManager(modelsDir)

	if aiRemote {
		// Show all available models (remote)
		fmt.Println("Available models (remote):")
		fmt.Println()
		for _, m := range transcriber.ASRModels {
			downloaded := ""
			if mm.IsModelDownloaded(m.Name) {
				downloaded = " [downloaded]"
			}
			fmt.Printf("  %-24s %8s  %s%s\n", m.Name, m.Size, m.Description, downloaded)
		}
		fmt.Println()
		fmt.Println("Download a model:")
		fmt.Println("  vget ai models download <model-name>")
	} else {
		// Show downloaded models only
		downloaded := mm.ListDownloadedModels()
		if len(downloaded) == 0 {
			fmt.Println("No models downloaded.")
			fmt.Println()
			fmt.Println("Download a model:")
			fmt.Println("  vget ai models download whisper-large-v3-turbo")
			fmt.Println()
			fmt.Println("See available models:")
			fmt.Println("  vget ai models -r")
			return
		}

		fmt.Println("Downloaded models:")
		fmt.Println()
		for _, name := range downloaded {
			model := transcriber.GetModel(name)
			if model != nil {
				fmt.Printf("  %-24s %8s  %s\n", model.Name, model.Size, model.Description)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
		fmt.Println()
		fmt.Printf("Models directory: %s\n", modelsDir)
	}
}

func runModelsDownload(cmd *cobra.Command, args []string) {
	modelName := args[0]

	// Validate model name
	model := transcriber.GetModel(modelName)
	if model == nil {
		fmt.Fprintf(os.Stderr, "Error: unknown model '%s'\n\n", modelName)
		fmt.Println("Available models:")
		for _, m := range transcriber.ASRModels {
			fmt.Printf("  %-24s (%s) - %s\n", m.Name, m.Size, m.Description)
		}
		os.Exit(1)
	}

	// Get models directory
	modelsDir, err := transcriber.DefaultModelsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mm := transcriber.NewModelManager(modelsDir)

	// Check if already downloaded
	if mm.IsModelDownloaded(modelName) {
		fmt.Printf("Model '%s' is already downloaded.\n", modelName)
		fmt.Printf("Location: %s\n", mm.ModelPath(modelName))
		return
	}

	// Determine download URL based on --from flag
	downloadURL := model.URL // Default: Hugging Face
	source := "Hugging Face"

	switch strings.ToLower(aiFrom) {
	case "vmirror":
		// vmirror.org mirror (faster in China)
		downloadURL = fmt.Sprintf("https://vmirror.org/models/whisper/%s", model.DirName)
		source = "vmirror.org"
	case "huggingface", "":
		// Default: Hugging Face (already set)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown source '%s'\n", aiFrom)
		fmt.Fprintln(os.Stderr, "Available sources: huggingface (default), vmirror")
		os.Exit(1)
	}

	// Show download info
	fmt.Printf("\nDownloading %s (%s)\n", model.Name, model.Size)
	fmt.Printf("Source: %s\n", source)

	// Get language for i18n
	cfg := config.LoadOrDefault()

	// Download with progress bar
	modelPath, err := mm.DownloadModelWithProgress(modelName, downloadURL, cfg.Language)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		if aiFrom != "vmirror" {
			fmt.Fprintf(os.Stderr, "\nTip: Try vmirror if Hugging Face is slow or blocked:\n")
			fmt.Fprintf(os.Stderr, "  vget ai models download %s --from=vmirror\n", modelName)
		}
		os.Exit(1)
	}

	fmt.Printf("\nDownload complete!\n")
	fmt.Printf("Location: %s\n", modelPath)
}

func runModelsRm(cmd *cobra.Command, args []string) {
	modelName := args[0]

	// Get models directory
	modelsDir, err := transcriber.DefaultModelsDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	mm := transcriber.NewModelManager(modelsDir)

	// Check if model exists
	if !mm.IsModelDownloaded(modelName) {
		fmt.Fprintf(os.Stderr, "Error: model '%s' is not downloaded\n", modelName)
		os.Exit(1)
	}

	modelPath := mm.ModelPath(modelName)

	// Remove the model
	if err := os.RemoveAll(modelPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing model: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed model: %s\n", modelName)
}

func init() {
	// Flags for transcribe command
	aiTranscribeCmd.Flags().StringVar(&aiModel, "model", "", "model to use (default: whisper-large-v3-turbo)")
	aiTranscribeCmd.Flags().StringVarP(&aiLanguage, "language", "l", "", "language code (required, e.g., zh, en, ja)")
	aiTranscribeCmd.Flags().StringVarP(&aiOutput, "output", "o", "", "output file path")

	// Flags for convert command
	aiConvertCmd.Flags().StringVar(&aiToFormat, "to", "", "output format: srt, vtt, txt (required)")
	aiConvertCmd.Flags().StringVarP(&aiOutput, "output", "o", "", "output file path")

	// Flags for models command
	aiModelsCmd.Flags().BoolVarP(&aiRemote, "remote", "r", false, "list models available for download")

	// Flags for models download command
	aiModelsDownloadCmd.Flags().StringVar(&aiFrom, "from", "huggingface", "download source: huggingface (default), vmirror")

	// Flags for download alias command
	aiDownloadCmd.Flags().StringVar(&aiFrom, "from", "huggingface", "download source: huggingface (default), vmirror")

	// Add subcommands to models
	aiModelsCmd.AddCommand(aiModelsDownloadCmd)
	aiModelsCmd.AddCommand(aiModelsRmCmd)

	// Add subcommands to ai
	aiCmd.AddCommand(aiTranscribeCmd)
	aiCmd.AddCommand(aiConvertCmd)
	aiCmd.AddCommand(aiModelsCmd)
	aiCmd.AddCommand(aiDownloadCmd) // Alias for models download

	// Add ai command to root
	rootCmd.AddCommand(aiCmd)
}
