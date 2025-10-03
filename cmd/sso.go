package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with AWS SSO and select account/role",
	Long:  `Authenticate with AWS SSO, list available accounts and roles, and set up credentials`,
	Run:   runSSOLogin,
}

var forceAuth bool
var accountName string
var roleName string

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().BoolVar(&forceAuth, "force", false, "Force re-authentication by clearing cached tokens")
	loginCmd.Flags().StringVar(&accountName, "account", "", "Account name to connect to (optional)")
	loginCmd.Flags().StringVar(&roleName, "role", "", "Role name to assume (optional)")
}

func runSSOLogin(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Create SSO manager and run login
	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if err := ssoManager.RunLogin(ctx, forceAuth, accountName, roleName); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
