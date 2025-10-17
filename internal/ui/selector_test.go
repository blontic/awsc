package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestNewSelector(t *testing.T) {
	title := "Test Title"
	choices := []string{"Option 1", "Option 2", "Option 3"}

	model := NewSelector(title, choices)

	if model.title != title {
		t.Errorf("Expected title %s, got %s", title, model.title)
	}

	if len(model.choices) != len(choices) {
		t.Errorf("Expected %d choices, got %d", len(choices), len(model.choices))
	}

	// Check filtered choices are initialized
	if len(model.filteredChoices) != len(choices) {
		t.Errorf("Expected %d filtered choices, got %d", len(choices), len(model.filteredChoices))
	}

	if model.cursor != 0 {
		t.Errorf("Expected cursor to start at 0, got %d", model.cursor)
	}

	if model.selected != -1 {
		t.Errorf("Expected selected to start at -1, got %d", model.selected)
	}

	if model.done {
		t.Error("Expected done to start as false")
	}

	if model.filter != "" {
		t.Error("Expected filter to start empty")
	}
}

func TestSelectorModel_Selected(t *testing.T) {
	model := NewSelector("Test", []string{"A", "B", "C"})

	// Initially no selection
	if model.Selected() != -1 {
		t.Errorf("Expected -1 for no selection, got %d", model.Selected())
	}

	// Set selection
	model.selected = 1
	if model.Selected() != 1 {
		t.Errorf("Expected 1 for selection, got %d", model.Selected())
	}
}

func TestSelectorModel_View(t *testing.T) {
	title := "Select Option"
	choices := []string{"Option A", "Option B"}
	model := NewSelector(title, choices)

	view := model.View()

	// Should contain title
	if !contains(view, title) {
		t.Error("View should contain title")
	}

	// Should contain choices
	for _, choice := range choices {
		if !contains(view, choice) {
			t.Errorf("View should contain choice: %s", choice)
		}
	}

	// Should contain instructions
	if !contains(view, "Press ↑/↓ to navigate") {
		t.Error("View should contain navigation instructions")
	}
}

func TestSelectorModel_ViewWhenDone(t *testing.T) {
	model := NewSelector("Test", []string{"A", "B"})
	model.done = true

	view := model.View()
	if view != "" {
		t.Error("View should be empty when done")
	}
}

func TestSelectorModel_Filtering(t *testing.T) {
	choices := []string{"Apple", "Banana", "Cherry", "Date"}
	model := NewSelector("Test", choices)

	// Test initial state - no filter
	if len(model.filteredChoices) != 4 {
		t.Errorf("Expected 4 filtered choices initially, got %d", len(model.filteredChoices))
	}

	// Test filtering
	model.filter = "a"
	model.updateFilter()

	// Should match "Apple", "Banana", "Date" (case insensitive)
	expected := 3
	if len(model.filteredChoices) != expected {
		t.Errorf("Expected %d filtered choices for 'a', got %d", expected, len(model.filteredChoices))
	}

	// Test exact match
	model.filter = "Cherry"
	model.updateFilter()
	if len(model.filteredChoices) != 1 {
		t.Errorf("Expected 1 filtered choice for 'Cherry', got %d", len(model.filteredChoices))
	}
	if model.filteredChoices[0] != "Cherry" {
		t.Errorf("Expected 'Cherry', got %s", model.filteredChoices[0])
	}

	// Test no matches
	model.filter = "xyz"
	model.updateFilter()
	if len(model.filteredChoices) != 0 {
		t.Errorf("Expected 0 filtered choices for 'xyz', got %d", len(model.filteredChoices))
	}

	// Test clearing filter
	model.filter = ""
	model.updateFilter()
	if len(model.filteredChoices) != 4 {
		t.Errorf("Expected 4 filtered choices after clearing filter, got %d", len(model.filteredChoices))
	}
}

func TestSelectorModel_FilteringWithSelectability(t *testing.T) {
	choices := []string{"Available", "Disabled", "Another"}
	selectable := []bool{true, false, true}
	model := NewSelectorWithSelectability("Test", choices, selectable)

	// Test filtering maintains selectability
	model.filter = "a"
	model.updateFilter()

	// Should match "Available", "Disabled", and "Another" (all contain 'a')
	if len(model.filteredChoices) != 3 {
		t.Errorf("Expected 3 filtered choices, got %d", len(model.filteredChoices))
	}

	// Check selectability is maintained
	if !model.filteredSelectable[0] { // "Available" should be selectable
		t.Error("Expected first filtered item to be selectable")
	}
	if model.filteredSelectable[1] { // "Disabled" should not be selectable
		t.Error("Expected second filtered item to be unselectable")
	}
	if !model.filteredSelectable[2] { // "Another" should be selectable
		t.Error("Expected third filtered item to be selectable")
	}

	// Test filtering disabled item
	model.filter = "Disabled"
	model.updateFilter()
	if len(model.filteredChoices) != 1 {
		t.Errorf("Expected 1 filtered choice, got %d", len(model.filteredChoices))
	}
	if model.filteredSelectable[0] { // "Disabled" should not be selectable
		t.Error("Expected filtered disabled item to remain unselectable")
	}
}

