package aws

import (
	"context"
	"testing"
)

func TestDisplayAWSContext(t *testing.T) {
	ctx := context.Background()

	// Test that DisplayAWSContext doesn't panic even without valid credentials
	// This function silently skips errors, so we just verify it doesn't crash
	DisplayAWSContext(ctx)

	// Test passes if no panic occurs
}
