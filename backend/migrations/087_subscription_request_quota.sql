-- 087_subscription_request_quota.sql
-- 将按次配额能力合并到订阅机制中

-- 1. 扩展 user_subscriptions 表
ALTER TABLE user_subscriptions
  ADD COLUMN IF NOT EXISTS request_quota BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS request_quota_used BIGINT NOT NULL DEFAULT 0;

-- 2. 数据迁移：将活跃的 grant → 订阅记录
-- 对于已有订阅的 (user_id, group_id)：累加 request_quota/used，取最远 expires_at
-- 对于没有订阅的：创建新订阅
-- 注意：唯一约束是 (user_id, group_id) WHERE deleted_at IS NULL
INSERT INTO user_subscriptions (
  user_id, group_id, starts_at, expires_at, status,
  request_quota, request_quota_used,
  assigned_at, notes, created_at, updated_at
)
SELECT
  g.user_id,
  g.group_id,
  MIN(g.created_at),
  MAX(g.expires_at),
  'active',
  SUM(g.request_quota_total),
  SUM(g.request_quota_used),
  NOW(),
  '从按次配额迁移',
  NOW(), NOW()
FROM user_group_request_quota_grants g
WHERE g.expires_at > NOW()
GROUP BY g.user_id, g.group_id
ON CONFLICT (user_id, group_id) WHERE deleted_at IS NULL
DO UPDATE SET
  request_quota = user_subscriptions.request_quota + EXCLUDED.request_quota,
  request_quota_used = user_subscriptions.request_quota_used + EXCLUDED.request_quota_used,
  expires_at = GREATEST(user_subscriptions.expires_at, EXCLUDED.expires_at),
  -- 仅将 expired 状态恢复为 active，保留 suspended/revoked 状态不变
  status = CASE
    WHEN user_subscriptions.status = 'expired' THEN 'active'
    ELSE user_subscriptions.status
  END,
  updated_at = NOW();

-- 3. 确保迁移目标 Group 的 subscription_type = 'subscription'
UPDATE groups
SET subscription_type = 'subscription', updated_at = NOW()
WHERE id IN (
  SELECT DISTINCT group_id FROM user_group_request_quota_grants WHERE expires_at > NOW()
)
AND subscription_type != 'subscription'
AND deleted_at IS NULL;

-- 4. 将未使用的 group_request_quota 兑换码转为 subscription 类型
UPDATE redeem_codes
SET type = 'subscription'
WHERE type = 'group_request_quota'
  AND used_at IS NULL;
