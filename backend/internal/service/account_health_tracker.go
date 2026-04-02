package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// AccountHealthTracker 基于滑动时间窗口的账号健康度追踪器。
//
// 为每个账号维护最近 N 分钟（默认 5 分钟）的成功/失败计数，
// 计算健康分数（0-100）供调度选号时加权使用。
//
// 设计目标：
//   - 零配置：自动统计，无需手动配置
//   - 低开销：只用原子操作 + 分钟级时间桶，无锁竞争
//   - 自愈性：窗口自动滑动，历史错误自然过期
//   - 区分度：仅在有统计意义时（>=3 次请求）才降权
type AccountHealthTracker struct {
	mu      sync.RWMutex
	buckets map[int64]*accountHealthBuckets
}

const (
	// healthWindowMinutes 滑动窗口大小（分钟）
	healthWindowMinutes = 5
	// healthMinSamples 最少样本数，低于此值视为健康（数据不足不惩罚）
	healthMinSamples = 3
)

// accountHealthBuckets 每个账号的时间桶数组
type accountHealthBuckets struct {
	minutes [healthWindowMinutes]minuteBucket
}

// minuteBucket 单个分钟的统计
type minuteBucket struct {
	minute   int64 // Unix 分钟数（time.Now().Unix() / 60）
	success  atomic.Int32
	failures atomic.Int32
}

// NewAccountHealthTracker 创建健康度追踪器。
func NewAccountHealthTracker() *AccountHealthTracker {
	return &AccountHealthTracker{
		buckets: make(map[int64]*accountHealthBuckets),
	}
}

// RecordSuccess 记录一次成功请求。
func (t *AccountHealthTracker) RecordSuccess(accountID int64) {
	b := t.getBuckets(accountID)
	bucket := b.currentBucket()
	bucket.success.Add(1)
}

// RecordFailure 记录一次失败请求。
func (t *AccountHealthTracker) RecordFailure(accountID int64) {
	b := t.getBuckets(accountID)
	bucket := b.currentBucket()
	bucket.failures.Add(1)
}

// HealthScore 返回账号健康分数（0-100）。
// 100 = 完全健康或数据不足（不惩罚），0 = 窗口内全部失败。
func (t *AccountHealthTracker) HealthScore(accountID int64) int {
	t.mu.RLock()
	b, ok := t.buckets[accountID]
	t.mu.RUnlock()
	if !ok {
		return 100
	}

	now := currentMinute()
	var totalSuccess, totalFailures int32
	for i := range b.minutes {
		m := &b.minutes[i]
		age := now - m.minute
		if age < 0 || age >= healthWindowMinutes {
			continue
		}
		totalSuccess += m.success.Load()
		totalFailures += m.failures.Load()
	}

	total := totalSuccess + totalFailures
	if total < healthMinSamples {
		return 100 // 样本不足，视为健康
	}

	// 成功率 * 100
	return int(totalSuccess * 100 / total)
}

// getBuckets 获取或创建账号的时间桶
func (t *AccountHealthTracker) getBuckets(accountID int64) *accountHealthBuckets {
	t.mu.RLock()
	b, ok := t.buckets[accountID]
	t.mu.RUnlock()
	if ok {
		return b
	}

	t.mu.Lock()
	b, ok = t.buckets[accountID]
	if !ok {
		b = &accountHealthBuckets{}
		t.buckets[accountID] = b
	}
	t.mu.Unlock()
	return b
}

// currentBucket 获取当前分钟对应的桶（自动轮转）
func (b *accountHealthBuckets) currentBucket() *minuteBucket {
	now := currentMinute()
	idx := int(now % healthWindowMinutes)
	bucket := &b.minutes[idx]

	// 如果桶对应的分钟已过期，重置
	if bucket.minute != now {
		bucket.minute = now
		bucket.success.Store(0)
		bucket.failures.Store(0)
	}
	return bucket
}

func currentMinute() int64 {
	return time.Now().Unix() / 60
}
