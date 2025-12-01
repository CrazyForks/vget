package cli

import (
	"fmt"
	"os"

	"github.com/guiyumin/vget/internal/config"
	"github.com/guiyumin/vget/internal/downloader"
	"github.com/guiyumin/vget/internal/extractor"
	"github.com/guiyumin/vget/internal/i18n"
	"github.com/guiyumin/vget/internal/version"
	"github.com/spf13/cobra"
)

var (
	output  string
	quality string
	info    bool
)

var rootCmd = &cobra.Command{
	Use:     "vget [url]",
	Short:   "Versatile command-line toolkit for downloading audio, video, podcasts, and more",
	Version: version.Version,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		if err := runDownload(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "output filename")
	rootCmd.Flags().StringVarP(&quality, "quality", "q", "", "preferred quality (e.g., 1080p, 720p)")
	rootCmd.Flags().BoolVar(&info, "info", false, "show video info without downloading")
}

func Execute() error {
	return rootCmd.Execute()
}

func runDownload(url string) error {
	cfg := config.LoadOrDefault()
	t := i18n.T(cfg.Language)

	// Check for config file and warn if missing
	if !config.Exists() {
		fmt.Fprintf(os.Stderr, "\033[33m%s. Run 'vget init'.\033[0m\n", t.Errors.ConfigNotFound)
	}

	// Find matching extractor
	ext := extractor.Match(url)
	if ext == nil {
		return fmt.Errorf("%s: %s", t.Errors.NoExtractor, url)
	}

	// Extract media info with spinner
	media, err := runExtractWithSpinner(ext, url, cfg.Language)
	if err != nil {
		return err
	}

	dl := downloader.New(cfg.Language)

	// Handle based on media type
	switch m := media.(type) {
	case *extractor.VideoMedia:
		return downloadVideo(m, dl, t)
	case *extractor.AudioMedia:
		return downloadAudio(m, dl)
	case *extractor.ImageMedia:
		return downloadImages(m, dl)
	default:
		return fmt.Errorf("unsupported media type")
	}
}

func downloadVideo(m *extractor.VideoMedia, dl *downloader.Downloader, t *i18n.Translations) error {
	// Info only mode
	if info {
		for i, f := range m.Formats {
			fmt.Printf("  [%d] %s %dx%d (%s)\n", i, f.Quality, f.Width, f.Height, f.Ext)
		}
		return nil
	}

	// Select best format (or by quality flag)
	format := selectVideoFormat(m.Formats, quality)
	if format == nil {
		return fmt.Errorf(t.Download.NoFormats)
	}

	fmt.Printf("  %s: %s (%s)\n", t.Download.SelectedFormat, format.Quality, format.Ext)

	// Determine output filename
	outputFile := output
	if outputFile == "" {
		if m.Title != "" {
			outputFile = fmt.Sprintf("%s.%s", m.Title, format.Ext)
		} else {
			outputFile = fmt.Sprintf("%s.%s", m.ID, format.Ext)
		}
	}

	return dl.Download(format.URL, outputFile, m.ID)
}

func downloadAudio(m *extractor.AudioMedia, dl *downloader.Downloader) error {
	// Info only mode
	if info {
		fmt.Printf("  Audio: %s (%s)\n", m.Title, m.Ext)
		return nil
	}

	// Determine output filename
	outputFile := output
	if outputFile == "" {
		if m.Title != "" {
			outputFile = fmt.Sprintf("%s.%s", m.Title, m.Ext)
		} else {
			outputFile = fmt.Sprintf("%s.%s", m.ID, m.Ext)
		}
	}

	return dl.Download(m.URL, outputFile, m.ID)
}

func downloadImages(m *extractor.ImageMedia, dl *downloader.Downloader) error {
	// Info only mode
	if info {
		fmt.Printf("  Images (%d):\n", len(m.Images))
		for i, img := range m.Images {
			fmt.Printf("    [%d] %dx%d (%s)\n", i+1, img.Width, img.Height, img.Ext)
		}
		return nil
	}

	fmt.Printf("  Downloading %d image(s)...\n", len(m.Images))

	for i, img := range m.Images {
		var outputFile string
		if output != "" {
			// If custom output specified, add suffix for multiple images
			if len(m.Images) > 1 {
				outputFile = fmt.Sprintf("%s_%d.%s", output, i+1, img.Ext)
			} else {
				outputFile = fmt.Sprintf("%s.%s", output, img.Ext)
			}
		} else {
			// Use ID with index suffix
			if len(m.Images) > 1 {
				outputFile = fmt.Sprintf("%s_%d.%s", m.ID, i+1, img.Ext)
			} else {
				outputFile = fmt.Sprintf("%s.%s", m.ID, img.Ext)
			}
		}

		if err := dl.Download(img.URL, outputFile, m.ID); err != nil {
			return fmt.Errorf("failed to download image %d: %w", i+1, err)
		}
	}
	return nil
}

func selectVideoFormat(formats []extractor.VideoFormat, preferred string) *extractor.VideoFormat {
	if len(formats) == 0 {
		return nil
	}

	// If quality specified, try to match
	if preferred != "" {
		for i := range formats {
			if formats[i].Quality == preferred {
				return &formats[i]
			}
		}
	}

	// Otherwise return highest bitrate
	best := &formats[0]
	for i := range formats {
		if formats[i].Bitrate > best.Bitrate {
			best = &formats[i]
		}
	}
	return best
}
