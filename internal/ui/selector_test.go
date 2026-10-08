package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	awscconfig "github.com/blontic/awsc/internal/config"
	"github.com/charmbracelet/x/ansi"
	"io"
	"os"
	"path/filepath"
	"strings"
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

	view := model.render()

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
	if !contains(view, strings.TrimSpace(selectorFooter)) {
		t.Error("View should contain navigation instructions")
	}
}

func TestSelectorModel_ViewWhenDone(t *testing.T) {
	model := NewSelector("Test", []string{"A", "B"})
	model.done = true

	view := model.render()
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

	view := model.render()

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

	view := model.render()

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

	// Set up active settings
	awscconfig.SetActive(awscconfig.Settings{DefaultRegion: "us-west-2"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

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

	// Set up active settings
	awscconfig.SetActive(awscconfig.Settings{Org: "woodside", DefaultRegion: "ap-southeast-2"})
	defer func() { awscconfig.SetActive(awscconfig.Settings{}) }()

	// Call getAWSContext
	ctx := getAWSContext()
	if ctx != nil && ctx.Org != "woodside" {
		t.Errorf("Expected org 'woodside' from active settings, got '%s'", ctx.Org)
	}

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

// The selector must never render more rows than the terminal has: otherwise
// the terminal scrolls and the title (e.g. "Select AWS Account:") and context
// header disappear off the top.

func manyChoices(n int) []string {
	choices := make([]string, n)
	for i := range choices {
		choices[i] = fmt.Sprintf("wpl-wrk-account-%02d (1234567890%02d)", i, i)
	}
	return choices
}

func sized(t *testing.T, m SelectorModel, width, height int) SelectorModel {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(SelectorModel)
}

func assertFits(t *testing.T, m SelectorModel, title string) {
	t.Helper()
	view := m.render()
	if rows := m.screenLines(view); rows > m.height {
		t.Errorf("view uses %d terminal rows, terminal has %d:\n%s", rows, m.height, view)
	}
	if !strings.Contains(view, title) {
		t.Errorf("title %q not visible:\n%s", title, view)
	}
	if len(m.filteredChoices) > 0 && !strings.Contains(view, "▶ ") {
		t.Errorf("cursor row not visible:\n%s", view)
	}
}

func TestSelectorModel_ViewFitsTerminal(t *testing.T) {
	const title = "Select AWS Account:"
	m := sized(t, NewSelector(title, manyChoices(50)), 80, 20)
	m.awsContext = nil

	for _, cursor := range []int{0, 1, 10, 25, 48, 49} {
		m.cursor = cursor
		assertFits(t, m, title)
		if !strings.Contains(m.render(), m.filteredChoices[cursor]) {
			t.Errorf("cursor %d: selected item not visible", cursor)
		}
	}

	m.cursor = 25
	view := m.render()
	if !strings.Contains(view, "↑") || !strings.Contains(view, "↓") {
		t.Errorf("expected scroll indicators in the middle of the list:\n%s", view)
	}
}

func TestSelectorModel_ViewFitsTerminal_WithContextHeader(t *testing.T) {
	const title = "Select AWS Account:"
	m := NewSelector(title, manyChoices(32))
	m.awsContext = &AWSContext{Org: "woodside", Account: "wpl-wrk-rbm-np", Role: "cops-permset-teamadmin", Region: "ap-southeast-2"}

	// Wide terminal: header on one line.
	m = sized(t, m, 200, 30)
	assertFits(t, m, title)

	// Narrow terminal: the long header wraps over several rows.
	m = sized(t, m, 40, 30)
	assertFits(t, m, title)
	if !strings.Contains(m.render(), "woodside") {
		t.Error("context header must stay visible")
	}
}

func TestSelectorModel_ViewFitsTerminal_WhileNavigatingAndFiltering(t *testing.T) {
	const title = "Select AWS Account:"
	m := sized(t, NewSelector(title, manyChoices(40)), 80, 15)
	m.awsContext = nil

	press := func(key string) {
		updated, _ := m.Update(tea.KeyPressMsg{Code: []rune(key)[0], Text: key})
		m = updated.(SelectorModel)
	}
	down := func() {
		updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = updated.(SelectorModel)
	}

	for i := 0; i < 39; i++ {
		down()
		assertFits(t, m, title)
	}
	press("1")
	assertFits(t, m, title)
	if !strings.Contains(m.render(), "Filter: 1") {
		t.Error("filter line should be visible")
	}
}

func TestSelectorModel_ShowsAllWhenItFits(t *testing.T) {
	m := sized(t, NewSelector("Pick:", manyChoices(5)), 80, 40)
	m.awsContext = nil
	view := m.render()
	if strings.Contains(view, "more") {
		t.Errorf("short list should not scroll:\n%s", view)
	}
	for _, c := range manyChoices(5) {
		if !strings.Contains(view, c) {
			t.Errorf("missing %q", c)
		}
	}

	// Height unknown (no WindowSizeMsg yet): show everything.
	m = NewSelector("Pick:", manyChoices(50))
	m.awsContext = nil
	if strings.Contains(m.render(), "more") {
		t.Error("all choices should show when the terminal size is unknown")
	}
}

func TestSelectorModel_LongChoicesDoNotWrap(t *testing.T) {
	long := strings.Repeat("very-long-account-name-", 5)
	choices := []string{long + "a", long + "b", "short"}
	m := sized(t, NewSelector("Pick:", choices), 40, 20)
	m.awsContext = nil

	for _, line := range strings.Split(strings.TrimSuffix(m.render(), "\n"), "\n") {
		if line == strings.TrimSpace(selectorFooter) {
			continue // the footer may wrap; visibleRange accounts for it
		}
		if w := lipgloss.Width(line); w > 40 {
			t.Errorf("line wider than terminal (%d > 40): %q", w, line)
		}
	}
	if !strings.Contains(m.render(), "…") || !strings.Contains(m.render(), "short") {
		t.Errorf("expected long choices shortened and short ones intact:\n%s", m.render())
	}
}

func press(t *testing.T, m SelectorModel, keys ...tea.KeyPressMsg) (SelectorModel, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, k := range keys {
		var updated tea.Model
		updated, cmd = m.Update(k)
		m = updated.(SelectorModel)
	}
	return m, cmd
}

func TestSelectorModel_Keys(t *testing.T) {
	down := tea.KeyPressMsg{Code: tea.KeyDown}
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	space := tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	letter := func(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

	t.Run("enter selects", func(t *testing.T) {
		m, cmd := press(t, NewSelector("Pick:", []string{"a", "b", "c"}), down, enter)
		if m.Selected() != 1 || cmd == nil {
			t.Errorf("selected = %d, want 1 and a quit command", m.Selected())
		}
	})

	t.Run("space selects", func(t *testing.T) {
		m, _ := press(t, NewSelector("Pick:", []string{"a", "b"}), down, space)
		if m.Selected() != 1 {
			t.Errorf("selected = %d, want 1", m.Selected())
		}
	})

	t.Run("typing filters and selects the original index", func(t *testing.T) {
		m, _ := press(t, NewSelector("Pick:", []string{"prod", "dev", "test"}), letter('d'), letter('e'), letter('v'), enter)
		if m.Selected() != 1 {
			t.Errorf("selected = %d, want 1 (dev)", m.Selected())
		}
	})

	esc := tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}

	t.Run("q is typed into the filter", func(t *testing.T) {
		m, cmd := press(t, NewSelector("Pick:", []string{"prod", "sqs-queue"}), letter('q'))
		if cmd != nil || m.filter != "q" || len(m.filteredChoices) != 1 {
			t.Errorf("filter = %q, choices = %v, cmd = %v", m.filter, m.filteredChoices, cmd)
		}
		if m, _ = press(t, m, enter); m.Selected() != 1 {
			t.Errorf("selected = %d, want 1 (sqs-queue)", m.Selected())
		}
	})

	t.Run("esc clears the filter, then quits", func(t *testing.T) {
		m, cmd := press(t, NewSelector("Pick:", []string{"a", "b"}), letter('b'), esc)
		if cmd != nil || m.filter != "" || len(m.filteredChoices) != 2 {
			t.Errorf("first esc should clear the filter: filter=%q cmd=%v", m.filter, cmd)
		}
		if m, cmd = press(t, m, esc); cmd == nil || m.Selected() != -1 {
			t.Errorf("second esc should quit without selecting: selected=%d", m.Selected())
		}
	})

	t.Run("ctrl+c quits without selecting", func(t *testing.T) {
		m, cmd := press(t, NewSelector("Pick:", []string{"a"}), letter('a'), ctrlC)
		if cmd == nil || m.Selected() != -1 {
			t.Errorf("expected quit without selection, selected = %d", m.Selected())
		}
	})
}

func TestSelectorModel_ViewIsInline(t *testing.T) {
	m := NewSelector("Pick:", []string{"a"})
	if m.View().AltScreen {
		t.Error("selector should render inline, not full screen")
	}
	if m.done = true; m.render() != "" {
		t.Error("selector should clear its content when done")
	}
}

// Runs the real Bubble Tea program end to end with scripted input.
func TestRunSelectorProgram(t *testing.T) {
	m := NewSelector("Pick:", []string{"a", "b", "c"})
	m.awsContext = nil
	p := tea.NewProgram(m,
		tea.WithInput(strings.NewReader("\x1b[B\x1b[B\r")), // down, down, enter
		tea.WithOutput(io.Discard),
		tea.WithWindowSize(80, 24),
		tea.WithoutSignals(),
	)
	final, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	if got := final.(SelectorModel).Selected(); got != 2 {
		t.Errorf("selected = %d, want 2", got)
	}
}

func TestAWSContextHeader(t *testing.T) {
	cases := []struct {
		ctx  *AWSContext
		want string
	}{
		{nil, ""},
		{&AWSContext{}, ""},
		{&AWSContext{Org: "o", Region: "r"}, "Org: o | Region: r"},
		{&AWSContext{Org: "o", Account: "new-acct", Region: "r"}, "Org: o | Account: new-acct | Region: r"},
		{&AWSContext{Org: "o", Account: "a", Role: "x", Region: "r"}, "Org: o | Account: a | Role: x | Region: r"},
	}
	for _, c := range cases {
		if got := ansi.Strip(c.ctx.header()); got != c.want {
			t.Errorf("header(%+v) = %q, want %q", c.ctx, got, c.want)
		}
	}
}

// Login pickers must show the account being chosen, not the terminal's
// previous account.
func TestSelector_LoginContextReplacesSessionContext(t *testing.T) {
	m := NewSelector("Select role for new-acct:", []string{"Admin"})
	m.awsContext = &AWSContext{Org: "o", Account: "new-acct", Region: "r"}
	view := ansi.Strip(m.render())
	if !strings.Contains(view, "Account: new-acct") || strings.Contains(view, "Role:") {
		t.Errorf("unexpected header:\n%s", view)
	}
}

func TestSelectorModel_ResizeClearsScreen(t *testing.T) {
	isClear := func(cmd tea.Cmd) bool {
		return cmd != nil && fmt.Sprintf("%T", cmd()) == fmt.Sprintf("%T", tea.ClearScreen())
	}
	m := NewSelector("Pick:", []string{"a"})

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Error("the initial size report must not clear the screen")
	}
	m = updated.(SelectorModel)

	updated, cmd = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if cmd != nil {
		t.Error("an unchanged size must not clear the screen")
	}
	m = updated.(SelectorModel)

	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 30}, {Width: 60, Height: 20}, {Width: 120, Height: 20}} {
		updated, cmd = m.Update(size)
		if !isClear(cmd) {
			t.Errorf("resize to %dx%d should clear the screen", size.Width, size.Height)
		}
		m = updated.(SelectorModel)
		if m.width != size.Width || m.height != size.Height {
			t.Errorf("size not recorded: %dx%d", m.width, m.height)
		}
	}
}

func TestGetAWSContext_ParsesRoleFromEnvProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AWSC_PROFILE", "awsc-prod/ReadOnly")
	awscconfig.SetActive(awscconfig.Settings{DefaultRegion: "us-west-2"})
	defer awscconfig.SetActive(awscconfig.Settings{})

	ctx := getAWSContext()
	if ctx == nil || ctx.Account != "prod" || ctx.Role != "ReadOnly" {
		t.Errorf("expected account prod and role ReadOnly, got %+v", ctx)
	}
}
