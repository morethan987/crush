# Agent Tools

**Stats:** 34 Go files, 20 .md tool descriptions + 6 MCP files

## Overview

Self-documenting tool system for the LLM agent. Each tool is a `.go` +
`.md` pair: the Go file implements `fantasy.AgentTool`, the markdown file
provides the LLM-visible description.

## Tool Registration

Tools are constructed in `coordinator.buildTools()` (in parent `agent/`
package). Each `NewXxxTool()` returns a `fantasy.AgentTool`. Tools are
filtered by `agent.AllowedTools` before being passed to the session agent.

## Creating a New Tool

1. Create `your_tool.go` with `func NewYourTool(deps...) fantasy.AgentTool`
2. Create `your_tool.md` with the tool description (LLM-visible)
3. Use `fantasy.NewAgentTool(name, description, handler)` to construct
4. Add the `NewYourTool()` call in `coordinator.buildTools()` (in
   `internal/agent/coordinator.go`)
5. Add tool name to `allToolNames` in `internal/config/config.go` so it
   appears in the allow-list

### Handler Pattern

```go
func NewYourTool(deps Deps) fantasy.AgentTool {
    return fantasy.NewAgentTool(
        "your_tool",
        string(yourToolDescription),
        func(ctx context.Context, params YourParams, call fantasy.ToolCall) (fantasy.ToolResponse, error) {
            // ... implementation
            return fantasy.NewTextResponse(result), nil
        },
    )
}
```

Parameters are auto-deserialized from JSON via the params struct. Use
`fantasy.NewTextResponse()` for success, `fantasy.NewTextErrorResponse()`
for errors.

## Built-in Tools

| Tool | File | Purpose |
|------|------|---------|
| Bash | `bash.go` | Shell execution with background jobs, safety blocklist |
| Edit | `edit.go` | Single-file text replacement with LSP integration |
| MultiEdit | `multiedit.go` | Multiple edits to one file in one operation |
| Write | `write.go` | Create or overwrite files |
| View | `view.go` | Read files with line numbers, LSP symbol lookup |
| Glob | `glob.go` | File pattern matching (find by name) |
| Grep | `grep.go` | Content search with regex, file filtering |
| LS | `ls.go` | Directory tree listing |
| Fetch | `fetch.go` | Download files from URLs |
| Download | `download.go` | Download binary data to local file |
| WebFetch | `web_fetch.go` | Fetch web content → markdown/text |
| WebSearch | `web_search.go` | Web search via DuckDuckGo |
| Sourcegraph | `sourcegraph.go` | Code search across public repos |
| Diagnostics | `diagnostics.go` | LSP diagnostics for files/directories |
| References | `references.go` | LSP symbol reference lookup |
| LSPRestart | `lsp_restart.go` | Restart LSP servers |
| Todos | `todos.go` | Task list management |
| JobOutput | `job_output.go` | Background shell job output |
| JobKill | `job_kill.go` | Kill background shell jobs |
| ListMCPResources | `list_mcp_resources.go` | List MCP server resources |
| ReadMCPResource | `read_mcp_resource.go` | Read MCP server resources |

## MCP Tools (`mcp/`)

Dynamic tools discovered from MCP servers at runtime. `mcp/tools.go`
calls `GetMCPTools()` which connects to configured MCP servers (stdio/http/sse)
and wraps their tools as `fantasy.AgentTool` instances.

## Context Keys

Tool handlers receive context with injected values via `tools.go`:

- `SessionIDContextKey` — current session ID
- `MessageIDContextKey` — current message ID
- `SupportsImagesContextKey` — model image support flag
- `ModelNameContextKey` — current model name

## Permission Model

Write/execute tools call `permissions.Request()` to prompt the user for
approval. Read-only tools (Glob, Grep, View, LS) skip permission checks
but still respect `.gitignore` and `.crushignore`.

## Conventions

- Tool descriptions in `.md` files are the LLM-visible docs — keep them
  clear and include parameter descriptions.
- Use `cmp.Or(param.WorkingDir, defaultWorkingDir)` for working directory.
- Return `fantasy.ToolResponse` with text content, not structured JSON.
- Background jobs are managed via the `shell` package, not tools directly.
- The `safe` package provides command safety classification for Bash.
