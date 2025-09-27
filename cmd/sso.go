package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/swa/internal/aws"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with AWS SSO and select account/role",
	Long:  `Authenticate with AWS SSO, list available accounts and roles, and set up credentials`,
	Run:   runSSOLogin,
}

var forceAuth bool

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().BoolVar(&forceAuth, "force", false, "Force re-authentication by clearing cached tokens")
}

func runSSOLogin(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create SSO manager and run login
	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := ssoManager.RunLogin(ctx, forceAuth); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
