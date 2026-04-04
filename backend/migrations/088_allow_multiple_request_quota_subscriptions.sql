-- 088_allow_multiple_request_quota_subscriptions.sql
-- 移除 user_subscriptions 的 (user_id, group_id) 唯一约束
-- 允许同一用户在同一分组下拥有多个订阅（每个按次配额兑换码创建独立订阅）

-- 删除旧的唯一索引
DROP INDEX IF EXISTS user_subscriptions_user_group_unique_active;

-- 创建普通索引（非唯一）用于查询加速
CREATE INDEX IF NOT EXISTS idx_user_subscriptions_user_group_active
  ON user_subscriptions (user_id, group_id)
  WHERE deleted_at IS NULL;
