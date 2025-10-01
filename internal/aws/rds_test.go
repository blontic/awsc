package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/blontic/swa/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewRDSManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
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

func TestNewRDSManagerWithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
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

func TestRDSManager_ListRDSInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name          string
		mockResponse  *rds.DescribeDBInstancesOutput
		mockError     error
		expectedCount int
		expectedError bool
	}{
		{
			name: "successful response with available instances",
			mockResponse: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{
					{
						DBInstanceIdentifier: aws.String("test-db-1"),
						DBInstanceStatus:     aws.String("available"),
						Engine:               aws.String("mysql"),
						Endpoint: &rdstypes.Endpoint{
							Address: aws.String("test-db-1.cluster-xyz.us-east-1.rds.amazonaws.com"),
							Port:    aws.Int32(3306),
						},
					},
					{
						DBInstanceIdentifier: aws.String("test-db-2"),
						DBInstanceStatus:     aws.String("available"),
						Engine:               aws.String("postgres"),
						Endpoint: &rdstypes.Endpoint{
							Address: aws.String("test-db-2.cluster-abc.us-east-1.rds.amazonaws.com"),
							Port:    aws.Int32(5432),
						},
					},
				},
			},
			expectedCount: 2,
			expectedError: false,
		},
		{
			name: "response with unavailable instances",
			mockResponse: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{
					{
						DBInstanceIdentifier: aws.String("test-db-stopped"),
						DBInstanceStatus:     aws.String("stopped"),
						Engine:               aws.String("mysql"),
						Endpoint: &rdstypes.Endpoint{
							Address: aws.String("test-db-stopped.cluster-xyz.us-east-1.rds.amazonaws.com"),
							Port:    aws.Int32(3306),
						},
					},
				},
			},
			expectedCount: 0,
			expectedError: false,
		},
		{
			name:          "empty response",
			mockResponse:  &rds.DescribeDBInstancesOutput{DBInstances: []rdstypes.DBInstance{}},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRDS.EXPECT().
				DescribeDBInstances(gomock.Any(), gomock.Any()).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			instances, err := manager.ListRDSInstances(context.Background())

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
				for i, instance := range instances {
					expectedDB := tt.mockResponse.DBInstances[i]
					if instance.Identifier != *expectedDB.DBInstanceIdentifier {
						t.Errorf("Expected identifier %s, got %s", *expectedDB.DBInstanceIdentifier, instance.Identifier)
					}
					if instance.Engine != *expectedDB.Engine {
						t.Errorf("Expected engine %s, got %s", *expectedDB.Engine, instance.Engine)
					}
				}
			}
		})
	}
}

func TestRDSManager_getRDSSecurityGroups(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name         string
		dbIdentifier string
		mockResponse *rds.DescribeDBInstancesOutput
		mockError    error
		expectedSGs  []string
		expectedErr  bool
	}{
		{
			name:         "successful response with security groups",
			dbIdentifier: "test-db",
			mockResponse: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{
					{
						VpcSecurityGroups: []rdstypes.VpcSecurityGroupMembership{
							{VpcSecurityGroupId: aws.String("sg-123456")},
							{VpcSecurityGroupId: aws.String("sg-789012")},
						},
					},
				},
			},
			expectedSGs: []string{"sg-123456", "sg-789012"},
			expectedErr: false,
		},
		{
			name:         "empty response",
			dbIdentifier: "nonexistent-db",
			mockResponse: &rds.DescribeDBInstancesOutput{DBInstances: []rdstypes.DBInstance{}},
			expectedSGs:  nil,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRDS.EXPECT().
				DescribeDBInstances(gomock.Any(), &rds.DescribeDBInstancesInput{
					DBInstanceIdentifier: aws.String(tt.dbIdentifier),
				}).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			sgs, err := manager.getRDSSecurityGroups(context.Background(), tt.dbIdentifier)

			if tt.expectedErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(sgs) != len(tt.expectedSGs) {
				t.Errorf("Expected %d security groups, got %d", len(tt.expectedSGs), len(sgs))
			}
			for i, sg := range sgs {
				if i < len(tt.expectedSGs) && sg != tt.expectedSGs[i] {
					t.Errorf("Expected security group %s, got %s", tt.expectedSGs[i], sg)
				}
			}
		})
	}
}

