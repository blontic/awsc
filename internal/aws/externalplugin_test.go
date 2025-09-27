package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewExternalPluginForwarder(t *testing.T) {
	// Create a basic AWS config for testing
	cfg := aws.Config{
		Region: "us-east-1",
	}

	forwarder := NewExternalPluginForwarder(cfg)
	if forwarder == nil {
		t.Error("Expected NewExternalPluginForwarder to return non-nil forwarder")
	}

	if forwarder.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", forwarder.region)
	}
}

func TestExternalPluginForwarder_StartInteractiveSession(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	forwarder := NewExternalPluginForwarder(cfg)

	ctx := context.Background()

	// This will fail because we don't have session-manager-plugin in test env
	err := forwarder.StartInteractiveSession(ctx, "i-1234567890abcdef0")
	if err == nil {
		t.Error("Expected StartInteractiveSession to fail in test environment")
	} else {
		t.Logf("StartInteractiveSession failed as expected: %v", err)
	}
}

func TestExternalPluginForwarder_StartPortForwardingToRemoteHost(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	forwarder := NewExternalPluginForwarder(cfg)

	ctx := context.Background()

	// This will fail because we don't have session-manager-plugin in test env
	err := forwarder.StartPortForwardingToRemoteHost(ctx, "i-1234567890abcdef0", "db.example.com", 5432, 5432)
	if err == nil {
		t.Error("Expected StartPortForwardingToRemoteHost to fail in test environment")
	} else {
		t.Logf("StartPortForwardingToRemoteHost failed as expected: %v", err)
	}
}
