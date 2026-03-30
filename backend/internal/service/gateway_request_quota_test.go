//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildUsageBillingCommand_PrefersRequestQuotaWhenAvailable(t *testing.T) {
	cmd := buildUsageBillingCommand("req-request-quota", &UsageLog{
		Model: "glm-5",
	}, &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: 1.5,
			TotalCost:  1.5,
		},
		User: &User{ID: 1},
		APIKey: &APIKey{
			ID:               2,
			RequestQuota:     10,
			RequestQuotaUsed: 4,
			Quota:            100,
			RateLimit5h:      100,
		},
		Account: &Account{
			ID:   3,
			Type: AccountTypeOAuth,
		},
		APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
	})

	require.NotNil(t, cmd)
	require.Equal(t, int64(1), cmd.APIKeyRequestQuotaCount)
	require.Zero(t, cmd.BalanceCost)
	require.Zero(t, cmd.SubscriptionCost)
	require.Zero(t, cmd.APIKeyQuotaCost)
	require.Zero(t, cmd.APIKeyRateLimitCost)
}

func TestBuildUsageBillingCommand_FallsBackToExistingBillingWhenRequestQuotaExhausted(t *testing.T) {
	cmd := buildUsageBillingCommand("req-request-quota-exhausted", &UsageLog{
		Model: "glm-5",
	}, &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: 2.5,
			TotalCost:  2.5,
		},
		User: &User{ID: 1},
		APIKey: &APIKey{
			ID:               2,
			RequestQuota:     10,
			RequestQuotaUsed: 10,
			Quota:            100,
			RateLimit5h:      100,
		},
		Account: &Account{
			ID:   3,
			Type: AccountTypeOAuth,
		},
		APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
	})

	require.NotNil(t, cmd)
	require.Zero(t, cmd.APIKeyRequestQuotaCount)
	require.InDelta(t, 2.5, cmd.BalanceCost, 0.000001)
	require.InDelta(t, 2.5, cmd.APIKeyQuotaCost, 0.000001)
	require.InDelta(t, 2.5, cmd.APIKeyRateLimitCost, 0.000001)
}

func TestBuildUsageBillingCommand_PrefersUserGroupRequestQuotaOverAPIKeyRequestQuota(t *testing.T) {
	cmd := buildUsageBillingCommand("req-user-group-request-quota", &UsageLog{
		Model: "glm-5",
	}, &postUsageBillingParams{
		Cost: &CostBreakdown{
			ActualCost: 1.2,
			TotalCost:  1.2,
		},
		User: &User{ID: 1},
		APIKey: &APIKey{
			ID:                        2,
			GroupID:                   i64p(88),
			RequestQuota:              10,
			RequestQuotaUsed:          4,
			UserGroupRequestQuota:     6,
			UserGroupRequestQuotaUsed: 1,
			Quota:                     100,
			RateLimit5h:               100,
		},
		Account: &Account{
			ID:   3,
			Type: AccountTypeOAuth,
		},
		APIKeyService: &openAIRecordUsageAPIKeyQuotaStub{},
	})

	require.NotNil(t, cmd)
	require.Equal(t, int64(88), cmd.UserGroupRequestQuotaGroupID)
	require.Equal(t, int64(1), cmd.UserGroupRequestQuotaCount)
	require.Zero(t, cmd.APIKeyRequestQuotaCount)
	require.Zero(t, cmd.BalanceCost)
	require.Zero(t, cmd.SubscriptionCost)
	require.Zero(t, cmd.APIKeyQuotaCost)
	require.Zero(t, cmd.APIKeyRateLimitCost)
}
