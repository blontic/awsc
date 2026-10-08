package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/sso"
	awscconfig "github.com/blontic/awsc/internal/config"
)

// SSOLogoutClient interface for mocking
type SSOLogoutClient interface {
	Logout(ctx context.Context, params *sso.LogoutInput, optFns ...func(*sso.Options)) (*sso.LogoutOutput, error)
}

// LogoutManager signs out of IAM Identity Center: it ends the SSO session with
// AWS and removes the cached SSO token and terminal sessions.
type LogoutManager struct {
	// client returns an SSO client for an IAM Identity Center region; each
	// org may use a different region.
	client func(region string) SSOLogoutClient
}

type LogoutManagerOptions struct {
	Client func(region string) SSOLogoutClient
}

func NewLogoutManager(ctx context.Context, opts ...LogoutManagerOptions) (*LogoutManager, error) {
	if len(opts) > 0 && opts[0].Client != nil {
		return &LogoutManager{client: opts[0].Client}, nil
	}
	cfg, err := awscconfig.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &LogoutManager{client: func(region string) SSOLogoutClient {
		return sso.NewFromConfig(cfg, func(o *sso.Options) { o.Region = region })
	}}, nil
}

// RunLogout logs out of the active org and clears this terminal's session, or
// with all, logs out of every org and clears every terminal's session.
func (m *LogoutManager) RunLogout(ctx context.Context, all bool) error {
	cfg, err := awscconfig.ReadFileConfig()
	if err != nil {
		return err
	}

	var orgs []string
	if all {
		for name := range cfg.Orgs {
			orgs = append(orgs, name)
		}
		sort.Strings(orgs)
	} else {
		org := awscconfig.Active().Org
		if org == "" {
			return fmt.Errorf("no org selected; use --org <name> or --all")
		}
		orgs = []string{org}
	}

	for _, org := range orgs {
		loggedIn, err := m.logoutOrg(ctx, org, cfg.Orgs[org].SSO.Region)
		switch {
		case err != nil:
			return err
		case loggedIn:
			fmt.Fprintf(os.Stderr, "Logged out of org %q\n", org)
		default:
			fmt.Fprintf(os.Stderr, "Not logged in to org %q\n", org)
		}
	}

	if all {
		return awscconfig.ClearAllSessions()
	}
	return awscconfig.ClearCurrentSession()
}

// logoutOrg ends the org's SSO session with AWS and deletes its cached token,
// reporting whether there was a login. The token is deleted even if AWS can't
// be reached; the server-side session then ends when it expires.
func (m *LogoutManager) logoutOrg(ctx context.Context, org, ssoRegion string) (bool, error) {
	path, err := awscconfig.SSOTokenCachePath(org)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read the SSO token for org %q: %w", org, err)
	}

	var cache ssoCache
	if json.Unmarshal(data, &cache) == nil && cache.AccessToken != "" {
		region := cache.Region
		if region == "" {
			region = ssoRegion
		}
		_, err := m.client(region).Logout(ctx, &sso.LogoutInput{AccessToken: &cache.AccessToken})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not end the SSO session for org %q with AWS (%v); removing the local login anyway\n", org, err)
		}
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to remove the SSO token for org %q: %w", org, err)
	}
	return true, nil
}
