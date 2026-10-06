package aws

import (
	"fmt"
	"os"

	awscconfig "github.com/blontic/awsc/internal/config"
)

// location describes where a lookup ran, for "not found" messages, e.g.
// "wpl-wrk-prod (ap-southeast-2) as Admin".
func location(region string) string {
	if profile := os.Getenv("AWSC_PROFILE"); profile != "" {
		return fmt.Sprintf("profile %s (%s)", profile, region)
	}
	if session, err := awscconfig.GetCurrentSession(); err == nil {
		return fmt.Sprintf("%s (%s) as %s", session.AccountName, region, session.RoleName)
	}
	return "region " + region
}

// namedNotFoundError reports that a resource requested by name does not exist
// where the lookup ran.
func namedNotFoundError(kind, name, region string) error {
	return fmt.Errorf("%s '%s' not found in %s", kind, name, location(region))
}

// notFoundError reports that no resources of a kind exist where the lookup
// ran, and how to look elsewhere.
func notFoundError(what, region string) error {
	return fmt.Errorf("no %s found in %s; use -s to switch account or --region to change region", what, location(region))
}
