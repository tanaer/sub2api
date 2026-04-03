package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const upsertOpsRequestTraceSQL = `
INSERT INTO ops_request_traces (
  client_request_id,
  local_request_id,
  usage_request_id,
  upstream_request_ids,
  original_requested_model,
  group_resolved_model,
  account_support_lookup_model,
  final_upstream_model,
  status,
  final_status_code,
  trace_incomplete,
  platform,
  request_path,
  inbound_endpoint,
  upstream_endpoint,
  user_id,
  api_key_id,
  group_id,
  final_account_id,
  created_at,
  finished_at,
  duration_ms,
  trace_events
) VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23
)
ON CONFLICT (client_request_id) DO UPDATE SET
  local_request_id = EXCLUDED.local_request_id,
  usage_request_id = EXCLUDED.usage_request_id,
  upstream_request_ids = EXCLUDED.upstream_request_ids,
  original_requested_model = EXCLUDED.original_requested_model,
  group_resolved_model = EXCLUDED.group_resolved_model,
  account_support_lookup_model = EXCLUDED.account_support_lookup_model,
  final_upstream_model = EXCLUDED.final_upstream_model,
  status = EXCLUDED.status,
  final_status_code = EXCLUDED.final_status_code,
  trace_incomplete = EXCLUDED.trace_incomplete,
  platform = EXCLUDED.platform,
  request_path = EXCLUDED.request_path,
  inbound_endpoint = EXCLUDED.inbound_endpoint,
  upstream_endpoint = EXCLUDED.upstream_endpoint,
  user_id = EXCLUDED.user_id,
  api_key_id = EXCLUDED.api_key_id,
  group_id = EXCLUDED.group_id,
  final_account_id = EXCLUDED.final_account_id,
  finished_at = EXCLUDED.finished_at,
  duration_ms = EXCLUDED.duration_ms,
  trace_events = EXCLUDED.trace_events
`

func (r *opsRepository) UpsertRequestTrace(ctx context.Context, trace *service.OpsRequestTrace) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if trace == nil {
		return fmt.Errorf("nil request trace")
	}
	clientRequestID := strings.TrimSpace(trace.ClientRequestID)
	if clientRequestID == "" {
		return fmt.Errorf("client_request_id is required")
	}

	createdAt := trace.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	var traceEventsJSON []byte
	if len(trace.TraceEvents) > 0 {
		payload, err := json.Marshal(trace.TraceEvents)
		if err != nil {
			return err
		}
		traceEventsJSON = payload
	}

	_, err := r.db.ExecContext(
		ctx,
		upsertOpsRequestTraceSQL,
		clientRequestID,
		opsNullString(trace.LocalRequestID),
		opsNullString(trace.UsageRequestID),
		pq.Array(trace.UpstreamRequestIDs),
		opsNullString(trace.OriginalRequestedModel),
		opsNullString(trace.GroupResolvedModel),
		opsNullString(trace.AccountSupportLookupModel),
		opsNullString(trace.FinalUpstreamModel),
		opsNullString(trace.Status),
		opsNullInt(trace.FinalStatusCode),
		trace.TraceIncomplete,
		opsNullString(trace.Platform),
		opsNullString(trace.RequestPath),
		opsNullString(trace.InboundEndpoint),
		opsNullString(trace.UpstreamEndpoint),
		opsNullInt64(trace.UserID),
		opsNullInt64(trace.APIKeyID),
		opsNullInt64(trace.GroupID),
		opsNullInt64(trace.FinalAccountID),
		createdAt,
		opsTraceNullTime(trace.FinishedAt),
		opsNullInt64(trace.DurationMs),
		opsNullJSON(traceEventsJSON),
	)
	return err
}

