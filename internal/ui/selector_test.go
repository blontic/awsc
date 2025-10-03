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
