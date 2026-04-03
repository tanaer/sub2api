package service

type SystemSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	FrontendURL                      string
	InvitationCodeEnabled            bool
	TotpEnabled                      bool // TOTP 双因素认证

	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPPasswordConfigured bool
	SMTPFrom               string
	SMTPFromName           string
	SMTPUseTLS             bool

	TurnstileEnabled             bool
	TurnstileSiteKey             string
	TurnstileSecretKey           string
	TurnstileSecretKeyConfigured bool

	// LinuxDo Connect OAuth 登录
	LinuxDoConnectEnabled                bool
	LinuxDoConnectClientID               string
	LinuxDoConnectClientSecret           string
	LinuxDoConnectClientSecretConfigured bool
	LinuxDoConnectRedirectURL            string

	SiteName                    string
	SiteLogo                    string
	SiteSubtitle                string
	APIBaseURL                  string
	ContactInfo                 string
	DocURL                      string
	UserAgreementContent        string
	HomeContent                 string
	SupportedAIModels           []string
	HideCcsImportButton         bool
	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	SoraClientEnabled           bool
	CustomMenuItems             string // JSON array of custom menu items
	CustomEndpoints             string // JSON array of custom endpoints

	DefaultConcurrency   int
	DefaultBalance       float64
	DefaultSubscriptions []DefaultSubscriptionSetting

	// Model fallback configuration
	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	// Identity patch configuration (Claude -> Gemini)
	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	// Ops monitoring (vNext)
	OpsMonitoringEnabled         bool
	OpsRealtimeMonitoringEnabled bool
	OpsQueryModeDefault          string
	OpsMetricsIntervalSeconds    int

	// Claude Code version check
	MinClaudeCodeVersion string
	MaxClaudeCodeVersion string

	// 分组隔离：允许未分组 Key 调度（默认 false → 403）
	AllowUngroupedKeyScheduling bool

	// Backend 模式：禁用用户注册和自助服务，仅管理员可登录
	BackendModeEnabled bool

	// Gateway forwarding behavior
	EnableFingerprintUnification bool // 是否统一 OAuth 账号的指纹头（默认 true）
	EnableMetadataPassthrough    bool // 是否透传客户端原始 metadata（默认 false）

	// Gateway failover status codes
	FailoverStatusCodes []int `json:"failover_status_codes"`
	FailoverInclude5xx  bool  `json:"failover_include_5xx"`

	// Health circuit breaker configuration
	HealthCircuitBreakerConfig *HealthCircuitBreakerConfig `json:"health_circuit_breaker_config"`
}

type DefaultSubscriptionSetting struct {
	GroupID      int64 `json:"group_id"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	InvitationCodeEnabled            bool
	TotpEnabled                      bool // TOTP 双因素认证
	TurnstileEnabled                 bool
	TurnstileSiteKey                 string
	SiteName                         string
	SiteLogo                         string
	SiteSubtitle                     string
	APIBaseURL                       string
	ContactInfo                      string
	DocURL                           string
	UserAgreementContent             string
	HomeContent                      string
	SupportedAIModels                []string
	HideCcsImportButton              bool

	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	SoraClientEnabled           bool
	CustomMenuItems             string // JSON array of custom menu items
	CustomEndpoints             string // JSON array of custom endpoints

	LinuxDoOAuthEnabled bool
	BackendModeEnabled  bool
	Version             string
}

// SoraS3Settings Sora S3 存储配置
type SoraS3Settings struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"secret_access_key"`            // 仅内部使用，不直接返回前端
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"` // 前端展示用
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
}

// SoraS3Profile Sora S3 多配置项（服务内部模型）
type SoraS3Profile struct {
	ProfileID                 string `json:"profile_id"`
	Name                      string `json:"name"`
	IsActive                  bool   `json:"is_active"`
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"-"`                            // 仅内部使用，不直接返回前端
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"` // 前端展示用
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
	UpdatedAt                 string `json:"updated_at"`
}

