package aws

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	awscconfig "github.com/blontic/awsc/internal/config"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewCredentialsManager(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials but shouldn't panic
	_, err := NewCredentialsManager(ctx)
	if err != nil {
		t.Logf("NewCredentialsManager failed as expected in test environment: %v", err)
	} else {
		t.Log("NewCredentialsManager succeeded unexpectedly")
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "auth failure error",
			err:      fmt.Errorf("AuthFailure: invalid credentials"),
			expected: true,
		},
		{
			name:     "signature error",
			err:      fmt.Errorf("SignatureDoesNotMatch: signature mismatch"),
			expected: true,
		},
		{
			name:     "expired token error",
			err:      fmt.Errorf("ExpiredToken: token has expired"),
			expected: true,
		},
		{
			name:     "invalid token error",
			err:      fmt.Errorf("InvalidToken: token is invalid"),
			expected: true,
		},
		{
			name:     "request expired error (expired SSO credentials reported as clock skew)",
			err:      fmt.Errorf("operation error EC2: DescribeInstances, exceeded maximum number of attempts, 3, Probable clock skew error: api error RequestExpired: Request has expired."),
			expected: true,
		},
		{
			name:     "get credentials error",
			err:      fmt.Errorf("failed to get credentials"),
			expected: true,
		},
		{
			name:     "no active session",
			err:      fmt.Errorf("no active session"),
			expected: true,
		},
		{
			name:     "missing SSO token",
			err:      fmt.Errorf("no SSO cache found, please run 'awsc login'"),
			expected: true,
		},
		{
			name:     "profile missing",
			err:      fmt.Errorf("failed to get shared config profile, awsc-prod"),
			expected: true,
		},
		{
			name:     "SDK could not refresh SSO role credentials",
			err:      fmt.Errorf("operation error RDS: DescribeDBInstances, get identity: get credentials: failed to refresh cached credentials, refresh cached SSO token failed, unable to refresh SSO token"),
			expected: true,
		},
		{
			name:     "wrapped no active session",
			err:      fmt.Errorf("failed to load AWS config: %w", fmt.Errorf("no active session")),
			expected: true,
		},
		{
			name:     "DNS failure is not an auth error",
			err:      fmt.Errorf("dial tcp: lookup rds.ap-southeast-2.amazonaws.com: no such host"),
			expected: false,
		},
		{
			name:     "malformed config is not an auth error",
			err:      fmt.Errorf("failed to load AWS config: failed to parse ini file"),
			expected: false,
		},
		{
			name:     "throttling is not an auth error",
			err:      fmt.Errorf("operation error EC2: DescribeInstances, api error Throttling: Rate exceeded"),
			expected: false,
		},
		{
			name:     "permission error (not auth error)",
			err:      fmt.Errorf("User is not authorized to perform action"),
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func writeTestCache(t *testing.T, cache ssoCache) string {
	t.Helper()
	path, err := ssoTokenCachePath()
	if err != nil {
		t.Fatalf("ssoTokenCachePath failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("Failed to write cache file: %v", err)
	}
	return path
}

func TestSSOTokenCachePath_MatchesAWSCLI(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	awscconfig.SetActive(awscconfig.Settings{Org: "my-org"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

	path, err := ssoTokenCachePath()
	if err != nil {
		t.Fatalf("ssoTokenCachePath failed: %v", err)
	}
	h := sha1.New()
	h.Write([]byte("awsc-my-org"))
	expected := filepath.Join(tempDir, ".aws", "sso", "cache", fmt.Sprintf("%x.json", h.Sum(nil)))
	if path != expected {
		t.Errorf("Expected %s, got %s", expected, path)
	}
}

func TestCredentialsManager_GetCachedToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	awscconfig.SetActive(awscconfig.Settings{StartURL: "https://test.awsapps.com/start"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

	manager := &CredentialsManager{}
	ctx := context.Background()

	if _, err := manager.GetCachedToken(ctx); err == nil || !IsAuthError(err) {
		t.Errorf("Expected auth error when no cache exists, got %v", err)
	}

	writeTestCache(t, ssoCache{
		AccessToken: "valid-token",
		ExpiresAt:   time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		Region:      "us-east-1",
		StartURL:    "https://test.awsapps.com/start",
	})

	token, err := manager.GetCachedToken(ctx)
	if err != nil {
		t.Fatalf("GetCachedToken failed: %v", err)
	}
	if *token != "valid-token" {
		t.Errorf("Expected 'valid-token', got %s", *token)
	}
}

func TestCredentialsManager_GetCachedToken_ExpiredWithoutRefreshToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	awscconfig.SetActive(awscconfig.Settings{StartURL: "https://test.awsapps.com/start"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

	writeTestCache(t, ssoCache{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		Region:      "us-east-1",
		StartURL:    "https://test.awsapps.com/start",
	})

	manager := &CredentialsManager{}
	_, err := manager.GetCachedToken(context.Background())
	if err == nil {
		t.Fatal("Expected error for expired token without refresh token")
	}
	if !IsAuthError(err) {
		t.Errorf("Expected expired token error to be an auth error, got %v", err)
	}
}

func TestSaveTokenToCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	in := ssoCache{
		StartURL:              "https://test.awsapps.com/start",
		Region:                "us-east-1",
		AccessToken:           "test-access-token",
		ExpiresAt:             time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		RefreshToken:          "test-refresh-token",
		ClientID:              "client-id",
		ClientSecret:          "client-secret",
		RegistrationExpiresAt: time.Now().UTC().Add(90 * 24 * time.Hour).Format(time.RFC3339),
	}
	if err := saveTokenToCache(in); err != nil {
		t.Fatalf("saveTokenToCache failed: %v", err)
	}

	path, _ := ssoTokenCachePath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("cache file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected 0600 permissions, got %o", info.Mode().Perm())
	}

	got := loadCache()
	if got == nil || *got != in {
		t.Errorf("Round-tripped cache mismatch: got %+v, want %+v", got, in)
	}
}

func TestSaveTokenToCache_TightensExistingPermissions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := writeTestCache(t, ssoCache{AccessToken: "old"})
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	if err := saveTokenToCache(ssoCache{AccessToken: "new"}); err != nil {
		t.Fatalf("saveTokenToCache failed: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "authorization pending",
			err:      fmt.Errorf("AuthorizationPendingException: authorization pending"),
			expected: true,
		},
		{
			name:     "slow down",
			err:      fmt.Errorf("SlowDownException: slow down"),
			expected: true,
		},
		{
			name:     "authorization_pending",
			err:      fmt.Errorf("authorization_pending"),
			expected: true,
		},
		{
			name:     "slow_down",
			err:      fmt.Errorf("slow_down"),
			expected: true,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for error: %v", tt.expected, result, tt.err)
			}
		})
	}
}

func TestCredentialsManager_GetCachedToken_InvalidJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	awscconfig.SetActive(awscconfig.Settings{StartURL: "https://test.awsapps.com/start"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

	path, _ := ssoTokenCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	manager := &CredentialsManager{}
	if _, err := manager.GetCachedToken(context.Background()); err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestBrowserCommand(t *testing.T) {
	cases := []struct {
		goos     string
		wsl      bool
		wantName string
	}{
		{"darwin", false, "open"},
		{"linux", false, "xdg-open"},
		{"linux", true, "cmd.exe"},
	}
	for _, c := range cases {
		if name, _ := browserCommand(c.goos, c.wsl); name != c.wantName {
			t.Errorf("browserCommand(%s, wsl=%v) = %s, want %s", c.goos, c.wsl, name, c.wantName)
		}
	}
}

func TestOpenBrowser_RejectsNonHTTPS(t *testing.T) {
	for _, u := range []string{"invalid-url", "http://example.com", "file:///etc/passwd", "https://"} {
		if err := openBrowser(u); err == nil {
			t.Errorf("openBrowser(%q) should be rejected", u)
		}
	}
}

func TestBrowserLogin(t *testing.T) {
	const url, code = "https://x.awsapps.com/start/#/device?user_code=ABCD-EFGH", "ABCD-EFGH"

	t.Run("interactive: shows code, waits for Enter, then opens", func(t *testing.T) {
		var out strings.Builder
		var opened []string
		err := browserLogin(strings.NewReader("\n"), &out, true, "woodside", url, code, func(u string) error {
			opened = append(opened, u)
			return nil
		})
		if err != nil || len(opened) != 1 || opened[0] != url {
			t.Fatalf("err=%v opened=%v", err, opened)
		}
		msg := out.String()
		for _, want := range []string{`org "woodside"`, code, "Press Enter to open " + url} {
			if !strings.Contains(msg, want) {
				t.Errorf("output missing %q:\n%s", want, msg)
			}
		}
		if strings.Index(msg, code) > strings.Index(msg, "Press Enter") {
			t.Error("the code should be shown before the browser is opened")
		}
	})

	t.Run("interactive: no Enter (EOF) cancels without opening", func(t *testing.T) {
		opened := false
		err := browserLogin(strings.NewReader(""), io.Discard, true, "o", url, code, func(string) error { opened = true; return nil })
		if err == nil || opened {
			t.Errorf("err=%v opened=%v", err, opened)
		}
	})

	t.Run("interactive: browser fails to open, URL still shown", func(t *testing.T) {
		var out strings.Builder
		err := browserLogin(strings.NewReader("\n"), &out, true, "o", url, code, func(string) error { return errors.New("no display") })
		if err != nil || !strings.Contains(out.String(), "Could not open the browser") || !strings.Contains(out.String(), url) {
			t.Errorf("err=%v out=%s", err, out.String())
		}
	})

	t.Run("non-interactive: prints URL, does not wait or open", func(t *testing.T) {
		var out strings.Builder
		opened := false
		err := browserLogin(strings.NewReader(""), &out, false, "o", url, code, func(string) error { opened = true; return nil })
		if err != nil || opened {
			t.Fatalf("err=%v opened=%v", err, opened)
		}
		if !strings.Contains(out.String(), "Open this URL to approve the login: "+url) || strings.Contains(out.String(), "Press Enter") {
			t.Errorf("unexpected output:\n%s", out.String())
		}
	})
}
