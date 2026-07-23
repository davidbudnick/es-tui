# CLAUDE.md

## Project Overview

Elasticsearch/OpenSearch TUI Manager — a terminal user interface for managing Elasticsearch and OpenSearch clusters, built with Go and Bubble Tea. Alternative to curl/Dev Tools with interactive browsing, search, and cluster monitoring.

## Build & Test Commands

```bash
make build              # Build binary to bin/es-tui
make test               # Run tests: go test -v -race ./...
make test-cover         # Tests with coverage report
make test-cover-check   # ≥80% overall; 100% on cmd/types/service
make lint               # Run go vet
make fmt                # Format code
make run                # Build and run
make docker-up          # Start ES (:9200) and OpenSearch (:9201)
make docker-seed        # Seed demo indices on both
make docker-down        # Stop demo clusters
make demo               # Render README GIF via vhs (needs Docker + vhs)
```

CI runs lint, `go test -race` with coverage, multi-OS build, and GoReleaser snapshot. Coverage floor is **≥80%** overall (examples excluded); packages `internal/cmd`, `internal/types`, and `internal/service` must stay at **100%**.

## Architecture

```
main.go                    # Entry point, CLI flag parsing, config init
internal/
  cmd/                     # Bubble Tea command factories (return tea.Cmd)
  ui/                      # Bubble Tea UI (Model/Update/View pattern)
  es/                      # Elasticsearch/OpenSearch HTTP client
  types/                   # Shared type definitions and messages
  db/                      # Config persistence (~/.config/es-tui/config.json)
  service/                 # Interfaces (ConfigService, ESService) and DI container
  testutil/                # Test helpers and mock implementations
```

**Bubble Tea message flow**: KeyMsg → handleKeyPress() → Command (tea.Cmd) → async execution → Message (tea.Msg) → Update() → View()

## Code Conventions

- **Go version**: 1.26.5 (set in go.mod)
- **Package names**: lowercase, single word
- **Receivers**: short names (`c *Client`, `m Model`, `cfg *Config`)
- **Message types**: PascalCase with `Msg` suffix
- **Command methods**: return `tea.Cmd`
- **Error handling**: wrap with `fmt.Errorf("context: %w", err)`
- **Logging**: `log/slog` structured logging
- **Dependency injection**: services via `Commands{config, es}`

## Testing

- Standard `testing` package
- ES client tests use `net/http/httptest`
- Mock ES in `internal/testutil/mock_es.go`
- Helpers: `AssertEqual`, `AssertNoError`, `AssertError`
- **Never suppress errors** in tests
- Config persistence must round-trip via reload

## Key Dependencies

- `charm.land/bubbletea/v2` — TUI framework
- `charm.land/bubbles/v2` — TUI components
- `charm.land/lipgloss/v2` — Terminal styling

## Git Conventions

- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `chore:`
- Subject ≤50 chars, imperative mood
- Net new changes ship via PR — never push directly to main

## Guardrails

- All ES/OS operations go through `internal/es/` — no raw HTTP in UI/cmd
- Password and API key stripping on config save
- New commands go through `Commands` with injected services
- Message types use `Msg` suffix in `types/messages.go`
