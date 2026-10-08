package cmd

import (
	"context"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of AWS SSO",
	Long: `Log out of the active org: end its SSO session with AWS, delete the cached
SSO login and clear this terminal's account. Other terminals using the org
are asked to log in again on their next command.

With --all, log out of every org and clear every terminal's account.`,
	Run: runLogoutCommand,
}

var logoutAll bool

func init() {
	logoutCmd.Flags().BoolVar(&logoutAll, "all", false, "Log out of every org and clear all terminals")
	rootCmd.AddCommand(logoutCmd)
}

func runLogoutCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	logoutManager, err := aws.NewLogoutManager(ctx)
	exitOnError(err)
	exitOnError(logoutManager.RunLogout(ctx, logoutAll))
}
