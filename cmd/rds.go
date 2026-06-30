package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var rdsCmd = &cobra.Command{
	Use:   "rds",
	Short: "RDS database connections",
	Long:  `Connect to RDS instances via EC2 bastion hosts using SSM port forwarding`,
}

var rdsConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an RDS instance via bastion host",
	Long:  `List RDS instances, find suitable bastion hosts, and establish SSM port forwarding connection`,
	Run:   runRDSConnect,
}

var localPort int
var rdsInstanceName string
var switchAccount bool
var rdsListBastions bool

func init() {
	rootCmd.AddCommand(rdsCmd)
	rdsCmd.AddCommand(rdsConnectCmd)
	rdsConnectCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port for port forwarding (defaults to RDS port)")
	rdsConnectCmd.Flags().StringVar(&rdsInstanceName, "name", "", "Name of the RDS instance to connect to directly")
	rdsConnectCmd.Flags().BoolVarP(&switchAccount, "switch-account", "s", false, "Switch AWS account before connecting")
	rdsConnectCmd.Flags().BoolVarP(&rdsListBastions, "list-bastions", "l", false, "List and select from available bastion hosts")
}

func runRDSConnect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	if err := validateLocalPort(localPort); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Track if we just authenticated (to avoid double-login with -s flag)
	justAuthenticated := false

	// Create RDS manager
	rdsManager, err := aws.NewRDSManager(ctx)
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
			justAuthenticated = true
			// Retry creating manager after successful login
			rdsManager, err = aws.NewRDSManager(ctx)
			if err != nil {
				fmt.Printf("Error creating RDS manager after re-authentication: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating RDS manager: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle account switching if requested (skip if we just authenticated)
	if switchAccount && !justAuthenticated {
		if err := handleAccountSwitch(ctx); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		// Recreate RDS manager with new credentials
		rdsManager, err = aws.NewRDSManager(ctx)
		if err != nil {
			fmt.Printf("Error creating RDS manager after account switch: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the RDS connect workflow
	if err := rdsManager.RunConnect(ctx, rdsInstanceName, int32(localPort), rdsListBastions); err != nil {
		fmt.Printf("\n✗ Error: %v\n", err)
		os.Exit(1)
	}
}
