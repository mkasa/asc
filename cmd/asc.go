package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose bool
	debug   bool

	// Root command
	rootCmd = &cobra.Command{
		Use:   "asc",
		Short: "Interactive AI Command Line Tool",
		Long: `ASC (AI Shell Chat) is a command-line tool for interacting with AI.
You can have natural conversations with AI and perform tasks or get information.

Examples:
  asc new "What's the weather like?"
  asc n "Tell me about Go"
  asc append "Can you explain more about that?"
  asc a "What else should I know?"
  asc edit "Can you explain that differently?"
  asc e "Let me rephrase that"
  asc view  # View conversation history
  asc help  # Show detailed usage`,
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
		},
		Run: func(cmd *cobra.Command, args []string) {
			logger.Info("Starting AI conversation")
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
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Version information", "version", "1.0.0")
	},
}

var newCmd = &cobra.Command{
	Use:     "new [message]",
	Aliases: []string{"n"},
	Short:   "Start a new conversation with AI",
	Long:    `Start a new interactive conversation session with AI. Supports continuous conversation.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting new conversation")
		// TODO: Implement new conversation mode
	},
}

var viewCmd = &cobra.Command{
	Use:     "view",
	Aliases: []string{"v"},
	Short:   "View conversation history",
	Long:    `Display the history of your conversations with AI.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Viewing conversation history")
		// TODO: Implement conversation history view
	},
}

var appendCmd = &cobra.Command{
	Use:     "append [message]",
	Aliases: []string{"a"},
	Short:   "Continue a previous conversation",
	Long:    `Add a follow-up question or message to a previous conversation. If no conversation ID is specified, continues with the most recent conversation.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Continuing previous conversation")
		// TODO: Implement conversation continuation
	},
}

var editCmd = &cobra.Command{
	Use:     "edit [message]",
	Aliases: []string{"e"},
	Short:   "Edit and resend a previous message",
	Long:    `Modify a previous message and resend it to AI. If no message ID is specified, edits the most recent message.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Editing previous message")
		// TODO: Implement message editing
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("An error occurred", "error", err)
		os.Exit(1)
	}
}
