package aws

import (
	"context"
	"testing"

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
