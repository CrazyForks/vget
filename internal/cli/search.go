package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	podcastFlag bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for podcasts and episodes",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !podcastFlag {
			fmt.Fprintln(os.Stderr, "Please specify a search type: --podcast")
			os.Exit(1)
		}

		if podcastFlag {
			if err := searchXiaoyuzhou(args[0]); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	searchCmd.Flags().BoolVar(&podcastFlag, "podcast", false, "search for podcasts")
	rootCmd.AddCommand(searchCmd)
}

// XiaoyuzhouSearchResponse represents the API response
type XiaoyuzhouSearchResponse struct {
	Data struct {
		Episodes []XiaoyuzhouEpisode `json:"episodes"`
		Podcasts []XiaoyuzhouPodcast `json:"podcasts"`
	} `json:"data"`
}

type XiaoyuzhouPodcast struct {
	Type              string `json:"type"`
	Pid               string `json:"pid"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	Brief             string `json:"brief"`
	SubscriptionCount int    `json:"subscriptionCount"`
	EpisodeCount      int    `json:"episodeCount"`
}

type XiaoyuzhouEpisode struct {
	Type      string `json:"type"`
	Eid       string `json:"eid"`
	Pid       string `json:"pid"`
	Title     string `json:"title"`
	Duration  int    `json:"duration"`
	PlayCount int    `json:"playCount"`
	PubDate   string `json:"pubDate"`
	Enclosure struct {
		URL string `json:"url"`
	} `json:"enclosure"`
	Podcast struct {
		Title string `json:"title"`
	} `json:"podcast"`
}

func searchXiaoyuzhou(query string) error {
	// Call Xiaoyuzhou search API
	url := "https://ask.xiaoyuzhoufm.com/api/keyword/search"
	payload := fmt.Sprintf(`{"query": "%s"}`, query)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result XiaoyuzhouSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Display podcasts first
	if len(result.Data.Podcasts) > 0 {
		fmt.Println("\n\033[1;36m=== Podcasts ===\033[0m")
		for i, p := range result.Data.Podcasts {
			fmt.Printf("\033[1;33m[P%d]\033[0m %s\n", i+1, p.Title)
			fmt.Printf("     Author: %s | Episodes: %d | Subscribers: %d\n", p.Author, p.EpisodeCount, p.SubscriptionCount)
			fmt.Printf("     \033[90mhttps://www.xiaoyuzhoufm.com/podcast/%s\033[0m\n", p.Pid)
			if p.Brief != "" {
				brief := p.Brief
				if len(brief) > 80 {
					brief = brief[:80] + "..."
				}
				fmt.Printf("     %s\n", brief)
			}
			fmt.Println()
		}
	}

	// Display episodes
	if len(result.Data.Episodes) > 0 {
		fmt.Println("\033[1;36m=== Episodes ===\033[0m")
		for i, e := range result.Data.Episodes {
			duration := formatEpisodeDuration(e.Duration)
			fmt.Printf("\033[1;33m[E%d]\033[0m %s - %s\n", i+1, e.Podcast.Title, e.Title)
			fmt.Printf("     Duration: %s | Plays: %d\n", duration, e.PlayCount)
			fmt.Printf("     \033[90mhttps://www.xiaoyuzhoufm.com/episode/%s\033[0m\n", e.Eid)
			fmt.Println()
		}
	}

	if len(result.Data.Podcasts) == 0 && len(result.Data.Episodes) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	// Show download hint
	fmt.Println("\033[90m---")
	fmt.Println("To download, copy the URL and run: vget <url>\033[0m")

	return nil
}

func formatEpisodeDuration(seconds int) string {
	if seconds <= 0 {
		return "?"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
