package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	swaconfig "github.com/blontic/swa/internal/config"
	"github.com/blontic/swa/internal/ui"
)

// SSMClient interface for mocking
type SSMClient interface {
	DescribeInstanceInformation(ctx context.Context, params *ssm.DescribeInstanceInformationInput, optFns ...func(*ssm.Options)) (*ssm.DescribeInstanceInformationOutput, error)
}

type EC2Manager struct {
	ec2Client EC2Client
	ssmClient SSMClient
	region    string
}

type EC2Instance struct {
	InstanceId   string
	Name         string
	InstanceType string
	State        string
	Platform     string
	IsSelectable bool
}

type EC2ManagerOptions struct {
	EC2Client EC2Client
	SSMClient SSMClient
	Region    string
}

func NewEC2Manager(ctx context.Context, opts ...EC2ManagerOptions) (*EC2Manager, error) {
	if len(opts) > 0 && opts[0].EC2Client != nil {
		// Use provided clients (for testing)
		return &EC2Manager{
			ec2Client: opts[0].EC2Client,
			ssmClient: opts[0].SSMClient,
			region:    opts[0].Region,
		}, nil
	}

	// Production path
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &EC2Manager{
		ec2Client: ec2.NewFromConfig(cfg),
		ssmClient: ssm.NewFromConfig(cfg),
		region:    cfg.Region,
	}, nil
}

func (e *EC2Manager) RunConnect(ctx context.Context, instanceId string) error {
	// Get all instances first
	allInstances, err := e.ListAllInstances(ctx)
	if err != nil {
		return fmt.Errorf("error listing EC2 instances: %v", err)
	}

	if len(allInstances) == 0 {
		return fmt.Errorf("no EC2 instances found")
	}

	// If instance ID provided, try to connect directly
	if instanceId != "" {
		var targetInstance *EC2Instance
		for _, instance := range allInstances {
			if instance.InstanceId == instanceId {
				targetInstance = &instance
				break
			}
		}

		if targetInstance != nil && targetInstance.IsSelectable {
			fmt.Printf("Connecting to instance: %s (%s)\n", targetInstance.Name, targetInstance.InstanceId)

			// Start SSM session for all instances
			return e.StartSSMSession(ctx, targetInstance.InstanceId)
		}

		// Instance not found or not selectable - show error and fall through to list
		if targetInstance == nil {
			fmt.Printf("Instance '%s' not found. Available instances:\n\n", instanceId)
		} else {
			fmt.Printf("Instance '%s' is not available (state: %s). Available instances:\n\n", instanceId, targetInstance.State)
		}
	}

	// List all EC2 instances (show stopped ones as non-selectable)
	instances, err := e.ListAllInstances(ctx)
	if err != nil {
		return fmt.Errorf("error listing EC2 instances: %v", err)
	}

	if len(instances) == 0 {
		return fmt.Errorf("no EC2 instances found")
	}

	// Check if any instances are selectable
	hasSelectable := false
	for _, instance := range instances {
		if instance.IsSelectable {
			hasSelectable = true
			break
		}
	}

	if !hasSelectable {
		return fmt.Errorf("no running EC2 instances with SSM agent found - you may need to start an instance")
	}

	// Select instance
	selectedInstance, err := e.selectInstance("Select EC2 Instance:", instances)
	if err != nil {
		return err
	}

	// Start SSM session for all instances
	return e.StartSSMSession(ctx, selectedInstance.InstanceId)
}

func (e *EC2Manager) RunRDP(ctx context.Context, instanceId string) error {
	// Get all instances first
	allInstances, err := e.ListAllInstances(ctx)
	if err != nil {
		return fmt.Errorf("error listing EC2 instances: %v", err)
	}

	// Filter for Windows instances (include stopped ones but mark as non-selectable)
	var windowsInstances []EC2Instance
	for _, instance := range allInstances {
		if strings.ToLower(instance.Platform) == "windows" {
			// Only running instances with SSM are selectable for RDP
			instance.IsSelectable = instance.State == "running" && instance.IsSelectable
			windowsInstances = append(windowsInstances, instance)
		}
	}

	if len(windowsInstances) == 0 {
		return fmt.Errorf("no Windows EC2 instances found")
	}

	// If instance ID provided, try to connect directly
	if instanceId != "" {
		var targetInstance *EC2Instance
		for _, instance := range windowsInstances {
			if instance.InstanceId == instanceId {
				targetInstance = &instance
				break
			}
		}

		if targetInstance != nil && targetInstance.IsSelectable {
			fmt.Printf("Starting RDP to instance: %s (%s)\n", targetInstance.Name, targetInstance.InstanceId)
			return e.startRDPPortForwarding(ctx, targetInstance.InstanceId)
		}

		// Instance not found or not selectable - show error and fall through to list
		if targetInstance == nil {
			fmt.Printf("Windows instance '%s' not found. Available Windows instances:\n\n", instanceId)
		} else {
			fmt.Printf("Windows instance '%s' is not available for RDP (state: %s). Available Windows instances:\n\n", instanceId, targetInstance.State)
		}
	}

	// Select Windows instance
	selectedInstance, err := e.selectInstance("Select Windows EC2 Instance:", windowsInstances)
	if err != nil {
		return err
	}

	// Start RDP port forwarding
	return e.startRDPPortForwarding(ctx, selectedInstance.InstanceId)
}

