ALTER TABLE account_throttle_rules
    ADD COLUMN IF NOT EXISTS recovery_check_interval INTEGER NOT NULL DEFAULT 0;
