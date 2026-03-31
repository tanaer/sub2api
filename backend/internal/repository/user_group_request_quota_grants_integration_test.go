//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserGroupRateRepositoryIncrementRequestQuotaUsed_ConsumesGrantsBeforePermanentByExpiry(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUserGroupRateRepository(integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("user-group-rq-grant-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("grant-priority-%d", time.Now().UnixNano()),
		Platform:         service.PlatformOpenAI,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	})

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_group_request_quotas (user_id, group_id, request_quota, request_quota_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, group.ID, 4, 0)
	require.NoError(t, err)

	expiredAt := time.Now().Add(-2 * time.Hour)
	soonAt := time.Now().Add(2 * time.Hour)
	laterAt := time.Now().Add(24 * time.Hour)

	require.NoError(t, repo.CreateRequestQuotaGrant(ctx, &service.UserGroupRequestQuotaGrant{
		UserID:            user.ID,
		GroupID:           group.ID,
		RequestQuotaTotal: 9,
		RequestQuotaUsed:  0,
		ExpiresAt:         expiredAt,
	}))
	require.NoError(t, repo.CreateRequestQuotaGrant(ctx, &service.UserGroupRequestQuotaGrant{
		UserID:            user.ID,
		GroupID:           group.ID,
		RequestQuotaTotal: 2,
		RequestQuotaUsed:  0,
		ExpiresAt:         soonAt,
	}))
	require.NoError(t, repo.CreateRequestQuotaGrant(ctx, &service.UserGroupRequestQuotaGrant{
		UserID:            user.ID,
		GroupID:           group.ID,
		RequestQuotaTotal: 3,
		RequestQuotaUsed:  0,
		ExpiresAt:         laterAt,
	}))

	applied, err := repo.IncrementRequestQuotaUsed(ctx, user.ID, group.ID, 6)
	require.NoError(t, err)
	require.True(t, applied)

	rows, err := integrationDB.QueryContext(ctx, `
		SELECT request_quota_total, request_quota_used, expires_at
		FROM user_group_request_quota_grants
		WHERE user_id = $1 AND group_id = $2
		ORDER BY expires_at ASC, id ASC
	`, user.ID, group.ID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	type grantUsageRow struct {
		total     int64
		used      int64
		expiresAt time.Time
	}
	var grantRows []grantUsageRow
	for rows.Next() {
		var row grantUsageRow
		require.NoError(t, rows.Scan(&row.total, &row.used, &row.expiresAt))
		grantRows = append(grantRows, row)
	}
	require.NoError(t, rows.Err())
	require.Len(t, grantRows, 3)
	require.Equal(t, int64(9), grantRows[0].total)
	require.Equal(t, int64(0), grantRows[0].used)
	require.Equal(t, int64(2), grantRows[1].total)
	require.Equal(t, int64(2), grantRows[1].used)
	require.Equal(t, int64(3), grantRows[2].total)
	require.Equal(t, int64(3), grantRows[2].used)

	var permanentUsed int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT request_quota_used
		FROM user_group_request_quotas
		WHERE user_id = $1 AND group_id = $2
	`, user.ID, group.ID).Scan(&permanentUsed))
	require.Equal(t, int64(1), permanentUsed)

	quota, err := repo.GetRequestQuotaByUserAndGroup(ctx, user.ID, group.ID)
	require.NoError(t, err)
	require.NotNil(t, quota)
	require.Equal(t, int64(9), quota.RequestQuota)
	require.Equal(t, int64(6), quota.RequestQuotaUsed)
	require.Equal(t, int64(3), quota.Remaining())
}
