package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"asc/internal/config"

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

		// Execute sgpt with --stream option
		sgptCmd := exec.Command("sgpt", "--stream", message)
		stdout, err := sgptCmd.StdoutPipe()
		if err != nil {
			logger.Error("Failed to create stdout pipe", "error", err)
			os.Exit(1)
		}
		sgptCmd.Stderr = os.Stderr

		if err := sgptCmd.Start(); err != nil {
			logger.Error("Failed to start sgpt", "error", err)
			os.Exit(1)
		}

		// Check if style file exists
		shareDir, err := config.GetShareDir()
		if err != nil {
			logger.Error("Failed to get share directory", "error", err)
			os.Exit(1)
		}
		stylePath := filepath.Join(shareDir, "ggpt_glow_style.json")
		hasStyleFile := false
		if _, err := os.Stat(stylePath); err == nil {
			logger.Debug("Using custom style", "path", stylePath)
			hasStyleFile = true
		}

		// Buffer for storing all output
		var buffer strings.Builder
		scanner := bufio.NewScanner(stdout)
		var previousGlowOutput string
		previousGlowOutput = ""

		for {
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					if err != io.EOF {
						logger.Error("Error reading sgpt output", "error", err)
						os.Exit(1)
					}
					// Stream is closed (EOF)
					// break
				}
				// No more data and no error (EOF)
				previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
				for i := max(0, len(previousGlowOutputLines)-2); i < len(previousGlowOutputLines); i++ {
					fmt.Println(previousGlowOutputLines[i])
				}
				saveNewConversation(previousGlowOutput, message)
				break
			}
			buffer.WriteString(scanner.Text() + "\n")

			// Execute glow command with buffer content
			glowCmd := exec.Command("glow")
			glowCmd.Env = append(os.Environ(), "CLICOLOR_FORCE=1")

			if hasStyleFile {
				glowCmd.Args = append(glowCmd.Args, "--style", stylePath)
			}

			glowCmd.Stdin = strings.NewReader(buffer.String())
			glowCmd.Stderr = os.Stderr
			var glowOutput strings.Builder
			glowOutput = strings.Builder{}
			glowCmd.Stdout = &glowOutput
			if err := glowCmd.Run(); err != nil {
				logger.Error("Failed to execute glow", "error", err)
				os.Exit(1)
			}
			if previousGlowOutput != glowOutput.String() {
				previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
				glowOutputLines := strings.Split(glowOutput.String(), "\n")
				for i := max(0, len(previousGlowOutputLines)-2); i < len(glowOutputLines)-2; i++ {
					fmt.Println(glowOutputLines[i])
				}
				previousGlowOutput = glowOutput.String()
			}
		}

		if err := sgptCmd.Wait(); err != nil {
			logger.Error("sgpt command failed", "error", err)
			os.Exit(1)
		}

		return nil
	},
}

var viewCmd = &cobra.Command{
	Use:     "view",
	Aliases: []string{"v"},
	Short:   "View conversation history",
	Long: `Display the history of your conversations with AI.
Shows a list of all conversations with their IDs, timestamps, and previews.
You can use these IDs with other commands like 'append' and 'edit'.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Viewing conversation history")
		// TODO: Implement conversation history view
	},
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
	Run: func(cmd *cobra.Command, args []string) {
		logger.Debug("Editing previous message")
		// TODO: Implement message editing
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
