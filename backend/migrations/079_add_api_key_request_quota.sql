-- Add request-count quota fields to api_keys table
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS request_quota BIGINT NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS request_quota_used BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_api_keys_request_quota_request_quota_used
	ON api_keys(request_quota, request_quota_used)
	WHERE deleted_at IS NULL;

COMMENT ON COLUMN api_keys.request_quota IS 'Request-count quota for this API key (0 = disabled)';
COMMENT ON COLUMN api_keys.request_quota_used IS 'Used request count for this API key';
