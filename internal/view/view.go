package view

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"asc/internal/conversation"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

type model struct {
	table         table.Model
	conversations []conversation.Conversation
	logger        *log.Logger
}

func initialModel(logger *log.Logger) model {
	columns := []table.Column{
		{Title: "ID", Width: 15},
		{Title: "Date", Width: 20},
		{Title: "Message", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		table:  t,
		logger: logger,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func openPager(selected conversation.Conversation, logger *log.Logger) tea.Cmd {
	// create a temporary file to save the conversation message
	tempFile, err := os.CreateTemp("", "conversation.md")
	if err != nil {
		logger.Error("Failed to create temporary file", "error", err)
		return nil
	}

	// Format conversation content
	content := fmt.Sprintf("# Conversation %s\n\n", selected.ID)
	content += fmt.Sprintf("## User\n\n%s\n\n", selected.Message)
	content += fmt.Sprintf("## AI\n\n%s\n", selected.Response)

	// write the conversation content to the temporary file
	if _, err := tempFile.WriteString(content); err != nil {
		logger.Error("Failed to write conversation content to temporary file", "error", err)
		return nil
	}

	c := exec.Command("less", "-SR", tempFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, tea.Quit
		case "enter", "v":
			if len(m.conversations) > 0 {
				selected := m.conversations[m.table.Cursor()]
				return m, openPager(selected, m.logger)
			}
			return m, nil
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return m.table.View()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func StartView(logger *log.Logger) error {
	logger.Debug("Viewing conversation history")

	conversations, err := conversation.LoadConversations(logger)
	if err != nil {
		return err
	}

	// Sort conversations by timestamp (newest first)
	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].Timestamp.After(conversations[j].Timestamp)
	})

	// Create table rows
	var rows []table.Row
	for _, conv := range conversations {
		rows = append(rows, table.Row{
			conv.ID,
			conv.Timestamp.Format("2006-01-02 15:04:05"),
			truncateString(conv.Message, 47),
		})
	}

	// Initialize and run the table UI
	m := initialModel(logger)
	m.table.SetRows(rows)
	m.conversations = conversations

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
