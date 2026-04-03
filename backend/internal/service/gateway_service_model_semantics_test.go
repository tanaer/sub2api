//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectAccount_UsesResolvedLookupModel(t *testing.T) {
	ctx := context.Background()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"sonnet": "glm-4.5-air"}},
			},
			{
				ID:          2,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"model_mapping": map[string]any{"glm-4.5-air": "glm-4.5-air"}},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	svc := &GatewayService{
		accountRepo: repo,
		cache:       &mockGatewayCacheForPlatform{},
		cfg:         testConfig(),
	}

	acc, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "glm-4.5-air", nil, PlatformAnthropic)
	require.NoError(t, err)
	require.NotNil(t, acc)
	require.Equal(t, int64(2), acc.ID)
}

func TestAccountModelMapping_DoesNotPromoteAliasIntoSupport(t *testing.T) {
	acc := &Account{
		Platform: PlatformAnthropic,
		Credentials: map[string]any{
			"model_mapping": map[string]any{"sonnet": "glm-4.5-air"},
		},
	}
	require.False(t, acc.IsModelSupported("glm-4.5-air"))

	whitelist := &Account{
		Platform: PlatformAnthropic,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"glm-4.5-air": "glm-4.5-air",
				"gpt-4":       "gpt-4",
			},
		},
	}
	require.True(t, whitelist.IsModelSupported("glm-4.5-air"))
	require.False(t, whitelist.IsModelSupported("sonnet"))
}
