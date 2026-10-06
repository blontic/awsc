package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// AWS regions list
var awsRegions = map[string]bool{
	"us-east-1":      true,
	"us-east-2":      true,
	"us-west-1":      true,
	"us-west-2":      true,
	"af-south-1":     true,
	"ap-east-1":      true,
	"ap-south-1":     true,
	"ap-south-2":     true,
	"ap-southeast-1": true,
	"ap-southeast-2": true,
	"ap-southeast-3": true,
	"ap-southeast-4": true,
	"ap-northeast-1": true,
	"ap-northeast-2": true,
	"ap-northeast-3": true,
	"ca-central-1":   true,
	"ca-west-1":      true,
	"eu-central-1":   true,
	"eu-central-2":   true,
	"eu-west-1":      true,
	"eu-west-2":      true,
	"eu-west-3":      true,
	"eu-north-1":     true,
	"eu-south-1":     true,
	"eu-south-2":     true,
	"il-central-1":   true,
	"me-central-1":   true,
	"me-south-1":     true,
	"sa-east-1":      true,
}

// SSO URL regex pattern
var ssoURLPattern = regexp.MustCompile(`^https://[a-zA-Z0-9-]+\.awsapps\.com/start/?$`)

// ValidateRegion reports whether region is a recognized AWS region.
func ValidateRegion(region string) bool {
	return awsRegions[region]
}

// validateSSOURL checks if the SSO URL matches the expected pattern
func validateSSOURL(url string) bool {
	return ssoURLPattern.MatchString(url)
}

// EnsureConfigExists runs first-time setup if no org is configured.
func EnsureConfigExists() error {
	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}
	if len(cfg.Orgs) > 0 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "No awsc configuration found. Let's set up AWSC.\n\n")
	_, err = AddOrg("")
	return err
}

// stdin is the source of interactive input (overridable in tests).
var stdin io.Reader = os.Stdin

func prompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// promptValid asks until valid accepts the answer.
func promptValid(reader *bufio.Reader, label, invalidMsg string, valid func(string) bool) (string, error) {
	for {
		v, err := prompt(reader, label)
		if err != nil || valid(v) {
			return v, err
		}
		fmt.Fprintln(os.Stderr, invalidMsg)
	}
}

// promptOrgSettings interactively collects and validates an org's settings.
func promptOrgSettings(reader *bufio.Reader) (OrgConfig, error) {
	const badRegion = "Invalid AWS region. Please enter a valid region like us-east-1, us-west-2, etc."
	var org OrgConfig
	var err error
	if org.SSO.StartURL, err = promptValid(reader, "SSO Start URL: ", "Invalid SSO URL format. Expected: https://your-org.awsapps.com/start", validateSSOURL); err != nil {
		return org, err
	}
	if org.SSO.Region, err = promptValid(reader, "SSO Region (e.g., us-east-1): ", badRegion, ValidateRegion); err != nil {
		return org, err
	}
	org.DefaultRegion, err = promptValid(reader, "Default AWS Region (e.g., us-east-1): ", badRegion, ValidateRegion)
	return org, err
}

// syncAndReport syncs ~/.aws/config with the configured orgs and reports changes.
func syncAndReport(cfg *FileConfig) error {
	res, err := syncAWSConfig(cfg.Orgs)
	if err != nil {
		return fmt.Errorf("failed to update ~/.aws/config: %w", err)
	}
	logSyncResult(res)
	return nil
}

