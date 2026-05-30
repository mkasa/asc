package conversation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"asc/internal/config"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"golang.org/x/term"
)

// Turn is a single question/answer exchange within a conversation.
type Turn struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Conversation struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	FilePath  string    `json:"file_path"`
	Context   string    `json:"context,omitempty"`
	// Turns holds the full multi-round transcript for interactive conversations.
	// It is absent for legacy single-turn files; Message/Response are kept
	// populated for backward compatibility (preview column, edit/append prefill).
	Turns []Turn `json:"turns,omitempty"`
}

// writeConversation marshals c and writes it to its canonical path. FilePath is
// set to the final filename before marshaling so a single write suffices.
func writeConversation(c *Conversation, logger *log.Logger) error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	// Create conversations directory if it doesn't exist
	conversationsDir := filepath.Join(dataDir, "conversations")
	if err := os.MkdirAll(conversationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create conversations directory: %w", err)
	}

	c.FilePath = filepath.Join(conversationsDir, c.ID+".json")
	data, err := json.MarshalIndent(*c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}
	if err := os.WriteFile(c.FilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	logger.Debug("Saved conversation", "id", c.ID, "path", c.FilePath)
	return nil
}

// SaveNewConversation saves a single question/answer pair as a new conversation.
func SaveNewConversation(response, message, context string, logger *log.Logger) error {
	conversation := Conversation{
		ID:        time.Now().Format("20060102150405"),
		Timestamp: time.Now(),
		Message:   message,
		Response:  response,
		Context:   context,
	}
	return writeConversation(&conversation, logger)
}

// SaveConversationTurns writes or rewrites a multi-turn conversation. When id is
// empty a fresh ID (and file) is generated; otherwise the existing file with that
// ID is rewritten in place with a refreshed timestamp. Returns the conversation ID.
func SaveConversationTurns(id string, turns []Turn, context string, logger *log.Logger) (string, error) {
	if id == "" {
		id = time.Now().Format("20060102150405")
	}
	conversation := Conversation{
		ID:        id,
		Timestamp: time.Now(),
		Context:   context,
		Turns:     turns,
	}
	conversation.syncCompatFields()
	if err := writeConversation(&conversation, logger); err != nil {
		return "", err
	}
	return id, nil
}

// syncCompatFields keeps the legacy Message/Response fields populated from Turns
// so the view preview column and edit/append prefill keep working.
func (c *Conversation) syncCompatFields() {
	if len(c.Turns) == 0 {
		return
	}
	c.Message = c.Turns[0].Question
	c.Response = c.Turns[len(c.Turns)-1].Answer
}

// BuildTranscriptInput renders prior turns plus the new question into the text a
// stateless provider receives. Mirrors the format used by the append command.
func BuildTranscriptInput(prior []Turn, newQuestion string) string {
	if len(prior) == 0 {
		return newQuestion
	}
	var b strings.Builder
	b.WriteString("Previous conversation:\n")
	for _, t := range prior {
		fmt.Fprintf(&b, "User: %s\nAI: %s\n", t.Question, t.Answer)
	}
	b.WriteString("\n# Follow-up question\n")
	b.WriteString(newQuestion)
	return b.String()
}

// RenderMarkdown produces the glow/pager document for a conversation, rendering
// all turns when present and falling back to Message/Response for legacy files.
func (c Conversation) RenderMarkdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Conversation %s\n", c.ID)
	if c.Context != "" {
		fmt.Fprintf(&b, "\n## Context\n%s\n", c.Context)
	}
	if len(c.Turns) > 0 {
		for i, t := range c.Turns {
			fmt.Fprintf(&b, "\n## User (turn %d)\n%s\n\n## AI (turn %d)\n%s\n", i+1, t.Question, i+1, t.Answer)
		}
	} else {
		fmt.Fprintf(&b, "\n## User\n%s\n\n## AI\n%s", c.Message, c.Response)
	}
	return b.String()
}

