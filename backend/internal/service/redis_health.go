package service

import (
	"sync"
	"time"
)

// RedisHealthStatus exposes the Redis health state needed by upper layers.
type RedisHealthStatus interface {
	Available() bool
	LastError() string
	DownSince() time.Time
	DownDuration() time.Duration
}

var (
	redisHealthStatusMu sync.RWMutex
	redisHealthStatus   RedisHealthStatus
)

// SetRedisHealthStatus stores the process-wide Redis health monitor.
func SetRedisHealthStatus(status RedisHealthStatus) {
	redisHealthStatusMu.Lock()
	defer redisHealthStatusMu.Unlock()
	redisHealthStatus = status
}

// GetRedisHealthStatus returns the process-wide Redis health monitor.
func GetRedisHealthStatus() RedisHealthStatus {
	redisHealthStatusMu.RLock()
	defer redisHealthStatusMu.RUnlock()
	return redisHealthStatus
}
