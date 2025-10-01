package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func TestNewLogsManager(t *testing.T) {
	ctx := context.Background()

	// Test with mock client
	mockClient := &cloudwatchlogs.Client{}
	manager, err := NewLogsManager(ctx, LogsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager.client != mockClient {
		t.Error("Expected mock client to be used")
	}

	if manager.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", manager.region)
	}
}

func TestParseSince(t *testing.T) {
	ctx := context.Background()
	mockClient := &cloudwatchlogs.Client{}
	manager, _ := NewLogsManager(ctx, LogsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})

	tests := []struct {
		since    string
		expected bool // true if should succeed
	}{
		{"5m", true},
		{"1h", true},
		{"2d", true},
		{"1w", true},
		{"30s", true},
		{"", true}, // should default to 10m
		{"invalid", false},
		{"5x", false}, // invalid unit
		{"m", false},  // no number
	}

	for _, test := range tests {
		_, err := manager.parseSince(test.since)
		if test.expected && err != nil {
			t.Errorf("Expected parseSince('%s') to succeed, got error: %v", test.since, err)
		}
		if !test.expected && err == nil {
			t.Errorf("Expected parseSince('%s') to fail, but it succeeded", test.since)
		}
	}
}
