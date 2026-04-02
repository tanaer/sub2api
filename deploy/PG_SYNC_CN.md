# PostgreSQL 增量同步脚本说明

- 日期：2026-04-02
- 执行者：Codex

## 目标

`deploy/pg_sync_incremental.sh` 用于把当前服务器上的本地 PostgreSQL 业务库同步到远端 PostgreSQL。

默认流程：

1. 从 `deploy/.pg_sync.env` 读取远端连接参数
2. 从 `/etc/aiapi/config.yaml` 自动读取本地 `aiapi` 源库配置
3. 如果远端 `aiapi` 数据库为空，先执行一次全量初始化
4. 每次增量同步前，先补齐远端缺失的源端新增列
5. 后续执行混合增量同步

## 同步策略

脚本不是 CDC，也不是逻辑复制。它采用低复杂度的混合方案：

- `updated_at` 型业务表：按时间水位做 `UPSERT`
- 日志型大表：按单调递增 `id` 追加
- 少量关系表和聚合表：整表刷新

当前脚本适合这类目标：

- 先快速把本地库复制到远端
- 后续重复执行同步脚本
- 允许软删除字段通过 `updated_at` 同步
- 允许源库新增“只追加的列”自动补到远端

当前脚本不处理这类场景：

- 物理删除的全库精确镜像
- 亚秒级实时同步
- PostgreSQL 逻辑复制 / WAL 订阅
- 已有列的类型变更、删除列、复杂索引/约束差异自动迁移

## 运行前准备

在 `deploy/.pg_sync.env` 写入远端配置。该文件已被 `deploy/.gitignore` 忽略，不会进入 git。

示例：

```bash
PG_SYNC_TARGET_HOST=pgm-xxxx.pg.rds.aliyuncs.com
PG_SYNC_TARGET_PORT=5432
PG_SYNC_TARGET_USER=remote_user
PG_SYNC_TARGET_PASSWORD='replace_me'
PG_SYNC_TARGET_DB=aiapi
PG_SYNC_TARGET_SSLMODE=prefer
```

说明：

- 本地源库默认从 `/etc/aiapi/config.yaml` 的 `database` 段读取
- 如果你想覆盖本地源库，也可以在 `deploy/.pg_sync.env` 里额外设置：
  `PG_SYNC_SOURCE_HOST`、`PG_SYNC_SOURCE_PORT`、`PG_SYNC_SOURCE_USER`、`PG_SYNC_SOURCE_PASSWORD`、`PG_SYNC_SOURCE_DB`、`PG_SYNC_SOURCE_SSLMODE`

## 一键执行

直接执行：

```bash
bash deploy/pg_sync_incremental.sh
```

首次执行：

- 自动创建远端 `aiapi` 数据库（若不存在）
- 检查远端 `public` 是否为空
- 空库时执行全量初始化
- 初始化完成后，把本地当前最大水位写入 `deploy/.pg_sync_state.env`

后续执行：

- 使用 `deploy/.pg_sync_state.env` 记录的水位做增量同步
- 如果源库比远端多出新列，脚本会先执行 `ALTER TABLE ... ADD COLUMN ...`

## 常用选项

仅初始化，不做后续增量：

```bash
bash deploy/pg_sync_incremental.sh --init-only
```

仅做增量，不允许空库初始化：

```bash
bash deploy/pg_sync_incremental.sh --sync-only
```

只重建本地状态文件，不写远端数据：

```bash
bash deploy/pg_sync_incremental.sh --rebuild-state
```

预演模式，只打印动作不真正写入：

```bash
bash deploy/pg_sync_incremental.sh --dry-run
```

## 状态文件

状态文件默认路径：

```text
deploy/.pg_sync_state.env
```

用途：

- 记录每张增量表的同步水位
- 避免每次都全量扫描远端表来决定起点

如果远端已经是正确基线，但本地状态文件丢了，可以执行：

```bash
bash deploy/pg_sync_incremental.sh --rebuild-state
```

该命令会按远端当前最大 `id` / `updated_at` / `created_at` 回填状态文件。

## 注意事项

- 远端目标默认是 `aiapi`，不要把业务表直接同步到远端 `postgres`
- 脚本默认允许对少量关系表与聚合表执行整表刷新
- 如果本地新增了新表，但脚本里还没分配同步策略，脚本会直接失败，避免静默漏表
- 对于仅靠 `updated_at` 增量的表，脚本会回看最近 `5` 分钟窗口，防止边界时间遗漏；重复行由 `UPSERT` 去重
