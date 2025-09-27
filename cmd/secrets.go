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

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List and view secrets from AWS Secrets Manager",
	Run:   runSecretsListCommand,
}

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	rootCmd.AddCommand(secretsCmd)
}

func runSecretsListCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create secrets manager
	secretsManager, err := aws.NewSecretsManager(ctx)
	if err != nil {
		fmt.Printf("Error creating secrets manager: %v\n", err)
		return
	}

	// Run the secrets list operation
	if err := secretsManager.RunListSecrets(ctx); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
