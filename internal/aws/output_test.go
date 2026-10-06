package aws

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// captureOutput runs fn and returns what it wrote to stdout and stderr.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	read := func(f **os.File) func() string {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		orig := *f
		*f = w
		done := make(chan string)
		go func() { b, _ := io.ReadAll(r); done <- string(b) }()
		return func() string { w.Close(); *f = orig; return <-done }
	}
	stopOut, stopErr := read(&os.Stdout), read(&os.Stderr)
	fn()
	return stopOut(), stopErr()
}

// The secret value is the only thing on stdout, so it can be piped or captured.
func TestDisplaySecret_OnlySecretOnStdout(t *testing.T) {
	m := &SecretsManager{}
	for _, tc := range []struct{ value, want string }{
		{"plain-value", "plain-value\n"},
		{`{"user":"a"}`, "{\n  \"user\": \"a\"\n}\n"},
	} {
		stdout, _ := captureOutput(t, func() { m.DisplaySecret(context.Background(), "s", tc.value) })
		if stdout != tc.want {
			t.Errorf("stdout = %q, want %q", stdout, tc.want)
		}
	}
}

// Status and progress messages must go to stderr. Only DisplaySecret may
// write to stdout in this package.
func TestNoStatusOutputOnStdout(t *testing.T) {
	stdoutCall := regexp.MustCompile(`fmt\.Print(f|ln)?\(`)
	files, _ := filepath.Glob("*.go")
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(data)
		if f == "secrets.go" {
			start := strings.Index(src, "func (s *SecretsManager) DisplaySecret(")
			end := start + strings.Index(src[start:], "\n}\n")
			src = src[:start] + src[end:]
		}
		for i, line := range strings.Split(src, "\n") {
			if stdoutCall.MatchString(line) {
				t.Errorf("%s:%d writes to stdout; use fmt.Fprint*(os.Stderr, ...): %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
