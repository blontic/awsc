package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2sdk "github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	opensearchtypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/blontic/awsc/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewOpenSearchManager(t *testing.T) {
	ctx := context.Background()

	// Test with mock clients
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearchClient := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2Client := mocks.NewMockEC2Client(ctrl)

	manager, err := NewOpenSearchManager(ctx, OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearchClient,
		EC2Client:        mockEC2Client,
		Region:           "us-east-1",
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.region != "us-east-1" {
		t.Errorf("Expected region to be us-east-1, got %s", manager.region)
	}
}

func TestListOpenSearchDomains(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearchClient := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2Client := mocks.NewMockEC2Client(ctrl)

	manager, _ := NewOpenSearchManager(ctx, OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearchClient,
		EC2Client:        mockEC2Client,
		Region:           "us-east-1",
	})

	// Mock domain list response
	domainName := "test-domain"
	mockOpenSearchClient.EXPECT().
		ListDomainNames(ctx, &opensearch.ListDomainNamesInput{}).
		Return(&opensearch.ListDomainNamesOutput{
			DomainNames: []opensearchtypes.DomainInfo{
				{DomainName: &domainName},
			},
		}, nil)

	// Mock domain details response
	enforceHTTPS := true
	processing := false
	engineVersion := "OpenSearch_2.3"
	endpoints := map[string]string{"vpc": "vpc-test-domain-123.us-east-1.es.amazonaws.com"}
	mockOpenSearchClient.EXPECT().
		DescribeDomain(ctx, &opensearch.DescribeDomainInput{
			DomainName: &domainName,
		}).
		Return(&opensearch.DescribeDomainOutput{
			DomainStatus: &opensearchtypes.DomainStatus{
				DomainName:    &domainName,
				Processing:    &processing,
				EngineVersion: &engineVersion,
				Endpoints:     endpoints,
				DomainEndpointOptions: &opensearchtypes.DomainEndpointOptions{
					EnforceHTTPS: &enforceHTTPS,
				},
				VPCOptions: &opensearchtypes.VPCDerivedInfo{
					SecurityGroupIds: []string{"sg-123456"},
				},
			},
		}, nil)

	domains, err := manager.ListOpenSearchDomains(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(domains) != 1 {
		t.Fatalf("Expected 1 domain, got %d", len(domains))
	}

	domain := domains[0]
	if domain.Name != "test-domain" {
		t.Errorf("Expected domain name to be test-domain, got %s", domain.Name)
	}

	if domain.Endpoint != "vpc-test-domain-123.us-east-1.es.amazonaws.com" {
		t.Errorf("Expected endpoint to be vpc-test-domain-123.us-east-1.es.amazonaws.com, got %s", domain.Endpoint)
	}

	if domain.Port != 443 {
		t.Errorf("Expected port to be 443, got %d", domain.Port)
	}

	if domain.Version != "OpenSearch_2.3" {
		t.Errorf("Expected version to be OpenSearch_2.3, got %s", domain.Version)
	}
}

func TestOpenSearchManager_FindBastionHosts_EmptyResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearch := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, _ := NewOpenSearchManager(context.Background(), OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearch,
		EC2Client:        mockEC2,
		Region:           "us-east-1",
	})

	domain := OpenSearchDomain{Name: "test-domain", Port: 443}

	// Mock OpenSearch security groups
	mockOpenSearch.EXPECT().
		DescribeDomain(gomock.Any(), gomock.Any()).
		Return(&opensearch.DescribeDomainOutput{
			DomainStatus: &opensearchtypes.DomainStatus{
				VPCOptions: &opensearchtypes.VPCDerivedInfo{
					SecurityGroupIds: []string{"sg-os-123"},
				},
			},
		}, nil)

	// Mock pre-fetch of SG rules
	mockEC2.EXPECT().
		DescribeSecurityGroups(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2types.SecurityGroup{
				{IpPermissions: []ec2types.IpPermission{}},
			},
		}, nil)

	// Mock EC2 instances - empty
	mockEC2.EXPECT().
		DescribeInstances(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeInstancesOutput{Reservations: []ec2types.Reservation{}}, nil)

	bastions, err := manager.FindBastionHosts(context.Background(), domain, false)
	if err == nil {
		t.Error("Expected error when no running instances found")
	}
	if len(bastions) != 0 {
		t.Errorf("Expected 0 bastions, got %d", len(bastions))
	}
}

