package model

import "time"

// AccountThrottleRule 账户智能限流规则
type AccountThrottleRule struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Enabled           bool      `json:"enabled"`
	Priority          int       `json:"priority"`
	Keywords          []string  `json:"keywords"`
	MatchMode         string    `json:"match_mode"`          // "contains" / "exact"
	TriggerMode       string    `json:"trigger_mode"`        // "immediate" / "accumulated"
	AccumulatedCount  int       `json:"accumulated_count"`   // 累计阈值
	AccumulatedWindow int       `json:"accumulated_window"`  // 累计窗口（秒）
	ActionType        string    `json:"action_type"`         // "duration" / "scheduled_recovery"
	ActionDuration    int       `json:"action_duration"`     // 限流时长（秒）
	ActionRecoverHour int       `json:"action_recover_hour"` // 恢复时刻（0-23）
	Platforms         []string  `json:"platforms"`
	Description       *string   `json:"description"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

const (
	ThrottleMatchContains = "contains"
	ThrottleMatchExact    = "exact"

	ThrottleTriggerImmediate    = "immediate"
	ThrottleTriggerAccumulated  = "accumulated"

	ThrottleActionDuration          = "duration"
	ThrottleActionScheduledRecovery = "scheduled_recovery"
)

// Validate 验证规则配置的有效性
func (r *AccountThrottleRule) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(r.Keywords) == 0 {
		return &ValidationError{Field: "keywords", Message: "at least one keyword is required"}
	}
	if r.MatchMode != ThrottleMatchContains && r.MatchMode != ThrottleMatchExact {
		return &ValidationError{Field: "match_mode", Message: "match_mode must be 'contains' or 'exact'"}
	}
	if r.TriggerMode != ThrottleTriggerImmediate && r.TriggerMode != ThrottleTriggerAccumulated {
		return &ValidationError{Field: "trigger_mode", Message: "trigger_mode must be 'immediate' or 'accumulated'"}
	}
	if r.TriggerMode == ThrottleTriggerAccumulated {
		if r.AccumulatedCount <= 0 {
			return &ValidationError{Field: "accumulated_count", Message: "accumulated_count must be > 0"}
		}
		if r.AccumulatedWindow <= 0 {
			return &ValidationError{Field: "accumulated_window", Message: "accumulated_window must be > 0"}
		}
	}
	if r.ActionType != ThrottleActionDuration && r.ActionType != ThrottleActionScheduledRecovery {
		return &ValidationError{Field: "action_type", Message: "action_type must be 'duration' or 'scheduled_recovery'"}
	}
	if r.ActionType == ThrottleActionDuration && r.ActionDuration <= 0 {
		return &ValidationError{Field: "action_duration", Message: "action_duration must be > 0"}
	}
	if r.ActionType == ThrottleActionScheduledRecovery && (r.ActionRecoverHour < 0 || r.ActionRecoverHour > 23) {
		return &ValidationError{Field: "action_recover_hour", Message: "action_recover_hour must be 0-23"}
	}
	return nil
}
