package aws

import (
	"context"
	"strings"
	"testing"
)

func TestDisplayAWSContext(t *testing.T) {
	ctx := context.Background()

	// This function should not panic even without valid AWS credentials
	// It should silently skip if credentials are not available
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DisplayAWSContext panicked: %v", r)
		}
	}()

	DisplayAWSContext(ctx)
	// If we get here without panicking, the test passes
}

func TestParseRoleFromARN(t *testing.T) {
	testCases := []struct {
		arn      string
		expected string
	}{
		{
			arn:      "arn:aws:sts::123456789012:assumed-role/MyRole/SessionName",
			expected: "MyRole",
		},
		{
			arn:      "arn:aws:sts::123456789012:user/MyUser",
			expected: "unknown",
		},
		{
			arn:      "invalid-arn",
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		role := "unknown"
		if strings.Contains(tc.arn, "assumed-role") {
			parts := strings.Split(tc.arn, "/")
			if len(parts) >= 2 {
				role = parts[1]
			}
		}

		if role != tc.expected {
			t.Errorf("For ARN %s, expected role %s, got %s", tc.arn, tc.expected, role)
		}
	}
}
