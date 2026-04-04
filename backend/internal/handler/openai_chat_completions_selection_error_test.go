package handler

import (
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestResolveChatCompletionsSelectionErrorMessage_AnthropicGroupExplainsRoutingInsteadOfProtocolRejection(t *testing.T) {
	msg := resolveChatCompletionsSelectionErrorMessage(
		&service.APIKey{
			Group: &service.Group{Platform: service.PlatformAnthropic},
		},
		errors.New("no available OpenAI accounts"),
	)

	require.Contains(t, msg, "Anthropic group")
	require.Contains(t, msg, "routing")
	require.NotContains(t, msg, "use /v1/messages instead")
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
