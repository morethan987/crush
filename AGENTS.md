# Crush Development Guide

**Commit:** 8b30fad1
**Branch:** main
**Stats:** 285 Go files, ~60k LoC

## Overview

Terminal-based AI coding assistant (Go, Charm ecosystem). Multi-provider LLM
gateway with tool execution, LSP integration, MCP extensibility, and Bubble Tea
v2 TUI. Module path: `github.com/charmbracelet/crush`.

## Structure

```
main.go                          CLI entry → internal/cmd
internal/
  app/app.go                       Top-level wiring: DB, config, agents, LSP, MCP, events
  cmd/                             CLI commands (root, run, login, models, stats, sessions)
  config/
    config.go                      Config struct, context file paths, agent definitions
    load.go                        crush.json loading and validation
    provider.go                    Provider configuration and model resolution
  agent/
    agent.go                       SessionAgent: runs LLM conversations per session
    coordinator.go                 Coordinator: manages named agents ("coder", "task")
    hooked_tool.go                 Decorator that runs PreToolUse hooks before tool execution
    prompts.go                     Loads Go-template system prompts
    templates/                     System prompt templates (coder.md.tpl, task.md.tpl, etc.)
    tools/                         All built-in tools (bash, edit, view, grep, glob, etc.)
      mcp/                         MCP client integration
  hooks/                           Hook engine: runs user shell commands on hook events
    hooks.go                       Decision types, aggregation logic, event constants
    runner.go                      Parallel hook execution, timeout, dedup
    input.go                       Stdin payload builder, env vars, stdout parsing (Crush + Claude Code compat)
  session/session.go               Session CRUD backed by SQLite
  message/                         Message model and content types
  db/                              SQLite via sqlc, with migrations
    sql/                           Raw SQL queries (consumed by sqlc)
    migrations/                    Schema migrations
  lsp/                             LSP client manager, auto-discovery, on-demand startup
  ui/                              Bubble Tea v2 TUI (see internal/ui/AGENTS.md)
  permission/                      Tool permission checking and allow-lists
  skills/                          Skill file discovery and loading
  shell/                           Bash command execution with background job support
  event/                           Telemetry (PostHog)
  pubsub/                          Internal pub/sub for cross-component messaging
  filetracker/                     Tracks files touched per session
  history/                         Prompt history
```

## Where to Look

| Task | Location | Notes |
|------|----------|-------|
| Add a new tool | `internal/agent/tools/` | Create .go + .md pair, register in buildTools() |
| Change system prompt | `internal/agent/templates/` | Go templates with runtime injection |
| Add CLI command | `internal/cmd/` | Cobra command pattern |
| Change config schema | `internal/config/config.go` | Config struct + JSON schema |
| Add provider | `internal/config/` | ProviderConfig + provider file |
| DB schema change | `internal/db/sql/` + `migrations/` | Write SQL → run sqlc generate |
| TUI component | `internal/ui/` | See internal/ui/AGENTS.md |
| Chat message rendering | `internal/ui/chat/` | Tool renderers by file |
| MCP integration | `internal/agent/tools/mcp/` | stdio/http/sse transports |

## Key Dependencies

- **Config is a Service**: accessed via `config.Service`, not global state.
- **Tools are self-documenting**: each tool has a `.go` implementation and a
  `.md` description file in `internal/agent/tools/`.
- **System prompts are Go templates**: `internal/agent/templates/*.md.tpl`
  with runtime data injected.
- **Context files**: Crush reads AGENTS.md, CRUSH.md, CLAUDE.md, GEMINI.md
  (and `.local` variants) from the working directory for project-specific
  instructions.
- **Persistence**: SQLite + sqlc. All queries live in `internal/db/sql/`,
  generated code in `internal/db/`. Migrations in `internal/db/migrations/`.
- **Pub/sub**: `internal/pubsub` for decoupled communication between agent,
  UI, and services.
- **Hooks**: User-defined shell commands in `crush.json` that fire before
  tool execution. The engine (`internal/hooks/`) is independent of fantasy
  and agent — it takes inputs, runs commands, returns decisions. The
  `hookedTool` decorator in `internal/agent/hooked_tool.go` wraps tools at
  the coordinator level. Hooks run before permission checks. See
  `HOOKS.md` for the user-facing protocol.
