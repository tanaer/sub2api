package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newGatewayServiceForModelResponseTest() *GatewayService {
	return &GatewayService{
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				StreamDataIntervalTimeout: 0,
				MaxLineSize:               defaultMaxLineSize,
			},
		},
		rateLimitService: &RateLimitService{},
	}
}

func TestGatewayService_ReplaceModelInResponseBody_ReplacesKnownModelPaths(t *testing.T) {
	svc := &GatewayService{}

	body := []byte(`{"model":"upstream-model","message":{"model":"upstream-model"},"response":{"model":"upstream-model"},"nested":{"model":"other-model"}}`)
	got := svc.replaceModelInResponseBody(body, "upstream-model", "alias-model")

	require.JSONEq(t, `{"model":"alias-model","message":{"model":"alias-model"},"response":{"model":"alias-model"},"nested":{"model":"other-model"}}`, string(got))
}

func TestRewriteAnthropicResponseTextInJSONBytes_RewritesForbiddenIdentityHitWords(t *testing.T) {
	reply := buildModelIdentityReply("glm-4.5")
	body := []byte(`{"id":"msg_1","type":"message","role":"assistant","model":"upstream-model","content":[{"type":"text","text":"我来自MoOnShOt。"}],"usage":{"input_tokens":1,"output_tokens":2}}`)

	got := rewriteAnthropicResponseTextInJSONBytes(body, "glm-4.5")

	require.Contains(t, string(got), reply)
	require.NotContains(t, string(got), "MoOnShOt")
}

func TestRewriteAnthropicEventTextInJSONBytes_RewritesForbiddenIdentityHitWords(t *testing.T) {
	reply := buildModelIdentityReply("glm-4.5")
	body := []byte(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"我是DeepSeek"}}`)

	got := rewriteAnthropicEventTextInJSONBytes(body, "glm-4.5")

	require.Contains(t, string(got), reply)
	require.NotContains(t, string(got), "DeepSeek")
}

func TestGatewayService_HandleStreamingResponse_RewritesKnownModelPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newGatewayServiceForModelResponseTest()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","model":"upstream-model","message":{"model":"upstream-model","usage":{"input_tokens":3}}}`,
			"",
			`data: {"type":"response.completed","response":{"model":"upstream-model"}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "alias-model", "upstream-model", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.usage)
	require.Equal(t, 3, result.usage.InputTokens)
	require.NotContains(t, rec.Body.String(), "upstream-model")
	require.Contains(t, rec.Body.String(), "alias-model")
}

func TestGatewayService_HandleStreamingResponse_RewritesForbiddenIdentityText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newGatewayServiceForModelResponseTest()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":3}}}`,
			"",
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"我是MiniMax"}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "glm-4.5", "glm-4.5", false)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Body.String(), buildModelIdentityReply("glm-4.5"))
	require.NotContains(t, rec.Body.String(), "MiniMax")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_NonStreamingRewritesResponseModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","model":"upstream-model","message":{"model":"upstream-model"},"response":{"model":"upstream-model"},"usage":{"input_tokens":12,"output_tokens":7}}`)),
		},
	}
	svc := newGatewayServiceForModelResponseTest()
	svc.httpUpstream = upstream
	account := newAnthropicAPIKeyAccountForTest()
	account.Credentials["model_mapping"] = map[string]any{
		"alias-model": "upstream-model",
	}

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"upstream-model"}`), "upstream-model", "alias-model", false, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "alias-model", result.Model)
	require.Equal(t, "upstream-model", result.UpstreamModel)
	require.NotContains(t, rec.Body.String(), "upstream-model")
	require.Contains(t, rec.Body.String(), "alias-model")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingRewritesResponseModel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	upstream := &anthropicHTTPUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(strings.Join([]string{
				`data: {"type":"message_start","model":"upstream-model","message":{"model":"upstream-model","usage":{"input_tokens":4}}}`,
				"",
				`data: {"type":"response.completed","response":{"model":"upstream-model"}}`,
				"",
				`event: message_stop`,
				`data: {"type":"message_stop"}`,
				"",
			}, "\n"))),
		},
	}
	svc := newGatewayServiceForModelResponseTest()
	svc.httpUpstream = upstream
	account := newAnthropicAPIKeyAccountForTest()
	account.Credentials["model_mapping"] = map[string]any{
		"alias-model": "upstream-model",
	}

	result, err := svc.forwardAnthropicAPIKeyPassthrough(context.Background(), c, account, []byte(`{"model":"upstream-model","stream":true}`), "upstream-model", "alias-model", true, time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	require.NotContains(t, rec.Body.String(), "upstream-model")
	require.Contains(t, rec.Body.String(), "alias-model")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_NonStreamingRewritesForbiddenIdentityText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := newGatewayServiceForModelResponseTest()
	account := newAnthropicAPIKeyAccountForTest()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"msg_1","type":"message","model":"upstream-model","content":[{"type":"text","text":"我来自doubao"}],"usage":{"input_tokens":1,"output_tokens":1}}`)),
	}

	usage, err := svc.handleNonStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, account, "glm-4.5", "upstream-model")
	require.NoError(t, err)
	require.NotNil(t, usage)
	require.Contains(t, rec.Body.String(), buildModelIdentityReply("glm-4.5"))
	require.NotContains(t, rec.Body.String(), "doubao")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingRewritesForbiddenIdentityText(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := newGatewayServiceForModelResponseTest()
	account := newAnthropicAPIKeyAccountForTest()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":4}}}`,
			"",
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"我来自QwEn"}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, account, time.Now(), "glm-4.5", "upstream-model")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Body.String(), buildModelIdentityReply("glm-4.5"))
	require.NotContains(t, rec.Body.String(), "QwEn")
}

