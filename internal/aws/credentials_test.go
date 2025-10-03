package aws

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/spf13/viper"
)

func TestNewCredentialsManager(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials but shouldn't panic
	_, err := NewCredentialsManager(ctx)
	if err != nil {
		t.Logf("NewCredentialsManager failed as expected in test environment: %v", err)
	} else {
		t.Log("NewCredentialsManager succeeded unexpectedly")
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "auth failure error",
			err:      fmt.Errorf("AuthFailure: invalid credentials"),
			expected: true,
		},
		{
			name:     "signature error",
			err:      fmt.Errorf("SignatureDoesNotMatch: signature mismatch"),
			expected: true,
		},
		{
			name:     "expired token error",
			err:      fmt.Errorf("ExpiredToken: token has expired"),
			expected: true,
		},
		{
			name:     "invalid token error",
			err:      fmt.Errorf("InvalidToken: token is invalid"),
			expected: true,
		},
		{
			name:     "get credentials error",
			err:      fmt.Errorf("failed to get credentials"),
			expected: true,
		},
		{
			name:     "permission error (not auth error)",
			err:      fmt.Errorf("User is not authorized to perform action"),
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "substring found",
			s:        "hello world",
			substr:   "world",
			expected: true,
		},
		{
			name:     "substring not found",
			s:        "hello world",
			substr:   "foo",
			expected: false,
		},
		{
			name:     "empty substring",
			s:        "hello world",
			substr:   "",
			expected: true,
		},
		{
			name:     "empty string",
			s:        "",
			substr:   "foo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetupCredentials(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test credentials
	creds := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIATEST123456789"),
		SecretAccessKey: aws.String("test-secret-key"),
		SessionToken:    aws.String("test-session-token"),
	}

	err := SetupCredentials("123456789012", "TestRole", creds)
	if err != nil {
		t.Fatalf("SetupCredentials failed: %v", err)
	}

	// Verify config file was created
	configFile := filepath.Join(tempDir, ".aws", "config")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Read and verify config content
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "[profile awsc]") {
		t.Error("Config file does not contain awsc profile")
	}
	if !strings.Contains(configStr, "AKIATEST123456789") {
		t.Error("Config file does not contain access key")
	}
	if !strings.Contains(configStr, "test-secret-key") {
		t.Error("Config file does not contain secret key")
	}
	if !strings.Contains(configStr, "test-session-token") {
		t.Error("Config file does not contain session token")
	}
}

func TestWriteAWSProfile(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test credentials
	creds := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIATEST123456789"),
		SecretAccessKey: aws.String("test-secret-key"),
		SessionToken:    aws.String("test-session-token"),
	}

	// Test writing new profile
	err := writeAWSProfile("test-profile", "123456789012", "TestRole", creds)
	if err != nil {
		t.Fatalf("writeAWSProfile failed: %v", err)
	}

	// Verify config file content
	configFile := filepath.Join(tempDir, ".aws", "config")
	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr := string(content)
	if !strings.Contains(configStr, "[profile test-profile]") {
		t.Error("Config file does not contain test-profile")
	}

	// Test overwriting existing profile
	newCreds := &types.RoleCredentials{
		AccessKeyId:     aws.String("AKIANEW123456789"),
		SecretAccessKey: aws.String("new-secret-key"),
		SessionToken:    aws.String("new-session-token"),
	}

	err = writeAWSProfile("test-profile", "123456789012", "NewRole", newCreds)
	if err != nil {
		t.Fatalf("writeAWSProfile overwrite failed: %v", err)
	}

	// Verify old credentials are replaced
	content, err = os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	configStr = string(content)
	if strings.Contains(configStr, "AKIATEST123456789") {
		t.Error("Old credentials still present in config file")
	}
	if !strings.Contains(configStr, "AKIANEW123456789") {
		t.Error("New credentials not found in config file")
	}
}

