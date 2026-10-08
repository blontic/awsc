package aws

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	awscconfig "github.com/blontic/awsc/internal/config"
)

// ConsoleManager opens the AWS console as the terminal's account and role
// through an IAM Identity Center shortcut link, so the access portal signs the
// browser in with its own session:
// https://docs.aws.amazon.com/singlesignon/latest/userguide/createshortcutlink.html
type ConsoleManager struct {
	startURL  string
	accountID string
	roleName  string
	region    string
	open      func(string) error
}

type ConsoleManagerOptions struct {
	StartURL  string
	AccountID string
	RoleName  string
	Region    string
	Open      func(string) error
}

func NewConsoleManager(ctx context.Context, opts ...ConsoleManagerOptions) (*ConsoleManager, error) {
	if len(opts) > 0 && opts[0].AccountID != "" {
		o := opts[0]
		m := &ConsoleManager{startURL: o.StartURL, accountID: o.AccountID, roleName: o.RoleName, region: o.Region, open: o.Open}
		if m.open == nil {
			m.open = openBrowser
		}
		return m, nil
	}

	// Loading the profile fails with an auth error when the terminal has no
	// account selected, so the caller runs login first.
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}
	accountID, roleName, err := awscconfig.CurrentAccountRole()
	if err != nil {
		return nil, err
	}
	return &ConsoleManager{
		startURL:  awscconfig.Active().StartURL,
		accountID: accountID,
		roleName:  roleName,
		region:    cfg.Region,
		open:      openBrowser,
	}, nil
}

var consoleNamePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// RunOpen opens the console home page, or the given service's page (the path
// segment of its console URL, e.g. "ec2", "rds", "cloudwatch"), in the
// browser.
func (m *ConsoleManager) RunOpen(ctx context.Context, service string) error {
	link, err := m.shortcutURL(service)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Opening the AWS console for account %s as %s...\n", m.accountID, m.roleName)
	if err := m.open(link); err != nil {
		return fmt.Errorf("failed to open the browser: %w", err)
	}
	return nil
}

// shortcutURL builds the access portal link that signs in to the account and
// role and redirects to the service's console page in the region.
func (m *ConsoleManager) shortcutURL(service string) (string, error) {
	if service == "" {
		service = "console"
	}
	if !consoleNamePattern.MatchString(service) {
		return "", fmt.Errorf("invalid service %q: use the name from its console URL, e.g. ec2 or cloudwatch", service)
	}
	if m.region != "" && !consoleNamePattern.MatchString(m.region) {
		return "", fmt.Errorf("invalid region %q", m.region)
	}
	portal, err := url.Parse(strings.TrimSpace(m.startURL))
	if err != nil || portal.Scheme != "https" || portal.Host == "" {
		return "", fmt.Errorf("invalid SSO start URL %q", m.startURL)
	}
	portal.Path = strings.TrimRight(portal.Path, "/")
	portal.RawPath, portal.RawQuery, portal.Fragment = "", "", ""

	destination := "https://" + consoleHost(m.region) + "/" + service + "/home"
	if m.region != "" {
		destination += "?region=" + m.region
	}
	query := url.Values{
		"account_id":  {m.accountID},
		"role_name":   {m.roleName},
		"destination": {destination},
	}
	return portal.String() + "/#/console?" + query.Encode(), nil
}

// consoleHost returns the console host for a region's partition. Regional
// hosts are used except in each partition's primary region.
func consoleHost(region string) string {
	host, primary := "console.aws.amazon.com", "us-east-1"
	switch {
	case strings.HasPrefix(region, "cn-"):
		host, primary = "console.amazonaws.cn", "cn-north-1"
	case strings.HasPrefix(region, "us-gov-"):
		host, primary = "console.amazonaws-us-gov.com", "us-gov-west-1"
	}
	if region != "" && region != primary {
		host = region + "." + host
	}
	return host
}
