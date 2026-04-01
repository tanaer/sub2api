//go:build unit

package service

import (
	"reflect"
	"testing"
)

func TestAccountResolveModelRetryChain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		credentials    map[string]any
		requestedModel string
		maxAttempts    int
		expected       []string
	}{
		{
			name: "mapped primary plus explicit fallback targets are deduplicated and capped",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-5": "gpt-5-high",
				},
				"model_fallbacks": map[string]any{
					"gpt-5": []any{"gpt-5-high", "gpt-4.1", "gpt-4o", "gpt-4o-mini"},
				},
			},
			requestedModel: "gpt-5",
			maxAttempts:    3,
			expected:       []string{"gpt-5-high", "gpt-4.1", "gpt-4o"},
		},
		{
			name: "exact fallback rule wins over wildcard rules",
			credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-*": "claude-sonnet-4-5",
				},
				"model_fallbacks": map[string]any{
					"claude-*":            []any{"claude-haiku-3-5"},
					"claude-sonnet-*":     []any{"claude-opus-4-1", "claude-haiku-3-5"},
					"claude-sonnet-4-5":   []any{"claude-opus-4-1"},
					"claude-sonnet-4-5-x": []any{"should-not-match"},
				},
			},
			requestedModel: "claude-sonnet-4-5",
			maxAttempts:    4,
			expected:       []string{"claude-sonnet-4-5", "claude-opus-4-1"},
		},
		{
			name: "requested model is kept when no mapping or fallback exists",
			credentials: map[string]any{
				"model_fallbacks": map[string]any{
					"claude-*": []any{"claude-haiku-3-5"},
				},
			},
			requestedModel: "gemini-2.5-pro",
			maxAttempts:    3,
			expected:       []string{"gemini-2.5-pro"},
		},
		{
			name: "known glm fallback typos are normalized before retrying",
			credentials: map[string]any{
				"model_fallbacks": map[string]any{
					"glm-5.1": []any{"glm4.7", "GLM-4.7-FlashX", "glm4.5", "GLM-Z1-Air"},
				},
			},
			requestedModel: "glm-5.1",
			maxAttempts:    5,
			expected:       []string{"glm-5.1", "glm-4.7", "GLM-4.7-FlashX", "glm-4.5", "GLM-Z1-Air"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			account := &Account{Credentials: tt.credentials}
			got := account.ResolveModelRetryChain(tt.requestedModel, tt.maxAttempts)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Fatalf("ResolveModelRetryChain(%q) = %#v, want %#v", tt.requestedModel, got, tt.expected)
			}
		})
	}
}

func TestShouldFallbackToNextModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       []byte
		expected   bool
	}{
		{
			name:       "429 always falls back",
			statusCode: 429,
			expected:   true,
		},
		{
			name:       "529 always falls back",
			statusCode: 529,
			expected:   true,
		},
		{
			name:       "500 falls back on overloaded keyword",
			statusCode: 500,
			body:       []byte(`{"error":{"message":"server overloaded, please retry"}}`),
			expected:   true,
		},
		{
			name:       "503 falls back on concurrency keyword",
			statusCode: 503,
			body:       []byte(`concurrency limit reached`),
			expected:   true,
		},
		{
			name:       "500 falls back on resource exhausted keyword",
			statusCode: 500,
			body:       []byte(`RESOURCE_EXHAUSTED`),
			expected:   true,
		},
		{
			name:       "400 never falls back even with keyword",
			statusCode: 400,
			body:       []byte(`rate limit exceeded`),
			expected:   false,
		},
		{
			name:       "401 never falls back",
			statusCode: 401,
			body:       []byte(`capacity exceeded`),
			expected:   false,
		},
		{
			name:       "403 never falls back",
			statusCode: 403,
			body:       []byte(`too many requests`),
			expected:   false,
		},
		{
			name:       "5xx without degrade signal does not fall back",
			statusCode: 503,
			body:       []byte(`internal server error`),
			expected:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := shouldFallbackToNextModel(tt.statusCode, tt.body)
			if got != tt.expected {
				t.Fatalf("shouldFallbackToNextModel(%d, %q) = %v, want %v", tt.statusCode, string(tt.body), got, tt.expected)
			}
		})
	}
}
