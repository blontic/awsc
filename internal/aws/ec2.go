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

type EC2Manager struct {
	ec2Client *ec2.Client
	ssmClient *ssm.Client
	region    string
}

type EC2Instance struct {
	InstanceId   string
	Name         string
	InstanceType string
	State        string
}

func NewEC2Manager(ctx context.Context) (*EC2Manager, error) {
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
		instanceOptions[i] = fmt.Sprintf("%s (%s) - %s", instance.Name, instance.InstanceId, instance.State)
	}

	// Interactive instance selection
	selectedIndex, err := ui.RunSelector("Select EC2 Instance:", instanceOptions)
	if err != nil {
		return fmt.Errorf("error selecting instance: %v", err)
	}
	if selectedIndex == -1 {
		return fmt.Errorf("no instance selected")
	}

	selectedInstance := instances[selectedIndex]
	fmt.Printf("Selected: %s\n", selectedInstance.Name)

	// Start SSM session
	return e.StartSSMSession(ctx, selectedInstance.InstanceId)
}

func (e *EC2Manager) ListSSMInstances(ctx context.Context) ([]EC2Instance, error) {
	result, err := e.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running"},
			},
		},
	})
	if err != nil {
		if IsAuthError(err) {
			if handleErr := HandleExpiredCredentials(ctx); handleErr != nil {
				return nil, handleErr
			}
			// Reload client with fresh credentials
			cfg, cfgErr := swaconfig.LoadSWAConfigWithProfile(ctx)
			if cfgErr != nil {
				return nil, cfgErr
			}
			e.ec2Client = ec2.NewFromConfig(cfg)
			// Retry after re-authentication
			result, err = e.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
				Filters: []types.Filter{
					{
						Name:   aws.String("instance-state-name"),
						Values: []string{"running"},
					},
				},
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var instances []EC2Instance
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			// Check if instance has SSM agent (has managed instance)
			if e.hasSSMAgent(ctx, *instance.InstanceId) {
				instances = append(instances, EC2Instance{
					InstanceId:   *instance.InstanceId,
					Name:         e.getInstanceName(instance.Tags),
					InstanceType: string(instance.InstanceType),
					State:        string(instance.State.Name),
				})
			}
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
		return false
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

func (e *EC2Manager) fallbackToCommand(instanceId string) error {
	fmt.Printf("\nRun this command manually:\n\n")
	fmt.Printf("aws ssm start-session --target %s --region %s\n\n", instanceId, e.region)
	return nil
}
