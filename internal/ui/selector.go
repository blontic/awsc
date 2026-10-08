package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	awscconfig "github.com/blontic/awsc/internal/config"
)

type SelectorModel struct {
	choices            []string
	selectable         []bool
	filteredChoices    []string
	filteredSelectable []bool
	filterIndices      []int
	filter             string
	cursor             int
	selected           int
	title              string
	done               bool
	awsContext         *AWSContext
	width, height      int // terminal size; 0 until known
}

type AWSContext struct {
	Org     string
	Account string
	Role    string
	Region  string
}

func NewSelector(title string, choices []string) SelectorModel {
	selectable := make([]bool, len(choices))
	for i := range selectable {
		selectable[i] = true
	}
	m := SelectorModel{
		choices:    choices,
		selectable: selectable,
		title:      title,
		selected:   -1,
		awsContext: getAWSContext(),
	}
	m.updateFilter()
	return m
}

func NewSelectorWithSelectability(title string, choices []string, selectable []bool) SelectorModel {
	m := SelectorModel{
		choices:    choices,
		selectable: selectable,
		title:      title,
		selected:   -1,
		awsContext: getAWSContext(),
	}
	m.updateFilter()
	// Find first selectable item in filtered results
	for i, sel := range m.filteredSelectable {
		if sel {
			m.cursor = i
			break
		}
	}
	return m
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		resized := m.width != 0 && (msg.Width != m.width || msg.Height != m.height)
		m.width, m.height = msg.Width, msg.Height
		if resized {
			// The terminal re-wraps lines drawn at the old width, so an inline
			// redraw would leave stale lines behind; redraw from a clear screen.
			return m, tea.ClearScreen
		}
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			for i := m.cursor - 1; i >= 0; i-- {
				if m.filteredSelectable[i] {
					m.cursor = i
					break
				}
			}
		case "down", "j":
			for i := m.cursor + 1; i < len(m.filteredChoices); i++ {
				if m.filteredSelectable[i] {
					m.cursor = i
					break
				}
			}
		case "enter", "space":
			if len(m.filteredChoices) > 0 && m.cursor < len(m.filteredSelectable) && m.filteredSelectable[m.cursor] {
				m.selected = m.filterIndices[m.cursor]
				m.done = true
				return m, tea.Quit
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.updateFilter()
				m.resetCursor()
			}
		case "esc":
			// Esc clears the filter, or quits if there is none.
			if m.filter == "" {
				return m, tea.Quit
			}
			m.filter = ""
			m.updateFilter()
			m.resetCursor()
		default:
			// Handle typing for filtering
			if len(msg.Text) == 1 && msg.Text > " " && msg.Text <= "~" {
				m.filter += msg.Text
				m.updateFilter()
				m.resetCursor()
			}
		}
	}
	return m, nil
}

// View draws the selector inline below the existing output; it clears itself
// when a choice is made.
func (m SelectorModel) View() tea.View {
	return tea.NewView(m.render())
}

