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

func TestRedeemServiceRedeem_GroupRequestQuotaCreatesIndependentGrantWithExpiry(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)

	userRepo := NewUserRepository(client, integrationDB)
	redeemRepo := NewRedeemCodeRepository(client)
	userGroupRateRepo := NewUserGroupRateRepository(integrationDB)
	redeemService := service.NewRedeemService(
		redeemRepo,
		userRepo,
		nil,
		nil,
		nil,
		client,
		nil,
		userGroupRateRepo,
	)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("redeem-group-rq-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             fmt.Sprintf("redeem-rq-group-%d", time.Now().UnixNano()),
		Platform:         service.PlatformOpenAI,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
		IsExclusive:      true,
	})

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_group_request_quotas (user_id, group_id, request_quota, request_quota_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, group.ID, 3, 1)
	require.NoError(t, err)

	code := &service.RedeemCode{
		Code:         fmt.Sprintf("RQ-%d", time.Now().UnixNano()),
		Type:         "group_request_quota",
		Value:        5,
		Status:       service.StatusUnused,
		GroupID:      &group.ID,
		ValidityDays: 30,
	}
	require.NoError(t, redeemRepo.Create(ctx, code))

	result, err := redeemService.Redeem(ctx, user.ID, code.Code)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "group_request_quota", result.Type)
	require.NotNil(t, result.GroupID)
	require.Equal(t, group.ID, *result.GroupID)
	require.NotNil(t, result.UsedAt)
	require.NotNil(t, result.ExpiresAt)
	require.WithinDuration(t, result.UsedAt.Add(30*24*time.Hour), *result.ExpiresAt, 2*time.Second)

	var permanentQuotaTotal int64
	var permanentQuotaUsed int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT request_quota, request_quota_used
		FROM user_group_request_quotas
		WHERE user_id = $1 AND group_id = $2
	`, user.ID, group.ID).Scan(&permanentQuotaTotal, &permanentQuotaUsed))
	require.Equal(t, int64(3), permanentQuotaTotal)
	require.Equal(t, int64(1), permanentQuotaUsed)

	var grantCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_group_request_quota_grants
		WHERE user_id = $1 AND group_id = $2
	`, user.ID, group.ID).Scan(&grantCount))
	require.Equal(t, 1, grantCount)

	var grantRedeemCodeID int64
	var grantQuotaTotal int64
	var grantQuotaUsed int64
	var grantExpiresAt time.Time
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT redeem_code_id, request_quota_total, request_quota_used, expires_at
		FROM user_group_request_quota_grants
		WHERE user_id = $1 AND group_id = $2
	`, user.ID, group.ID).Scan(&grantRedeemCodeID, &grantQuotaTotal, &grantQuotaUsed, &grantExpiresAt))
	require.Equal(t, code.ID, grantRedeemCodeID)
	require.Equal(t, int64(5), grantQuotaTotal)
	require.Equal(t, int64(0), grantQuotaUsed)
	require.WithinDuration(t, *result.ExpiresAt, grantExpiresAt, 2*time.Second)

	quota, err := userGroupRateRepo.GetRequestQuotaByUserAndGroup(ctx, user.ID, group.ID)
	require.NoError(t, err)
	require.NotNil(t, quota)
	require.Equal(t, int64(8), quota.RequestQuota)
	require.Equal(t, int64(1), quota.RequestQuotaUsed)
	require.Equal(t, int64(7), quota.Remaining())

	updatedUser, err := userRepo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Contains(t, updatedUser.AllowedGroups, group.ID)
}
