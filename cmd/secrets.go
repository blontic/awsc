package cmd

import (
	"context"
	"fmt"
	"os"

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
			secretsManager, err = aws.NewSecretsManager(ctx)
			if err != nil {
				fmt.Printf("Error creating secrets manager after re-authentication: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating secrets manager: %v\n", err)
			os.Exit(1)
		}
	}

	// Run the secrets show operation
	if err := secretsManager.RunShowSecrets(ctx, secretName); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
