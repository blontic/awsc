package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/credentials/ssocreds"
)

var (
	alphaOrg = OrgConfig{SSO: SSOConfig{StartURL: "https://alpha.awsapps.com/start", Region: "us-east-1"}, DefaultRegion: "ap-southeast-2"}
	betaOrg  = OrgConfig{SSO: SSOConfig{StartURL: "https://beta.awsapps.com/start", Region: "eu-west-1"}, DefaultRegion: "eu-west-1"}
)

const alphaSession = `[sso-session awsc-alpha]
sso_start_url = https://alpha.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
`

func TestSyncSessions_CreatesAndIsIdempotent(t *testing.T) {
	content := "[default]\nregion = us-west-2\n"
	got, res, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil {
		t.Fatal(err)
	}
	if got != content+"\n"+alphaSession || len(res.Created) != 1 {
		t.Errorf("unexpected result (%+v):\n%s", res, got)
	}

	again, res, err := syncSessions(got, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil || again != got || len(res.Created)+len(res.Renamed)+len(res.Removed) != 0 {
		t.Errorf("second sync should be a no-op: %+v %v", res, err)
	}
}

func TestSyncSessions_UpdatesChangedSettingsInPlace(t *testing.T) {
	content := "[default]\n\n" + strings.Replace(alphaSession, "us-east-1", "us-west-2", 1) + "\n[profile other]\nregion = x\n"
	got, _, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil {
		t.Fatal(err)
	}
	if want := "[default]\n\n" + alphaSession + "\n[profile other]\nregion = x\n"; got != want {
		t.Errorf("expected in-place update:\n%s", got)
	}
}

func TestSyncSessions_RenamesSessionForSameStartURL(t *testing.T) {
	content := strings.Replace(alphaSession, "awsc-alpha", "awsc-old", 1) + `
[profile awsc-a/R]
sso_session = awsc-old
sso_account_id = 111111111111
sso_role_name = R
`
	got, res, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil {
		t.Fatal(err)
	}
	if res.Renamed["awsc-old"] != "awsc-alpha" || strings.Contains(got, "awsc-old") {
		t.Errorf("expected rename to awsc-alpha (%+v):\n%s", res, got)
	}
	if !strings.Contains(got, "[profile awsc-a/R]\nsso_session = awsc-alpha") || strings.Count(got, "[sso-session") != 1 {
		t.Errorf("profile not repointed:\n%s", got)
	}
}

func TestSyncSessions_RemovesOrphansAndTheirProfiles(t *testing.T) {
	content := alphaSession + `
[sso-session awsc-gone]
sso_start_url = https://gone.awsapps.com/start
sso_region = us-east-1

[profile awsc-gone-acct/R]
sso_session = awsc-gone
sso_account_id = 111111111111
sso_role_name = R

[profile awsc-custom]
sso_session = awsc-gone
role_arn = arn:aws:iam::111111111111:role/X

[profile awsc-no-session/R]
sso_session = awsc-deleted-by-hand
sso_account_id = 222222222222
sso_role_name = R

[sso-session work]
sso_start_url = https://gone.awsapps.com/start
`
	got, res, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "[sso-session awsc-gone]") || strings.Contains(got, "awsc-gone-acct") {
		t.Errorf("orphaned session and profile not removed:\n%s", got)
	}
	if strings.Contains(got, "awsc-no-session") {
		t.Errorf("profile whose session no longer exists not removed:\n%s", got)
	}
	if len(res.Removed) != 1 || len(res.RemovedProfiles) != 2 {
		t.Errorf("unexpected result: %+v", res)
	}
	for _, keep := range []string{"[profile awsc-custom]", "role_arn", "[sso-session work]", "[sso-session awsc-alpha]"} {
		if !strings.Contains(got, keep) {
			t.Errorf("%q should be kept:\n%s", keep, got)
		}
	}
}

func TestSyncSessions_RemovesProfilesWithoutRole(t *testing.T) {
	content := alphaSession + `
[profile awsc-prod]
sso_session = awsc-alpha
sso_account_id = 111111111111
sso_role_name = Admin

[profile awsc-custom]
sso_session = awsc-alpha
role_arn = arn:aws:iam::111111111111:role/X

[profile awsc-prod/Admin]
sso_session = awsc-alpha
sso_account_id = 111111111111
sso_role_name = Admin
`
	got, res, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "[profile awsc-prod]") || len(res.RemovedProfiles) != 1 {
		t.Errorf("profile without a role not removed (%+v):\n%s", res, got)
	}
	for _, keep := range []string{"[profile awsc-custom]", "[profile awsc-prod/Admin]"} {
		if !strings.Contains(got, keep) {
			t.Errorf("%q should be kept:\n%s", keep, got)
		}
	}
}

func TestSyncSessions_RefusesUserOwnedSessionName(t *testing.T) {
	content := "[sso-session awsc-alpha]\nsso_start_url = https://alpha.awsapps.com/start\ncustom = x\n"
	if _, _, err := syncSessions(content, map[string]OrgConfig{"alpha": alphaOrg}); err == nil {
		t.Error("expected error for a user-customised session with an org's name")
	}
}

