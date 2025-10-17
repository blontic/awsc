package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var opensearchCmd = &cobra.Command{
	Use:   "opensearch",
	Short: "OpenSearch domain connections",
	Long:  `Connect to OpenSearch domains via EC2 bastion hosts using SSM port forwarding`,
}

var opensearchConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an OpenSearch domain via bastion host",
	Long:  `List OpenSearch domains, find suitable bastion hosts, and establish SSM port forwarding connection`,
	Run:   runOpenSearchConnect,
}

var opensearchLocalPort int
var opensearchDomainName string
var opensearchSwitchAccount bool

func init() {
	rootCmd.AddCommand(opensearchCmd)
	opensearchCmd.AddCommand(opensearchConnectCmd)
	opensearchConnectCmd.Flags().IntVar(&opensearchLocalPort, "local-port", 443, "Local port for port forwarding (defaults to 443)")
	opensearchConnectCmd.Flags().StringVar(&opensearchDomainName, "name", "", "Name of the OpenSearch domain to connect to directly")
	opensearchConnectCmd.Flags().BoolVarP(&opensearchSwitchAccount, "switch-account", "s", false, "Switch AWS account before connecting")
}

func runOpenSearchConnect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create OpenSearch manager
	opensearchManager, err := aws.NewOpenSearchManager(ctx)
	if err != nil {
		// Check if this is a "no active session" error
		if aws.IsAuthError(err) {
			shouldReauth, reAuthErr := aws.PromptForReauth(ctx)
			if reAuthErr != nil {
				fmt.Printf("Error during re-authentication: %v\n", reAuthErr)
				os.Exit(1)
			}
			if !shouldReauth {
				fmt.Printf("Authentication cancelled\n")
				os.Exit(1)
			}
			// Retry creating manager after successful login
			opensearchManager, err = aws.NewOpenSearchManager(ctx)
			if err != nil {
				fmt.Printf("Error creating OpenSearch manager after re-authentication: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating OpenSearch manager: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle account switching if requested
	if opensearchSwitchAccount {
		if err := handleAccountSwitch(ctx); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		// Recreate OpenSearch manager with new credentials
		opensearchManager, err = aws.NewOpenSearchManager(ctx)
		if err != nil {
			fmt.Printf("Error creating OpenSearch manager after account switch: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the OpenSearch connect workflow
	if err := opensearchManager.RunConnect(ctx, opensearchDomainName, int32(opensearchLocalPort)); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