// render returns the selector's screen content.
func (m SelectorModel) render() string {
	if m.done {
		return ""
	}

	s := strings.Builder{}

	// AWS Context Header
	if header := m.awsContext.header(); header != "" {
		s.WriteString(header)
		s.WriteString("\n\n")
	}

	s.WriteString(fmt.Sprintf("%s\n", m.title))
	if m.filter != "" {
		s.WriteString(fmt.Sprintf("Filter: %s\n\n", m.filter))
	} else {
		s.WriteString("\n")
	}

	if len(m.filteredChoices) == 0 {
		s.WriteString("No matches found\n")
	} else {
		start, end := m.visibleRange(m.screenLines(s.String()))
		if start > 0 {
			s.WriteString(fmt.Sprintf("  ↑ %d more\n", start))
		}
		for i := start; i < end; i++ {
			choice := m.fitWidth(m.filteredChoices[i])
			if !m.filteredSelectable[i] {
				s.WriteString(fmt.Sprintf("  %s (disabled)\n", choice))
			} else if m.cursor == i {
				boldStyle := lipgloss.NewStyle().Bold(true)
				s.WriteString(fmt.Sprintf("▶ %s\n", boldStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("  %s\n", choice))
			}
		}
		if end < len(m.filteredChoices) {
			s.WriteString(fmt.Sprintf("  ↓ %d more\n", len(m.filteredChoices)-end))
		}
	}

	s.WriteString(selectorFooter)
	return s.String()
}

// fitWidth shortens a choice with "…" so its row (with the cursor prefix and
// " (disabled)" suffix) does not wrap.
func (m SelectorModel) fitWidth(choice string) string {
	maxWidth := m.width - lipgloss.Width("▶ ") - lipgloss.Width(" (disabled)")
	if m.width <= 0 || maxWidth < 1 || lipgloss.Width(choice) <= maxWidth {
		return choice
	}
	runes := []rune(choice)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > maxWidth {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

const selectorFooter = "\n↑/↓ navigate · Enter select · type to filter · Esc clear filter / quit\n"

// screenLines returns how many terminal rows text occupies, including lines
// that wrap because they are wider than the terminal.
func (m SelectorModel) screenLines(text string) int {
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	if m.width <= 0 {
		return len(lines)
	}
	total := 0
	for _, line := range lines {
		total += max(1, (lipgloss.Width(line)+m.width-1)/m.width)
	}
	return total
}

// visibleRange returns the slice of filtered choices that fits the terminal
// below usedLines of header, keeping the cursor in view, so the title and
// context header never scroll off screen.
func (m SelectorModel) visibleRange(usedLines int) (start, end int) {
	n := len(m.filteredChoices)
	const indicatorLines = 2
	rows := m.height - usedLines - m.screenLines(selectorFooter) - indicatorLines
	if m.height == 0 || n <= rows+indicatorLines {
		return 0, n
	}
	if rows < 1 {
		rows = 1
	}
	start = m.cursor - rows/2
	start = max(0, min(start, n-rows))
	return start, start + rows
}

func (m *SelectorModel) updateFilter() {
	m.filteredChoices = nil
	m.filteredSelectable = nil
	m.filterIndices = nil

	filterLower := strings.ToLower(m.filter)
	for i, choice := range m.choices {
		if m.filter == "" || strings.Contains(strings.ToLower(choice), filterLower) {
			m.filteredChoices = append(m.filteredChoices, choice)
			m.filteredSelectable = append(m.filteredSelectable, m.selectable[i])
			m.filterIndices = append(m.filterIndices, i)
		}
	}
}

func (m *SelectorModel) resetCursor() {
	m.cursor = 0
	// Find first selectable item in filtered results
	for i, sel := range m.filteredSelectable {
		if sel {
			m.cursor = i
			break
		}
	}
}

func (m SelectorModel) Selected() int {
	return m.selected
}

// header renders the context line ("Org: x | Account: y | ..."), showing only
// the fields that are set.
func (c *AWSContext) header() string {
	if c == nil {
		return ""
	}
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	var parts []string
	for _, f := range []struct{ label, value string }{
		{"Org", c.Org}, {"Account", c.Account}, {"Role", c.Role}, {"Region", c.Region},
	} {
		if f.value != "" {
			parts = append(parts, f.label+": "+valueStyle.Render(f.value))
		}
	}
	return strings.Join(parts, " | ")
}

// RunSelector shows a selector whose header is the terminal's current AWS
// context.
func RunSelector(title string, choices []string) (int, error) {
	return RunSelectorWithContext(title, choices, getAWSContext())
}

// RunSelectorWithContext shows a selector with the given header context, for
// choices made before the terminal's context exists (e.g. picking an account
// and role during login).
func RunSelectorWithContext(title string, choices []string, ctx *AWSContext) (int, error) {
	// Try interactive mode first
	model := NewSelector(title, choices)
	model.awsContext = ctx
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		// Fallback to simple numbered selection
		return runSimpleSelector(title, choices)
	}

	if m, ok := finalModel.(SelectorModel); ok {
		return m.Selected(), nil
	}

	return -1, fmt.Errorf("unexpected model type")
}

func RunSelectorWithSelectability(title string, choices []string, selectable []bool) (int, error) {
	// Try interactive mode first
	model := NewSelectorWithSelectability(title, choices, selectable)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		// Fallback to simple numbered selection (only show selectable items)
		return runSimpleSelectorWithSelectability(title, choices, selectable)
	}

	if m, ok := finalModel.(SelectorModel); ok {
		return m.Selected(), nil
	}

	return -1, fmt.Errorf("unexpected model type")
}

func runSimpleSelector(title string, choices []string) (int, error) {
	fmt.Fprintln(os.Stderr, title)
	for i, choice := range choices {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, choice)
	}

	fmt.Fprint(os.Stderr, "Select (number): ")
	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return -1, err
	}

	if choice < 1 || choice > len(choices) {
		return -1, fmt.Errorf("invalid selection")
	}

	return choice - 1, nil
}

