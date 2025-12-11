package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/guiyumin/vget/internal/config"
	"github.com/guiyumin/vget/internal/tracker"
	"github.com/spf13/cobra"
)

var (
	trackExpress string // --express flag
	trackCourier string // --courier flag for courier code
)

var trackCmd = &cobra.Command{
	Use:   "track <tracking_number>",
	Short: "Track package delivery status",
	Long: `Track package delivery status using express tracking APIs.

Examples:
  vget track 73123456789 --courier yt        # Track YTO Express package
  vget track SF1234567890 --courier sf       # Track SF Express package
  vget track 1234567890 --courier jt         # Track JiTu Express package

Supported courier codes:
  sf       - 顺丰速运 (SF Express)
  yt       - 圆通速递 (YTO Express)
  sto      - 申通快递 (STO Express)
  zto      - 中通快递 (ZTO Express)
  yd       - 韵达快递 (Yunda Express)
  jt       - 极兔速递 (JiTu Express)
  jd       - 京东物流 (JD Logistics)
  ems      - EMS
  yzgn     - 邮政国内 (China Post)
  dbwl     - 德邦物流 (Deppon)
  anneng   - 安能物流 (Anneng)
  best     - 百世快递 (Best Express)
  kuayue   - 跨越速运 (Kuayue)
  ups      - UPS
  fedex    - FedEx
  dhl      - DHL

Configuration:
  Set your kuaidi100 API credentials:
  vget config set express.kuaidi100.key <your_key>
  vget config set express.kuaidi100.customer <your_customer_id>

  Get credentials at: https://api.kuaidi100.com/manager/v2/myinfo/enterprise`,
	Args: cobra.ExactArgs(1),
	RunE: runTrack,
}

func init() {
	trackCmd.Flags().StringVar(&trackExpress, "express", "kuaidi100", "Express tracking service (default: kuaidi100)")
	trackCmd.Flags().StringVarP(&trackCourier, "courier", "c", "auto", "Courier company code (e.g., sf, yt, zto, or auto for auto-detect)")
	rootCmd.AddCommand(trackCmd)
}

func runTrack(cmd *cobra.Command, args []string) error {
	trackingNumber := args[0]

	// Currently only kuaidi100 is supported
	if trackExpress != "kuaidi100" {
		return fmt.Errorf("unsupported express service: %s (only kuaidi100 is supported)", trackExpress)
	}

	// Load config
	cfg := config.LoadOrDefault()

	// Get kuaidi100 credentials from express config
	expressCfg := cfg.GetExpressConfig("kuaidi100")
	if expressCfg == nil || expressCfg["key"] == "" || expressCfg["customer"] == "" {
		fmt.Fprintln(os.Stderr, color.RedString("Error: kuaidi100 API credentials not configured"))
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Please set your credentials:")
		fmt.Fprintln(os.Stderr, "  vget config set express.kuaidi100.key <your_key>")
		fmt.Fprintln(os.Stderr, "  vget config set express.kuaidi100.customer <your_customer_id>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Get your credentials at: https://api.kuaidi100.com/manager/v2/myinfo/enterprise")
		return fmt.Errorf("missing kuaidi100 credentials")
	}

	// Create tracker
	t := tracker.NewKuaidi100Tracker(expressCfg["key"], expressCfg["customer"])

	// Convert courier alias to kuaidi100 code
	courierCode := tracker.GetCourierCode(trackCourier)

	// Get courier info for display
	courierInfo := tracker.GetCourierInfo(trackCourier)
	if courierInfo != nil {
		fmt.Printf("Courier: %s (%s)\n", courierInfo.Name, courierCode)
	} else {
		fmt.Printf("Courier: %s\n", courierCode)
	}
	fmt.Printf("Tracking: %s\n\n", trackingNumber)

	// Track the package
	result, err := t.Track(courierCode, trackingNumber)
	if err != nil {
		return fmt.Errorf("tracking failed: %w", err)
	}

	// Display results
	printTrackingResult(result)

	return nil
}

func printTrackingResult(result *tracker.TrackingResponse) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	cyan := color.New(color.FgCyan)

	// Status
	bold.Printf("Status: ")
	if result.IsDelivered() {
		green.Println(result.StateDescription() + " ✓")
	} else {
		yellow.Println(result.StateDescription())
	}

	fmt.Println()

	// Tracking events
	if len(result.Data) == 0 {
		fmt.Println("No tracking information available yet.")
		return
	}

	bold.Println("Tracking History:")
	fmt.Println(strings.Repeat("-", 60))

	for i, event := range result.Data {
		// Time
		timeStr := event.Ftime
		if timeStr == "" {
			timeStr = event.Time
		}
		cyan.Printf("[%s]", timeStr)
		fmt.Println()

		// Context/description
		fmt.Printf("  %s", event.Context)

		// Location if available
		if event.Location != "" {
			fmt.Printf(" (%s)", event.Location)
		} else if event.AreaName != "" {
			fmt.Printf(" (%s)", event.AreaName)
		}
		fmt.Println()

		// Add separator except for last item
		if i < len(result.Data)-1 {
			fmt.Println()
		}
	}

	fmt.Println(strings.Repeat("-", 60))
}
