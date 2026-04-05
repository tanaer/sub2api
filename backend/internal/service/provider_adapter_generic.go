package service

import (
	"context"
	"net/http"
	"strings"
)

// GenericAdapter 是配置驱动的通用供应商适配器。
// 所有行为由 ProviderConfig 控制，无需编写代码即可接入新供应商。
// 对于需要特殊逻辑的供应商（如 MiniMax 工具循环），应嵌入此结构并覆盖特定方法。
type GenericAdapter struct {
	config *ProviderConfig
}

// NewGenericAdapter 创建一个配置驱动的通用适配器。
func NewGenericAdapter(cfg *ProviderConfig) *GenericAdapter {
	return &GenericAdapter{config: cfg}
}

func (a *GenericAdapter) ID() string          { return a.config.ProviderID }
func (a *GenericAdapter) DisplayName() string { return a.config.DisplayName }

// TransformRequest 根据配置变换请求体：
//   - 非 native 上游：剥离 Anthropic 专有字段（thinking/cache_control）
//   - 有 max_tokens_limit 配置：裁剪 max_tokens
func (a *GenericAdapter) TransformRequest(_ context.Context, req *ProviderRequest) (*ProviderRequest, error) {
	if req == nil {
		return req, nil
	}

	body := req.Body

	// 剥离 Anthropic 专有字段（除非供应商声明支持）
	if !a.config.Features.SupportsThinking || !a.config.Features.SupportsCacheControl {
		body = StripAnthropicExtensionsForGLM(body)
	}

	// 裁剪 max_tokens
	if a.config.MaxTokensLimit > 0 {
		body = ClampMaxTokens(body, a.config.MaxTokensLimit)
	}

	out := *req
	out.Body = body
	return &out, nil
}

// TransformResponse 直接透传响应，不做变换。
func (a *GenericAdapter) TransformResponse(_ context.Context, resp *ProviderResponse) (*ProviderResponse, error) {
	return resp, nil
}

// ClassifyError 根据状态码和错误模式配置对错误进行分类。
func (a *GenericAdapter) ClassifyError(statusCode int, body []byte) ErrorClassification {
	bodyStr := string(body)

	// 按配置的错误模式匹配
	if matchesAnyPattern(bodyStr, a.config.ErrorPatterns.Auth) {
		return ErrorClassification{
			Kind:           ErrorKindAuth,
			ShouldFailover: true,
		}
	}
	if matchesAnyPattern(bodyStr, a.config.ErrorPatterns.Billing) {
		return ErrorClassification{
			Kind:           ErrorKindBilling,
			ShouldFailover: true,
		}
	}
	if matchesAnyPattern(bodyStr, a.config.ErrorPatterns.RateLimit) {
		return ErrorClassification{
			Kind:           ErrorKindRateLimit,
			ShouldFailover: true,
		}
	}

	// 按状态码分类
	switch {
	case statusCode == http.StatusTooManyRequests:
		return ErrorClassification{
			Kind:                   ErrorKindRateLimit,
			RetryableOnSameAccount: true,
			ShouldFailover:         true,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrorClassification{
			Kind:           ErrorKindAuth,
			ShouldFailover: true,
		}
	case statusCode >= 500:
		return ErrorClassification{
			Kind:                   ErrorKindServer,
			RetryableOnSameAccount: true,
			ShouldFailover:         true,
		}
	case statusCode >= 400:
		return ErrorClassification{
			Kind:           ErrorKindBadRequest,
			ShouldFailover: isFailoverCode(statusCode, a.config.FailoverCodes),
		}
	default:
		return ErrorClassification{Kind: ErrorKindUnknown}
	}
}

// AuthHeaders 从 account.Credentials 中读取 api_key，设置标准 Bearer 认证头。
func (a *GenericAdapter) AuthHeaders(account *Account) map[string]string {
	if account == nil {
		return nil
	}
	apiKey := account.GetCredential("api_key")
	if apiKey == "" {
		return nil
	}
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
	}
}

// ProviderConfig 返回当前配置。
func (a *GenericAdapter) ProviderConfig() *ProviderConfig {
	return a.config
}

// matchesAnyPattern 检查 text 是否包含 patterns 中的任一子串（大小写不敏感）。
func matchesAnyPattern(text string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	lower := strings.ToLower(text)
	for _, p := range patterns {
		if p != "" && strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// isFailoverCode 检查状态码是否在 failover 列表中。
func isFailoverCode(code int, codes []int) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
