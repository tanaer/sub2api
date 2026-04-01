//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type workbenchSettingRepoStub struct {
	values map[string]string
}

func (s *workbenchSettingRepoStub) Get(ctx context.Context, key string) (*Setting, error) {
	panic("unexpected Get call")
}

func (s *workbenchSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (s *workbenchSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *workbenchSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *workbenchSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *workbenchSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *workbenchSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func TestSettingService_WorkbenchRedeemPresets_DefaultsToEmptySlice(t *testing.T) {
	svc := NewSettingService(&workbenchSettingRepoStub{}, &config.Config{})

	presets, err := svc.GetWorkbenchRedeemPresets(context.Background())
	require.NoError(t, err)
	require.Empty(t, presets)
}

func TestSettingService_WorkbenchRedeemPresets_SetAndGet(t *testing.T) {
	repo := &workbenchSettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	expected := []WorkbenchRedeemPreset{
		{
			ID:           "preset-b",
			Name:         "B套餐",
			Enabled:      true,
			SortOrder:    20,
			Type:         RedeemTypeBalance,
			Value:        88,
			Template:     "密钥：{{code}}",
			ValidityDays: 30,
		},
		{
			ID:           "preset-a",
			Name:         "A套餐",
			Enabled:      true,
			SortOrder:    10,
			Type:         RedeemTypeGroupRequestQuota,
			Value:        100,
			GroupID:      workbenchInt64Ptr(9),
			Template:     "{{code}}",
			ValidityDays: 30,
		},
	}

	err := svc.SetWorkbenchRedeemPresets(context.Background(), expected)
	require.NoError(t, err)

	got, err := svc.GetWorkbenchRedeemPresets(context.Background())
	require.NoError(t, err)
	require.Equal(t, []WorkbenchRedeemPreset{
		expected[1],
		expected[0],
	}, got)
}

func TestSettingService_WorkbenchRedeemTemplates_SetAndGet(t *testing.T) {
	repo := &workbenchSettingRepoStub{}
	svc := NewSettingService(repo, &config.Config{})
	expected := []WorkbenchRedeemTemplate{
		{
			ID:        "template-b",
			Name:      "B模板",
			Enabled:   true,
			SortOrder: 20,
			Content:   "密钥：{{code}}",
		},
		{
			ID:        "template-a",
			Name:      "A模板",
			Enabled:   true,
			SortOrder: 10,
			Content:   "{{code}}",
		},
	}

	err := svc.SetWorkbenchRedeemTemplates(context.Background(), expected)
	require.NoError(t, err)

	got, err := svc.GetWorkbenchRedeemTemplates(context.Background())
	require.NoError(t, err)
	require.Equal(t, []WorkbenchRedeemTemplate{
		expected[1],
		expected[0],
	}, got)
}

func workbenchInt64Ptr(v int64) *int64 {
	return &v
}
