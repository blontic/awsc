package aws

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/blontic/awsc/internal/aws/mocks"
	awscconfig "github.com/blontic/awsc/internal/config"
	"go.uber.org/mock/gomock"
)

const logoutTestConfig = `default_org: alpha
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

// setupLogoutHome creates an awsc config with orgs alpha and beta, a cached
// SSO token for each org in tokens, and a session for this terminal.
func setupLogoutHome(t *testing.T, tokens map[string]string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	write := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(home, ".awsc", "config.yaml"), logoutTestConfig)
	for org, content := range tokens {
		path, err := awscconfig.SSOTokenCachePath(org)
		if err != nil {
			t.Fatal(err)
		}
		write(path, content)
	}
	if err := awscconfig.SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "alpha"); err != nil {
		t.Fatal(err)
	}
	awscconfig.SetActive(awscconfig.Settings{Org: "alpha"})
	t.Cleanup(func() { awscconfig.SetActive(awscconfig.Settings{}) })
}

func tokenExists(t *testing.T, org string) bool {
	t.Helper()
	path, err := awscconfig.SSOTokenCachePath(org)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(path)
	return err == nil
}

func TestRunLogout_ActiveOrg(t *testing.T) {
	setupLogoutHome(t, map[string]string{
		"alpha": `{"accessToken":"alpha-token","region":"ap-south-1"}`,
		"beta":  `{"accessToken":"beta-token","region":"eu-west-1"}`,
	})
	ctrl := gomock.NewController(t)
	client := mocks.NewMockSSOLogoutClient(ctrl)
	client.EXPECT().Logout(gomock.Any(), &sso.LogoutInput{AccessToken: aws.String("alpha-token")}).Return(&sso.LogoutOutput{}, nil)

	var regions []string
	m, _ := NewLogoutManager(context.Background(), LogoutManagerOptions{Client: func(region string) SSOLogoutClient {
		regions = append(regions, region)
		return client
	}})
	if err := m.RunLogout(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	if tokenExists(t, "alpha") || !tokenExists(t, "beta") {
		t.Error("only the active org's token should be removed")
	}
	if len(regions) != 1 || regions[0] != "us-east-1" {
		t.Errorf("logout should use the org's configured SSO region, not the token file's, got %v", regions)
	}
	if _, err := awscconfig.GetCurrentSession(); err == nil {
		t.Error("this terminal's session should be cleared")
	}
}

func TestRunLogout_All(t *testing.T) {
	setupLogoutHome(t, map[string]string{
		"alpha": `{"accessToken":"alpha-token","region":"us-east-1"}`,
		"beta":  `{"accessToken":"beta-token"}`,
	})
	ctrl := gomock.NewController(t)
	client := mocks.NewMockSSOLogoutClient(ctrl)
	client.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(&sso.LogoutOutput{}, nil).Times(2)

	var regions []string
	m, _ := NewLogoutManager(context.Background(), LogoutManagerOptions{Client: func(region string) SSOLogoutClient {
		regions = append(regions, region)
		return client
	}})
	if err := m.RunLogout(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	if tokenExists(t, "alpha") || tokenExists(t, "beta") {
		t.Error("all tokens should be removed")
	}
	// Each org's configured SSO region is used.
	if len(regions) != 2 || regions[0] != "us-east-1" || regions[1] != "eu-west-1" {
		t.Errorf("unexpected regions %v", regions)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".awsc", "sessions")); !os.IsNotExist(err) {
		t.Error("all sessions should be cleared")
	}
}

func TestRunLogout_RemovesTokenWhenAWSUnreachable(t *testing.T) {
	setupLogoutHome(t, map[string]string{"alpha": `{"accessToken":"alpha-token","region":"us-east-1"}`})
	ctrl := gomock.NewController(t)
	client := mocks.NewMockSSOLogoutClient(ctrl)
	client.EXPECT().Logout(gomock.Any(), gomock.Any()).Return(nil, errors.New("network unreachable"))

	m, _ := NewLogoutManager(context.Background(), LogoutManagerOptions{Client: func(string) SSOLogoutClient { return client }})
	if err := m.RunLogout(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if tokenExists(t, "alpha") {
		t.Error("token should be removed even if AWS can't be reached")
	}
}

func TestRunLogout_NotLoggedIn(t *testing.T) {
	setupLogoutHome(t, nil)
	m, _ := NewLogoutManager(context.Background(), LogoutManagerOptions{Client: func(string) SSOLogoutClient {
		t.Error("AWS should not be called without a token")
		return nil
	}})
	if err := m.RunLogout(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := awscconfig.GetCurrentSession(); err == nil {
		t.Error("this terminal's session should still be cleared")
	}
}
