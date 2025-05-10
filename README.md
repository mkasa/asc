# ASC (AI Shell Chat)

A command-line tool for interacting with AI. Have natural conversations with AI to perform tasks or get information.

## Features

- Natural conversation on the command line
- Simple and intuitive interface
- Detailed logging with debug mode
- Short command aliases for quick access
- Conversation history tracking
- Follow-up questions support

## Installation

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
# Start a new conversation
asc new

# Start with a specific message
asc new "Hello"

# Using the short alias
asc n "Tell me about Go"
```

### Continue Previous Conversation
```bash
# Add a follow-up question
asc append "Can you explain more about that?"

# Using the short alias
asc a "What else should I know?"
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
asc --debug new
```

## License

MIT License 