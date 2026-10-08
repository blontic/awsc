package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
	"github.com/blontic/awsc/internal/debug"
)

// This file manages the parts of ~/.aws/config that awsc owns:
//
//   - one [sso-session awsc-<org>] per org in the awsc config
//   - SSO profiles [profile awsc-<account>/<role>] (or
//     awsc-<org>-<account>/<role> when the name is taken by another org) that
//     point at their org's session
//
// No credentials are ever written; the AWS SDKs/CLI resolve short-lived role
// credentials on demand from the cached SSO token, exactly like profiles
// created by `aws configure sso`. Sections awsc does not own are never
// modified: an awsc-* section containing anything awsc would not write is
// treated as the user's.

const ssoSessionPrefix = "awsc-"

var (
	awscProfileKeys = map[string]bool{"sso_session": true, "sso_account_id": true, "sso_role_name": true, "region": true}
	awscSessionKeys = map[string]bool{"sso_start_url": true, "sso_region": true, "sso_registration_scopes": true}
)

// SSOSessionName returns the [sso-session] name for an org.
func SSOSessionName(org string) string {
	return ssoSessionPrefix + org
}

// SSOTokenCachePath returns the SSO token cache file for an org, shared with
// the AWS CLI and SDKs (~/.aws/sso/cache/<sha1(session name)>.json).
func SSOTokenCachePath(org string) (string, error) {
	return ssocreds.StandardCachedTokenFilepath(SSOSessionName(org))
}

// Profile identifies an account/role to write as an SSO profile.
type Profile struct {
	Org         string
	AccountName string
	AccountID   string
	RoleName    string
}

// syncResult describes the changes a sync made to ~/.aws/config.
type syncResult struct {
	Created         []string          // sessions added
	Renamed         map[string]string // old session -> new session (same start URL)
	Removed         []string          // sessions with no matching org
	RemovedProfiles []string          // awsc profiles removed with their session
}

func isAwscSession(s *iniSection) (string, bool) {
	name, ok := strings.CutPrefix(s.name, "sso-session ")
	return name, ok && strings.HasPrefix(name, ssoSessionPrefix)
}

func isOwnedSession(s *iniSection) bool {
	_, foreign := sectionKeys(s, awscSessionKeys)
	return !foreign
}

func isOwnedProfile(s *iniSection) bool {
	if !strings.HasPrefix(s.name, "profile awsc-") {
		return false
	}
	keys, foreign := sectionKeys(s, awscProfileKeys)
	return !foreign && strings.HasPrefix(keys["sso_session"], ssoSessionPrefix)
}

func sessionLines(org string, cfg OrgConfig) []string {
	return []string{
		"[sso-session " + SSOSessionName(org) + "]",
		"sso_start_url = " + cfg.SSO.StartURL,
		"sso_region = " + cfg.SSO.Region,
		"sso_registration_scopes = sso:account:access",
	}
}

func profileLines(name string, p Profile, region string) []string {
	lines := []string{
		"[profile " + name + "]",
		"# Account: " + p.AccountName,
		"sso_session = " + SSOSessionName(p.Org),
		"sso_account_id = " + p.AccountID,
		"sso_role_name = " + p.RoleName,
	}
	if region != "" {
		lines = append(lines, "region = "+region)
	}
	return lines
}

func normaliseStartURL(u string) string {
	return strings.TrimRight(strings.TrimSpace(u), "/")
}

func validateOrgs(orgs map[string]OrgConfig) error {
	for name, cfg := range orgs {
		for _, v := range []string{name, cfg.SSO.StartURL, cfg.SSO.Region, cfg.DefaultRegion} {
			if err := validateINIValue("org setting", v); err != nil {
				return err
			}
		}
	}
	return nil
}

