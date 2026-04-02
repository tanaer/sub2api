package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type userGroupRateRepository struct {
	sql sqlExecutor
}

// NewUserGroupRateRepository 创建用户专属分组倍率仓储
func NewUserGroupRateRepository(sqlDB *sql.DB) service.UserGroupRateRepository {
	return &userGroupRateRepository{sql: sqlDB}
}

// GetByUserID 获取用户的所有专属分组倍率
func (r *userGroupRateRepository) GetByUserID(ctx context.Context, userID int64) (map[int64]float64, error) {
	query := `SELECT group_id, rate_multiplier FROM user_group_rate_multipliers WHERE user_id = $1`
	rows, err := r.sql.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]float64)
	for rows.Next() {
		var groupID int64
		var rate float64
		if err := rows.Scan(&groupID, &rate); err != nil {
			return nil, err
		}
		result[groupID] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetByUserIDs 批量获取多个用户的专属分组倍率。
// 返回结构：map[userID]map[groupID]rate
func (r *userGroupRateRepository) GetByUserIDs(ctx context.Context, userIDs []int64) (map[int64]map[int64]float64, error) {
	result := make(map[int64]map[int64]float64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	uniqueIDs := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		uniqueIDs = append(uniqueIDs, userID)
		result[userID] = make(map[int64]float64)
	}
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	rows, err := r.sql.QueryContext(ctx, `
		SELECT user_id, group_id, rate_multiplier
		FROM user_group_rate_multipliers
		WHERE user_id = ANY($1)
	`, pq.Array(uniqueIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID int64
		var groupID int64
		var rate float64
		if err := rows.Scan(&userID, &groupID, &rate); err != nil {
			return nil, err
		}
		if _, ok := result[userID]; !ok {
			result[userID] = make(map[int64]float64)
		}
		result[userID][groupID] = rate
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetByGroupID 获取指定分组下所有用户的专属倍率
func (r *userGroupRateRepository) GetByGroupID(ctx context.Context, groupID int64) ([]service.UserGroupRateEntry, error) {
	query := `
		SELECT ugr.user_id, u.username, u.email, COALESCE(u.notes, ''), u.status, ugr.rate_multiplier
		FROM user_group_rate_multipliers ugr
		JOIN users u ON u.id = ugr.user_id
		WHERE ugr.group_id = $1
		ORDER BY ugr.user_id
	`
	rows, err := r.sql.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []service.UserGroupRateEntry
	for rows.Next() {
		var entry service.UserGroupRateEntry
		if err := rows.Scan(&entry.UserID, &entry.UserName, &entry.UserEmail, &entry.UserNotes, &entry.UserStatus, &entry.RateMultiplier); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetByUserAndGroup 获取用户在特定分组的专属倍率
func (r *userGroupRateRepository) GetByUserAndGroup(ctx context.Context, userID, groupID int64) (*float64, error) {
	query := `SELECT rate_multiplier FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = $2`
	var rate float64
	err := scanSingleRow(ctx, r.sql, query, []any{userID, groupID}, &rate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rate, nil
}

// SyncUserGroupRates 同步用户的分组专属倍率
func (r *userGroupRateRepository) SyncUserGroupRates(ctx context.Context, userID int64, rates map[int64]*float64) error {
	if len(rates) == 0 {
		// 如果传入空 map，删除该用户的所有专属倍率
		_, err := r.sql.ExecContext(ctx, `DELETE FROM user_group_rate_multipliers WHERE user_id = $1`, userID)
		return err
	}

	// 分离需要删除和需要 upsert 的记录
	var toDelete []int64
	upsertGroupIDs := make([]int64, 0, len(rates))
	upsertRates := make([]float64, 0, len(rates))
	for groupID, rate := range rates {
		if rate == nil {
			toDelete = append(toDelete, groupID)
		} else {
			upsertGroupIDs = append(upsertGroupIDs, groupID)
			upsertRates = append(upsertRates, *rate)
		}
	}

	// 删除指定的记录
	if len(toDelete) > 0 {
		if _, err := r.sql.ExecContext(ctx,
			`DELETE FROM user_group_rate_multipliers WHERE user_id = $1 AND group_id = ANY($2)`,
			userID, pq.Array(toDelete)); err != nil {
			return err
		}
	}

	// Upsert 记录
	now := time.Now()
	if len(upsertGroupIDs) > 0 {
		_, err := r.sql.ExecContext(ctx, `
			INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
			SELECT
				$1::bigint,
				data.group_id,
				data.rate_multiplier,
				$2::timestamptz,
				$2::timestamptz
			FROM unnest($3::bigint[], $4::double precision[]) AS data(group_id, rate_multiplier)
			ON CONFLICT (user_id, group_id)
			DO UPDATE SET
				rate_multiplier = EXCLUDED.rate_multiplier,
				updated_at = EXCLUDED.updated_at
		`, userID, now, pq.Array(upsertGroupIDs), pq.Array(upsertRates))
		if err != nil {
			return err
		}
	}

	return nil
}

// SyncGroupRateMultipliers 批量同步分组的用户专属倍率（先删后插）
func (r *userGroupRateRepository) SyncGroupRateMultipliers(ctx context.Context, groupID int64, entries []service.GroupRateMultiplierInput) error {
	if _, err := r.sql.ExecContext(ctx, `DELETE FROM user_group_rate_multipliers WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}
	userIDs := make([]int64, len(entries))
	rates := make([]float64, len(entries))
	for i, e := range entries {
		userIDs[i] = e.UserID
		rates[i] = e.RateMultiplier
	}
	now := time.Now()
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO user_group_rate_multipliers (user_id, group_id, rate_multiplier, created_at, updated_at)
		SELECT data.user_id, $1::bigint, data.rate_multiplier, $2::timestamptz, $2::timestamptz
		FROM unnest($3::bigint[], $4::double precision[]) AS data(user_id, rate_multiplier)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET rate_multiplier = EXCLUDED.rate_multiplier, updated_at = EXCLUDED.updated_at
	`, groupID, now, pq.Array(userIDs), pq.Array(rates))
	return err
}

// DeleteByGroupID 删除指定分组的所有用户专属倍率
func (r *userGroupRateRepository) DeleteByGroupID(ctx context.Context, groupID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM user_group_rate_multipliers WHERE group_id = $1`, groupID)
	return err
}

// DeleteByUserID 删除指定用户的所有专属倍率
func (r *userGroupRateRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM user_group_rate_multipliers WHERE user_id = $1`, userID)
	return err
}

// GetRequestQuotasByUserID 获取用户的所有分组按次配额配置。
func (r *userGroupRateRepository) GetRequestQuotasByUserID(ctx context.Context, userID int64) (map[int64]int64, error) {
	rows, err := sqlExecutorFromContext(ctx, r.sql).QueryContext(ctx, `
		WITH permanent AS (
			SELECT group_id, request_quota
			FROM user_group_request_quotas
			WHERE user_id = $1
		),
		active_grants AS (
			SELECT group_id, COALESCE(SUM(request_quota_total - request_quota_used), 0) AS active_quota
			FROM user_group_request_quota_grants
			WHERE user_id = $1
				AND expires_at > NOW()
				AND request_quota_total > request_quota_used
			GROUP BY group_id
		)
		SELECT
			COALESCE(p.group_id, g.group_id) AS group_id,
			COALESCE(p.request_quota, 0) + COALESCE(g.active_quota, 0) AS request_quota
		FROM permanent p
		FULL OUTER JOIN active_grants g ON g.group_id = p.group_id
		WHERE COALESCE(p.request_quota, 0) + COALESCE(g.active_quota, 0) > 0
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]int64)
	for rows.Next() {
		var groupID int64
		var requestQuota int64
		if err := rows.Scan(&groupID, &requestQuota); err != nil {
			return nil, err
		}
		result[groupID] = requestQuota
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRequestQuotasByUserIDs 批量获取多个用户的分组按次配额剩余。
// 返回结构：map[userID]map[groupID]remainingQuota
func (r *userGroupRateRepository) GetRequestQuotasByUserIDs(ctx context.Context, userIDs []int64) (map[int64]map[int64]int64, error) {
	result := make(map[int64]map[int64]int64, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	uniqueIDs := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, uid := range userIDs {
		if uid <= 0 {
			continue
		}
		if _, exists := seen[uid]; exists {
			continue
		}
		seen[uid] = struct{}{}
		uniqueIDs = append(uniqueIDs, uid)
	}
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	rows, err := sqlExecutorFromContext(ctx, r.sql).QueryContext(ctx, `
		WITH permanent AS (
			SELECT user_id, group_id, request_quota
			FROM user_group_request_quotas
			WHERE user_id = ANY($1)
		),
		active_grants AS (
			SELECT user_id, group_id, COALESCE(SUM(request_quota_total - request_quota_used), 0) AS active_quota
			FROM user_group_request_quota_grants
			WHERE user_id = ANY($1)
				AND expires_at > NOW()
				AND request_quota_total > request_quota_used
			GROUP BY user_id, group_id
		)
		SELECT
			COALESCE(p.user_id, g.user_id) AS user_id,
			COALESCE(p.group_id, g.group_id) AS group_id,
			COALESCE(p.request_quota, 0) + COALESCE(g.active_quota, 0) AS request_quota
		FROM permanent p
		FULL OUTER JOIN active_grants g ON g.user_id = p.user_id AND g.group_id = p.group_id
		WHERE COALESCE(p.request_quota, 0) + COALESCE(g.active_quota, 0) > 0
	`, pq.Array(uniqueIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID, groupID, quota int64
		if err := rows.Scan(&userID, &groupID, &quota); err != nil {
			return nil, err
		}
		if _, ok := result[userID]; !ok {
			result[userID] = make(map[int64]int64)
		}
		result[userID][groupID] = quota
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRequestQuotaByUserAndGroup 获取用户在指定分组下的按次配额状态。
func (r *userGroupRateRepository) GetRequestQuotaByUserAndGroup(ctx context.Context, userID, groupID int64) (*service.UserGroupRequestQuota, error) {
	quota := &service.UserGroupRequestQuota{}
	err := scanSingleRow(ctx, sqlExecutorFromContext(ctx, r.sql), `
		WITH permanent AS (
			SELECT request_quota, request_quota_used
			FROM user_group_request_quotas
			WHERE user_id = $1 AND group_id = $2
		),
		active_grants AS (
			SELECT
				COALESCE(SUM(request_quota_total), 0) AS request_quota,
				COALESCE(SUM(request_quota_used), 0) AS request_quota_used
			FROM user_group_request_quota_grants
			WHERE user_id = $1
				AND group_id = $2
				AND expires_at > NOW()
		)
		SELECT
			COALESCE((SELECT request_quota FROM permanent), 0) + COALESCE((SELECT request_quota FROM active_grants), 0) AS request_quota,
			COALESCE((SELECT request_quota_used FROM permanent), 0) + COALESCE((SELECT request_quota_used FROM active_grants), 0) AS request_quota_used
	`, []any{userID, groupID}, &quota.RequestQuota, &quota.RequestQuotaUsed)
	if err == sql.ErrNoRows || quota.RequestQuota <= 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return quota, nil
}

// SyncUserGroupRequestQuotas 同步用户的分组按次配额配置。
func (r *userGroupRateRepository) SyncUserGroupRequestQuotas(ctx context.Context, userID int64, quotas map[int64]*int64) error {
	sqlq := sqlExecutorFromContext(ctx, r.sql)
	if len(quotas) == 0 {
		_, err := sqlq.ExecContext(ctx, `DELETE FROM user_group_request_quotas WHERE user_id = $1`, userID)
		return err
	}

	var toDelete []int64
	upsertGroupIDs := make([]int64, 0, len(quotas))
	upsertRequestQuotas := make([]int64, 0, len(quotas))
	for groupID, quota := range quotas {
		if quota == nil || *quota <= 0 {
			toDelete = append(toDelete, groupID)
			continue
		}
		upsertGroupIDs = append(upsertGroupIDs, groupID)
		upsertRequestQuotas = append(upsertRequestQuotas, *quota)
	}

	if len(toDelete) > 0 {
		if _, err := sqlq.ExecContext(ctx, `
			DELETE FROM user_group_request_quotas
			WHERE user_id = $1 AND group_id = ANY($2)
		`, userID, pq.Array(toDelete)); err != nil {
			return err
		}
	}

	if len(upsertGroupIDs) == 0 {
		return nil
	}

	now := time.Now()
	_, err := sqlq.ExecContext(ctx, `
		INSERT INTO user_group_request_quotas (user_id, group_id, request_quota, created_at, updated_at)
		SELECT
			$1::bigint,
			data.group_id,
			data.request_quota,
			$2::timestamptz,
			$2::timestamptz
		FROM unnest($3::bigint[], $4::bigint[]) AS data(group_id, request_quota)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET
			request_quota = EXCLUDED.request_quota,
			updated_at = EXCLUDED.updated_at
	`, userID, now, pq.Array(upsertGroupIDs), pq.Array(upsertRequestQuotas))
	return err
}

// IncrementRequestQuotaUsed 原子增加用户分组按次配额的已用次数。
func (r *userGroupRateRepository) IncrementRequestQuotaUsed(ctx context.Context, userID, groupID, amount int64) (bool, error) {
	if amount <= 0 {
		return false, nil
	}

	if dbent.TxFromContext(ctx) != nil {
		return incrementUserGroupRequestQuotaWithExecutor(ctx, sqlExecutorFromContext(ctx, r.sql), userID, groupID, amount)
	}

	db, ok := r.sql.(*sql.DB)
	if !ok {
		return false, errors.New("user group request quota repository requires *sql.DB for atomic increment")
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	applied, err := incrementUserGroupRequestQuotaWithExecutor(ctx, tx, userID, groupID, amount)
	if err != nil {
		return false, err
	}
	if !applied {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (r *userGroupRateRepository) CreateRequestQuotaGrant(ctx context.Context, grant *service.UserGroupRequestQuotaGrant) error {
	if grant == nil {
		return nil
	}
	sqlq := sqlExecutorFromContext(ctx, r.sql)
	now := time.Now()
	if grant.CreatedAt.IsZero() {
		grant.CreatedAt = now
	}
	if grant.UpdatedAt.IsZero() {
		grant.UpdatedAt = grant.CreatedAt
	}

	var redeemCodeID any
	if grant.RedeemCodeID != nil && *grant.RedeemCodeID > 0 {
		redeemCodeID = *grant.RedeemCodeID
	}

	row, err := sqlq.QueryContext(ctx, `
		INSERT INTO user_group_request_quota_grants (
			user_id,
			group_id,
			redeem_code_id,
			request_quota_total,
			request_quota_used,
			expires_at,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, grant.UserID, grant.GroupID, redeemCodeID, grant.RequestQuotaTotal, grant.RequestQuotaUsed, grant.ExpiresAt, grant.CreatedAt, grant.UpdatedAt)
	if err != nil {
		return err
	}
	defer func() { _ = row.Close() }()
	if row.Next() {
		if err := row.Scan(&grant.ID, &grant.CreatedAt, &grant.UpdatedAt); err != nil {
			return err
		}
	}
	return row.Err()
}

type userGroupRequestQuotaGrantRow struct {
	ID                int64
	RequestQuotaTotal int64
	RequestQuotaUsed  int64
}

func incrementUserGroupRequestQuotaWithExecutor(ctx context.Context, sqlq sqlExecutor, userID, groupID, amount int64) (bool, error) {
	if amount <= 0 {
		return false, nil
	}

	grantRows, grantAvailable, err := lockActiveUserGroupRequestQuotaGrants(ctx, sqlq, userID, groupID)
	if err != nil {
		return false, err
	}
	permanentQuota, permanentUsed, hasPermanentQuota, err := lockPermanentUserGroupRequestQuota(ctx, sqlq, userID, groupID)
	if err != nil {
		return false, err
	}
	permanentAvailable := permanentQuota - permanentUsed
	if permanentAvailable < 0 {
		permanentAvailable = 0
	}
	if grantAvailable+permanentAvailable < amount {
		return false, nil
	}

	remaining := amount
	for _, grant := range grantRows {
		if remaining <= 0 {
			break
		}
		available := grant.RequestQuotaTotal - grant.RequestQuotaUsed
		if available <= 0 {
			continue
		}
		consume := available
		if consume > remaining {
			consume = remaining
		}
		if _, err := sqlq.ExecContext(ctx, `
			UPDATE user_group_request_quota_grants
			SET request_quota_used = request_quota_used + $2,
				updated_at = NOW()
			WHERE id = $1
		`, grant.ID, consume); err != nil {
			return false, err
		}
		remaining -= consume
	}

	if remaining > 0 && hasPermanentQuota {
		if _, err := sqlq.ExecContext(ctx, `
			UPDATE user_group_request_quotas
			SET request_quota_used = request_quota_used + $3,
				updated_at = NOW()
			WHERE user_id = $1 AND group_id = $2
		`, userID, groupID, remaining); err != nil {
			return false, err
		}
	}

	return true, nil
}

func lockActiveUserGroupRequestQuotaGrants(ctx context.Context, sqlq sqlExecutor, userID, groupID int64) ([]userGroupRequestQuotaGrantRow, int64, error) {
	rows, err := sqlq.QueryContext(ctx, `
		SELECT id, request_quota_total, request_quota_used
		FROM user_group_request_quota_grants
		WHERE user_id = $1
			AND group_id = $2
			AND expires_at > NOW()
			AND request_quota_total > request_quota_used
		ORDER BY expires_at ASC, id ASC
		FOR UPDATE
	`, userID, groupID)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	result := make([]userGroupRequestQuotaGrantRow, 0)
	var totalAvailable int64
	for rows.Next() {
		var row userGroupRequestQuotaGrantRow
		if err := rows.Scan(&row.ID, &row.RequestQuotaTotal, &row.RequestQuotaUsed); err != nil {
			return nil, 0, err
		}
		result = append(result, row)
		totalAvailable += row.RequestQuotaTotal - row.RequestQuotaUsed
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return result, totalAvailable, nil
}

func lockPermanentUserGroupRequestQuota(ctx context.Context, sqlq sqlExecutor, userID, groupID int64) (quota int64, used int64, found bool, err error) {
	err = scanSingleRow(ctx, sqlq, `
		SELECT request_quota, request_quota_used
		FROM user_group_request_quotas
		WHERE user_id = $1 AND group_id = $2
		FOR UPDATE
	`, []any{userID, groupID}, &quota, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, err
	}
	return quota, used, true, nil
}
