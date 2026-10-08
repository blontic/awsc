package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const v05Config = `sso:
  start_url: https://d-976710a35d.awsapps.com/start
  region: ap-southeast-2
default_region: ap-southeast-2
`

const v05AWSConfig = `[default]
region = us-west-2

[profile awsc-prod]
# Account: prod (111111111111)
# Role: Admin
aws_access_key_id = AKIA
aws_secret_access_key = secret
aws_session_token = token

[profile awsc-custom]
role_arn = arn:aws:iam::111111111111:role/X
source_profile = default

[profile mine]
aws_access_key_id = KEEP
`

func TestMigrateIfNeeded(t *testing.T) {
	home := setupHome(t, v05Config)
	awsConfig := filepath.Join(home, ".aws", "config")
	writeFile(t, awsConfig, v05AWSConfig)
	writeFile(t, filepath.Join(home, ".awsc", "sessions", "session-1.json"), `{}`)
	writeFile(t, filepath.Join(home, ".awsc", "accounts.json"), `{"accounts":{}}`)

	if err := MigrateIfNeeded(); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadFileConfig()
	if err != nil {
		t.Fatal(err)
	}
	org := cfg.Orgs["d-976710a35d"]
	if cfg.DefaultOrg != "d-976710a35d" || org.SSO.StartURL != "https://d-976710a35d.awsapps.com/start" || org.DefaultRegion != "ap-southeast-2" {
		t.Errorf("unexpected migrated config: %+v", cfg)
	}
	if readFile(t, GetConfigPath()+".bak") != v05Config {
		t.Error("config backup missing")
	}

	got := readFile(t, awsConfig)
	if strings.Contains(got, "awsc-prod") || strings.Contains(got, "AKIA") {
		t.Errorf("old awsc profiles not removed:\n%s", got)
	}
	for _, keep := range []string{"[default]", "[profile awsc-custom]", "role_arn", "[profile mine]", "KEEP"} {
		if !strings.Contains(got, keep) {
			t.Errorf("%q should be kept:\n%s", keep, got)
		}
	}

	if _, err := os.Stat(filepath.Join(home, ".awsc", "sessions")); !os.IsNotExist(err) {
		t.Error("old terminal sessions should be cleared")
	}
	if _, err := os.Stat(filepath.Join(home, ".awsc", "accounts.json")); !os.IsNotExist(err) {
		t.Error("old account cache should be cleared")
	}

	// Already migrated: no-op.
	before := readFile(t, GetConfigPath())
	if err := MigrateIfNeeded(); err != nil || readFile(t, GetConfigPath()) != before {
		t.Errorf("second migration should be a no-op: %v", err)
	}
}

func TestMigrateIfNeeded_ThenActivateCreatesSession(t *testing.T) {
	home := setupHome(t, v05Config)
	writeFile(t, filepath.Join(home, ".aws", "config"), v05AWSConfig)

	if err := ActivateOrg(""); err != nil {
		t.Fatal(err)
	}
	if name := active.Org; name != "d-976710a35d" {
		t.Errorf("migrated org not active: %q", name)
	}
	if !strings.Contains(readFile(t, filepath.Join(home, ".aws", "config")), "[sso-session awsc-d-976710a35d]") {
		t.Error("migrated org's sso-session not created")
	}
}

func TestMigrateIfNeeded_NoConfig(t *testing.T) {
	setupHome(t, "")
	if err := MigrateIfNeeded(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(GetConfigPath()); !os.IsNotExist(err) {
		t.Error("no config should be created")
	}
}

func TestMigrateIfNeeded_RetriesAfterFailure(t *testing.T) {
	home := setupHome(t, v05Config)
	awsDir := filepath.Join(home, ".aws")
	writeFile(t, filepath.Join(awsDir, "config"), v05AWSConfig)

	// Make ~/.aws read-only so cleaning ~/.aws/config fails.
	if err := os.Chmod(awsDir, 0500); err != nil {
		t.Fatal(err)
	}
	err := MigrateIfNeeded()
	_ = os.Chmod(awsDir, 0700)
	if err == nil {
		t.Fatal("expected migration to fail")
	}
	if readFile(t, GetConfigPath()) != v05Config {
		t.Fatal("config must stay in the old format so migration is retried")
	}

	if err := MigrateIfNeeded(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readFile(t, filepath.Join(awsDir, "config")), "AKIA") {
		t.Error("old credentials not removed on retry")
	}
}
