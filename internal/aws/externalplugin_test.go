package aws

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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

func TestMarshalSessionResponse(t *testing.T) {
	// Valid response marshals to the exact JSON the plugin expects.
	result := &ssm.StartSessionOutput{
		SessionId:  aws.String("sess-123"),
		StreamUrl:  aws.String("wss://example/stream"),
		TokenValue: aws.String("tok-abc"),
	}

	data, err := marshalSessionResponse(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got["SessionId"] != "sess-123" || got["StreamUrl"] != "wss://example/stream" || got["TokenValue"] != "tok-abc" {
		t.Errorf("unexpected marshalled payload: %s", string(data))
	}
}

func TestMarshalSessionResponse_NilFields(t *testing.T) {
	cases := map[string]*ssm.StartSessionOutput{
		"nil result":     nil,
		"nil SessionId":  {StreamUrl: aws.String("u"), TokenValue: aws.String("t")},
		"nil StreamUrl":  {SessionId: aws.String("s"), TokenValue: aws.String("t")},
		"nil TokenValue": {SessionId: aws.String("s"), StreamUrl: aws.String("u")},
	}

	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := marshalSessionResponse(in); err == nil {
				t.Errorf("expected error for %s, got nil", name)
			}
		})
	}
}

func TestPluginParameters_PortForwardingJSON(t *testing.T) {
	// A remoteHost containing quotes must not break the JSON (regression for the
	// previous fmt.Sprintf-based construction).
	data, err := json.Marshal(pluginParameters{
		Target:       "i-123",
		DocumentName: "AWS-StartPortForwardingSessionToRemoteHost",
		Parameters: map[string][]string{
			"host":            {`evil"host`},
			"portNumber":      {"5432"},
			"localPortNumber": {"5432"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rt pluginParameters
	if err := json.Unmarshal(data, &rt); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if rt.Parameters["host"][0] != `evil"host` {
		t.Errorf("host not round-tripped safely: %q", rt.Parameters["host"][0])
	}
}
