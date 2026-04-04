package service

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// TempUnschedState 临时不可调度状态
type TempUnschedState struct {
	UntilUnix       int64  `json:"until_unix"`        // 解除时间（Unix 时间戳）
	TriggeredAtUnix int64  `json:"triggered_at_unix"` // 触发时间（Unix 时间戳）
	StatusCode      int    `json:"status_code"`       // 触发的错误码
	MatchedKeyword  string `json:"matched_keyword"`   // 匹配的关键词
	RuleIndex       int    `json:"rule_index"`        // 触发的规则索引
	ErrorMessage    string `json:"error_message"`     // 错误消息

	RequestID            string `json:"request_id,omitempty"`             // 请求追踪 ID
	UpstreamStatusCode   int    `json:"upstream_status_code,omitempty"`   // 上游状态码
	UpstreamErrorMessage string `json:"upstream_error_message,omitempty"` // 上游错误摘要
	UpstreamErrorDetail  string `json:"upstream_error_detail,omitempty"`  // 上游错误详情
}

func applyTempUnschedTrace(ctx context.Context, state *TempUnschedState, upstreamStatusCode int, responseBody []byte) {
	if state == nil {
		return
	}

	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		state.RequestID = strings.TrimSpace(requestID)
	}
	if upstreamStatusCode > 0 {
		state.UpstreamStatusCode = upstreamStatusCode
	}
	if msg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(responseBody))); msg != "" {
		state.UpstreamErrorMessage = truncateForLog([]byte(msg), 512)
	}
	if detail := truncateTempUnschedMessage(responseBody, tempUnschedMessageMaxBytes); detail != "" {
		state.UpstreamErrorDetail = detail
	}
}

// TempUnschedCache 临时不可调度缓存接口
type TempUnschedCache interface {
	SetTempUnsched(ctx context.Context, accountID int64, state *TempUnschedState) error
	GetTempUnsched(ctx context.Context, accountID int64) (*TempUnschedState, error)
	DeleteTempUnsched(ctx context.Context, accountID int64) error
}

// TimeoutCounterCache 超时计数器缓存接口
type TimeoutCounterCache interface {
	// IncrementTimeoutCount 增加账户的超时计数，返回当前计数值
	// windowMinutes 是计数窗口时间（分钟），超过此时间计数器会自动重置
	IncrementTimeoutCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error)
	// GetTimeoutCount 获取账户当前的超时计数
	GetTimeoutCount(ctx context.Context, accountID int64) (int64, error)
	// ResetTimeoutCount 重置账户的超时计数
	ResetTimeoutCount(ctx context.Context, accountID int64) error
	// GetTimeoutCountTTL 获取计数器剩余过期时间
	GetTimeoutCountTTL(ctx context.Context, accountID int64) (time.Duration, error)
}
