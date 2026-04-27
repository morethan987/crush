# Config Package

**Stats:** 23 Go files, ~1800 LoC (config.go: 661, load.go: 840, provider.go: 231)

## Overview

Configuration service for Crush. Loads from multiple sources (global + workspace),
manages providers, resolves variables, and exposes a `ConfigStore` as the single
entry point for all config access.

## Architecture

### ConfigStore (store.go)

The sole owner of config state. Constructed by `Load()` in `load.go`.

```go
store := config.Load(workingDir, dataDir, debug)
store.Config()          // *Config (read-only)
store.WorkingDir()      // Current working directory
store.KnownProviders()  // From Catwalk
```

Mutation methods persist changes to disk immediately:
- `UpdatePreferredModel(scope, modelType, model)` — changes model + writes JSON
- `SetProviderAPIKey(scope, providerID, key)` — sets key + writes JSON
- `SetConfigField(scope, key, value)` — generic JSON field setter
- `SetCompactMode(scope, enabled)` — TUI setting

### Config Loading (load.go)

Priority order (later overrides earlier):
1. Embedded defaults (from Catwalk)
2. Global config: `~/.config/crush/crush.json`
3. Workspace config: `.crush.json` or `crush.json` in project root

Uses `go-jsons` for deep merging of JSON configs.

### Scopes (scope.go)

- `ScopeGlobal` — `~/.local/share/crush/crush.json` (user-wide)
- `ScopeWorkspace` — `.crush/crush.json` (project-specific)

## Key Types

### Config (config.go)

Top-level struct. Fields:

- `Models` — `map[SelectedModelType]SelectedModel` (large/small)
- `Providers` — `*csync.Map[string, ProviderConfig]` (thread-safe)
- `MCP` — MCP server configs (stdio/http/sse)
- `LSP` — LSP server configs
- `Options` — Global settings (context paths, tools, permissions, etc.)
- `Permissions` — Tool allow-lists
- `Agents` — Agent definitions (coder, task, custom)

### ProviderConfig

Per-provider settings: ID, Name, BaseURL, Type, APIKey, Models, OAuth, etc.
Method `ToProvider()` converts to `catwalk.Provider`.

### SelectedModel

Model selection with overrides: Provider, Model, MaxTokens, Temperature,
TopP, TopK, FrequencyPenalty, PresencePenalty.

## Variable Resolution (resolve.go)

Shell-style variable expansion in config values:
- `$VAR` / `${VAR}` — environment variable lookup
- `$(command)` — command substitution (5min timeout)

Two resolvers:
- `ShellVariableResolver` — supports both env vars and command substitution
- `EnvironmentVariableResolver` — env vars only

Access via `store.Resolve(key)` or `store.Resolver()`.

## Provider Management (provider.go)

- Auto-updates from Catwalk (`https://catwalk.charm.sh`)
- `ResetProviders()` / `Providers()` for test mock injection
- `UseMockProviders` flag for testing

## File Organization

| File | Purpose |
|------|---------|
| `config.go` | Core types: Config, ProviderConfig, MCPConfig, Options, etc. |
| `store.go` | ConfigStore: mutation, persistence, OAuth refresh |
| `load.go` | Config loading, merging, defaults, JSON schema generation |
| `provider.go` | Provider auto-updates, Catwalk integration |
| `resolve.go` | Variable resolution (env vars, command substitution) |
| `scope.go` | Scope enum (Global vs Workspace) |
| `init.go` | Initialization/copying logic |
| `catwalk.go` | Catwalk API client |
| `copilot.go` | GitHub Copilot provider setup |
| `hyper.go` | Hyper provider setup |
| `docker_mcp.go` | Docker MCP host detection |

## Conventions

- **Config is a Service**: Access via `ConfigStore` methods, not global vars.
- **Thread-safe providers**: `csync.Map` for concurrent provider access.
- **Immutable after load**: `Config()` returns a pointer but treat as read-only.
- **JSON schema**: Use `jsonschema` tags for auto-generated schema.
- **Test with mocks**: Set `UseMockProviders = true` before tests.
- **Scope-aware writes**: Always pass `Scope` when persisting changes.
