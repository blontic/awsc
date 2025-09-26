package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/pkg/aws"
	"github.com/blontic/swa/pkg/ui"
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

func init() {
	rootCmd.AddCommand(rdsCmd)
	rdsCmd.AddCommand(rdsConnectCmd)
	rdsConnectCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port for port forwarding (defaults to RDS port)")
}

func runRDSConnect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create RDS manager
	rdsManager, err := aws.NewRDSManager(ctx)
	if err != nil {
		fmt.Printf("Error creating RDS manager: %v\n", err)
		return
	}

	// List RDS instances
	rdsInstances, err := rdsManager.ListRDSInstances(ctx)
	if err != nil {
		// Handle authentication error
		if aws.IsAuthError(err) {
			if err := handleAuthenticationError(ctx); err != nil {
				fmt.Printf("Authentication failed: %v\n", err)
				return
			}
			
			// Recreate RDS manager with new credentials
			rdsManager, err = aws.NewRDSManager(ctx)
			if err != nil {
				fmt.Printf("Error creating RDS manager after authentication: %v\n", err)
				return
			}
			
			// Try to list RDS instances again
			rdsInstances, err = rdsManager.ListRDSInstances(ctx)
			if err != nil {
				fmt.Printf("Error listing RDS instances after authentication: %v\n", err)
				return
			}
		} else {
			fmt.Printf("Error listing RDS instances: %v\n", err)
			return
		}
	}

	if len(rdsInstances) == 0 {
		fmt.Println("No available RDS instances found")
		return
	}

	// Create RDS options for selection
	rdsOptions := make([]string, len(rdsInstances))
	for i, instance := range rdsInstances {
		rdsOptions[i] = fmt.Sprintf("%s (%s:%d - %s)", 
			instance.Identifier, instance.Endpoint, instance.Port, instance.Engine)
	}

	// Interactive RDS selection
	selectedRDSIndex, err := ui.RunSelector("Select RDS instance:", rdsOptions)
	if err != nil {
		fmt.Printf("Error selecting RDS instance: %v\n", err)
		return
	}
	if selectedRDSIndex == -1 {
		return // User quit, exit gracefully
	}

	selectedRDS := rdsInstances[selectedRDSIndex]
	fmt.Printf("✓ Selected: %s\n", selectedRDS.Identifier)

	// Find bastion hosts
	bastionHosts, err := rdsManager.FindBastionHosts(ctx, selectedRDS)
	if err != nil {
		fmt.Printf("Error finding bastion hosts: %v\n", err)
		return
	}

	if len(bastionHosts) == 0 {
		fmt.Printf("No suitable bastion hosts found for RDS instance %s\n", selectedRDS.Identifier)
		fmt.Println("Make sure you have EC2 instances with proper security group access to the RDS instance")
		return
	}

	// Create bastion options for selection
	bastionOptions := make([]string, len(bastionHosts))
	for i, bastion := range bastionHosts {
		bastionOptions[i] = fmt.Sprintf("%s (%s)", bastion.Name, bastion.InstanceId)
	}

	// Interactive bastion selection
	selectedBastionIndex, err := ui.RunSelector("Select bastion host:", bastionOptions)
	if err != nil {
		fmt.Printf("Error selecting bastion host: %v\n", err)
		return
	}
	if selectedBastionIndex == -1 {
		return // User quit, exit gracefully
	}

	selectedBastion := bastionHosts[selectedBastionIndex]
	fmt.Printf("✓ Selected: %s\n", selectedBastion.Name)

	// Determine local port
	targetLocalPort := selectedRDS.Port
	if localPort != 0 {
		targetLocalPort = int32(localPort)
	}

	fmt.Printf("Connecting to %s via %s...\n", selectedRDS.Identifier, selectedBastion.Name)

	// Start port forwarding
	err = rdsManager.StartPortForwarding(ctx, selectedBastion.InstanceId, 
		selectedRDS.Endpoint, selectedRDS.Port, targetLocalPort)
	if err != nil {
		fmt.Printf("Error starting port forwarding: %v\n", err)
		return
	}
}

