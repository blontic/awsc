package aws

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"

	"github.com/spf13/viper"
)

type SSOManager struct {
	client     *sso.Client
	oidcClient *ssooidc.Client
}

type SSOCache struct {
	AccessToken string    `json:"accessToken"`
	ExpiresAt   time.Time `json:"expiresAt"`
	Region      string    `json:"region"`
	StartURL    string    `json:"startUrl"`
}

func NewSSOManager(ctx context.Context) (*SSOManager, error) {
	// Load config with SSO region (uses --region override if provided)
	var cfg aws.Config
	var err error
	
	// Use region override if provided, otherwise use SSO region from config
	region := viper.GetString("default_region")
	if region == "" {
		region = viper.GetString("sso.region")
	}
	
	if region != "" {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	} else {
		cfg, err = config.LoadDefaultConfig(ctx)
	}
	
	if err != nil {
		return nil, err
	}

	return &SSOManager{
		client:     sso.NewFromConfig(cfg),
		oidcClient: ssooidc.NewFromConfig(cfg),
	}, nil
}

func (s *SSOManager) GetCachedToken() (*string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	files, err := ioutil.ReadDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("no SSO cache found, please run 'aws sso login' first")
	}

	// Find the most recent cache file
	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		if file.ModTime().After(latestTime) {
			latestTime = file.ModTime()
			latestFile = file.Name()
		}
	}

	if latestFile == "" {
		return nil, fmt.Errorf("no valid SSO cache files found")
	}

	cacheFile := filepath.Join(cacheDir, latestFile)
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var cache SSOCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// Check if token is expired
	if time.Now().After(cache.ExpiresAt) {
		return nil, fmt.Errorf("SSO token expired, please run 'aws sso login'")
	}

	return &cache.AccessToken, nil
}

func (s *SSOManager) ListAccounts(ctx context.Context) ([]types.AccountInfo, error) {
	token, err := s.GetCachedToken()
	if err != nil {
		return nil, err
	}

	input := &sso.ListAccountsInput{
		AccessToken: token,
	}

	result, err := s.client.ListAccounts(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.AccountList, nil
}

func (s *SSOManager) ListRoles(ctx context.Context, accountId string) ([]types.RoleInfo, error) {
	token, err := s.GetCachedToken()
	if err != nil {
		return nil, err
	}

	input := &sso.ListAccountRolesInput{
		AccessToken: token,
		AccountId:   &accountId,
	}

	result, err := s.client.ListAccountRoles(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.RoleList, nil
}

func (s *SSOManager) GetRoleCredentials(ctx context.Context, accountId, roleName string) (*types.RoleCredentials, error) {
	token, err := s.GetCachedToken()
	if err != nil {
		return nil, err
	}

	input := &sso.GetRoleCredentialsInput{
		AccessToken: token,
		AccountId:   &accountId,
		RoleName:    &roleName,
	}

	result, err := s.client.GetRoleCredentials(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.RoleCredentials, nil
}

func (s *SSOManager) Authenticate(ctx context.Context, startURL, ssoRegion string) error {
	// Register client
	registerResp, err := s.oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("swa"),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return fmt.Errorf("failed to register client: %v", err)
	}

	// Start device authorization
	deviceResp, err := s.oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
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

	// Poll for token
	fmt.Println("Waiting for authentication...")
	for {
		tokenResp, err := s.oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     registerResp.ClientId,
			ClientSecret: registerResp.ClientSecret,
			DeviceCode:   deviceResp.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		
		if err != nil {
			// Check if we should continue polling
			if isRetryableError(err) {
				time.Sleep(time.Duration(deviceResp.Interval) * time.Second)
				continue
			}
			return fmt.Errorf("failed to create token: %v", err)
		}

		// Save token to cache
		if err := s.saveTokenToCache(startURL, ssoRegion, tokenResp.AccessToken, &tokenResp.ExpiresIn); err != nil {
			return fmt.Errorf("failed to save token: %v", err)
		}

		break
	}

	return nil
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
	return true // Simplified for now
}

func (s *SSOManager) saveTokenToCache(startURL, ssoRegion string, accessToken *string, expiresIn *int32) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
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

	return ioutil.WriteFile(cacheFile, data, 0644)
}