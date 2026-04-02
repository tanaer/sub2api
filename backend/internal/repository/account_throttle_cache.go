package repository

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	accountThrottleCacheKey  = "account_throttle_rules"
	accountThrottlePubSubKey = "account_throttle_rules_updated"
	accountThrottleCacheTTL  = 24 * time.Hour
)

type accountThrottleCache struct {
	rdb        *redis.Client
	localCache []*model.AccountThrottleRule
	localMu    sync.RWMutex
}

func NewAccountThrottleCache(rdb *redis.Client) service.AccountThrottleCache {
	return &accountThrottleCache{
		rdb: rdb,
	}
}

func (c *accountThrottleCache) Get(ctx context.Context) ([]*model.AccountThrottleRule, bool) {
	c.localMu.RLock()
	if c.localCache != nil {
		rules := c.localCache
		c.localMu.RUnlock()
		return rules, true
	}
	c.localMu.RUnlock()

	data, err := c.rdb.Get(ctx, accountThrottleCacheKey).Bytes()
	if err != nil {
		if err != redis.Nil {
			log.Printf("[AccountThrottleCache] Failed to get from Redis: %v", err)
		}
		return nil, false
	}

	var rules []*model.AccountThrottleRule
	if err := json.Unmarshal(data, &rules); err != nil {
		log.Printf("[AccountThrottleCache] Failed to unmarshal rules: %v", err)
		return nil, false
	}

	c.localMu.Lock()
	c.localCache = rules
	c.localMu.Unlock()

	return rules, true
}

func (c *accountThrottleCache) Set(ctx context.Context, rules []*model.AccountThrottleRule) error {
	data, err := json.Marshal(rules)
	if err != nil {
		return err
	}

	if err := c.rdb.Set(ctx, accountThrottleCacheKey, data, accountThrottleCacheTTL).Err(); err != nil {
		return err
	}

	c.localMu.Lock()
	c.localCache = rules
	c.localMu.Unlock()

	return nil
}

func (c *accountThrottleCache) Invalidate(ctx context.Context) error {
	c.localMu.Lock()
	c.localCache = nil
	c.localMu.Unlock()

	return c.rdb.Del(ctx, accountThrottleCacheKey).Err()
}

func (c *accountThrottleCache) NotifyUpdate(ctx context.Context) error {
	return c.rdb.Publish(ctx, accountThrottlePubSubKey, "refresh").Err()
}

func (c *accountThrottleCache) SubscribeUpdates(ctx context.Context, handler func()) {
	go func() {
		sub := c.rdb.Subscribe(ctx, accountThrottlePubSubKey)
		defer func() { _ = sub.Close() }()

		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				if msg == nil {
					return
				}
				c.localMu.Lock()
				c.localCache = nil
				c.localMu.Unlock()

				handler()
			}
		}
	}()
}
