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
	"golang.org/x/term"
)

type model struct {
	table         table.Model
	conversations []conversation.Conversation
	logger        *log.Logger
	showConfirm   bool
	selectedID    string
	terminalWidth int
}

type editCompleteMsg struct {
	message string
}

func initialModel(logger *log.Logger, terminalWidth int) model {
	// Calculate column widths based on terminal width
	// Account for borders (2 chars) and padding (2 chars) = 4 chars total
	availableWidth := terminalWidth - 4

	// Fixed widths for ID and Date columns
	idWidth := 16
	dateWidth := 20
	messageWidth := availableWidth - idWidth - dateWidth

	columns := []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Date", Width: dateWidth},
		{Title: "Message", Width: messageWidth},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return model{
		table:         t,
		logger:        logger,
		terminalWidth: terminalWidth,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func openGlow(selected conversation.Conversation, logger *log.Logger, terminalWidth int) tea.Cmd {
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

	// Set glow width to terminal width minus 2
	glowWidth := terminalWidth - 2
	logger.Debug("Terminal width", "width", terminalWidth, "glow_width", glowWidth)

	c := exec.Command("glow", "-p", "-w", fmt.Sprintf("%d", glowWidth), tempFile.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return nil
	})
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

func editConversation(selected conversation.Conversation, logger *log.Logger) tea.Cmd {
	// Create a temporary file with the message
	tmpFile, err := os.CreateTemp("", "edit-*.txt")
	if err != nil {
		logger.Error("Failed to create temp file", "error", err)
		return nil
	}

	if _, err := tmpFile.WriteString(selected.Message); err != nil {
		logger.Error("Failed to write to temp file", "error", err)
		return nil
	}
	tmpFile.Close()

	// Get editor from environment variable
	editor := os.Getenv("EDITOR")
	if editor == "" {
		logger.Error("EDITOR environment variable is not set")
		return nil
	}

	// Open the file in the editor
	editCmd := exec.Command(editor, tmpFile.Name())
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr
	logger.Info("Opening editor", "editor", editor, "file", tmpFile.Name())

	return tea.ExecProcess(editCmd, func(err error) tea.Msg {
		defer os.Remove(tmpFile.Name())
		if err != nil {
			logger.Error("Failed to open editor", "error", err)
			return err
		}
		// Read the edited message
		editedMessageByte, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			logger.Error("Failed to read edited message", "error", err)
			return err
		}
		editedMessageString := string(editedMessageByte)
		logger.Info("Edited message", "message", editedMessageString)
		return editCompleteMsg{message: editedMessageString}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.showConfirm {
				m.showConfirm = false
				return m, nil
			}
			return m, tea.Quit
		case "enter", "v":
			if m.showConfirm {
				// Delete the conversation
				if err := conversation.DeleteConversation(m.selectedID, m.logger); err != nil {
					m.logger.Error("Failed to delete conversation", "error", err)
					return m, nil
				}
				// Remove from the list
				for i, conv := range m.conversations {
					if conv.ID == m.selectedID {
						m.conversations = append(m.conversations[:i], m.conversations[i+1:]...)
						break
					}
				}
				// Update table rows
				var rows []table.Row
				for _, conv := range m.conversations {
					rows = append(rows, table.Row{
						conv.ID,
						conv.Timestamp.Format("2006-01-02 15:04:05"),
						truncateString(conv.Message, int(float64(m.terminalWidth-4)*0.65)),
					})
				}
				m.table.SetRows(rows)
				m.showConfirm = false
				return m, nil
			}
			if len(m.conversations) > 0 {
				selected := m.conversations[m.table.Cursor()]
				return m, openGlow(selected, m.logger, m.terminalWidth)
			}
			return m, nil
		case "V":
			if len(m.conversations) > 0 {
				selected := m.conversations[m.table.Cursor()]
				return m, openPager(selected, m.logger)
			}
			return m, nil
		case "e":
			if len(m.conversations) > 0 {
				selected := m.conversations[m.table.Cursor()]
				return m, editConversation(selected, m.logger)
			}
			return m, nil
		case "d":
			if !m.showConfirm && len(m.conversations) > 0 {
				m.showConfirm = true
				m.selectedID = m.conversations[m.table.Cursor()].ID
				return m, nil
			}
			return m, nil
		case "n":
			if m.showConfirm {
				m.showConfirm = false
				return m, nil
			}
			return m, nil
		}
	case editCompleteMsg:
		// Start new conversation with edited message
		return m, tea.ExecProcess(exec.Command("asc", "new", msg.message), func(err error) tea.Msg {
			if err != nil {
				m.logger.Error("Failed to execute asc new", "error", err)
			}
			return tea.Quit
		})
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.showConfirm {
		style := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2)

		content := fmt.Sprintf("Delete conversation %s?\n\n", m.selectedID)
		content += "Press Enter to confirm, 'n' to cancel"
		return style.Render(content)
	}
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

	// Get terminal width using term.GetSize
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("failed to get terminal width: %w", err)
	}
	logger.Debug("Terminal width", "width", width, "source", "term.GetSize")

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
			truncateString(conv.Message, width-42), // 42 = 16 (ID) + 20 (Date) + 4 (borders) + 2 (padding)
		})
	}

	// Initialize and run the table UI
	m := initialModel(logger, width)
	m.table.SetRows(rows)
	m.conversations = conversations

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
