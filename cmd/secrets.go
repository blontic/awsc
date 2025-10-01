package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/internal/aws"
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

func init() {
	secretsShowCmd.Flags().StringVar(&secretName, "name", "", "Name of the secret to show directly")
	secretsCmd.AddCommand(secretsShowCmd)
	rootCmd.AddCommand(secretsCmd)
}

func runSecretsShowCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create secrets manager
	secretsManager, err := aws.NewSecretsManager(ctx)
	if err != nil {
		fmt.Printf("Error creating secrets manager: %v\n", err)
		return
	}

	// Run the secrets show operation
	if err := secretsManager.RunShowSecrets(ctx, secretName); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
