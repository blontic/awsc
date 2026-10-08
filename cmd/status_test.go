package cmd

import (
	"bytes"
	"testing"

	"github.com/blontic/awsc/internal/config"
)

func TestStatusCommand(t *testing.T) {
	if statusCmd.Use != "status" || statusCmd.Run == nil {
		t.Errorf("status command not set up: %+v", statusCmd)
	}
	for _, name := range []string{"short", "check"} {
		if statusCmd.Flags().Lookup(name) == nil {
			t.Errorf("missing --%s flag", name)
		}
	}
}

func TestStatusOutput(t *testing.T) {
	s := &config.Status{Org: "woodside", AccountName: "prod", AccountID: "111111111111", RoleName: "Admin", Profile: "awsc-prod", Region: "ap-southeast-2"}

	if got := shortStatus(s); got != "prod/Admin" {
		t.Errorf("shortStatus = %q", got)
	}
	if got := shortStatus(&config.Status{AccountID: "111111111111", RoleName: "Admin"}); got != "111111111111/Admin" {
		t.Errorf("shortStatus without account name = %q", got)
	}
	// Shell prompt metacharacters must never reach the prompt.
	evil := &config.Status{AccountName: "x$(touch /tmp/pwned)`id`%F{red}\x1b[2J", RoleName: "Admin;rm"}
	if got, want := shortStatus(evil), "x__touch /tmp/pwned__id__F_red___2J/Admin_rm"; got != want {
		t.Errorf("shortStatus = %q, want %q", got, want)
	}

	var buf bytes.Buffer
	printStatus(&buf, s)
	want := "Org:      woodside\nAccount:  prod (111111111111)\nRole:     Admin\nRegion:   ap-southeast-2\nProfile:  awsc-prod\n"
	if buf.String() != want {
		t.Errorf("printStatus =\n%s\nwant\n%s", buf.String(), want)
	}
}
