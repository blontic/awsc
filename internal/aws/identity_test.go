package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/blontic/awsc/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

func TestIdentityCallerARN(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockSTSClient(ctrl)
	arn := "arn:aws:sts::111111111111:assumed-role/AWSReservedSSO_Admin_abc/me@example.com"
	client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(&sts.GetCallerIdentityOutput{Arn: aws.String(arn)}, nil)

	m, err := NewIdentityManager(context.Background(), IdentityManagerOptions{Client: client})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := m.CallerARN(context.Background()); err != nil || got != arn {
		t.Errorf("CallerARN = %q, %v", got, err)
	}
}

func TestIdentityCallerARN_Errors(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mocks.NewMockSTSClient(ctrl)
	client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(&sts.GetCallerIdentityOutput{}, nil)
	client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(nil, errors.New("AccessDenied"))

	m, _ := NewIdentityManager(context.Background(), IdentityManagerOptions{Client: client})
	if _, err := m.CallerARN(context.Background()); err == nil {
		t.Error("expected an error for an empty response")
	}
	if _, err := m.CallerARN(context.Background()); err == nil {
		t.Error("expected the STS error")
	}
}

func TestIdentityCallerARN_ReauthOnExpiredCredentials(t *testing.T) {
	original := promptForReauth
	defer func() { promptForReauth = original }()
	prompted := false
	promptForReauth = func(context.Context) (bool, error) { prompted = true; return false, nil }

	ctrl := gomock.NewController(t)
	client := mocks.NewMockSTSClient(ctrl)
	client.EXPECT().GetCallerIdentity(gomock.Any(), gomock.Any()).Return(nil, errors.New("ExpiredToken: token expired"))

	m, _ := NewIdentityManager(context.Background(), IdentityManagerOptions{Client: client})
	if _, err := m.CallerARN(context.Background()); err == nil || !prompted {
		t.Errorf("expected a reauth prompt and error, got prompted=%v err=%v", prompted, err)
	}
}
