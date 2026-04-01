//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectAccountForModelWithPlatform_SkipsMiniMaxForGLMToolHistory(t *testing.T) {
	minimax := Account{
		ID:          8,
		Name:        "minimax",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Priority:    1,
		Credentials: map[string]any{
			"base_url": "https://api.minimaxi.com/anthropic",
		},
	}
	standard := Account{
		ID:          3,
		Name:        "glm-upstream",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Priority:    2,
	}

	repo := &mockAccountRepoForPlatform{
		accounts:     []Account{minimax, standard},
		accountsByID: map[int64]*Account{8: &minimax, 3: &standard},
	}
	svc := &GatewayService{
		accountRepo: repo,
		cfg:         testConfig(),
	}

	ctx := WithGLMRequestTraits(context.Background(), GLMRequestTraits{HasToolResult: true})
	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "glm-5.1", map[int64]struct{}{}, PlatformAnthropic)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(3), account.ID)
}

func TestSelectAccountForModelWithPlatform_GLMToolHistoryOnlyMiniMaxReturnsNoAccounts(t *testing.T) {
	minimax := Account{
		ID:          8,
		Name:        "minimax",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Priority:    1,
		Credentials: map[string]any{
			"base_url": "https://api.minimaxi.com/anthropic",
		},
	}

	repo := &mockAccountRepoForPlatform{
		accounts:     []Account{minimax},
		accountsByID: map[int64]*Account{8: &minimax},
	}
	svc := &GatewayService{
		accountRepo: repo,
		cfg:         testConfig(),
	}

	ctx := WithGLMRequestTraits(context.Background(), GLMRequestTraits{HasToolResult: true})
	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "glm-5.1", map[int64]struct{}{}, PlatformAnthropic)
	require.Nil(t, account)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
}
