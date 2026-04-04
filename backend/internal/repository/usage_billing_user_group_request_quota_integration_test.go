//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUsageBillingRepositoryApply_DeduplicatesUserGroupRequestQuotaBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-user-group-rq-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "user-group-rq",
		Platform:         service.PlatformOpenAI,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		GroupID: &group.ID,
		Key:     "sk-usage-billing-user-group-rq-" + uuid.NewString(),
		Name:    "user-group-request-quota",
	})

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_group_request_quotas (user_id, group_id, request_quota, request_quota_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, group.ID, 5, 1)
	require.NoError(t, err)

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:                    requestID,
		APIKeyID:                     apiKey.ID,
		UserID:                       user.ID,
		UserGroupRequestQuotaGroupID: group.ID,
		UserGroupRequestQuotaCount:   1,
		RequestPayloadHash:           "payload-hash-" + uuid.NewString(),
	}

	result1, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result1)
	require.True(t, result1.Applied)

	result2, err := repo.Apply(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, result2)
	require.False(t, result2.Applied)

	var used int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		SELECT request_quota_used
		FROM user_group_request_quotas
		WHERE user_id = $1 AND group_id = $2
	`, user.ID, group.ID).Scan(&used))
	require.Equal(t, int64(2), used)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 100, balance, 0.000001)
}
