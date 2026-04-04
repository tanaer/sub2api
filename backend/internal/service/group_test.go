//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGroup_GetImagePrice_1K 测试 1K 尺寸返回正确价格
func TestGroup_GetImagePrice_1K(t *testing.T) {
	price := 0.10
	group := &Group{
		ImagePrice1K: &price,
	}

	result := group.GetImagePrice("1K")
	require.NotNil(t, result)
	require.InDelta(t, 0.10, *result, 0.0001)
}

// TestGroup_GetImagePrice_2K 测试 2K 尺寸返回正确价格
func TestGroup_GetImagePrice_2K(t *testing.T) {
	price := 0.15
	group := &Group{
		ImagePrice2K: &price,
	}

	result := group.GetImagePrice("2K")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)
}

// TestGroup_GetImagePrice_4K 测试 4K 尺寸返回正确价格
func TestGroup_GetImagePrice_4K(t *testing.T) {
	price := 0.30
	group := &Group{
		ImagePrice4K: &price,
	}

	result := group.GetImagePrice("4K")
	require.NotNil(t, result)
	require.InDelta(t, 0.30, *result, 0.0001)
}

// TestGroup_GetImagePrice_UnknownSize 测试未知尺寸回退 2K
func TestGroup_GetImagePrice_UnknownSize(t *testing.T) {
	price2K := 0.15
	group := &Group{
		ImagePrice2K: &price2K,
	}

	// 未知尺寸 "3K" 应该回退到 2K
	result := group.GetImagePrice("3K")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)

	// 空字符串也回退到 2K
	result = group.GetImagePrice("")
	require.NotNil(t, result)
	require.InDelta(t, 0.15, *result, 0.0001)
}

// TestGroup_GetImagePrice_NilValues 测试未配置时返回 nil
func TestGroup_GetImagePrice_NilValues(t *testing.T) {
	group := &Group{
		// 所有 ImagePrice 字段都是 nil
	}

	require.Nil(t, group.GetImagePrice("1K"))
	require.Nil(t, group.GetImagePrice("2K"))
	require.Nil(t, group.GetImagePrice("4K"))
	require.Nil(t, group.GetImagePrice("unknown"))
}

// TestGroup_GetImagePrice_PartialConfig 测试部分配置
func TestGroup_GetImagePrice_PartialConfig(t *testing.T) {
	price1K := 0.10
	group := &Group{
		ImagePrice1K: &price1K,
		// ImagePrice2K 和 ImagePrice4K 未配置
	}

	result := group.GetImagePrice("1K")
	require.NotNil(t, result)
	require.InDelta(t, 0.10, *result, 0.0001)

	// 2K 和 4K 返回 nil
	require.Nil(t, group.GetImagePrice("2K"))
	require.Nil(t, group.GetImagePrice("4K"))
}

func TestGroup_ResolveModelAlias(t *testing.T) {
	tests := []struct {
		name          string
		aliases       map[string]string
		fallbackModel string
		input         string
		expected      string
	}{
		{
			name:     "nil aliases returns original",
			aliases:  nil,
			input:    "claude-opus-4-6",
			expected: "claude-opus-4-6",
		},
		{
			name:     "empty aliases returns original",
			aliases:  map[string]string{},
			input:    "claude-opus-4-6",
			expected: "claude-opus-4-6",
		},
		{
			name:     "empty model returns empty",
			aliases:  map[string]string{"claude-*": "glm-4-flash"},
			input:    "",
			expected: "",
		},
		{
			name:     "exact match",
			aliases:  map[string]string{"claude-opus-4-6": "glm-4-plus"},
			input:    "claude-opus-4-6",
			expected: "glm-4-plus",
		},
		{
			name:     "wildcard match",
			aliases:  map[string]string{"claude-opus-*": "glm-4-plus"},
			input:    "claude-opus-4-6",
			expected: "glm-4-plus",
		},
		{
			name: "longest wildcard wins",
			aliases: map[string]string{
				"claude-*":      "glm-4-flash",
				"claude-opus-*": "glm-4-plus",
			},
			input:    "claude-opus-4-6",
			expected: "glm-4-plus",
		},
		{
			name: "exact match over wildcard",
			aliases: map[string]string{
				"claude-opus-*":   "glm-4-plus",
				"claude-opus-4-6": "glm-4-0520",
			},
			input:    "claude-opus-4-6",
			expected: "glm-4-0520",
		},
		{
			name: "no match returns original",
			aliases: map[string]string{
				"claude-opus-*": "glm-4-plus",
			},
			input:    "gpt-5.4",
			expected: "gpt-5.4",
		},
		{
			name: "multiple wildcards choose longest",
			aliases: map[string]string{
				"claude-*":           "glm-4-flash",
				"claude-haiku-*":     "glm-4-flash-lite",
				"claude-haiku-4-5-*": "glm-4-flash-v2",
			},
			input:    "claude-haiku-4-5-20251001",
			expected: "glm-4-flash-v2",
		},
		// 兜底模型测试
		{
			name:          "fallback model when no aliases and no match",
			aliases:       nil,
			fallbackModel: "claude-sonnet-4-6-20250514",
			input:         "some-wrong-model",
			expected:      "claude-sonnet-4-6-20250514",
		},
		{
			name:          "fallback model when aliases exist but no match",
			aliases:       map[string]string{"claude-opus-*": "glm-4-plus"},
			fallbackModel: "claude-sonnet-4-6-20250514",
			input:         "gpt-5.4",
			expected:      "claude-sonnet-4-6-20250514",
		},
		{
			name:          "alias match takes priority over fallback",
			aliases:       map[string]string{"claude-opus-*": "glm-4-plus"},
			fallbackModel: "claude-sonnet-4-6-20250514",
			input:         "claude-opus-4-6",
			expected:      "glm-4-plus",
		},
		{
			name:          "empty fallback model returns original",
			aliases:       map[string]string{"claude-opus-*": "glm-4-plus"},
			fallbackModel: "",
			input:         "gpt-5.4",
			expected:      "gpt-5.4",
		},
		{
			name:          "fallback with empty aliases map",
			aliases:       map[string]string{},
			fallbackModel: "default-model",
			input:         "unknown-model",
			expected:      "default-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{ModelAliases: tt.aliases, FallbackModel: tt.fallbackModel}
			result := g.ResolveModelAlias(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
