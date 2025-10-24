package conversation

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
	"golang.org/x/term"
)

type Conversation struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	FilePath  string    `json:"file_path"`
	Context   string    `json:"context,omitempty"`
}

func SaveNewConversation(response, message, context string, logger *log.Logger) error {
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
		Context:   context,
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

	// ファイルパスを設定
	conversation.FilePath = filename

	// ファイルパスを含めて再度保存
	data, err = json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal conversation with file path: %w", err)
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save conversation with file path: %w", err)
	}

	logger.Debug("Saved conversation", "id", conversation.ID, "path", filename)
	return nil
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
	content := fmt.Sprintf("# Conversation %s\n\n## User\n%s\n\n## AI\n%s",
		conv.ID, conv.Message, conv.Response)

	glowCmd.Stdin = strings.NewReader(content)
	glowCmd.Stdout = os.Stdout
	glowCmd.Stderr = os.Stderr
	if err := glowCmd.Run(); err != nil {
		return fmt.Errorf("failed to execute glow: %w", err)
	}
	return nil
}

func StartNewConversation(message string, usePerplexity bool, logger *log.Logger) error {
	// Load context if exists
	context, err := LoadContext(logger)
	if err != nil {
		logger.Error("Failed to load context", "error", err)
		return err
	}

	// Prepend context to message if it exists (only for sgpt)
	var fullMessage string
	if !usePerplexity && context != "" {
		fullMessage = fmt.Sprintf("# Context\n%s\n\n# Question\n%s", context, message)
	} else {
		fullMessage = message
	}

	// Execute AI command based on provider
	var aiCmd *exec.Cmd
	if usePerplexity {
		aiCmd = exec.Command("perplexity", "-g", "--stream", "--citation", fullMessage)
	} else {
		aiCmd = exec.Command("sgpt", "--stream", fullMessage)
	}
	stdout, err := aiCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	aiCmd.Stderr = os.Stderr

	if err := aiCmd.Start(); err != nil {
		return fmt.Errorf("failed to start AI command: %w", err)
	}

	// Check if style file exists
	shareDir, err := config.GetShareDir()
	if err != nil {
		return fmt.Errorf("failed to get share directory: %w", err)
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

	const HELD_OUT_LINE_COUNT = 4
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				if err != io.EOF {
					return fmt.Errorf("error reading AI output: %w", err)
				}
				// Stream is closed (EOF)
				// break
			}
			// No more data and no error (EOF)
			previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
			for i := max(0, len(previousGlowOutputLines)-HELD_OUT_LINE_COUNT); i < len(previousGlowOutputLines); i++ {
				fmt.Println(previousGlowOutputLines[i])
			}
			// Trim excessive trailing newlines before saving
			response := strings.TrimRightFunc(buffer.String(), func(r rune) bool {
				return r == '\n' || r == '\r'
			})
			if err := SaveNewConversation(response, message, context, logger); err != nil {
				return fmt.Errorf("failed to save conversation: %w", err)
			}
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
			return fmt.Errorf("failed to execute glow: %w", err)
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
		return fmt.Errorf("AI command failed: %w", err)
	}

	return nil
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
