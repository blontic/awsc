package config

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file defines the awsc config (~/.awsc/config.yaml), which holds one or
// more orgs (IAM Identity Center start URLs), and selects the active org.

// OrgEnvVar selects the active org for a command or shell.
const OrgEnvVar = "AWSC_ORG"

// SSOConfig holds the IAM Identity Center settings for an org.
type SSOConfig struct {
	StartURL string `yaml:"start_url"`
	Region   string `yaml:"region"`
}

// OrgConfig is one AWS org awsc can log in to.
type OrgConfig struct {
	SSO           SSOConfig `yaml:"sso"`
	DefaultRegion string    `yaml:"default_region"`
}

// FileConfig is the format of ~/.awsc/config.yaml.
type FileConfig struct {
	DefaultOrg string               `yaml:"default_org,omitempty"`
	Orgs       map[string]OrgConfig `yaml:"orgs,omitempty"`
}

var (
	orgNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)
	nonSlugChars   = regexp.MustCompile(`[^a-z0-9-]+`)
)

// GetConfigPath returns the awsc config file path.
func GetConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".awsc", "config.yaml")
}

// validateOrgName checks an org name is a short lowercase identifier.
func validateOrgName(name string) error {
	if !orgNamePattern.MatchString(name) {
		return fmt.Errorf("invalid org name %q: use lowercase letters, numbers and dashes (max 40)", name)
	}
	return nil
}

// suggestOrgName derives an org name from a start URL: the subdomain of
// https://<sub>.awsapps.com/start, otherwise a short hash of the URL.
func suggestOrgName(startURL string) string {
	u := normaliseStartURL(startURL)
	host := strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	host, _, _ = strings.Cut(host, "/")
	if sub, ok := strings.CutSuffix(strings.ToLower(host), ".awsapps.com"); ok {
		if name := strings.Trim(nonSlugChars.ReplaceAllString(sub, "-"), "-"); validateOrgName(name) == nil {
			return name
		}
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(u)))[:8]
}

// sortedOrgNames returns org names in alphabetical order.
func (c *FileConfig) sortedOrgNames() []string {
	names := make([]string, 0, len(c.Orgs))
	for name := range c.Orgs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ReadFileConfig reads the awsc config. A missing file yields an empty config.
func ReadFileConfig() (*FileConfig, error) {
	cfg := &FileConfig{}
	data, err := os.ReadFile(GetConfigPath())
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", GetConfigPath(), err)
	}
	if cfg.Orgs == nil {
		cfg.Orgs = map[string]OrgConfig{}
	}
	for name := range cfg.Orgs {
		if err := validateOrgName(name); err != nil {
			return nil, fmt.Errorf("%s: %w", GetConfigPath(), err)
		}
	}
	return cfg, nil
}

// writeFileConfig writes the awsc config atomically with 0600 permissions.
func writeFileConfig(cfg *FileConfig) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return WriteFileAtomic(GetConfigPath(), buf.Bytes(), 0600, false)
}

// resolveActiveOrg picks the org to use, in priority order: explicit flag,
// AWSC_ORG, the org this terminal last logged in to, default_org, or the only
// org configured. It returns "" if no org can be determined.
func resolveActiveOrg(cfg *FileConfig, flagOrg string) (string, error) {
	for _, explicit := range []string{flagOrg, os.Getenv(OrgEnvVar)} {
		if explicit == "" {
			continue
		}
		if _, ok := cfg.Orgs[explicit]; !ok {
			return "", unknownOrgError(cfg, explicit)
		}
		return explicit, nil
	}
	if session, err := GetCurrentSession(); err == nil {
		if _, ok := cfg.Orgs[session.Org]; ok {
			return session.Org, nil
		}
	}
	if _, ok := cfg.Orgs[cfg.DefaultOrg]; ok {
		return cfg.DefaultOrg, nil
	}
	if len(cfg.Orgs) == 1 {
		return cfg.sortedOrgNames()[0], nil
	}
	return "", nil
}

func unknownOrgError(cfg *FileConfig, name string) error {
	return fmt.Errorf("unknown org %q (configured: %s). Add it with 'awsc config add %s'",
		name, strings.Join(cfg.sortedOrgNames(), ", "), name)
}

// Settings are the active org's settings for this process: chosen by
// ActivateOrg (or AddOrg/UseOrg) and adjusted by the --region flag.
type Settings struct {
	Org           string
	StartURL      string
	SSORegion     string
	DefaultRegion string
}

var active Settings

// Active returns the active org's settings.
func Active() Settings { return active }

// SetActive replaces the active settings.
func SetActive(s Settings) { active = s }

// applyOrg makes an org's settings the active ones.
func applyOrg(name string, org OrgConfig) {
	active = Settings{
		Org:           name,
		StartURL:      org.SSO.StartURL,
		SSORegion:     org.SSO.Region,
		DefaultRegion: org.DefaultRegion,
	}
}

// ActivateOrg prepares awsc for a command: migrates a pre-multi-org install,
// keeps ~/.aws/config in sync with the configured orgs, and applies the
// resolved org's settings.
func ActivateOrg(flagOrg string) error {
	if err := MigrateIfNeeded(); err != nil {
		return err
	}

	cfg, err := ReadFileConfig()
	if err != nil {
		return err
	}

	// With no orgs (e.g. the awsc config was deleted) there is nothing to sync
	// against; leave ~/.aws/config alone so re-adding the org restores it.
	if len(cfg.Orgs) > 0 {
		res, err := syncAWSConfig(cfg.Orgs)
		if err != nil {
			return err
		}
		logSyncResult(res)
	}

	name, err := resolveActiveOrg(cfg, flagOrg)
	if err != nil {
		return err
	}
	if name == "" {
		if len(cfg.Orgs) > 1 {
			return fmt.Errorf("multiple orgs configured and no default set; use --org <name> or 'awsc config use <name>' (configured: %s)",
				strings.Join(cfg.sortedOrgNames(), ", "))
		}
		return nil
	}
	applyOrg(name, cfg.Orgs[name])
	return nil
}
