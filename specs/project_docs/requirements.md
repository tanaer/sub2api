# 需求文档

## 介绍

本需求用于为 Sub2API 仓库补齐“面向开发者的工程文档”，统一沉淀到 `docs/` 目录，覆盖后端、前端与运维/测试关键知识点，使后续功能开发与问题定位可快速完成。

## 范围

### in-scope

- 在仓库内新增 `docs/` 文档体系（索引、架构、后端、前端、测试/CI、runbook）。
- 文档必须以“如何定位与修改”为核心：入口、分层、关键流程、关键配置、扩展点、常见坑。
- 文档引用关键源码位置（可点击跳转），并在适用处提供流程/结构图（mermaid）。

### out-of-scope

- 不编写面向用户的营销说明与运营文案。
- 不对每一个 handler/service/repo 的业务细节逐行解释；但必须覆盖关键边界与扩展点。
- 不在本需求内新增/修改业务功能（除必要的 README 文档入口修正）。

## 需求

### 需求 1 - 文档索引与可导航性

**用户故事：** 作为开发者，我希望从一个入口页快速找到启动、配置、网关、数据库、鉴权与前端相关资料，从而快速开始修改代码。

#### 验收标准

1. When 开发者打开 `docs/README.md`, the 文档系统 shall 提供按“任务导向”的目录与链接（例如：启动服务、增加新 API、排查迁移失败、增加新上游适配）。  
2. When 开发者需要定位关键入口, the 文档系统 shall 在索引页提供“入口清单”（后端 main、router、routes、Wire、迁移、前端入口）与对应源码链接。  

### 需求 2 - 架构与分层说明

**用户故事：** 作为开发者，我希望理解后端的分层与数据流（handler/service/repository），以便知道该把逻辑放在哪里。

#### 验收标准

1. When 需要理解请求处理链路, the 文档系统 shall 给出从 HTTP 路由到 handler/service/repository 的主路径描述，并包含一张流程图（mermaid）。  
2. When 需要理解模块职责边界, the 文档系统 shall 描述 `internal/handler`、`internal/service`、`internal/repository`、`internal/server`、`internal/config` 的职责与典型依赖方向。  

### 需求 3 - 配置体系与生产建议

**用户故事：** 作为部署者/开发者，我希望知道配置从哪里来、如何覆盖、哪些配置在生产应固定，以避免重启导致不可预期行为。

#### 验收标准

1. When 配置加载发生, the 文档系统 shall 描述配置文件搜索路径优先级、环境变量覆盖规则（`.`→`_`）以及默认值策略。  
2. When 生产环境部署, the 文档系统 shall 明确列出“建议固定配置项”（例如 JWT secret、TOTP encryption key）与原因。  

### 需求 4 - 数据库与迁移机制说明

**用户故事：** 作为开发者，我希望清楚数据库 schema 的权威来源与迁移执行方式，从而安全地演进数据库。

#### 验收标准

1. When 启动服务, the 文档系统 shall 说明迁移执行机制（advisory lock、checksum 防篡改、跳过已执行迁移）。  
2. When 需要修改 schema, the 文档系统 shall 指导“新增迁移文件而非修改已应用迁移”的正确做法，并解释校验失败的含义与修复路径。  

### 需求 5 - 网关兼容层与扩展点

**用户故事：** 作为开发者，我希望知道 Claude/OpenAI/Gemini/Antigravity 兼容层的路由入口与主要处理链路，以便增加新 provider/新路径或修 bug。

#### 验收标准

1. When 需要新增一个网关路径或兼容层, the 文档系统 shall 指明路由注册位置、中间件链、鉴权方式与 handler 入口。  
2. When 需要新增一个上游适配, the 文档系统 shall 给出扩展点清单（service/repository/pkg 层）与最小改动路径。  

### 需求 6 - 前端结构与后端契约

**用户故事：** 作为前端/全栈开发者，我希望了解前端目录结构、路由与状态管理，并知道与后端 settings 注入/接口调用的契约。

#### 验收标准

1. When 需要修改页面或新增 API 调用, the 文档系统 shall 描述前端 `src/api`、`src/router`、`src/stores` 的组织方式与约定。  
2. When 后端启用嵌入式前端, the 文档系统 shall 说明 settings 注入机制与前端读取方式（window.__APP_CONFIG__）。  

### 需求 7 - 测试、CI 与自检

**用户故事：** 作为开发者，我希望知道如何运行不同层级测试以及 CI 约束，从而在提交前自检通过。

#### 验收标准

1. When 运行测试, the 文档系统 shall 描述后端测试分层（unit/integration/e2e）与相关 build tags/Makefile 入口。  
2. When CI 执行, the 文档系统 shall 描述 CI 主要 job（unit、integration、lint）与关键版本约束（例如 Go 版本）。  

