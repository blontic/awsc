package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/swa/internal/aws"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "CloudWatch Logs operations",
	Long:  "List log groups and tail log streams from CloudWatch Logs",
}

var logsTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Tail CloudWatch logs",
	Long:  "Tail logs from a CloudWatch log group",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		manager, err := createLogsManager(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logs manager: %v\n", err)
			os.Exit(1)
		}

		groupName, _ := cmd.Flags().GetString("group")
		since, _ := cmd.Flags().GetString("since")
		follow, _ := cmd.Flags().GetBool("follow")

		if err := manager.RunTail(ctx, groupName, since, follow); err != nil {
			fmt.Fprintf(os.Stderr, "Error tailing logs: %v\n", err)
			os.Exit(1)
		}
	},
}

func createLogsManager(ctx context.Context) (*aws.LogsManager, error) {
	return aws.NewLogsManager(ctx)
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(logsTailCmd)

	logsTailCmd.Flags().StringP("group", "g", "", "Log group name")
	logsTailCmd.Flags().StringP("since", "s", "10m", "From what time to begin displaying logs (e.g., 5m, 1h, 2d)")
	logsTailCmd.Flags().BoolP("follow", "f", false, "Follow log output")
}
