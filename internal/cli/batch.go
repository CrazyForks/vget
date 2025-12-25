package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// runBatch reads URLs from a file and downloads each one
func runBatch(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs found in file")
	}

	fmt.Printf("Found %d URL(s) to download\n\n", len(urls))

	// Separate Telegram URLs from other URLs
	var telegramURLs, otherURLs []string
	for _, url := range urls {
		if isTelegramURL(url) {
			telegramURLs = append(telegramURLs, url)
		} else {
			otherURLs = append(otherURLs, url)
		}
	}

	var succeeded, failed int
	var failedURLs []string

	// Handle Telegram URLs with batch function (uses takeout if multiple)
	if len(telegramURLs) >= 2 {
		s, f, fURLs := runTelegramBatchDownload(telegramURLs)
		succeeded += s
		failed += f
		failedURLs = append(failedURLs, fURLs...)
	} else {
		// Single Telegram URL - use regular download
		for _, url := range telegramURLs {
			fmt.Printf("[1/%d] %s\n", len(urls), truncateURL(url, 60))
			if err := runTelegramDownload(url, ""); err != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
				failed++
				failedURLs = append(failedURLs, url)
			} else {
				succeeded++
			}
			fmt.Println()
		}
	}

	// Download other URLs
	startIdx := len(telegramURLs) + 1
	for i, url := range otherURLs {
		fmt.Printf("[%d/%d] %s\n", startIdx+i, len(urls), truncateURL(url, 60))

		if err := runDownload(url); err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			failed++
			failedURLs = append(failedURLs, url)
		} else {
			succeeded++
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("----------------------------------------")
	fmt.Printf("Completed: %d/%d", succeeded, len(urls))
	if failed > 0 {
		fmt.Printf(", Failed: %d", failed)
	}
	fmt.Println()

	// List failed URLs if any
	if len(failedURLs) > 0 {
		fmt.Println("\nFailed URLs:")
		for _, url := range failedURLs {
			fmt.Printf("  - %s\n", url)
		}
	}

	return nil
}

// truncateURL shortens a URL for display
func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
