package aws

import (
	"context"
	"encoding/json"
	"fmt"
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
	responseJson, _ := json.Marshal(map[string]interface{}{
		"SessionId":  *result.SessionId,
		"StreamUrl":  *result.StreamUrl,
		"TokenValue": *result.TokenValue,
	})

	// Prepare parameters for plugin
	parametersJson := fmt.Sprintf(`{"Target":"%s","DocumentName":"AWS-StartPortForwardingSessionToRemoteHost","Parameters":{"host":["%s"],"portNumber":["%s"],"localPortNumber":["%s"]}}`,
		bastionId, remoteHost, strconv.Itoa(remotePort), strconv.Itoa(localPort))

	// Call session-manager-plugin with exact same arguments as AWS CLI
	cmd := exec.CommandContext(ctx, "session-manager-plugin",
		string(responseJson), // Session response
		pf.region,            // Region
		"StartSession",       // Operation
		"",                   // Profile (empty)
		parametersJson,       // Parameters
		"")                   // Endpoint (empty)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

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
	responseJson, _ := json.Marshal(map[string]interface{}{
		"SessionId":  *result.SessionId,
		"StreamUrl":  *result.StreamUrl,
		"TokenValue": *result.TokenValue,
	})

	// Prepare parameters for plugin
	parametersJson := fmt.Sprintf(`{"Target":"%s"}`, instanceId)

	// Call session-manager-plugin with exact same arguments as AWS CLI
	cmd := exec.CommandContext(ctx, "session-manager-plugin",
		string(responseJson), // Session response
		pf.region,            // Region
		"StartSession",       // Operation
		"",                   // Profile (empty)
		parametersJson,       // Parameters
		"")                   // Endpoint (empty)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
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
