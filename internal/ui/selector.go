package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectorModel struct {
	choices  []string
	cursor   int
	selected int
	title    string
	done     bool
}

func NewSelector(title string, choices []string) SelectorModel {
	return SelectorModel{
		choices:  choices,
		title:    title,
		selected: -1,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectorModel) View() string {
	if m.done {
		return ""
	}

	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("%s\n\n", m.title))

	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		s.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
	}

	s.WriteString("\nPress ↑/↓ to navigate, Enter to select, q to quit\n")
	return s.String()
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
