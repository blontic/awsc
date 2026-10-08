package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	awscconfig "github.com/blontic/awsc/internal/config"
)

// STSClient interface for mocking
type STSClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

// IdentityManager checks which identity the terminal's credentials belong to.
type IdentityManager struct {
	client STSClient
}

type IdentityManagerOptions struct {
	Client STSClient
}

func NewIdentityManager(ctx context.Context, opts ...IdentityManagerOptions) (*IdentityManager, error) {
	if len(opts) > 0 && opts[0].Client != nil {
		return &IdentityManager{client: opts[0].Client}, nil
	}
	m := &IdentityManager{}
	if err := m.reloadClients(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *IdentityManager) reloadClients(ctx context.Context) error {
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return err
	}
	m.client = sts.NewFromConfig(cfg)
	return nil
}

// CallerARN returns the ARN of the identity the credentials belong to, which
// also proves they are valid.
func (m *IdentityManager) CallerARN(ctx context.Context) (string, error) {
	out, err := withReauth(ctx, m.reloadClients, func() (*sts.GetCallerIdentityOutput, error) {
		return m.client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	})
	if err != nil {
		return "", err
	}
	if out == nil || out.Arn == nil {
		return "", fmt.Errorf("unexpected empty response from STS")
	}
	return aws.ToString(out.Arn), nil
}
