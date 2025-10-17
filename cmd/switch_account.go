package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
)

// handleAccountSwitch handles the account switching logic
// Returns error if switching fails
func handleAccountSwitch(ctx context.Context) error {
	// Unset AWSC_PROFILE so the new session takes priority
	os.Unsetenv("AWSC_PROFILE")

	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		return fmt.Errorf("error creating SSO manager: %w", err)
	}

	// Use existing SSO session, don't force reauth
	if err := ssoManager.RunLogin(ctx, false, "", ""); err != nil {
		return fmt.Errorf("error switching account: %w", err)
	}

	return nil
}
