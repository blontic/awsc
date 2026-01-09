package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/spf13/viper"
)

// ConsoleManager handles opening the AWS Management Console in a web browser
type ConsoleManager struct {
	cfg            aws.Config
	session        *awscconfig.SessionInfo
	logoutFirst    bool
}

// NewConsoleManager creates a new ConsoleManager
func NewConsoleManager(ctx context.Context) (*ConsoleManager, error) {
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	session, err := awscconfig.GetCurrentSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get current session: %w", err)
	}

	return &ConsoleManager{
		cfg:         cfg,
		session:     session,
		logoutFirst: false,
	}, nil
}

// SetLogoutFirst configures the manager to logout before opening console
func (c *ConsoleManager) SetLogoutFirst(logout bool) {
	c.logoutFirst = logout
}

// federationSigninResponse represents the response from AWS federation signin endpoint
type federationSigninResponse struct {
	SigninToken string `json:"SigninToken"`
}

// OpenConsole opens the AWS Management Console in the default web browser
func (c *ConsoleManager) OpenConsole(ctx context.Context, service string) error {
	// If logoutFirst is set, logout of AWS console first
	if c.logoutFirst {
		fmt.Printf("Logging out of existing AWS console session...\n")
		logoutURL := "https://signin.aws.amazon.com/oauth?Action=logout"
		if err := OpenBrowser(logoutURL); err != nil {
			return fmt.Errorf("failed to open logout URL: %w", err)
		}
		// Wait a moment for logout to process
		time.Sleep(2 * time.Second)
	}

	// Get credentials from the config
	creds, err := c.cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	// Check if credentials are expired
	if creds.Expired() {
		return fmt.Errorf("credentials have expired, please run 'awsc login' again")
	}

	// Generate federation signin token
	signinToken, err := c.getSigninToken(ctx, creds)
	if err != nil {
		return fmt.Errorf("failed to get signin token: %w", err)
	}

	// Construct destination URL
	destinationURL := c.getDestinationURL(service)

	// Construct the final console URL
	// Note: SigninToken should NOT be URL escaped, only the destination
	consoleURL := fmt.Sprintf(
		"https://signin.aws.amazon.com/federation?Action=login&Issuer=awsc&Destination=%s&SigninToken=%s",
		url.QueryEscape(destinationURL),
		signinToken,
	)

	// Display information
	fmt.Printf("Opening AWS Console for account: %s (%s)\n", c.session.AccountName, c.session.AccountID)
	if service != "" {
		fmt.Printf("Service: %s\n", service)
	}
	fmt.Printf("Region: %s\n", c.cfg.Region)

	// Open in browser
	if err := OpenBrowser(consoleURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	fmt.Printf("✓ AWS Console opened in your default browser\n")
	return nil
}

// getSigninToken requests a federation signin token from AWS
func (c *ConsoleManager) getSigninToken(ctx context.Context, creds aws.Credentials) (string, error) {
	// Create session JSON for AWS federation
	sessionJSON := map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	}

	sessionData, err := json.Marshal(sessionJSON)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Construct the federation endpoint URL
	federationURL := fmt.Sprintf(
		"https://signin.aws.amazon.com/federation?Action=getSigninToken&SessionDuration=43200&Session=%s",
		url.QueryEscape(string(sessionData)),
	)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, federationURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request signin token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("federation endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var signinResp federationSigninResponse
	if err := json.NewDecoder(resp.Body).Decode(&signinResp); err != nil {
		return "", fmt.Errorf("failed to decode signin token response: %w", err)
	}

	if signinResp.SigninToken == "" {
		return "", fmt.Errorf("received empty signin token")
	}

	return signinResp.SigninToken, nil
}

// getDestinationURL constructs the destination URL within the AWS Console
func (c *ConsoleManager) getDestinationURL(service string) string {
	region := c.cfg.Region
	if region == "" {
		region = viper.GetString("default_region")
	}
	if region == "" {
		region = "us-east-1"
	}

	// If no service specified, go to console home
	if service == "" {
		return fmt.Sprintf("https://console.aws.amazon.com/console/home?region=%s", region)
	}

	// Service-specific URLs
	// Some services have custom URL patterns
	switch service {
	case "ec2":
		return fmt.Sprintf("https://console.aws.amazon.com/ec2/home?region=%s", region)
	case "s3":
		// S3 console doesn't use region in the same way
		return "https://s3.console.aws.amazon.com/s3/home"
	case "rds":
		return fmt.Sprintf("https://console.aws.amazon.com/rds/home?region=%s", region)
	case "lambda":
		return fmt.Sprintf("https://console.aws.amazon.com/lambda/home?region=%s", region)
	case "cloudwatch":
		return fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home?region=%s", region)
	case "iam":
		// IAM is global
		return "https://console.aws.amazon.com/iam/home"
	case "vpc":
		return fmt.Sprintf("https://console.aws.amazon.com/vpc/home?region=%s", region)
	case "cloudformation":
		return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s", region)
	case "dynamodb":
		return fmt.Sprintf("https://console.aws.amazon.com/dynamodb/home?region=%s", region)
	case "sqs":
		return fmt.Sprintf("https://console.aws.amazon.com/sqs/home?region=%s", region)
	case "sns":
		return fmt.Sprintf("https://console.aws.amazon.com/sns/home?region=%s", region)
	case "opensearch":
		return fmt.Sprintf("https://console.aws.amazon.com/aos/home?region=%s", region)
	case "secretsmanager", "secrets":
		return fmt.Sprintf("https://console.aws.amazon.com/secretsmanager/home?region=%s", region)
	default:
		// Generic service URL pattern
		return fmt.Sprintf("https://console.aws.amazon.com/%s/home?region=%s", service, region)
	}
}