// syncSessions makes the awsc sso-sessions in content match orgs exactly:
//
//   - each org has [sso-session awsc-<org>] with its current settings
//   - an awsc session for an org's start URL under another name (e.g. after the
//     org was re-created with a new name) is renamed and its profiles repointed
//   - awsc sessions matching no org are removed
//   - awsc profiles whose session is not an org's session are removed
//   - awsc profiles named without a role (awsc-<account>, the previous naming)
//     are removed; they are recreated under the new name on next use
func syncSessions(content string, orgs map[string]OrgConfig) (string, syncResult, error) {
	res := syncResult{Renamed: map[string]string{}}
	if err := validateOrgs(orgs); err != nil {
		return "", res, err
	}

	orgNames := make([]string, 0, len(orgs))
	sessionByURL := map[string]string{}
	for org := range orgs {
		orgNames = append(orgNames, org)
	}
	sort.Strings(orgNames)
	for _, org := range orgNames {
		url := normaliseStartURL(orgs[org].SSO.StartURL)
		if _, taken := sessionByURL[url]; !taken {
			sessionByURL[url] = SSOSessionName(org)
		}
	}

	sections := parseSections(content)
	present := map[string]bool{}

	for _, s := range sections[1:] {
		name, ok := isAwscSession(s)
		if !ok {
			continue
		}
		org := strings.TrimPrefix(name, ssoSessionPrefix)
		cfg, wanted := orgs[org]
		if !isOwnedSession(s) {
			if wanted {
				return "", res, fmt.Errorf("[sso-session %s] in ~/.aws/config was not created by awsc; rename or remove it and try again", name)
			}
			continue
		}
		switch {
		case wanted && present[name]:
			s.lines = nil // duplicate section
		case wanted:
			present[name] = true
			replaceSectionBody(s, sessionLines(org, cfg))
		default:
			keys, _ := sectionKeys(s, awscSessionKeys)
			if target, ok := sessionByURL[normaliseStartURL(keys["sso_start_url"])]; ok {
				res.Renamed[name] = target
			} else {
				res.Removed = append(res.Removed, name)
			}
			s.lines = nil
		}
	}

	for _, s := range sections[1:] {
		if !isOwnedProfile(s) {
			continue
		}
		keys, _ := sectionKeys(s, awscProfileKeys)
		session := keys["sso_session"]
		if !strings.Contains(s.name, "/") {
			res.RemovedProfiles = append(res.RemovedProfiles, strings.TrimPrefix(s.name, "profile "))
			s.lines = nil
		} else if target, ok := res.Renamed[session]; ok {
			setSectionKey(s, "sso_session", target)
		} else if _, isOrg := orgs[strings.TrimPrefix(session, ssoSessionPrefix)]; !isOrg {
			res.RemovedProfiles = append(res.RemovedProfiles, strings.TrimPrefix(s.name, "profile "))
			s.lines = nil
		}
	}

	var missing [][]string
	for _, org := range orgNames {
		if name := SSOSessionName(org); !present[name] {
			res.Created = append(res.Created, name)
			missing = append(missing, sessionLines(org, orgs[org]))
		}
	}
	return appendSections(joinSections(sections), missing), res, nil
}

// profileName picks the profile name for an account and role:
// "awsc-<account>/<role>", or "awsc-<org>-<account>/<role>" if that name
// belongs to another org. Whitespace in account names becomes "-".
func profileName(sections []*iniSection, p Profile) string {
	account := strings.Join(strings.Fields(p.AccountName), "-")
	plain := "awsc-" + account + "/" + p.RoleName
	prefixed := "awsc-" + p.Org + "-" + account + "/" + p.RoleName
	for _, s := range sections[1:] {
		switch s.name {
		case "profile " + prefixed:
			return prefixed
		case "profile " + plain:
			keys, _ := sectionKeys(s, awscProfileKeys)
			if session := keys["sso_session"]; session != "" && session != SSOSessionName(p.Org) {
				return prefixed
			}
		}
	}
	return plain
}

// upsertProfile writes the profile into content, updating it in place if it
// exists. It refuses to overwrite a same-named profile awsc did not create.
func upsertProfile(content string, p Profile, region string) (string, string, error) {
	sections := parseSections(content)
	name := profileName(sections, p)
	lines := profileLines(name, p, region)

	found := false
	for _, s := range sections[1:] {
		if s.name != "profile "+name {
			continue
		}
		if !isOwnedProfile(s) {
			return "", "", fmt.Errorf("profile %q in ~/.aws/config was not created by awsc; rename or remove it and try again", name)
		}
		if found {
			s.lines = nil // duplicate
			continue
		}
		found = true
		replaceSectionBody(s, lines)
	}

	content = joinSections(sections)
	if !found {
		content = appendSections(content, [][]string{lines})
	}
	return content, name, nil
}

func awsConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "config"), nil
}

