package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
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
var rdpLocalPort int32
var ec2SwitchAccount bool

func init() {
	rootCmd.AddCommand(ec2Cmd)
	ec2Cmd.AddCommand(ec2ConnectCmd)
	ec2Cmd.AddCommand(ec2RdpCmd)

	// Add instance-id flag to both commands
	ec2ConnectCmd.Flags().StringVar(&instanceId, "instance-id", "", "EC2 instance ID to connect to (optional)")
	ec2RdpCmd.Flags().StringVar(&instanceId, "instance-id", "", "EC2 instance ID to connect to (optional)")
	ec2RdpCmd.Flags().Int32Var(&rdpLocalPort, "local-port", 3389, "Local port for RDP forwarding (default: 3389)")

	// Add switch-account flag to both commands
	ec2ConnectCmd.Flags().BoolVarP(&ec2SwitchAccount, "switch-account", "s", false, "Switch AWS account before connecting")
	ec2RdpCmd.Flags().BoolVarP(&ec2SwitchAccount, "switch-account", "s", false, "Switch AWS account before connecting")
}

func createEC2Manager() (*aws.EC2Manager, error) {
	ctx := context.Background()
	return aws.NewEC2Manager(ctx)
}

func runEC2Connect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Handle account switching if requested
	if ec2SwitchAccount {
		if err := handleAccountSwitch(ctx); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	}

	ec2Manager, err := createEC2Manager()
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
			ec2Manager, err = createEC2Manager()
			if err != nil {
				fmt.Printf("Error creating EC2 manager after re-authentication: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating EC2 manager: %v\n", err)
			os.Exit(1)
		}
	}

	// Get instance-id flag value
	instanceIdFlag, _ := cmd.Flags().GetString("instance-id")

	if err := ec2Manager.RunConnect(ctx, instanceIdFlag); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runEC2RDP(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Handle account switching if requested
	if ec2SwitchAccount {
		if err := handleAccountSwitch(ctx); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	}

	ec2Manager, err := createEC2Manager()
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
			ec2Manager, err = createEC2Manager()
			if err != nil {
				fmt.Printf("Error creating EC2 manager after re-authentication: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating EC2 manager: %v\n", err)
			os.Exit(1)
		}
	}

	// Get flag values
	instanceIdFlag, _ := cmd.Flags().GetString("instance-id")
	localPortFlag, _ := cmd.Flags().GetInt32("local-port")

	if err := ec2Manager.RunRDP(ctx, instanceIdFlag, localPortFlag); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
