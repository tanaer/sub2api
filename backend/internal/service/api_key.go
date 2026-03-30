package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
)

// API Key status constants
const (
	StatusAPIKeyActive         = "active"
	StatusAPIKeyDisabled       = "disabled"
	StatusAPIKeyQuotaExhausted = "quota_exhausted"
	StatusAPIKeyExpired        = "expired"
)

// Rate limit window durations
const (
	RateLimitWindow5h = 5 * time.Hour
	RateLimitWindow1d = 24 * time.Hour
	RateLimitWindow7d = 7 * 24 * time.Hour
)

// IsWindowExpired returns true if the window starting at windowStart has exceeded the given duration.
// A nil windowStart is treated as expired — no initialized window means any accumulated usage is stale.
func IsWindowExpired(windowStart *time.Time, duration time.Duration) bool {
	return windowStart == nil || time.Since(*windowStart) >= duration
}

type APIKey struct {
	ID          int64
	UserID      int64
	Key         string
	Name        string
	GroupID     *int64
	Status      string
	IPWhitelist []string
	IPBlacklist []string
	// 预编译的 IP 规则，用于认证热路径避免重复 ParseIP/ParseCIDR。
	CompiledIPWhitelist *ip.CompiledIPRules `json:"-"`
	CompiledIPBlacklist *ip.CompiledIPRules `json:"-"`
	LastUsedAt          *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	User                *User
	Group               *Group

	// Quota fields
	Quota            float64    // Quota limit in USD (0 = unlimited)
	QuotaUsed        float64    // Used quota amount
	RequestQuota     int64      // 按次配额总量（0 = 不启用）
	RequestQuotaUsed int64      // 已使用次数
	// 用户+分组维度按次配额，优先级高于 API Key 自身按次配额。
	UserGroupRequestQuota     int64
	UserGroupRequestQuotaUsed int64
	ExpiresAt        *time.Time // Expiration time (nil = never expires)

	// Rate limit fields
	RateLimit5h   float64    // Rate limit in USD per 5h (0 = unlimited)
	RateLimit1d   float64    // Rate limit in USD per 1d (0 = unlimited)
	RateLimit7d   float64    // Rate limit in USD per 7d (0 = unlimited)
	Usage5h       float64    // Used amount in current 5h window
	Usage1d       float64    // Used amount in current 1d window
	Usage7d       float64    // Used amount in current 7d window
	Window5hStart *time.Time // Start of current 5h window
	Window1dStart *time.Time // Start of current 1d window
	Window7dStart *time.Time // Start of current 7d window
}

func (k *APIKey) IsActive() bool {
	return k.Status == StatusActive
}

// HasRateLimits returns true if any rate limit window is configured
func (k *APIKey) HasRateLimits() bool {
	return k.RateLimit5h > 0 || k.RateLimit1d > 0 || k.RateLimit7d > 0
}

// IsExpired checks if the API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsQuotaExhausted checks if the API key quota is exhausted
func (k *APIKey) IsQuotaExhausted() bool {
	if k.Quota <= 0 {
		return false // unlimited
	}
	return k.QuotaUsed >= k.Quota
}

