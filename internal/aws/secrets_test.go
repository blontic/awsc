package aws

import (
	"context"
	"testing"
)

func TestSecretsManager_DisplaySecret_JSON(t *testing.T) {
	sm := &SecretsManager{
		region: "us-east-1",
	}
	
	// Test JSON formatting
	secretName := "test-secret"
	secretValue := `{"username":"admin","password":"secret123"}`
	
	// This would normally print to stdout, but we can't easily capture that in tests
	// In a real scenario, we'd refactor DisplaySecret to return formatted string
	sm.DisplaySecret(context.Background(), secretName, secretValue)
	
	// Test passes if no panic occurs
}

func TestSecretsManager_DisplaySecret_PlainText(t *testing.T) {
	sm := &SecretsManager{
		region: "us-east-1",
	}
	
	// Test plain text display
	secretName := "test-secret"
	secretValue := "plain-text-secret"
	
	// This would normally print to stdout
	sm.DisplaySecret(context.Background(), secretName, secretValue)
	
	// Test passes if no panic occurs
}