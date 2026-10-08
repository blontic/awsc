package config

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The AWS CLI (botocore) caches the role credentials it gets through an SSO
// profile in ~/.aws/cli/cache, keyed by a hash of the account, role and
// sso-session (SSOCredentialFetcher._create_cache_key). They stay usable
// until they expire, so logging out removes those of the org's roles.

type accountRole struct{ accountID, roleName string }

// RemoveCLIRoleCredentials deletes the AWS CLI's cached role credentials for
// the org's roles: those of its awsc profiles and of terminal sessions.
func RemoveCLIRoleCredentials(org string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	roles, err := orgAccountRoles(org)
	if err != nil {
		return err
	}
	for _, r := range roles {
		path := filepath.Join(home, ".aws", "cli", "cache", cliCacheKey(r, SSOSessionName(org))+".json")
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove cached AWS CLI credentials %s: %w", path, err)
		}
	}
	return nil
}

// cliCacheKey is botocore's cache key for SSO role credentials:
// sha1(json.dumps({"accountId", "roleName", "sessionName"}, sort_keys=True,
// separators=(',', ':'))).
func cliCacheKey(r accountRole, sessionName string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	// Map keys are encoded sorted, matching sort_keys=True.
	_ = enc.Encode(map[string]string{"accountId": r.accountID, "roleName": r.roleName, "sessionName": sessionName})
	return fmt.Sprintf("%x", sha1.Sum(bytes.TrimSuffix(buf.Bytes(), []byte("\n"))))
}

// orgAccountRoles lists the account/role pairs of the org's awsc profiles in
// ~/.aws/config and of terminal sessions for the org.
func orgAccountRoles(org string) ([]accountRole, error) {
	seen := map[accountRole]bool{}
	var roles []accountRole
	add := func(r accountRole) {
		if r.accountID != "" && r.roleName != "" && !seen[r] {
			seen[r] = true
			roles = append(roles, r)
		}
	}

	path, err := awsConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	for _, s := range parseSections(string(data)) {
		if !isOwnedProfile(s) {
			continue
		}
		keys, _ := sectionKeys(s, awscProfileKeys)
		if keys["sso_session"] == SSOSessionName(org) {
			add(accountRole{keys["sso_account_id"], keys["sso_role_name"]})
		}
	}

	sessionsDir := filepath.Join(filepath.Dir(GetConfigPath()), "sessions")
	entries, _ := os.ReadDir(sessionsDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(sessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var session SessionInfo
		if json.Unmarshal(raw, &session) == nil && session.Org == org {
			add(accountRole{session.AccountID, session.RoleName})
		}
	}
	return roles, nil
}
