package aws

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/blontic/swa/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewEC2Manager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEC2 := mocks.NewMockEC2Client(ctrl)
	mockSSM := mocks.NewMockSSMClient(ctrl)

	manager, err := NewEC2Manager(context.Background(), EC2ManagerOptions{
		EC2Client: mockEC2,
		SSMClient: mockSSM,
		Region:    "us-east-1",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", manager.region)
	}
}

func TestNewEC2ManagerWithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEC2 := mocks.NewMockEC2Client(ctrl)
	mockSSM := mocks.NewMockSSMClient(ctrl)

	manager, err := NewEC2Manager(context.Background(), EC2ManagerOptions{
		EC2Client: mockEC2,
		SSMClient: mockSSM,
		Region:    "us-east-1",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", manager.region)
	}
}

func TestEC2Manager_ListAllInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEC2 := mocks.NewMockEC2Client(ctrl)
	mockSSM := mocks.NewMockSSMClient(ctrl)

	manager, err := NewEC2Manager(context.Background(), EC2ManagerOptions{
		EC2Client: mockEC2,
		SSMClient: mockSSM,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name                string
		ec2MockResponse     *ec2.DescribeInstancesOutput
		ec2MockError        error
		ssmMockResponse     *ssm.DescribeInstanceInformationOutput
		ssmMockError        error
		expectedCount       int
		expectedError       bool
		expectedInstanceIds []string
	}{
		{
			name: "successful response with SSM instances",
			ec2MockResponse: &ec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{
					{
						Instances: []types.Instance{
							{
								InstanceId:   aws.String("i-123456789"),
								InstanceType: types.InstanceTypeT3Micro,
								State:        &types.InstanceState{Name: types.InstanceStateNameRunning},
								Tags: []types.Tag{
									{Key: aws.String("Name"), Value: aws.String("WebServer")},
								},
							},
							{
								InstanceId:   aws.String("i-987654321"),
								InstanceType: types.InstanceTypeT3Small,
								State:        &types.InstanceState{Name: types.InstanceStateNameRunning},
								Tags: []types.Tag{
									{Key: aws.String("Name"), Value: aws.String("DatabaseServer")},
								},
							},
						},
					},
				},
			},
			ssmMockResponse: &ssm.DescribeInstanceInformationOutput{
				InstanceInformationList: []ssmtypes.InstanceInformation{
					{InstanceId: aws.String("i-123456789")},
				},
			},
			expectedCount:       2,
			expectedError:       false,
			expectedInstanceIds: []string{"i-123456789", "i-987654321"},
		},
		{
			name: "no instances with SSM agent",
			ec2MockResponse: &ec2.DescribeInstancesOutput{
				Reservations: []types.Reservation{
					{
						Instances: []types.Instance{
							{
								InstanceId:   aws.String("i-noSSM"),
								InstanceType: types.InstanceTypeT3Micro,
								State:        &types.InstanceState{Name: types.InstanceStateNameRunning},
								Tags: []types.Tag{
									{Key: aws.String("Name"), Value: aws.String("NoSSMServer")},
								},
							},
						},
					},
				},
			},
			ssmMockResponse: &ssm.DescribeInstanceInformationOutput{
				InstanceInformationList: []ssmtypes.InstanceInformation{},
			},
			expectedCount: 1,
			expectedError: false,
		},
		{
			name:            "empty EC2 response",
			ec2MockResponse: &ec2.DescribeInstancesOutput{Reservations: []types.Reservation{}},
			expectedCount:   0,
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock EC2 DescribeInstances call
			mockEC2.EXPECT().
				DescribeInstances(gomock.Any(), gomock.Any()).
				Return(tt.ec2MockResponse, tt.ec2MockError).
				Times(1)

			// Mock SSM calls for each instance
			if tt.ec2MockResponse != nil {
				for _, reservation := range tt.ec2MockResponse.Reservations {
					for _, instance := range reservation.Instances {
						// For the first test case, only first instance has SSM
						if tt.name == "successful response with SSM instances" {
							if *instance.InstanceId == "i-123456789" {
								mockSSM.EXPECT().
									DescribeInstanceInformation(gomock.Any(), &ssm.DescribeInstanceInformationInput{
										Filters: []ssmtypes.InstanceInformationStringFilter{
											{
												Key:    aws.String("InstanceIds"),
												Values: []string{*instance.InstanceId},
											},
										},
									}).
									Return(tt.ssmMockResponse, tt.ssmMockError).
									Times(1)
							} else {
								// Second instance doesn't have SSM
								mockSSM.EXPECT().
									DescribeInstanceInformation(gomock.Any(), &ssm.DescribeInstanceInformationInput{
										Filters: []ssmtypes.InstanceInformationStringFilter{
											{
												Key:    aws.String("InstanceIds"),
												Values: []string{*instance.InstanceId},
											},
										},
									}).
									Return(&ssm.DescribeInstanceInformationOutput{InstanceInformationList: []ssmtypes.InstanceInformation{}}, nil).
									Times(1)
							}
						} else {
							mockSSM.EXPECT().
								DescribeInstanceInformation(gomock.Any(), &ssm.DescribeInstanceInformationInput{
									Filters: []ssmtypes.InstanceInformationStringFilter{
										{
											Key:    aws.String("InstanceIds"),
											Values: []string{*instance.InstanceId},
										},
									},
								}).
								Return(tt.ssmMockResponse, tt.ssmMockError).
								Times(1)
						}
					}
				}
			}

			instances, err := manager.ListAllInstances(context.Background())

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(instances) != tt.expectedCount {
				t.Errorf("Expected %d instances, got %d", tt.expectedCount, len(instances))
			}

			// Verify instance details for successful cases
			if !tt.expectedError && tt.expectedCount > 0 {
				// Check that all expected instances are present (order may vary due to sorting)
				foundIds := make(map[string]bool)
				for _, instance := range instances {
					foundIds[instance.InstanceId] = true
				}
				for _, expectedId := range tt.expectedInstanceIds {
					if !foundIds[expectedId] {
						t.Errorf("Expected instance ID %s not found", expectedId)
					}
				}
			}
		})
	}
}

