package aws

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sso/types"
)

type CredentialsCache struct {
	AccessKeyId     string    `json:"accessKeyId"`
	SecretAccessKey string    `json:"secretAccessKey"`
	SessionToken    string    `json:"sessionToken"`
	ExpiresAt       time.Time `json:"expiresAt"`
	AccountId       string    `json:"accountId"`
	RoleName        string    `json:"roleName"`
}

func (s *SSOManager) GetCachedCredentials(ctx context.Context, accountId, roleName string) (*types.RoleCredentials, error) {
	// Check cache first
	if creds := s.loadCredentialsFromCache(accountId, roleName); creds != nil {
		if time.Now().Before(creds.ExpiresAt) {
			return &types.RoleCredentials{
				AccessKeyId:     &creds.AccessKeyId,
				SecretAccessKey: &creds.SecretAccessKey,
				SessionToken:    &creds.SessionToken,
				Expiration:      creds.ExpiresAt.Unix(),
			}, nil
		}
	}

	// Get fresh credentials
	creds, err := s.GetRoleCredentials(ctx, accountId, roleName)
	if err != nil {
		return nil, err
	}

	// Try to get max session duration for better caching
	maxDuration := time.Hour // Default fallback
	if duration, err := s.GetRoleMaxSessionDuration(ctx, accountId, roleName); err == nil {
		maxDuration = duration
	}

	// Cache credentials with proper expiration
	s.saveCredentialsToCache(accountId, roleName, creds, maxDuration)

	return creds, nil
}

func (s *SSOManager) loadCredentialsFromCache(accountId, roleName string) *CredentialsCache {
	cacheFile := s.getCredentialsCacheFile(accountId, roleName)
	
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var cache CredentialsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}

	return &cache
}

func (s *SSOManager) saveCredentialsToCache(accountId, roleName string, creds *types.RoleCredentials, maxDuration time.Duration) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	// Use 90% of max duration to ensure we refresh before expiration
	cacheDuration := time.Duration(float64(maxDuration) * 0.9)
	
	cache := CredentialsCache{
		AccessKeyId:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
		ExpiresAt:       time.Now().Add(cacheDuration),
		AccountId:       accountId,
		RoleName:        roleName,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	cacheFile := s.getCredentialsCacheFile(accountId, roleName)
	return ioutil.WriteFile(cacheFile, data, 0644)
}

func (s *SSOManager) getCredentialsCacheFile(accountId, roleName string) string {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	
	// Create unique filename for account+role combination
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s-%s", accountId, roleName)))
	filename := fmt.Sprintf("creds-%x.json", h.Sum(nil))
	
	return filepath.Join(cacheDir, filename)
}