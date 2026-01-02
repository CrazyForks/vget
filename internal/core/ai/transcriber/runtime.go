package transcriber

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guiyumin/vget/internal/core/downloader"
)

// RuntimeVersion is the current version of whisper.cpp binaries.
const RuntimeVersion = "v1.8.2"

// CUDA version for Windows cuBLAS builds.
const CUDAVersion = "12.6.3"

// Runtime represents an AI runtime binary (e.g., whisper.cpp, piper, tesseract).
type Runtime struct {
	Name        string // e.g., "whisper"
	Version     string // e.g., "v1.8.2"
	Platform    string // e.g., "darwin-arm64"
	URL         string // Download URL
	Size        string // Human-readable size
	Accelerator string // "metal", "cuda", "cpu"
}

// whisperRuntimes lists available whisper.cpp binaries for each platform.
// These binaries include GPU acceleration where available:
// - macOS ARM64: Metal GPU acceleration
// - macOS x64: Accelerate framework
// - Windows: cuBLAS (CUDA) for NVIDIA GPUs
// - Linux: OpenBLAS CPU (CUDA requires custom build)
var whisperRuntimes = map[string]Runtime{
	"darwin-arm64": {
		Name:        "whisper",
		Version:     RuntimeVersion,
		Platform:    "darwin-arm64",
		URL:         "https://github.com/ggerganov/whisper.cpp/releases/download/" + RuntimeVersion + "/whisper-" + RuntimeVersion + "-bin-macos-arm64.zip",
		Size:        "~3MB",
		Accelerator: "metal",
	},
	"darwin-amd64": {
		Name:        "whisper",
		Version:     RuntimeVersion,
		Platform:    "darwin-amd64",
		URL:         "https://github.com/ggerganov/whisper.cpp/releases/download/" + RuntimeVersion + "/whisper-" + RuntimeVersion + "-bin-macos-x64.zip",
		Size:        "~3MB",
		Accelerator: "accelerate",
	},
	"linux-amd64": {
		Name:        "whisper",
		Version:     RuntimeVersion,
		Platform:    "linux-amd64",
		URL:         "https://github.com/ggerganov/whisper.cpp/releases/download/" + RuntimeVersion + "/whisper-" + RuntimeVersion + "-bin-ubuntu-x64.tar.gz",
		Size:        "~3MB",
		Accelerator: "cpu",
	},
	"linux-arm64": {
		Name:        "whisper",
		Version:     RuntimeVersion,
		Platform:    "linux-arm64",
		// Note: No official arm64 linux release, use x64 binary
		// For ARM64 Linux (e.g., Raspberry Pi), users should build from source
		URL:         "https://github.com/ggerganov/whisper.cpp/releases/download/" + RuntimeVersion + "/whisper-" + RuntimeVersion + "-bin-ubuntu-x64.tar.gz",
		Size:        "~3MB",
		Accelerator: "cpu",
	},
	"windows-amd64": {
		Name:        "whisper",
		Version:     RuntimeVersion,
		Platform:    "windows-amd64",
		// Use cuBLAS build for NVIDIA GPU acceleration on Windows
		URL:         "https://github.com/ggerganov/whisper.cpp/releases/download/" + RuntimeVersion + "/whisper-" + RuntimeVersion + "-bin-win-cublas-" + CUDAVersion + "-x64.zip",
		Size:        "~50MB",
		Accelerator: "cuda",
	},
}

// RuntimeManager handles downloading and managing AI runtime binaries.
type RuntimeManager struct {
	binDir string
}

// NewRuntimeManager creates a new runtime manager.
func NewRuntimeManager(binDir string) *RuntimeManager {
	return &RuntimeManager{binDir: binDir}
}

// DefaultBinDir returns the default bin directory (~/.config/vget/bin).
func DefaultBinDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "vget", "bin"), nil
}

// getPlatformKey returns the current platform key (e.g., "darwin-arm64").
func getPlatformKey() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	return goos + "-" + goarch
}

// GetWhisperRuntime returns the whisper runtime for the current platform.
func GetWhisperRuntime() (*Runtime, error) {
	platform := getPlatformKey()
	rt, ok := whisperRuntimes[platform]
	if !ok {
		return nil, fmt.Errorf("whisper.cpp not available for platform: %s", platform)
	}
	return &rt, nil
}

// WhisperBinaryPath returns the path to the whisper binary.
func (r *RuntimeManager) WhisperBinaryPath() string {
	platform := getPlatformKey()
	binaryName := "whisper-cli"
	if strings.HasPrefix(platform, "windows") {
		binaryName = "whisper-cli.exe"
	}
	return filepath.Join(r.binDir, binaryName)
}

// IsWhisperInstalled checks if whisper.cpp is installed.
func (r *RuntimeManager) IsWhisperInstalled() bool {
	path := r.WhisperBinaryPath()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}

