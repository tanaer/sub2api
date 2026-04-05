package service

import (
	"context"
	"net/http"
)

// ProviderAdapter 定义上游供应商的适配行为。
// 协议层（Gateway Service）通过此接口与供应商层交互，
// 无需关心具体供应商的差异行为。
//
// 实现要求：
//   - 所有方法必须线程安全（adapter 实例在多个请求间共享）
//   - TransformRequest/TransformResponse 不应修改传入的 slice，应返回新 slice
type ProviderAdapter interface {
	// ID 返回供应商唯一标识（如 "native", "minimax", "zhipu", "kimi"）。
	ID() string

	// DisplayName 返回供应商显示名称（如 "MiniMax", "智谱 (GLM)"）。
	DisplayName() string

	// TransformRequest 在发送到上游前变换请求体。
	// 包括：参数裁剪（max_tokens）、字段剥离（thinking/cache_control）、工具注入等。
	TransformRequest(ctx context.Context, req *ProviderRequest) (*ProviderRequest, error)

	// TransformResponse 变换上游返回的响应。
	// 大多数供应商直接透传，部分需要格式转换。
	TransformResponse(ctx context.Context, resp *ProviderResponse) (*ProviderResponse, error)

	// ClassifyError 对上游错误进行分类，决定重试/failover/透传策略。
	ClassifyError(statusCode int, body []byte) ErrorClassification

	// AuthHeaders 构建发往上游的认证头。
	AuthHeaders(account *Account) map[string]string

	// ProviderConfig 返回供应商的运行时配置。
	ProviderConfig() *ProviderConfig
}

// ProviderRequest 封装发往上游的请求上下文。
type ProviderRequest struct {
	Body         []byte
	Account      *Account
	RequestModel string
	Parsed       *ParsedRequest // 已有的解析结构，复用
	Protocol     string         // "anthropic" | "openai" | "gemini"
}

// ProviderResponse 封装上游的响应。
type ProviderResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
	IsStream   bool
}

// ErrorClassification 上游错误分类结果。
type ErrorClassification struct {
	Kind                   ErrorKind
	RetryableOnSameAccount bool
	ShouldFailover         bool
	Message                string
}

// ErrorKind 错误类别。
type ErrorKind string

const (
	ErrorKindTransient  ErrorKind = "transient"   // 暂时性错误，可重试
	ErrorKindRateLimit  ErrorKind = "rate_limit"  // 速率限制
	ErrorKindAuth       ErrorKind = "auth"        // 认证失败
	ErrorKindBilling    ErrorKind = "billing"     // 余额不足
	ErrorKindBadRequest ErrorKind = "bad_request" // 客户端错误，不应重试
	ErrorKindServer     ErrorKind = "server"      // 上游服务端错误
	ErrorKindUnknown    ErrorKind = "unknown"
)

// ProviderConfig 供应商运行时配置（从 DB 或代码默认值加载）。
type ProviderConfig struct {
	ProviderID     string            `json:"provider_id"`
	DisplayName    string            `json:"display_name"`
	Enabled        bool              `json:"enabled"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	MaxTokensLimit int               `json:"max_tokens_limit,omitempty"`
	FailoverCodes  []int             `json:"failover_codes,omitempty"`
	ModelMapping   map[string]string `json:"model_mapping,omitempty"`
	ErrorPatterns  ErrorPatterns     `json:"error_patterns,omitempty"`
	Features       ProviderFeatures  `json:"features,omitempty"`
}

// ErrorPatterns 供应商特有的错误模式匹配规则。
// 每个字段包含一组子串，当上游错误响应体中包含任一子串时，匹配对应类别。
type ErrorPatterns struct {
	Billing   []string `json:"billing,omitempty"`
	RateLimit []string `json:"rate_limit,omitempty"`
	Auth      []string `json:"auth,omitempty"`
}

// ProviderFeatures 供应商特性开关。
type ProviderFeatures struct {
	WebSearchInjection   bool `json:"web_search_injection,omitempty"`
	ImageUnderstanding   bool `json:"image_understanding,omitempty"`
	SupportsThinking     bool `json:"supports_thinking,omitempty"`
	SupportsCacheControl bool `json:"supports_cache_control,omitempty"`
	ToolLoop             bool `json:"tool_loop,omitempty"`
}