// SoraS3ProfileList Sora S3 多配置列表
type SoraS3ProfileList struct {
	ActiveProfileID string          `json:"active_profile_id"`
	Items           []SoraS3Profile `json:"items"`
}

// StreamTimeoutSettings 流超时处理配置（仅控制超时后的处理方式，超时判定由网关配置控制）
type StreamTimeoutSettings struct {
	// Enabled 是否启用流超时处理
	Enabled bool `json:"enabled"`
	// Action 超时后的处理方式: "temp_unsched" | "error" | "none"
	Action string `json:"action"`
	// TempUnschedMinutes 临时不可调度持续时间（分钟）
	TempUnschedMinutes int `json:"temp_unsched_minutes"`
	// ThresholdCount 触发阈值次数（累计多少次超时才触发）
	ThresholdCount int `json:"threshold_count"`
	// ThresholdWindowMinutes 阈值窗口时间（分钟）
	ThresholdWindowMinutes int `json:"threshold_window_minutes"`
}

// StreamTimeoutAction 流超时处理方式常量
const (
	StreamTimeoutActionTempUnsched = "temp_unsched" // 临时不可调度
	StreamTimeoutActionError       = "error"        // 标记为错误状态
	StreamTimeoutActionNone        = "none"         // 不处理
)

// DefaultStreamTimeoutSettings 返回默认的流超时配置
func DefaultStreamTimeoutSettings() *StreamTimeoutSettings {
	return &StreamTimeoutSettings{
		Enabled:                false,
		Action:                 StreamTimeoutActionTempUnsched,
		TempUnschedMinutes:     5,
		ThresholdCount:         3,
		ThresholdWindowMinutes: 10,
	}
}

// RectifierSettings 请求整流器配置
type RectifierSettings struct {
	Enabled                  bool     `json:"enabled"`                    // 总开关
	ThinkingSignatureEnabled bool     `json:"thinking_signature_enabled"` // Thinking 签名整流
	ThinkingBudgetEnabled    bool     `json:"thinking_budget_enabled"`    // Thinking Budget 整流
	APIKeySignatureEnabled   bool     `json:"apikey_signature_enabled"`   // API Key 签名整流开关
	APIKeySignaturePatterns  []string `json:"apikey_signature_patterns"`  // API Key 自定义匹配关键词
}

// DefaultRectifierSettings 返回默认的整流器配置（全部启用）
func DefaultRectifierSettings() *RectifierSettings {
	return &RectifierSettings{
		Enabled:                  true,
		ThinkingSignatureEnabled: true,
		ThinkingBudgetEnabled:    true,
		APIKeySignatureEnabled:   true,
	}
}

// Beta Policy 策略常量
const (
	BetaPolicyActionPass   = "pass"   // 透传，不做任何处理
	BetaPolicyActionFilter = "filter" // 过滤，从 beta header 中移除该 token
	BetaPolicyActionBlock  = "block"  // 拦截，直接返回错误

	BetaPolicyScopeAll     = "all"     // 所有账号类型
	BetaPolicyScopeOAuth   = "oauth"   // 仅 OAuth 账号
	BetaPolicyScopeAPIKey  = "apikey"  // 仅 API Key 账号
	BetaPolicyScopeBedrock = "bedrock" // 仅 AWS Bedrock 账号
)

// BetaPolicyRule 单条 Beta 策略规则
type BetaPolicyRule struct {
	BetaToken    string `json:"beta_token"`              // beta token 值
	Action       string `json:"action"`                  // "pass" | "filter" | "block"
	Scope        string `json:"scope"`                   // "all" | "oauth" | "apikey" | "bedrock"
	ErrorMessage string `json:"error_message,omitempty"` // 自定义错误消息 (action=block 时生效)
}

// BetaPolicySettings Beta 策略配置
type BetaPolicySettings struct {
	Rules []BetaPolicyRule `json:"rules"`
}

// ProviderTimeoutSettings 按上游供应商配置的响应超时（秒）
type ProviderTimeoutSettings struct {
	Enabled  bool           `json:"enabled"`
	Timeouts map[string]int `json:"timeouts"` // upstream_provider -> timeout seconds
}

