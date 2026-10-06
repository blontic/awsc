package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const twoOrgsYAML = `default_org: alpha
orgs:
  alpha:
    sso:
      start_url: https://alpha.awsapps.com/start
      region: us-east-1
    default_region: us-east-1
  beta:
    sso:
      start_url: https://beta.awsapps.com/start
      region: eu-west-1
    default_region: eu-west-1
`

// setupHome points HOME at a temp dir with the given awsc config (if any) and
// resets global state.
func setupHome(t *testing.T, awscConfig string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(OrgEnvVar, "")
	active = Settings{}
	t.Cleanup(func() { active = Settings{} })
	if awscConfig != "" {
		writeFile(t, filepath.Join(home, ".awsc", "config.yaml"), awscConfig)
	}
	return home
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSuggestOrgName(t *testing.T) {
	cases := map[string]string{
		"https://d-976710a35d.awsapps.com/start": "d-976710a35d",
		"https://My_Org.awsapps.com/start/":      "my-org",
	}
	for url, want := range cases {
		if got := suggestOrgName(url); got != want {
			t.Errorf("suggestOrgName(%q) = %q, want %q", url, got, want)
		}
	}
	if got := suggestOrgName("https://identitycenter.example.com/start"); len(got) != 8 {
		t.Errorf("expected 8-char hash for non-awsapps URL, got %q", got)
	}
}

func TestReadWriteFileConfig(t *testing.T) {
	setupHome(t, "")

	cfg, err := ReadFileConfig()
	if err != nil || len(cfg.Orgs) != 0 {
		t.Fatalf("missing file should give empty config: %+v %v", cfg, err)
	}

	cfg.DefaultOrg = "alpha"
	cfg.Orgs["alpha"] = OrgConfig{SSO: SSOConfig{StartURL: "https://alpha.awsapps.com/start", Region: "us-east-1"}, DefaultRegion: "us-east-1"}
	if err := writeFileConfig(cfg); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(GetConfigPath())
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}

	got, err := ReadFileConfig()
	if err != nil || got.DefaultOrg != "alpha" || got.Orgs["alpha"] != cfg.Orgs["alpha"] {
		t.Errorf("round trip mismatch: %+v %v", got, err)
	}
	if strings.Contains(readFile(t, GetConfigPath()), "\nsso:") {
		t.Error("no top-level sso key should be written")
	}
}

func TestResolveActiveOrg_Priority(t *testing.T) {
	setupHome(t, twoOrgsYAML)
	cfg, err := ReadFileConfig()
	if err != nil {
		t.Fatal(err)
	}

	check := func(flag, want string) {
		t.Helper()
		if got, err := resolveActiveOrg(cfg, flag); err != nil || got != want {
			t.Errorf("resolveActiveOrg(%q) = %q, %v; want %q", flag, got, err, want)
		}
	}

	check("", "alpha") // default

	if err := SaveSession(os.Getppid(), "awsc-x", "1", "x", "r", "beta"); err != nil {
		t.Fatal(err)
	}
	check("", "beta") // terminal session

	t.Setenv(OrgEnvVar, "alpha")
	check("", "alpha")    // AWSC_ORG
	check("beta", "beta") // --org

	if _, err := resolveActiveOrg(cfg, "nope"); err == nil {
		t.Error("expected error for unknown --org")
	}
	t.Setenv(OrgEnvVar, "nope")
	if _, err := resolveActiveOrg(cfg, ""); err == nil {
		t.Error("expected error for unknown AWSC_ORG")
	}
}

func TestActivateOrg(t *testing.T) {
	home := setupHome(t, twoOrgsYAML)

	if err := ActivateOrg("beta"); err != nil {
		t.Fatal(err)
	}
	if active.Org != "beta" ||
		active.StartURL != "https://beta.awsapps.com/start" ||
		active.SSORegion != "eu-west-1" ||
		active.DefaultRegion != "eu-west-1" {
		t.Errorf("beta settings not applied: %v", active)
	}

	// ~/.aws/config is created with a session per org.
	awsConfig := readFile(t, filepath.Join(home, ".aws", "config"))
	for _, want := range []string{"[sso-session awsc-alpha]", "[sso-session awsc-beta]"} {
		if !strings.Contains(awsConfig, want) {
			t.Errorf("missing %s:\n%s", want, awsConfig)
		}
	}
}

func TestActivateOrg_MultipleOrgsWithoutDefault(t *testing.T) {
	setupHome(t, strings.Replace(twoOrgsYAML, "default_org: alpha\n", "", 1))
	if err := ActivateOrg(""); err == nil {
		t.Error("expected error asking the user to pick an org")
	}
}