func TestCredentialsManager_GetCachedToken(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Set up SSO start URL for test
	viper.Set("sso.start_url", "https://test.awsapps.com/start")
	defer viper.Reset()

	manager := &CredentialsManager{}

	// Test no cache directory
	_, err := manager.GetCachedToken()
	if err == nil {
		t.Error("Expected error when no cache directory exists")
	}

	// Create cache directory and valid token file
	cacheDir := filepath.Join(tempDir, ".aws", "sso", "cache")
	err = os.MkdirAll(cacheDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create valid cache file with correct filename based on start URL
	startURL := "https://test.awsapps.com/start"
	cache := SSOCache{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Region:      "us-east-1",
		StartURL:    startURL,
	}

	cacheData, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}

	// Create filename using same logic as saveTokenToCache
	h := sha1.New()
	h.Write([]byte(startURL))
	filename := fmt.Sprintf("%x.json", h.Sum(nil))
	cacheFile := filepath.Join(cacheDir, filename)
	err = ioutil.WriteFile(cacheFile, cacheData, 0600)
	if err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}

	// Test getting valid token
	token, err := manager.GetCachedToken()
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}
	if *token != "valid-token" {
		t.Errorf("Expected 'valid-token', got %s", *token)
	}

	// Test expired token - should still return token (no expiration check)
	expiredCache := SSOCache{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().Add(-1 * time.Hour),
		Region:      "us-east-1",
		StartURL:    "https://test.awsapps.com/start",
	}

	expiredData, err := json.Marshal(expiredCache)
	if err != nil {
		t.Fatalf("Failed to marshal expired cache: %v", err)
	}

	err = ioutil.WriteFile(cacheFile, expiredData, 0600)
	if err != nil {
		t.Fatalf("Failed to write expired cache file: %v", err)
	}

	// Should return token even if expired - let API calls handle expiration
	token, err = manager.GetCachedToken()
	if err != nil {
		t.Fatalf("GetCachedToken should not fail for expired token: %v", err)
	}
	if *token != "expired-token" {
		t.Errorf("Expected 'expired-token', got %s", *token)
	}
}

func TestCredentialsManager_saveTokenToCache(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager := &CredentialsManager{}

	startURL := "https://test.awsapps.com/start"
	ssoRegion := "us-east-1"
	accessToken := "test-access-token"
	expiresIn := int32(3600)

	err := manager.saveTokenToCache(startURL, ssoRegion, &accessToken, &expiresIn)
	if err != nil {
		t.Fatalf("saveTokenToCache failed: %v", err)
	}

	// Verify cache file was created
	cacheDir := filepath.Join(tempDir, ".aws", "sso", "cache")
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache directory: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 cache file, got %d", len(files))
	}

	// Read and verify cache content
	cacheFile := filepath.Join(cacheDir, files[0].Name())
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var cache SSOCache
	err = json.Unmarshal(data, &cache)
	if err != nil {
		t.Fatalf("Failed to unmarshal cache: %v", err)
	}

	if cache.AccessToken != accessToken {
		t.Errorf("Expected access token %s, got %s", accessToken, cache.AccessToken)
	}
	if cache.Region != ssoRegion {
		t.Errorf("Expected region %s, got %s", ssoRegion, cache.Region)
	}
	if cache.StartURL != startURL {
		t.Errorf("Expected start URL %s, got %s", startURL, cache.StartURL)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "authorization pending",
			err:      fmt.Errorf("AuthorizationPendingException: authorization pending"),
			expected: true,
		},
		{
			name:     "slow down",
			err:      fmt.Errorf("SlowDownException: slow down"),
			expected: true,
		},
		{
			name:     "authorization_pending",
			err:      fmt.Errorf("authorization_pending"),
			expected: true,
		},
		{
			name:     "slow_down",
			err:      fmt.Errorf("slow_down"),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestCheckCredentialsValid(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials
	err := CheckCredentialsValid(ctx)
	if err != nil {
		t.Logf("CheckCredentialsValid failed as expected in test environment: %v", err)
	} else {
		t.Log("CheckCredentialsValid succeeded unexpectedly")
	}
}

func TestCheckAWSSession(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials
	err := CheckAWSSession(ctx)
	if err != nil {
		t.Logf("CheckAWSSession failed as expected in test environment: %v", err)
	} else {
		t.Log("CheckAWSSession succeeded unexpectedly")
	}
}

func TestOpenBrowser(t *testing.T) {
	// Test that openBrowser doesn't panic with invalid URL
	err := openBrowser("invalid-url")
	// We don't check for specific error as it varies by OS
	// Just verify it doesn't panic
	if err != nil {
		t.Logf("openBrowser failed as expected: %v", err)
	}
}

func TestCredentialsManager_GetCachedToken_InvalidJSON(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	manager := &CredentialsManager{}

	// Create cache directory and invalid JSON file
	cacheDir := filepath.Join(tempDir, ".aws", "sso", "cache")
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	// Create invalid JSON cache file
	cacheFile := filepath.Join(cacheDir, "invalid.json")
	err = os.WriteFile(cacheFile, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid cache file: %v", err)
	}

	// Test getting token with invalid JSON
	_, err = manager.GetCachedToken()
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
