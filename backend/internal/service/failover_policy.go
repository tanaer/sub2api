package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// FailoverStatusCodesConfig 故障转移状态码配置
type FailoverStatusCodesConfig struct {
	Codes      []int `json:"codes"`
	Include5xx bool  `json:"include_5xx"`
}

// defaultFailoverCodes 所有路径当前状态码的并集
var defaultFailoverCodes = &FailoverStatusCodesConfig{
	Codes:      []int{400, 401, 402, 403, 404, 429, 529},
	Include5xx: true,
}

// FailoverPolicy 统一故障转移状态码判断。
// 从 SettingService 缓存读取配置，O(1) map 查找。
type FailoverPolicy struct {
	settingService *SettingService

	// 内联缓存：避免每次请求都读 SettingService
	codeMap    atomic.Value // map[int]bool
	include5xx atomic.Bool
}

var failoverPolicySF singleflight.Group

const (
	failoverPolicyCacheTTL  = 60 * time.Second
	failoverPolicyErrorTTL  = 5 * time.Second
	failoverPolicyDBTimeout = 5 * time.Second
)

// cachedFailoverPolicy 缓存的故障转移策略
type cachedFailoverPolicy struct {
	codeMap    map[int]bool
	include5xx bool
	expiresAt  int64 // unix nano
}

var failoverPolicyCache atomic.Value // *cachedFailoverPolicy

// NewFailoverPolicy 创建故障转移策略实例
func NewFailoverPolicy(settingService *SettingService) *FailoverPolicy {
	p := &FailoverPolicy{
		settingService: settingService,
	}
	// 初始化默认值
	p.applyConfig(defaultFailoverCodes)
	return p
}

// ShouldFailover 统一判断是否需要故障转移。
// nil-safe：当 p == nil 时使用默认规则（与旧硬编码逻辑一致）。
func (p *FailoverPolicy) ShouldFailover(statusCode int) bool {
	if p == nil {
		// nil-safe fallback: apply default codes inline to avoid allocation
		switch statusCode {
		case 400, 401, 402, 403, 404, 429, 529:
			return true
		default:
			return statusCode >= 500
		}
	}
	p.ensureLoaded()
	if m, ok := p.codeMap.Load().(map[int]bool); ok && m[statusCode] {
		return true
	}
	return p.include5xx.Load() && statusCode >= 500
}

// ensureLoaded 确保缓存已加载且未过期
func (p *FailoverPolicy) ensureLoaded() {
	if cached, ok := failoverPolicyCache.Load().(*cachedFailoverPolicy); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return
		}
	}
	p.refresh()
}

// refresh 从 SettingService 刷新配置
func (p *FailoverPolicy) refresh() {
	if p.settingService == nil {
		p.applyConfig(defaultFailoverCodes)
		p.storeCache(defaultFailoverCodes, failoverPolicyCacheTTL)
		return
	}
	_, _, _ = failoverPolicySF.Do("failover_policy", func() (any, error) {
		if cached, ok := failoverPolicyCache.Load().(*cachedFailoverPolicy); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return nil, nil
			}
		}
		dbCtx, cancel := context.WithTimeout(context.Background(), failoverPolicyDBTimeout)
		defer cancel()
		value, err := p.settingService.settingRepo.GetValue(dbCtx, SettingKeyGatewayFailoverStatusCodes)
		if err != nil || value == "" {
			if err != nil {
				slog.Warn("failed to get failover status codes setting", "error", err)
			}
			p.applyConfig(defaultFailoverCodes)
			p.storeCache(defaultFailoverCodes, failoverPolicyErrorTTL)
			return nil, nil
		}
		var cfg FailoverStatusCodesConfig
		if err := json.Unmarshal([]byte(value), &cfg); err != nil {
			slog.Warn("failed to parse failover status codes setting", "error", err, "value", value)
			p.applyConfig(defaultFailoverCodes)
			p.storeCache(defaultFailoverCodes, failoverPolicyErrorTTL)
			return nil, nil
		}
		if len(cfg.Codes) == 0 {
			p.applyConfig(defaultFailoverCodes)
			p.storeCache(defaultFailoverCodes, failoverPolicyCacheTTL)
			return nil, nil
		}
		p.applyConfig(&cfg)
		p.storeCache(&cfg, failoverPolicyCacheTTL)
		return nil, nil
	})
}

// applyConfig 应用配置到内联缓存
func (p *FailoverPolicy) applyConfig(cfg *FailoverStatusCodesConfig) {
	if cfg == nil {
		cfg = defaultFailoverCodes
	}
	m := make(map[int]bool, len(cfg.Codes))
	for _, code := range cfg.Codes {
		m[code] = true
	}
	p.codeMap.Store(m)
	p.include5xx.Store(cfg.Include5xx)
}

// storeCache 存储缓存
func (p *FailoverPolicy) storeCache(cfg *FailoverStatusCodesConfig, ttl time.Duration) {
	m := make(map[int]bool, len(cfg.Codes))
	for _, code := range cfg.Codes {
		m[code] = true
	}
	failoverPolicyCache.Store(&cachedFailoverPolicy{
		codeMap:    m,
		include5xx: cfg.Include5xx,
		expiresAt:  time.Now().Add(ttl).UnixNano(),
	})
}

// InvalidateCache 刷新缓存（UpdateSettings 调用）
func (p *FailoverPolicy) InvalidateCache(cfg *FailoverStatusCodesConfig) {
	failoverPolicySF.Forget("failover_policy")
	if cfg != nil && len(cfg.Codes) > 0 {
		p.applyConfig(cfg)
		p.storeCache(cfg, failoverPolicyCacheTTL)
	} else {
		p.applyConfig(defaultFailoverCodes)
		p.storeCache(defaultFailoverCodes, failoverPolicyCacheTTL)
	}
}
