package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/blontic/swa/internal/aws"
	"github.com/blontic/swa/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	
	// Check if config exists
	if viper.GetString("sso.start_url") == "" {
		fmt.Println("No SSO configuration found. Please run 'swa config init' first.")
		return
	}
	
	// Create SSO manager
	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		fmt.Printf("Error creating SSO manager: %v\n", err)
		return
	}
	
	// Try to list accounts first
	accounts, err := ssoManager.ListAccounts(ctx)
	if err != nil || forceAuth {
		// Handle authentication if needed
		if forceAuth {
			if err := clearSSOCache(); err != nil {
				fmt.Printf("Warning: Failed to clear SSO cache: %v\n", err)
			}
			fmt.Printf("Forcing re-authentication...\n")
		} else {
			fmt.Printf("Starting SSO authentication...\n")
		}
		
		// Authenticate using SSO
		startURL := viper.GetString("sso.start_url")
		ssoRegion := viper.GetString("sso.region")
		
		if err := ssoManager.Authenticate(ctx, startURL, ssoRegion); err != nil {
			fmt.Printf("SSO authentication failed: %v\n", err)
			return
		}
		
		fmt.Printf("Authentication completed!\n")
		
		// Try listing accounts again after authentication
		accounts, err = ssoManager.ListAccounts(ctx)
		if err != nil {
			fmt.Printf("Error listing accounts after authentication: %v\n", err)
			return
		}
	}

	if len(accounts) == 0 {
		fmt.Println("No accounts found. Make sure you're authenticated with AWS SSO.")
		return
	}

	// Sort accounts alphabetically by name
	sort.Slice(accounts, func(i, j int) bool {
		return *accounts[i].AccountName < *accounts[j].AccountName
	})

	// Create account options for selection
	accountOptions := make([]string, len(accounts))
	for i, account := range accounts {
		accountOptions[i] = fmt.Sprintf("%s (%s)", *account.AccountName, *account.AccountId)
	}

	// Interactive account selection
	selectedAccountIndex, err := ui.RunSelector("Select AWS Account:", accountOptions)
	if err != nil {
		fmt.Printf("Error selecting account: %v\n", err)
		return
	}
	if selectedAccountIndex == -1 {
		return // User quit, exit gracefully
	}

	selectedAccount := accounts[selectedAccountIndex]
	fmt.Printf("✓ Selected: %s\n", *selectedAccount.AccountName)

	// List roles for selected account
	roles, err := ssoManager.ListRoles(ctx, *selectedAccount.AccountId)
	if err != nil {
		fmt.Printf("Error listing roles: %v\n", err)
		return
	}

	if len(roles) == 0 {
		fmt.Println("No roles found for this account.")
		return
	}

	// Sort roles alphabetically by name
	sort.Slice(roles, func(i, j int) bool {
		return *roles[i].RoleName < *roles[j].RoleName
	})

	// Create role options for selection
	roleOptions := make([]string, len(roles))
	for i, role := range roles {
		roleOptions[i] = *role.RoleName
	}

	// Interactive role selection
	selectedRoleIndex, err := ui.RunSelector(fmt.Sprintf("Select role for %s:", *selectedAccount.AccountName), roleOptions)
	if err != nil {
		fmt.Printf("Error selecting role: %v\n", err)
		return
	}
	if selectedRoleIndex == -1 {
		return // User quit, exit gracefully
	}

	selectedRole := roles[selectedRoleIndex]
	fmt.Printf("✓ Selected: %s\n", *selectedRole.RoleName)

	// Get role credentials (with caching)
	creds, err := ssoManager.GetCachedCredentials(ctx, *selectedAccount.AccountId, *selectedRole.RoleName)
	if err != nil {
		fmt.Printf("Error getting role credentials: %v\n", err)
		return
	}

	// Set environment variables in current process
	os.Setenv("AWS_ACCESS_KEY_ID", *creds.AccessKeyId)
	os.Setenv("AWS_SECRET_ACCESS_KEY", *creds.SecretAccessKey)
	os.Setenv("AWS_SESSION_TOKEN", *creds.SessionToken)
	os.Setenv("AWS_DEFAULT_REGION", viper.GetString("default_region"))

	// Set up credentials in swa profile
	err = aws.SetupCredentials(*selectedAccount.AccountId, *selectedRole.RoleName, creds)
	if err != nil {
		fmt.Printf("Error setting up credentials: %v\n", err)
		return
	}

	fmt.Printf("Assumed role %s in account %s\n", *selectedRole.RoleName, *selectedAccount.AccountName)
	fmt.Println("Credentials saved to ~/.aws/config (swa profile)")
}





func clearSSOCache() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	return os.RemoveAll(cacheDir)
}

