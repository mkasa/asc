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
)

type Conversation struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Response  string    `json:"response"`
	FilePath  string    `json:"file_path"`
}

func SaveNewConversation(response, message string, logger *log.Logger) error {
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

func ShowConversation(conv Conversation, logger *log.Logger) error {
	// Execute glow command with conversation content
	glowCmd := exec.Command("glow", "-p")

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

func StartNewConversation(message string, logger *log.Logger) error {
	// Execute sgpt with --stream option
	sgptCmd := exec.Command("sgpt", "--stream", message)
	stdout, err := sgptCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	sgptCmd.Stderr = os.Stderr

	if err := sgptCmd.Start(); err != nil {
		return fmt.Errorf("failed to start sgpt: %w", err)
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

	const HELD_OUT_LINE_COUNT = 3
	for {
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				if err != io.EOF {
					return fmt.Errorf("error reading sgpt output: %w", err)
				}
				// Stream is closed (EOF)
				// break
			}
			// No more data and no error (EOF)
			previousGlowOutputLines := strings.Split(previousGlowOutput, "\n")
			for i := max(0, len(previousGlowOutputLines)-HELD_OUT_LINE_COUNT); i < len(previousGlowOutputLines); i++ {
				fmt.Println(previousGlowOutputLines[i])
			}
			if err := SaveNewConversation(buffer.String(), message, logger); err != nil {
				return fmt.Errorf("failed to save conversation: %w", err)
			}
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

	if err := sgptCmd.Wait(); err != nil {
		return fmt.Errorf("sgpt command failed: %w", err)
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
