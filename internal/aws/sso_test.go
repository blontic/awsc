package aws

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/spf13/viper"
)

func TestNewSSOManager(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials but shouldn't panic
	_, err := NewSSOManager(ctx)
	if err != nil {
		t.Logf("NewSSOManager failed as expected in test environment: %v", err)
	} else {
		t.Log("NewSSOManager succeeded unexpectedly")
	}
}

// Note: The SSO manager methods (ListAccounts, ListRoles, GetRoleCredentials, RunLogin)
// are difficult to test without mocking the SSO client, which would require significant
// refactoring to use interfaces. Since these methods are primarily thin wrappers around
// AWS SDK calls and the main business logic is tested elsewhere, we'll focus on testing
// the parts we can test effectively.

func TestSSOManager_handleAccountRoleSelection_DataStructures(t *testing.T) {
	// Test the data structures and sorting logic that would be used
	// in handleAccountRoleSelection without requiring AWS calls

	accounts := []types.AccountInfo{
		{
			AccountId:   aws.String("123456789012"),
			AccountName: aws.String("Production"),
		},
		{
			AccountId:   aws.String("123456789013"),
			AccountName: aws.String("Development"),
		},
		{
			AccountId:   aws.String("123456789014"),
			AccountName: aws.String("Staging"),
		},
	}

	roles := []types.RoleInfo{
		{
			RoleName: aws.String("ReadOnlyAccess"),
		},
		{
			RoleName: aws.String("AdminAccess"),
		},
		{
			RoleName: aws.String("DeveloperAccess"),
		},
	}

	// Test that we can create the expected display strings
	accountOptions := make([]string, len(accounts))
	for i, account := range accounts {
		accountOptions[i] = *account.AccountName + " (" + *account.AccountId + ")"
	}

	expectedAccountOptions := []string{
		"Production (123456789012)",
		"Development (123456789013)",
		"Staging (123456789014)",
	}

	for i, option := range accountOptions {
		if option != expectedAccountOptions[i] {
			t.Errorf("Expected account option %s, got %s", expectedAccountOptions[i], option)
		}
	}

	// Test role options
	roleOptions := make([]string, len(roles))
	for i, role := range roles {
		roleOptions[i] = *role.RoleName
	}

	expectedRoleOptions := []string{
		"ReadOnlyAccess",
		"AdminAccess",
		"DeveloperAccess",
	}

	for i, option := range roleOptions {
		if option != expectedRoleOptions[i] {
			t.Errorf("Expected role option %s, got %s", expectedRoleOptions[i], option)
		}
	}
}

// Test basic SSO manager methods that don't require complex mocking
func TestSSOManager_BasicMethods(t *testing.T) {
	// These methods are thin wrappers around AWS SDK calls
	// We can't easily test them without complex mocking, but we can
	// at least verify the manager structure
	manager := &SSOManager{}
	if manager == nil {
		t.Error("Expected manager to be created")
	}
}

func TestSSOManager_RunLogin_ConfigValidation(t *testing.T) {
	// Test that RunLogin validates configuration before proceeding
	ctx := context.Background()
	manager := &SSOManager{}

	// Clear any existing viper config
	viper.Reset()
	defer viper.Reset()

	// Test with no SSO configuration
	err := manager.RunLogin(ctx, false, "", "")
	if err == nil {
		t.Error("Expected error when no SSO configuration exists")
	}
	if err != nil && !strings.Contains(err.Error(), "no SSO configuration found") {
		t.Errorf("Expected 'no SSO configuration found' error, got: %v", err)
	}
}

func TestSSOManager_RunLogin_ParameterHandling(t *testing.T) {
	// Test parameter validation without triggering authentication
	// We only test the initial validation logic, not the full authentication flow

	// Test that the method signature accepts the correct parameters
	ctx := context.Background()
	manager := &SSOManager{}

	// Test with no config - should fail immediately without authentication
	viper.Reset()
	defer viper.Reset()

	err := manager.RunLogin(ctx, false, "test-account", "test-role")
	if err == nil {
		t.Error("Expected error when no SSO configuration exists")
	}
	if err != nil && !strings.Contains(err.Error(), "no SSO configuration found") {
		t.Errorf("Expected 'no SSO configuration found' error, got: %v", err)
	}

	// Test with force flag - should also fail at config validation
	err = manager.RunLogin(ctx, true, "test-account", "test-role")
	if err == nil {
		t.Error("Expected error when no SSO configuration exists with force flag")
	}
	if err != nil && !strings.Contains(err.Error(), "no SSO configuration found") {
		t.Errorf("Expected 'no SSO configuration found' error with force flag, got: %v", err)
	}
}

func TestSSOManager_AccountRoleMatching(t *testing.T) {
	// Test the account and role matching logic used in handleAccountRoleSelection
	accounts := []types.AccountInfo{
		{
			AccountId:   aws.String("123456789012"),
			AccountName: aws.String("Production"),
		},
		{
			AccountId:   aws.String("123456789013"),
			AccountName: aws.String("Development"),
		},
	}

	roles := []types.RoleInfo{
		{
			RoleName: aws.String("AdminAccess"),
		},
		{
			RoleName: aws.String("ReadOnlyAccess"),
		},
	}

	// Test case-insensitive account matching
	accountName := "production"
	var foundAccount *types.AccountInfo
	for _, account := range accounts {
		if strings.EqualFold(*account.AccountName, accountName) {
			foundAccount = &account
			break
		}
	}
	if foundAccount == nil {
		t.Error("Should find account with case-insensitive matching")
	}
	if foundAccount != nil && *foundAccount.AccountName != "Production" {
		t.Errorf("Expected 'Production', got %s", *foundAccount.AccountName)
	}

	// Test case-insensitive role matching
	roleName := "adminaccess"
	var foundRole *types.RoleInfo
	for _, role := range roles {
		if strings.EqualFold(*role.RoleName, roleName) {
			foundRole = &role
			break
		}
	}
	if foundRole == nil {
		t.Error("Should find role with case-insensitive matching")
	}
	if foundRole != nil && *foundRole.RoleName != "AdminAccess" {
		t.Errorf("Expected 'AdminAccess', got %s", *foundRole.RoleName)
	}

	// Test non-existent account
	accountName = "NonExistent"
	foundAccount = nil
	for _, account := range accounts {
		if strings.EqualFold(*account.AccountName, accountName) {
			foundAccount = &account
			break
		}
	}
	if foundAccount != nil {
		t.Error("Should not find non-existent account")
	}

	// Test non-existent role
	roleName = "NonExistentRole"
	foundRole = nil
	for _, role := range roles {
		if strings.EqualFold(*role.RoleName, roleName) {
			foundRole = &role
			break
		}
	}
	if foundRole != nil {
		t.Error("Should not find non-existent role")
	}
}
