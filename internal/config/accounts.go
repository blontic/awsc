package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

type AccountCache struct {
	Accounts map[string]string `json:"accounts"` // accountId -> accountName
}

func GetAccountCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".awsc", "accounts.json")
}

// SaveAccountCache saves account ID to name mappings
func SaveAccountCache(accounts []types.AccountInfo) error {
	cache := AccountCache{
		Accounts: make(map[string]string),
	}

	for _, account := range accounts {
		if account.AccountId != nil && account.AccountName != nil {
			cache.Accounts[*account.AccountId] = *account.AccountName
		}
	}

	// Create .awsc directory if it doesn't exist with secure permissions
	cacheDir := filepath.Dir(GetAccountCachePath())
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	// Write cache file
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(GetAccountCachePath(), data, 0600)
}

// GetAccountName returns account name for given account ID, or the ID if not found
func GetAccountName(accountId string) string {
	cachePath := GetAccountCachePath()
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return accountId // Cache doesn't exist, return ID
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return accountId // Error reading cache, return ID
	}

	var cache AccountCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return accountId // Error parsing cache, return ID
	}

	if name, exists := cache.Accounts[accountId]; exists {
		return name
	}

	return accountId // Account not found in cache, return ID
}