// AddOrg interactively adds an org and creates its sso-session in
// ~/.aws/config. If name is empty the user is asked for one (suggested from the
// start URL). The first org becomes the default.
func AddOrg(name string) (string, error) {
	cfg, err := ReadFileConfig()
	if err != nil {
		return "", err
	}
	if name != "" {
		if err := validateOrgName(name); err != nil {
			return "", err
		}
		if _, exists := cfg.Orgs[name]; exists {
			return "", fmt.Errorf("org %q already exists; remove it first with 'awsc config remove %s'", name, name)
		}
	}

	reader := bufio.NewReader(stdin)
	org, err := promptOrgSettings(reader)
	if err != nil {
		return "", err
	}
	for existing, cfgOrg := range cfg.Orgs {
		if normaliseStartURL(cfgOrg.SSO.StartURL) == normaliseStartURL(org.SSO.StartURL) {
			return "", fmt.Errorf("%s is already configured as org %q", org.SSO.StartURL, existing)
		}
	}

	for name == "" {
		suggested := suggestOrgName(org.SSO.StartURL)
		v, err := prompt(reader, fmt.Sprintf("Org name [%s]: ", suggested))
		if err != nil {
			return "", err
		}
		if v == "" {
			v = suggested
		}
		if err := validateOrgName(v); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if _, exists := cfg.Orgs[v]; exists {
			fmt.Fprintf(os.Stderr, "Org %q already exists, choose another name.\n", v)
		} else {
			name = v
		}
	}

	cfg.Orgs[name] = org
	if cfg.DefaultOrg == "" {
		cfg.DefaultOrg = name
	}
	if err := writeFileConfig(cfg); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}
	applyOrg(name, org)

	suffix := ""
	if cfg.DefaultOrg == name {
		suffix = " (default)"
	}
	fmt.Fprintf(os.Stderr, "Org %q saved to %s%s\n", name, GetConfigPath(), suffix)
	return name, syncAndReport(cfg)
}

// ListOrgs prints configured orgs, marking the default (*) and the active one.
func ListOrgs(w io.Writer) error {
	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}
	if len(cfg.Orgs) == 0 {
		fmt.Fprintln(os.Stderr, "No orgs configured. Run 'awsc config add'.")
		return nil
	}
	activeOrg := active.Org
	for _, name := range cfg.sortedOrgNames() {
		org := cfg.Orgs[name]
		marker := " "
		var tags []string
		if name == cfg.DefaultOrg {
			marker = "*"
			tags = append(tags, "default")
		}
		if name == activeOrg {
			tags = append(tags, "active")
		}
		label := name
		if len(tags) > 0 {
			label += " (" + strings.Join(tags, ", ") + ")"
		}
		fmt.Fprintf(w, "%s %s\n    %s  sso:%s  region:%s\n", marker, label, org.SSO.StartURL, org.SSO.Region, org.DefaultRegion)
	}
	return nil
}

// UseOrg sets the default org and switches the current terminal to it. If the
// terminal is logged in to another org, its session is cleared so the next
// command uses the new org (logging in if needed). Other terminals keep
// their org.
func UseOrg(name string) error {
	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Orgs[name]; !ok {
		return unknownOrgError(cfg, name)
	}
	cfg.DefaultOrg = name
	if err := writeFileConfig(cfg); err != nil {
		return err
	}
	if session, err := GetCurrentSession(); err == nil && session.Org != name {
		if err := ClearCurrentSession(); err != nil {
			return err
		}
	}
	applyOrg(name, cfg.Orgs[name])
	fmt.Fprintf(os.Stderr, "Now using org %q (default)\n", name)
	return nil
}

// RemoveOrg removes an org, its sso-session and awsc profiles in
// ~/.aws/config, and its cached SSO token.
func RemoveOrg(name string) error {
	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Orgs[name]; !ok {
		return unknownOrgError(cfg, name)
	}
	delete(cfg.Orgs, name)
	if cfg.DefaultOrg == name {
		cfg.DefaultOrg = ""
		if len(cfg.Orgs) == 1 {
			cfg.DefaultOrg = cfg.sortedOrgNames()[0]
		}
	}
	if err := writeFileConfig(cfg); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Removed org %q\n", name)
	if cfg.DefaultOrg == "" && len(cfg.Orgs) > 1 {
		fmt.Fprintln(os.Stderr, "No default org set; run 'awsc config use <name>'")
	}
	return syncAndReport(cfg)
}

// ShowConfig writes the named org's settings, or the active org's, to w.
func ShowConfig(w io.Writer, name string) error {
	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}
	if len(cfg.Orgs) == 0 {
		fmt.Fprintln(os.Stderr, "No orgs configured. Run 'awsc config add'.")
		return nil
	}
	if name == "" {
		name = active.Org
	}
	org, ok := cfg.Orgs[name]
	if !ok {
		return unknownOrgError(cfg, name)
	}

	fmt.Fprintf(w, "Configuration file: %s\n\n", GetConfigPath())
	fmt.Fprintf(w, "Org: %s", name)
	if name == cfg.DefaultOrg {
		fmt.Fprint(w, " (default)")
	}
	fmt.Fprintf(w, "\nSSO Start URL: %s\nSSO Region: %s\nDefault Region: %s\n", org.SSO.StartURL, org.SSO.Region, org.DefaultRegion)
	return nil
}
