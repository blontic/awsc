package aws

import (
	"context"
	"errors"
	"testing"
)

func stubReauth(t *testing.T, ok bool, err error) *int {
	t.Helper()
	calls := new(int)
	orig := promptForReauth
	t.Cleanup(func() { promptForReauth = orig })
	promptForReauth = func(context.Context) (bool, error) { *calls++; return ok, err }
	return calls
}

// sequence returns a call that yields the given errors in turn, then "ok".
func sequence(errs ...error) (func() (string, error), *int) {
	n := 0
	return func() (string, error) {
		n++
		if n <= len(errs) && errs[n-1] != nil {
			return "", errs[n-1]
		}
		return "ok", nil
	}, &n
}

func TestWithReauth(t *testing.T) {
	ctx := context.Background()
	expired := errors.New("api error ExpiredToken: token expired")
	reloaded := 0
	reload := func(context.Context) error { reloaded++; return nil }

	t.Run("success: no login", func(t *testing.T) {
		prompts := stubReauth(t, true, nil)
		call, n := sequence()
		got, err := withReauth(ctx, reload, call)
		if got != "ok" || err != nil || *n != 1 || *prompts != 0 {
			t.Errorf("got=%q err=%v calls=%d prompts=%d", got, err, *n, *prompts)
		}
	})

	t.Run("non-auth error: returned as is", func(t *testing.T) {
		prompts := stubReauth(t, true, nil)
		call, n := sequence(errors.New("Throttling"))
		if _, err := withReauth(ctx, reload, call); err == nil || *n != 1 || *prompts != 0 {
			t.Errorf("err=%v calls=%d prompts=%d", err, *n, *prompts)
		}
	})

	t.Run("auth error: login, reload, retry", func(t *testing.T) {
		stubReauth(t, true, nil)
		reloaded = 0
		call, n := sequence(expired)
		got, err := withReauth(ctx, reload, call)
		if got != "ok" || err != nil || *n != 2 || reloaded != 1 {
			t.Errorf("got=%q err=%v calls=%d reloaded=%d", got, err, *n, reloaded)
		}
	})

	t.Run("login declined: original error, no retry", func(t *testing.T) {
		stubReauth(t, false, nil)
		call, n := sequence(expired)
		if _, err := withReauth(ctx, reload, call); err != expired || *n != 1 {
			t.Errorf("err=%v calls=%d", err, *n)
		}
	})

	t.Run("login fails: original error", func(t *testing.T) {
		stubReauth(t, false, errors.New("login failed"))
		call, _ := sequence(expired)
		if _, err := withReauth(ctx, reload, call); err != expired {
			t.Errorf("err=%v", err)
		}
	})

	t.Run("reload fails: reload error, no retry", func(t *testing.T) {
		stubReauth(t, true, nil)
		call, n := sequence(expired)
		reloadErr := errors.New("reload failed")
		if _, err := withReauth(ctx, func(context.Context) error { return reloadErr }, call); err != reloadErr || *n != 1 {
			t.Errorf("err=%v calls=%d", err, *n)
		}
	})

	t.Run("retry fails: retry error", func(t *testing.T) {
		stubReauth(t, true, nil)
		call, _ := sequence(expired, errors.New("still failing"))
		if _, err := withReauth(ctx, reload, call); err == nil || err.Error() != "still failing" {
			t.Errorf("err=%v", err)
		}
	})
}
