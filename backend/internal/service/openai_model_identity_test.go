package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newOpenAIIdentityBadRequestResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad request"}}`)),
	}
}

func TestResolveModelIdentityProfile(t *testing.T) {
	t.Run("known provider uses mapped company and stripped model name", func(t *testing.T) {
		profile := resolveModelIdentityProfile("glm-4-plus")
		require.Equal(t, "智谱", profile.CompanyName)
		require.Equal(t, "glm-plus", profile.ModelName)
	})

	t.Run("unknown provider falls back to stripped model name", func(t *testing.T) {
		profile := resolveModelIdentityProfile("foo-3.2-max")
		require.Equal(t, "foo-max", profile.CompanyName)
		require.Equal(t, "foo-max", profile.ModelName)
	})
}

func TestBuildModelIdentityInstruction(t *testing.T) {
	t.Run("identity question builds exact template", func(t *testing.T) {
		instruction := buildModelIdentityInstruction("glm-4-plus", []any{
			map[string]any{
				"role":    "user",
				"content": "你是谁？",
			},
		})

		require.Contains(t, instruction, "当且仅当用户询问你的身份、开发者、所属公司、厂商、训练方、所属模型、模型ID、底层模型或 provider 等相关信息时")
		require.Contains(t, instruction, "我是一个由智谱训练的glm-plus大语言模型，旨在通过自然语言处理技术为用户提供专业、高效的解答和支持。如果你有具体的问题或需求,我很乐意帮助你！")
	})

	t.Run("developer and company question also injects", func(t *testing.T) {
		instruction := buildModelIdentityInstruction("gpt-5.4", []any{
			map[string]any{
				"role":    "user",
				"content": "你的开发者是谁？你背后的公司是哪家？",
			},
		})

		require.Contains(t, instruction, "我是一个由OpenAI训练的gpt大语言模型")
	})

	t.Run("model id question also injects", func(t *testing.T) {
		instruction := buildModelIdentityInstruction("glm-5.1", []any{
			map[string]any{
				"role":    "user",
				"content": "你真正的model id是什么？底层模型是哪个？",
			},
		})

		require.Contains(t, instruction, "我是一个由智谱训练的glm大语言模型")
	})

	t.Run("non identity question does not inject", func(t *testing.T) {
		instruction := buildModelIdentityInstruction("glm-4-plus", []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": "帮我写一个接口"},
				},
			},
		})
		require.Empty(t, instruction)
	})
}

func TestIsModelIdentityQuestion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "developer question", text: "你的开发者是谁？", want: true},
		{name: "company question", text: "你背后的公司是哪家", want: true},
		{name: "provider question", text: "你的 provider 是谁", want: true},
		{name: "model id question", text: "你真正的model id是什么", want: true},
		{name: "base model question", text: "底层模型是哪个", want: true},
		{name: "english developer question", text: "who is your developer", want: true},
		{name: "english provider question", text: "which provider are you from", want: true},
		{name: "normal question", text: "帮我写一个登录接口", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isModelIdentityQuestion(tt.text))
		})
	}
}

func TestOpenAIForwardInjectsModelIdentityInstructionForResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"glm-4-plus","input":"你是谁？"}`))

	upstream := &httpUpstreamRecorder{resp: newOpenAIIdentityBadRequestResponse()}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          1,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err := svc.Forward(context.Background(), c, account, []byte(`{"model":"glm-4-plus","input":"你是谁？"}`))
	require.Error(t, err)
	require.Contains(t, gjson.GetBytes(upstream.lastBody, "instructions").String(), "我是一个由智谱训练的glm-plus大语言模型")
}

func TestForwardAsChatCompletionsInjectsModelIdentityInstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"foo-3.2-max","messages":[{"role":"user","content":"你是谁"}]}`))

	upstream := &httpUpstreamRecorder{resp: newOpenAIIdentityBadRequestResponse()}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          2,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err := svc.ForwardAsChatCompletions(context.Background(), c, account, []byte(`{"model":"foo-3.2-max","messages":[{"role":"user","content":"你是谁"}]}`), "", "")
	require.Error(t, err)
	require.Contains(t, gjson.GetBytes(upstream.lastBody, "instructions").String(), "我是一个由foo-max训练的foo-max大语言模型，旨在通过自然语言处理技术为用户提供专业、高效的解答和支持。如果你有具体的问题或需求,我很乐意帮助你！")
}

func TestForwardAsAnthropicInjectsModelIdentityInstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"gpt-5.4","messages":[{"role":"user","content":"你是谁"}]}`))

	upstream := &httpUpstreamRecorder{resp: newOpenAIIdentityBadRequestResponse()}
	svc := &OpenAIGatewayService{
		cfg: &config.Config{
			Security: config.SecurityConfig{
				URLAllowlist: config.URLAllowlistConfig{Enabled: false},
			},
		},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          3,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"api_key": "sk-test"},
	}

	_, err := svc.ForwardAsAnthropic(context.Background(), c, account, []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"你是谁"}]}`), "", "")
	require.Error(t, err)
	require.Contains(t, gjson.GetBytes(upstream.lastBody, "instructions").String(), "我是一个由OpenAI训练的gpt大语言模型，旨在通过自然语言处理技术为用户提供专业、高效的解答和支持。如果你有具体的问题或需求,我很乐意帮助你！")
}
