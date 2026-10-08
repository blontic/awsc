package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	ostypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/blontic/awsc/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

// When credentials have expired and the user declines to log in, each AWS
// call made by a manager must be tried once and return the original error.

var errExpired = errors.New("api error ExpiredToken: The security token included in the request is expired")

func TestManagers_ExpiredCredentials_LoginDeclined(t *testing.T) {
	ctx := context.Background()

	t.Run("RDS", func(t *testing.T) {
		prompts := stubReauth(t, false, nil)
		ctrl := gomock.NewController(t)
		c := mocks.NewMockRDSClient(ctrl)
		c.EXPECT().DescribeDBInstances(gomock.Any(), gomock.Any()).Return(nil, errExpired).Times(1)
		m, _ := NewRDSManager(ctx, RDSManagerOptions{RDSClient: c, EC2Client: mocks.NewMockEC2Client(ctrl), Region: testRegion})
		if _, err := m.ListRDSInstances(ctx); !errors.Is(err, errExpired) || *prompts != 1 {
			t.Errorf("err=%v prompts=%d", err, *prompts)
		}
	})

	t.Run("EC2", func(t *testing.T) {
		prompts := stubReauth(t, false, nil)
		ctrl := gomock.NewController(t)
		c := mocks.NewMockEC2Client(ctrl)
		c.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(nil, errExpired).Times(1)
		m, _ := NewEC2Manager(ctx, EC2ManagerOptions{EC2Client: c, SSMClient: mocks.NewMockSSMClient(ctrl), Region: testRegion})
		if _, err := m.ListAllInstances(ctx); !errors.Is(err, errExpired) || *prompts != 1 {
			t.Errorf("err=%v prompts=%d", err, *prompts)
		}
	})

	t.Run("OpenSearch", func(t *testing.T) {
		prompts := stubReauth(t, false, nil)
		ctrl := gomock.NewController(t)
		c := mocks.NewMockOpenSearchClient(ctrl)
		c.EXPECT().ListDomainNames(gomock.Any(), gomock.Any()).Return(nil, errExpired).Times(1)
		m, _ := NewOpenSearchManager(ctx, OpenSearchManagerOptions{OpenSearchClient: c, EC2Client: mocks.NewMockEC2Client(ctrl), Region: testRegion})
		if _, err := m.ListOpenSearchDomains(ctx); !errors.Is(err, errExpired) || *prompts != 1 {
			t.Errorf("err=%v prompts=%d", err, *prompts)
		}
	})

	t.Run("Secrets list and get", func(t *testing.T) {
		prompts := stubReauth(t, false, nil)
		ctrl := gomock.NewController(t)
		c := mocks.NewMockSecretsManagerClient(ctrl)
		c.EXPECT().ListSecrets(gomock.Any(), gomock.Any()).Return(nil, errExpired).Times(1)
		c.EXPECT().GetSecretValue(gomock.Any(), gomock.Any()).Return(nil, errExpired).Times(1)
		m, _ := NewSecretsManager(ctx, SecretsManagerOptions{Client: c, Region: testRegion})
		if _, err := m.ListSecrets(ctx); !errors.Is(err, errExpired) {
			t.Errorf("ListSecrets err=%v", err)
		}
		if _, err := m.GetSecretValue(ctx, "s"); !errors.Is(err, errExpired) || *prompts != 2 {
			t.Errorf("GetSecretValue err=%v prompts=%d", err, *prompts)
		}
	})
}

func TestManagers_NonAuthErrors_NoLoginPrompt(t *testing.T) {
	ctx := context.Background()
	prompts := stubReauth(t, true, nil)
	ctrl := gomock.NewController(t)
	throttled := errors.New("api error Throttling: Rate exceeded")
	c := mocks.NewMockRDSClient(ctrl)
	c.EXPECT().DescribeDBInstances(gomock.Any(), gomock.Any()).Return(nil, throttled).Times(1)
	m, _ := NewRDSManager(ctx, RDSManagerOptions{RDSClient: c, EC2Client: mocks.NewMockEC2Client(ctrl), Region: testRegion})
	if _, err := m.ListRDSInstances(ctx); !errors.Is(err, throttled) || *prompts != 0 {
		t.Errorf("err=%v prompts=%d", err, *prompts)
	}
}

// A domain that cannot be described (for a non-auth reason) is skipped; the
// others are still listed.
func TestOpenSearch_SkipsDomainThatFailsToDescribe(t *testing.T) {
	ctx := context.Background()
	stubReauth(t, false, nil)
	ctrl := gomock.NewController(t)
	c := mocks.NewMockOpenSearchClient(ctrl)
	c.EXPECT().ListDomainNames(gomock.Any(), gomock.Any()).Return(&opensearch.ListDomainNamesOutput{
		DomainNames: []ostypes.DomainInfo{{DomainName: aws.String("broken")}, {DomainName: aws.String("good")}},
	}, nil)
	c.EXPECT().DescribeDomain(gomock.Any(), &opensearch.DescribeDomainInput{DomainName: aws.String("broken")}).Return(nil, errors.New("ResourceNotFoundException"))
	c.EXPECT().DescribeDomain(gomock.Any(), &opensearch.DescribeDomainInput{DomainName: aws.String("good")}).Return(&opensearch.DescribeDomainOutput{
		DomainStatus: &ostypes.DomainStatus{
			DomainName:            aws.String("good"),
			DomainEndpointOptions: &ostypes.DomainEndpointOptions{EnforceHTTPS: aws.Bool(true)},
			VPCOptions:            &ostypes.VPCDerivedInfo{SecurityGroupIds: []string{"sg-1"}},
			Endpoints:             map[string]string{"vpc": "vpc-good.es.amazonaws.com"},
		},
	}, nil)
	m, _ := NewOpenSearchManager(ctx, OpenSearchManagerOptions{OpenSearchClient: c, EC2Client: mocks.NewMockEC2Client(ctrl), Region: testRegion})

	domains, err := m.ListOpenSearchDomains(ctx)
	if err != nil || len(domains) != 1 || domains[0].Name != "good" {
		t.Errorf("domains=%+v err=%v", domains, err)
	}
}

func TestEC2_HasSSMAgent_ErrorMeansNoAgent(t *testing.T) {
	ctx := context.Background()
	stubReauth(t, false, nil)
	ctrl := gomock.NewController(t)
	s := mocks.NewMockSSMClient(ctrl)
	s.EXPECT().DescribeInstanceInformation(gomock.Any(), gomock.Any()).Return(nil, errExpired)
	s.EXPECT().DescribeInstanceInformation(gomock.Any(), gomock.Any()).Return(&ssm.DescribeInstanceInformationOutput{}, nil)
	m, _ := NewEC2Manager(ctx, EC2ManagerOptions{EC2Client: mocks.NewMockEC2Client(ctrl), SSMClient: s, Region: testRegion})
	if m.hasSSMAgent(ctx, "i-1") {
		t.Error("an error checking SSM should mean no agent")
	}
	if m.hasSSMAgent(ctx, "i-2") {
		t.Error("no instance information should mean no agent")
	}
}
