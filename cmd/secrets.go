package cmd

import (
	"context"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "AWS Secrets Manager operations",
}

var secretsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "List and view secrets from AWS Secrets Manager",
	Run:   runSecretsShowCommand,
}

var secretName string
var secretsSwitchAccount bool

func init() {
	secretsShowCmd.Flags().StringVar(&secretName, "name", "", "Name of the secret to show directly")
	secretsShowCmd.Flags().BoolVarP(&secretsSwitchAccount, "switch-account", "s", false, "Switch AWS account before showing secrets")
	secretsCmd.AddCommand(secretsShowCmd)
	rootCmd.AddCommand(secretsCmd)
}

func runSecretsShowCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	secretsManager, err := newManager(ctx, func(ctx context.Context) (*aws.SecretsManager, error) {
		return aws.NewSecretsManager(ctx)
	}, secretsSwitchAccount)
	exitOnError(err)

	exitOnError(secretsManager.RunShowSecrets(ctx, secretName))
}