func (r *opsRepository) ListRequestTracesByKey(ctx context.Context, key string, keyType string, limit int) ([]*service.OpsRequestTrace, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" {
		return nil, fmt.Errorf("trace key is required")
	}
	if limit <= 0 {
		limit = 20
	}

	var where string
	switch strings.TrimSpace(strings.ToLower(keyType)) {
	case service.OpsRequestTraceKeyTypeClient:
		where = "client_request_id = $1"
	case service.OpsRequestTraceKeyTypeLocal:
		where = "local_request_id = $1"
	case service.OpsRequestTraceKeyTypeUsage:
		where = "usage_request_id = $1"
	case service.OpsRequestTraceKeyTypeUpstream:
		where = "upstream_request_ids @> ARRAY[$1]::text[]"
	default:
		return nil, fmt.Errorf("invalid trace key_type")
	}

	query := fmt.Sprintf(`
SELECT
  client_request_id,
  COALESCE(local_request_id, ''),
  COALESCE(usage_request_id, ''),
  COALESCE(upstream_request_ids, ARRAY[]::text[]),
  COALESCE(original_requested_model, ''),
  COALESCE(group_resolved_model, ''),
  COALESCE(account_support_lookup_model, ''),
  COALESCE(final_upstream_model, ''),
  COALESCE(status, ''),
  final_status_code,
  trace_incomplete,
  COALESCE(platform, ''),
  COALESCE(request_path, ''),
  COALESCE(inbound_endpoint, ''),
  COALESCE(upstream_endpoint, ''),
  user_id,
  api_key_id,
  group_id,
  final_account_id,
  created_at,
  finished_at,
  duration_ms,
  COALESCE(trace_events, '[]'::jsonb)
FROM ops_request_traces
WHERE %s
ORDER BY created_at DESC
LIMIT $2
`, where)

	rows, err := r.db.QueryContext(ctx, query, normalizedKey, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	traces := make([]*service.OpsRequestTrace, 0, limit)
	for rows.Next() {
		var (
			trace           service.OpsRequestTrace
			upstreamIDs     pq.StringArray
			finalStatusCode sql.NullInt64
			userID          sql.NullInt64
			apiKeyID        sql.NullInt64
			groupID         sql.NullInt64
			finalAccountID  sql.NullInt64
			finishedAt      sql.NullTime
			durationMs      sql.NullInt64
			traceEventsJSON []byte
		)
		if err := rows.Scan(
			&trace.ClientRequestID,
			&trace.LocalRequestID,
			&trace.UsageRequestID,
			&upstreamIDs,
			&trace.OriginalRequestedModel,
			&trace.GroupResolvedModel,
			&trace.AccountSupportLookupModel,
			&trace.FinalUpstreamModel,
			&trace.Status,
			&finalStatusCode,
			&trace.TraceIncomplete,
			&trace.Platform,
			&trace.RequestPath,
			&trace.InboundEndpoint,
			&trace.UpstreamEndpoint,
			&userID,
			&apiKeyID,
			&groupID,
			&finalAccountID,
			&trace.CreatedAt,
			&finishedAt,
			&durationMs,
			&traceEventsJSON,
		); err != nil {
			return nil, err
		}

		trace.UpstreamRequestIDs = []string(upstreamIDs)
		trace.FinalStatusCode = nullIntToPtr(finalStatusCode)
		trace.UserID = nullInt64ToPtr(userID)
		trace.APIKeyID = nullInt64ToPtr(apiKeyID)
		trace.GroupID = nullInt64ToPtr(groupID)
		trace.FinalAccountID = nullInt64ToPtr(finalAccountID)
		trace.FinishedAt = nullTimeToPtr(finishedAt)
		trace.DurationMs = nullInt64ToPtr(durationMs)
		if len(traceEventsJSON) > 0 {
			if err := json.Unmarshal(traceEventsJSON, &trace.TraceEvents); err != nil {
				return nil, err
			}
		}
		traces = append(traces, &trace)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return traces, nil
}

func opsTraceNullTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}

func opsNullJSON(value []byte) any {
	if len(value) == 0 {
		return []byte("[]")
	}
	return value
}

func nullIntToPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
}

func nullInt64ToPtr(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

func nullTimeToPtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	value := v.Time.UTC()
	return &value
}
