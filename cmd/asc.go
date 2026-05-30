package main

import (
	"fmt"
	"os"
	"os/exec"

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
	verbose         bool
	debug           bool
	usePerplexity   bool
	pickInteractive bool

	// Version information
	version = "dev"

	// Logger
	logger *log.Logger

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

  # Start an interactive multi-round chat
  asc interactive "Let's discuss Go concurrency"
  asc i            # resume the most recent conversation

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

				// Check AI provider command
				aiCommand := "sgpt"
				if usePerplexity {
					aiCommand = "perplexity"
				}
				if _, err := exec.LookPath(aiCommand); err != nil {
					logger.Error("Required command not found", "command", aiCommand, "error", err)
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
)

func init() {
	// Global flags configuration
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show verbose output")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(newCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(appendCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(clearCmd)

	// Add perplexity flag to commands that interact with AI
	newCmd.Flags().BoolVarP(&usePerplexity, "perplexity", "p", false, "Use perplexity command instead of sgpt")
	interactiveCmd.Flags().BoolVarP(&usePerplexity, "perplexity", "p", false, "Use perplexity command instead of sgpt")
	interactiveCmd.Flags().BoolVarP(&pickInteractive, "pick", "P", false, "Pick the conversation to resume from an interactive list")
	appendCmd.Flags().BoolVarP(&usePerplexity, "perplexity", "p", false, "Use perplexity command instead of sgpt")
	editCmd.Flags().BoolVarP(&usePerplexity, "perplexity", "p", false, "Use perplexity command instead of sgpt")
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
	Long: `Start a new conversation with AI by sending a single message.
The conversation will be saved in your data directory for future reference.

A message is required. For a multi-round, back-and-forth chat, use 'asc interactive'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			logger.Error("Message is required")
			os.Exit(1)
		}

		message := args[0]
		logger.Debug("Starting new conversation", "message", message)

		return conversation.StartNewConversation(message, usePerplexity, logger)
	},
}

var interactiveCmd = &cobra.Command{
	Use:     "interactive [message]",
	Aliases: []string{"i"},
	Short:   "Start or resume an interactive multi-round chat",
	Long: `Start an interactive, multi-round conversation with AI.

You'll be prompted for a message, the reply is streamed, and then you're
prompted again — repeating for as many rounds as you like. Each turn re-sends
the accumulated transcript so the AI keeps context.

If a message is provided, a new conversation is started with it as the first
message. If no message is provided, the most recent conversation is resumed and
continued.

Use -P/--pick to choose which conversation to resume from an interactive list
instead of defaulting to the most recent one. A message given alongside --pick is
sent as the next turn of the chosen conversation.

Type /exit or /quit (or press Ctrl-D) to end the session. The conversation is
saved after every turn.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		initial := ""
		if len(args) > 0 {
			initial = args[0]
		}
		logger.Debug("Starting interactive conversation", "initial", initial)

		return conversation.RunInteractive(initial, usePerplexity, pickInteractive, logger)
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
	Aliases: []string{"v", "V"},
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("message is required")
		}

		message := args[0]
		logger.Debug("Continuing previous conversation", "message", message)

		// Get the most recent conversation
		latest, ok, err := conversation.LatestConversation(logger)
		if err != nil {
			return fmt.Errorf("failed to load conversations: %w", err)
		}
		if !ok {
			return fmt.Errorf("no conversations found")
		}

		// Build the follow-up message, including the full prior transcript when the
		// conversation has multiple turns (falls back to the single Q/A pair otherwise).
		prior := latest.Turns
		if len(prior) == 0 {
			prior = []conversation.Turn{{Question: latest.Message, Answer: latest.Response}}
		}
		contextMessage := conversation.BuildTranscriptInput(prior, message)

		// Start a new conversation with the context
		return conversation.StartNewConversation(contextMessage, usePerplexity, logger)
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

		// Get the most recent conversation
		latest, ok, err := conversation.LatestConversation(logger)
		if err != nil {
			return fmt.Errorf("failed to load conversations: %w", err)
		}
		if !ok {
			return fmt.Errorf("no conversations found")
		}

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
		return conversation.StartNewConversation(string(editedMessage), usePerplexity, logger)
	},
}

var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"c"},
	Short:   "Edit the context file",
	Long:    `Open the context file in your default editor. The context is used to provide additional information to AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load existing context
		context, err := conversation.LoadContext(logger)
		if err != nil {
			return err
		}

		// Create a temporary file with the context
		tmpFile, err := os.CreateTemp("", "context-*.txt")
		if err != nil {
			logger.Error("Failed to create temp file", "error", err)
			return err
		}

		if _, err := tmpFile.WriteString(context); err != nil {
			logger.Error("Failed to write to temp file", "error", err)
			return err
		}
		tmpFile.Close()

		// Get editor from environment variable
		editor := os.Getenv("EDITOR")
		if editor == "" {
			logger.Error("EDITOR environment variable is not set")
			return err
		}

		// Open the file in the editor
		editCmd := exec.Command(editor, tmpFile.Name())
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr
		logger.Info("Opening editor", "editor", editor, "file", tmpFile.Name())

		if err := editCmd.Run(); err != nil {
			logger.Error("Failed to open editor", "error", err)
			return err
		}

		// Read the edited context
		editedContext, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			logger.Error("Failed to read edited context", "error", err)
			return err
		}

		// Save the edited context
		if err := conversation.SaveContext(string(editedContext), logger); err != nil {
			logger.Error("Failed to save context", "error", err)
			return err
		}

		// Clean up
		if err := os.Remove(tmpFile.Name()); err != nil {
			logger.Error("Failed to remove temporary file", "error", err)
		}

		return nil
	},
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the context file",
	Long:  `Remove the context file. This will clear any additional context provided to AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := conversation.ClearContext(logger); err != nil {
			logger.Error("Failed to clear context", "error", err)
			return err
		}
		return nil
	},
}

func main() {
	// Initialize logger with default options
	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		Level:           log.InfoLevel,
	})

	if err := rootCmd.Execute(); err != nil {
		logger.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
