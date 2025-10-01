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

var ec2RdpCmd = &cobra.Command{
	Use:   "rdp",
	Short: "Start RDP port forwarding to Windows EC2 instance",
	Long:  `List Windows EC2 instances and start RDP port forwarding on localhost:3389`,
	Run:   runEC2RDP,
}

var instanceId string

func init() {
	rootCmd.AddCommand(ec2Cmd)
	ec2Cmd.AddCommand(ec2ConnectCmd)
	ec2Cmd.AddCommand(ec2RdpCmd)

	// Add instance-id flag to both commands
	ec2ConnectCmd.Flags().StringVar(&instanceId, "instance-id", "", "EC2 instance ID to connect to (optional)")
	ec2RdpCmd.Flags().StringVar(&instanceId, "instance-id", "", "EC2 instance ID to connect to (optional)")
}

func createEC2Manager() (*aws.EC2Manager, error) {
	ctx := context.Background()
	return aws.NewEC2Manager(ctx)
}

func runEC2Connect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	ec2Manager, err := createEC2Manager()
	if err != nil {
		fmt.Printf("Error creating EC2 manager: %v\n", err)
		return
	}

	// Get instance-id flag value
	instanceIdFlag, _ := cmd.Flags().GetString("instance-id")

	if err := ec2Manager.RunConnect(ctx, instanceIdFlag); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func runEC2RDP(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	ec2Manager, err := createEC2Manager()
	if err != nil {
		fmt.Printf("Error creating EC2 manager: %v\n", err)
		return
	}

	// Get instance-id flag value
	instanceIdFlag, _ := cmd.Flags().GetString("instance-id")

	if err := ec2Manager.RunRDP(ctx, instanceIdFlag); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
