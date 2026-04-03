# Verification

- 日期：2026-03-23 22:34:33 +0800
- 执行者：Codex

## 构建与部署验证

- `CGO_ENABLED=0 go build -tags embed -trimpath -ldflags='-s -w -X main.BuildType=release' -o /tmp/aiapi ./cmd/server`
  - 结果：成功
- `install -o aiapi -g aiapi -m 0755 /tmp/aiapi /opt/aiapi/aiapi`
  - 结果：成功
- `systemctl restart aiapi.service`
  - 结果：成功
- `systemctl is-enabled aiapi.service`
  - 输出：`enabled`

## 运行态验证

- `systemctl status aiapi.service --no-pager -l`
  - 结果：`active (running)`
- `ss -ltnp | grep ':7777 '`
  - 结果：`aiapi` 正在监听 `*:7777`
- `curl -i http://127.0.0.1:7777/health`
  - 结果：`HTTP/1.1 200 OK`
  - 响应体：`{"status":"ok"}`
- `curl -I http://127.0.0.1:7777/`
  - 结果：`HTTP/1.1 200 OK`
- `find /etc/aiapi/sora -maxdepth 2 -type d | sort`
  - 结果：
    - `/etc/aiapi/sora`
    - `/etc/aiapi/sora/image`
    - `/etc/aiapi/sora/video`

## 日志验证

- `journalctl -u aiapi.service --since '2026-03-23 22:33:59' --no-pager | grep -n 'SoraStorage\\|permission denied' || true`
  - 结果：无输出
  - 结论：本次重启后的日志窗口内已不存在 `SoraStorage` 创建 `/app/data/sora` 失败的权限告警

## HTTPS 修复验证

- 背景：`https://aiapi.999194.xyz` 先前经 Cloudflare 返回 `525 SSL handshake failed`
- 根因：源站 `443` 由 `nps` 占用，且仅配置了 `futuapi.890214.net` 证书与主机映射；`aiapi.999194.xyz` 的 SNI 握手被直接断开
- 已执行：
  - `certbot certonly --webroot -w /www/wwwroot/aiapi.999194.xyz -d aiapi.999194.xyz --non-interactive --agree-tos -m admin@aiapi.local`
  - `nginx -t`
  - `systemctl restart Nps.service`
  - `systemctl reload nginx`
- 本地握手验证：
  - `openssl s_client -connect 127.0.0.1:443 -servername aiapi.999194.xyz -brief`
    - 结果：成功
    - 证书：`CN=aiapi.999194.xyz`
  - `openssl s_client -connect 127.0.0.1:443 -servername futuapi.890214.net -brief`
    - 结果：成功
    - 证书：`CN=futuapi.890214.net`
- 公网验证：
  - `curl -I https://aiapi.999194.xyz --max-time 15`
    - 结果：`HTTP/2 200`
  - `curl -I https://futuapi.890214.net --max-time 15`
    - 结果：返回业务侧 `404`，但 TLS 正常建立

---

- 日期：2026-04-01 13:45:00 +0800
- 执行者：Codex
- 任务：账号管理复制功能

## 前端验证

- `npm run test:run -- src/components/account/__tests__/CreateAccountModal.spec.ts`
  - 结果：通过
  - 覆盖点：
    - API Key 账号复制后会把源账号配置带入新增弹窗并提交克隆 payload
    - OAuth/SetupToken 账号复制后会直接复用源凭据创建，不再强制重新授权
- `npm run typecheck`
  - 结果：通过
- `npm run test:run -- src/components/account/__tests__/EditAccountModal.spec.ts`
  - 结果：通过
  - 说明：相邻编辑弹窗回归仍然通过，未被复制功能改坏

## 结论

- 账号页复制功能的核心链路已通过自动化回归测试与类型检查验证。
- 本次未运行端到端浏览器测试；当前验证覆盖组件级复制流程与静态类型层面。

---

- 日期：2026-04-01 13:55:10 +0800
- 执行者：Codex
- 任务：部署账号复制功能到线上

## 上线前验证

