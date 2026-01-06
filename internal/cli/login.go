package cli

import (
	"github.com/guiyumin/vget/internal/cli/login"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to media platforms",
	Long:  "Login to various media platforms to download member-only content",
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from media platforms",
	Long:  "Clear saved credentials for media platforms",
}

func init() {
	loginCmd.AddCommand(login.BilibiliCmd())
	loginCmd.AddCommand(login.GoogleCmd())
	logoutCmd.AddCommand(login.BilibiliLogoutCmd())
	logoutCmd.AddCommand(login.GoogleLogoutCmd())
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
}
