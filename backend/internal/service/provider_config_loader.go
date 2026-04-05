package service

import (
	"context"
	"encoding/json"
)

// builtinProviderDefaults 内置供应商的默认配置。
// 这些配置作为基线，可被数据库中的配置覆盖。
var builtinProviderDefaults = map[string]*ProviderConfig{
	"zhipu": {
		ProviderID:     "zhipu",
		DisplayName:    "智谱 (GLM)",
		Enabled:        true,
		MaxTokensLimit: 32768,
		ErrorPatterns: ErrorPatterns{
			Billing:   []string{"INSUFFICIENT_BALANCE", "insufficient_balance"},
			RateLimit: []string{"rate limit exceeded", "rate_limit_exceeded"},
			Auth:      []string{"invalid_api_key", "authentication_error"},
		},
	},
	"kimi": {
		ProviderID:     "kimi",
		DisplayName:    "Kimi (Moonshot)",
		Enabled:        true,
		MaxTokensLimit: 32768,
		ErrorPatterns: ErrorPatterns{
			Billing:   []string{"insufficient_balance"},
			RateLimit: []string{"rate_limit_reached"},
			Auth:      []string{"invalid_api_key"},
		},
	},
	"minimax": {
		ProviderID:  "minimax",
		DisplayName: "MiniMax",
		Enabled:     true,
		Features: ProviderFeatures{
			WebSearchInjection: true,
			ImageUnderstanding: true,
			SupportsThinking:   true,
			ToolLoop:           true,
		},
		ErrorPatterns: ErrorPatterns{
			Billing:   []string{"insufficient_balance"},
			RateLimit: []string{"rate_limit"},
			Auth:      []string{"invalid_api_key"},
		},
	},
	"volcengine": {
		ProviderID:     "volcengine",
		DisplayName:    "火山引擎",
		Enabled:        true,
		MaxTokensLimit: 32768,
		ErrorPatterns: ErrorPatterns{
			Auth: []string{"invalid_api_key", "Unauthorized"},
		},
	},
	"aliyun": {
		ProviderID:     "aliyun",
		DisplayName:    "阿里云 (通义)",
		Enabled:        true,
		MaxTokensLimit: 32768,
		ErrorPatterns: ErrorPatterns{
			Auth: []string{"InvalidApiKey", "Unauthorized"},
		},
	},
	"baidu": {
		ProviderID:  "baidu",
		DisplayName: "百度 (文心)",
		Enabled:     true,
		ErrorPatterns: ErrorPatterns{
			Auth: []string{"invalid_api_key", "IAM Certification failed"},
		},
	},
	"xunfei": {
		ProviderID:  "xunfei",
		DisplayName: "讯飞 (星火)",
		Enabled:     true,
		ErrorPatterns: ErrorPatterns{
			Auth: []string{"invalid_api_key"},
		},
	},
}

// LoadProviderConfigs 从 Setting 存储加载供应商配置，与内置默认值合并。
// 数据库配置覆盖内置默认值中的非零字段。
func LoadProviderConfigs(ctx context.Context, settingRepo SettingRepository) map[string]*ProviderConfig {
	result := make(map[string]*ProviderConfig, len(builtinProviderDefaults))

	// 复制内置默认值
	for k, v := range builtinProviderDefaults {
		copied := *v
		result[k] = &copied
	}

	// 从数据库加载覆盖配置
	raw, err := settingRepo.GetValue(ctx, SettingKeyProviderAdapterConfigs)
	if err != nil || raw == "" {
		return result
	}

	var dbConfigs map[string]*ProviderConfig
	if err := json.Unmarshal([]byte(raw), &dbConfigs); err != nil {
		return result
	}

	for id, dbCfg := range dbConfigs {
		if dbCfg == nil {
			continue
		}
		if existing, ok := result[id]; ok {
			mergeProviderConfig(existing, dbCfg)
		} else {
			result[id] = dbCfg
		}
	}

	return result
}

// SaveProviderConfigs 将供应商配置保存到 Setting 存储。
func SaveProviderConfigs(ctx context.Context, settingRepo SettingRepository, configs map[string]*ProviderConfig) error {
	data, err := json.Marshal(configs)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, SettingKeyProviderAdapterConfigs, string(data))
}

// InitProviderRegistry 创建并初始化 ProviderRegistry，加载所有已配置的供应商 adapter。
func InitProviderRegistry(ctx context.Context, settingRepo SettingRepository) *ProviderRegistry {
	registry := NewProviderRegistry()

	configs := LoadProviderConfigs(ctx, settingRepo)
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		adapter := NewGenericAdapter(cfg)
		registry.Register(adapter)
	}

	return registry
}

// mergeProviderConfig 将 src 中的非零值覆盖到 dst。
func mergeProviderConfig(dst, src *ProviderConfig) {
	if src.DisplayName != "" {
		dst.DisplayName = src.DisplayName
	}
	dst.Enabled = src.Enabled
	if src.TimeoutSeconds > 0 {
		dst.TimeoutSeconds = src.TimeoutSeconds
	}
	if src.MaxTokensLimit > 0 {
		dst.MaxTokensLimit = src.MaxTokensLimit
	}
	if len(src.FailoverCodes) > 0 {
		dst.FailoverCodes = src.FailoverCodes
	}
	if len(src.ModelMapping) > 0 {
		dst.ModelMapping = src.ModelMapping
	}
	if len(src.ErrorPatterns.Billing) > 0 {
		dst.ErrorPatterns.Billing = src.ErrorPatterns.Billing
	}
	if len(src.ErrorPatterns.RateLimit) > 0 {
		dst.ErrorPatterns.RateLimit = src.ErrorPatterns.RateLimit
	}
	if len(src.ErrorPatterns.Auth) > 0 {
		dst.ErrorPatterns.Auth = src.ErrorPatterns.Auth
	}
	// Features: 直接使用 src 的值（bool 零值也是有意义的）
	dst.Features = src.Features
}
