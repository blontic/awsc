package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
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
	cfg, err := LoadSWAConfig(ctx)
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
		return nil, err
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