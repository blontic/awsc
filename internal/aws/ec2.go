package aws

import (
	"context"
	"fmt"
	"sort"

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

func (e *EC2Manager) RunConnect(ctx context.Context) error {
	// Display AWS context
	DisplayAWSContext(ctx)

	// List EC2 instances with SSM capability
	instances, err := e.ListSSMInstances(ctx)
	if err != nil {
		return fmt.Errorf("error listing EC2 instances: %v", err)
	}

	if len(instances) == 0 {
		return fmt.Errorf("no EC2 instances with SSM agent found")
	}

	// Create instance options for selection
	instanceOptions := make([]string, len(instances))
	for i, instance := range instances {
		if instance.IsSelectable {
			instanceOptions[i] = fmt.Sprintf("%s (%s) - %s - %s", instance.Name, instance.InstanceId, instance.Platform, instance.State)
		} else {
			instanceOptions[i] = fmt.Sprintf("%s (%s) - %s - %s (unavailable)", instance.Name, instance.InstanceId, instance.Platform, instance.State)
		}
	}

	// Create selectability array
	selectableOptions := make([]bool, len(instances))
	for i, instance := range instances {
		selectableOptions[i] = instance.IsSelectable
	}

	// Interactive instance selection
	selectedIndex, err := ui.RunSelectorWithSelectability("Select EC2 Instance:", instanceOptions, selectableOptions)
	if err != nil {
		return fmt.Errorf("error selecting instance: %v", err)
	}
	if selectedIndex == -1 {
		return fmt.Errorf("no instance selected")
	}

	selectedInstance := instances[selectedIndex]
	fmt.Printf("Selected: %s\n", selectedInstance.Name)

	// Check if Windows instance and offer RDP option
	if selectedInstance.Platform == "Windows" {
		return e.handleWindowsConnection(ctx, selectedInstance)
	}

	// Start SSM session for non-Windows instances
	return e.StartSSMSession(ctx, selectedInstance.InstanceId)
}

func (e *EC2Manager) ListSSMInstances(ctx context.Context) ([]EC2Instance, error) {
	var allReservations []types.Reservation
	var nextToken *string

	for {
		result, err := e.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			NextToken: nextToken,
		})
		if err != nil {
			if IsAuthError(err) {
				if handleErr := HandleExpiredCredentials(ctx); handleErr != nil {
					return nil, handleErr
				}
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
			hasSSM := isRunning && e.hasSSMAgent(ctx, *instance.InstanceId)

			instances = append(instances, EC2Instance{
				InstanceId:   *instance.InstanceId,
				Name:         e.getInstanceName(instance.Tags),
				InstanceType: string(instance.InstanceType),
				State:        string(instance.State.Name),
				Platform:     e.getPlatform(instance),
				IsSelectable: hasSSM,
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
	err = pf.StartInteractiveSession(ctx, instanceId)
	if err != nil {
		fmt.Printf("SSM session failed: %v\n", err)
		return e.fallbackToCommand(instanceId)
	}
	return nil
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
			if handleErr := HandleExpiredCredentials(ctx); handleErr != nil {
				return false
			}
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
	if instance.Platform != "" {
		return string(instance.Platform)
	}
	// Default to Linux if no platform specified
	return "Linux"
}

func (e *EC2Manager) handleWindowsConnection(ctx context.Context, instance EC2Instance) error {
	fmt.Printf("\nWindows instance detected. Choose connection method:\n")
	fmt.Printf("1. SSM Session (PowerShell)\n")
	fmt.Printf("2. RDP Port Forwarding\n")
	fmt.Printf("\nEnter choice (1 or 2): ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "2":
		return e.startRDPPortForwarding(ctx, instance.InstanceId)
	default:
		return e.StartSSMSession(ctx, instance.InstanceId)
	}
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
	err = pf.StartPortForwardingToRemoteHost(ctx, instanceId, "localhost", remotePort, localPort)
	if err != nil {
		fmt.Printf("RDP port forwarding failed: %v\n", err)
		return e.fallbackToRDPCommand(instanceId)
	}
	return nil
}

func (e *EC2Manager) fallbackToRDPCommand(instanceId string) error {
	fmt.Printf("\nRun this command manually for RDP port forwarding:\n\n")
	fmt.Printf("aws ssm start-session --target %s --document-name AWS-StartPortForwardingSession --parameters '{\"portNumber\":[\"3389\"],\"localPortNumber\":[\"3389\"]}' --region %s\n\n", instanceId, e.region)
	fmt.Printf("Then connect with RDP to: localhost:3389\n")
	return nil
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

func (e *EC2Manager) fallbackToCommand(instanceId string) error {
	fmt.Printf("\nRun this command manually:\n\n")
	fmt.Printf("aws ssm start-session --target %s --region %s\n\n", instanceId, e.region)
	return nil
}
