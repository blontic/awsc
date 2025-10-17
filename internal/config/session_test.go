package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndGetSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Test data
	ppid := 12345
	profileName := "awsc-test-account"
	accountID := "123456789012"
	accountName := "test-account"
	roleName := "TestRole"

	// Save session
	err := SaveSession(ppid, profileName, accountID, accountName, roleName)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Verify session file was created
	sessionFile := filepath.Join(tempDir, ".awsc", "sessions", "session-12345.json")
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatal("Session file was not created")
	}

	// Verify file permissions
	info, err := os.Stat(sessionFile)
	if err != nil {
		t.Fatalf("Failed to stat session file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", info.Mode().Perm())
	}

	// Read session back (we can't use GetCurrentSession since PPID won't match)
	// So we'll read the file directly
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatalf("Failed to read session file: %v", err)
	}

	// Verify content contains expected values
	content := string(data)
	if !contains(content, profileName) {
		t.Error("Session file does not contain profile name")
	}
	if !contains(content, accountID) {
		t.Error("Session file does not contain account ID")
	}
	if !contains(content, accountName) {
		t.Error("Session file does not contain account name")
	}
	if !contains(content, roleName) {
		t.Error("Session file does not contain role name")
	}
}

func TestGetCurrentSession_NoSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Try to get session when none exists
	_, err := GetCurrentSession()
	if err == nil {
		t.Fatal("Expected error when no session exists, got nil")
	}

	if err.Error() != "no active session" {
		t.Errorf("Expected 'no active session' error, got: %v", err)
	}
}

func TestCleanupStaleSessions(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create sessions directory
	sessionsDir := filepath.Join(tempDir, ".awsc", "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	// Create a session file for a non-existent PID (999999 should not exist)
	staleSessionFile := filepath.Join(sessionsDir, "session-999999.json")
	if err := os.WriteFile(staleSessionFile, []byte(`{"profile_name":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to create stale session file: %v", err)
	}

	// Create a session file for current process (should not be deleted)
	currentPID := os.Getpid()
	currentSessionFile := filepath.Join(sessionsDir, "session-"+string(rune(currentPID))+".json")
	if err := os.WriteFile(currentSessionFile, []byte(`{"profile_name":"current"}`), 0600); err != nil {
		t.Fatalf("Failed to create current session file: %v", err)
	}

	// Run cleanup
	err := CleanupStaleSessions()
	if err != nil {
		t.Fatalf("CleanupStaleSessions failed: %v", err)
	}

	// Verify stale session was removed
	if _, err := os.Stat(staleSessionFile); !os.IsNotExist(err) {
		t.Error("Stale session file was not removed")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
