package service

import (
	"context"
	"net/http"
)

// NativeAdapter 处理原生 Anthropic/OpenAI 账号（upstream_provider 为空或 "native"）。
// 所有方法基本透传，不做变换——原生账号使用原始 API，无需适配。
type NativeAdapter struct{}

// NewNativeAdapter 创建原生适配器。
func NewNativeAdapter() *NativeAdapter {
	return &NativeAdapter{}
}

func (a *NativeAdapter) ID() string          { return "native" }
func (a *NativeAdapter) DisplayName() string { return "Native (原生)" }

// TransformRequest 直接透传，不做变换。
func (a *NativeAdapter) TransformRequest(_ context.Context, req *ProviderRequest) (*ProviderRequest, error) {
	return req, nil
}

// TransformResponse 直接透传，不做变换。
func (a *NativeAdapter) TransformResponse(_ context.Context, resp *ProviderResponse) (*ProviderResponse, error) {
	return resp, nil
}

// ClassifyError 按标准 HTTP 状态码分类，不做供应商特定匹配。
func (a *NativeAdapter) ClassifyError(statusCode int, _ []byte) ErrorClassification {
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
	default:
		return ErrorClassification{Kind: ErrorKindUnknown}
	}
}

// AuthHeaders 原生账号不由 adapter 设置认证头——
// 认证由 Gateway Service 中的现有逻辑处理（x-api-key / Bearer / OAuth 等）。
func (a *NativeAdapter) AuthHeaders(_ *Account) map[string]string {
	return nil
}

// ProviderConfig 返回原生供应商配置（全部默认值）。
func (a *NativeAdapter) ProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		ProviderID:  "native",
		DisplayName: "Native (原生)",
		Enabled:     true,
		Features: ProviderFeatures{
			SupportsThinking:     true,
			SupportsCacheControl: true,
		},
	}
}