func TestRDSManager_checkSecurityGroupRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name         string
		rdsSgId      string
		ec2SgIds     map[string]bool
		port         int32
		mockResponse *ec2.DescribeSecurityGroupsOutput
		mockError    error
		expected     bool
	}{
		{
			name:     "security group allows access from EC2 SG",
			rdsSgId:  "sg-rds-123",
			ec2SgIds: map[string]bool{"sg-ec2-456": true},
			port:     3306,
			mockResponse: &ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{
						IpPermissions: []types.IpPermission{
							{
								FromPort: aws.Int32(3306),
								ToPort:   aws.Int32(3306),
								UserIdGroupPairs: []types.UserIdGroupPair{
									{GroupId: aws.String("sg-ec2-456")},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name:     "security group allows open access",
			rdsSgId:  "sg-rds-123",
			ec2SgIds: map[string]bool{"sg-ec2-456": true},
			port:     3306,
			mockResponse: &ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{
						IpPermissions: []types.IpPermission{
							{
								FromPort: aws.Int32(3306),
								ToPort:   aws.Int32(3306),
								IpRanges: []types.IpRange{
									{CidrIp: aws.String("0.0.0.0/0")},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name:     "security group denies access",
			rdsSgId:  "sg-rds-123",
			ec2SgIds: map[string]bool{"sg-ec2-456": true},
			port:     3306,
			mockResponse: &ec2.DescribeSecurityGroupsOutput{
				SecurityGroups: []types.SecurityGroup{
					{
						IpPermissions: []types.IpPermission{
							{
								FromPort: aws.Int32(5432),
								ToPort:   aws.Int32(5432),
								UserIdGroupPairs: []types.UserIdGroupPair{
									{GroupId: aws.String("sg-ec2-different")},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEC2.EXPECT().
				DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
					GroupIds: []string{tt.rdsSgId},
				}).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			result := manager.checkSecurityGroupRules(context.Background(), tt.rdsSgId, tt.ec2SgIds, tt.port)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRDSManager_ruleMatchesPort(t *testing.T) {
	manager := &RDSManager{}

	tests := []struct {
		name     string
		rule     types.IpPermission
		port     int32
		expected bool
	}{
		{
			name: "port matches exactly",
			rule: types.IpPermission{
				FromPort: aws.Int32(3306),
				ToPort:   aws.Int32(3306),
			},
			port:     3306,
			expected: true,
		},
		{
			name: "port within range",
			rule: types.IpPermission{
				FromPort: aws.Int32(3000),
				ToPort:   aws.Int32(4000),
			},
			port:     3306,
			expected: true,
		},
		{
			name: "port outside range",
			rule: types.IpPermission{
				FromPort: aws.Int32(5000),
				ToPort:   aws.Int32(6000),
			},
			port:     3306,
			expected: false,
		},
		{
			name: "nil ports",
			rule: types.IpPermission{
				FromPort: nil,
				ToPort:   nil,
			},
			port:     3306,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.ruleMatchesPort(tt.rule, tt.port)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRDSManager_getInstanceName(t *testing.T) {
	manager := &RDSManager{}

	tests := []struct {
		name     string
		tags     []types.Tag
		expected string
	}{
		{
			name: "has name tag",
			tags: []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("MyInstance")},
				{Key: aws.String("Environment"), Value: aws.String("prod")},
			},
			expected: "MyInstance",
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

func TestRDSManager_getSecurityGroupIds(t *testing.T) {
	manager := &RDSManager{}

	tests := []struct {
		name     string
		sgs      []types.GroupIdentifier
		expected []string
	}{
		{
			name: "multiple security groups",
			sgs: []types.GroupIdentifier{
				{GroupId: aws.String("sg-123")},
				{GroupId: aws.String("sg-456")},
			},
			expected: []string{"sg-123", "sg-456"},
		},
		{
			name:     "nil security groups",
			sgs:      nil,
			expected: []string{},
		},
		{
			name:     "empty security groups",
			sgs:      []types.GroupIdentifier{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getSecurityGroupIds(tt.sgs)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d security groups, got %d", len(tt.expected), len(result))
			}
			for i, sg := range result {
				if i < len(tt.expected) && sg != tt.expected[i] {
					t.Errorf("Expected security group %s, got %s", tt.expected[i], sg)
				}
			}
		})
	}
}

func TestRDSManager_canConnectToRDS(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	ec2SecurityGroups := []types.GroupIdentifier{
		{GroupId: aws.String("sg-ec2-123")},
	}
	rdsSecurityGroups := []string{"sg-rds-456"}

	// Mock the security group rules check
	mockEC2.EXPECT().
		DescribeSecurityGroups(gomock.Any(), &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{"sg-rds-456"},
		}).
		Return(&ec2.DescribeSecurityGroupsOutput{
			SecurityGroups: []types.SecurityGroup{
				{
					IpPermissions: []types.IpPermission{
						{
							FromPort: aws.Int32(3306),
							ToPort:   aws.Int32(3306),
							UserIdGroupPairs: []types.UserIdGroupPair{
								{GroupId: aws.String("sg-ec2-123")},
							},
						},
					},
				},
			},
		}, nil).
		Times(1)

	result := manager.canConnectToRDS(context.Background(), ec2SecurityGroups, rdsSecurityGroups, 3306)
	if !result {
		t.Error("Expected canConnectToRDS to return true")
	}
}

func TestRDSManager_FindBastionHosts_EmptyResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRDS := mocks.NewMockRDSClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, err := NewRDSManager(context.Background(), RDSManagerOptions{
		RDSClient: mockRDS,
		EC2Client: mockEC2,
		SSMClient: nil,
		Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	rdsInstance := RDSInstance{
		Identifier: "test-db",
		Port:       3306,
	}

	// Mock RDS security groups call
	mockRDS.EXPECT().
		DescribeDBInstances(gomock.Any(), &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String("test-db"),
		}).
		Return(&rds.DescribeDBInstancesOutput{
			DBInstances: []rdstypes.DBInstance{
				{
					VpcSecurityGroups: []rdstypes.VpcSecurityGroupMembership{
						{VpcSecurityGroupId: aws.String("sg-rds-123")},
					},
				},
			},
		}, nil).
		Times(1)

	// Mock EC2 instances call - empty response
	mockEC2.EXPECT().
		DescribeInstances(gomock.Any(), gomock.Any()).
		Return(&ec2.DescribeInstancesOutput{
			Reservations: []types.Reservation{},
		}, nil).
		Times(1)

	bastions, err := manager.FindBastionHosts(context.Background(), rdsInstance)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(bastions) != 0 {
		t.Errorf("Expected 0 bastions, got %d", len(bastions))
	}
}
