package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdkconfig "github.com/aws/aws-sdk-go-v2/config"
)

// These tests cover every path the active org's settings flow through, so
// that the settings (formerly held in viper) stay wired up correctly.

func TestActiveSettings_AppliedByOrgCommands(t *testing.T) {
	want := Settings{Org: "beta", StartURL: "https://beta.awsapps.com/start", SSORegion: "eu-west-1", DefaultRegion: "eu-west-1"}

	t.Run("ActivateOrg", func(t *testing.T) {
		setupHome(t, twoOrgsYAML)
		if err := ActivateOrg("beta"); err != nil {
			t.Fatal(err)
		}
		if Active() != want {
			t.Errorf("Active() = %+v, want %+v", Active(), want)
		}
	})

	t.Run("UseOrg", func(t *testing.T) {
		setupHome(t, twoOrgsYAML)
		if err := UseOrg("beta"); err != nil {
			t.Fatal(err)
		}
		if Active() != want {
			t.Errorf("Active() = %+v, want %+v", Active(), want)
		}
	})

	t.Run("AddOrg", func(t *testing.T) {
		setupHome(t, "")
		stdin = strings.NewReader("https://gamma.awsapps.com/start\nap-southeast-2\nus-west-2\n\n")
		t.Cleanup(func() { stdin = os.Stdin })
		if _, err := AddOrg(""); err != nil {
			t.Fatal(err)
		}
		gamma := Settings{Org: "gamma", StartURL: "https://gamma.awsapps.com/start", SSORegion: "ap-southeast-2", DefaultRegion: "us-west-2"}
		if Active() != gamma {
			t.Errorf("Active() = %+v, want %+v", Active(), gamma)
		}
	})

	t.Run("no orgs leaves settings empty", func(t *testing.T) {
		setupHome(t, "")
		if err := ActivateOrg(""); err != nil {
			t.Fatal(err)
		}
		if Active() != (Settings{}) {
			t.Errorf("expected empty settings, got %+v", Active())
		}
	})
}

func TestSetActive(t *testing.T) {
	defer SetActive(Settings{})
	s := Settings{Org: "x", StartURL: "u", SSORegion: "r1", DefaultRegion: "r2"}
	SetActive(s)
	if Active() != s {
		t.Errorf("Active() = %+v, want %+v", Active(), s)
	}
}

func TestLoadAWSConfigWithProfile_UsesDefaultRegion(t *testing.T) {
	home := setupHome(t, "")
	path := filepath.Join(home, ".aws", "config")
	writeFile(t, path, "[profile test-profile]\nregion = us-west-1\n")
	original := sdkconfig.DefaultSharedConfigFiles
	sdkconfig.DefaultSharedConfigFiles = []string{path}
	defer func() { sdkconfig.DefaultSharedConfigFiles = original }()

	t.Setenv("AWSC_PROFILE", "test-profile")
	SetActive(Settings{Org: "alpha", DefaultRegion: "eu-west-2", SSORegion: "us-east-1"})

	cfg, err := LoadAWSConfigWithProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Region != "eu-west-2" {
		t.Errorf("service calls should use the active default region, got %s", cfg.Region)
	}
}
