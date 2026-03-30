-- 080: 用户 + 分组维度按次配额
-- 用于管理员为特定用户在特定分组下设置请求次数配额。

CREATE TABLE IF NOT EXISTS user_group_request_quotas (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    request_quota BIGINT NOT NULL DEFAULT 0 CHECK (request_quota >= 0),
    request_quota_used BIGINT NOT NULL DEFAULT 0 CHECK (request_quota_used >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_user_group_request_quotas_group_id
    ON user_group_request_quotas(group_id);

CREATE INDEX IF NOT EXISTS idx_user_group_request_quotas_user_id
    ON user_group_request_quotas(user_id);
