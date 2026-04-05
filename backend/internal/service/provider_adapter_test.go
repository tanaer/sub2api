//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- NativeAdapter tests ----

func TestNativeAdapter_TransformRequest_Passthrough(t *testing.T) {
	adapter := NewNativeAdapter()
	req := &ProviderRequest{
		Body:         []byte(`{"model":"claude-sonnet-4-20250514","max_tokens":64000}`),
		RequestModel: "claude-sonnet-4-20250514",
		Protocol:     "anthropic",
	}

	out, err := adapter.TransformRequest(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, req.Body, out.Body, "native adapter should not modify body")
}

func TestNativeAdapter_TransformResponse_Passthrough(t *testing.T) {
	adapter := NewNativeAdapter()
	resp := &ProviderResponse{StatusCode: 200, Body: []byte(`{"content":"hello"}`)}

	out, err := adapter.TransformResponse(context.Background(), resp)
	require.NoError(t, err)
	assert.Equal(t, resp, out)
}

func TestNativeAdapter_ClassifyError(t *testing.T) {
	adapter := NewNativeAdapter()

	tests := []struct {
		status   int
		wantKind ErrorKind
	}{
		{429, ErrorKindRateLimit},
		{401, ErrorKindAuth},
		{403, ErrorKindAuth},
		{500, ErrorKindServer},
		{502, ErrorKindServer},
		{200, ErrorKindUnknown},
	}
	for _, tt := range tests {
		c := adapter.ClassifyError(tt.status, nil)
		assert.Equal(t, tt.wantKind, c.Kind, "status %d", tt.status)
	}
}

func TestNativeAdapter_AuthHeaders_Nil(t *testing.T) {
	adapter := NewNativeAdapter()
	assert.Nil(t, adapter.AuthHeaders(nil))
	assert.Nil(t, adapter.AuthHeaders(&Account{}))
}

func TestNativeAdapter_Config(t *testing.T) {
	adapter := NewNativeAdapter()
	cfg := adapter.ProviderConfig()
	assert.Equal(t, "native", cfg.ProviderID)
	assert.True(t, cfg.Features.SupportsThinking)
	assert.True(t, cfg.Features.SupportsCacheControl)
}

// ---- GenericAdapter tests ----

func TestGenericAdapter_TransformRequest_ClampMaxTokens(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{
		ProviderID:     "zhipu",
		DisplayName:    "智谱",
		MaxTokensLimit: 4096,
	})

	req := &ProviderRequest{
		Body: []byte(`{"max_tokens":64000,"messages":[]}`),
	}

	out, err := adapter.TransformRequest(context.Background(), req)
	require.NoError(t, err)

	// max_tokens should be clamped
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(out.Body, &parsed))
	maxTokens, _ := parsed["max_tokens"].(float64)
	assert.Equal(t, float64(4096), maxTokens)
}

func TestGenericAdapter_TransformRequest_NilPassthrough(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{ProviderID: "test"})
	out, err := adapter.TransformRequest(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestGenericAdapter_TransformResponse_Passthrough(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{ProviderID: "test"})
	resp := &ProviderResponse{StatusCode: 200, Body: []byte("ok")}

	out, err := adapter.TransformResponse(context.Background(), resp)
	require.NoError(t, err)
	assert.Equal(t, resp, out)
}

func TestGenericAdapter_ClassifyError_Patterns(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{
		ProviderID: "zhipu",
		ErrorPatterns: ErrorPatterns{
			Billing:   []string{"INSUFFICIENT_BALANCE"},
			RateLimit: []string{"rate limit exceeded"},
			Auth:      []string{"invalid_api_key"},
		},
		FailoverCodes: []int{400},
	})

	tests := []struct {
		name     string
		status   int
		body     string
		wantKind ErrorKind
	}{
		{"billing pattern", 400, `{"error":"INSUFFICIENT_BALANCE"}`, ErrorKindBilling},
		{"rate limit pattern", 400, `{"error":"rate limit exceeded"}`, ErrorKindRateLimit},
		{"auth pattern", 401, `{"error":"invalid_api_key"}`, ErrorKindAuth},
		{"429 status", 429, `{}`, ErrorKindRateLimit},
		{"500 status", 500, `{}`, ErrorKindServer},
		{"400 with failover", 400, `{"error":"something else"}`, ErrorKindBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := adapter.ClassifyError(tt.status, []byte(tt.body))
			assert.Equal(t, tt.wantKind, c.Kind)
		})
	}
}