func TestSelectorModel_FilterIndices(t *testing.T) {
	choices := []string{"First", "Second", "Third"}
	model := NewSelector("Test", choices)

	// Filter to get only "First" and "Third"
	model.filter = "ir"
	model.updateFilter()

	// Should have 2 matches: "First" (index 0) and "Third" (index 2)
	if len(model.filterIndices) != 2 {
		t.Errorf("Expected 2 filter indices, got %d", len(model.filterIndices))
	}
	if model.filterIndices[0] != 0 {
		t.Errorf("Expected first filter index to be 0, got %d", model.filterIndices[0])
	}
	if model.filterIndices[1] != 2 {
		t.Errorf("Expected second filter index to be 2, got %d", model.filterIndices[1])
	}
}

func TestSelectorModel_ViewWithFilter(t *testing.T) {
	model := NewSelector("Test", []string{"Apple", "Banana"})
	model.filter = "app"
	model.updateFilter()

	view := model.View()

	// Should show filter
	if !contains(view, "Filter: app") {
		t.Error("View should show current filter")
	}

	// Should show filtered results
	if !contains(view, "Apple") {
		t.Error("View should contain filtered choice")
	}
	if contains(view, "Banana") {
		t.Error("View should not contain unfiltered choice")
	}
}

func TestSelectorModel_ViewNoMatches(t *testing.T) {
	model := NewSelector("Test", []string{"Apple", "Banana"})
	model.filter = "xyz"
	model.updateFilter()

	view := model.View()

	// Should show no matches message
	if !contains(view, "No matches found") {
		t.Error("View should show no matches message")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetAWSContext_WithEnvVar(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Create a session file
	sessionsDir := filepath.Join(tempDir, ".awsc", "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	sessionContent := `{
  "profile_name": "awsc-test-account",
  "account_id": "123456789012",
  "account_name": "test-account",
  "role_name": "TestRole"
}`
	sessionPath := filepath.Join(sessionsDir, "session-12345.json")
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0600); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Set AWSC_PROFILE environment variable
	originalProfile := os.Getenv("AWSC_PROFILE")
	os.Setenv("AWSC_PROFILE", "awsc-test-account")
	defer func() {
		if originalProfile != "" {
			os.Setenv("AWSC_PROFILE", originalProfile)
		} else {
			os.Unsetenv("AWSC_PROFILE")
		}
	}()

	// Set up viper config
	viper.Set("default_region", "us-west-2")
	defer viper.Reset()

	// Call getAWSContext
	ctx := getAWSContext()

	// Verify results
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}

	if ctx.Account != "test-account" {
		t.Errorf("Expected account 'test-account', got '%s'", ctx.Account)
	}

	if ctx.Role != "TestRole" {
		t.Errorf("Expected role 'TestRole', got '%s'", ctx.Role)
	}

	if ctx.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got '%s'", ctx.Region)
	}
}

func TestGetAWSContext_NoEnvVar(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Ensure AWSC_PROFILE is not set
	originalProfile := os.Getenv("AWSC_PROFILE")
	os.Unsetenv("AWSC_PROFILE")
	defer func() {
		if originalProfile != "" {
			os.Setenv("AWSC_PROFILE", originalProfile)
		}
	}()

	// Create a session file for current PPID
	ppid := os.Getppid()
	sessionsDir := filepath.Join(tempDir, ".awsc", "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		t.Fatalf("Failed to create sessions directory: %v", err)
	}

	sessionContent := `{
  "profile_name": "awsc-ppid-account",
  "account_id": "999888777666",
  "account_name": "ppid-account",
  "role_name": "PPIDRole"
}`
	sessionPath := filepath.Join(sessionsDir, fmt.Sprintf("session-%d.json", ppid))
	if err := os.WriteFile(sessionPath, []byte(sessionContent), 0600); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Set up viper config
	viper.Set("default_region", "ap-southeast-2")
	defer viper.Reset()

	// Call getAWSContext
	ctx := getAWSContext()

	// Verify results
	if ctx == nil {
		t.Fatal("Expected context, got nil")
	}

	if ctx.Account != "ppid-account" {
		t.Errorf("Expected account 'ppid-account', got '%s'", ctx.Account)
	}

	if ctx.Role != "PPIDRole" {
		t.Errorf("Expected role 'PPIDRole', got '%s'", ctx.Role)
	}

	if ctx.Region != "ap-southeast-2" {
		t.Errorf("Expected region 'ap-southeast-2', got '%s'", ctx.Region)
	}
}

func TestGetAWSContext_NoSession(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Override home directory for test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	// Ensure AWSC_PROFILE is not set
	originalProfile := os.Getenv("AWSC_PROFILE")
	os.Unsetenv("AWSC_PROFILE")
	defer func() {
		if originalProfile != "" {
			os.Setenv("AWSC_PROFILE", originalProfile)
		}
	}()

	// Don't create any session files

	// Call getAWSContext
	ctx := getAWSContext()

	// Should return nil when no session exists
	if ctx != nil {
		t.Error("Expected nil context when no session exists")
	}
}
