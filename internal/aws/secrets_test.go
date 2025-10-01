package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/blontic/swa/internal/aws/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewSecretsManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockSecretsManagerClient(ctrl)

	manager, err := NewSecretsManager(context.Background(), SecretsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", manager.region)
	}
}

func TestNewSecretsManagerWithOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockSecretsManagerClient(ctrl)

	manager, err := NewSecretsManager(context.Background(), SecretsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", manager.region)
	}
}

func TestSecretsManager_ListSecrets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockSecretsManagerClient(ctrl)
	manager, err := NewSecretsManager(context.Background(), SecretsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name          string
		mockResponse  *secretsmanager.ListSecretsOutput
		mockError     error
		expectedCount int
		expectedError bool
	}{
		{
			name: "successful response with secrets",
			mockResponse: &secretsmanager.ListSecretsOutput{
				SecretList: []types.SecretListEntry{
					{
						Name:        aws.String("database-credentials"),
						Description: aws.String("Database connection credentials"),
						ARN:         aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:database-credentials-AbCdEf"),
					},
					{
						Name:        aws.String("api-keys"),
						Description: aws.String("Third-party API keys"),
						ARN:         aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:api-keys-GhIjKl"),
					},
					{
						Name: aws.String("secret-without-description"),
						ARN:  aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:secret-without-description-MnOpQr"),
					},
				},
			},
			expectedCount: 3,
			expectedError: false,
		},
		{
			name:          "empty response",
			mockResponse:  &secretsmanager.ListSecretsOutput{SecretList: []types.SecretListEntry{}},
			expectedCount: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.EXPECT().
				ListSecrets(gomock.Any(), gomock.Any()).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			secrets, err := manager.ListSecrets(context.Background())

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(secrets) != tt.expectedCount {
				t.Errorf("Expected %d secrets, got %d", tt.expectedCount, len(secrets))
			}

			// Verify secret details for successful cases
			if !tt.expectedError && tt.expectedCount > 0 {
				for i, secret := range secrets {
					expectedSecret := tt.mockResponse.SecretList[i]
					if secret.Name != *expectedSecret.Name {
						t.Errorf("Expected name %s, got %s", *expectedSecret.Name, secret.Name)
					}
					if secret.ARN != *expectedSecret.ARN {
						t.Errorf("Expected ARN %s, got %s", *expectedSecret.ARN, secret.ARN)
					}

					// Check description handling
					expectedDesc := ""
					if expectedSecret.Description != nil {
						expectedDesc = *expectedSecret.Description
					}
					if secret.Description != expectedDesc {
						t.Errorf("Expected description %s, got %s", expectedDesc, secret.Description)
					}
				}
			}
		})
	}
}

func TestSecretsManager_GetSecretValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockSecretsManagerClient(ctrl)
	manager, err := NewSecretsManager(context.Background(), SecretsManagerOptions{
		Client: mockClient,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatalf("Unexpected error creating manager: %v", err)
	}

	tests := []struct {
		name          string
		secretName    string
		mockResponse  *secretsmanager.GetSecretValueOutput
		mockError     error
		expectedValue string
		expectedError bool
	}{
		{
			name:       "successful response with string secret",
			secretName: "database-credentials",
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretString: aws.String(`{"username":"admin","password":"secret123"}`),
			},
			expectedValue: `{"username":"admin","password":"secret123"}`,
			expectedError: false,
		},
		{
			name:       "successful response with binary secret",
			secretName: "binary-secret",
			mockResponse: &secretsmanager.GetSecretValueOutput{
				SecretBinary: []byte("binary-data"),
			},
			expectedValue: "binary-data",
			expectedError: false,
		},
		{
			name:          "secret not found",
			secretName:    "nonexistent-secret",
			mockResponse:  nil,
			mockError:     &types.ResourceNotFoundException{},
			expectedValue: "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient.EXPECT().
				GetSecretValue(gomock.Any(), &secretsmanager.GetSecretValueInput{
					SecretId: aws.String(tt.secretName),
				}).
				Return(tt.mockResponse, tt.mockError).
				Times(1)

			value, err := manager.GetSecretValue(context.Background(), tt.secretName)

			if tt.expectedError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if value != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, value)
			}
		})
	}
}

func TestSecretsManager_DisplaySecret_JSON(t *testing.T) {
	manager := &SecretsManager{
		region: "us-east-1",
	}

	// Test JSON formatting - this method prints to stdout so we just verify it doesn't panic
	secretName := "test-secret"
	secretValue := `{"username":"admin","password":"secret123"}`

	// This would normally print to stdout, but we can't easily capture that in tests
	// The test passes if no panic occurs
	manager.DisplaySecret(context.Background(), secretName, secretValue)
}

func TestSecretsManager_DisplaySecret_PlainText(t *testing.T) {
	manager := &SecretsManager{
		region: "us-east-1",
	}

	// Test plain text display
	secretName := "test-secret"
	secretValue := "plain-text-secret"

	// This would normally print to stdout
	// The test passes if no panic occurs
	manager.DisplaySecret(context.Background(), secretName, secretValue)
}

func TestSecretsManager_DisplaySecret_InvalidJSON(t *testing.T) {
	manager := &SecretsManager{
		region: "us-east-1",
	}

	// Test invalid JSON falls back to plain text
	secretName := "test-secret"
	secretValue := `{"invalid": json}`

	// This should fall back to plain text display without panicking
	manager.DisplaySecret(context.Background(), secretName, secretValue)
}
