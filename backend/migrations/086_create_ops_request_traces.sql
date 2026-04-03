CREATE TABLE IF NOT EXISTS ops_request_traces (
    id BIGSERIAL PRIMARY KEY,
    client_request_id TEXT NOT NULL,
    local_request_id TEXT,
    usage_request_id TEXT,
    upstream_request_ids TEXT[] NOT NULL DEFAULT ARRAY[]::text[],

    original_requested_model TEXT,
    group_resolved_model TEXT,
    account_support_lookup_model TEXT,
    final_upstream_model TEXT,

    status TEXT NOT NULL DEFAULT '',
    final_status_code INT,
    trace_incomplete BOOLEAN NOT NULL DEFAULT FALSE,

    platform TEXT,
    request_path TEXT,
    inbound_endpoint TEXT,
    upstream_endpoint TEXT,

    user_id BIGINT,
    api_key_id BIGINT,
    group_id BIGINT,
    final_account_id BIGINT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT,

    trace_events JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_ops_request_traces_client_request_id
    ON ops_request_traces (client_request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_local_request_id
    ON ops_request_traces (local_request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_usage_request_id
    ON ops_request_traces (usage_request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_created_at
    ON ops_request_traces (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_upstream_request_ids_gin
    ON ops_request_traces USING GIN (upstream_request_ids);

COMMENT ON TABLE ops_request_traces IS 'Ops request trace timeline storage for request lookup and drilldown';
COMMENT ON COLUMN ops_request_traces.original_requested_model IS 'Model name requested by the client before group/model routing';
COMMENT ON COLUMN ops_request_traces.group_resolved_model IS 'Model after group-level defaulting or fallback resolution';
COMMENT ON COLUMN ops_request_traces.account_support_lookup_model IS 'Model key used when checking account support';
COMMENT ON COLUMN ops_request_traces.final_upstream_model IS 'Final model sent upstream';
COMMENT ON COLUMN ops_request_traces.trace_events IS 'Chronological request trace events stored as JSONB';
