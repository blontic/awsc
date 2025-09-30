package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewExternalPluginForwarder(t *testing.T) {
	cfg := aws.Config{
		Region: "us-east-1",
	}

	forwarder := NewExternalPluginForwarder(cfg)

	if forwarder == nil {
		t.Fatal("Expected forwarder to be created")
	}
	if forwarder.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", forwarder.region)
	}
}

func TestExternalPluginForwarder_handleMissingPlugin(t *testing.T) {
	forwarder := &ExternalPluginForwarder{
		region: "us-east-1",
	}

	err := forwarder.handleMissingPlugin()
	if err == nil {
		t.Error("Expected error when plugin is missing")
	}
	if err.Error() != "session-manager-plugin not installed" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestExternalPluginForwarder_StartPortForwardingToRemoteHost(t *testing.T) {
	// Skip this test as it requires session-manager-plugin to be installed
	// and valid AWS credentials
	t.Skip("Skipping StartPortForwardingToRemoteHost test - requires session-manager-plugin and AWS credentials")
}

func TestExternalPluginForwarder_StartInteractiveSession(t *testing.T) {
	// Skip this test as it requires session-manager-plugin to be installed
	// and valid AWS credentials
	t.Skip("Skipping StartInteractiveSession test - requires session-manager-plugin and AWS credentials")
}