// DefaultProviderTimeoutSettings 返回默认配置（禁用，空映射）
func DefaultProviderTimeoutSettings() *ProviderTimeoutSettings {
	return &ProviderTimeoutSettings{
		Enabled:  false,
		Timeouts: map[string]int{},
	}
}

// ProviderLatencyStats 某个上游供应商的请求时长统计
type ProviderLatencyStats struct {
	Provider   string  `json:"provider"`
	Count      int64   `json:"count"`
	P50Ms      int     `json:"p50_ms"`
	P90Ms      int     `json:"p90_ms"`
	P99Ms      int     `json:"p99_ms"`
	AvgMs      int     `json:"avg_ms"`
	MaxMs      int     `json:"max_ms"`
	TimeoutPct float64 `json:"timeout_pct"` // 超时比例（%）
}

// SLAReport 运维监控 SLA 报告
type SLAReport struct {
	ClientMetrics      SLAClientMetrics     `json:"client_metrics"`
	FailoverMetrics    SLAFailoverMetrics   `json:"failover_metrics"`
	UpstreamErrors     []UpstreamErrorStat  `json:"upstream_errors"`
	ClientErrors       []ClientErrorStat    `json:"client_errors"`
	FailoverPaths      []FailoverPath       `json:"failover_paths"`
	ProviderLatency    []ProviderSLALatency `json:"provider_latency"`
	AccountSuccessRate []AccountSuccessRate `json:"account_success_rate"`
}

type SLAClientMetrics struct {
	Successful              int64   `json:"successful"`
	UsageLogFailed          int64   `json:"usage_log_failed"`
	ClientErrors            int64   `json:"client_errors"`
	RecoveredUpstreamErrors int64   `json:"recovered_upstream_errors"`
	TotalRequests           int64   `json:"total_requests"`
	SuccessRate             float64 `json:"success_rate"`
}

type SLAFailoverMetrics struct {
	TotalWithFailover      int64   `json:"total_with_failover"`
	AvgAttempts            float64 `json:"avg_attempts"`
	MaxAttempts            int     `json:"max_attempts"`
	RecoveredAfterFailover int64   `json:"recovered_after_failover"`
	FailedAfterFailover    int64   `json:"failed_after_failover"`
	FailoverSuccessRate    float64 `json:"failover_success_rate"`
}

type UpstreamErrorStat struct {
	Account        string `json:"account"`
	Provider       string `json:"provider"`
	UpstreamStatus int    `json:"upstream_status"`
	Total          int64  `json:"total"`
	Recovered      int64  `json:"recovered"`
	ClientFacing   int64  `json:"client_facing"`
}

type ClientErrorStat struct {
	StatusCode   int    `json:"status_code"`
	ErrorPhase   string `json:"error_phase"`
	ErrorMessage string `json:"error_message"`
	Count        int64  `json:"count"`
}

type FailoverPath struct {
	RequestID         *string `json:"request_id"`
	Model             *string `json:"model"`
	FinalStatus       int     `json:"final_status"`
	FinalError        *string `json:"final_error"`
	UpstreamErrorsRaw string  `json:"upstream_errors"`
	DurationMs        int     `json:"duration_ms"`
	CreatedAt         any     `json:"created_at"`
}