func (e *EC2Manager) ListAllInstances(ctx context.Context) ([]EC2Instance, error) {
	var allReservations []types.Reservation
	var nextToken *string

	for {
		result, err := e.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					// Reload all clients with fresh credentials
					if reloadErr := e.reloadClients(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					// Retry after re-authentication
					result, err = e.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
						NextToken: nextToken,
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		allReservations = append(allReservations, result.Reservations...)

		// Check if there are more pages
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	var instances []EC2Instance
	for _, reservation := range allReservations {
		for _, instance := range reservation.Instances {
			isRunning := string(instance.State.Name) == "running"
			// Only check SSM for running instances to avoid unnecessary API calls
			hasSSM := isRunning && e.hasSSMAgent(ctx, *instance.InstanceId)

			instances = append(instances, EC2Instance{
				InstanceId:   *instance.InstanceId,
				Name:         e.getInstanceName(instance.Tags),
				InstanceType: string(instance.InstanceType),
				State:        string(instance.State.Name),
				Platform:     e.getPlatform(instance),
				IsSelectable: hasSSM, // Only running instances with SSM are selectable
			})
		}
	}

	// Sort instances by name
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Name < instances[j].Name
	})

	return instances, nil
}

func (e *EC2Manager) StartSSMSession(ctx context.Context, instanceId string) error {
	// Start SSM session using external plugin
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	pf := NewExternalPluginForwarder(cfg)

	// Start interactive session
	return pf.StartInteractiveSession(ctx, instanceId)
}

func (e *EC2Manager) hasSSMAgent(ctx context.Context, instanceId string) bool {
	// Check if instance is managed by SSM
	result, err := e.ssmClient.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
		Filters: []ssmtypes.InstanceInformationStringFilter{
			{
				Key:    aws.String("InstanceIds"),
				Values: []string{instanceId},
			},
		},
	})
	if err != nil {
		if IsAuthError(err) {
			if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
				// Reload all clients with fresh credentials
				if reloadErr := e.reloadClients(ctx); reloadErr != nil {
					return false
				}
				// Retry after re-authentication
				result, err = e.ssmClient.DescribeInstanceInformation(ctx, &ssm.DescribeInstanceInformationInput{
					Filters: []ssmtypes.InstanceInformationStringFilter{
						{
							Key:    aws.String("InstanceIds"),
							Values: []string{instanceId},
						},
					},
				})
				if err != nil {
					return false
				}
			} else {
				return false
			}
		} else {
			return false
		}
	}

	return len(result.InstanceInformationList) > 0
}

func (e *EC2Manager) getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == "Name" && tag.Value != nil {
			return *tag.Value
		}
	}
	return "Unnamed"
}

func (e *EC2Manager) getPlatform(instance types.Instance) string {
	// Check platform field first
	if instance.Platform != "" {
		return string(instance.Platform)
	}

	// Check AMI name for Windows indicators
	if instance.ImageId != nil {
		// This is a simplified check - in practice you might want to describe the AMI
		// But we can also check instance metadata or tags
	}

	// Check for Windows-specific instance types or other indicators
	// For now, check if it's a known Windows AMI pattern or has Windows in tags
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			if *tag.Key == "Platform" && *tag.Value == "Windows" {
				return "Windows"
			}
			if *tag.Key == "OS" && strings.Contains(strings.ToLower(*tag.Value), "windows") {
				return "Windows"
			}
		}
	}

	// Default to Linux if no platform specified
	return "Linux"
}

func (e *EC2Manager) startRDPPortForwarding(ctx context.Context, instanceId string) error {
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	pf := NewExternalPluginForwarder(cfg)
	localPort := 3389
	remotePort := 3389

	fmt.Printf("Starting RDP port forwarding on localhost:%d...\n", localPort)

	// Start port forwarding for RDP
	return pf.StartPortForwardingToRemoteHost(ctx, instanceId, "localhost", remotePort, localPort)
}



func (e *EC2Manager) reloadClients(ctx context.Context) error {
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return err
	}

	e.ec2Client = ec2.NewFromConfig(cfg)
	e.ssmClient = ssm.NewFromConfig(cfg)
	e.region = cfg.Region

	return nil
}

func (e *EC2Manager) selectInstance(title string, instances []EC2Instance) (*EC2Instance, error) {
	// Create instance options for selection
	instanceOptions := make([]string, len(instances))
	for i, instance := range instances {
		instanceOptions[i] = fmt.Sprintf("%s (%s) - %s - %s", instance.Name, instance.InstanceId, instance.Platform, instance.State)
	}

	// Create selectability array
	selectableOptions := make([]bool, len(instances))
	for i, instance := range instances {
		selectableOptions[i] = instance.IsSelectable
	}

	// Interactive instance selection
	selectedIndex, err := ui.RunSelectorWithSelectability(title, instanceOptions, selectableOptions)
	if err != nil {
		return nil, fmt.Errorf("error selecting instance: %v", err)
	}
	if selectedIndex == -1 {
		return nil, fmt.Errorf("no instance selected")
	}

	selectedInstance := instances[selectedIndex]
	fmt.Printf("Selected: %s\n", selectedInstance.Name)
	return &selectedInstance, nil
}


