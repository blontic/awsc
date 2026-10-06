package aws

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	awscconfig "github.com/blontic/awsc/internal/config"
)

type CredentialsManager struct {
	oidcClient *ssooidc.Client
}

// ssoCache is the SSO token cache file format shared with the AWS CLI and
// SDKs (~/.aws/sso/cache/<sha1(sso-session name)>.json).
type ssoCache struct {
	StartURL              string `json:"startUrl"`
	Region                string `json:"region"`
	AccessToken           string `json:"accessToken"`
	ExpiresAt             string `json:"expiresAt"`
	RefreshToken          string `json:"refreshToken,omitempty"`
	ClientID              string `json:"clientId,omitempty"`
	ClientSecret          string `json:"clientSecret,omitempty"`
	RegistrationExpiresAt string `json:"registrationExpiresAt,omitempty"`
}

const ssoAccountAccessScope = "sso:account:access"

// ssoTokenCachePath returns the token cache path for the active org.
func ssoTokenCachePath() (string, error) {
	return awscconfig.SSOTokenCachePath(awscconfig.Active().Org)
}

func NewCredentialsManager(ctx context.Context) (*CredentialsManager, error) {
	cfg, err := awscconfig.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &CredentialsManager{oidcClient: ssooidc.NewFromConfig(cfg)}, nil
}

// authErrorMarkers are substrings of errors caused by missing or expired
// credentials, which a new login can fix.
var authErrorMarkers = []string{
	"no active session",                    // terminal has no account selected (awsc)
	"no SSO cache found",                   // no SSO token for the org (awsc)
	"failed to get shared config profile",  // profile missing from ~/.aws/config
	"failed to refresh cached credentials", // SDK could not get role credentials
	"refresh cached SSO token failed",      // SSO token expired and could not be refreshed
	"get credentials",                      // SDK credential resolution failed
	"no EC2 IMDS role found",               // SDK found no credentials at all
	"AuthFailure",
	"SignatureDoesNotMatch",
	"TokenRefreshRequired",
	"ExpiredToken",
	"InvalidToken",
	"RequestExpired", // expired SSO credentials (SDK may misreport as clock skew)
}

// IsAuthError reports whether err is caused by missing or expired
// credentials. Permission errors and network errors are not auth errors.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "is not authorized to perform") {
		return false
	}
	for _, marker := range authErrorMarkers {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

// GetCachedToken returns a valid SSO access token from the shared cache,
// transparently refreshing it with the cached refresh token when expired.
func (c *CredentialsManager) GetCachedToken(ctx context.Context) (*string, error) {
	if awscconfig.Active().StartURL == "" {
		return nil, fmt.Errorf("no SSO start URL configured")
	}

	cacheFile, err := ssoTokenCachePath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(cacheFile); err != nil {
		return nil, fmt.Errorf("no SSO cache found, please run 'awsc login'")
	}

	tok, err := ssocreds.NewSSOTokenProvider(c.oidcClient, cacheFile).RetrieveBearerToken(ctx)
	if err != nil {
		return nil, err
	}
	return &tok.Value, nil
}

// loadCache reads the cached token file, returning nil if absent or unreadable.
func loadCache() *ssoCache {
	cacheFile, err := ssoTokenCachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}
	var cache ssoCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil
	}
	return &cache
}

// clientRegistration is an OIDC public client registration.
type clientRegistration struct {
	clientID     string
	clientSecret string
	expiresAt    time.Time
}

// getClientRegistration reuses a cached client registration for the same start
// URL when it is valid for at least another hour, otherwise registers anew.
func (c *CredentialsManager) getClientRegistration(ctx context.Context, startURL string) (*clientRegistration, error) {
	if cache := loadCache(); cache != nil && cache.StartURL == startURL && cache.ClientID != "" && cache.ClientSecret != "" {
		if exp, err := time.Parse(time.RFC3339, cache.RegistrationExpiresAt); err == nil && time.Until(exp) > time.Hour {
			return &clientRegistration{cache.ClientID, cache.ClientSecret, exp}, nil
		}
	}

	resp, err := c.oidcClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String("awsc"),
		ClientType: aws.String("public"),
		Scopes:     []string{ssoAccountAccessScope},
	})
	if err != nil {
		return nil, err
	}
	if resp.ClientId == nil || resp.ClientSecret == nil {
		return nil, fmt.Errorf("incomplete RegisterClient response from AWS")
	}
	return &clientRegistration{
		clientID:     *resp.ClientId,
		clientSecret: *resp.ClientSecret,
		expiresAt:    time.Unix(resp.ClientSecretExpiresAt, 0),
	}, nil
}

