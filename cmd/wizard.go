package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type wizardModel struct {
	step            int
	cursor          int
	dbChoices       []string
	dbValues        []string
	dbSelected      string
	storageChoices  []string
	storageValues   []string
	storageSelected string
	deployChoices   []string
	deployValues    []string
	deploySelected  string
	quitting        bool
}

func initialWizardModel() wizardModel {
	return wizardModel{
		step:   0,
		cursor: 0,
		dbChoices: []string{
			"None (In-Memory Repository)",
			"MongoDB",
			"PostgreSQL",
			"MySQL",
			"DynamoDB",
		},
		dbValues: []string{"none", "mongodb", "postgres", "mysql", "dynamodb"},
		storageChoices: []string{
			"None",
			"AWS S3 (R2 compatible)",
		},
		storageValues: []string{"none", "s3"},
		deployChoices: []string{
			"Standard HTTP Server (Persistent)",
			"AWS Lambda (Serverless)",
		},
		deployValues: []string{"http", "lambda"},
	}
}

func (m wizardModel) Init() tea.Cmd {
	return nil
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			var max int
			switch m.step {
			case 0:
				max = len(m.dbChoices) - 1
			case 1:
				max = len(m.storageChoices) - 1
			case 2:
				max = len(m.deployChoices) - 1
			}
			if m.cursor < max {
				m.cursor++
			}
		case "enter":
			switch m.step {
			case 0:
				m.dbSelected = m.dbValues[m.cursor]
				m.step = 1
				m.cursor = 0
			case 1:
				m.storageSelected = m.storageValues[m.cursor]
				m.step = 2
				m.cursor = 0
			case 2:
				m.deploySelected = m.deployValues[m.cursor]
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

// Lipgloss styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF5733")).
			Background(lipgloss.Color("#1A1A1A")).
			PaddingLeft(2).
			PaddingRight(2).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00ADB5")).
			MarginBottom(1)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00ADB5")).
			Bold(true)

	activeItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EEEEEE")).
			Bold(true)

	inactiveItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Italic(true).
			MarginTop(1)

	summaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#95A5A6")).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#95A5A6")).
			PaddingLeft(2).
			MarginTop(1)
)

func (m wizardModel) View() string {
	if m.quitting {
		return "Project generation cancelled.\n"
	}

	var s string

	// Title Banner
	s += titleStyle.Render("Ginboot Project Scaffolding Wizard") + "\n"

	// Selection Summary (shows choices made in previous steps)
	if m.step > 0 {
		var summary string
		if m.step >= 1 {
			dbName := m.dbChoices[0]
			for idx, val := range m.dbValues {
				if val == m.dbSelected {
					dbName = m.dbChoices[idx]
					break
				}
			}
			summary += fmt.Sprintf("  Database:  %s\n", dbName)
		}
		if m.step >= 2 {
			storageName := m.storageChoices[0]
			for idx, val := range m.storageValues {
				if val == m.storageSelected {
					storageName = m.storageChoices[idx]
					break
				}
			}
			summary += fmt.Sprintf("  Storage:   %s\n", storageName)
		}
		s += summaryStyle.Render("Selections:\n"+summary) + "\n"
	}

	// Active Question & Menu Choices
	switch m.step {
	case 0:
		s += headerStyle.Render("Choose a Database Integration:") + "\n"
		for i, choice := range m.dbChoices {
			if m.cursor == i {
				s += cursorStyle.Render(" ➜ ") + activeItemStyle.Render(choice) + "\n"
			} else {
				s += inactiveItemStyle.Render("   "+choice) + "\n"
			}
		}
	case 1:
		s += headerStyle.Render("Choose a File Storage Service:") + "\n"
		for i, choice := range m.storageChoices {
			if m.cursor == i {
				s += cursorStyle.Render(" ➜ ") + activeItemStyle.Render(choice) + "\n"
			} else {
				s += inactiveItemStyle.Render("   "+choice) + "\n"
			}
		}
	case 2:
		s += headerStyle.Render("Choose a Deployment Runtime Target:") + "\n"
		for i, choice := range m.deployChoices {
			if m.cursor == i {
				s += cursorStyle.Render(" ➜ ") + activeItemStyle.Render(choice) + "\n"
			} else {
				s += inactiveItemStyle.Render("   "+choice) + "\n"
			}
		}
	}

	// Help instructions
	s += helpStyle.Render("Use arrow keys / j / k to navigate • enter to confirm • ctrl+c to quit") + "\n"

	return s
}

func runWizard() (db string, storage string, deploy string, err error) {
	p := tea.NewProgram(initialWizardModel())
	resModel, err := p.Run()
	if err != nil {
		return "", "", "", err
	}

	m, ok := resModel.(wizardModel)
	if !ok {
		return "", "", "", fmt.Errorf("invalid wizard model state")
	}

	if m.quitting {
		return "", "", "", fmt.Errorf("wizard was cancelled by user")
	}

	return m.dbSelected, m.storageSelected, m.deploySelected, nil
}
