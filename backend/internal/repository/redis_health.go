package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// globalRedisHealth is the package-level singleton, initialized by InitRedisHealth.
var globalRedisHealth *RedisHealth

// GetRedisHealth returns the global Redis health monitor (nil if not initialized).
func GetRedisHealth() *RedisHealth {
	return globalRedisHealth
}

// InitRedisHealth initializes the global Redis health monitor.
// Called once during application startup (from ProvideRedis or wire setup).
func InitRedisHealth(rdb *redis.Client) *RedisHealth {
	h := NewRedisHealth(rdb, 3*time.Second)
	globalRedisHealth = h
	return h
}

// RedisHealth tracks Redis availability via periodic PING checks.
// When Redis is down, critical-path code can fail-open instead of
// returning errors to every user request.
type RedisHealth struct {
	rdb *redis.Client

	available atomic.Bool
	mu        sync.RWMutex
	lastError atomic.Value // stores string
	downSince atomic.Value // stores time.Time (zero = not down)

	stopCh chan struct{}
	once   sync.Once
}

// NewRedisHealth creates a health monitor and starts background probing.
// interval controls how often PING is sent (recommended: 3-5s).
func NewRedisHealth(rdb *redis.Client, interval time.Duration) *RedisHealth {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	h := &RedisHealth{
		rdb:    rdb,
		stopCh: make(chan struct{}),
	}
	// Assume available until first probe completes.
	h.available.Store(true)
	h.downSince.Store(time.Time{})
	h.lastError.Store("")

	// Run initial probe synchronously so callers get accurate state immediately.
	h.probe()

	go h.loop(interval)
	return h
}

// Available returns true if the last probe succeeded.
func (h *RedisHealth) Available() bool {
	return h.available.Load()
}

// LastError returns the last error string (empty if healthy).
func (h *RedisHealth) LastError() string {
	if v := h.lastError.Load(); v != nil {
		return v.(string)
	}
	return ""
}

// DownSince returns when Redis became unavailable (zero value if currently up).
func (h *RedisHealth) DownSince() time.Time {
	if v := h.downSince.Load(); v != nil {
		return v.(time.Time)
	}
	return time.Time{}
}

// DownDuration returns how long Redis has been down (0 if up).
func (h *RedisHealth) DownDuration() time.Duration {
	ds := h.DownSince()
	if ds.IsZero() {
		return 0
	}
	return time.Since(ds)
}

// Stop terminates the background health loop.
func (h *RedisHealth) Stop() {
	h.once.Do(func() { close(h.stopCh) })
}

func (h *RedisHealth) probe() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := h.rdb.Ping(ctx).Err()
	if err == nil {
		wasDown := !h.available.Swap(true)
		if wasDown {
			h.downSince.Store(time.Time{})
			h.lastError.Store("")
		}
		return
	}

	wasUp := h.available.Swap(false)
	h.lastError.Store(err.Error())
	if wasUp {
		// Just went down — record timestamp.
		h.downSince.Store(time.Now())
	}
}

func (h *RedisHealth) loop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.probe()
		}
	}
}
