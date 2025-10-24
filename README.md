# ASC (AI Shell Chat)

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
- **glow** - For markdown rendering and display
- **sgpt** - Default AI provider (streaming mode)
- **perplexity** (optional) - Alternative AI provider

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