package handler

import (
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestResolveChatCompletionsSelectionErrorMessage_AnthropicGroupReturnsExplicitHint(t *testing.T) {
	msg := resolveChatCompletionsSelectionErrorMessage(
		&service.APIKey{
			Group: &service.Group{Platform: service.PlatformAnthropic},
		},
		errors.New("no available OpenAI accounts"),
	)

	require.Contains(t, msg, "/v1/chat/completions")
	require.Contains(t, msg, "/v1/messages")
}

func TestResolveChatCompletionsSelectionErrorMessage_DefaultsToGenericMessage(t *testing.T) {
	msg := resolveChatCompletionsSelectionErrorMessage(
		&service.APIKey{
			Group: &service.Group{Platform: service.PlatformOpenAI},
		},
		errors.New("query accounts failed"),
	)

	require.Equal(t, "Service temporarily unavailable", msg)
}
