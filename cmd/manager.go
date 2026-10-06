package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/blontic/awsc/internal/aws"
)

// Login steps used by newManager; variables so tests can replace them.
var (
	reauthFn        = aws.PromptForReauth
	switchAccountFn = handleAccountSwitch
)

// newManager creates an AWS service manager for the terminal's account. If
// the terminal has no usable credentials it runs login first; otherwise, if
// switch is set, it lets the user pick another account. Either way the manager
// is created for the account the terminal ends up using.
func newManager[T any](ctx context.Context, create func(context.Context) (T, error), switchFirst bool) (T, error) {
	m, err := create(ctx)
	switch {
	case err != nil && aws.IsAuthError(err):
		ok, reauthErr := reauthFn(ctx)
		if reauthErr != nil {
			return m, reauthErr
		}
		if !ok {
			return m, errors.New("authentication cancelled")
		}
		// Login already included choosing an account, so -s is not repeated.
	case err != nil:
		return m, err
	case switchFirst:
		if err := switchAccountFn(ctx); err != nil {
			return m, err
		}
	default:
		return m, nil
	}
	return create(ctx)
}

// handleAccountSwitch lets the user pick another account and role for this
// terminal, reusing the current SSO login.
func handleAccountSwitch(ctx context.Context) error {
	// Unset AWSC_PROFILE so the new session takes priority
	os.Unsetenv("AWSC_PROFILE")

	ssoManager, err := aws.NewSSOManager(ctx)
	if err != nil {
		return fmt.Errorf("error creating SSO manager: %w", err)
	}
	if err := ssoManager.RunLogin(ctx, false, "", ""); err != nil {
		return fmt.Errorf("error switching account: %w", err)
	}
	return nil
}

// exitOnError prints err to stderr and exits with status 1.
func exitOnError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n✗ Error: %v\n", err)
		os.Exit(1)
	}
}