func LoadConversations(logger *log.Logger) ([]Conversation, error) {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return nil, err
	}

	conversationsDir := filepath.Join(dataDir, "conversations")
	files, err := os.ReadDir(conversationsDir)
	if err != nil {
		return nil, err
	}

	var conversations []Conversation
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			filePath := filepath.Join(conversationsDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				logger.Error("Failed to read conversation file", "file", file.Name(), "error", err)
				continue
			}

			var conv Conversation
			if err := json.Unmarshal(data, &conv); err != nil {
				logger.Error("Failed to unmarshal conversation", "file", file.Name(), "error", err)
				continue
			}

			// ファイルパスが設定されていない場合は設定
			if conv.FilePath == "" {
				conv.FilePath = filePath
				// ファイルパスを含めて再度保存
				data, err = json.MarshalIndent(conv, "", "  ")
				if err != nil {
					logger.Error("Failed to marshal conversation with file path", "file", file.Name(), "error", err)
					continue
				}
				if err := os.WriteFile(filePath, data, 0644); err != nil {
					logger.Error("Failed to save conversation with file path", "file", file.Name(), "error", err)
					continue
				}
			}

			conversations = append(conversations, conv)
		}
	}

	return conversations, nil
}

// LatestConversation returns the most recent conversation by timestamp. The bool
// is false when no conversations exist. LoadConversations returns files in an
// unspecified order, so this sorts newest-first before picking.
func LatestConversation(logger *log.Logger) (Conversation, bool, error) {
	convs, err := LoadConversations(logger)
	if err != nil {
		return Conversation{}, false, err
	}
	if len(convs) == 0 {
		return Conversation{}, false, nil
	}
	sort.Slice(convs, func(i, j int) bool {
		return convs[i].Timestamp.After(convs[j].Timestamp)
	})
	return convs[0], true, nil
}

// getTerminalWidth returns the terminal width, defaulting to 80 if unable to determine
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 // default width if unable to get terminal size
	}
	return width
}

func ShowConversation(conv Conversation, logger *log.Logger) error {
	// Get terminal width
	terminalWidth := getTerminalWidth()

	// Execute glow command with conversation content
	glowCmd := exec.Command("glow", "-p", "-w", fmt.Sprintf("%d", terminalWidth-2))

	// Check if style file exists
	shareDir, err := config.GetShareDir()
	if err != nil {
		return fmt.Errorf("failed to get share directory: %w", err)
	}
	stylePath := filepath.Join(shareDir, "ggpt_glow_style.json")
	if _, err := os.Stat(stylePath); err == nil {
		glowCmd.Args = append(glowCmd.Args, "--style", stylePath)
	}

	// Format conversation content
	content := conv.RenderMarkdown()

	glowCmd.Stdin = strings.NewReader(content)
	glowCmd.Stdout = os.Stdout
	glowCmd.Stderr = os.Stderr
	if err := glowCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute glow: %w", err)
	}
	return nil
}

// buildAICmd constructs the provider subprocess for a single message.
func buildAICmd(aiInput string, usePerplexity bool) *exec.Cmd {
	if usePerplexity {
		return exec.Command("perplexity", "-g", "--stream", "--citation", aiInput)
	}
	return exec.Command("sgpt", "--stream", aiInput)
}

