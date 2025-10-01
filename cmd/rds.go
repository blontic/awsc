package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/internal/aws"
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

func init() {
	rootCmd.AddCommand(rdsCmd)
	rdsCmd.AddCommand(rdsConnectCmd)
	rdsConnectCmd.Flags().IntVar(&localPort, "local-port", 0, "Local port for port forwarding (defaults to RDS port)")
	rdsConnectCmd.Flags().StringVar(&rdsInstanceName, "name", "", "Name of the RDS instance to connect to directly")
}

func runRDSConnect(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create RDS manager
	rdsManager, err := aws.NewRDSManager(ctx)
	if err != nil {
		fmt.Printf("Error creating RDS manager: %v\n", err)
		return
	}

	// Run the RDS connect workflow
	if err := rdsManager.RunConnect(ctx, rdsInstanceName, int32(localPort)); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
