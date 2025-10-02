package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	swaconfig "github.com/blontic/swa/internal/config"
	"github.com/blontic/swa/internal/ui"
	"github.com/spf13/viper"
)

type SSOManager struct {
	client *sso.Client
}

func NewSSOManager(ctx context.Context) (*SSOManager, error) {
	cfg, err := swaconfig.LoadSWAConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &SSOManager{
		client: sso.NewFromConfig(cfg),
	}, nil
}

func (s *SSOManager) ListAccounts(ctx context.Context, accessToken string) ([]types.AccountInfo, error) {
	var allAccounts []types.AccountInfo
	var nextToken *string

	for {
		input := &sso.ListAccountsInput{
			AccessToken: &accessToken,
			NextToken:   nextToken,
		}

		result, err := s.client.ListAccounts(ctx, input)
		if err != nil {
			return nil, err
		}

		allAccounts = append(allAccounts, result.AccountList...)

		// Check if there are more pages
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return allAccounts, nil
}

func (s *SSOManager) ListRoles(ctx context.Context, accessToken, accountId string) ([]types.RoleInfo, error) {
	var allRoles []types.RoleInfo
	var nextToken *string

	for {
		input := &sso.ListAccountRolesInput{
			AccessToken: &accessToken,
			AccountId:   &accountId,
			NextToken:   nextToken,
		}

		result, err := s.client.ListAccountRoles(ctx, input)
		if err != nil {
			return nil, err
		}

		allRoles = append(allRoles, result.RoleList...)

		// Check if there are more pages
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return allRoles, nil
}

func (s *SSOManager) GetRoleCredentials(ctx context.Context, accessToken, accountId, roleName string) (*types.RoleCredentials, error) {
	input := &sso.GetRoleCredentialsInput{
		AccessToken: &accessToken,
		AccountId:   &accountId,
		RoleName:    &roleName,
	}

	result, err := s.client.GetRoleCredentials(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.RoleCredentials, nil
}

// RunLogin handles the complete SSO login workflow
func (s *SSOManager) RunLogin(ctx context.Context, force bool, accountName, roleName string) error {


	// Check if config exists
	if viper.GetString("sso.start_url") == "" {
		return fmt.Errorf("no SSO configuration found. Please run 'swa config init' first")
	}

	// Create credentials manager for authentication
	credentialsManager, err := NewCredentialsManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create credentials manager: %v", err)
	}

	// Try to get cached SSO token and use it if valid (unless force is true)
	if !force {
		accessToken, err := credentialsManager.GetCachedToken()
		if err == nil {
			// Try listing accounts to see if SSO token works
			accounts, listErr := s.ListAccounts(ctx, *accessToken)
			if listErr == nil && len(accounts) > 0 {
					// SSO token works, save account cache and proceed with account/role selection
				if err := swaconfig.SaveAccountCache(accounts); err != nil {
					// Don't fail login if cache save fails
					fmt.Printf("Warning: failed to save account cache: %v\n", err)
				}
				return s.handleAccountRoleSelection(ctx, *accessToken, accounts, accountName, roleName)
			}
		}
	}

	// If we get here, need to re-authenticate
	fmt.Printf("Starting SSO authentication...\n")

	// Try authentication
	startURL := viper.GetString("sso.start_url")
	ssoRegion := viper.GetString("sso.region")

	if err := credentialsManager.Authenticate(ctx, startURL, ssoRegion); err != nil {
		return fmt.Errorf("SSO authentication failed: %v", err)
	}

	// Get fresh access token
	accessToken, err := credentialsManager.GetCachedToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %v", err)
	}

	// List accounts for selection
	accounts, err := s.ListAccounts(ctx, *accessToken)
	if err != nil {
		return fmt.Errorf("error listing accounts: %v", err)
	}

	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Save account cache
	if err := swaconfig.SaveAccountCache(accounts); err != nil {
		// Don't fail login if cache save fails, just log it
		fmt.Printf("Warning: failed to save account cache: %v\n", err)
	}

	return s.handleAccountRoleSelection(ctx, *accessToken, accounts, accountName, roleName)
}

func (s *SSOManager) handleAccountRoleSelection(ctx context.Context, accessToken string, accounts []types.AccountInfo, accountName, roleName string) error {
	// Sort accounts alphabetically
	sort.Slice(accounts, func(i, j int) bool {
		return *accounts[i].AccountName < *accounts[j].AccountName
	})

	var selectedAccount types.AccountInfo
	var selectedAccountIndex int = -1

	// If account name provided, try to find exact match
	if accountName != "" {
		for i, account := range accounts {
			if strings.EqualFold(*account.AccountName, accountName) {
				selectedAccount = account
				selectedAccountIndex = i
				break
			}
		}
		if selectedAccountIndex == -1 {
			fmt.Printf("Account '%s' not found. Available accounts:\n\n", accountName)
			// Fall through to show account list
		} else {
			fmt.Printf("Found account: %s\n", *selectedAccount.AccountName)
		}
	}

	// Show account selection if no account specified or account not found
	if accountName == "" || selectedAccountIndex == -1 {
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
		selectedAccount = accounts[selectedAccountIndex]
	}
	fmt.Printf("Selected: %s\n", *selectedAccount.AccountName)

	// List roles
	roles, err := s.ListRoles(ctx, accessToken, *selectedAccount.AccountId)
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

	var selectedRole types.RoleInfo
	var selectedRoleIndex int = -1

	// If role name provided, try to find exact match
	if roleName != "" {
		for i, role := range roles {
			if strings.EqualFold(*role.RoleName, roleName) {
				selectedRole = role
				selectedRoleIndex = i
				break
			}
		}
		if selectedRoleIndex == -1 {
			fmt.Printf("Role '%s' not found in account %s. Available roles:\n\n", roleName, *selectedAccount.AccountName)
			// Fall through to show role list
		} else {
			fmt.Printf("Found role: %s\n", *selectedRole.RoleName)
		}
	}

	// Show role selection if no role specified or role not found
	if roleName == "" || selectedRoleIndex == -1 {
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
		selectedRole = roles[selectedRoleIndex]
	}
	fmt.Printf("Selected: %s\n", *selectedRole.RoleName)

	// Get credentials (AWS SSO automatically uses max duration for the role)
	creds, err := s.GetRoleCredentials(ctx, accessToken, *selectedAccount.AccountId, *selectedRole.RoleName)
	if err != nil {
		return fmt.Errorf("error getting role credentials: %v", err)
	}

	// Set up credentials
	err = SetupCredentials(*selectedAccount.AccountId, *selectedRole.RoleName, creds)
	if err != nil {
		return fmt.Errorf("error setting up credentials: %v", err)
	}

	fmt.Printf("Assumed role %s in account %s\n", *selectedRole.RoleName, *selectedAccount.AccountName)
	fmt.Printf("Credentials written to AWS config profile 'swa'\n")
	return nil
}
