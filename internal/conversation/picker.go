package conversation

import (
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// pickerModel is a minimal Bubble Tea table for choosing a conversation to
// resume in interactive mode. Unlike the full view UI it offers no edit/delete
// actions — it just returns the row the user selects.
type pickerModel struct {
	table         table.Model
	conversations []Conversation
	selected      Conversation
	picked        bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if len(m.conversations) > 0 {
				m.selected = m.conversations[m.table.Cursor()]
				m.picked = true
			}
			return m, tea.Quit
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m pickerModel) View() string {
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("↑/↓: move • enter: resume • q/esc: cancel")
	return lipgloss.JoinVertical(lipgloss.Left, m.table.View(), help)
}

// truncatePicker shortens s to maxLen runes, appending an ellipsis when cut.
func truncatePicker(s string, maxLen int) string {
	if maxLen <= 3 || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// PickConversation shows an interactive list of saved conversations and returns
// the one the user selects. ok is false when the user cancels or none exist.
// The picker is rendered on stderr so the chat session's piped stdout stays clean.
func PickConversation(logger *log.Logger) (Conversation, bool, error) {
	convs, err := LoadConversations(logger)
	if err != nil {
		return Conversation{}, false, err
	}
	if len(convs) == 0 {
		return Conversation{}, false, nil
	}

	// Newest first, matching the view command's ordering.
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].Timestamp.After(convs[j].Timestamp)
	})

	idWidth, dateWidth := 14, 19
	messageWidth := getTerminalWidth() - idWidth - dateWidth - 8
	if messageWidth < 10 {
		messageWidth = 10
	}

	columns := []table.Column{
		{Title: "ID", Width: idWidth},
		{Title: "Date", Width: dateWidth},
		{Title: "Message", Width: messageWidth},
	}

	var rows []table.Row
	for _, conv := range convs {
		rows = append(rows, table.Row{
			conv.ID,
			conv.Timestamp.Format("2006-01-02 15:04:05"),
			truncatePicker(conv.Message, messageWidth),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
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

	p := tea.NewProgram(
		pickerModel{table: t, conversations: convs},
		tea.WithOutput(os.Stderr),
	)
	finalModel, err := p.Run()
	if err != nil {
		return Conversation{}, false, err
	}
	fm, ok := finalModel.(pickerModel)
	if !ok || !fm.picked {
		return Conversation{}, false, nil
	}
	return fm.selected, true, nil
}