// EnsureWhisper downloads whisper.cpp if not already present.
func (r *RuntimeManager) EnsureWhisper() (string, error) {
	if r.IsWhisperInstalled() {
		return r.WhisperBinaryPath(), nil
	}

	rt, err := GetWhisperRuntime()
	if err != nil {
		return "", err
	}

	// Show what acceleration will be used
	accelInfo := ""
	switch rt.Accelerator {
	case "metal":
		accelInfo = " (Metal GPU)"
	case "cuda":
		accelInfo = " (CUDA GPU)"
	case "accelerate":
		accelInfo = " (Accelerate)"
	case "cpu":
		accelInfo = " (CPU)"
	}

	fmt.Printf("  Downloading whisper.cpp %s for %s%s...\n", rt.Version, rt.Platform, accelInfo)

	if err := r.downloadAndExtract(rt); err != nil {
		return "", fmt.Errorf("failed to download whisper.cpp: %w", err)
	}

	return r.WhisperBinaryPath(), nil
}

// DownloadWhisperWithProgress downloads whisper.cpp with progress display.
func (r *RuntimeManager) DownloadWhisperWithProgress(lang string) (string, error) {
	rt, err := GetWhisperRuntime()
	if err != nil {
		return "", err
	}

	// Check if already installed
	if r.IsWhisperInstalled() {
		return r.WhisperBinaryPath(), nil
	}

	// Ensure bin directory exists
	if err := os.MkdirAll(r.binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	fmt.Printf("  Downloading whisper.cpp %s for %s...\n", rt.Version, rt.Platform)
	fmt.Printf("  URL: %s\n\n", rt.URL)

	// Download to temp file first
	tmpFile, err := os.CreateTemp("", "whisper-download-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Try TUI progress bar
	err = downloader.RunDownloadTUI(rt.URL, tmpPath, "whisper.cpp", lang, nil)
	if err != nil && isNoTTYError(err) {
		// Fall back to simple progress
		if err := r.downloadWithSimpleProgress(rt.URL, tmpPath); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// Extract the downloaded archive
	if err := r.extractArchive(tmpPath, rt.URL); err != nil {
		return "", fmt.Errorf("failed to extract whisper.cpp: %w", err)
	}

	return r.WhisperBinaryPath(), nil
}

// downloadAndExtract downloads and extracts a runtime binary.
func (r *RuntimeManager) downloadAndExtract(rt *Runtime) error {
	// Ensure bin directory exists
	if err := os.MkdirAll(r.binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "whisper-download-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	resp, err := http.Get(rt.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return err
	}
	tmpFile.Close()

	// Extract based on file extension
	return r.extractArchive(tmpPath, rt.URL)
}

// extractArchive extracts a zip or tar.gz archive.
func (r *RuntimeManager) extractArchive(archivePath, url string) error {
	if strings.HasSuffix(url, ".zip") {
		return r.extractZip(archivePath)
	} else if strings.HasSuffix(url, ".tar.gz") {
		return r.extractTarGz(archivePath)
	}
	return fmt.Errorf("unsupported archive format: %s", url)
}

// extractZip extracts a zip archive, looking for the whisper-cli binary and required DLLs.
func (r *RuntimeManager) extractZip(archivePath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	foundBinary := false
	for _, file := range reader.File {
		baseName := filepath.Base(file.Name)

		// Extract whisper-cli binary
		if baseName == "whisper-cli" || baseName == "whisper-cli.exe" {
			if err := r.extractSingleFile(file); err != nil {
				return err
			}
			foundBinary = true
			continue
		}

		// Extract required DLLs for cuBLAS builds (Windows)
		// These include: cublas64_*.dll, cublasLt64_*.dll, cudart64_*.dll, etc.
		if strings.HasSuffix(baseName, ".dll") {
			if err := r.extractSingleFile(file); err != nil {
				return err
			}
		}
	}

	if !foundBinary {
		return fmt.Errorf("whisper-cli binary not found in archive")
	}

	return nil
}

// extractSingleFile extracts a single file from a zip entry.
func (r *RuntimeManager) extractSingleFile(file *zip.File) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	targetPath := filepath.Join(r.binDir, filepath.Base(file.Name))
	outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// extractTarGz extracts a tar.gz archive, looking for the whisper-cli binary.
func (r *RuntimeManager) extractTarGz(archivePath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for whisper-cli binary
		baseName := filepath.Base(header.Name)
		if baseName == "whisper-cli" && header.Typeflag == tar.TypeReg {
			targetPath := filepath.Join(r.binDir, baseName)
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("whisper-cli binary not found in archive")
}

// downloadWithSimpleProgress downloads a file with simple console progress.
func (r *RuntimeManager) downloadWithSimpleProgress(url, targetPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	total := resp.ContentLength
	var current int64
	buf := make([]byte, 32*1024)
	lastPercent := -1

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := file.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			current += int64(n)

			if total > 0 {
				percent := int(float64(current) / float64(total) * 100)
				if percent/5 > lastPercent/5 {
					fmt.Printf("\r  Progress: %d%% (%s / %s)", percent, formatBytes(current), formatBytes(total))
					lastPercent = percent
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println()

	return nil
}
