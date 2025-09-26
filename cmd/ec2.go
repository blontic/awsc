package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/pkg/aws"
	"github.com/blontic/swa/pkg/ui"
	"github.com/spf13/cobra"
)

var ec2Cmd = &cobra.Command{
	Use:   "ec2",
	Short: "EC2 instance connections",
	Long:  `Connect to EC2 instances using SSM sessions`,
}

var ec2ConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to an EC2 instance via SSM session",
	Long:  `List EC2 instances and establish SSM session for remote shell access`,
	Run:   runEC2Connect,
}

func init() {
	rootCmd.AddCommand(ec2Cmd)
	ec2Cmd.AddCommand(ec2ConnectCmd)
}

func runEC2Connect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create EC2 manager
	ec2Manager, err := aws.NewEC2Manager(ctx)
	if err != nil {
		fmt.Printf("Error creating EC2 manager: %v\n", err)
		return
	}

	// List EC2 instances
	ec2Instances, err := ec2Manager.ListEC2Instances(ctx)
	if err != nil {
		// Handle authentication error
		if aws.IsAuthError(err) {
			if err := handleAuthenticationError(ctx); err != nil {
				fmt.Printf("Authentication failed: %v\n", err)
				return
			}
			
			// Recreate EC2 manager with new credentials
			ec2Manager, err = aws.NewEC2Manager(ctx)
			if err != nil {
				fmt.Printf("Error creating EC2 manager after authentication: %v\n", err)
				return
			}
			
			// Try to list EC2 instances again
			ec2Instances, err = ec2Manager.ListEC2Instances(ctx)
			if err != nil {
				fmt.Printf("Error listing EC2 instances after authentication: %v\n", err)
				return
			}
		} else {
			fmt.Printf("Error listing EC2 instances: %v\n", err)
			return
		}
	}

	if len(ec2Instances) == 0 {
		fmt.Println("No available EC2 instances found")
		return
	}

	// Create EC2 options for selection
	ec2Options := make([]string, len(ec2Instances))
	for i, instance := range ec2Instances {
		ec2Options[i] = fmt.Sprintf("%s (%s - %s)", 
			instance.Name, instance.InstanceId, instance.InstanceType)
	}

	// Interactive EC2 selection
	selectedEC2Index, err := ui.RunSelector("Select EC2 instance:", ec2Options)
	if err != nil {
		fmt.Printf("Error selecting EC2 instance: %v\n", err)
		return
	}
	if selectedEC2Index == -1 {
		return // User quit, exit gracefully
	}

	selectedEC2 := ec2Instances[selectedEC2Index]
	fmt.Printf("✓ Selected: %s\n", selectedEC2.Name)

	fmt.Printf("Connecting to %s via SSM...\n", selectedEC2.Name)

	// Start SSM session
	err = ec2Manager.StartSSMSession(ctx, selectedEC2.InstanceId)
	if err != nil {
		fmt.Printf("Error starting SSM session: %v\n", err)
		return
	}
}