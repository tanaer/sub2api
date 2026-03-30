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

func TestUsageLogRepositoryGetUserDashboardStats_IncludesGroupRequestQuotas(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewUsageLogRepository(client, integrationDB)

	user := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("dashboard-group-rq-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Balance:      12,
	})
	group := mustCreateGroup(t, client, &service.Group{
		Name:             "glm-5",
		Platform:         service.PlatformOpenAI,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	})

	_, err := integrationDB.ExecContext(ctx, `
		INSERT INTO user_group_request_quotas (user_id, group_id, request_quota, request_quota_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, user.ID, group.ID, 9, 5)
	require.NoError(t, err)

	stats, err := repo.GetUserDashboardStats(ctx, user.ID)
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Len(t, stats.GroupRequestQuotas, 1)
	require.Equal(t, group.ID, stats.GroupRequestQuotas[0].GroupID)
	require.Equal(t, group.Name, stats.GroupRequestQuotas[0].GroupName)
	require.Equal(t, int64(9), stats.GroupRequestQuotas[0].RequestQuota)
	require.Equal(t, int64(5), stats.GroupRequestQuotas[0].RequestQuotaUsed)
	require.Equal(t, int64(4), stats.GroupRequestQuotas[0].RequestQuotaRemaining)
}
