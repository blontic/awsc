package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/internal/aws"
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

	// Run the EC2 connect workflow
	if err := ec2Manager.RunConnect(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