func TestUpsertProfile(t *testing.T) {
	p := Profile{Org: "alpha", AccountName: "prod", AccountID: "111111111111", RoleName: "Admin"}

	got, name, err := upsertProfile("[default]\n", p, "ap-southeast-2")
	if err != nil || name != "awsc-prod/Admin" {
		t.Fatalf("upsertProfile = %q, %v", name, err)
	}
	want := `[default]

[profile awsc-prod/Admin]
# Account: prod
sso_session = awsc-alpha
sso_account_id = 111111111111
sso_role_name = Admin
region = ap-southeast-2
`
	if got != want {
		t.Errorf("unexpected content:\n%s", got)
	}

	again, _, err := upsertProfile(got, p, "ap-southeast-2")
	if err != nil || again != got {
		t.Errorf("expected rewriting the same profile to be a no-op (%v):\n%s", err, again)
	}

	// Another role in the same account gets its own profile alongside.
	p.RoleName = "ReadOnly"
	both, name, err := upsertProfile(got, p, "ap-southeast-2")
	if err != nil || name != "awsc-prod/ReadOnly" || !strings.Contains(both, "[profile awsc-prod/Admin]") || !strings.Contains(both, "[profile awsc-prod/ReadOnly]") {
		t.Errorf("expected separate profiles per role (%q, %v):\n%s", name, err, both)
	}

	// Same account name in another org gets a prefixed profile.
	other := Profile{Org: "beta", AccountName: "prod", AccountID: "222222222222", RoleName: "Admin"}
	if _, name, _ := upsertProfile(got, other, ""); name != "awsc-beta-prod/Admin" {
		t.Errorf("expected prefixed name for clash, got %s", name)
	}

	// User-customised profile is never overwritten.
	p.RoleName = "Admin"
	custom := "[profile awsc-prod/Admin]\nrole_arn = arn:aws:iam::111111111111:role/X\n"
	if _, _, err := upsertProfile(custom, p, ""); err == nil {
		t.Error("expected error overwriting a user profile")
	}
}

func TestWriteProfile(t *testing.T) {
	home := setupHome(t, "")
	path := filepath.Join(home, ".aws", "config")
	writeFile(t, path, "[default]\nregion = us-west-2\n")
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	orgs := map[string]OrgConfig{"alpha": alphaOrg}
	name, err := WriteProfile(orgs, Profile{Org: "alpha", AccountName: "prod", AccountID: "111111111111", RoleName: "Admin"})
	if err != nil || name != "awsc-prod/Admin" {
		t.Fatalf("WriteProfile = %q, %v", name, err)
	}

	got := readFile(t, path)
	for _, want := range []string{"[default]", "[sso-session awsc-alpha]", "[profile awsc-prod/Admin]", "sso_session = awsc-alpha"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	for _, secret := range []string{"aws_access_key_id", "aws_secret_access_key", "aws_session_token"} {
		if strings.Contains(got, secret) {
			t.Errorf("credentials must never be written: %q", secret)
		}
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0644 {
		t.Errorf("existing permissions should be kept, got %o", info.Mode().Perm())
	}
	if readFile(t, path+".awsc.bak") != "[default]\nregion = us-west-2\n" {
		t.Error("backup of previous config not written")
	}

	if _, err := WriteProfile(orgs, Profile{Org: "alpha", AccountName: "bad\n[x]", AccountID: "1", RoleName: "R"}); err == nil {
		t.Error("expected INI injection to be rejected")
	}
	if _, err := WriteProfile(orgs, Profile{Org: "nope", AccountName: "a", AccountID: "1", RoleName: "R"}); err == nil {
		t.Error("expected error for unknown org")
	}
}

func TestWriteProfile_PreservesSymlink(t *testing.T) {
	home := setupHome(t, "")
	target := filepath.Join(home, "dotfiles-config")
	writeFile(t, target, "[default]\n")
	link := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if _, err := WriteProfile(map[string]OrgConfig{"alpha": alphaOrg}, Profile{Org: "alpha", AccountName: "a", AccountID: "1", RoleName: "R"}); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced with a regular file")
	}
	if !strings.Contains(readFile(t, target), "[profile awsc-a/R]") {
		t.Error("profile not written to symlink target")
	}
}

func TestSyncAWSConfig_RestoresDeletedFile(t *testing.T) {
	home := setupHome(t, "")
	orgs := map[string]OrgConfig{"alpha": alphaOrg, "beta": betaOrg}

	if _, err := syncAWSConfig(orgs); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".aws", "config")
	first := readFile(t, path)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := syncAWSConfig(orgs); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != first {
		t.Errorf("sessions not restored after deletion:\n%s", got)
	}
}

func TestSyncAWSConfig_MovesAndRemovesTokens(t *testing.T) {
	home := setupHome(t, "")
	writeFile(t, filepath.Join(home, ".aws", "config"),
		strings.Replace(alphaSession, "awsc-alpha", "awsc-old", 1)+"\n[sso-session awsc-gone]\nsso_start_url = https://gone.awsapps.com/start\n")

	tokenPath := func(session string) string {
		p, _ := ssocreds.StandardCachedTokenFilepath(session)
		return p
	}
	writeFile(t, tokenPath("awsc-old"), `{"accessToken":"alpha-token"}`)
	writeFile(t, tokenPath("awsc-gone"), `{"accessToken":"gone-token"}`)

	if _, err := syncAWSConfig(map[string]OrgConfig{"alpha": alphaOrg}); err != nil {
		t.Fatal(err)
	}
	if readFile(t, tokenPath("awsc-alpha")) != `{"accessToken":"alpha-token"}` {
		t.Error("renamed session's token not moved")
	}
	for _, gone := range []string{"awsc-old", "awsc-gone"} {
		if _, err := os.Stat(tokenPath(gone)); !os.IsNotExist(err) {
			t.Errorf("token for %s should be removed", gone)
		}
	}
}

func TestSectionHeaderWithTrailingComment(t *testing.T) {
	content := "[sso-session awsc-gone]\nsso_start_url = https://gone.awsapps.com/start\n[default] # main\nregion = us-west-2\n"
	got, _, err := syncSessions(content, map[string]OrgConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "[default] # main\nregion = us-west-2\n" {
		t.Errorf("header with trailing comment not recognised:\n%s", got)
	}
}
