ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS config_templates TEXT;
