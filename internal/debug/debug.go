package debug

import (
	"fmt"
	"os"
)

var isVerbose bool

// SetVerbose sets the global verbose flag
func SetVerbose(v bool) {
	isVerbose = v
}

// IsVerbose returns the current verbose setting
func IsVerbose() bool {
	return isVerbose
}

// Printf prints debug output only if verbose mode is enabled
func Printf(format string, args ...interface{}) {
	if isVerbose {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Println prints debug output only if verbose mode is enabled
func Println(args ...interface{}) {
	if isVerbose {
		fmt.Fprintln(os.Stderr, args...)
	}
}
