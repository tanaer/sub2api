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

func TestUsageBillingRepositoryApply_DeduplicatesRequestQuotaBilling(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageBillingRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("usage-billing-request-quota-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      100,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:           user.ID,
		Key:              "sk-usage-billing-request-quota-" + uuid.NewString(),
		Name:             "request-quota",
		RequestQuota:     5,
		RequestQuotaUsed: 1,
	})

	requestID := uuid.NewString()
	cmd := &service.UsageBillingCommand{
		RequestID:               requestID,
		APIKeyID:                apiKey.ID,
		UserID:                  user.ID,
		APIKeyRequestQuotaCount: 1,
		RequestPayloadHash:      "payload-hash-" + uuid.NewString(),
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
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT request_quota_used FROM api_keys WHERE id = $1", apiKey.ID).Scan(&used))
	require.Equal(t, int64(2), used)

	var balance float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.InDelta(t, 100, balance, 0.000001)
}
