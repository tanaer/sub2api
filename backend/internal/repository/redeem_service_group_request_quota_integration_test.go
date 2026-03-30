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

func TestRedeemServiceRedeem_GroupRequestQuotaAccumulatesQuotaAndAddsAllowedGroup(t *testing.T) {
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
		Code:    fmt.Sprintf("RQ-%d", time.Now().UnixNano()),
		Type:    "group_request_quota",
		Value:   5,
		Status:  service.StatusUnused,
		GroupID: &group.ID,
	}
	require.NoError(t, redeemRepo.Create(ctx, code))

	result, err := redeemService.Redeem(ctx, user.ID, code.Code)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "group_request_quota", result.Type)
	require.NotNil(t, result.GroupID)
	require.Equal(t, group.ID, *result.GroupID)

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
