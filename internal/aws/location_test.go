package aws

import (
	"os"
	"strings"
	"testing"

	awscconfig "github.com/blontic/awsc/internal/config"
)

func TestLocation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWSC_PROFILE", "")

	if got := location("ap-southeast-2"); got != "region ap-southeast-2" {
		t.Errorf("no session: got %q", got)
	}

	if err := awscconfig.SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "woodside"); err != nil {
		t.Fatal(err)
	}
	if got := location("ap-southeast-2"); got != "prod (ap-southeast-2) as Admin" {
		t.Errorf("with session: got %q", got)
	}

	t.Setenv("AWSC_PROFILE", "my-profile")
	if got := location("us-east-1"); got != "profile my-profile (us-east-1)" {
		t.Errorf("with AWSC_PROFILE: got %q", got)
	}
}

func TestNotFoundError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWSC_PROFILE", "")
	if err := awscconfig.SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "woodside"); err != nil {
		t.Fatal(err)
	}

	msg := notFoundError("OpenSearch domains", "ap-southeast-2").Error()
	for _, want := range []string{"no OpenSearch domains found", "prod (ap-southeast-2) as Admin", "-s", "--region"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q should contain %q", msg, want)
		}
	}
}