- `npm run test:run -- src/components/account/__tests__/CreateAccountModal.spec.ts src/components/account/__tests__/EditAccountModal.spec.ts`
  - 结果：通过
  - 输出摘要：`Test Files 2 passed`, `Tests 4 passed`
- `npm run typecheck`
  - 结果：通过
- `npm run build`
  - 结果：通过
  - 说明：前端产物已输出到 `backend/internal/web/dist`
- `CGO_ENABLED=0 go build -tags embed -trimpath -ldflags='-s -w -X main.BuildType=release' -o /tmp/aiapi-new ./cmd/server`
  - 结果：通过

## 部署动作

- 旧二进制备份：`/opt/aiapi/aiapi.bak-20260401135405`
- 新二进制安装：`install -o aiapi -g aiapi -m 0755 /tmp/aiapi-new /opt/aiapi/aiapi`
- 服务重启：`systemctl restart aiapi.service`

## 上线后验证

- `systemctl status aiapi.service --no-pager -l`
  - 结果：通过
  - 关键状态：`active (running)`，重启时间 `2026-04-01 13:54:10 CST`
- `curl -i http://127.0.0.1:7777/health`
  - 结果：通过
  - 输出：`HTTP/1.1 200 OK`，`{"status":"ok"}`
- `curl -I http://127.0.0.1:7777/`
  - 结果：通过
  - 输出：`HTTP/1.1 200 OK`
- `curl -I https://aiapi.999194.xyz --max-time 20`
  - 结果：通过
  - 输出：`HTTP/2 200`
- `journalctl -u aiapi.service --since '2026-04-01 13:54:05' --no-pager | tail -n 80`
  - 观察：启动后服务已正常接收请求；日志中仍有一条业务侧 `stream usage incomplete: missing terminal event` 错误，属于现网流量触发的应用问题，不影响本次部署成功。

## 结论

- 账号复制功能已部署到线上并完成服务切换。
- 主公网入口 `https://aiapi.999194.xyz` 可正常访问。

---

- 日期：2026-04-01 14:05:30 +0800
- 执行者：Codex
- 任务：重新顺序构建并校验账号复制功能已真实上线

## 根因复核

- 上一次上线时前端 `npm run build` 与后端 `go build -tags embed` 存在并行执行风险。
- 由于后端使用 `embed` 打包前端静态资源，若后端先编译完成，就会把旧前端资源嵌入 `/opt/aiapi/aiapi`。

## 重新部署动作

- `npm run build`
  - 结果：通过
  - 关键产物：`backend/internal/web/dist/assets/AccountsView-BAjh93aW.js`
- `CGO_ENABLED=0 go build -tags embed -trimpath -ldflags='-s -w -X main.BuildType=release' -o /tmp/aiapi-new ./cmd/server`
  - 结果：通过
- 旧二进制备份：`/opt/aiapi/aiapi.bak-20260401140413`
- 新二进制安装：`install -o aiapi -g aiapi -m 0755 /tmp/aiapi-new /opt/aiapi/aiapi`
- 服务重启：`systemctl restart aiapi.service`

## 重新上线后验证

- `systemctl status aiapi.service --no-pager -l`
  - 结果：通过
  - 关键状态：`active (running)`，重启时间 `2026-04-01 14:04:18 CST`
- `curl -i http://127.0.0.1:7777/health`
  - 第一次结果：失败
  - 说明：请求命中服务启动瞬间，端口尚未监听
- `sleep 2; curl -i http://127.0.0.1:7777/health`
  - 结果：通过
  - 输出摘要：`HTTP/1.1 200 OK`，`{"status":"ok"}`
- `curl -I https://aiapi.999194.xyz`
  - 结果：通过
  - 输出摘要：`HTTP/2 200`
- `curl -s https://aiapi.999194.xyz | grep -o 'assets/index-[^"]*\\.js' | head -n 1`
  - 结果：通过
  - 输出：`assets/index-D4JEi8VK.js`
- `curl -s https://aiapi.999194.xyz/assets/AccountsView-BAjh93aW.js | grep -oE 'template-account|common.copy' | sort | uniq`
  - 结果：通过
  - 输出：`common.copy`、`template-account`