// streamResponse runs the AI provider on aiInput, streams its output through glow
// to stdout (with held-back-line anti-flicker), and returns the trimmed response.
// It does not save anything.
func streamResponse(aiInput string, usePerplexity bool, logger *log.Logger) (string, error) {
	aiCmd := buildAICmd(aiInput, usePerplexity)
	stdout, err := aiCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	aiCmd.Stderr = os.Stderr

	if err := aiCmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start AI command: %w", err)
	}

	// Check if style file exists
	shareDir, err := config.GetShareDir()
	if err != nil {
		return "", fmt.Errorf("failed to get share directory: %w", err)
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

	var response string
	const HELD_OUT_LINE_COUNT = 4
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				if err != io.EOF {
					return "", fmt.Errorf("error reading AI output: %w", err)
				}
				// Stream is closed (EOF)
				// break
			}
			// No more data and no error (EOF)
			previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
			for i := max(0, len(previousGlowOutputLines)-HELD_OUT_LINE_COUNT); i < len(previousGlowOutputLines); i++ {
				fmt.Println(previousGlowOutputLines[i])
			}
			// Trim excessive trailing newlines before returning
			response = strings.TrimRightFunc(buffer.String(), func(r rune) bool {
				return r == '\n' || r == '\r'
			})
			break
		}
		buffer.WriteString(scanner.Text() + "\n")

		// Execute glow command with buffer content
		terminalWidth := getTerminalWidth()
		glowCmd := exec.Command("glow", "-w", fmt.Sprintf("%d", terminalWidth-2))
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
			return "", fmt.Errorf("failed to execute glow: %w", err)
		}
		if previousGlowOutput != glowOutput.String() {
			previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
			glowOutputLines := strings.Split(glowOutput.String(), "\n")
			for i := max(0, len(previousGlowOutputLines)-HELD_OUT_LINE_COUNT); i < len(glowOutputLines)-HELD_OUT_LINE_COUNT; i++ {
				fmt.Println(glowOutputLines[i])
			}
			previousGlowOutput = glowOutput.String()
		}
	}

	if err := aiCmd.Wait(); err != nil {
		return "", fmt.Errorf("AI command failed: %w", err)
	}

	return response, nil
}

func StartNewConversation(message string, usePerplexity bool, logger *log.Logger) error {
	// Load context if exists
	context, err := LoadContext(logger)
	if err != nil {
		logger.Error("Failed to load context", "error", err)
		return err
	}

	// Prepend context to message if it exists (only for sgpt)
	aiInput := message
	if !usePerplexity && context != "" {
		aiInput = fmt.Sprintf("# Context\n%s\n\n# Question\n%s", context, message)
	}

	response, err := streamResponse(aiInput, usePerplexity, logger)
	if err != nil {
		return err
	}

	if err := SaveNewConversation(response, message, context, logger); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}
	return nil
}

// RunInteractive runs a multi-round chat loop. When initialMessage is non-empty it
// starts a fresh conversation with that message as the first turn. Otherwise it
// resumes the most recent conversation (seeding the transcript from its turns, or
// a legacy Message/Response pair) and continues writing to the same file. The loop
// reads questions from stdin, streams each reply, and saves after every turn.
// conversationTurns returns a conversation's turns for resuming, falling back to
// the legacy single Message/Response pair when no multi-turn transcript exists.
func conversationTurns(c Conversation) []Turn {
	if len(c.Turns) > 0 {
		return append([]Turn(nil), c.Turns...)
	}
	if c.Message != "" || c.Response != "" {
		return []Turn{{Question: c.Message, Answer: c.Response}}
	}
	return nil
}

