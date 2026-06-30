package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// ExternalPluginForwarder uses the external session-manager-plugin binary
type ExternalPluginForwarder struct {
	ssmClient *ssm.Client
	region    string
}

func NewExternalPluginForwarder(cfg aws.Config) *ExternalPluginForwarder {
	return &ExternalPluginForwarder{
		ssmClient: ssm.NewFromConfig(cfg),
		region:    cfg.Region,
	}
}

func (pf *ExternalPluginForwarder) StartPortForwardingToRemoteHost(ctx context.Context, bastionId, remoteHost string, remotePort, localPort int) error {
	// Check if session-manager-plugin is available
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return pf.handleMissingPlugin()
	}

	// Check if local port is available
	if err := pf.checkPortAvailable(localPort); err != nil {
		return err
	}

	// Start SSM session
	sessionInput := &ssm.StartSessionInput{
		Target:       aws.String(bastionId),
		DocumentName: aws.String("AWS-StartPortForwardingSessionToRemoteHost"),
		Parameters: map[string][]string{
			"host":            {remoteHost},
			"portNumber":      {strconv.Itoa(remotePort)},
			"localPortNumber": {strconv.Itoa(localPort)},
		},
	}

	result, err := pf.ssmClient.StartSession(ctx, sessionInput)
	if err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	// Prepare session response for plugin
	responseJson, err := marshalSessionResponse(result)
	if err != nil {
		return err
	}

	// Prepare parameters for plugin
	parametersJson, err := json.Marshal(pluginParameters{
		Target:       bastionId,
		DocumentName: "AWS-StartPortForwardingSessionToRemoteHost",
		Parameters: map[string][]string{
			"host":            {remoteHost},
			"portNumber":      {strconv.Itoa(remotePort)},
			"localPortNumber": {strconv.Itoa(localPort)},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal session parameters: %w", err)
	}

	// Call session-manager-plugin with exact same arguments as AWS CLI
	cmd := exec.CommandContext(ctx, "session-manager-plugin",
		string(responseJson),   // Session response
		pf.region,              // Region
		"StartSession",         // Operation
		"",                     // Profile (empty)
		string(parametersJson), // Parameters
		"")                     // Endpoint (empty)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the plugin and wait for it to complete
	return cmd.Run()
}

func (pf *ExternalPluginForwarder) StartInteractiveSession(ctx context.Context, instanceId string) error {
	// Check if session-manager-plugin is available
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return pf.handleMissingPlugin()
	}

	// Start SSM session
	sessionInput := &ssm.StartSessionInput{
		Target: aws.String(instanceId),
	}

	result, err := pf.ssmClient.StartSession(ctx, sessionInput)
	if err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	// Prepare session response for plugin
	responseJson, err := marshalSessionResponse(result)
	if err != nil {
		return err
	}

	// Prepare parameters for plugin
	parametersJson, err := json.Marshal(pluginParameters{Target: instanceId})
	if err != nil {
		return fmt.Errorf("failed to marshal session parameters: %w", err)
	}

	// Call session-manager-plugin with exact same arguments as AWS CLI
	cmd := exec.CommandContext(ctx, "session-manager-plugin",
		string(responseJson),   // Session response
		pf.region,              // Region
		"StartSession",         // Operation
		"",                     // Profile (empty)
		string(parametersJson), // Parameters
		"")                     // Endpoint (empty)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the plugin and wait for it to complete
	return cmd.Run()
}

func (pf *ExternalPluginForwarder) checkPortAvailable(port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("port %d is already in use (try a different port with --local-port <port>): %w", port, err)
	}
	listener.Close()
	return nil
}

func (pf *ExternalPluginForwarder) handleMissingPlugin() error {
	fmt.Printf("\n❌ Session Manager Plugin not found\n\n")
	fmt.Printf("The AWS Session Manager Plugin is required for SSM sessions.\n")
	fmt.Printf("Please install it using one of these methods:\n\n")

	fmt.Printf("📦 macOS: brew install --cask session-manager-plugin\n")
	fmt.Printf("📦 Linux: curl -o plugin.deb https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb && sudo dpkg -i plugin.deb\n")
	fmt.Printf("📦 Windows: Download from https://s3.amazonaws.com/session-manager-downloads/plugin/latest/windows/SessionManagerPluginSetup.exe\n\n")

	fmt.Printf("After installation, run the command again.\n")
	return fmt.Errorf("session-manager-plugin not installed")
}

// sessionResponse is the JSON payload (StartSession result) passed as the first
// argument to session-manager-plugin.
type sessionResponse struct {
	SessionId  string `json:"SessionId"`
	StreamUrl  string `json:"StreamUrl"`
	TokenValue string `json:"TokenValue"`
}

// pluginParameters is the JSON payload describing the session request passed as
// the parameters argument to session-manager-plugin.
type pluginParameters struct {
	Target       string              `json:"Target"`
	DocumentName string              `json:"DocumentName,omitempty"`
	Parameters   map[string][]string `json:"Parameters,omitempty"`
}

// marshalSessionResponse converts an SSM StartSession result into the JSON the
// plugin expects, validating the required pointers are present first.
func marshalSessionResponse(result *ssm.StartSessionOutput) ([]byte, error) {
	if result == nil || result.SessionId == nil || result.StreamUrl == nil || result.TokenValue == nil {
		return nil, fmt.Errorf("incomplete StartSession response from AWS")
	}

	data, err := json.Marshal(sessionResponse{
		SessionId:  *result.SessionId,
		StreamUrl:  *result.StreamUrl,
		TokenValue: *result.TokenValue,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session response: %w", err)
	}
	return data, nil
}
