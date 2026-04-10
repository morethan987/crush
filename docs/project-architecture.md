# Crush 项目架构指南

> 本文档帮助你快速理解 Crush 项目的层次结构和核心设计，以便尽快上手开发。

## 1. 项目概览

Crush 是一个基于终端的 AI 编程助手，使用 Go 语言编写，构建在 [Charm 生态系统](https://charm.land) 之上。它提供多模型 LLM 网关、工具执行、LSP 集成、MCP 扩展，以及基于 Bubble Tea v2 的 TUI 界面。

**技术栈**：Go 1.26 · SQLite + sqlc · Bubble Tea v2 · Charm Fantasy（LLM 抽象层）

**模块路径**：`github.com/charmbracelet/crush`

**规模**：约 285 个 Go 文件，约 60k 行代码

---

## 2. 顶层目录结构

```
crush/
├── main.go                  # CLI 入口，调用 internal/cmd
├── go.mod / go.sum          # Go 模块定义
├── sqlc.yaml                # sqlc 代码生成配置（SQL → Go）
├── Taskfile.yaml            # Task 任务运行器（构建/测试/格式化）
├── schema.json              # 自动生成的配置 JSON Schema
├── crush.json               # 项目级配置文件
├── scripts/                 # 构建/CI 辅助脚本
├── .golangci.yml            # golangci-lint 配置
├── .goreleaser.yml          # GoReleaser 发布配置
├── AGENTS.md                # AI Agent 开发指南（项目上下文文件）
├── README.md                # 项目说明
├── LICENSE.md               # FSL-1.1-MIT 许可证
├── docs/                    # 文档目录
│   └── project-architecture.md  # ← 你正在看的这个文件
└── internal/                # 所有核心代码（Go 标准的 internal 包）
```

---

## 3. `internal/` 包层次结构

```
internal/
│
├── app/                     # 🔌 应用组装层 — 连接所有服务
├── cmd/                     # ⌨️ CLI 命令（Cobra）
├── agent/                   # 🤖 LLM Agent 核心
│   ├── tools/               #   内置工具 + MCP
│   │   └── mcp/             #     MCP 客户端集成
│   ├── templates/           #   Go 模板系统提示词
│   ├── prompt/              #   提示词辅助
│   ├── notify/              #   Agent 通知
│   └── hyper/               #   Hyper 提供者
│
├── config/                  # ⚙️ 配置服务
├── db/                      # 💾 数据持久层（SQLite + sqlc）
│   ├── sql/                 #   原始 SQL 查询
│   └── migrations/          #   数据库迁移
│
├── ui/                      # 🎨 TUI 界面（Bubble Tea v2）
│   ├── model/               #   主 UI 模型
│   ├── chat/                #   聊天消息渲染器
│   ├── dialog/              #   对话框系统
│   ├── list/                #   通用滚动列表
│   ├── common/              #   共享上下文和工具
│   ├── styles/              #   样式定义
│   ├── completions/         #   自动补全
│   ├── attachments/         #   文件附件管理
│   ├── diffview/            #   Diff 渲染
│   ├── anim/                #   动画（Spinner）
│   ├── image/               #   终端图片渲染
│   ├── logo/                #   Logo 渲染
│   └── util/                #   UI 工具函数
│
├── lsp/                     # 🔍 LSP 客户端管理器
├── session/                 # 📋 Session CRUD
├── message/                 # 💬 消息模型和内容类型
├── shell/                   # 🐚 Bash 执行 + 后台任务
├── permission/              # 🔐 工具权限管理
├── skills/                  # 📦 Agent Skills 发现/加载
│   └── builtin/             #   内置 Skills
│
├── pubsub/                  # 📡 内部发布/订阅
├── event/                   # 📊 遥测（PostHog）
├── fsext/                   # 📁 文件系统扩展
├── csync/                   # 🔄 并发同步原语
├── server/                  # 🌐 Unix Socket API 服务端
├── client/                  # 📡 Unix Socket API 客户端
├── backend/                 # 🔗 后端抽象层
├── workspace/               # 🗂️ 工作区抽象
├── proto/                   # 📜 Client-Server 协议定义
├── diff/                    # 🔀 Diff 算法
├── config/                  # ⚙️ 配置服务
│
├── history/                 # 📜 命令历史
├── filetracker/             # 👁️ 文件变更追踪
├── projects/                # 📂 项目管理
├── commands/                # 🎯 Slash 命令
├── oauth/                   # 🔑 OAuth 认证
│   ├── copilot/             #   GitHub Copilot OAuth
│   └── hyper/               #   Hyper OAuth
│
├── format/                  # ✨ 格式化工具
├── update/                  # 🔄 自动更新检查
├── version/                 # 📌 版本信息
├── home/                    # 🏠 用户目录工具
├── log/                     # 📝 日志工具
├── env/                     # 🌍 环境变量工具
├── ansiext/                 # 🎨 ANSI 扩展工具(将不可见控制字符可视化)
├── stringext/               # 🔤 字符串扩展工具
├── filepathext/             # 📎 文件路径扩展工具
└── swagger/                 # 📖 OpenAPI/Swagger 生成
```

---

## 4. 核心架构分层

### 4.1 应用入口与组装层

```
main.go → internal/cmd/root.go → internal/app/app.go
```

| 文件 | 职责 |
|------|------|
| `main.go` | 最简入口，启动 pprof（可选），调用 `cmd.Execute()` |
| `internal/cmd/` | Cobra CLI 命令定义（root、run、logs、models、schema 等） |
| `internal/app/app.go` | **核心组装器** — 实例化并连接所有服务（DB、Config、Agent、LSP、MCP、Events） |

`app.go` 中的 `App` 结构体是整个应用的中枢：

```go
type App struct {
    Sessions          session.Service      // Session 管理
    Messages          message.Service      // 消息管理
    History           history.Service      // 历史记录
    Permissions       permission.Service   // 权限管理
    FileTracker       filetracker.Service  // 文件追踪
    AgentCoordinator  agent.Coordinator    // Agent 协调器
    LSPManager        *lsp.Manager         // LSP 管理器
    config            *config.ConfigStore  // 配置存储
    // ...
}
```

### 4.2 LLM Agent 层（`internal/agent/`）

这是 AI 功能的核心，负责与 LLM 交互、管理会话、执行工具调用。

| 关键文件 | 职责 |
|----------|------|
| `agent.go` | `SessionAgent` — 单个会话的 Agent，管理消息循环、自动摘要、Token 管理 |
| `coordinator.go` | `Coordinator` — 管理所有命名 Agent（如 "coder"），构建工具集、选择模型 |
| `agent_tool.go` | Agent 工具的辅助逻辑 |
| `loop_detection.go` | 循环检测，防止 Agent 陷入重复操作 |
| `event.go` | Agent 事件定义 |
| `errors.go` | Agent 错误定义 |

**核心接口**：

```go
type Coordinator interface {
    Run(ctx, sessionID, prompt, attachments) (*AgentResult, error)
    Cancel(sessionID)
    Summarize(ctx, sessionID) error
    Model() Model
    UpdateModels(ctx) error
}
```

**工具系统**（`internal/agent/tools/`）：

每个工具是一个 `.go` + `.md` 文件对：
- `.go` 实现 `fantasy.AgentTool` 接口
- `.md` 提供 LLM 可读的工具描述

| 工具 | 文件 | 功能 |
|------|------|------|
| Bash | `bash.go` | Shell 执行，支持后台任务，有安全命令黑名单 |
| Edit | `edit.go` | 单文件文本替换（带 LSP 集成） |
| MultiEdit | `multiedit.go` | 一次操作中对同一文件做多处编辑 |
| Write | `write.go` | 创建或覆盖文件 |
| View | `view.go` | 读取文件（带行号，支持 LSP 符号查找） |
| Glob | `glob.go` | 文件名模式匹配 |
| Grep | `grep.go` | 正则内容搜索 |
| LS | `ls.go` | 目录树列表 |
| WebFetch | `web_fetch.go` | 获取网页内容 → Markdown |
| WebSearch | `web_search.go` | 网页搜索（DuckDuckGo） |
| Diagnostics | `diagnostics.go` | LSP 诊断 |
| References | `references.go` | LSP 符号引用查找 |
| Todos | `todos.go` | 任务列表管理 |

**创建新工具的步骤**：

1. 创建 `your_tool.go`，实现 `NewYourTool(deps...) fantasy.AgentTool`
2. 创建 `your_tool.md`，写 LLM 可见的工具描述
3. 在 `coordinator.go` 的 `buildTools()` 中注册
4. 在 `internal/config/config.go` 的 `allToolNames` 中添加工具名

**MCP 集成**（`internal/agent/tools/mcp/`）：

MCP (Model Context Protocol) 工具在运行时从配置的 MCP 服务器动态发现。支持 `stdio`、`http`、`sse` 三种传输方式。

### 4.3 配置服务（`internal/config/`）

配置作为服务访问，不使用全局变量。

| 文件 | 职责 |
|------|------|
| `config.go` | 核心类型定义：`Config`、`ProviderConfig`、`MCPConfig`、`Options` 等 |
| `store.go` | `ConfigStore` — 配置的单一所有者，提供读写方法 |
| `load.go` | 配置加载逻辑，多源合并（默认 → 全局 → 工作区） |
| `provider.go` | Provider 自动更新（从 Catwalk） |
| `resolve.go` | 变量解析（环境变量 `$VAR`、命令替换 `$(cmd)`） |
| `scope.go` | 作用域枚举（`ScopeGlobal` / `ScopeWorkspace`） |
| `catwalk.go` | Catwalk API 客户端 |
| `copilot.go` | GitHub Copilot 提供者设置 |
| `hyper.go` | Hyper 提供者设置 |
| `docker_mcp.go` | Docker MCP 主机检测 |

**配置加载优先级**（后者覆盖前者）：

1. 内嵌默认值（来自 Catwalk）
2. 全局配置：`~/.config/crush/crush.json`
3. 工作区配置：项目根目录的 `.crush.json` 或 `crush.json`

**核心类型**：

```go
type Config struct {
    Models      map[SelectedModelType]SelectedModel  // large/small 模型选择
    Providers   *csync.Map[string, ProviderConfig]   // 线程安全的提供者映射
    MCP         map[string]MCPConfig                 // MCP 服务器配置
    LSP         map[string]LSPConfig                 // LSP 服务器配置
    Options     *Options                             // 全局设置
    Permissions *Permissions                         // 工具权限白名单
    Agents      map[string]AgentConfig               // Agent 定义
}
```

### 4.4 数据持久层（`internal/db/`）

基于 SQLite + sqlc 的数据持久化方案。

| 文件/目录 | 职责 |
|-----------|------|
| `sql/` | 原始 SQL 查询文件（`.sql`） |
| `migrations/` | 数据库 schema 迁移文件 |
| `*.sql.go` | sqlc 自动生成的 Go 查询代码 |
| `models.go` | sqlc 生成的数据模型 |
| `querier.go` | sqlc 生成的查询接口 |
| `connect.go` | SQLite 连接管理 |
| `embed.go` | 迁移文件嵌入 |

**工作流**：

1. 在 `internal/db/sql/` 中编写 SQL 查询
2. 在 `internal/db/migrations/` 中编写 schema 迁移
3. 运行 `task sqlc` 或 `sqlc generate` 生成 Go 代码
4. 生成的代码在 `internal/db/` 根目录

**数据库表**（从 SQL 文件推断）：

| SQL 文件 | 管理的数据 |
|----------|-----------|
| `sessions.sql` | 会话（Session）CRUD |
| `messages.sql` | 消息（Message）CRUD |
| `files.sql` | 文件历史记录 |
| `read_files.sql` | 已读文件追踪 |
| `stats.sql` | 使用统计 |

### 4.5 TUI 界面层（`internal/ui/`）

基于 Bubble Tea v2 的终端 UI。

**架构特点**：

- **集中式模型**：`UI` 是唯一的 Bubble Tea Model，子组件是命令式结构体
- **混合渲染**：顶层使用 `uv.ScreenBuffer`（基于矩形区域），子组件渲染为字符串
- **渲染管线**：`View()` → 创建 ScreenBuffer → `Draw()` → `canvas.Render()` → 字符串

| 子目录 | 职责 |
|--------|------|
| `model/` | 主 UI 模型（`UI` 结构体）和主要子模型 |
| `chat/` | 聊天消息项类型和工具渲染器（每个工具一个渲染器文件） |
| `dialog/` | 对话框实现（模型选择、会话、权限、API Key、OAuth 等） |
| `list/` | 通用惰性渲染滚动列表（带视口追踪） |
| `common/` | 共享的 `Common` 结构体（持有 `*app.App` + `*styles.Styles`） |
| `styles/` | 所有样式定义、颜色 token、图标 |
| `completions/` | 自动补全弹出框 |
| `attachments/` | 文件附件管理 |
| `diffview/` | 统一/Split Diff 渲染（带语法高亮） |
| `anim/` | 动画 Spinner |
| `image/` | 终端图片渲染（Kitty 图形协议） |
| `logo/` | Logo 渲染 |
| `util/` | 小型共享工具和消息类型 |

**UI 状态机**：

```
uiOnboarding → uiInitialize → uiLanding → uiChat
```

**工具渲染器映射**（`chat/` 目录）：

| 文件 | 渲染的工具 |
|------|-----------|
| `bash.go` | Bash, JobOutput, JobKill |
| `file.go` | View, Write, Edit, MultiEdit, Download |
| `search.go` | Glob, Grep, LS, Sourcegraph |
| `fetch.go` | Fetch, WebFetch, WebSearch |
| `diagnostics.go` | Diagnostics |
| `references.go` | References |
| `todos.go` | Todos |
| `mcp.go` | MCP 工具（`mcp_` 前缀） |
| `generic.go` | 未识别工具的兜底渲染 |

### 4.6 LSP 集成（`internal/lsp/`）

| 文件 | 职责 |
|------|------|
| `manager.go` | LSP 服务器生命周期管理、自动发现 |
| `client.go` | LSP 客户端实现 |
| `handlers.go` | LSP 请求/响应处理 |
| `util/` | LSP 工具函数 |

LSP 集成用于：文件诊断（Diagnostics）、符号引用查找（References）、代码补全等。可在配置文件中声明式添加 LSP 服务器。

### 4.7 Shell 执行（`internal/shell/`）

| 文件 | 职责 |
|------|------|
| `shell.go` | Shell 命令执行 |
| `background.go` | 后台任务管理 |
| `coreutils.go` | 核心工具命令 |

支持后台任务、命令安全检查、超时控制。

### 4.8 其他核心包

| 包 | 路径 | 职责 |
|----|------|------|
| Session | `internal/session/` | 会话 CRUD 服务（SQLite 支持） |
| Message | `internal/message/` | 消息模型、内容类型、附件 |
| Permission | `internal/permission/` | 工具执行权限管理（白名单 + 用户审批） |
| Skills | `internal/skills/` | Agent Skills 发现与加载（遵循 agentskills.io 标准） |
| PubSub | `internal/pubsub/` | Agent ↔ UI ↔ 服务之间的内部消息总线 |
| Event | `internal/event/` | 遥测（PostHog），匿名使用指标 |
| FileTracker | `internal/filetracker/` | 文件变更追踪服务 |
| History | `internal/history/` | 命令/文件历史记录 |
| Projects | `internal/projects/` | 项目管理 |
| Commands | `internal/commands/` | Slash 命令处理 |
| Diff | `internal/diff/` | Diff 算法，用于可视化 git diff 信息 |
| CSync | `internal/csync/` | 并发安全的数据结构（Map、Value、VersionedMap） |
| FSExt | `internal/fsext/` | 文件系统扩展（忽略文件、路径查找、ls 等） |

### 4.9 Client-Server 架构（`internal/server/` + `internal/client/` + `internal/proto/`）

Crush 还提供了基于 Unix Socket（或 Windows Named Pipe）的 API 服务。

| 包 | 职责 |
|----|------|
| `server/` | API 服务端（通过 Unix Socket 暴露 RESTful API） |
| `client/` | API 客户端（连接 Unix Socket） |
| `proto/` | Client-Server 之间的协议类型定义 |
| `backend/` | 后端抽象层（桥接 server 和核心服务） |
| `workspace/` | 工作区抽象（App 端和 Client 端实现） |
| `swagger/` | 自动生成的 OpenAPI/Swagger 规范 |

---

## 5. 数据流

### 5.1 用户输入 → Agent 响应

```
用户在 TUI 输入
    → UI.Update() 处理键盘事件
    → textarea 提交
    → app.AgentCoordinator.Run(sessionID, prompt, attachments)
    → coordinator 选择模型、构建工具集
    → SessionAgent.Run() 执行 LLM 循环
        → 发送消息给 LLM
        → 接收回复（可能包含工具调用）
        → 执行工具调用
        → 通过 pubsub 发布事件
        → UI 监听事件，更新聊天视图
    → 返回 AgentResult
```

### 5.2 工具执行流

```
LLM 返回 tool_call
    → SessionAgent 解析调用
    → 检查权限（permission.Service）
        → 未授权：通过 pubsub 请求 UI 弹出权限对话框
        → 用户批准/拒绝
    → 执行工具（如 bash、edit、write）
    → 返回结果给 LLM
    → 继续对话循环
```

### 5.3 事件传播

```
Agent 事件 → pubsub.Broker → UI.Update()
LSP 事件   → app.lspEvents() → pubsub → UI
MCP 事件   → mcp.Initialize() → pubsub → UI
配置变更   → ConfigStore 方法 → 写入 JSON → 通知相关方
```

---

## 6. 关键设计模式

| 模式 | 说明 |
|------|------|
| **Config as a Service** | 通过 `config.ConfigStore` 访问配置，不使用全局变量 |
| **Self-documenting Tools** | 每个工具由 `.go` + `.md` 文件对组成 |
| **Go Template System Prompts** | `templates/*.md.tpl` 中使用 Go 模板，支持运行时数据注入 |
| **Context Files** | 读取 `AGENTS.md`、`CRUSH.md`、`CLAUDE.md`、`GEMINI.md`（及 `.local` 变体）作为项目上下文 |
| **SQLite + sqlc** | SQL 在 `db/sql/`，通过 `sqlc generate` 生成 Go 代码 |
| **Centralized UI Model** | `UI` 是唯一的 Bubble Tea Model，子组件是命令式结构体 |
| **Pub/Sub** | `internal/pubsub` 用于 Agent ↔ UI ↔ 服务之间的松耦合通信 |
| **Thread-safe Providers** | `csync.Map` 保证并发安全的 Provider 访问 |
| **Permission Model** | 写/执行工具需用户审批，只读工具跳过（但仍尊重 .gitignore） |
| **No CGO** | `CGO_ENABLED=0`，使用纯 Go SQLite 驱动 |

---

## 7. 常用开发命令

```bash
# 构建
go build .

# 运行测试
task test              # 或 go test ./...

# 代码检查
task lint              # 运行所有 linter
task lint:fix          # 运行并自动修复

# 格式化
task fmt               # gofumpt -w .

# 代码现代化
task modernize         # 自动简化代码

# 开发模式（带 pprof）
task dev

# 生成 golden 文件
go test ./... -update

# sqlc 代码生成
task sqlc              # 或 sqlc generate

# 生成 JSON Schema
task schema

# 生成 OpenAPI/Swagger
task swag

# 更新 Fantasy + Catwalk 依赖
task deps
```

---

## 8. 常见开发场景指引

### 添加新的内置工具

1. 在 `internal/agent/tools/` 创建 `your_tool.go` 和 `your_tool.md`
2. Go 文件中实现 `NewYourTool(deps) fantasy.AgentTool`
3. 在 `internal/agent/coordinator.go` 的 `buildTools()` 中调用 `NewYourTool()`
4. 在 `internal/config/config.go` 的 `allToolNames` 中添加工具名
5. 如果工具需要在 TUI 中有特殊渲染，在 `internal/ui/chat/` 中创建对应渲染器

### 修改系统提示词

- 编辑 `internal/agent/templates/*.md.tpl`（Go 模板文件）
- 支持运行时注入变量（如 `{{.WorkingDir}}`、`{{.ModelName}}` 等）

### 添加新的 CLI 命令

- 在 `internal/cmd/` 中创建新的 Cobra 命令文件
- 在 `root.go` 中注册到根命令

### 修改配置 Schema

- 编辑 `internal/config/config.go` 中的结构体定义
- 运行 `task schema` 重新生成 `schema.json`

### 数据库 Schema 变更

1. 在 `internal/db/migrations/` 中创建新迁移文件
2. 在 `internal/db/sql/` 中编写/更新 SQL 查询
3. 运行 `task sqlc` 重新生成 Go 代码

### 添加新的 LLM Provider

- 编辑 `internal/config/` 中的 `ProviderConfig` 和相关类型
- 添加 Provider 文件（参考 `copilot.go`、`hyper.go` 的模式）

### 添加 TUI 组件

- 遵循集中式模型模式：创建命令式结构体，暴露方法而非 `Update(tea.Msg)`
- 在 `UI.Update()` 中决定何时调用子组件
- 在 `styles/styles.go` 中定义样式
- 使用 `*common.Common` 传递 App 引用和样式

### 添加 MCP 服务器支持

- MCP 配置在 `crush.json` 的 `mcp` 字段中声明
- 运行时由 `internal/agent/tools/mcp/` 动态发现和包装为工具

---

## 9. 关键依赖说明

| 依赖 | 用途 |
|------|------|
| `charm.land/fantasy` | LLM Provider 抽象层（支持 Anthropic、OpenAI、Gemini 等） |
| `charm.land/bubbletea/v2` | TUI 框架（Elm 架构） |
| `charm.land/lipgloss/v2` | 终端样式/布局 |
| `charm.land/glamour/v2` | Markdown 渲染 |
| `charm.land/catwalk` | 模型/Provider 数据库 + Golden-file 测试 |
| `charm.land/fang/v2` | ANSI 差异渲染 |
| `charm.land/x/vcr` | HTTP VCR（录制/回放测试） |
| `sqlc` | SQL → Go 代码生成 |
| `github.com/go-git/go-git/v5` | Git 操作 |
| `github.com/swaggo/swag` | OpenAPI 文档生成 |

---

## 10. 代码风格要点

- **格式化**：使用 `gofumpt`（比 `gofmt` 更严格）
- **导入分组**：标准库 → 外部包 → 内部包
- **错误处理**：显式返回，使用 `fmt.Errorf` 包装
- **Context**：始终将 `context.Context` 作为第一个参数
- **JSON Tag**：使用 `snake_case`
- **文件权限**：使用八进制表示（`0o755`、`0o644`）
- **日志消息**：首字母大写，以 `.` 结尾
- **注释**：独立行注释首字母大写，以 `.` 结尾，78 列换行
- **提交信息**：使用语义化前缀（`fix:`、`feat:`、`chore:`、`refactor:`、`docs:`、`sec:`）

---

## 11. 测试要点

- 使用 `testify` 的 `require` 包
- 并行测试使用 `t.Parallel()`
- 设置环境变量使用 `t.SetEnv()`
- 临时目录使用 `t.TempDir()`（无需手动清理）
- Mock Provider：设置 `config.UseMockProviders = true`
- Golden-file 测试：`go test ./... -update` 重新生成

---

## 12. 子目录详细指南

项目中还有更详细的子目录级 `AGENTS.md` 文件：

| 路径 | 内容 |
|------|------|
| `internal/ui/AGENTS.md` | TUI 架构、渲染管线、组件模式 |
| `internal/agent/tools/AGENTS.md` | 工具系统、注册机制、创建新工具 |
| `internal/config/AGENTS.md` | 配置服务、Provider 管理、模型解析 |
