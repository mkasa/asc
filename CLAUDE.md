# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Installation
- `make build` - Build the asc binary
- `make install` - Install the application to ~/.local/bin and assets to ~/.local/share/asc
- `make uninstall` - Remove the application and assets
- `make clean` - Clean build artifacts
- `go mod tidy` - Install and clean up dependencies

### Development
- `go run ./cmd/asc.go` - Run the application directly during development
- Use `--debug` flag for detailed logging: `asc --debug new "test message"`

### Dependencies
The application requires external commands to be installed:
- `glow` - For markdown rendering and display
- `sgpt` - For AI interaction (streaming mode) - default provider
- `perplexity` - Alternative AI provider (optional, use with `--perplexity` or `-p` flag)

## Architecture

### Core Structure
This is a Go CLI application using Cobra for command handling and Bubble Tea for TUI components.

**Main Components:**
- `cmd/asc.go` - Main entry point with Cobra command definitions
- `internal/conversation/` - Conversation management (save, load, display)
- `internal/view/` - Interactive TUI for viewing conversation history  
- `internal/config/` - Configuration and path management

### Key Design Patterns

**Data Flow:**
1. User input → Cobra commands → Internal packages
2. Messages sent to AI provider (`sgpt` by default, or `perplexity` with `-p` flag)
3. Responses processed through `glow` for real-time markdown rendering
4. Conversations saved as JSON files in data directory

**File Storage:**
- Conversations: `~/.local/share/asc/data/conversations/` (JSON files)
- Context: `~/.local/share/asc/context.txt` 
- Assets: `~/.local/share/asc/` (glow style files)

**Real-time Streaming:**
The app streams AI responses in real-time by:
1. Piping AI provider output through a scanner
2. Buffering accumulated content
3. Rendering through glow with each new line
4. Managing display with held-out lines to prevent flicker

**AI Provider Selection:**
- Default: `sgpt --stream <message>` with context prepending
- With `-p/--perplexity` flag: `perplexity <message>` (no context prepending)
- Provider is checked at startup to ensure availability

### Command Architecture
- `new` (alias: `n`) - Start new conversation with context prepending
  - Supports `-p/--perplexity` flag to use perplexity instead of sgpt
- `append` (alias: `a`) - Continue previous conversation with context
  - Supports `-p/--perplexity` flag to use perplexity instead of sgpt
- `edit` (alias: `e`) - Edit and resend previous message using $EDITOR
  - Supports `-p/--perplexity` flag to use perplexity instead of sgpt
- `view` (alias: `v`) - Interactive table view of conversation history
- `context` (alias: `c`) - Edit context file that gets prepended to all messages
- `clear` - Remove context file

### TUI Components
The view command uses Bubble Tea with:
- Table widget for conversation listing
- Dynamic column width calculation based on terminal size
- Keybindings: v (glow), V (less), e (edit), d (delete), q (quit)
- Confirmation dialogs for destructive actions

### Context System
Context is prepended to new conversations when using sgpt (default) in format:
```
# Context
[context content]

# Question
[user message]
```
Note: Context prepending is disabled when using perplexity (`-p` flag), as perplexity only accepts the query message.

### Logging
Uses charmbracelet/log with configurable levels (Info/Debug) and structured logging throughout.