func TestOpenSearchManager_FindBastionHosts_WithStoppedInstances(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearch := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, _ := NewOpenSearchManager(context.Background(), OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearch,
		EC2Client:        mockEC2,
		Region:           "us-east-1",
	})

	domain := OpenSearchDomain{Name: "test-domain", Port: 443}

	// Mock OpenSearch security groups
	mockOpenSearch.EXPECT().
		DescribeDomain(gomock.Any(), gomock.Any()).
		Return(&opensearch.DescribeDomainOutput{
			DomainStatus: &opensearchtypes.DomainStatus{
				VPCOptions: &opensearchtypes.VPCDerivedInfo{
					SecurityGroupIds: []string{"sg-os-123"},
				},
			},
		}, nil)

	// Mock pre-fetch of SG rules
	mockEC2.EXPECT().
		DescribeSecurityGroups(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2types.SecurityGroup{
				{IpPermissions: []ec2types.IpPermission{}},
			},
		}, nil)

	// Mock EC2 instances - only stopped
	mockEC2.EXPECT().
		DescribeInstances(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeInstancesOutput{
			Reservations: []ec2types.Reservation{
				{
					Instances: []ec2types.Instance{
						{
							InstanceId: aws.String("i-stopped-1"),
							State:      &ec2types.InstanceState{Name: "stopped"},
							Tags:       []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("bastion")}},
							SecurityGroups: []ec2types.GroupIdentifier{
								{GroupId: aws.String("sg-ec2-456")},
							},
						},
					},
				},
			},
		}, nil)

	bastions, err := manager.FindBastionHosts(context.Background(), domain, false)
	if err == nil {
		t.Error("Expected error when only stopped instances found")
	}
	if len(bastions) != 0 {
		t.Errorf("Expected 0 bastions, got %d", len(bastions))
	}
}

func TestOpenSearchManager_canConnectWithCachedRules(t *testing.T) {
	manager := &OpenSearchManager{}

	tests := []struct {
		name         string
		ec2SGs       []ec2types.GroupIdentifier
		sgRulesCache map[string][]ec2types.IpPermission
		port         int32
		expected     bool
	}{
		{
			name:   "SG allows access from EC2 SG",
			ec2SGs: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
			sgRulesCache: map[string][]ec2types.IpPermission{
				"sg-os-123": {
					{
						FromPort: aws.Int32(443),
						ToPort:   aws.Int32(443),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: aws.String("sg-ec2-456")},
						},
					},
				},
			},
			port:     443,
			expected: true,
		},
		{
			name:   "SG allows open access",
			ec2SGs: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
			sgRulesCache: map[string][]ec2types.IpPermission{
				"sg-os-123": {
					{
						FromPort: aws.Int32(443),
						ToPort:   aws.Int32(443),
						IpRanges: []ec2types.IpRange{
							{CidrIp: aws.String("0.0.0.0/0")},
						},
					},
				},
			},
			port:     443,
			expected: true,
		},
		{
			name:   "SG denies access - wrong port and SG",
			ec2SGs: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
			sgRulesCache: map[string][]ec2types.IpPermission{
				"sg-os-123": {
					{
						FromPort: aws.Int32(5432),
						ToPort:   aws.Int32(5432),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: aws.String("sg-ec2-different")},
						},
					},
				},
			},
			port:     443,
			expected: false,
		},
		{
			name:   "all-traffic rule matches",
			ec2SGs: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
			sgRulesCache: map[string][]ec2types.IpPermission{
				"sg-os-123": {
					{
						IpProtocol: aws.String("-1"),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: aws.String("sg-ec2-456")},
						},
					},
				},
			},
			port:     443,
			expected: true,
		},
		{
			name:         "empty rules cache",
			ec2SGs:       []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
			sgRulesCache: map[string][]ec2types.IpPermission{},
			port:         443,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.canConnectWithCachedRules(tt.ec2SGs, tt.sgRulesCache, tt.port)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestOpenSearchManager_ruleMatchesPort(t *testing.T) {
	manager := &OpenSearchManager{}

	tests := []struct {
		name     string
		rule     ec2types.IpPermission
		port     int32
		expected bool
	}{
		{
			name: "port matches exactly",
			rule: ec2types.IpPermission{
				FromPort: aws.Int32(443),
				ToPort:   aws.Int32(443),
			},
			port:     443,
			expected: true,
		},
		{
			name: "port within range",
			rule: ec2types.IpPermission{
				FromPort: aws.Int32(400),
				ToPort:   aws.Int32(500),
			},
			port:     443,
			expected: true,
		},
		{
			name: "port outside range",
			rule: ec2types.IpPermission{
				FromPort: aws.Int32(5000),
				ToPort:   aws.Int32(6000),
			},
			port:     443,
			expected: false,
		},
		{
			name: "nil ports without protocol",
			rule: ec2types.IpPermission{
				FromPort: nil,
				ToPort:   nil,
			},
			port:     443,
			expected: false,
		},
		{
			name: "all-traffic rule (protocol -1)",
			rule: ec2types.IpPermission{
				IpProtocol: aws.String("-1"),
				FromPort:   nil,
				ToPort:     nil,
			},
			port:     443,
			expected: true,
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

func TestOpenSearchManager_FindBastionHosts_SuccessfulMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearch := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, _ := NewOpenSearchManager(context.Background(), OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearch,
		EC2Client:        mockEC2,
		Region:           "us-east-1",
	})

	domain := OpenSearchDomain{Name: "test-domain", Port: 443}

	// Mock OpenSearch security groups
	mockOpenSearch.EXPECT().
		DescribeDomain(gomock.Any(), gomock.Any()).
		Return(&opensearch.DescribeDomainOutput{
			DomainStatus: &opensearchtypes.DomainStatus{
				VPCOptions: &opensearchtypes.VPCDerivedInfo{
					SecurityGroupIds: []string{"sg-os-123"},
				},
			},
		}, nil)

	// Mock pre-fetch of SG rules - allows sg-ec2-456 on port 443
	mockEC2.EXPECT().
		DescribeSecurityGroups(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2types.SecurityGroup{
				{IpPermissions: []ec2types.IpPermission{
					{
						FromPort: aws.Int32(443),
						ToPort:   aws.Int32(443),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: aws.String("sg-ec2-456")},
						},
					},
				}},
			},
		}, nil)

	// Mock EC2 instances - one running with matching SG
	mockEC2.EXPECT().
		DescribeInstances(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeInstancesOutput{
			Reservations: []ec2types.Reservation{
				{Instances: []ec2types.Instance{
					{
						InstanceId:     aws.String("i-bastion-1"),
						State:          &ec2types.InstanceState{Name: "running"},
						Tags:           []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("bastion")}},
						SecurityGroups: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
					},
				}},
			},
		}, nil)

	bastions, err := manager.FindBastionHosts(context.Background(), domain, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(bastions) != 1 {
		t.Fatalf("Expected 1 bastion, got %d", len(bastions))
	}
	if bastions[0].InstanceId != "i-bastion-1" {
		t.Errorf("Expected instance i-bastion-1, got %s", bastions[0].InstanceId)
	}
}

