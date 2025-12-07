package youtube

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/guiyumin/vget/internal/config"
)

func (e *Extractor) extractSessionTokens(videoID string) (*Session, error) {
	l := e.createLauncher(!e.visible)
	defer l.Cleanup()

	fmt.Println("Launching browser for token extraction...")

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := stealth.MustPage(browser)
	defer page.MustClose()

	var session Session
	var capturedPOToken bool
	var capturedVisitorData bool
	var mu sync.Mutex

	router := page.HijackRequests()

	// Intercept /player requests for POToken
	router.MustAdd("*youtubei/v1/player*", func(ctx *rod.Hijack) {
		mu.Lock()
		defer mu.Unlock()

		body := ctx.Request.Body()
		if body != "" {
			var reqBody map[string]any
			if err := json.Unmarshal([]byte(body), &reqBody); err == nil {
				if sid, ok := reqBody["serviceIntegrityDimensions"].(map[string]any); ok {
					if pot, ok := sid["poToken"].(string); ok && pot != "" {
						session.POToken = pot
						capturedPOToken = true
						fmt.Printf("Captured POToken: %d chars\n", len(pot))
					}
				}

				if ctxMap, ok := reqBody["context"].(map[string]any); ok {
					if client, ok := ctxMap["client"].(map[string]any); ok {
						if vd, ok := client["visitorData"].(string); ok && vd != "" {
							session.VisitorData = vd
							capturedVisitorData = true
							fmt.Printf("Captured VisitorData: %s...\n", truncate(vd, 20))
						}
					}
				}
			}
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	// Also intercept /next requests which often contain tokens
	router.MustAdd("*youtubei/v1/next*", func(ctx *rod.Hijack) {
		mu.Lock()
		defer mu.Unlock()

		body := ctx.Request.Body()
		if body != "" {
			var reqBody map[string]any
			if err := json.Unmarshal([]byte(body), &reqBody); err == nil {
				if sid, ok := reqBody["serviceIntegrityDimensions"].(map[string]any); ok {
					if pot, ok := sid["poToken"].(string); ok && pot != "" && !capturedPOToken {
						session.POToken = pot
						capturedPOToken = true
						fmt.Printf("Captured POToken from /next: %d chars\n", len(pot))
					}
				}
			}
		}

		ctx.ContinueRequest(&proto.FetchContinueRequest{})
	})

	go router.Run()

	watchURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	fmt.Printf("Navigating to: %s\n", watchURL)

	err = page.Navigate(watchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	page.MustWaitDOMStable()

	maxWait := 25 * time.Second
	start := time.Now()
	triedPlay := false
	triedEmbed := false

	for {
		mu.Lock()
		hasPOToken := capturedPOToken
		hasVisitorData := capturedVisitorData
		mu.Unlock()

		if hasPOToken && hasVisitorData {
			fmt.Println("Token capture complete!")
			break
		}

		elapsed := time.Since(start)

		// After 8 seconds, try to trigger playback
		if elapsed > 8*time.Second && !triedPlay {
			triedPlay = true
			fmt.Println("Trying to trigger playback...")

			_ = page.MustEval(`() => {
				const dismissBtns = document.querySelectorAll('button[aria-label*="Dismiss"], .ytp-ad-skip-button, paper-button[aria-label*="No thanks"]');
				dismissBtns.forEach(btn => btn.click());

				const playBtn = document.querySelector('button.ytp-large-play-button, button.ytp-play-button');
				if (playBtn) playBtn.click();

				const video = document.querySelector('video');
				if (video) {
					video.muted = true;
					video.play().catch(() => {});
				}
			}`)

			time.Sleep(3 * time.Second)
		}

		// After 15 seconds, try embed page
		if elapsed > 15*time.Second && !triedEmbed && !hasPOToken {
			triedEmbed = true
			fmt.Println("Trying embed page...")

			embedURL := fmt.Sprintf("https://www.youtube.com/embed/%s?autoplay=1", videoID)
			page.MustNavigate(embedURL)
			page.MustWaitDOMStable()

			time.Sleep(2 * time.Second)

			_ = page.MustEval(`() => {
				const playBtn = document.querySelector('button.ytp-large-play-button');
				if (playBtn) playBtn.click();
				const video = document.querySelector('video');
				if (video) {
					video.muted = true;
					video.play().catch(() => {});
				}
			}`)
		}

		// Timeout - try to get visitorData from page config as fallback
		if elapsed > maxWait {
			if !hasVisitorData {
				visitorData := page.MustEval(`() => {
					try {
						return ytcfg.get('VISITOR_DATA') ||
						       window.ytInitialPlayerResponse?.responseContext?.visitorData ||
						       '';
					} catch(e) {
						return '';
					}
				}`).String()

				if visitorData != "" {
					session.VisitorData = visitorData
					capturedVisitorData = true
					fmt.Printf("Got VisitorData from page config: %s...\n", truncate(visitorData, 20))
				}
			}
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	// Extract dynamic client context from ytcfg
	clientContext := page.MustEval(`() => {
		try {
			return {
				clientVersion: ytcfg.get('INNERTUBE_CLIENT_VERSION') || '',
				sts: ytcfg.get('STS') || 0
			};
		} catch(e) {
			return { clientVersion: '', sts: 0 };
		}
	}`)

	if cv := clientContext.Get("clientVersion").String(); cv != "" {
		session.ClientVersion = cv
		fmt.Printf("Got client version: %s\n", cv)
	}
	if sts := clientContext.Get("sts").Int(); sts > 0 {
		session.SignatureTimestamp = sts
		fmt.Printf("Got signature timestamp: %d\n", sts)
	}

	// Get cookies
	cookies, err := browser.GetCookies()
	if err == nil {
		session.Cookies = cookies
	}

	e.saveSession(&session)

	if session.VisitorData == "" {
		return nil, fmt.Errorf("failed to capture visitorData")
	}

	mu.Lock()
	if capturedPOToken {
		fmt.Printf("Session ready: POToken (%d chars), VisitorData (%d chars)\n",
			len(session.POToken), len(session.VisitorData))
	} else {
		fmt.Println("Warning: No POToken captured, proceeding with VisitorData only...")
	}
	mu.Unlock()

	return &session, nil
}

func (e *Extractor) createLauncher(headless bool) *launcher.Launcher {
	userDataDir := e.getUserDataDir()

	l := launcher.New().
		Headless(headless).
		UserDataDir(userDataDir).
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("window-size", "1920,1080").
		Set("lang", "en-US")

	// Support HTTP_PROXY / HTTPS_PROXY environment variables
	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		l = l.Proxy(proxy)
		fmt.Printf("Using proxy: %s\n", proxy)
	} else if proxy := os.Getenv("HTTP_PROXY"); proxy != "" {
		l = l.Proxy(proxy)
		fmt.Printf("Using proxy: %s\n", proxy)
	}

	return l
}

func (e *Extractor) getUserDataDir() string {
	configDir, err := config.ConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "vget-browser")
	}
	return filepath.Join(configDir, "browser")
}
