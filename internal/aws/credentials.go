package aws

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"

	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/spf13/viper"
)

type CredentialsManager struct {
	oidcClient *ssooidc.Client
	ssoManager *SSOManager
}

type SSOCache struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Region      string    `json:"region"`
	StartURL    string    `json:"startUrl"`
}

func NewCredentialsManager(ctx context.Context) (*CredentialsManager, error) {
	cfg, err := awscconfig.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	ssoManager, err := NewSSOManager(ctx)
	if err != nil {
		return nil, err
	}

	return &CredentialsManager{
		oidcClient: ssooidc.NewFromConfig(cfg),
		ssoManager: ssoManager,
	}, nil
}

// IsAuthError checks if an error is related to authentication/credentials
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	// Don't treat permission errors as auth errors
	if contains(errorStr, "is not authorized to perform") {
		return false
	}
	return contains(errorStr, "no active session") ||
		contains(errorStr, "failed to get shared config profile") ||
		contains(errorStr, "AuthFailure") ||
		contains(errorStr, "SignatureDoesNotMatch") ||
		contains(errorStr, "TokenRefreshRequired") ||
		contains(errorStr, "ExpiredToken") ||
		contains(errorStr, "InvalidToken") ||
		contains(errorStr, "get credentials") ||
		contains(errorStr, "no EC2 IMDS role found") ||
		contains(errorStr, "failed to refresh cached credentials") ||
		contains(errorStr, "no such host") || // DNS resolution failure due to missing region
		contains(errorStr, "failed to load AWS config")
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func (c *CredentialsManager) GetCachedToken() (*string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")

	// Get the start URL to find the correct cache file
	startURL := viper.GetString("sso.start_url")
	if startURL == "" {
		return nil, fmt.Errorf("no SSO start URL configured")
	}

	// Create cache filename based on start URL (same logic as saveTokenToCache)
	h := sha1.New()
	h.Write([]byte(startURL))
	filename := fmt.Sprintf("%x.json", h.Sum(nil))
	cacheFile := filepath.Join(cacheDir, filename)

	// Check if the specific cache file exists
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("no SSO cache found for this start URL, please run 'awsc login'")
	}

	var cache SSOCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Return token without checking expiration - let API calls fail naturally
	return &cache.AccessToken, nil
}

func (c *CredentialsManager) Authenticate(ctx context.Context, startURL, ssoRegion string) error {
	// Register client
	registerResp, err := c.oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("awsc"),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return fmt.Errorf("failed to register client: %v", err)
	}

	// Start device authorization
	deviceResp, err := c.oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerResp.ClientId,
		ClientSecret: registerResp.ClientSecret,
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %v", err)
	}

	// Open browser
	fmt.Printf("Opening browser to: %s\n", *deviceResp.VerificationUriComplete)
	fmt.Printf("If browser doesn't open, visit: %s\n", *deviceResp.VerificationUriComplete)
	fmt.Printf("And enter code: %s\n", *deviceResp.UserCode)

	if err := openBrowser(*deviceResp.VerificationUriComplete); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
	}

	// Poll for token with timeout
	timeoutMinutes := int(deviceResp.ExpiresIn / 60)
	fmt.Printf("Waiting for authentication (timeout in %d minutes)...\n", timeoutMinutes)
	timeout := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
	interval := time.Duration(deviceResp.Interval) * time.Second

	for time.Now().Before(timeout) {
		tokenResp, err := c.oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     registerResp.ClientId,
			ClientSecret: registerResp.ClientSecret,
			DeviceCode:   deviceResp.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})

		if err != nil {
			// Check if we should continue polling
			if isRetryableError(err) {
				fmt.Print(".")
				time.Sleep(interval)
				continue
			}
			return fmt.Errorf("failed to create token: %v", err)
		}

		// Success! Save token to cache
		fmt.Println("\nAuthentication successful!")
		if err := c.saveTokenToCache(startURL, ssoRegion, tokenResp.AccessToken, &tokenResp.ExpiresIn); err != nil {
			return fmt.Errorf("failed to save token: %v", err)
		}

		return nil
	}

	return fmt.Errorf("authentication timed out - please try again")

	return nil
}

func (c *CredentialsManager) saveTokenToCache(startURL, ssoRegion string, accessToken *string, expiresIn *int32) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	// Create cache filename based on start URL
	h := sha1.New()
	h.Write([]byte(startURL))
	filename := fmt.Sprintf("%x.json", h.Sum(nil))
	cacheFile := filepath.Join(cacheDir, filename)

	// Create cache entry
	cache := SSOCache{
		AccessToken: *accessToken,
		ExpiresAt:   time.Now().Add(time.Duration(*expiresIn) * time.Second),
		Region:      ssoRegion,
		StartURL:    startURL,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0600)
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func isRetryableError(err error) bool {
	// Check for authorization_pending or slow_down errors
	errorStr := err.Error()
	return strings.Contains(errorStr, "AuthorizationPendingException") ||
		strings.Contains(errorStr, "authorization_pending") ||
		strings.Contains(errorStr, "SlowDownException") ||
		strings.Contains(errorStr, "slow_down")
}

// PromptForReauth asks the user if they want to re-authenticate and runs login if yes
func PromptForReauth(ctx context.Context) (bool, error) {
	// Check if this is a "no active session" error
	cfg, loadErr := awscconfig.LoadAWSConfigWithProfile(ctx)
	_ = cfg // Unused, just checking the error

	if loadErr != nil && strings.Contains(loadErr.Error(), "no active session") {
		fmt.Fprintf(os.Stderr, "No active session found. Please login first.\n")

		// Auto-trigger login
		ssoManager, err := NewSSOManager(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to create SSO manager: %w", err)
		}

		if err := ssoManager.RunLogin(ctx, false, "", ""); err != nil {
			return false, fmt.Errorf("authentication failed: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Authentication successful. Retrying operation...\n")
		return true, nil
	}

	// Check if this is a config issue
	if viper.GetString("sso.start_url") == "" {
		fmt.Fprintf(os.Stderr, "No SSO configuration found. Set up configuration? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return false, nil
		}

		// Initialize config
		if err := awscconfig.InitializeConfig(); err != nil {
			return false, fmt.Errorf("failed to initialize config: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Configuration complete. Now authenticating...\n")
	} else {
		fmt.Fprintf(os.Stderr, "Credentials expired. Re-authenticate? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			return false, nil
		}
	}

	// Run login automatically
	ssoManager, err := NewSSOManager(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to create SSO manager: %w", err)
	}

	if err := ssoManager.RunLogin(ctx, false, "", ""); err != nil {
		return false, fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authentication successful. Retrying operation...\n")
	return true, nil
}
