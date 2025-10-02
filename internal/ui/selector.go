package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	swaconfig "github.com/blontic/swa/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
}

type AWSContext struct {
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
		// Handle window resize - no action needed, just return
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
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
		case "enter", " ":
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
			m.filter = ""
			m.updateFilter()
			m.resetCursor()
		default:
			// Handle typing for filtering
			if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				m.filter += msg.String()
				m.updateFilter()
				m.resetCursor()
			}
		}
	}
	return m, nil
}

func (m SelectorModel) View() string {
	if m.done {
		return ""
	}

	s := strings.Builder{}

	// AWS Context Header
	if m.awsContext != nil {
		headerText := fmt.Sprintf("Account: %s | Role: %s | Region: %s",
			m.awsContext.Account,
			m.awsContext.Role,
			m.awsContext.Region)

		s.WriteString(headerText)
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
		for i, choice := range m.filteredChoices {
			if !m.filteredSelectable[i] {
				s.WriteString(fmt.Sprintf("  %s (disabled)\n", choice))
			} else if m.cursor == i {
				boldStyle := lipgloss.NewStyle().Bold(true)
				s.WriteString(fmt.Sprintf("▶ %s\n", boldStyle.Render(choice)))
			} else {
				s.WriteString(fmt.Sprintf("  %s\n", choice))
			}
		}
	}

	s.WriteString("\nPress ↑/↓ to navigate, Enter to select, type to filter, ESC to clear filter, q to quit\n")
	return s.String()
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

func RunSelector(title string, choices []string) (int, error) {
	// Try interactive mode first
	model := NewSelector(title, choices)
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
	fmt.Println(title)
	for i, choice := range choices {
		fmt.Printf("%d. %s\n", i+1, choice)
	}

	fmt.Print("Select (number): ")
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
	fmt.Println(title)
	fmt.Println("(Filtering not available in non-interactive mode)")
	selectableChoices := make([]string, 0)
	indexMap := make([]int, 0)

	for i, choice := range choices {
		if selectable[i] {
			selectableChoices = append(selectableChoices, choice)
			indexMap = append(indexMap, i)
		} else {
			fmt.Printf("   %s (unavailable)\n", choice)
		}
	}

	for i, choice := range selectableChoices {
		fmt.Printf("%d. %s\n", i+1, choice)
	}

	fmt.Print("Select (number): ")
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
	ctx := context.Background()
	cfg, err := swaconfig.LoadSWAConfigWithProfile(ctx)
	if err != nil {
		return nil
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil
	}

	// Parse ARN to get account and role
	accountId := *identity.Account
	role := "unknown"

	if identity.Arn != nil {
		// ARN format: arn:aws:sts::123456789012:assumed-role/RoleName/SessionName
		parts := strings.Split(*identity.Arn, "/")
		if len(parts) >= 2 && strings.Contains(*identity.Arn, "assumed-role") {
			role = parts[1]
		}
	}

	// Get account name from cache, fallback to account ID
	account := swaconfig.GetAccountName(accountId)

	region := cfg.Region
	if region == "" {
		region = "default"
	}

	return &AWSContext{
		Account: account,
		Role:    role,
		Region:  region,
	}
}