## 结论

- 公网当前返回的新前端资源已包含账号复制按钮与模板预填逻辑。
- “强制刷新也看不到”的根因已修正，线上实际静态资源已更新。

---

- 日期：2026-04-01 19:28:00 +0800
- 执行者：Codex
- 任务：按分组自定义 API 密钥使用说明

## 后端验证

- `cd backend && go generate ./ent`
  - 结果：通过
- `cd backend && go test ./internal/handler/dto ./internal/handler/admin`
  - 结果：通过
  - 覆盖点：分组 DTO 映射包含 `use_key_instructions`，管理员分组创建/更新接口可透传该字段
- `cd backend && go build ./...`
  - 结果：通过
- `cd backend && go test ./...`
  - 结果：失败
  - 失败摘要：`backend/internal/service/ratelimit_service_anthropic_test.go:11:2: undefined: mockAccountRepoForGemini`
  - 结论：该失败来自仓库内已有的无关测试问题，不属于本次分组说明功能链路

## 前端验证

- `cd frontend && pnpm test:run src/components/keys/__tests__/UseKeyModal.spec.ts`
  - 结果：通过
  - 覆盖点：存在分组自定义说明时展示自定义文案，并隐藏平台默认说明文案
- `cd frontend && pnpm typecheck`
  - 结果：通过
- `cd frontend && pnpm build`
  - 结果：通过
  - 说明：前端构建产物已输出到 `backend/internal/web/dist`

## 结论

- 分组级 `use_key_instructions` 已打通后台配置、接口返回和用户侧“使用密钥”弹层展示。
- 当前仅有一个与本需求无关的后端既有测试失败；本次功能相关测试、类型检查和构建均已通过。

---

- 日期：2026-04-01 20:08:30 +0800
- 执行者：Codex

---

- 日期：2026-04-02 01:26:30 +0800
- 执行者：Codex
- 任务：本地 PostgreSQL 到阿里云 RDS 的一键增量同步脚本

## 脚本与文档验证

- `bash deploy/test_pg_sync_incremental.sh`
  - 结果：通过
  - 覆盖点：
    - 表同步策略分类
    - 状态文件读写
    - 本地 CSV 中转导入链路
    - 无 `id` 表跳过序列重置
    - 源库新增列 DDL 生成
- `PG_SYNC_ENABLE_REMOTE_INTEGRATION=1 bash deploy/test_pg_sync_incremental.sh`
  - 结果：通过
  - 覆盖点：本地源库到远端 `aiapi` 库的真实 `COPY` 链路，不再出现 `\.` 被误当作 CSV 数据的问题

## 真实同步验证

- `bash deploy/pg_sync_incremental.sh --sync-only`
  - 第一次结果：通过
  - 关键现象：
    - 自动补齐远端列：`accounts.upstream_provider`
    - 成功同步 `ops_system_logs`、`usage_logs`、`scheduler_outbox` 等增量表
    - 最终输出：`增量同步完成`
- `bash deploy/pg_sync_incremental.sh --sync-only`
  - 第二次结果：通过
  - 关键现象：
    - 脚本可重复执行
    - 由于应用仍在持续写入日志、指标、队列表，第二次运行继续同步新增数据，属于预期行为
    - 最终输出：`增量同步完成`

## 远端结果核对

- `psql ... -d aiapi -Atqc "select count(*) from information_schema.columns where table_schema='public' and table_name='accounts' and column_name='upstream_provider';"`
  - 结果：`1`
  - 结论：远端 `accounts.upstream_provider` 已自动补齐
- `psql ... -d aiapi -Atqc "select (select count(*) from public.account_groups), (select count(*) from public.accounts), (select count(*) from public.usage_logs), (select count(*) from public.ops_system_logs);"`
  - 结果：`12|13|24189|303673`
  - 结论：远端关键业务表与增量日志表已有数据

## 结论

- `deploy/pg_sync_incremental.sh` 已可在当前服务器上一条命令执行增量同步。
- 当前脚本已修复两类真实故障：
  - `COPY FROM STDIN` 管道把 `\.` 误识别为 CSV 数据
  - 复合主键表无 `id` 时错误执行序列重置
