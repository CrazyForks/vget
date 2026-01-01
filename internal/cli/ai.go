package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/guiyumin/vget/internal/core/ai"
	"github.com/guiyumin/vget/internal/core/ai/transcriber"
	"github.com/guiyumin/vget/internal/core/config"
	"github.com/spf13/cobra"
)

var (
	aiModel    string
	aiLanguage string
	aiFrom     string
)

// aiCmd is the parent command for all AI features
var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered transcription and more",
	Long: `AI features for vget including speech-to-text transcription.

Models are downloaded on first use and stored in ~/.config/vget/models/

Examples:
  vget ai transcribe audio.mp3
  vget ai download whisper-large-v3-turbo
  vget ai models`,
}

// aiTranscribeCmd transcribes audio/video files
var aiTranscribeCmd = &cobra.Command{
	Use:   "transcribe <file>",
	Short: "Transcribe audio/video to text",
	Long: `Transcribe audio or video files to text using local Whisper models.

The transcript is saved as <filename>.transcript.md

Examples:
  vget ai transcribe podcast.mp3
  vget ai transcribe video.mp4 --model whisper-small
  vget ai transcribe audio.m4a --language zh`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]

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
			fmt.Printf("Whisper model not found. Download it with:\n")
			fmt.Printf("  vget ai download %s\n\n", modelName)
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

		fmt.Printf("\nTranscript saved: %s\n", result.TranscriptPath)
	},
}

// aiDownloadCmd downloads a model
var aiDownloadCmd = &cobra.Command{
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

Examples:
  vget ai download whisper-large-v3-turbo
  vget ai download whisper-small
  vget ai download whisper-small --from=vget   # Use vget mirror`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
		if aiFrom == "vget" {
			// vget mirror (Cloudflare R2)
			downloadURL = fmt.Sprintf("https://models.vget.io/%s.bin", modelName)
			source = "vget mirror"
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
			if aiFrom != "vget" {
				fmt.Fprintf(os.Stderr, "\nTip: Try the vget mirror if Hugging Face is slow or blocked:\n")
				fmt.Fprintf(os.Stderr, "  vget ai download %s --from=vget\n", modelName)
			}
			os.Exit(1)
		}

		fmt.Printf("Location: %s\n", modelPath)
	},
}

// aiModelsCmd lists available and downloaded models
var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available transcription models",
	Run: func(cmd *cobra.Command, args []string) {
		modelsDir, err := transcriber.DefaultModelsDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		mm := transcriber.NewModelManager(modelsDir)
		models := mm.ListAvailableModels()

		fmt.Println("Available models:")
		fmt.Println()

		for _, m := range models {
			status := "  "
			if m.Downloaded {
				status = "✓ "
			}
			fmt.Printf("  %s%-24s %8s  %s\n", status, m.Name, m.Size, m.Description)
		}

		fmt.Println()
		fmt.Printf("Models directory: %s\n", modelsDir)
		fmt.Println()
		fmt.Println("Download a model:")
		fmt.Println("  vget ai download <model-name>")
	},
}

func init() {
	// Flags for transcribe command
	aiTranscribeCmd.Flags().StringVar(&aiModel, "model", "", "model to use (default: whisper-large-v3-turbo)")
	aiTranscribeCmd.Flags().StringVar(&aiLanguage, "language", "", "language hint (e.g., zh, en, ja)")

	// Flags for download command
	aiDownloadCmd.Flags().StringVar(&aiFrom, "from", "huggingface", "download source: huggingface (default) or vget")

	// Add subcommands
	aiCmd.AddCommand(aiTranscribeCmd)
	aiCmd.AddCommand(aiDownloadCmd)
	aiCmd.AddCommand(aiModelsCmd)

	// Add ai command to root
	rootCmd.AddCommand(aiCmd)
}
