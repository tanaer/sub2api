# 技术方案设计

## 目标

在仓库内建立一套可持续维护的开发者文档体系，重点解决：

- 新开发者能在 10 分钟内找到入口、启动项目、理解分层与关键数据流。
- 维护者能在新增功能（新路由/新上游/新配置/新迁移/新页面）时，按约定补齐对应文档，不产生知识债。

## 现有架构要点（文档需要覆盖的“真相源”）

- 启动入口：后端主入口在 `backend/cmd/server/main.go`，包含 `-setup`、首次启动 setup server、正常模式启动与优雅退出。
- 依赖注入：Wire 负责把 config/repository/service/middleware/handler/server 链路装配成 Application。
- 分层：`internal/handler`（HTTP/DTO）→ `internal/service`（业务编排）→ `internal/repository`（存储/外部依赖与缓存）；`internal/server` 负责 Router 与 HTTP Server；`internal/config` 负责配置加载与校验。
- 网关兼容层：对外提供 `/v1`、`/v1beta`、`/responses`、`/antigravity/*` 等兼容路径，并通过 API Key 鉴权与订阅策略控制请求。
- 数据库迁移：SQL 迁移文件为 schema 权威来源；启动时自动迁移并通过 checksum 防篡改；使用 PostgreSQL advisory lock 保证多实例串行迁移。
- 嵌入式前端：build tag `embed` 时后端通过 embed.FS 提供静态资源，并注入公开设置到 `index.html`。

## 文档信息架构（IA）

### 目录结构

```
docs/
  README.md
  architecture/
    overview.md
  backend/
    boot.md
    routing.md
    auth.md
    config.md
    database.md
    gateway.md
  frontend/
    overview.md
  dev/
    testing-ci.md
  runbook/
    ops.md
```

### 导航策略

- `docs/README.md` 是唯一入口，按“任务”组织链接（What you want to do）。
- 各主题页顶部包含：适用场景、关键入口文件、主流程图、常见修改点、常见坑。

## 内容规范

### 源码引用规范

- 文档中必须使用可点击的源码引用（Trae 支持 `file:///` 链接）。
- 每个主题页至少包含：
  - 3 个关键源码链接（入口/核心实现/扩展点）
  - 1 个“建议从这里读起”的路径

### 图表规范

- 仅在能显著降低理解成本时使用图表。
- 优先使用 mermaid（sequence/flowchart/class）：
  - 请求处理链路：flowchart/sequence
  - 分层与依赖：flowchart
  - 数据表关系：class/er（如需要）

### 术语与命名

- “网关(Gateway)”：面向外部 SDK 的兼容 API 层（Claude/OpenAI/Gemini/Antigravity）。
- “上游(Upstream)”：被代理/转发的外部 AI 服务。
- “分层”固定用法：handler/service/repository。

## 测试与一致性策略

- 文档与代码的一致性以源码为准：文档必须引用入口文件，关键行为必须能在源码中找到对应实现。
- 自检项：
  - 文档目录完整性（索引可达）
  - README 文档入口无 404（仓库内路径存在）
  -（可选但推荐）运行后端 `go test ./...` 与前端 lint/typecheck 以防“文档承诺与现实不符”

## 安全性与敏感信息

- 文档不得包含任何真实密钥/Token/生产环境连接串。
- 文档可描述“应设置哪些环境变量/配置项”，但不得给出真实值。
- 在涉及 SSRF/allowlist、响应头过滤、安全头策略时，必须提示风险边界与默认行为。

