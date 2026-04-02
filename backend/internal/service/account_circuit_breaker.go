package service

import (
	"sync"
	"time"
)

// AccountCircuitBreaker 进程内存级账号熔断器。
//
// 当上游账号返回 401/403 等致命错误时，立即在内存中标记该账号不可调度，
// 后续请求在选号阶段即时跳过，无需等待 outbox → snapshot 的异步更新周期。
//
// 熔断条目会在 TTL 到期后自动失效（保守默认 2 分钟），
// 此时 outbox 早已完成 snapshot 重建，DB 中的真实状态接管调度决策。
type AccountCircuitBreaker struct {
	mu      sync.RWMutex
	tripped map[int64]time.Time // account_id → 过期时间
}

// NewAccountCircuitBreaker 创建账号熔断器实例。
func NewAccountCircuitBreaker() *AccountCircuitBreaker {
	cb := &AccountCircuitBreaker{
		tripped: make(map[int64]time.Time),
	}
	go cb.cleanupLoop()
	return cb
}

// defaultCircuitBreakerTTL 熔断条目的默认存活时间。
// 应远大于 outbox 轮询间隔（默认 1s），但足够小以免影响已恢复的账号。
const defaultCircuitBreakerTTL = 2 * time.Minute

// Trip 熔断指定账号（立即从调度中剔除）。
func (cb *AccountCircuitBreaker) Trip(accountID int64) {
	cb.TripWithTTL(accountID, defaultCircuitBreakerTTL)
}

// TripWithTTL 熔断指定账号，自定义 TTL。
func (cb *AccountCircuitBreaker) TripWithTTL(accountID int64, ttl time.Duration) {
	if accountID <= 0 || ttl <= 0 {
		return
	}
	cb.mu.Lock()
	cb.tripped[accountID] = time.Now().Add(ttl)
	cb.mu.Unlock()
}

// IsTripped 检查账号是否处于熔断状态。
func (cb *AccountCircuitBreaker) IsTripped(accountID int64) bool {
	cb.mu.RLock()
	deadline, ok := cb.tripped[accountID]
	cb.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(deadline) {
		// 惰性清除过期条目
		cb.mu.Lock()
		if d, exists := cb.tripped[accountID]; exists && time.Now().After(d) {
			delete(cb.tripped, accountID)
		}
		cb.mu.Unlock()
		return false
	}
	return true
}

// Reset 清除指定账号的熔断状态（账号恢复时调用）。
func (cb *AccountCircuitBreaker) Reset(accountID int64) {
	cb.mu.Lock()
	delete(cb.tripped, accountID)
	cb.mu.Unlock()
}

// cleanupLoop 定期清理过期条目，防止 map 无限增长。
func (cb *AccountCircuitBreaker) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cb.cleanup()
	}
}

func (cb *AccountCircuitBreaker) cleanup() {
	now := time.Now()
	cb.mu.Lock()
	for id, deadline := range cb.tripped {
		if now.After(deadline) {
			delete(cb.tripped, id)
		}
	}
	cb.mu.Unlock()
}
