package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretstypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/blontic/awsc/internal/ui"
)

// SecretsManagerClient interface for mocking
type SecretsManagerClient interface {
	ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type SecretsManager struct {
	client SecretsManagerClient
	region string
}

type Secret struct {
	Name        string
	Description string
	ARN         string
}

type SecretsManagerOptions struct {
	Client SecretsManagerClient
	Region string
}

func NewSecretsManager(ctx context.Context, opts ...SecretsManagerOptions) (*SecretsManager, error) {
	if len(opts) > 0 && opts[0].Client != nil {
		// Use provided client (for testing)
		return &SecretsManager{
			client: opts[0].Client,
			region: opts[0].Region,
		}, nil
	}

	// Production path
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &SecretsManager{
		client: secretsmanager.NewFromConfig(cfg),
		region: cfg.Region,
	}, nil
}

func (s *SecretsManager) ListSecrets(ctx context.Context) ([]Secret, error) {
	var allSecrets []secretstypes.SecretListEntry
	var nextToken *string

	for {
		result, err := s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
			NextToken: nextToken,
		})
		if err != nil {
			if IsAuthError(err) {
				if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
					// Reload client with fresh credentials
					if reloadErr := s.reloadClient(ctx); reloadErr != nil {
						return nil, reloadErr
					}
					// Retry after re-authentication
					result, err = s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{
						NextToken: nextToken,
					})
					if err != nil {
						return nil, err
					}
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		allSecrets = append(allSecrets, result.SecretList...)

		// Check if there are more pages
		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	var secrets []Secret
	for _, secret := range allSecrets {
		description := ""
		if secret.Description != nil {
			description = *secret.Description
		}

		secrets = append(secrets, Secret{
			Name:        *secret.Name,
			Description: description,
			ARN:         *secret.ARN,
		})
	}

	return secrets, nil
}

func (s *SecretsManager) GetSecretValue(ctx context.Context, secretName string) (string, error) {
	result, err := s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		if IsAuthError(err) {
			if shouldReauth, reAuthErr := PromptForReauth(ctx); shouldReauth && reAuthErr == nil {
				// Reload client with fresh credentials
				if reloadErr := s.reloadClient(ctx); reloadErr != nil {
					return "", reloadErr
				}
				// Retry after re-authentication
				result, err = s.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
					SecretId: aws.String(secretName),
				})
				if err != nil {
					return "", err
				}
			} else {
				return "", err
			}
		} else {
			return "", err
		}
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	return string(result.SecretBinary), nil
}

func (s *SecretsManager) DisplaySecret(ctx context.Context, secretName, secretValue string) {
	fmt.Printf("\n")

	// Try to parse as JSON for pretty printing
	var jsonData interface{}
	if err := json.Unmarshal([]byte(secretValue), &jsonData); err == nil {
		prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
		fmt.Printf("%s\n", string(prettyJSON))
	} else {
		// Display as plain text
		fmt.Printf("%s\n", secretValue)
	}
}

func (s *SecretsManager) RunShowSecrets(ctx context.Context, secretName string) error {
	// If secret name provided, try to show it directly
	if secretName != "" {
		fmt.Printf("Showing secret: %s\n", secretName)

		// Get secret value
		secretValue, err := s.GetSecretValue(ctx, secretName)
		if err != nil {
			fmt.Printf("Secret '%s' not found. Available secrets:\n\n", secretName)
			// Fall through to show list of available secrets
		} else {
			// Display the secret and return
			s.DisplaySecret(ctx, secretName, secretValue)
			return nil
		}
	}

	// List secrets for selection
	secrets, err := s.ListSecrets(ctx)
	if err != nil {
		return fmt.Errorf("error listing secrets: %v", err)
	}

	if len(secrets) == 0 {
		fmt.Printf("No secrets found in this account\n")
		return nil
	}

	// Create selection choices
	var choices []string
	for _, secret := range secrets {
		description := secret.Description
		if description == "" {
			description = "No description"
		}
		choices = append(choices, fmt.Sprintf("%s - %s", secret.Name, description))
	}

	// Interactive secret selection
	selectedIndex, err := ui.RunSelector("Select Secret:", choices)
	if err != nil {
		return fmt.Errorf("error selecting secret: %v", err)
	}
	if selectedIndex == -1 {
		return fmt.Errorf("no secret selected")
	}

	selectedSecret := secrets[selectedIndex].Name
	fmt.Printf("✓ Selected: %s\n", selectedSecret)

	// Get secret value
	secretValue, err := s.GetSecretValue(ctx, selectedSecret)
	if err != nil {
		return fmt.Errorf("error getting secret value: %v", err)
	}

	// Display the secret
	s.DisplaySecret(ctx, selectedSecret, secretValue)
	return nil
}

func (s *SecretsManager) reloadClient(ctx context.Context) error {
	cfg, err := awscconfig.LoadAWSConfigWithProfile(ctx)
	if err != nil {
		return err
	}

	s.client = secretsmanager.NewFromConfig(cfg)
	s.region = cfg.Region

	return nil
}
