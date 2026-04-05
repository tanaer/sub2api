//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeProviderSettingRepo struct {
	values map[string]string
}

func (r *fakeProviderSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	v, ok := r.values[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: v}, nil
}

func (r *fakeProviderSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	v, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return v, nil
}

func (r *fakeProviderSettingRepo) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = make(map[string]string)
	}
	r.values[key] = value
	return nil
}

func (r *fakeProviderSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := r.values[k]; ok {
			result[k] = v
		}
	}
	return result, nil
}

func (r *fakeProviderSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	for k, v := range settings {
		r.values[k] = v
	}
	return nil
}

func (r *fakeProviderSettingRepo) GetAll(_ context.Context) (map[string]string, error) {
	return r.values, nil
}

func (r *fakeProviderSettingRepo) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestLoadProviderConfigs_DefaultsOnly(t *testing.T) {
	repo := &fakeProviderSettingRepo{values: map[string]string{}}
	configs := LoadProviderConfigs(context.Background(), repo)

	require.Contains(t, configs, "zhipu")
	assert.Equal(t, "智谱 (GLM)", configs["zhipu"].DisplayName)
	assert.Equal(t, 32768, configs["zhipu"].MaxTokensLimit)

	require.Contains(t, configs, "minimax")
	assert.True(t, configs["minimax"].Features.WebSearchInjection)
	assert.True(t, configs["minimax"].Features.ToolLoop)
}

func TestLoadProviderConfigs_DBOverride(t *testing.T) {
	dbConfigs := map[string]*ProviderConfig{
		"zhipu": {
			ProviderID:     "zhipu",
			DisplayName:    "智谱 (自定义)",
			Enabled:        true,
			MaxTokensLimit: 16384,
		},
		"custom_provider": {
			ProviderID:  "custom_provider",
			DisplayName: "Custom",
			Enabled:     true,
		},
	}
	data, _ := json.Marshal(dbConfigs)

	repo := &fakeProviderSettingRepo{values: map[string]string{
		SettingKeyProviderAdapterConfigs: string(data),
	}}
	configs := LoadProviderConfigs(context.Background(), repo)

	// zhipu should have overridden display name and max_tokens
	require.Contains(t, configs, "zhipu")
	assert.Equal(t, "智谱 (自定义)", configs["zhipu"].DisplayName)
	assert.Equal(t, 16384, configs["zhipu"].MaxTokensLimit)

	// custom_provider should be added
	require.Contains(t, configs, "custom_provider")
	assert.Equal(t, "Custom", configs["custom_provider"].DisplayName)

	// minimax should still have defaults
	require.Contains(t, configs, "minimax")
}

func TestLoadProviderConfigs_InvalidJSON(t *testing.T) {
	repo := &fakeProviderSettingRepo{values: map[string]string{
		SettingKeyProviderAdapterConfigs: "not-json",
	}}
	configs := LoadProviderConfigs(context.Background(), repo)

	// Should fall back to defaults
	require.Contains(t, configs, "zhipu")
	require.Contains(t, configs, "minimax")
}

func TestSaveProviderConfigs(t *testing.T) {
	repo := &fakeProviderSettingRepo{values: map[string]string{}}
	configs := map[string]*ProviderConfig{
		"test": {ProviderID: "test", DisplayName: "Test", Enabled: true},
	}
	err := SaveProviderConfigs(context.Background(), repo, configs)
	require.NoError(t, err)

	raw := repo.values[SettingKeyProviderAdapterConfigs]
	var loaded map[string]*ProviderConfig
	require.NoError(t, json.Unmarshal([]byte(raw), &loaded))
	assert.Equal(t, "Test", loaded["test"].DisplayName)
}

func TestInitProviderRegistry(t *testing.T) {
	repo := &fakeProviderSettingRepo{values: map[string]string{}}
	registry := InitProviderRegistry(context.Background(), repo)

	// Should have native + all enabled builtins
	assert.True(t, registry.Has("native"))
	assert.True(t, registry.Has("zhipu"))
	assert.True(t, registry.Has("minimax"))

	// Get zhipu adapter
	zhipu := registry.Get("zhipu")
	assert.Equal(t, "zhipu", zhipu.ID())
}

func TestMergeProviderConfig(t *testing.T) {
	dst := &ProviderConfig{
		ProviderID:     "test",
		DisplayName:    "Old",
		Enabled:        true,
		MaxTokensLimit: 32768,
		ErrorPatterns:  ErrorPatterns{Billing: []string{"old_pattern"}},
	}
	src := &ProviderConfig{
		DisplayName:    "New",
		Enabled:        false,
		MaxTokensLimit: 16384,
	}

	mergeProviderConfig(dst, src)
	assert.Equal(t, "New", dst.DisplayName)
	assert.False(t, dst.Enabled)
	assert.Equal(t, 16384, dst.MaxTokensLimit)
	// Billing patterns should remain since src has none
	assert.Equal(t, []string{"old_pattern"}, dst.ErrorPatterns.Billing)
}