- 当前脚本支持自动补齐“源端新增、远端缺失”的列，但不负责复杂 schema 演进（删列、改类型、复杂约束变更）。
- 任务：隔离部署分组 API 密钥说明功能

## 隔离构建

- worktree：`/root/.config/superpowers/worktrees/sub2api/deploy-use-key-instructions-20260401`
- `cd backend && go generate ./ent`
  - 结果：通过
- `cd backend && go test ./internal/handler/dto ./internal/handler/admin`
  - 结果：通过
- `cd frontend && pnpm test:run src/components/keys/__tests__/UseKeyModal.spec.ts`
  - 结果：通过
- `cd frontend && pnpm typecheck`
  - 结果：通过
- `cd frontend && pnpm build`
  - 结果：通过
- `cd backend && CGO_ENABLED=0 go build -tags embed -trimpath -ldflags='-s -w -X main.BuildType=release' -o /tmp/aiapi-use-key-instructions-20260401 ./cmd/server`
  - 结果：通过

## 部署动作

- 现网二进制备份：`/opt/aiapi/aiapi.bak-20260401200738`
- 新二进制安装：`install -o aiapi -g aiapi -m 0755 /tmp/aiapi-use-key-instructions-20260401 /opt/aiapi/aiapi`
- 服务重启：`systemctl restart aiapi.service`

## 上线后验证

- `systemctl status aiapi.service --no-pager -l`
  - 结果：通过
  - 关键状态：`active (running)`，重启时间 `2026-04-01 20:07:43 CST`
- `curl -i http://127.0.0.1:7777/health`
  - 结果：通过
  - 输出：`HTTP/1.1 200 OK`，`{"status":"ok"}`
- `curl -I http://127.0.0.1:7777/`
  - 结果：通过
  - 输出：`HTTP/1.1 200 OK`
- `curl -I https://aiapi.999194.xyz --max-time 20`
  - 结果：通过
  - 输出：`HTTP/2 200`
- `curl -s http://127.0.0.1:7777/ | grep -o 'assets/index-[^"]*\.js' | head -n 1`
  - 结果：通过
  - 输出：`assets/index-CAVVFcZ-.js`
- `curl -s http://127.0.0.1:7777/assets/GroupsView-CLfOi2x8.js | grep -o 'use_key_instructions' | head -n 5`
  - 结果：通过
  - 说明：线上已下发包含分组说明字段的新分组管理前端包
- `curl -s http://127.0.0.1:7777/assets/KeysView-Cix2Pmkp.js | grep -o 'custom-instructions\|use_key_instructions' | head -n 5`
  - 结果：通过
  - 说明：线上已下发包含用户侧自定义说明展示逻辑的新密钥页面前端包

## 结论

- 本次功能已通过隔离 worktree 完成构建并上线，未携带当前工作区里的无关改动。
- 服务启动成功，迁移链路正常，前后端产物均已切换到本次版本。

---

- 日期：2026-04-03 23:59:30 +0800
- 执行者：Codex
- 任务：分组默认映射模型未显示到“使用密钥”弹层

## 验证命令

- `cd backend && go test -tags=unit ./internal/handler/dto`
  - 结果：通过
- `cd frontend && pnpm test:run src/components/keys/__tests__/UseKeyModal.spec.ts src/views/user/__tests__/KeysView.spec.ts`
  - 结果：通过
  - 关键输出：`Test Files  2 passed (2)`，`Tests  5 passed (5)`
- `cd frontend && pnpm typecheck`
  - 结果：通过

## 非阻塞项

- `cd backend && go test -tags=unit ./internal/handler/...`
  - 结果：失败
  - 原因：现有 `stubAdminService` 未实现 `BatchUpdateUsers`，属于仓库内既有测试基建问题，不由本次修改引入

## 结论

- 后端已把 `default_mapped_model` 返回给用户侧 API Key 关联分组。
- 前端“使用密钥”弹层已能在无自定义模型列表时展示分组默认映射模型对应的 OpenAI 配置。