func TestOpenSearchManager_FindBastionHosts_RunningButNoSGMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOpenSearch := mocks.NewMockOpenSearchClient(ctrl)
	mockEC2 := mocks.NewMockEC2Client(ctrl)

	manager, _ := NewOpenSearchManager(context.Background(), OpenSearchManagerOptions{
		OpenSearchClient: mockOpenSearch,
		EC2Client:        mockEC2,
		Region:           "us-east-1",
	})

	domain := OpenSearchDomain{Name: "test-domain", Port: 443}

	// Mock OpenSearch security groups
	mockOpenSearch.EXPECT().
		DescribeDomain(gomock.Any(), gomock.Any()).
		Return(&opensearch.DescribeDomainOutput{
			DomainStatus: &opensearchtypes.DomainStatus{
				VPCOptions: &opensearchtypes.VPCDerivedInfo{
					SecurityGroupIds: []string{"sg-os-123"},
				},
			},
		}, nil)

	// Mock pre-fetch of SG rules - allows sg-other only
	mockEC2.EXPECT().
		DescribeSecurityGroups(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeSecurityGroupsOutput{
			SecurityGroups: []ec2types.SecurityGroup{
				{IpPermissions: []ec2types.IpPermission{
					{
						FromPort: aws.Int32(443),
						ToPort:   aws.Int32(443),
						UserIdGroupPairs: []ec2types.UserIdGroupPair{
							{GroupId: aws.String("sg-other")},
						},
					},
				}},
			},
		}, nil)

	// Mock EC2 instances - running but SG doesn't match
	mockEC2.EXPECT().
		DescribeInstances(gomock.Any(), gomock.Any()).
		Return(&ec2sdk.DescribeInstancesOutput{
			Reservations: []ec2types.Reservation{
				{Instances: []ec2types.Instance{
					{
						InstanceId:     aws.String("i-wrong-sg"),
						State:          &ec2types.InstanceState{Name: "running"},
						Tags:           []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("web-server")}},
						SecurityGroups: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-ec2-456")}},
					},
				}},
			},
		}, nil)

	bastions, err := manager.FindBastionHosts(context.Background(), domain, false)
	if err == nil {
		t.Error("Expected error when no SG match")
	}
	if len(bastions) != 0 {
		t.Errorf("Expected 0 bastions, got %d", len(bastions))
	}
}
