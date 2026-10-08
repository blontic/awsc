package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentStatus_FromSession(t *testing.T) {
	setupHome(t, twoOrgsYAML)
	t.Setenv("AWSC_PROFILE", "")
	if err := SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "alpha"); err != nil {
		t.Fatal(err)
	}

	got, err := CurrentStatus("", "")
	if err != nil {
		t.Fatal(err)
	}
	want := Status{Org: "alpha", AccountName: "prod", AccountID: "111111111111", RoleName: "Admin", Profile: "awsc-prod", Region: "us-east-1"}
	if *got != want {
		t.Errorf("CurrentStatus = %+v, want %+v", *got, want)
	}

	if got, err := CurrentStatus("", "ap-southeast-2"); err != nil || got.Region != "ap-southeast-2" {
		t.Errorf("--region should override the org's region: %+v %v", got, err)
	}
}

func TestCurrentStatus_FromAWSCProfile(t *testing.T) {
	home := setupHome(t, twoOrgsYAML)
	writeFile(t, filepath.Join(home, ".aws", "config"), "[profile awsc-dev]\n# Account: dev\nsso_session = awsc-alpha\nsso_account_id = 222222222222\nsso_role_name = ReadOnly\n\n[profile static]\nregion = us-east-1\n")

	t.Setenv("AWSC_PROFILE", "awsc-dev")
	got, err := CurrentStatus("", "")
	if err != nil || got.AccountName != "dev" || got.AccountID != "222222222222" || got.RoleName != "ReadOnly" || got.Profile != "awsc-dev" {
		t.Errorf("unexpected status %+v %v", got, err)
	}

	for _, p := range []string{"static", "missing"} {
		t.Setenv("AWSC_PROFILE", p)
		if _, err := CurrentStatus("", ""); err == nil {
			t.Errorf("expected an error for profile %q", p)
		}
	}
}

func TestCurrentStatus_NoAccount(t *testing.T) {
	setupHome(t, twoOrgsYAML)
	t.Setenv("AWSC_PROFILE", "")
	if _, err := CurrentStatus("", ""); err == nil || !strings.Contains(err.Error(), "awsc login") {
		t.Errorf("expected a 'run awsc login' error, got %v", err)
	}

	// A session for another org does not count for the requested org.
	if err := SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := CurrentStatus("beta", ""); err == nil || !strings.Contains(err.Error(), `"beta"`) {
		t.Errorf("expected no account for org beta, got %v", err)
	}
}

func TestCurrentStatus_DoesNotWriteFiles(t *testing.T) {
	home := setupHome(t, twoOrgsYAML)
	t.Setenv("AWSC_PROFILE", "")
	if err := SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := CurrentStatus("", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".aws")); !os.IsNotExist(err) {
		t.Error("status must not create or sync ~/.aws/config")
	}
}

func TestCurrentStatus_NoOrgs(t *testing.T) {
	setupHome(t, "")
	if _, err := CurrentStatus("", ""); err == nil || !strings.Contains(err.Error(), "awsc config add") {
		t.Errorf("expected a 'no orgs' error, got %v", err)
	}
}
