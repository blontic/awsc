package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// SessionInfo contains information about the current session
type SessionInfo struct {
	ProfileName string `json:"profile_name"`
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	RoleName    string `json:"role_name"`
}

// SaveSession saves session information for the given PPID
func SaveSession(ppid int, profileName, accountID, accountName, roleName string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionsDir := filepath.Join(homeDir, ".awsc", "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	session := SessionInfo{
		ProfileName: profileName,
		AccountID:   accountID,
		AccountName: accountName,
		RoleName:    roleName,
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionFile := filepath.Join(sessionsDir, fmt.Sprintf("session-%d.json", ppid))
	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// GetCurrentSession retrieves the session for the current shell (PPID)
func GetCurrentSession() (*SessionInfo, error) {
	ppid := os.Getppid()
	if ppid <= 0 {
		return nil, fmt.Errorf("no active session")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no active session")
	}

	sessionFile := filepath.Join(homeDir, ".awsc", "sessions", fmt.Sprintf("session-%d.json", ppid))
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no active session")
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session SessionInfo
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	return &session, nil
}

// CleanupStaleSessions removes session files for processes that no longer exist
func CleanupStaleSessions() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Best effort
	}

	sessionsDir := filepath.Join(homeDir, ".awsc", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil // Best effort
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		var ppid int
		if _, err := fmt.Sscanf(entry.Name(), "session-%d.json", &ppid); err != nil {
			continue
		}

		if !processExists(ppid) {
			sessionFile := filepath.Join(sessionsDir, entry.Name())
			_ = os.Remove(sessionFile) // Ignore errors
		}
	}

	return nil
}

// processExists checks if a process with the given PID exists
func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists without actually sending a signal
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
