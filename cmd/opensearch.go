package cmd

import (
	"context"

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
var opensearchListBastions bool

func init() {
	rootCmd.AddCommand(opensearchCmd)
	opensearchCmd.AddCommand(opensearchConnectCmd)
	opensearchConnectCmd.Flags().IntVar(&opensearchLocalPort, "local-port", 443, "Local port for port forwarding (defaults to 443)")
	opensearchConnectCmd.Flags().StringVar(&opensearchDomainName, "name", "", "Name of the OpenSearch domain to connect to directly")
	opensearchConnectCmd.Flags().BoolVarP(&opensearchSwitchAccount, "switch-account", "s", false, "Switch AWS account before connecting")
	opensearchConnectCmd.Flags().BoolVarP(&opensearchListBastions, "list-bastions", "l", false, "List and select from available bastion hosts")
}

func runOpenSearchConnect(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	exitOnError(validateLocalPort(opensearchLocalPort))

	opensearchManager, err := newManager(ctx, func(ctx context.Context) (*aws.OpenSearchManager, error) {
		return aws.NewOpenSearchManager(ctx)
	}, opensearchSwitchAccount)
	exitOnError(err)

	exitOnError(opensearchManager.RunConnect(ctx, opensearchDomainName, int32(opensearchLocalPort), opensearchListBastions))
}