func TestEC2Manager_hasSSMAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEC2 := mocks.NewMockEC2Client(ctrl)
	mockSSM := mocks.NewMockSSMClient(ctrl)

	manager, err := NewEC2Manager(context.Background(), EC2ManagerOptions{
		EC2Client: mockEC2,
		SSMClient: mockSSM,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name         string
		instanceId   string
		mockResponse *ssm.DescribeInstanceInformationOutput
		mockError    error
		expected     bool
	}{
		{
			name:       "instance has SSM agent",
			instanceId: "i-123456789",
			mockResponse: &ssm.DescribeInstanceInformationOutput{
				InstanceInformationList: []ssmtypes.InstanceInformation{
					{InstanceId: aws.String("i-123456789")},
				},
			},
			expected: true,
		},
		{
			name:       "instance does not have SSM agent",
			instanceId: "i-987654321",
			mockResponse: &ssm.DescribeInstanceInformationOutput{
				InstanceInformationList: []ssmtypes.InstanceInformation{},
			},
			expected: false,
		},
		{
			name:         "SSM API error",
			instanceId:   "i-error",
			mockResponse: nil,
			mockError:    fmt.Errorf("request error"),
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSM.EXPECT().
				DescribeInstanceInformation(gomock.Any(), &ssm.DescribeInstanceInformationInput{
					Filters: []ssmtypes.InstanceInformationStringFilter{
						{
							Key:    aws.String("InstanceIds"),
							Values: []string{tt.instanceId},
						},
					},
				}).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			result := manager.hasSSMAgent(context.Background(), tt.instanceId)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestEC2Manager_getInstanceName(t *testing.T) {
	manager := &EC2Manager{}

	tests := []struct {
		name     string
		tags     []types.Tag
		expected string
	}{
		{
			name: "has name tag",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("WebServer")},
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			expected: "WebServer",
		},
		{
			name: "no name tag",
			tags: []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			expected: "Unnamed",
		},
		{
			name:     "nil tags",
			tags:     nil,
			expected: "Unnamed",
		},
		{
			name:     "empty tags",
			tags:     []types.Tag{},
			expected: "Unnamed",
		},
		{
			name: "name tag with nil value",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: nil},
			},
			expected: "Unnamed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getInstanceName(tt.tags)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEC2Manager_getPlatform(t *testing.T) {
	manager := &EC2Manager{}

	tests := []struct {
		name     string
		instance types.Instance
		expected string
	}{
		{
			name:     "Windows platform",
			instance: types.Instance{Platform: types.PlatformValuesWindows},
			expected: "Windows",
		},
		{
			name:     "empty platform defaults to Linux",
			instance: types.Instance{},
			expected: "Linux",
		},
		{
			name: "Windows detected from Platform tag",
			instance: types.Instance{
				Tags: []types.Tag{
					{Key: aws.String("Platform"), Value: aws.String("Windows")},
				},
			},
			expected: "Windows",
		},
		{
			name: "Windows detected from OS tag",
			instance: types.Instance{
				Tags: []types.Tag{
					{Key: aws.String("OS"), Value: aws.String("Windows Server 2019")},
				},
			},
			expected: "Windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getPlatform(tt.instance)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestEC2Manager_fallbackToCommand(t *testing.T) {
	manager := &EC2Manager{region: "us-east-1"}

	// Test that fallbackToCommand doesn't panic and returns nil
	err := manager.fallbackToCommand("i-123456789")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestEC2Manager_RunRDP_WindowsFiltering(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEC2 := mocks.NewMockEC2Client(ctrl)
	mockSSM := mocks.NewMockSSMClient(ctrl)

	manager, err := NewEC2Manager(ctx, EC2ManagerOptions{
		EC2Client: mockEC2,
		SSMClient: mockSSM,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Failed to create EC2Manager: %v", err)
	}

	// Test data - mix of Windows and Linux instances
	windowsInstanceId := "i-windows123"
	linuxInstanceId := "i-linux456"

	// Mock EC2 response with Windows and Linux instances
	mockEC2.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(
		&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{
				{
					Instances: []types.Instance{
						{
							InstanceId: &windowsInstanceId,
							Platform:   types.PlatformValuesWindows,
							State:      &types.InstanceState{Name: types.InstanceStateNameRunning},
							Tags:       []types.Tag{{Key: aws.String("Name"), Value: aws.String("Windows-Server")}},
						},
						{
							InstanceId: &linuxInstanceId,
							Platform:   "",
							State:      &types.InstanceState{Name: types.InstanceStateNameRunning},
							Tags:       []types.Tag{{Key: aws.String("Name"), Value: aws.String("Linux-Server")}},
						},
					},
				},
			},
		}, nil)

	// Mock SSM response for Windows instance
	mockSSM.EXPECT().DescribeInstanceInformation(gomock.Any(), gomock.Any()).Return(
		&ssm.DescribeInstanceInformationOutput{
			InstanceInformationList: []ssmtypes.InstanceInformation{
				{InstanceId: &windowsInstanceId},
			},
		}, nil)

	// Mock SSM response for Linux instance (no SSM)
	mockSSM.EXPECT().DescribeInstanceInformation(gomock.Any(), gomock.Any()).Return(
		&ssm.DescribeInstanceInformationOutput{
			InstanceInformationList: []ssmtypes.InstanceInformation{},
		}, nil)

	// Test the Windows filtering logic by calling ListAllInstances and filtering
	allInstances, err := manager.ListAllInstances(ctx)
	if err != nil {
		t.Fatalf("ListAllInstances failed: %v", err)
	}

	// Filter for Windows instances (same logic as RunRDP)
	var windowsInstances []EC2Instance
	for _, instance := range allInstances {
		if strings.ToLower(instance.Platform) == "windows" {
			// Only running instances with SSM are selectable for RDP
			instance.IsSelectable = instance.State == "running" && instance.IsSelectable
			windowsInstances = append(windowsInstances, instance)
		}
	}

	// Verify filtering worked correctly
	if len(windowsInstances) != 1 {
		t.Errorf("Expected 1 Windows instance, got %d", len(windowsInstances))
	}

	if len(windowsInstances) > 0 {
		if windowsInstances[0].InstanceId != windowsInstanceId {
			t.Errorf("Expected Windows instance ID %s, got %s", windowsInstanceId, windowsInstances[0].InstanceId)
		}
		if !windowsInstances[0].IsSelectable {
			t.Error("Windows instance should be selectable (running with SSM)")
		}
	}
}
