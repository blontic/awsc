package aws

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/blontic/awsc/internal/aws/mocks"
	awscconfig "github.com/blontic/awsc/internal/config"
	"go.uber.org/mock/gomock"
)

// Every "not found" error must say where awsc looked (account, role and
// region), so the user can tell a wrong account or region from a real absence.

const testRegion = "ap-southeast-2"

func withTestSession(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWSC_PROFILE", "")
	if err := awscconfig.SaveSession(os.Getppid(), "awsc-prod", "111111111111", "prod", "Admin", "woodside"); err != nil {
		t.Fatal(err)
	}
}

func assertLocated(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %q error, got nil", want)
	}
	msg := err.Error()
	for _, part := range []string{want, "prod (" + testRegion + ") as Admin"} {
		if !strings.Contains(msg, part) {
			t.Errorf("error %q should contain %q", msg, part)
		}
	}
}

func emptyRDS(ctrl *gomock.Controller) *mocks.MockRDSClient {
	c := mocks.NewMockRDSClient(ctrl)
	c.EXPECT().DescribeDBInstances(gomock.Any(), gomock.Any()).Return(&rds.DescribeDBInstancesOutput{}, nil).AnyTimes()
	c.EXPECT().DescribeDBClusters(gomock.Any(), gomock.Any()).Return(&rds.DescribeDBClustersOutput{}, nil).AnyTimes()
	return c
}

func emptyEC2(ctrl *gomock.Controller) *mocks.MockEC2Client {
	c := mocks.NewMockEC2Client(ctrl)
	c.EXPECT().DescribeInstances(gomock.Any(), gomock.Any()).Return(&ec2.DescribeInstancesOutput{}, nil).AnyTimes()
	return c
}

func TestNotFound_RDS(t *testing.T) {
	withTestSession(t)
	ctrl := gomock.NewController(t)
	m, _ := NewRDSManager(context.Background(), RDSManagerOptions{RDSClient: emptyRDS(ctrl), EC2Client: emptyEC2(ctrl), Region: testRegion})
	assertLocated(t, m.RunConnect(context.Background(), "", 0, false), "no RDS instances found in")
}

func TestNotFound_EC2(t *testing.T) {
	withTestSession(t)
	ctrl := gomock.NewController(t)
	m, _ := NewEC2Manager(context.Background(), EC2ManagerOptions{EC2Client: emptyEC2(ctrl), SSMClient: mocks.NewMockSSMClient(ctrl), Region: testRegion})
	assertLocated(t, m.RunConnect(context.Background(), ""), "no EC2 instances found in")
	assertLocated(t, m.RunRDP(context.Background(), "", 0), "no Windows EC2 instances found in")
}

func TestNotFound_OpenSearch(t *testing.T) {
	withTestSession(t)
	ctrl := gomock.NewController(t)
	client := mocks.NewMockOpenSearchClient(ctrl)
	client.EXPECT().ListDomainNames(gomock.Any(), gomock.Any()).Return(&opensearch.ListDomainNamesOutput{}, nil).AnyTimes()
	m, _ := NewOpenSearchManager(context.Background(), OpenSearchManagerOptions{OpenSearchClient: client, EC2Client: emptyEC2(ctrl), Region: testRegion})
	assertLocated(t, m.RunConnect(context.Background(), "", 0, false), "no OpenSearch domains found in")
}

func TestNotFound_Secrets(t *testing.T) {
	withTestSession(t)
	ctrl := gomock.NewController(t)
	c := mocks.NewMockSecretsManagerClient(ctrl)
	c.EXPECT().ListSecrets(gomock.Any(), gomock.Any()).Return(&secretsmanager.ListSecretsOutput{}, nil).AnyTimes()
	m, _ := NewSecretsManager(context.Background(), SecretsManagerOptions{Client: c, Region: testRegion})
	assertLocated(t, m.RunShowSecrets(context.Background(), ""), "no secrets found in")
}

func TestNamedNotFoundError(t *testing.T) {
	withTestSession(t)
	assertLocated(t, namedNotFoundError("RDS instance", "missing-db", testRegion), "RDS instance 'missing-db' not found in")
}
