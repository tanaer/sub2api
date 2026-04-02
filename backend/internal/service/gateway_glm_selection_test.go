//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSelectAccountForModelWithPlatform_MiniMaxAcceptsGLMToolHistory(t *testing.T) {
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

	// MiniMax now supports tool_history, so it should be selected
	ctx := WithGLMRequestTraits(context.Background(), GLMRequestTraits{HasToolResult: true})
	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "glm-5.1", map[int64]struct{}{}, PlatformAnthropic)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(8), account.ID)
}

func TestSelectAccountForModelWithPlatform_MiniMaxDisabledViaExtraRejectsToolHistory(t *testing.T) {
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
		Extra: map[string]any{
			"glm_capabilities": map[string]any{
				"tool_history": false,
			},
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

	// When tool_history is explicitly disabled via Extra, MiniMax should be skipped
	ctx := WithGLMRequestTraits(context.Background(), GLMRequestTraits{HasToolResult: true})
	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "glm-5.1", map[int64]struct{}{}, PlatformAnthropic)
	require.Nil(t, account)
	require.ErrorIs(t, err, ErrNoAvailableAccounts)
}
