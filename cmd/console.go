package cmd

import (
	"context"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Open the AWS console in the browser as the current account and role",
	Run:   runConsoleCommand,
}

var consoleService string
var consoleSwitchAccount bool

func init() {
	consoleCmd.Flags().StringVar(&consoleService, "service", "", "Service page to open, as in its console URL (e.g. ec2, rds, cloudwatch)")
	consoleCmd.Flags().BoolVarP(&consoleSwitchAccount, "switch-account", "s", false, "Switch AWS account before opening the console")
	rootCmd.AddCommand(consoleCmd)
}

func runConsoleCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	consoleManager, err := newManager(ctx, func(ctx context.Context) (*aws.ConsoleManager, error) {
		return aws.NewConsoleManager(ctx)
	}, consoleSwitchAccount)
	exitOnError(err)

	exitOnError(consoleManager.RunOpen(ctx, consoleService))
}