// GetQuotaRemaining returns remaining quota (-1 for unlimited)
func (k *APIKey) GetQuotaRemaining() float64 {
	if k.Quota <= 0 {
		return -1 // unlimited
	}
	remaining := k.Quota - k.QuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasRequestQuota returns true when the API key is configured for request-count billing.
func (k *APIKey) HasRequestQuota() bool {
	return k.RequestQuota > 0
}

// HasUserGroupRequestQuota returns true when a user-group level request quota is configured.
func (k *APIKey) HasUserGroupRequestQuota() bool {
	return k.UserGroupRequestQuota > 0
}

// GetRemainingRequestQuota returns the remaining request count (0 when exhausted or disabled).
func (k *APIKey) GetRemainingRequestQuota() int64 {
	if k.RequestQuota <= 0 {
		return 0
	}
	remaining := k.RequestQuota - k.RequestQuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRemainingUserGroupRequestQuota returns the remaining request count for the user-group quota.
func (k *APIKey) GetRemainingUserGroupRequestQuota() int64 {
	if k.UserGroupRequestQuota <= 0 {
		return 0
	}
	remaining := k.UserGroupRequestQuota - k.UserGroupRequestQuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// HasRemainingRequestQuota returns true when request-count billing can still be used.
func (k *APIKey) HasRemainingRequestQuota() bool {
	return k.GetRemainingRequestQuota() > 0
}

// HasRemainingUserGroupRequestQuota returns true when the user-group request quota still has remaining count.
func (k *APIKey) HasRemainingUserGroupRequestQuota() bool {
	return k.GetRemainingUserGroupRequestQuota() > 0
}

// EffectiveRequestQuotaSource returns the active request-quota source.
// 用户+分组配额优先；未配置时才回退到 API Key 自身配额。
func (k *APIKey) EffectiveRequestQuotaSource() string {
	if k == nil {
		return ""
	}
	if k.HasUserGroupRequestQuota() {
		return "user_group"
	}
	if k.HasRequestQuota() {
		return "api_key"
	}
	return ""
}

// HasEffectiveRequestQuota returns true when an active request-quota source is configured.
func (k *APIKey) HasEffectiveRequestQuota() bool {
	return k.EffectiveRequestQuotaSource() != ""
}

// HasRemainingEffectiveRequestQuota returns true when the active request-quota source still has remaining count.
func (k *APIKey) HasRemainingEffectiveRequestQuota() bool {
	switch k.EffectiveRequestQuotaSource() {
	case "user_group":
		return k.HasRemainingUserGroupRequestQuota()
	case "api_key":
		return k.HasRemainingRequestQuota()
	default:
		return false
	}
}

// GetEffectiveRequestQuota returns the active request-quota limit.
func (k *APIKey) GetEffectiveRequestQuota() int64 {
	switch k.EffectiveRequestQuotaSource() {
	case "user_group":
		return k.UserGroupRequestQuota
	case "api_key":
		return k.RequestQuota
	default:
		return 0
	}
}

// GetEffectiveRequestQuotaUsed returns the active request-quota used count.
func (k *APIKey) GetEffectiveRequestQuotaUsed() int64 {
	switch k.EffectiveRequestQuotaSource() {
	case "user_group":
		return k.UserGroupRequestQuotaUsed
	case "api_key":
		return k.RequestQuotaUsed
	default:
		return 0
	}
}

// GetEffectiveRemainingRequestQuota returns the remaining count of the active request-quota source.
func (k *APIKey) GetEffectiveRemainingRequestQuota() int64 {
	switch k.EffectiveRequestQuotaSource() {
	case "user_group":
		return k.GetRemainingUserGroupRequestQuota()
	case "api_key":
		return k.GetRemainingRequestQuota()
	default:
		return 0
	}
}

// GetDaysUntilExpiry returns days until expiry (-1 for never expires)
func (k *APIKey) GetDaysUntilExpiry() int {
	if k.ExpiresAt == nil {
		return -1 // never expires
	}
	duration := time.Until(*k.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// EffectiveUsage5h returns the 5h window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage5h() float64 {
	if IsWindowExpired(k.Window5hStart, RateLimitWindow5h) {
		return 0
	}
	return k.Usage5h
}

// EffectiveUsage1d returns the 1d window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage1d() float64 {
	if IsWindowExpired(k.Window1dStart, RateLimitWindow1d) {
		return 0
	}
	return k.Usage1d
}

// EffectiveUsage7d returns the 7d window usage, or 0 if the window has expired.
func (k *APIKey) EffectiveUsage7d() float64 {
	if IsWindowExpired(k.Window7dStart, RateLimitWindow7d) {
		return 0
	}
	return k.Usage7d
}

// APIKeyListFilters holds optional filtering parameters for listing API keys.
type APIKeyListFilters struct {
	Search  string
	Status  string
	GroupID *int64 // nil=不筛选, 0=无分组, >0=指定分组
}