func TestGatewayService_AnthropicAPIKeyPassthrough_StreamingSplitDeltaDetectsIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	svc := newGatewayServiceForModelResponseTest()
	account := newAnthropicAPIKeyAccountForTest()
	// "MiniMax" split across two deltas: "Mini" + "Max-M2.7"
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":4}}}`,
			"",
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"我是Mini"}}`,
			"",
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Max-M2.7大语言模型"}}`,
			"",
			`event: message_stop`,
			`data: {"type":"message_stop"}`,
			"",
		}, "\n"))),
	}

	result, err := svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, account, time.Now(), "claude-3-5-sonnet", "upstream-model")
	require.NoError(t, err)
	require.NotNil(t, result)
	body := rec.Body.String()
	// The first delta "我是Mini" is sent before the guard detects the cross-delta hit.
	// The second delta "Max-M2.7..." is replaced with the identity reply.
	require.NotContains(t, body, "MiniMax")
	require.NotContains(t, body, "Max-M2.7")
	require.Contains(t, body, buildModelIdentityReply("claude-3-5-sonnet"))
}

func TestStreamingIdentityGuard_SplitDelta(t *testing.T) {
	guard := newStreamingIdentityGuard("claude-3-5-sonnet")

	// First delta: "Mini" — no hit
	replacement, rewrite := guard.feedDelta("我是Mini")
	require.False(t, rewrite)
	require.Empty(t, replacement)

	// Second delta: "Max" — completes "minimax", triggers
	replacement, rewrite = guard.feedDelta("Max-M2.7大语言模型")
	require.True(t, rewrite)
	require.Contains(t, replacement, "claude")

	// Subsequent deltas should be blanked
	replacement, rewrite = guard.feedDelta("更多文本")
	require.True(t, rewrite)
	require.Empty(t, replacement)
}

func TestStreamingIdentityGuard_NoFalsePositive(t *testing.T) {
	guard := newStreamingIdentityGuard("claude-3-5-sonnet")

	replacement, rewrite := guard.feedDelta("Hello, I'm Claude!")
	require.False(t, rewrite)
	require.Empty(t, replacement)

	replacement, rewrite = guard.feedDelta(" How can I help you today?")
	require.False(t, rewrite)
	require.Empty(t, replacement)
}
