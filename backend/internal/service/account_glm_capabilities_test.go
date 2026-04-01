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
	require.False(t, caps.ToolHistory)
	require.False(t, caps.ContextManagement)
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
				"tool_history":       true,
				"context_management": true,
			},
		},
	}

	caps := account.ResolveGLMCapabilities()
	require.True(t, caps.ToolHistory)
	require.True(t, caps.ContextManagement)
	require.True(t, account.SupportsGLMRequestTraits(GLMRequestTraits{
		HasToolResult:  true,
		HasContextMgmt: true,
	}))
}

func TestAccountSupportsGLMRequestTraits_MinimaxRejectsToolHistoryByDefault(t *testing.T) {
	account := &Account{
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}

	require.False(t, account.SupportsGLMRequestTraits(GLMRequestTraits{HasToolResult: true}))
	require.True(t, account.SupportsGLMRequestTraits(GLMRequestTraits{HasTools: true}))
}
