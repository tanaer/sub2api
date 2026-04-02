package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *opsRepository) GetSLAReport(ctx context.Context, minutes int) (*service.SLAReport, error) {
	if minutes <= 0 || minutes > 10080 {
		minutes = 60
	}
	interval := fmt.Sprintf("%d minutes", minutes)

	report := &service.SLAReport{}

	// === 1. Client-side success rate ===
	// success = usage_logs with duration_ms > 0
	// client error = ops_error_logs where status_code != 200 (200 = recovered upstream error)
	err := r.db.QueryRowContext(ctx, `
		WITH success AS (
			SELECT COUNT(*) AS cnt FROM usage_logs WHERE created_at >= NOW() - $1::interval AND duration_ms > 0
		),
		usage_fail AS (
			SELECT COUNT(*) AS cnt FROM usage_logs WHERE created_at >= NOW() - $1::interval AND (duration_ms = 0 OR duration_ms IS NULL)
		),
		client_errors AS (
			SELECT COUNT(*) AS cnt FROM ops_error_logs WHERE created_at >= NOW() - $1::interval AND status_code != 200
		),
		recovered AS (
			SELECT COUNT(*) AS cnt FROM ops_error_logs WHERE created_at >= NOW() - $1::interval AND status_code = 200
		)
		SELECT s.cnt, uf.cnt, ce.cnt, r.cnt
		FROM success s, usage_fail uf, client_errors ce, recovered r
	`, interval).Scan(
		&report.ClientMetrics.Successful,
		&report.ClientMetrics.UsageLogFailed,
		&report.ClientMetrics.ClientErrors,
		&report.ClientMetrics.RecoveredUpstreamErrors,
	)
	if err != nil {
		return nil, fmt.Errorf("client metrics: %w", err)
	}

	// === 2. Failover stats ===
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE upstream_errors IS NOT NULL AND jsonb_typeof(upstream_errors) = 'array' AND jsonb_array_length(upstream_errors) > 0) AS with_failover,
			COALESCE(AVG(jsonb_array_length(upstream_errors)) FILTER (WHERE upstream_errors IS NOT NULL AND jsonb_typeof(upstream_errors) = 'array' AND jsonb_array_length(upstream_errors) > 0), 0) AS avg_attempts,
			COALESCE(MAX(jsonb_array_length(upstream_errors)) FILTER (WHERE upstream_errors IS NOT NULL AND jsonb_typeof(upstream_errors) = 'array' AND jsonb_array_length(upstream_errors) > 0), 0) AS max_attempts,
			COUNT(*) FILTER (WHERE status_code = 200 AND upstream_errors IS NOT NULL AND jsonb_typeof(upstream_errors) = 'array' AND jsonb_array_length(upstream_errors) > 0) AS recovered_after_failover,
			COUNT(*) FILTER (WHERE status_code != 200 AND upstream_errors IS NOT NULL AND jsonb_typeof(upstream_errors) = 'array' AND jsonb_array_length(upstream_errors) > 0) AS failed_after_failover
		FROM ops_error_logs
		WHERE created_at >= NOW() - $1::interval
	`, interval).Scan(
		&report.FailoverMetrics.TotalWithFailover,
		&report.FailoverMetrics.AvgAttempts,
		&report.FailoverMetrics.MaxAttempts,
		&report.FailoverMetrics.RecoveredAfterFailover,
		&report.FailoverMetrics.FailedAfterFailover,
	)
	if err != nil {
		return nil, fmt.Errorf("failover metrics: %w", err)
	}

	// === 3. Upstream errors by account ===
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(a.name, 'unknown') AS account_name,
			COALESCE(NULLIF(a.upstream_provider, ''), NULLIF(a.platform, ''), 'unknown') AS provider,
			COALESCE(e.upstream_status_code, 0) AS upstream_status,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE e.status_code = 200) AS recovered,
			COUNT(*) FILTER (WHERE e.status_code != 200) AS client_facing
		FROM ops_error_logs e
		LEFT JOIN accounts a ON a.id = e.account_id
		WHERE e.created_at >= NOW() - $1::interval
			AND e.error_phase = 'upstream'
		GROUP BY a.name, COALESCE(NULLIF(a.upstream_provider, ''), NULLIF(a.platform, ''), 'unknown'), e.upstream_status_code
		ORDER BY total DESC
		LIMIT 30
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("upstream errors: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var s service.UpstreamErrorStat
		if err := rows.Scan(&s.Account, &s.Provider, &s.UpstreamStatus, &s.Total, &s.Recovered, &s.ClientFacing); err != nil {
			return nil, err
		}
		report.UpstreamErrors = append(report.UpstreamErrors, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// === 4. Client-facing errors (what users actually see) ===
	rows2, err := r.db.QueryContext(ctx, `
		SELECT
			e.status_code,
			e.error_phase,
			LEFT(e.error_message, 120) AS err_msg,
			COUNT(*) AS cnt
		FROM ops_error_logs e
		WHERE e.created_at >= NOW() - $1::interval
			AND e.status_code != 200
		GROUP BY e.status_code, e.error_phase, LEFT(e.error_message, 120)
		ORDER BY cnt DESC
		LIMIT 20
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("client errors: %w", err)
	}
	defer func() { _ = rows2.Close() }()
	for rows2.Next() {
		var s service.ClientErrorStat
		if err := rows2.Scan(&s.StatusCode, &s.ErrorPhase, &s.ErrorMessage, &s.Count); err != nil {
			return nil, err
		}
		report.ClientErrors = append(report.ClientErrors, s)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	// === 5. Failover path details (recent failover chains) ===
	rows3, err := r.db.QueryContext(ctx, `
		SELECT
			e.request_id,
			e.model,
			e.status_code AS final_status,
			LEFT(e.error_message, 150) AS final_error,
			e.upstream_errors,
			e.duration_ms,
			e.created_at
		FROM ops_error_logs e
		WHERE e.created_at >= NOW() - $1::interval
			AND e.upstream_errors IS NOT NULL
			AND jsonb_typeof(e.upstream_errors) = 'array'
			AND jsonb_array_length(e.upstream_errors) > 0
		ORDER BY e.created_at DESC
		LIMIT 20
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("failover paths: %w", err)
	}
	defer func() { _ = rows3.Close() }()
	for rows3.Next() {
		var fp service.FailoverPath
		var upstreamErrorsJSON []byte
		var durationMs *int
		if err := rows3.Scan(&fp.RequestID, &fp.Model, &fp.FinalStatus, &fp.FinalError, &upstreamErrorsJSON, &durationMs, &fp.CreatedAt); err != nil {
			return nil, err
		}
		if durationMs != nil {
			fp.DurationMs = *durationMs
		}
		fp.UpstreamErrorsRaw = string(upstreamErrorsJSON)
		report.FailoverPaths = append(report.FailoverPaths, fp)
	}
	if err := rows3.Err(); err != nil {
		return nil, err
	}

	// Backfill account names in failover paths from accounts table
	report.FailoverPaths = r.backfillFailoverAccountNames(ctx, report.FailoverPaths)

	// === 6. Per-provider latency ===
	rows4, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(a.upstream_provider, a.platform, 'unknown') AS provider,
			COUNT(*) AS total,
			COALESCE(ROUND(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY u.duration_ms))::int, 0) AS p50,
			COALESCE(ROUND(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY u.duration_ms))::int, 0) AS p90,
			COALESCE(ROUND(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY u.duration_ms))::int, 0) AS p99,
			ROUND(AVG(u.first_token_ms) FILTER (WHERE u.first_token_ms IS NOT NULL))::int AS ttfb_avg
		FROM usage_logs u
		JOIN accounts a ON a.id = u.account_id
		WHERE u.created_at >= NOW() - $1::interval
			AND u.duration_ms > 0
		GROUP BY COALESCE(a.upstream_provider, a.platform, 'unknown')
		HAVING COUNT(*) >= 5
		ORDER BY total DESC
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("provider latency: %w", err)
	}
	defer func() { _ = rows4.Close() }()
	for rows4.Next() {
		var s service.ProviderSLALatency
		if err := rows4.Scan(&s.Provider, &s.Total, &s.P50Ms, &s.P90Ms, &s.P99Ms, &s.TTFBAvgMs); err != nil {
			return nil, err
		}
		report.ProviderLatency = append(report.ProviderLatency, s)
	}
	if err := rows4.Err(); err != nil {
		return nil, err
	}

	// === 7. Per-account success rate ===
	rows5, err := r.db.QueryContext(ctx, `
		WITH account_success AS (
			SELECT
				u.account_id,
				COUNT(*) AS success_count
			FROM usage_logs u
			WHERE u.created_at >= NOW() - $1::interval
				AND u.duration_ms > 0
			GROUP BY u.account_id
		),
		account_errors AS (
			SELECT
				e.account_id,
				COUNT(*) AS error_count
			FROM ops_error_logs e
			WHERE e.created_at >= NOW() - $1::interval
				AND e.status_code != 200
				AND e.account_id IS NOT NULL
			GROUP BY e.account_id
		)
		SELECT
			COALESCE(s.account_id, f.account_id) AS account_id,
			COALESCE(a.name, 'unknown') AS account_name,
			COALESCE(NULLIF(a.upstream_provider, ''), NULLIF(a.platform, ''), 'unknown') AS provider,
			COALESCE(s.success_count, 0) AS successful,
			COALESCE(f.error_count, 0) AS failed
		FROM account_success s
		FULL OUTER JOIN account_errors f ON s.account_id = f.account_id
		LEFT JOIN accounts a ON a.id = COALESCE(s.account_id, f.account_id)
		WHERE COALESCE(s.success_count, 0) + COALESCE(f.error_count, 0) >= 1
		ORDER BY (COALESCE(s.success_count, 0) + COALESCE(f.error_count, 0)) DESC
		LIMIT 50
	`, interval)
	if err != nil {
		return nil, fmt.Errorf("account success rate: %w", err)
	}
	defer func() { _ = rows5.Close() }()
	for rows5.Next() {
		var s service.AccountSuccessRate
		if err := rows5.Scan(&s.AccountID, &s.AccountName, &s.Provider, &s.Successful, &s.Failed); err != nil {
			return nil, err
		}
		s.Total = s.Successful + s.Failed
		if s.Total > 0 {
			s.SuccessRate = float64(s.Successful) / float64(s.Total) * 100
		}
		report.AccountSuccessRate = append(report.AccountSuccessRate, s)
	}
	if err := rows5.Err(); err != nil {
		return nil, err
	}

	return report, nil
}

// backfillFailoverAccountNames parses upstream_errors JSON in each FailoverPath,
// collects account IDs that are missing names, looks them up from the accounts table,
// and re-serializes the JSON with names filled in.
func (r *opsRepository) backfillFailoverAccountNames(ctx context.Context, paths []service.FailoverPath) []service.FailoverPath {
	if len(paths) == 0 {
		return paths
	}

	// Collect all account IDs that need name lookup
	needIDs := make(map[int64]struct{})
	for _, fp := range paths {
		var events []map[string]any
		if err := json.Unmarshal([]byte(fp.UpstreamErrorsRaw), &events); err != nil {
			continue
		}
		for _, ev := range events {
			name, _ := ev["account_name"].(string)
			if name != "" {
				continue
			}
			var accountID int64
			switch v := ev["account_id"].(type) {
			case float64:
				accountID = int64(v)
			case int64:
				accountID = v
			}
			if accountID > 0 {
				needIDs[accountID] = struct{}{}
			}
		}
	}
	if len(needIDs) == 0 {
		return paths
	}

	// Batch lookup account names
	nameMap := make(map[int64]string, len(needIDs))
	ids := make([]int64, 0, len(needIDs))
	for id := range needIDs {
		ids = append(ids, id)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, name FROM accounts WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return paths
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err == nil {
			nameMap[id] = name
		}
	}
	if err := rows.Err(); err != nil {
		return paths
	}

	// Backfill names into JSON
	for i, fp := range paths {
		var events []map[string]any
		if err := json.Unmarshal([]byte(fp.UpstreamErrorsRaw), &events); err != nil {
			continue
		}
		changed := false
		for _, ev := range events {
			name, _ := ev["account_name"].(string)
			if name != "" {
				continue
			}
			var accountID int64
			switch v := ev["account_id"].(type) {
			case float64:
				accountID = int64(v)
			case int64:
				accountID = v
			}
			if accountID > 0 {
				if n, ok := nameMap[accountID]; ok && n != "" {
					ev["account_name"] = n
					changed = true
				}
			}
		}
		if changed {
			if raw, err := json.Marshal(events); err == nil {
				paths[i].UpstreamErrorsRaw = string(raw)
			}
		}
	}

	return paths
}
