-- 089_subscription_plans.sql
-- 新增订阅计划(subscription_plans)表 + redeem_codes 加 plan_id

-- 1. 创建 subscription_plans 表
CREATE TABLE IF NOT EXISTS subscription_plans (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT,
  group_id BIGINT REFERENCES groups(id),
  billing_mode VARCHAR(20) NOT NULL DEFAULT 'per_request',
  request_quota BIGINT NOT NULL DEFAULT 0,
  daily_limit_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
  weekly_limit_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
  monthly_limit_usd DECIMAL(20,10) NOT NULL DEFAULT 0,
  validity_days INT NOT NULL DEFAULT 30,
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_subscription_plans_status ON subscription_plans(status);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_group_id ON subscription_plans(group_id);
CREATE INDEX IF NOT EXISTS idx_subscription_plans_billing_mode ON subscription_plans(billing_mode);

-- 2. redeem_codes 加 plan_id
ALTER TABLE redeem_codes ADD COLUMN IF NOT EXISTS plan_id BIGINT REFERENCES subscription_plans(id);
CREATE INDEX IF NOT EXISTS idx_redeem_codes_plan_id ON redeem_codes(plan_id);

-- 3. 从现有 Group USD 限额配置生成订阅计划
INSERT INTO subscription_plans (name, group_id, billing_mode, daily_limit_usd, weekly_limit_usd, monthly_limit_usd, validity_days)
SELECT
  g.name || ' USD月卡',
  g.id,
  'per_usd',
  COALESCE(g.daily_limit_usd, 0),
  COALESCE(g.weekly_limit_usd, 0),
  COALESCE(g.monthly_limit_usd, 0),
  30
FROM groups g
WHERE g.subscription_type = 'subscription'
  AND g.deleted_at IS NULL
  AND (COALESCE(g.daily_limit_usd, 0) > 0 OR COALESCE(g.weekly_limit_usd, 0) > 0 OR COALESCE(g.monthly_limit_usd, 0) > 0);
