# ASC (AI Shell Chat)

> **Pronunciation:** `asc` is read as "ask".

A command-line tool for interacting with AI. Have natural conversations with AI to perform tasks or get information.

## Features

- Natural conversation on the command line
- Multiple AI provider support (sgpt and perplexity)
- Simple and intuitive interface
- Detailed logging with debug mode
- Short command aliases for quick access
- Conversation history tracking
- Follow-up questions support

## Installation

### Prerequisites

The following external commands are required:
- **[glow](https://github.com/charmbracelet/glow)** - For markdown rendering and display
- **[sgpt](https://github.com/tbckr/sgpt)** - Default AI provider (streaming mode)
- **[perplexity](https://github.com/mkasa/perplexity-cli)** (optional) - Alternative AI provider

### From Source

1. Make sure you have Go 1.21 or later installed
2. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/asc.git
   cd asc
   ```
3. Build and install:
   ```bash
   make install
   ```

### Uninstall

To uninstall the program:
```bash
make uninstall
```

## Project Structure

```
.
├── cmd/        # Main application
│   └── asc.go  # Main entry point
├── internal/   # Private application and library code
├── pkg/        # Public library code
└── go.mod      # Module definition
```

## Development Setup

1. Install Go 1.21 or later
2. Clone the repository
3. Run `go mod tidy` to install dependencies

## Usage

### New Conversation Mode
```bash
# Start a new conversation (uses sgpt by default)
asc new "Tell me about Go"

# Using the short alias
asc n "Tell me about Go"

# Use perplexity instead of sgpt
asc new -p "Tell me about Go"
asc new --perplexity "Tell me about Go"
```

### Continue Previous Conversation
```bash
# Add a follow-up question (uses sgpt by default)
asc append "Can you explain more about that?"

# Using the short alias
asc a "What else should I know?"

# Use perplexity for follow-up
asc append -p "Can you explain more about that?"
```

### Edit Previous Message
```bash
# Edit and resend the last message
asc edit

# Edit with perplexity
asc edit -p
```

### Interactive Mode
```bash
# Start a fresh interactive, multi-round chat
asc interactive "Tell me about Go"

# Using the short alias
asc i "Tell me about Go"

# Resume the most recent conversation and keep chatting
asc i

# Pick which conversation to resume from an interactive list
asc i -P
asc i --pick

# Use perplexity instead of sgpt for the session
asc i -p "Tell me about Go"
```

`asc i` (alias for `asc interactive`) opens a back-and-forth chat session:

1. You're prompted at a `you> ` prompt for a message.
2. The reply is streamed in real time (rendered through `glow`).
3. You're prompted again, repeating for as many rounds as you like.

Each turn re-sends the accumulated transcript, so the AI keeps the full context
of the conversation. The conversation is **saved after every turn**, so you never
lose progress.

**Starting vs. resuming:**
- If you pass a message (`asc i "..."`), a new conversation starts with it as the
  first turn.
- If you pass no message (`asc i`), the most recent conversation is resumed and
  continued. If there is no previous conversation, a fresh one is started.
- With `-P`/`--pick` (`asc i -P`), an interactive list of your saved
  conversations opens so you can choose exactly which one to resume — use the
  arrow keys to move, Enter to resume, and `q`/Esc to cancel. A message given
  alongside `--pick` is sent as the next turn of the chosen conversation.

**Ending the session:**
- Type `/exit` or `/quit` and press Enter.
- Or press `Ctrl-D` (EOF).

In all cases the session prints `Goodbye.` and exits cleanly.

**Context:** As with `new` and `append`, the context file is prepended to each
message when using `sgpt` (the default). Context is **not** prepended when using
perplexity (`-p`).

### View History
```bash
# View conversation history
asc view

# Using the short alias
asc v
```

### Other Commands
```bash
# Show version information
asc version

# Show help
asc --help

# Run in debug mode
asc --debug new "test message"
```

## AI Providers

ASC supports two AI providers:

### sgpt (Default)
- Requires the `sgpt` command to be installed
- Supports streaming output for real-time responses
- Supports context prepending for additional information
- Usage: `asc new "your question"`

### perplexity
- Requires the `perplexity` command to be installed
- Activated with `-p` or `--perplexity` flag
- Takes only the query message (no context prepending)
- Usage: `asc new -p "your question"`

The application will check for the appropriate AI provider command at startup based on the flags provided.

## License

MIT License 