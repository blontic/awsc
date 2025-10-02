package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

func TestSaveAndGetAccountCache(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Create test account data
	accountId1 := "123456789012"
	accountName1 := "Production Account"
	accountId2 := "210987654321"
	accountName2 := "Development Account"

	accounts := []types.AccountInfo{
		{
			AccountId:   &accountId1,
			AccountName: &accountName1,
		},
		{
			AccountId:   &accountId2,
			AccountName: &accountName2,
		},
	}

	// Save account cache
	err := SaveAccountCache(accounts)
	if err != nil {
		t.Errorf("SaveAccountCache failed: %v", err)
	}

	// Verify cache file exists
	cachePath := GetAccountCachePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Account cache file was not created")
	}

	// Test getting account names
	name1 := GetAccountName(accountId1)
	if name1 != accountName1 {
		t.Errorf("Expected account name %s, got %s", accountName1, name1)
	}

	name2 := GetAccountName(accountId2)
	if name2 != accountName2 {
		t.Errorf("Expected account name %s, got %s", accountName2, name2)
	}

	// Test unknown account ID returns the ID itself
	unknownId := "999999999999"
	name3 := GetAccountName(unknownId)
	if name3 != unknownId {
		t.Errorf("Expected unknown account ID %s, got %s", unknownId, name3)
	}
}

func TestGetAccountName_NoCache(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tempDir)

	// Test that non-existent cache returns account ID
	accountId := "123456789012"
	name := GetAccountName(accountId)
	if name != accountId {
		t.Errorf("Expected account ID %s when cache doesn't exist, got %s", accountId, name)
	}
}

func TestGetAccountCachePath(t *testing.T) {
	path := GetAccountCachePath()
	if path == "" {
		t.Error("GetAccountCachePath should return non-empty path")
	}

	if !filepath.IsAbs(path) {
		t.Error("GetAccountCachePath should return absolute path")
	}

	if filepath.Base(path) != "accounts.json" {
		t.Errorf("Expected path to end with accounts.json, got %s", path)
	}
}