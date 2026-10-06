package cmd

import (
	"context"
	"errors"
	"os"
	"testing"
)

type fakeManager struct{ build int }

// fakeCreate returns managers numbered by creation and fails with the given
// errors on the first calls.
func fakeCreate(errs ...error) (func(context.Context) (*fakeManager, error), *int) {
	calls := 0
	return func(context.Context) (*fakeManager, error) {
		calls++
		if calls <= len(errs) && errs[calls-1] != nil {
			return nil, errs[calls-1]
		}
		return &fakeManager{build: calls}, nil
	}, &calls
}

// stubLogin replaces the login steps, recording whether each ran.
func stubLogin(t *testing.T, reauthOK bool, reauthErr, switchErr error) (reauthed, switched *bool) {
	t.Helper()
	reauthed, switched = new(bool), new(bool)
	origReauth, origSwitch := reauthFn, switchAccountFn
	t.Cleanup(func() { reauthFn, switchAccountFn = origReauth, origSwitch })
	reauthFn = func(context.Context) (bool, error) { *reauthed = true; return reauthOK, reauthErr }
	switchAccountFn = func(context.Context) error { *switched = true; return switchErr }
	return reauthed, switched
}

var errNoSession = errors.New("no active session")

func TestNewManager(t *testing.T) {
	ctx := context.Background()

	t.Run("logged in, no switch: created once", func(t *testing.T) {
		reauthed, switched := stubLogin(t, true, nil, nil)
		create, calls := fakeCreate()
		m, err := newManager(ctx, create, false)
		if err != nil || m.build != 1 || *calls != 1 || *reauthed || *switched {
			t.Errorf("m=%+v err=%v calls=%d reauthed=%v switched=%v", m, err, *calls, *reauthed, *switched)
		}
	})

	t.Run("-s switches account then recreates", func(t *testing.T) {
		reauthed, switched := stubLogin(t, true, nil, nil)
		create, calls := fakeCreate()
		m, err := newManager(ctx, create, true)
		if err != nil || m.build != 2 || !*switched || *reauthed || *calls != 2 {
			t.Errorf("m=%+v err=%v calls=%d reauthed=%v switched=%v", m, err, *calls, *reauthed, *switched)
		}
	})

	t.Run("not logged in: logs in then recreates", func(t *testing.T) {
		reauthed, switched := stubLogin(t, true, nil, nil)
		create, calls := fakeCreate(errNoSession)
		m, err := newManager(ctx, create, false)
		if err != nil || m.build != 2 || !*reauthed || *switched || *calls != 2 {
			t.Errorf("m=%+v err=%v calls=%d reauthed=%v switched=%v", m, err, *calls, *reauthed, *switched)
		}
	})

	t.Run("not logged in with -s: login covers the switch", func(t *testing.T) {
		reauthed, switched := stubLogin(t, true, nil, nil)
		create, _ := fakeCreate(errNoSession)
		if _, err := newManager(ctx, create, true); err != nil || !*reauthed || *switched {
			t.Errorf("err=%v reauthed=%v switched=%v (account should be chosen once)", err, *reauthed, *switched)
		}
	})

	t.Run("login declined", func(t *testing.T) {
		stubLogin(t, false, nil, nil)
		create, calls := fakeCreate(errNoSession)
		if _, err := newManager(ctx, create, false); err == nil || err.Error() != "authentication cancelled" || *calls != 1 {
			t.Errorf("err=%v calls=%d", err, *calls)
		}
	})

	t.Run("login fails", func(t *testing.T) {
		stubLogin(t, false, errors.New("boom"), nil)
		create, _ := fakeCreate(errNoSession)
		if _, err := newManager(ctx, create, false); err == nil || err.Error() != "boom" {
			t.Errorf("err=%v", err)
		}
	})

	t.Run("switch fails", func(t *testing.T) {
		stubLogin(t, true, nil, errors.New("switch failed"))
		create, calls := fakeCreate()
		if _, err := newManager(ctx, create, true); err == nil || err.Error() != "switch failed" || *calls != 1 {
			t.Errorf("err=%v calls=%d", err, *calls)
		}
	})

	t.Run("non-auth error is returned without login", func(t *testing.T) {
		reauthed, _ := stubLogin(t, true, nil, nil)
		create, _ := fakeCreate(errors.New("throttled"))
		if _, err := newManager(ctx, create, false); err == nil || err.Error() != "throttled" || *reauthed {
			t.Errorf("err=%v reauthed=%v", err, *reauthed)
		}
	})

	t.Run("recreate after login fails", func(t *testing.T) {
		stubLogin(t, true, nil, nil)
		create, _ := fakeCreate(errNoSession, errors.New("still broken"))
		if _, err := newManager(ctx, create, false); err == nil || err.Error() != "still broken" {
			t.Errorf("err=%v", err)
		}
	})
}

func TestHandleAccountSwitch_UnsetsAWSC_PROFILE(t *testing.T) {
	t.Setenv("AWSC_PROFILE", "awsc-test-account")
	t.Setenv("HOME", t.TempDir())
	_ = handleAccountSwitch(context.Background()) // fails without an org; only the env var matters here
	if os.Getenv("AWSC_PROFILE") != "" {
		t.Error("AWSC_PROFILE should be unset when switching accounts")
	}
}
