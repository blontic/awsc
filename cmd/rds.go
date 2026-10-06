package cmd

import (
	"context"

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
	exitOnError(validateLocalPort(localPort))

	rdsManager, err := newManager(ctx, func(ctx context.Context) (*aws.RDSManager, error) {
		return aws.NewRDSManager(ctx)
	}, switchAccount)
	exitOnError(err)

	exitOnError(rdsManager.RunConnect(ctx, rdsInstanceName, int32(localPort), rdsListBastions))
}