// It exits cleanly on EOF (Ctrl-D) or the /exit and /quit commands.
//
// When pick is true, the user chooses which conversation to resume from an
// interactive list instead of defaulting to the most recent one. A provided
// initialMessage is still sent as the first turn of the chosen conversation.
func RunInteractive(initialMessage string, usePerplexity bool, pick bool, logger *log.Logger) error {
	// Load context once (prepended to provider input for sgpt only, like new/append).
	context, err := LoadContext(logger)
	if err != nil {
		logger.Error("Failed to load context", "error", err)
		return err
	}

	var turns []Turn
	convID := ""

	// banner prints the session's opening system message in bold, followed by a
	// blank line to visually separate it from the first streamed answer.
	bold := lipgloss.NewStyle().Bold(true)
	banner := func(msg string) {
		fmt.Fprintf(os.Stderr, "%s\n\n", bold.Render(msg))
	}

	switch {
	case pick:
		// Let the user choose which conversation to resume from a list.
		selected, ok, err := PickConversation(logger)
		if err != nil {
			logger.Error("Failed to pick conversation", "error", err)
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "No conversation selected.")
			return nil
		}
		convID = selected.ID
		turns = conversationTurns(selected)
		banner(fmt.Sprintf("Resuming conversation %s (%d turns). Type /exit to quit.", convID, len(turns)))
	case initialMessage == "":
		// Resume the most recent conversation, if any.
		latest, ok, err := LatestConversation(logger)
		if err != nil {
			logger.Debug("No conversations to resume", "error", err)
		}
		if ok {
			convID = latest.ID
			turns = conversationTurns(latest)
			banner(fmt.Sprintf("Resuming conversation %s (%d turns). Type /exit to quit.", convID, len(turns)))
		} else {
			banner("No previous conversation; starting fresh. Type /exit to quit.")
		}
	default:
		banner("Starting a new conversation. Type /exit to quit.")
	}

	reader := bufio.NewReader(os.Stdin)

	// processTurn sends a question, streams the reply, records and saves the turn.
	processTurn := func(question string) error {
		aiInput := BuildTranscriptInput(turns, question)
		if !usePerplexity && context != "" {
			aiInput = fmt.Sprintf("# Context\n%s\n\n# Question\n%s", context, aiInput)
		}
		response, err := streamResponse(aiInput, usePerplexity, logger)
		if err != nil {
			// Keep the session alive so the user can retry.
			logger.Error("AI turn failed", "error", err)
			return nil
		}
		turns = append(turns, Turn{Question: question, Answer: response})
		newID, err := SaveConversationTurns(convID, turns, context, logger)
		if err != nil {
			logger.Error("Failed to save turn", "error", err)
			return nil
		}
		convID = newID
		return nil
	}

	if initialMessage != "" {
		if err := processTurn(initialMessage); err != nil {
			return err
		}
	}

	for {
		fmt.Print("\nyou> ")
		line, err := reader.ReadString('\n')
		question := strings.TrimSpace(strings.TrimRight(line, "\r\n"))

		if err != nil {
			if err == io.EOF {
				// Process a trailing partial line (no newline) before exiting.
				if question != "" && question != "/exit" && question != "/quit" {
					fmt.Println()
					_ = processTurn(question)
				}
				fmt.Fprintln(os.Stderr, "\nGoodbye.")
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		if question == "" {
			continue
		}
		if question == "/exit" || question == "/quit" {
			fmt.Fprintln(os.Stderr, "Goodbye.")
			return nil
		}

		if err := processTurn(question); err != nil {
			return err
		}
	}
}

// DeleteConversation deletes a conversation by its ID
func DeleteConversation(id string, logger *log.Logger) error {
	dataDir, err := config.GetDataDir()
	if err != nil {
		return fmt.Errorf("failed to get data directory: %w", err)
	}

	conversationsDir := filepath.Join(dataDir, "conversations")
	filename := filepath.Join(conversationsDir, id+".json")

	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete conversation file: %w", err)
	}

	logger.Debug("Deleted conversation", "id", id)
	return nil
}

// GetContextPath returns the path to the context file
func GetContextPath(logger *log.Logger) (string, error) {
	shareDir, err := config.GetShareDir()
	if err != nil {
		return "", fmt.Errorf("failed to get share directory: %w", err)
	}
	return filepath.Join(shareDir, "context.txt"), nil
}

// LoadContext loads the context from the file
func LoadContext(logger *log.Logger) (string, error) {
	contextPath, err := GetContextPath(logger)
	if err != nil {
		return "", err
	}

	// Check if context file exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return "", nil
	}

	content, err := os.ReadFile(contextPath)
	if err != nil {
		return "", fmt.Errorf("failed to read context file: %w", err)
	}

	return string(content), nil
}

// SaveContext saves the context to the file
func SaveContext(context string, logger *log.Logger) error {
	contextPath, err := GetContextPath(logger)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(contextPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(contextPath, []byte(context), 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	return nil
}

// ClearContext removes the context file
func ClearContext(logger *log.Logger) error {
	contextPath, err := GetContextPath(logger)
	if err != nil {
		return err
	}

	if err := os.Remove(contextPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove context file: %w", err)
	}

	return nil
}