func runSimpleSelectorWithSelectability(title string, choices []string, selectable []bool) (int, error) {
	fmt.Fprintln(os.Stderr, title)
	fmt.Fprintln(os.Stderr, "(Filtering not available in non-interactive mode)")
	selectableChoices := make([]string, 0)
	indexMap := make([]int, 0)

	for i, choice := range choices {
		if selectable[i] {
			selectableChoices = append(selectableChoices, choice)
			indexMap = append(indexMap, i)
		} else {
			fmt.Fprintf(os.Stderr, "   %s (unavailable)\n", choice)
		}
	}

	for i, choice := range selectableChoices {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, choice)
	}

	fmt.Fprint(os.Stderr, "Select (number): ")
	var choice int
	if _, err := fmt.Scanln(&choice); err != nil {
		return -1, err
	}

	if choice < 1 || choice > len(selectableChoices) {
		return -1, fmt.Errorf("invalid selection")
	}

	return indexMap[choice-1], nil
}

func getAWSContext() *AWSContext {
	// Check AWSC_PROFILE environment variable first (same priority as LoadAWSConfigWithProfile)
	envProfile := os.Getenv("AWSC_PROFILE")

	var accountName, roleName string

	if envProfile != "" {
		// Parse profile name to extract account name
		// Profile format: awsc-{accountName}
		if strings.HasPrefix(envProfile, "awsc-") {
			accountName = strings.TrimPrefix(envProfile, "awsc-")
		} else {
			accountName = envProfile
		}

		// Try to get role from session files by matching profile name
		// This is best effort - if not found, we'll show "unknown"
		roleName = "unknown"
		homeDir, err := os.UserHomeDir()
		if err == nil {
			sessionsDir := filepath.Join(homeDir, ".awsc", "sessions")
			files, err := os.ReadDir(sessionsDir)
			if err == nil {
				for _, file := range files {
					if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
						sessionPath := filepath.Join(sessionsDir, file.Name())
						data, err := os.ReadFile(sessionPath)
						if err == nil {
							var session awscconfig.SessionInfo
							if err := json.Unmarshal(data, &session); err == nil {
								if session.ProfileName == envProfile {
									accountName = session.AccountName
									roleName = session.RoleName
									break
								}
							}
						}
					}
				}
			}
		}
	} else {
		// Fallback to PPID session
		session, err := awscconfig.GetCurrentSession()
		if err != nil {
			return nil
		}
		accountName = session.AccountName
		roleName = session.RoleName
	}

	// Get region from config
	region := awscconfig.Active().DefaultRegion
	if region == "" {
		region = awscconfig.Active().SSORegion
	}
	if region == "" {
		region = "default"
	}

	return &AWSContext{
		Org:     awscconfig.Active().Org,
		Account: accountName,
		Role:    roleName,
		Region:  region,
	}
}