type AccountSuccessRate struct {
	AccountID   int64   `json:"account_id"`
	AccountName string  `json:"account_name"`
	Provider    string  `json:"provider"`
	Total       int64   `json:"total"`
	Successful  int64   `json:"successful"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

type ProviderSLALatency struct {
	Provider  string `json:"provider"`
	Total     int64  `json:"total"`
	P50Ms     int    `json:"p50_ms"`
	P90Ms     int    `json:"p90_ms"`
	P99Ms     int    `json:"p99_ms"`
	TTFBAvgMs *int   `json:"ttfb_avg_ms"`
}

// OverloadCooldownSettings 529过载冷却配置
type OverloadCooldownSettings struct {
	// Enabled 是否在收到529时暂停账号调度
	Enabled bool `json:"enabled"`
	// CooldownMinutes 冷却时长（分钟）
	CooldownMinutes int `json:"cooldown_minutes"`
}

// DefaultOverloadCooldownSettings 返回默认的过载冷却配置（启用，10分钟）
func DefaultOverloadCooldownSettings() *OverloadCooldownSettings {
	return &OverloadCooldownSettings{
		Enabled:         true,
		CooldownMinutes: 10,
	}
}

// HealthCircuitBreakerConfig 健康度熔断器动态配置
type HealthCircuitBreakerConfig struct {
	// Enabled 总开关（关闭后不再基于健康分数触发短期熔断）
	Enabled bool `json:"enabled"`
	// Threshold 触发熔断的健康分数阈值（0-100），健康分 < Threshold 时熔断
	Threshold int `json:"threshold"`
	// TTLSeconds 短期熔断持续时长（秒）
	TTLSeconds int `json:"ttl_seconds"`
	// MinSamples 最少样本数，低于此值视为健康（数据不足不惩罚）
	MinSamples int `json:"min_samples"`
}

// DefaultHealthCircuitBreakerConfig 返回默认的健康度熔断器配置
func DefaultHealthCircuitBreakerConfig() *HealthCircuitBreakerConfig {
	return &HealthCircuitBreakerConfig{
		Enabled:    true,
		Threshold:  50,
		TTLSeconds: 30,
		MinSamples: 3,
	}
}

// ModelIdentitySettings controls model identity masking behavior.
type ModelIdentitySettings struct {
	LocalResponseEnabled        bool     `json:"local_response_enabled"`
	InstructionInjectionEnabled bool     `json:"instruction_injection_enabled"`
	ResponseRewriteEnabled      bool     `json:"response_rewrite_enabled"`
	HitWords                    []string `json:"hit_words"`
	IdentityPatterns            []string `json:"identity_patterns"`
	ReplyTemplate               string   `json:"reply_template"`
}

// DefaultModelIdentitySettings returns default settings matching previous hardcoded behavior.
func DefaultModelIdentitySettings() *ModelIdentitySettings {
	return &ModelIdentitySettings{
		LocalResponseEnabled:        true,
		InstructionInjectionEnabled: true,
		ResponseRewriteEnabled:      true,
		HitWords: []string{
			"kimi", "moonshot", "minimax", "abab", "qwen", "阿里",
			"doubao", "deepseek", "glm", "chatglm", "智谱",
			"ernie", "文心", "hunyuan", "混元", "grok",
			"阶跃星辰", "yi-lightning", "零一万物",
			"小米", "mimo",
		},
		IdentityPatterns: []string{
			`我是(一个)?(由)?(.{1,20})(训练|开发|创建|推出)的(.{1,20})(语言模型|大模型|大语言模型|AI助手|AI模型)`,
			`作为(一个|一款)?(.{1,20})(模型|AI助手|语言模型|大语言模型)`,
			`我(是|叫)\s*(kimi|moonshot|deepseek|qwen|glm|chatglm|doubao|grok|ernie|hunyuan|mimo)`,
			`I am .{0,30}(model|assistant) (developed|trained|created|built) by`,
			`I'm .{0,30}(model|assistant) (developed|trained|created|built) by`,
			`as (a |an )?.{0,20}(model|AI assistant) (by|from)`,
		},
		ReplyTemplate: "我是一个由{company}训练的{model}大语言模型，旨在通过自然语言处理技术为用户提供专业、高效的解答和支持。如果你有具体的问题或需求,我很乐意帮助你！",
	}
}

// DefaultBetaPolicySettings 返回默认的 Beta 策略配置
func DefaultBetaPolicySettings() *BetaPolicySettings {
	return &BetaPolicySettings{
		Rules: []BetaPolicyRule{
			{
				BetaToken: "fast-mode-2026-02-01",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
			{
				BetaToken: "context-1m-2025-08-07",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
		},
	}
}
