package aws

import "context"

// promptForReauth is PromptForReauth; a variable so tests can replace it.
var promptForReauth = PromptForReauth

// withReauth runs call. If it fails because credentials are missing or
// expired, the user is offered a new login; on success the manager's clients
// are reloaded via reload and call is retried once. If login is declined or
// fails, the original error is returned.
func withReauth[T any](ctx context.Context, reload func(context.Context) error, call func() (T, error)) (T, error) {
	result, err := call()
	if err == nil || !IsAuthError(err) {
		return result, err
	}
	if ok, reauthErr := promptForReauth(ctx); !ok || reauthErr != nil {
		return result, err
	}
	if err := reload(ctx); err != nil {
		var zero T
		return zero, err
	}
	return call()
}