func TestActivateOrg_NoOrgsLeavesAWSConfigAlone(t *testing.T) {
	home := setupHome(t, "")
	existing := "[sso-session awsc-alpha]\nsso_start_url = https://alpha.awsapps.com/start\nsso_region = us-east-1\n"
	writeFile(t, filepath.Join(home, ".aws", "config"), existing)

	if err := ActivateOrg(""); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(home, ".aws", "config")); got != existing {
		t.Errorf("~/.aws/config changed with no orgs configured:\n%s", got)
	}
}

func TestAddUseRemoveOrg(t *testing.T) {
	home := setupHome(t, twoOrgsYAML)
	awsConfigPath := filepath.Join(home, ".aws", "config")

	stdin = strings.NewReader("https://gamma.awsapps.com/start\nap-southeast-2\nap-southeast-2\n\n")
	t.Cleanup(func() { stdin = os.Stdin })
	name, err := AddOrg("")
	if err != nil || name != "gamma" {
		t.Fatalf("AddOrg = %q, %v", name, err)
	}
	if !strings.Contains(readFile(t, awsConfigPath), "[sso-session awsc-gamma]\nsso_start_url = https://gamma.awsapps.com/start") {
		t.Error("adding an org should create its sso-session")
	}

	if _, err := AddOrg("alpha"); err == nil {
		t.Error("expected error adding duplicate org")
	}
	if _, err := AddOrg("Bad Name"); err == nil {
		t.Error("expected error for invalid org name")
	}

	stdin = strings.NewReader("https://alpha.awsapps.com/start/\nus-east-1\nus-east-1\n")
	if _, err := AddOrg("alpha-again"); err == nil {
		t.Error("expected error adding an org whose start URL is already configured")
	}

	if err := UseOrg("beta"); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := ReadFileConfig(); cfg.DefaultOrg != "beta" {
		t.Errorf("default not updated: %q", cfg.DefaultOrg)
	}
	if err := UseOrg("nope"); err == nil {
		t.Error("expected error for unknown org")
	}

	if err := RemoveOrg("gamma"); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := ReadFileConfig(); len(cfg.Orgs) != 2 {
		t.Errorf("gamma not removed: %+v", cfg.Orgs)
	}
	if strings.Contains(readFile(t, awsConfigPath), "awsc-gamma") {
		t.Error("removing an org should remove its sso-session")
	}
}

func TestRemoveOrg_Default(t *testing.T) {
	gamma := `  gamma:
    sso:
      start_url: https://gamma.awsapps.com/start
      region: us-east-1
    default_region: us-east-1
`
	t.Run("one org left becomes the default", func(t *testing.T) {
		setupHome(t, twoOrgsYAML)
		if err := RemoveOrg("alpha"); err != nil {
			t.Fatal(err)
		}
		if cfg, _ := ReadFileConfig(); cfg.DefaultOrg != "beta" {
			t.Errorf("expected beta as default, got %q", cfg.DefaultOrg)
		}
	})

	t.Run("several orgs left: no default", func(t *testing.T) {
		setupHome(t, twoOrgsYAML+gamma)
		if err := RemoveOrg("alpha"); err != nil {
			t.Fatal(err)
		}
		cfg, _ := ReadFileConfig()
		if cfg.DefaultOrg != "" || len(cfg.Orgs) != 2 {
			t.Errorf("expected no default and 2 orgs, got %+v", cfg)
		}
		if err := ActivateOrg(""); err == nil {
			t.Error("expected an error asking the user to pick an org")
		}
	})
}

func TestUseOrg_SwitchesCurrentTerminal(t *testing.T) {
	setupHome(t, twoOrgsYAML)
	if err := SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "alpha"); err != nil {
		t.Fatal(err)
	}

	if err := UseOrg("beta"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetCurrentSession(); err == nil {
		t.Error("terminal session for the previous org should be cleared")
	}
	if got := active.Org; got != "beta" {
		t.Errorf("active org = %q, want beta", got)
	}

	// The next command resolves to the new default.
	if err := ActivateOrg(""); err != nil || active.Org != "beta" {
		t.Errorf("next command should use beta: active=%q err=%v", active.Org, err)
	}
}

func TestUseOrg_KeepsSessionForSameOrg(t *testing.T) {
	setupHome(t, twoOrgsYAML)
	if err := SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "beta"); err != nil {
		t.Fatal(err)
	}
	if err := UseOrg("beta"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetCurrentSession(); err != nil {
		t.Error("session for the chosen org should be kept")
	}
}
