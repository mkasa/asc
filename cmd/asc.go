package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"asc/internal/config"
	"asc/internal/conversation"
	"asc/internal/view"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	debug   bool

	// Version information
	version = "dev"

	// Root command
	rootCmd = &cobra.Command{
		Use:   "asc",
		Short: "AI Shell Chat - Interactive AI Command Line Tool",
		Long: `ASC (AI Shell Chat) is a command-line tool for interacting with AI.
You can have natural conversations with AI and perform tasks or get information.

Features:
  - Start new conversations with AI
  - Continue previous conversations
  - Edit and resend messages
  - View conversation history
  - Debug mode for detailed logging

Examples:
  # Start a new conversation
  asc new "What's the weather like?"
  asc n "Tell me about Go"

  # Continue a previous conversation
  asc append "Can you explain more about that?"
  asc a "What else should I know?"

  # Edit a previous message
  asc edit "Can you explain that differently?"
  asc e "Let me rephrase that"

  # View conversation history
  asc view
  asc v

  # Show help
  asc help`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Logger configuration
			level := log.InfoLevel
			if debug {
				level = log.DebugLevel
			}
			logger = log.NewWithOptions(os.Stderr, log.Options{
				ReportCaller:    true,
				ReportTimestamp: true,
				Level:           level,
			})

			// Check required commands
			if cmd.Name() != "version" {
				// Check glow command
				if _, err := exec.LookPath("glow"); err != nil {
					logger.Error("Required command not found", "command", "glow", "error", err)
					os.Exit(1)
				}

				// Check sgpt command
				if _, err := exec.LookPath("sgpt"); err != nil {
					logger.Error("Required command not found", "command", "sgpt", "error", err)
					os.Exit(1)
				}

				// Ensure share directory exists
				if err := config.EnsureShareDir(); err != nil {
					logger.Error("Failed to ensure share directory", "error", err)
					os.Exit(1)
				}
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Debug("Starting AI conversation")
		},
	}

	logger *log.Logger
)

func init() {
	// Global flags configuration
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(appendCmd)
	rootCmd.AddCommand(editCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display the current version of ASC and build information.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ASC version %s\n", version)
	},
}

var newCmd = &cobra.Command{
	Use:     "new [message]",
	Aliases: []string{"n"},
	Short:   "Start a new conversation with AI",
	Long: `Start a new interactive conversation session with AI.
The conversation will be saved in your data directory for future reference.

If a message is provided, it will be sent as the first message to AI.
Otherwise, you'll enter an interactive mode where you can type messages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			logger.Error("Message is required")
			os.Exit(1)
		}

		message := args[0]
		logger.Debug("Starting new conversation", "message", message)

		return conversation.StartNewConversation(message, logger)
	},
}

type model struct {
	table         table.Model
	conversations []conversation.Conversation
}

func initialModel() model {
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
		table: t,
	}
}

func (m model) Init() tea.Cmd {
	return nil
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
				if err := conversation.ShowConversation(selected, logger); err != nil {
					logger.Error("Failed to show conversation", "error", err)
				}
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

var viewCmd = &cobra.Command{
	Use:     "view",
	Aliases: []string{"v"},
	Short:   "View conversation history",
	Long: `Display the history of your conversations with AI.
Shows a list of all conversations with their IDs, timestamps, and previews.
You can use these IDs with other commands like 'append' and 'edit'.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := view.StartView(logger); err != nil {
			logger.Error("Failed to start view", "error", err)
			os.Exit(1)
		}
	},
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var appendCmd = &cobra.Command{
	Use:     "append [message]",
	Aliases: []string{"a"},
	Short:   "Continue a previous conversation",
	Long: `Add a follow-up question or message to a previous conversation.
If no conversation ID is specified, continues with the most recent conversation.

The message will be added to the existing conversation context,
allowing AI to maintain context from previous messages.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Continuing previous conversation")
		// TODO: Implement conversation continuation
	},
}

var editCmd = &cobra.Command{
	Use:     "edit [message]",
	Aliases: []string{"e"},
	Short:   "Edit and resend a previous message",
	Long: `Modify a previous message and resend it to AI.
If no message ID is specified, edits the most recent message.

This is useful when you want to rephrase a question or
correct a typo in a previous message.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Debug("Editing previous message")

		// Load conversations
		conversations, err := conversation.LoadConversations(logger)
		if err != nil {
			return fmt.Errorf("failed to load conversations: %w", err)
		}

		if len(conversations) == 0 {
			return fmt.Errorf("no conversations found")
		}

		// Get the most recent conversation
		latest := conversations[0]

		// Create a temporary file with the message
		tmpFile, err := os.CreateTemp("", "edit-*.txt")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(latest.Message); err != nil {
			return fmt.Errorf("failed to write to temp file: %w", err)
		}
		tmpFile.Close()

		// Get editor from environment variable
		editor := os.Getenv("EDITOR")
		if editor == "" {
			return fmt.Errorf("EDITOR environment variable is not set")
		}

		// Open the file in the editor
		editCmd := exec.Command(editor, tmpFile.Name())
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr
		if err := editCmd.Run(); err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		// Read the edited message
		editedMessage, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("failed to read edited message: %w", err)
		}

		// Start a new conversation with the edited message
		return conversation.StartNewConversation(string(editedMessage), logger)
	},
}

type Conversation struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
}

func saveNewConversation(response, message string) error {
	// Get data directory
	dataDir, err := config.GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	// Create conversations directory if it doesn't exist
	conversationsDir := filepath.Join(dataDir, "conversations")
	if err := os.MkdirAll(conversationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create conversations directory: %w", err)
	}

	// Create new conversation
	conversation := Conversation{
		ID:        time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		Message:   message,
		Response:  response,
	}

	// Convert to JSON
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	// Save to file
	filename := filepath.Join(conversationsDir, conversation.ID+".json")
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	logger.Debug("Saved conversation", "id", conversation.ID)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
