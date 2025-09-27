package ui

import (
	"testing"
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

	if model.cursor != 0 {
		t.Errorf("Expected cursor to start at 0, got %d", model.cursor)
	}

	if model.selected != -1 {
		t.Errorf("Expected selected to start at -1, got %d", model.selected)
	}

	if model.done {
		t.Error("Expected done to start as false")
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

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
