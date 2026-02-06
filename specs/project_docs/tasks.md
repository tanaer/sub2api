# 实施计划

- [ ] 1. 创建 docs 目录与总索引
  - 新增 `docs/README.md`，按“任务导向”组织入口与链接
  - _需求: 需求 1_

- [ ] 2. 编写架构总览文档
  - 分层与组件关系图（mermaid）
  - 请求处理主链路（mermaid）
  - _需求: 需求 2_

- [ ] 3. 编写后端启动与装配文档
  - setup/auto-setup/normal 启动路径说明
  - Wire 组装与生命周期说明
  - _需求: 需求 2_

- [ ] 4. 编写路由与中间件链路文档
  - `/api/v1` 与兼容网关路径说明
  - BodyLimit/ClientRequestID/OpsErrorLogger/鉴权中间件说明
  - _需求: 需求 2, 需求 5_

- [ ] 5. 编写鉴权与认证文档
  - JWT、Admin、API Key、订阅鉴权（Google/Gemini）说明
  - 登录/refresh/2FA/限流策略说明
  - _需求: 需求 5_

- [ ] 6. 编写配置文档
  - 配置来源优先级、env 覆盖规则、关键配置块说明
  - 生产环境建议（固定 JWT/TOTP key 等）
  - _需求: 需求 3_

- [ ] 7. 编写数据库与迁移文档
  - Ent 与 SQL migration 的关系
  - advisory lock、checksum、atlas baseline 机制
  - _需求: 需求 4_

- [ ] 8. 编写网关兼容层文档
  - Claude/OpenAI/Gemini/Antigravity 路由入口与扩展点
  - 新增 provider/新增路径的最小改动指南
  - _需求: 需求 5_

- [ ] 9. 编写前端结构与契约文档
  - 目录结构、router/store/api client 组织方式
  - settings 注入契约说明
  - _需求: 需求 6_

- [ ] 10. 编写测试/CI 文档与 runbook
  - Makefile 入口与测试分层、CI 约束
  - 常见故障排查清单（迁移失败/配置加载/限流/版本）
  - _需求: 需求 7_

- [ ] 11. 修正 README 文档入口并做自检
  - README 中的 docs 链接存在且可导航
  -（可选）运行后端/前端自检命令，确保描述与实际一致
  - _需求: 需求 1, 需求 7_

