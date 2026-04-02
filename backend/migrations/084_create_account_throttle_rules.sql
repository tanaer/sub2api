CREATE TABLE IF NOT EXISTS account_throttle_rules (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    priority        INTEGER NOT NULL DEFAULT 0,
    error_codes     JSONB DEFAULT '[]'::jsonb,
    keywords        JSONB DEFAULT '[]'::jsonb,
    match_mode      VARCHAR(20) NOT NULL DEFAULT 'contains',
    trigger_mode    VARCHAR(20) NOT NULL DEFAULT 'immediate',
    accumulated_count   INTEGER NOT NULL DEFAULT 3,
    accumulated_window  INTEGER NOT NULL DEFAULT 60,
    action_type         VARCHAR(30) NOT NULL DEFAULT 'duration',
    action_duration     INTEGER NOT NULL DEFAULT 300,
    action_recover_hour INTEGER NOT NULL DEFAULT 0,
    platforms       JSONB DEFAULT '[]'::jsonb,
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_account_throttle_rules_enabled ON account_throttle_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_account_throttle_rules_priority ON account_throttle_rules (priority);