// modifyAWSConfig applies fn to ~/.aws/config under an exclusive lock. If the
// content changes, the previous file is backed up to ~/.aws/config.awsc.bak
// and the new content written atomically, keeping the file's permissions.
func modifyAWSConfig(fn func(content string) (string, error)) error {
	path, err := awsConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	unlock, err := lockFile(filepath.Join(filepath.Dir(GetConfigPath()), "aws-config.lock"))
	if err != nil {
		return fmt.Errorf("failed to lock %s: %w", path, err)
	}
	defer unlock()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	content, err := fn(string(data))
	if err != nil || content == string(data) {
		return err
	}

	if len(data) > 0 {
		if err := WriteFileAtomic(path+".awsc.bak", data, 0600, false); err != nil {
			return fmt.Errorf("failed to back up %s: %w", path, err)
		}
	}
	// ~/.aws/config is shared with other tools and holds no secrets, so the
	// user's existing permissions are kept.
	if err := WriteFileAtomic(path, []byte(content), 0600, true); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

// applyTokenChanges keeps cached SSO tokens in line with a sync: a renamed
// session's token moves to the new name (unless one exists) and removed
// sessions' tokens are deleted.
func applyTokenChanges(res syncResult) {
	for old, target := range res.Renamed {
		oldPath, err1 := ssocreds.StandardCachedTokenFilepath(old)
		newPath, err2 := ssocreds.StandardCachedTokenFilepath(target)
		if err1 != nil || err2 != nil {
			continue
		}
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if data, err := os.ReadFile(oldPath); err == nil && WriteFileAtomic(newPath, data, 0600, false) != nil {
				continue // keep the old token if it could not be moved
			}
		}
		_ = os.Remove(oldPath)
	}
	for _, name := range res.Removed {
		if path, err := ssocreds.StandardCachedTokenFilepath(name); err == nil {
			_ = os.Remove(path)
		}
	}
}

// syncAWSConfig makes the awsc sso-sessions in ~/.aws/config match orgs,
// recreating missing ones and removing those (with their profiles and cached
// tokens) whose org no longer exists. Nothing is written if already in sync.
func syncAWSConfig(orgs map[string]OrgConfig) (syncResult, error) {
	var res syncResult
	err := modifyAWSConfig(func(content string) (string, error) {
		var err error
		content, res, err = syncSessions(content, orgs)
		return content, err
	})
	if err != nil {
		return syncResult{}, err
	}
	applyTokenChanges(res)
	return res, nil
}

// WriteProfile syncs the org sessions and writes an SSO profile for p,
// returning the profile name.
func WriteProfile(orgs map[string]OrgConfig, p Profile) (string, error) {
	cfg, ok := orgs[p.Org]
	if !ok {
		return "", fmt.Errorf("unknown org %q", p.Org)
	}
	for label, v := range map[string]string{"account name": p.AccountName, "account ID": p.AccountID, "role name": p.RoleName} {
		if err := validateINIValue(label, v); err != nil {
			return "", err
		}
	}

	var res syncResult
	var name string
	err := modifyAWSConfig(func(content string) (string, error) {
		var err error
		if content, res, err = syncSessions(content, orgs); err != nil {
			return "", err
		}
		content, name, err = upsertProfile(content, p, cfg.DefaultRegion)
		return content, err
	})
	if err != nil {
		return "", err
	}
	applyTokenChanges(res)
	return name, nil
}

// lookupProfile returns the account ID and role of an awsc profile in
// ~/.aws/config, and whether it exists.
func lookupProfile(name string) (accountID, roleName string, found bool, err error) {
	path, err := awsConfigPath()
	if err != nil {
		return "", "", false, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	for _, s := range parseSections(string(data)) {
		if s.name == "profile "+name {
			keys, _ := sectionKeys(s, awscProfileKeys)
			return keys["sso_account_id"], keys["sso_role_name"], true, nil
		}
	}
	return "", "", false, nil
}

// logSyncResult reports sync changes in verbose mode.
func logSyncResult(res syncResult) {
	for _, name := range res.Created {
		debug.Printf("Added [sso-session %s] to ~/.aws/config\n", name)
	}
	for old, target := range res.Renamed {
		debug.Printf("Renamed [sso-session %s] to [sso-session %s] in ~/.aws/config\n", old, target)
	}
	for _, name := range res.Removed {
		debug.Printf("Removed [sso-session %s] from ~/.aws/config\n", name)
	}
	for _, name := range res.RemovedProfiles {
		debug.Printf("Removed [profile %s] from ~/.aws/config\n", name)
	}
}
