package service

import "context"

// UserGroupRequestQuota 表示用户在某个分组下的按次配额状态。
type UserGroupRequestQuota struct {
	RequestQuota     int64 `json:"request_quota"`
	RequestQuotaUsed int64 `json:"request_quota_used"`
}

// Remaining 返回剩余可用次数。
func (q *UserGroupRequestQuota) Remaining() int64 {
	if q == nil || q.RequestQuota <= 0 {
		return 0
	}
	remaining := q.RequestQuota - q.RequestQuotaUsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// UserGroupRequestQuotaRepository 定义用户分组按次配额的存取能力。
type UserGroupRequestQuotaRepository interface {
	GetRequestQuotasByUserID(ctx context.Context, userID int64) (map[int64]int64, error)
	GetRequestQuotaByUserAndGroup(ctx context.Context, userID, groupID int64) (*UserGroupRequestQuota, error)
	SyncUserGroupRequestQuotas(ctx context.Context, userID int64, quotas map[int64]*int64) error
	IncrementRequestQuotaUsed(ctx context.Context, userID, groupID, amount int64) (bool, error)
}
