package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	swaconfig "github.com/blontic/swa/internal/config"
)

// DisplayAWSContext shows the current AWS account, role, and region
func DisplayAWSContext(ctx context.Context) {
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return // Silently skip if no config
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return // Silently skip if can't get identity
	}

	// Parse ARN to get account and role
	account := *identity.Account
	role := "unknown"

	if identity.Arn != nil {
		// ARN format: arn:aws:sts::123456789012:assumed-role/RoleName/SessionName
		parts := strings.Split(*identity.Arn, "/")
		if len(parts) >= 2 && strings.Contains(*identity.Arn, "assumed-role") {
			role = parts[1]
		}
	}

	region := cfg.Region
	if region == "" {
		region = "default"
	}

	fmt.Printf("| AccountID: %s | Role: %s | Region: %s |\n", account, role, region)
	fmt.Println()
}
