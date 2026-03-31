package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestInjectAnthropicModelIdentityInstruction(t *testing.T) {
	t.Run("nil system injects identity block", func(t *testing.T) {
		body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"你是谁？"}]}`)

		result := injectAnthropicModelIdentityInstruction(body, "glm-5.1", []any{
			map[string]any{
				"role":    "user",
				"content": "你是谁？",
			},
		})

		var parsed map[string]any
		err := json.Unmarshal(result, &parsed)
		require.NoError(t, err)

		system, ok := parsed["system"].([]any)
		require.True(t, ok, "system should be an array")
		require.Len(t, system, 1)

		first, ok := system[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "text", first["type"])
		require.Contains(t, first["text"], "我是一个由智谱训练的glm大语言模型")
	})

	t.Run("string system appends identity block", func(t *testing.T) {
		body := []byte(`{"model":"glm-5.1","system":"你是一个代码助手","messages":[{"role":"user","content":"你是谁？"}]}`)

		result := injectAnthropicModelIdentityInstruction(body, "glm-5.1", []any{
			map[string]any{
				"role":    "user",
				"content": "你是谁？",
			},
		})

		var parsed map[string]any
		err := json.Unmarshal(result, &parsed)
		require.NoError(t, err)

		system, ok := parsed["system"].([]any)
		require.True(t, ok, "system should be an array")
		require.Len(t, system, 2)

		first, ok := system[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "你是一个代码助手", first["text"])

		second, ok := system[1].(map[string]any)
		require.True(t, ok)
		require.Contains(t, second["text"], "我是一个由智谱训练的glm大语言模型")
	})

	t.Run("non identity question keeps body unchanged", func(t *testing.T) {
		body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"帮我写个接口"}]}`)

		result := injectAnthropicModelIdentityInstruction(body, "glm-5.1", []any{
			map[string]any{
				"role":    "user",
				"content": "帮我写个接口",
			},
		})

		require.JSONEq(t, string(body), string(result))
	})

	t.Run("developer question also injects identity block", func(t *testing.T) {
		body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"你的开发者是谁？背后的公司是哪家？"}]}`)

		result := injectAnthropicModelIdentityInstruction(body, "glm-5.1", []any{
			map[string]any{
				"role":    "user",
				"content": "你的开发者是谁？背后的公司是哪家？",
			},
		})

		var parsed map[string]any
		err := json.Unmarshal(result, &parsed)
		require.NoError(t, err)

		system, ok := parsed["system"].([]any)
		require.True(t, ok, "system should be an array")
		require.Len(t, system, 1)

		first, ok := system[0].(map[string]any)
		require.True(t, ok)
		require.Contains(t, first["text"], "我是一个由智谱训练的glm大语言模型")
	})
}

func TestInjectAnthropicModelIdentityInstruction_PreservesFieldOrder(t *testing.T) {
	body := []byte(`{"alpha":1,"system":[{"id":"block-1","type":"text","text":"Custom"}],"messages":[{"role":"user","content":"你是谁？"}],"omega":2}`)

	result := injectAnthropicModelIdentityInstruction(body, "glm-5.1", []any{
		map[string]any{
			"role":    "user",
			"content": "你是谁？",
		},
	})
	resultStr := string(result)

	assertJSONTokenOrder(t, resultStr, `"alpha"`, `"system"`, `"messages"`, `"omega"`)
	require.Contains(t, resultStr, `{"id":"block-1","type":"text","text":"Custom"}`)
	require.Contains(t, resultStr, `我是一个由智谱训练的glm大语言模型`)
}

func TestGatewayForwardInjectsAnthropicModelIdentityInstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"glm-5.1","messages":[{"role":"user","content":"你是谁？"}]}`)
	parsed, err := ParseGatewayRequest(body, domain.PlatformAnthropic)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: newOpenAIIdentityBadRequestResponse()}
	svc := &GatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          1,
		Name:        "anthropic-apikey",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err = svc.Forward(context.Background(), c, account, parsed)
	require.Error(t, err)
	require.Contains(t, gjson.GetBytes(upstream.lastBody, "system.0.text").String(), "我是一个由智谱训练的glm大语言模型")
}