func (c *CredentialsManager) Authenticate(ctx context.Context, startURL, ssoRegion string) error {
	reg, err := c.getClientRegistration(ctx, startURL)
	if err != nil {
		return fmt.Errorf("failed to register client: %v", err)
	}

	// Start device authorization
	deviceResp, err := c.oidcClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     aws.String(reg.clientID),
		ClientSecret: aws.String(reg.clientSecret),
		StartUrl:     aws.String(startURL),
	})
	if err != nil {
		return fmt.Errorf("failed to start device authorization: %v", err)
	}

	if deviceResp.VerificationUriComplete == nil || deviceResp.UserCode == nil || deviceResp.DeviceCode == nil {
		return fmt.Errorf("incomplete StartDeviceAuthorization response from AWS")
	}

	if err := browserLogin(os.Stdin, os.Stderr, stdinIsTerminal(), awscconfig.Active().Org,
		*deviceResp.VerificationUriComplete, *deviceResp.UserCode, openBrowser); err != nil {
		return err
	}

	// Poll for token with timeout
	timeoutMinutes := int(deviceResp.ExpiresIn / 60)
	fmt.Fprintf(os.Stderr, "Waiting for approval in the browser (expires in %d minutes)...\n", timeoutMinutes)
	timeout := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
	interval := time.Duration(deviceResp.Interval) * time.Second

	for time.Now().Before(timeout) {
		tokenResp, err := c.oidcClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     aws.String(reg.clientID),
			ClientSecret: aws.String(reg.clientSecret),
			DeviceCode:   deviceResp.DeviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})

		if err != nil {
			// Check if we should continue polling
			if isRetryableError(err) {
				fmt.Fprint(os.Stderr, ".")
				time.Sleep(interval)
				continue
			}
			return fmt.Errorf("failed to create token: %v", err)
		}

		// Success! Save token to cache
		if tokenResp.AccessToken == nil {
			return fmt.Errorf("incomplete CreateToken response from AWS")
		}
		fmt.Fprintln(os.Stderr, "\nAuthentication successful!")
		cache := ssoCache{
			StartURL:              startURL,
			Region:                ssoRegion,
			AccessToken:           *tokenResp.AccessToken,
			ExpiresAt:             time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
			RefreshToken:          aws.ToString(tokenResp.RefreshToken),
			ClientID:              reg.clientID,
			ClientSecret:          reg.clientSecret,
			RegistrationExpiresAt: reg.expiresAt.UTC().Format(time.RFC3339),
		}
		if err := saveTokenToCache(cache); err != nil {
			return fmt.Errorf("failed to save token: %v", err)
		}

		return nil
	}

	return fmt.Errorf("authentication timed out - please try again")

}

// saveTokenToCache writes the token cache atomically with 0600 permissions.
func saveTokenToCache(cache ssoCache) error {
	cacheFile, err := ssoTokenCachePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return awscconfig.WriteFileAtomic(cacheFile, data, 0600, false)
}

// browserLogin tells the user a browser login is needed and shows the code to
// confirm in the browser. In an interactive terminal it waits for Enter, then
// opens the verification page; otherwise (scripts, CI) it only prints the URL.
func browserLogin(in io.Reader, out io.Writer, interactive bool, org, verifyURL, code string, open func(string) error) error {
	fmt.Fprintf(out, "Logging in to org %q. Confirm this code in the browser: %s\n", org, code)
	if !interactive {
		fmt.Fprintf(out, "Open this URL to approve the login: %s\n", verifyURL)
		return nil
	}

	fmt.Fprintf(out, "Press Enter to open %s (Ctrl+C to cancel)", verifyURL)
	if _, err := bufio.NewReader(in).ReadString('\n'); err != nil {
		return fmt.Errorf("login cancelled: %w", err)
	}
	if err := open(verifyURL); err != nil {
		fmt.Fprintf(out, "Could not open the browser (%v). Open the URL above to approve the login.\n", err)
	}
	return nil
}

// stdinIsTerminal reports whether stdin is an interactive terminal.
func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// openBrowser opens an https URL (the SSO verification page) in the user's
// browser. Only https URLs are accepted, as the URL is passed to a shell
// command on WSL.
func openBrowser(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("refusing to open non-https URL %q", rawURL)
	}
	name, args := browserCommand(runtime.GOOS, isWSL())
	return exec.Command(name, append(args, rawURL)...).Start()
}

// browserCommand returns the command that opens a URL on macOS, Linux, or WSL
// (where the Windows browser is used).
func browserCommand(goos string, wsl bool) (string, []string) {
	switch {
	case goos == "darwin":
		return "open", nil
	case wsl:
		// The empty argument after "start" is the window title.
		return "cmd.exe", []string{"/c", "start", ""}
	default:
		return "xdg-open", nil
	}
}

// isWSL checks if the current environment is Windows Subsystem for Linux
func isWSL() bool {
	// Check for WSL-specific environment variables
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	// Check /proc/version for Microsoft/WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		if strings.Contains(version, "microsoft") || strings.Contains(version, "wsl") {
			return true
		}
	}

	return false
}

func isRetryableError(err error) bool {
	// Check for authorization_pending or slow_down errors
	errorStr := err.Error()
	return strings.Contains(errorStr, "AuthorizationPendingException") ||
		strings.Contains(errorStr, "authorization_pending") ||
		strings.Contains(errorStr, "SlowDownException") ||
		strings.Contains(errorStr, "slow_down")
}

// PromptForReauth runs login when the terminal has no active session, or asks
// the user to re-authenticate when credentials have expired. It reports
// whether login completed so the caller can reload clients and retry.
func PromptForReauth(ctx context.Context) (bool, error) {
	_, loadErr := awscconfig.LoadAWSConfigWithProfile(ctx)
	if loadErr != nil && strings.Contains(loadErr.Error(), "no active session") {
		fmt.Fprintf(os.Stderr, "No account selected in this terminal for org %q.\n", awscconfig.Active().Org)
	} else {
		fmt.Fprint(os.Stderr, "Credentials expired. Re-authenticate? (y/n): ")
		response, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return false, err
		}
		if response = strings.ToLower(strings.TrimSpace(response)); response != "y" && response != "yes" {
			return false, nil
		}
	}

	ssoManager, err := NewSSOManager(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to create SSO manager: %w", err)
	}
	if err := ssoManager.RunLogin(ctx, false, "", ""); err != nil {
		return false, fmt.Errorf("authentication failed: %w", err)
	}

	return true, nil
}
