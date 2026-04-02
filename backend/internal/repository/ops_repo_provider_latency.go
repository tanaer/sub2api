package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetProviderLatencyStats returns latency percentile stats grouped by upstream_provider.
// Only considers successful requests (duration_ms IS NOT NULL) in the given time window.
func (r *opsRepository) GetProviderLatencyStats(ctx context.Context, hours int) ([]*service.ProviderLatencyStats, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if hours < 1 {
		hours = 1
	}
	if hours > 168 {
		hours = 168 // max 7 days
	}

	q := `
SELECT
  COALESCE(a.upstream_provider, a.platform) AS provider,
  COUNT(*) AS cnt,
  COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY ul.duration_ms), 0)::int AS p50_ms,
  COALESCE(PERCENTILE_CONT(0.90) WITHIN GROUP (ORDER BY ul.duration_ms), 0)::int AS p90_ms,
  COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY ul.duration_ms), 0)::int AS p99_ms,
  COALESCE(AVG(ul.duration_ms), 0)::int AS avg_ms,
  COALESCE(MAX(ul.duration_ms), 0)::int AS max_ms
FROM usage_logs ul
JOIN accounts a ON a.id = ul.account_id
WHERE ul.created_at >= NOW() - ($1 || ' hours')::interval
  AND ul.duration_ms IS NOT NULL
  AND (a.upstream_provider IS NOT NULL AND a.upstream_provider != '' OR a.platform NOT IN ('anthropic'))
GROUP BY COALESCE(a.upstream_provider, a.platform)
HAVING COUNT(*) >= 5
ORDER BY cnt DESC`

	rows, err := r.db.QueryContext(ctx, q, hours)
	if err != nil {
		return nil, fmt.Errorf("query provider latency stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []*service.ProviderLatencyStats
	for rows.Next() {
		var s service.ProviderLatencyStats
		if err := rows.Scan(&s.Provider, &s.Count, &s.P50Ms, &s.P90Ms, &s.P99Ms, &s.AvgMs, &s.MaxMs); err != nil {
			return nil, fmt.Errorf("scan provider latency stats: %w", err)
		}
		result = append(result, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider latency stats: %w", err)
	}

	return result, nil
}
