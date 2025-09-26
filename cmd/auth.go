package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/blontic/swa/internal/aws"
	"github.com/blontic/swa/internal/ui"
	"github.com/spf13/viper"
)

// handleAuthenticationError provides shared authentication flow for all commands
func handleAuthenticationError(ctx context.Context) error {
	// Check if config exists
	if viper.GetString("sso.start_url") == "" {
		return fmt.Errorf("no SSO configuration found. Please run 'swa config init' first")
	}
	
	fmt.Printf("Starting SSO authentication...\n")
	
	// Create SSO manager
	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		return fmt.Errorf("error creating SSO manager: %v", err)
	}
	
	// Try authentication
	startURL := viper.GetString("sso.start_url")
	ssoRegion := viper.GetString("sso.region")
	
	if err := ssoManager.Authenticate(ctx, startURL, ssoRegion); err != nil {
		return fmt.Errorf("SSO authentication failed: %v", err)
	}
	
	// List accounts for selection
	accounts, err := ssoManager.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("error listing accounts: %v", err)
	}
	
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}
	
	// Sort accounts alphabetically
	sort.Slice(accounts, func(i, j int) bool {
		return *accounts[i].AccountName < *accounts[j].AccountName
	})
	
	// Create account options
	accountOptions := make([]string, len(accounts))
	for i, account := range accounts {
		accountOptions[i] = fmt.Sprintf("%s (%s)", *account.AccountName, *account.AccountId)
	}
	
	// Interactive account selection
	selectedAccountIndex, err := ui.RunSelector("Select AWS Account:", accountOptions)
	if err != nil {
		return fmt.Errorf("error selecting account: %v", err)
	}
	if selectedAccountIndex == -1 {
		return fmt.Errorf("no account selected")
	}
	
	selectedAccount := accounts[selectedAccountIndex]
	fmt.Printf("✓ Selected: %s\n", *selectedAccount.AccountName)
	
	// List roles
	roles, err := ssoManager.ListRoles(ctx, *selectedAccount.AccountId)
	if err != nil {
		return fmt.Errorf("error listing roles: %v", err)
	}
	
	if len(roles) == 0 {
		return fmt.Errorf("no roles found for this account")
	}
	
	// Sort roles alphabetically
	sort.Slice(roles, func(i, j int) bool {
		return *roles[i].RoleName < *roles[j].RoleName
	})
	
	// Create role options
	roleOptions := make([]string, len(roles))
	for i, role := range roles {
		roleOptions[i] = *role.RoleName
	}
	
	// Interactive role selection
	selectedRoleIndex, err := ui.RunSelector(fmt.Sprintf("Select role for %s:", *selectedAccount.AccountName), roleOptions)
	if err != nil {
		return fmt.Errorf("error selecting role: %v", err)
	}
	if selectedRoleIndex == -1 {
		return fmt.Errorf("no role selected")
	}
	
	selectedRole := roles[selectedRoleIndex]
	fmt.Printf("✓ Selected: %s\n", *selectedRole.RoleName)
	
	// Get credentials and set them up
	creds, err := ssoManager.GetRoleCredentials(ctx, *selectedAccount.AccountId, *selectedRole.RoleName)
	if err != nil {
		return fmt.Errorf("error getting role credentials: %v", err)
	}
	
	// Set up credentials
	err = aws.SetupCredentials(*selectedAccount.AccountId, *selectedRole.RoleName, creds)
	if err != nil {
		return fmt.Errorf("error setting up credentials: %v", err)
	}
	
	fmt.Printf("Assumed role %s in account %s\n", *selectedRole.RoleName, *selectedAccount.AccountName)
	return nil
}