func TestGenericAdapter_ClassifyError_FailoverCodes(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{
		ProviderID:    "test",
		FailoverCodes: []int{400, 403},
	})

	c400 := adapter.ClassifyError(400, []byte(`{}`))
	assert.True(t, c400.ShouldFailover, "400 is in failover codes")

	c403 := adapter.ClassifyError(403, []byte(`{}`))
	assert.True(t, c403.ShouldFailover, "403 is in failover codes")

	c404 := adapter.ClassifyError(404, []byte(`{}`))
	assert.False(t, c404.ShouldFailover, "404 is NOT in failover codes")
}

func TestGenericAdapter_AuthHeaders(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{ProviderID: "test"})

	account := &Account{
		Credentials: map[string]any{"api_key": "sk-test-123"},
	}
	headers := adapter.AuthHeaders(account)
	assert.Equal(t, "Bearer sk-test-123", headers["Authorization"])

	// nil account
	assert.Nil(t, adapter.AuthHeaders(nil))

	// no api_key
	assert.Nil(t, adapter.AuthHeaders(&Account{Credentials: map[string]any{}}))
}

func TestGenericAdapter_Config(t *testing.T) {
	cfg := &ProviderConfig{
		ProviderID:     "zhipu",
		DisplayName:    "智谱",
		MaxTokensLimit: 32768,
	}
	adapter := NewGenericAdapter(cfg)
	assert.Equal(t, cfg, adapter.ProviderConfig())
}

// ---- ProviderConfig JSON tests ----

func TestProviderConfig_JSON_RoundTrip(t *testing.T) {
	cfg := &ProviderConfig{
		ProviderID:     "zhipu",
		DisplayName:    "智谱 (GLM)",
		Enabled:        true,
		TimeoutSeconds: 60,
		MaxTokensLimit: 32768,
		FailoverCodes:  []int{400, 401, 429},
		ModelMapping:   map[string]string{"claude-sonnet-4-20250514": "glm-4.7"},
		ErrorPatterns: ErrorPatterns{
			Billing:   []string{"INSUFFICIENT_BALANCE"},
			RateLimit: []string{"rate limit exceeded"},
		},
		Features: ProviderFeatures{
			WebSearchInjection: true,
			ImageUnderstanding: true,
		},
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var restored ProviderConfig
	require.NoError(t, json.Unmarshal(data, &restored))

	assert.Equal(t, cfg.ProviderID, restored.ProviderID)
	assert.Equal(t, cfg.DisplayName, restored.DisplayName)
	assert.Equal(t, cfg.MaxTokensLimit, restored.MaxTokensLimit)
	assert.Equal(t, cfg.FailoverCodes, restored.FailoverCodes)
	assert.Equal(t, cfg.ModelMapping, restored.ModelMapping)
	assert.Equal(t, cfg.ErrorPatterns, restored.ErrorPatterns)
	assert.Equal(t, cfg.Features, restored.Features)
}

// ---- matchesAnyPattern tests ----

func TestMatchesAnyPattern(t *testing.T) {
	assert.True(t, matchesAnyPattern("INSUFFICIENT_BALANCE: no credits", []string{"INSUFFICIENT_BALANCE"}))
	assert.True(t, matchesAnyPattern("insufficient_balance", []string{"INSUFFICIENT_BALANCE"})) // case insensitive
	assert.False(t, matchesAnyPattern("all good", []string{"INSUFFICIENT_BALANCE"}))
	assert.False(t, matchesAnyPattern("anything", nil))
	assert.False(t, matchesAnyPattern("anything", []string{}))
	assert.False(t, matchesAnyPattern("anything", []string{""}))
}

// ---- isFailoverCode tests ----

func TestIsFailoverCode(t *testing.T) {
	assert.True(t, isFailoverCode(400, []int{400, 401}))
	assert.True(t, isFailoverCode(401, []int{400, 401}))
	assert.False(t, isFailoverCode(500, []int{400, 401}))
	assert.False(t, isFailoverCode(400, nil))
}

// ---- GenericAdapter ClassifyError: auth patterns take priority ----

func TestGenericAdapter_ClassifyError_PatternPriority(t *testing.T) {
	adapter := NewGenericAdapter(&ProviderConfig{
		ProviderID: "test",
		ErrorPatterns: ErrorPatterns{
			Auth:    []string{"auth_failed"},
			Billing: []string{"auth_failed_billing"},
		},
	})

	// "auth_failed" matches Auth first (checked before Billing)
	c := adapter.ClassifyError(http.StatusBadRequest, []byte(`auth_failed`))
	assert.Equal(t, ErrorKindAuth, c.Kind)
}
