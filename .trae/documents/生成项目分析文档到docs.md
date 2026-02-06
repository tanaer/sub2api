## 背景与目标
- 背景：你希望我能“完整详细分析这个项目”，把知识沉淀成文档放到 `docs/`，便于后续我持续开发/你也能快速上手。
- 目标：产出一套面向开发者的项目文档（架构、模块职责、关键流程、配置、数据库、API、前端结构、测试/CI、常见扩展点）。
- 非目标：不做产品宣传文案；不承诺覆盖每一个 handler 的业务细节，但会覆盖“如何定位/如何修改/关键边界”。

## 现状快速分析（基于代码现状）
- 技术栈：Go 1.25.7 + Gin + Wire + Ent；PostgreSQL + Redis；前端 Vue 3 + Vite + Tailwind。
- 启动入口：主服务在 [main.go](file:///d:/code/go/sub2api/backend/cmd/server/main.go)；支持 `-setup` CLI 安装向导；首次启动可进入 setup server。
- 依赖注入：Wire 在 [wire.go](file:///d:/code/go/sub2api/backend/cmd/server/wire.go) / [wire_gen.go](file:///d:/code/go/sub2api/backend/cmd/server/wire_gen.go)。
- Web 路由总装配：Gin 中间件与路由集中在 [router.go](file:///d:/code/go/sub2api/backend/internal/server/router.go)，统一挂载 `/api/v1` 与兼容网关路径。
- 兼容网关路由：Claude `/v1/*`、Gemini `/v1beta/*`、OpenAI Responses `/v1/responses` 与 `/responses`、Antigravity `/antigravity/*`，见 [gateway.go](file:///d:/code/go/sub2api/backend/internal/server/routes/gateway.go)。
- 配置加载：Viper 多路径 + 环境变量覆盖（`.`→`_`），默认值与校验在 [config.go](file:///d:/code/go/sub2api/backend/internal/config/config.go#L591-L739)。
- 数据库与迁移：SQL 迁移是 schema 的权威来源；启动时执行迁移并做 checksum 防篡改，见 [ent.go](file:///d:/code/go/sub2api/backend/internal/repository/ent.go) 与 [migrations_runner.go](file:///d:/code/go/sub2api/backend/internal/repository/migrations_runner.go)。
- 认证路由：登录/注册/refresh/2FA/邮件验证码等在 [auth.go](file:///d:/code/go/sub2api/backend/internal/server/routes/auth.go)。
- 前端可嵌入后端：build tag `embed` 时通过 embed.FS 服务静态文件，并在 `index.html` 注入公开设置，见 [embed_on.go](file:///d:/code/go/sub2api/backend/internal/web/embed_on.go)。

## 需求确认（将写入 specs，EARS）
- 需求 1：当开发者需要了解项目整体结构时，系统文档应当提供“模块地图 + 关键入口 + 数据流/请求流”。
- 需求 2：当开发者需要修改/新增 API 或网关适配时，文档应当明确路由组织方式、鉴权/中间件链路、handler/service/repository 分层与扩展点。
- 需求 3：当部署者需要配置运行时行为时，文档应当列出配置来源优先级、关键配置项、生产建议（例如 JWT/TOTP key 固定化）。
- 需求 4：当开发者需要演进数据库时，文档应当说明迁移机制（锁、checksum、基线对齐）与正确的变更方式。
- 需求 5：文档应当可持续维护：有统一目录结构与索引页，新增模块时能按模板补齐。

## 文档交付物（计划落地到仓库）
- `docs/README.md`：文档总索引（从“我想做什么”反向链接）。
- `docs/architecture/overview.md`：系统组件与分层、运行模式（standard/simple）、关键数据流（含 mermaid 图）。
- `docs/backend/boot.md`：启动流程（setup/auto-setup/normal）、Wire 装配点与生命周期（Cleanup/Shutdown）。
- `docs/backend/routing.md`：Gin Router 结构、路由分组、兼容网关路径、BodyLimit/ClientRequestID/OpsErrorLogger 等中间件链。
- `docs/backend/auth.md`：JWT/管理员鉴权/API Key 鉴权；认证相关端点与限流策略（Redis fail-close）。
- `docs/backend/config.md`：配置文件搜索路径、env 覆盖规则、关键配置块说明与生产建议。
- `docs/backend/database.md`：Ent 角色、SQL 迁移权威来源、迁移执行细节（advisory lock + checksum + atlas baseline）。
- `docs/backend/gateway.md`：Claude/OpenAI/Gemini/Antigravity 的适配入口、请求转发骨架、常见扩展点（新增 provider / 新增模型路由）。
- `docs/frontend/overview.md`：前端目录结构、路由/Store/API client 的组织方式、与后端 settings 注入的契约。
- `docs/dev/testing-ci.md`：Makefile 入口、build tags（unit/integration/e2e/embed）、CI（workflow）说明。
- `docs/runbook/ops.md`：常见运维问题定位（配置加载、迁移失败、redis 限流、版本信息、日志字段）。

## 需求/设计/任务文档（按你的工作流）
- 新增 `specs/project_docs/requirements.md`：按 EARS 写清需求与验收标准。
- 新增 `specs/project_docs/design.md`：文档体系的技术设计（目录结构、命名规范、链接规范、图表规范、未来如何增量维护）。
- 新增 `specs/project_docs/tasks.md`：实施任务清单（可勾选），并在执行阶段持续更新状态。

## 实施步骤（确认后我会执行）
1. 先在 `specs/project_docs/` 产出 requirements/design/tasks 三份文档，并与你对齐验收口径。
2. 创建 `docs/` 目录与索引页，按上面的结构逐篇写入。
3. 每篇文档引用关键源码位置（可点击 file 链接）与关键流程图（mermaid）。
4. 对照 README 中的“Documentation”段落，补齐/修正链接（例如目前仓库里未找到 `docs/dependency-security.md`，会在确认后处理）。
5. 执行快速自检：检查 docs 目录完整性、内部链接、并（如你允许）跑一次后端 `go test ./...` 与前端 lint/typecheck 以确保文档描述与实际保持一致。

## 验收标准（你确认后将用于自检）
- 文档覆盖：至少包含启动/路由/鉴权/配置/数据库迁移/网关/前端/测试CI 8 大主题。
- 可导航：`docs/README.md` 能从“常见问题/任务”入口跳转到对应文档。
- 可落地：每个主题至少给出 3 个关键源码定位链接与 1 个流程/结构图（适用处）。
- 可维护：提供统一的文档模板或约定（标题、目录、链接、图表规范）。

如果你确认该方案，我将开始创建 `specs/project_docs/*` 与 `docs/**` 并补齐 README 的文档入口。