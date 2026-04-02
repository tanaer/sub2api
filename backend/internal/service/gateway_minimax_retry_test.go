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

// TestGatewayForward_MiniMaxPreFilter_StripsIgnoredFieldsUpfront 验证前置过滤在首次请求时就剥除忽略参数。
func TestGatewayForward_MiniMaxPreFilter_StripsIgnoredFieldsUpfront(t *testing.T) {
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
			newMiniMaxCompatSuccessResponse(),
		},
	}
	svc := newMiniMaxTestService(upstream, false)
	account := newMiniMaxTestAccount()
	c, _ := newAnthropicTestContext(body)

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	// 首次请求就不应包含忽略的参数（前置过滤已剥除）
	require.Len(t, upstream.requestBodies, 1)
	require.NotContains(t, string(upstream.requestBodies[0]), `"context_management"`)
	require.NotContains(t, string(upstream.requestBodies[0]), `"top_k"`)
}

// TestGatewayForward_MiniMaxPreFilter_PreservesThinking 验证 thinking 参数不被剥除。
func TestGatewayForward_MiniMaxPreFilter_PreservesThinking(t *testing.T) {
	body := []byte(`{
		"model":"glm-5.1",
		"thinking":{"type":"enabled","budget_tokens":8000},
		"max_tokens":16000,
		"messages":[{"role":"user","content":"hello"}]
	}`)
	parsed, err := ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newMiniMaxCompatSuccessResponse(),
		},
	}
	svc := newMiniMaxTestService(upstream, false)
	account := newMiniMaxTestAccount()
	c, _ := newAnthropicTestContext(body)

	result, err := svc.Forward(context.Background(), c, account, parsed)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requestBodies, 1)
	// thinking 应保留
	require.Contains(t, string(upstream.requestBodies[0]), `"thinking"`)
	// max_tokens 不应被 clamp 到 32768
	require.Contains(t, string(upstream.requestBodies[0]), `"max_tokens"`)
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
