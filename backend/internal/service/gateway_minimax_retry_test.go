package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newMiniMaxCompatSuccessResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"msg_1","type":"message","role":"assistant","model":"MiniMax-M2.7-highspeed","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":12,"output_tokens":7}}`,
		)),
	}
}

func newMiniMaxInvalidParamsResponse(message string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"type":"error","error":{"message":"` + message + `","type":"invalid_request_error"}}`,
		)),
	}
}

func newMiniMaxTestService(upstream *queuedHTTPUpstreamStub, failoverOn400 bool) *GatewayService {
	return &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
			Gateway: config.GatewayConfig{
				FailoverOn400: failoverOn400,
			},
		},
		httpUpstream: upstream,
	}
}

func newMiniMaxTestAccount() *Account {
	return &Account{
		ID:       8,
		Name:     "minimax",
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://api.minimaxi.com/anthropic",
		},
	}
}

func newAnthropicTestContext(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func TestGatewayForward_MiniMaxInvalidParamsRetry_StripsIgnoredFields(t *testing.T) {
	body := []byte(`{
		"model":"glm-5.1",
		"top_k":10,
		"context_management":{"edits":[{"type":"clear_thinking_20251015"}]},
		"messages":[{"role":"user","content":"hello"}]
	}`)
	parsed, err := ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newMiniMaxInvalidParamsResponse("invalid params"),
			newMiniMaxCompatSuccessResponse(),
		},
	}
	svc := newMiniMaxTestService(upstream, false)
	account := newMiniMaxTestAccount()
	c, _ := newAnthropicTestContext(body)

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 2)
	require.Contains(t, string(upstream.requestBodies[0]), `"context_management"`)
	require.Contains(t, string(upstream.requestBodies[0]), `"top_k"`)
	require.NotContains(t, string(upstream.requestBodies[1]), `"context_management"`)
	require.NotContains(t, string(upstream.requestBodies[1]), `"top_k"`)
}

func TestGatewayForward_MiniMaxInvalidParamsRetry_DowngradesToolHistory(t *testing.T) {
	body := []byte(`{
		"model":"glm-5.1",
		"messages":[
			{"role":"assistant","content":[{"type":"tool_use","id":"toolu_1","name":"weather","input":{"city":"shanghai"}}]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"sunny"}]}
		]
	}`)
	parsed, err := ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newMiniMaxInvalidParamsResponse("invalid params, tool result's tool id(call_abc) not found (2013)"),
			newMiniMaxCompatSuccessResponse(),
		},
	}
	svc := newMiniMaxTestService(upstream, false)
	account := newMiniMaxTestAccount()
	c, _ := newAnthropicTestContext(body)

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 2)
	require.Contains(t, string(upstream.requestBodies[0]), `"type":"tool_result"`)
	require.NotContains(t, string(upstream.requestBodies[1]), `"type":"tool_result"`)
	require.Contains(t, string(upstream.requestBodies[1]), `(tool_result)`)
}

func TestGatewayForward_MiniMaxPersistentInvalidParamsFailsOverOn400(t *testing.T) {
	body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"hello"}]}`)
	parsed, err := ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newMiniMaxInvalidParamsResponse("invalid params"),
		},
	}
	svc := newMiniMaxTestService(upstream, true)
	account := newMiniMaxTestAccount()
	c, _ := newAnthropicTestContext(body)

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
}
