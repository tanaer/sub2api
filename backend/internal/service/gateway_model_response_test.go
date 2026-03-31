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
