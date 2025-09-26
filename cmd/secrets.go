package cmd

import (
	"context"
	"fmt"

	"github.com/blontic/swa/internal/aws"
	"github.com/blontic/swa/internal/ui"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "AWS Secrets Manager operations",
}

var secretsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List and view secrets from AWS Secrets Manager",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		// Create secrets manager
		secretsManager, err := aws.NewSecretsManager(ctx)
		if err != nil {
			fmt.Printf("Error creating secrets manager: %v\n", err)
			return
		}

		// List secrets
		secrets, err := secretsManager.ListSecrets(ctx)
		if err != nil {
			// Handle authentication error
			if aws.IsAuthError(err) {
				if err := handleAuthenticationError(ctx); err != nil {
					fmt.Printf("Authentication failed: %v\n", err)
					return
				}
				
				// Recreate secrets manager with new credentials
				secretsManager, err = aws.NewSecretsManager(ctx)
				if err != nil {
					fmt.Printf("Error creating secrets manager after authentication: %v\n", err)
					return
				}
				
				// Try to list secrets again
				secrets, err = secretsManager.ListSecrets(ctx)
				if err != nil {
					fmt.Printf("Error listing secrets after authentication: %v\n", err)
					return
				}
			} else {
				fmt.Printf("Error listing secrets: %v\n", err)
				return
			}
		}

		if len(secrets) == 0 {
			fmt.Printf("No secrets found in this account\n")
			return
		}

		// Create selection choices
		var choices []string
		for _, secret := range secrets {
			description := secret.Description
			if description == "" {
				description = "No description"
			}
			choices = append(choices, fmt.Sprintf("%s - %s", secret.Name, description))
		}

		// Let user select a secret
		selectedIndex, err := ui.RunSelector("Select a secret to view:", choices)
		if err != nil {
			fmt.Printf("Error selecting secret: %v\n", err)
			return
		}

		if selectedIndex == -1 {
			return // User quit
		}

		selectedSecret := secrets[selectedIndex].Name
		fmt.Printf("✓ Selected: %s\n", selectedSecret)

		// Get secret value
		secretValue, err := secretsManager.GetSecretValue(ctx, selectedSecret)
		if err != nil {
			fmt.Printf("Error getting secret value: %v\n", err)
			return
		}

		// Display the secret
		secretsManager.DisplaySecret(ctx, selectedSecret, secretValue)
	},
}

func init() {
	secretsCmd.AddCommand(secretsListCmd)
	rootCmd.AddCommand(secretsCmd)
}