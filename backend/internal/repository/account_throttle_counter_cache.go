package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const throttleCounterPrefix = "throttle_count:"

var throttleCounterIncrScript = redis.NewScript(`
	local key = KEYS[1]
	local ttl = tonumber(ARGV[1])

	local count = redis.call('INCR', key)
	if count == 1 then
		redis.call('EXPIRE', key, ttl)
	end

	return count
`)

type accountThrottleCounterCache struct {
	rdb *redis.Client
}

func NewAccountThrottleCounterCache(rdb *redis.Client) service.AccountThrottleCounterCache {
	return &accountThrottleCounterCache{rdb: rdb}
}

func (c *accountThrottleCounterCache) IncrementThrottleCount(ctx context.Context, accountID int64, ruleID int64, windowSeconds int) (int64, error) {
	key := fmt.Sprintf("%saccount:%d:rule:%d", throttleCounterPrefix, accountID, ruleID)

	if windowSeconds < 1 {
		windowSeconds = 60
	}

	result, err := throttleCounterIncrScript.Run(ctx, c.rdb, []string{key}, windowSeconds).Int64()
	if err != nil {
		return 0, fmt.Errorf("increment throttle count: %w", err)
	}

	return result, nil
}
