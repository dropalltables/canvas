# Canvas CLI - Agent Guidelines

## Build & Test
```bash
go build -o canvas .              # Build binary
go test ./...                     # Run all tests
go test ./internal/api -v         # Single package
go test -run TestName ./package   # Single test
go fmt ./...                      # Format code
go vet ./...                      # Lint
```

## Architecture
- **cmd/**: Cobra CLI command handlers (root, auth, assignments)
- **internal/api/**: Canvas API client with HTTP methods, Assignment/Plannable structs
- **internal/config/**: Config file I/O (~/.config/canvas/config.json), env vars support
- **internal/tui/**: Terminal UI using Bubble Tea
- **internal/ui/**: Simple output formatting (table, JSON, compact)
- No database; state from Canvas API only

## Code Style
- **Imports**: stdlib first, then github.com packages, one blank line between groups
- **Naming**: Exported=CamelCase, unexported=camelCase, vars use full names (not abbreviations)
- **Error handling**: Return error as last value, check with `if err != nil`, expose custom error types (e.g., APIError)
- **Formatting**: `go fmt`, 100-char soft limit, comments above code
- **Structs**: JSON tags use snake_case, pointer receivers for methods modifying state
- **Time**: Use `time.Time` with JSON marshaling, UTC/RFC3339 preferred
