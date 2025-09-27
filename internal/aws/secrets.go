package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	swaconfig "github.com/blontic/swa/internal/config"
	"github.com/blontic/swa/internal/ui"
)

type SecretsManager struct {
	client *secretsmanager.Client
	region string
}

type Secret struct {
	Name        string
	Description string
	ARN         string
}

func NewSecretsManager(ctx context.Context) (*SecretsManager, error) {
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return nil, err
	}

	return &SecretsManager{
		client: secretsmanager.NewFromConfig(cfg),
		region: cfg.Region,
	}, nil
}

func (s *SecretsManager) ListSecrets(ctx context.Context) ([]Secret, error) {
	result, err := s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		if IsAuthError(err) {
			if handleErr := HandleExpiredCredentials(ctx); handleErr != nil {
				return nil, handleErr
			}
			// Reload client with fresh credentials
			cfg, cfgErr := swaconfig.LoadSWAConfigWithProfile(ctx)
			if cfgErr != nil {
				return nil, cfgErr
			}
			s.client = secretsmanager.NewFromConfig(cfg)
			// Retry after re-authentication
			result, err = s.client.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var secrets []Secret
	for _, secret := range result.SecretList {
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
		return "", err
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

func (s *SecretsManager) RunListSecrets(ctx context.Context) error {
	// Display AWS context
	DisplayAWSContext(ctx)

	// List secrets
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
	fmt.Printf("Selected: %s\n", selectedSecret)

	// Get secret value
	secretValue, err := s.GetSecretValue(ctx, selectedSecret)
	if err != nil {
		return fmt.Errorf("error getting secret value: %v", err)
	}

	// Display the secret
	s.DisplaySecret(ctx, selectedSecret, secretValue)
	return nil
}
