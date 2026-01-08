# LazyBeads

A TUI (Terminal User Interface) for managing beads issues, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Project Structure

```
lazybeads/
├── main.go              # Entry point, CLI flags, initialization
├── internal/
│   ├── app/             # Main Bubble Tea application model
│   ├── beads/           # Client wrapper for bd CLI
│   ├── models/          # Shared data models
│   └── ui/              # Reusable UI components
└── .beads/              # Issue tracking (managed by bd)
```

## Development

### Prerequisites
- Go 1.25+
- `bd` CLI installed and available in PATH

### Build and Install
```bash
# Build locally
go build .

# Install globally (use this to test the app)
go install .
```

### Validation
```bash
# Run headless validation to verify bd integration works
lazybeads --check
```

## Issue Tracking

This project uses **beads** (`bd`) for issue tracking. All work should be tracked through beads.

### Common Commands
```bash
bd ready                  # Find work ready to start
bd show <id>              # View issue details
bd update <id> --claim    # Claim and start work
bd close <id>             # Complete work
bd sync --from-main       # Sync beads from main branch
```

### Workflow
1. Find available work with `bd ready`
2. Claim an issue before starting
3. Implement, test, commit
4. Close the issue when done
5. Sync and commit before ending session
