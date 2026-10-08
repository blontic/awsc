package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file contains the one-time migration from awsc <= 0.5, which used a
// single-org config and wrote static credentials into ~/.aws/config. It is the
// only code aware of the old format.

// legacyConfig is the awsc <= 0.5 config format.
type legacyConfig struct {
	SSO           SSOConfig `yaml:"sso"`
	DefaultRegion string    `yaml:"default_region"`
	Orgs          yaml.Node `yaml:"orgs"`
}

// legacyAwscKeys are all keys awsc has ever written to its ~/.aws/config
// sections. Sections with other keys were customised by the user and are kept.
var legacyAwscKeys = map[string]bool{
	"aws_access_key_id": true, "aws_secret_access_key": true, "aws_session_token": true,
	"sso_session": true, "sso_account_id": true, "sso_role_name": true, "region": true,
	"sso_start_url": true, "sso_region": true, "sso_registration_scopes": true,
}

// MigrateIfNeeded upgrades an awsc <= 0.5 install, detected by a config file
// with a top-level sso section and no orgs:
//
//  1. awsc sections in ~/.aws/config (including old static-credential
//     profiles) are removed; the org's sso-session is recreated by the next
//     sync and profiles are recreated on login
//  2. caches rebuilt on login (terminal sessions, account names) are cleared
//  3. the config becomes a single org named after the start URL (backup kept
//     at config.yaml.bak)
//
// The config is converted last, so a failure in an earlier step is retried on
// the next run.
func MigrateIfNeeded() error {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	var legacy legacyConfig
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	if legacy.SSO.StartURL == "" || !legacy.Orgs.IsZero() {
		return nil // not a legacy config
	}

	org := suggestOrgName(legacy.SSO.StartURL)
	fmt.Fprintf(os.Stderr, "Migrating awsc to the multi-org config format (org %q). You will need to log in again once.\n", org)

	if err := modifyAWSConfig(removeLegacyAwscSections); err != nil {
		return err
	}

	awscDir := filepath.Dir(path)
	for _, cache := range []string{filepath.Join(awscDir, "sessions"), filepath.Join(awscDir, "accounts.json")} {
		if err := os.RemoveAll(cache); err != nil {
			return fmt.Errorf("failed to remove %s: %w", cache, err)
		}
	}

	if err := WriteFileAtomic(path+".bak", data, 0600, false); err != nil {
		return fmt.Errorf("failed to back up %s: %w", path, err)
	}
	return writeFileConfig(&FileConfig{
		DefaultOrg: org,
		Orgs:       map[string]OrgConfig{org: {SSO: legacy.SSO, DefaultRegion: legacy.DefaultRegion}},
	})
}

// removeLegacyAwscSections removes [profile awsc-*], [sso-session awsc] and
// [sso-session awsc-*] sections that contain only keys awsc writes.
func removeLegacyAwscSections(content string) (string, error) {
	sections := parseSections(content)
	for _, s := range sections[1:] {
		isAwsc := strings.HasPrefix(s.name, "profile awsc-") ||
			s.name == "sso-session awsc" || strings.HasPrefix(s.name, "sso-session awsc-")
		if _, foreign := sectionKeys(s, legacyAwscKeys); isAwsc && !foreign {
			s.lines = nil
		}
	}
	return joinSections(sections), nil
}
