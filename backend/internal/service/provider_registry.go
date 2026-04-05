package service

import (
	"sort"
	"strings"
	"sync"
)

// ProviderRegistry 管理所有已注册的 Provider Adapter。
// Gateway Service 通过 account.UpstreamProvider 查找对应 adapter。
// 线程安全，支持运行时注册新 adapter。
type ProviderRegistry struct {
	mu       sync.RWMutex
	adapters map[string]ProviderAdapter // key: provider_id（小写）
	fallback ProviderAdapter            // 未注册的 provider 使用此 adapter
}

// NewProviderRegistry 创建 Registry 并注册 NativeAdapter 作为默认 fallback。
func NewProviderRegistry() *ProviderRegistry {
	native := NewNativeAdapter()
	r := &ProviderRegistry{
		adapters: make(map[string]ProviderAdapter),
		fallback: native,
	}
	r.adapters["native"] = native
	r.adapters[""] = native // 空 upstream_provider 也映射到 native
	return r
}

// Register 注册一个 Provider Adapter。
// 如果 ID 已存在则覆盖。
func (r *ProviderRegistry) Register(adapter ProviderAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[strings.ToLower(adapter.ID())] = adapter
}

// Get 根据 provider ID 查找对应的 adapter。
// 如果未注册则返回 fallback（NativeAdapter）。
func (r *ProviderRegistry) Get(providerID string) ProviderAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.adapters[strings.ToLower(strings.TrimSpace(providerID))]; ok {
		return a
	}
	return r.fallback
}

// GetForAccount 根据 Account 的 UpstreamProvider 查找对应的 adapter。
func (r *ProviderRegistry) GetForAccount(account *Account) ProviderAdapter {
	if account == nil {
		return r.Get("")
	}
	return r.Get(account.UpstreamProvider)
}

// List 返回所有已注册的 adapter（按 ID 排序），不包括空字符串映射。
func (r *ProviderRegistry) List() []ProviderAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[string]bool)
	result := make([]ProviderAdapter, 0, len(r.adapters))
	for id, a := range r.adapters {
		if id == "" {
			continue // 跳过空字符串别名
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, a)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}

// Has 检查指定 provider ID 是否已注册。
func (r *ProviderRegistry) Has(providerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.adapters[strings.ToLower(strings.TrimSpace(providerID))]
	return ok
}
