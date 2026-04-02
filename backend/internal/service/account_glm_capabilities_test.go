package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountResolveGLMCapabilities_DefaultMiniMaxPolicy(t *testing.T) {
	account := &Account{
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}

	caps := account.ResolveGLMCapabilities()
	// MiniMax Anthropic API 完全支持 tool_use / tool_result，默认启用
	require.True(t, caps.ToolHistory)
	require.True(t, caps.ContextManagement)
}

func TestAccountResolveGLMCapabilities_ExtraOverridesMiniMaxDefaults(t *testing.T) {
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://api.minimaxi.com/anthropic",
		},
		Extra: map[string]any{
			"glm_capabilities": map[string]any{
				"tool_history":       false,
				"context_management": false,
			},
		},
	}

	caps := account.ResolveGLMCapabilities()
	// Extra 可以覆盖默认值
	require.False(t, caps.ToolHistory)
	require.False(t, caps.ContextManagement)
}

func TestAccountSupportsGLMRequestTraits_MinimaxAcceptsToolHistoryByDefault(t *testing.T) {
	account := &Account{
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}

	require.True(t, account.SupportsGLMRequestTraits(GLMRequestTraits{HasToolResult: true}))
	require.True(t, account.SupportsGLMRequestTraits(GLMRequestTraits{HasTools: true}))
	require.True(t, account.SupportsGLMRequestTraits(GLMRequestTraits{HasContextMgmt: true}))
}

func TestAccountShouldStripAnthropicExtensions_MiniMaxReturnsFalse(t *testing.T) {
	account := &Account{
		Platform:         PlatformAnthropic,
		Type:             AccountTypeAPIKey,
		UpstreamProvider: "minimax",
		Credentials:      map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}
	require.False(t, account.ShouldStripAnthropicExtensions())
}

func TestAccountMaxTokensCap_MiniMaxNoLimit(t *testing.T) {
	account := &Account{
		Platform:         PlatformAnthropic,
		Type:             AccountTypeAPIKey,
		UpstreamProvider: "minimax",
		Credentials:      map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}
	require.Equal(t, 0, account.MaxTokensCap())
}
