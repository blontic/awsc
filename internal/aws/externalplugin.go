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
	doc, params := remoteHostForwardingRequest(remoteHost, remotePort, localPort)
	return pf.startPortForwardingSession(ctx, bastionId, doc, params, localPort)
}

// StartPortForwarding forwards a local port to a port on the target instance itself.
// Use this instead of StartPortForwardingToRemoteHost with "localhost", which newer
// SSM agents reject.
func (pf *ExternalPluginForwarder) StartPortForwarding(ctx context.Context, instanceId string, remotePort, localPort int) error {
	doc, params := instanceForwardingRequest(remotePort, localPort)
	return pf.startPortForwardingSession(ctx, instanceId, doc, params, localPort)
}

func remoteHostForwardingRequest(remoteHost string, remotePort, localPort int) (string, map[string][]string) {
	return "AWS-StartPortForwardingSessionToRemoteHost", map[string][]string{
		"host":            {remoteHost},
		"portNumber":      {strconv.Itoa(remotePort)},
		"localPortNumber": {strconv.Itoa(localPort)},
	}
}

func instanceForwardingRequest(remotePort, localPort int) (string, map[string][]string) {
	return "AWS-StartPortForwardingSession", map[string][]string{
		"portNumber":      {strconv.Itoa(remotePort)},
		"localPortNumber": {strconv.Itoa(localPort)},
	}
}

func (pf *ExternalPluginForwarder) startPortForwardingSession(ctx context.Context, target, documentName string, params map[string][]string, localPort int) error {
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
		Target:       aws.String(target),
		DocumentName: aws.String(documentName),
		Parameters:   params,
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
		Target:       target,
		DocumentName: documentName,
		Parameters:   params,
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
	fmt.Fprintf(os.Stderr, "\n❌ Session Manager Plugin not found\n\n")
	fmt.Fprintf(os.Stderr, "The AWS Session Manager Plugin is required for SSM sessions.\n")
	fmt.Fprintf(os.Stderr, "Please install it using one of these methods:\n\n")

	fmt.Fprintf(os.Stderr, "📦 macOS: brew install --cask session-manager-plugin\n")
	fmt.Fprintf(os.Stderr, "📦 Linux: curl -o plugin.deb https://s3.amazonaws.com/session-manager-downloads/plugin/latest/ubuntu_64bit/session-manager-plugin.deb && sudo dpkg -i plugin.deb\n")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintf(os.Stderr, "After installation, run the command again.\n")
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
