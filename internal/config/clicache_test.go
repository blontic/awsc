package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCLICacheKey_MatchesBotocore(t *testing.T) {
	// python3 -c 'import json,hashlib; print(hashlib.sha1(json.dumps({"roleName":"Admin","accountId":"111111111111","sessionName":"awsc-alpha"}, sort_keys=True, separators=(",",":")).encode()).hexdigest())'
	const want = "3560831d3e38e1f6923759940c8c313c0c7af476"
	if got := cliCacheKey(accountRole{"111111111111", "Admin"}, "awsc-alpha"); got != want {
		t.Errorf("cliCacheKey = %s, want %s", got, want)
	}
}

func TestRemoveCLIRoleCredentials(t *testing.T) {
	home := setupHome(t, twoOrgsYAML)
	writeFile(t, filepath.Join(home, ".aws", "config"), `[profile awsc-prod/Admin]
sso_session = awsc-alpha
sso_account_id = 111111111111
sso_role_name = Admin

[profile awsc-beta-prod/Admin]
sso_session = awsc-beta
sso_account_id = 222222222222
sso_role_name = Admin
`)
	// A terminal on a role whose profile is gone (e.g. renamed) is covered too.
	if err := SaveSession(os.Getppid(), "awsc-dev/ReadOnly", "333333333333", "dev", "ReadOnly", "alpha"); err != nil {
		t.Fatal(err)
	}

	cache := func(id, role, session string) string {
		return filepath.Join(home, ".aws", "cli", "cache", cliCacheKey(accountRole{id, role}, session)+".json")
	}
	alphaProd, alphaDev, betaProd := cache("111111111111", "Admin", "awsc-alpha"), cache("333333333333", "ReadOnly", "awsc-alpha"), cache("222222222222", "Admin", "awsc-beta")
	other := filepath.Join(home, ".aws", "cli", "cache", "unrelated.json")
	for _, p := range []string{alphaProd, alphaDev, betaProd, other} {
		writeFile(t, p, "{}")
	}

	if err := RemoveCLIRoleCredentials("alpha"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{alphaProd, alphaDev} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be removed", filepath.Base(p))
		}
	}
	for _, p := range []string{betaProd, other} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s should be kept", filepath.Base(p))
		}
	}
}