- **CGO disabled**: builds with `CGO_ENABLED=0` and
  `GOEXPERIMENT=greenteagc`.

## Key Patterns

- **Config is a Service**: `config.Service`, not global state.
- **Tools are self-documenting**: each tool has `.go` + `.md` in `internal/agent/tools/`.
- **System prompts are Go templates**: `templates/*.md.tpl` with runtime data.
- **Context files**: Reads AGENTS.md, CRUSH.md, CLAUDE.md, GEMINI.md (+ `.local` variants).
- **Persistence**: SQLite + sqlc. SQL in `db/sql/`, generated code in `db/`.
- **Pub/sub**: `internal/pubsub` for agent ↔ UI ↔ services.
- **CGO disabled**: `CGO_ENABLED=0`, `GOEXPERIMENT=greenteagc`.
- **Centralized UI model**: `UI` is the sole Bubble Tea model; sub-components are imperative structs.

## Commands

```bash
go build .              # Build
task test               # Test (or go test ./...)
task lint:fix           # Lint
task fmt                # Format (gofumpt -w .)
task modernize          # Code simplifications
task dev                # Dev mode with profiling
go test ./... -update   # Regenerate golden files
```

## Code Style Guidelines

- **Imports**: Use `goimports` formatting, group stdlib, external, internal
  packages.
- **Formatting**: Use gofumpt (stricter than gofmt), enabled in
  golangci-lint.
- **Naming**: Standard Go conventions — PascalCase for exported, camelCase
  for unexported.
- **Types**: Prefer explicit types, use type aliases for clarity (e.g.,
  `type AgentName string`).
- **Error handling**: Return errors explicitly, use `fmt.Errorf` for
  wrapping.
- **Context**: Always pass `context.Context` as first parameter for
  operations.
- **Interfaces**: Define interfaces in consuming packages, keep them small
  and focused.
- **Structs**: Use struct embedding for composition, group related fields.
- **Constants**: Use typed constants with iota for enums, group in const
  blocks.
- **Testing**: Use testify's `require` package, parallel tests with
  `t.Parallel()`, `t.SetEnv()` to set environment variables. Always use
  `t.Tempdir()` when in need of a temporary directory. This directory does
  not need to be removed.
- **JSON tags**: Use snake_case for JSON field names.
- **File permissions**: Use octal notation (0o755, 0o644) for file
  permissions.
- **Log messages**: Log messages must start with a capital letter (e.g.,
  "Failed to save session" not "failed to save session").
  - This is enforced by `task lint:log` which runs as part of `task lint`.
- **Comments**: End comments in periods unless comments are at the end of the
  line.

## Testing with Mock Providers

When writing tests that involve provider configurations, use the mock
providers to avoid API calls:

```go
func TestYourFunction(t *testing.T) {
    // Enable mock providers for testing
    originalUseMock := config.UseMockProviders
    config.UseMockProviders = true
    defer func() {
        config.UseMockProviders = originalUseMock
        config.ResetProviders()
    }()

    // Reset providers to ensure fresh mock data
    config.ResetProviders()

    // Your test code here - providers will now return mock data
    providers := config.Providers()
    // ... test logic
}
```

## Formatting

- ALWAYS format any Go code you write.
  - First, try `gofumpt -w .`.
  - If `gofumpt` is not available, use `goimports`.
  - If `goimports` is not available, use `gofmt`.
  - You can also use `task fmt` to run `gofumpt -w .` on the entire project,
    as long as `gofumpt` is on the `PATH`.

## Comments

- Comments that live on their own lines should start with capital letters and
  end with periods. Wrap comments at 78 columns.

## Committing

- ALWAYS use semantic commits (`fix:`, `feat:`, `chore:`, `refactor:`,
  `docs:`, `sec:`, etc).
- Try to keep commits to one line, not including your attribution. Only use
  multi-line commits when additional context is truly necessary.

## Subdirectory Guides

| Path | Focus |
|------|-------|
| `internal/ui/AGENTS.md` | TUI architecture, rendering pipeline, component patterns |
| `internal/agent/tools/AGENTS.md` | Tool system, registration, creating new tools |
| `internal/config/AGENTS.md` | Config service, provider management, model resolution |
