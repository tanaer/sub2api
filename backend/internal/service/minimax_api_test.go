package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiniMaxBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"standard", "https://api.minimaxi.com/anthropic", "https://api.minimaxi.com"},
		{"trailing slash", "https://api.minimaxi.com/anthropic/", "https://api.minimaxi.com"},
		{"no anthropic", "https://api.minimaxi.com", "https://api.minimaxi.com"},
		{"empty", "", "https://api.minimaxi.com"},
		{"custom", "https://custom.host/anthropic", "https://custom.host"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				Credentials: map[string]any{"base_url": tt.baseURL},
			}
			require.Equal(t, tt.expected, miniMaxBaseURL(account))
		})
	}
}

func TestBuildImageDataURL(t *testing.T) {
	t.Run("base64 source", func(t *testing.T) {
		source := map[string]any{
			"type":       "base64",
			"media_type": "image/png",
			"data":       "abc123",
		}
		result := buildImageDataURL(source)
		require.Equal(t, "data:image/png;base64,abc123", result)
	})

	t.Run("url source", func(t *testing.T) {
		source := map[string]any{
			"type": "url",
			"url":  "https://example.com/img.png",
		}
		result := buildImageDataURL(source)
		require.Equal(t, "https://example.com/img.png", result)
	})

	t.Run("missing data returns empty", func(t *testing.T) {
		source := map[string]any{
			"type":       "base64",
			"media_type": "image/png",
		}
		result := buildImageDataURL(source)
		require.Equal(t, "", result)
	})

	t.Run("unknown type returns empty", func(t *testing.T) {
		source := map[string]any{"type": "unknown"}
		require.Equal(t, "", buildImageDataURL(source))
	})
}

func TestBuildImagePromptFromContext(t *testing.T) {
	t.Run("extracts text from same message", func(t *testing.T) {
		messages := []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "What is in this image?"},
					map[string]any{"type": "image", "source": map[string]any{}},
				},
			},
		}
		prompt := buildImagePromptFromContext(messages, 0)
		require.Equal(t, "What is in this image?", prompt)
	})

	t.Run("default prompt when no text", func(t *testing.T) {
		messages := []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "image", "source": map[string]any{}},
				},
			},
		}
		prompt := buildImagePromptFromContext(messages, 0)
		require.Equal(t, "Describe this image in detail.", prompt)
	})
}

func TestContainsImageBlock(t *testing.T) {
	require.True(t, containsImageBlock([]byte(`{"content":[{"type":"image"}]}`)))
	require.True(t, containsImageBlock([]byte(`{"content":[{"type": "image"}]}`)))
	require.False(t, containsImageBlock([]byte(`{"content":[{"type":"text"}]}`)))
}

func TestReplaceImageBlocksWithPlaceholder(t *testing.T) {
	input := []byte(`{
		"messages":[{"role":"user","content":[
			{"type":"text","text":"hello"},
			{"type":"image","source":{"type":"base64","media_type":"image/png","data":"abc"}}
		]}]
	}`)

	out := replaceImageBlocksWithPlaceholder(input)
	require.Contains(t, string(out), "not supported")
	require.NotContains(t, string(out), `"type":"image"`)
}
