package aws

import (
	"context"
	"fmt"
	"testing"
)

func TestNewSSOManager(t *testing.T) {
	ctx := context.Background()

	// This will likely fail without valid AWS credentials but shouldn't panic
	_, err := NewSSOManager(ctx)
	if err != nil {
		t.Logf("NewSSOManager failed as expected in test environment: %v", err)
	} else {
		t.Log("NewSSOManager succeeded unexpectedly")
	}
}

func TestSSOManager_RunLogin(t *testing.T) {
	// Skip this test as it requires AWS clients to be initialized
	// In a real test environment, we'd use mocks
	t.Skip("Skipping RunLogin test - requires AWS client initialization")
}

func TestSSOManager_RunLogin_NoForce(t *testing.T) {
	// Skip this test as it requires AWS clients to be initialized
	// In a real test environment, we'd use mocks
	t.Skip("Skipping RunLogin test - requires AWS client initialization")
}

func TestIsAuthError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-auth error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
		{
			name:     "permission error should not be auth error",
			err:      fmt.Errorf("is not authorized to perform"),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsAuthError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
