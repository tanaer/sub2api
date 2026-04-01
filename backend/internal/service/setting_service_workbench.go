package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// GetWorkbenchRedeemPresets 获取运营工具页的一键兑换码预设。
func (s *SettingService) GetWorkbenchRedeemPresets(ctx context.Context) ([]WorkbenchRedeemPreset, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyWorkbenchRedeemPresets)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return []WorkbenchRedeemPreset{}, nil
		}
		return nil, fmt.Errorf("get workbench redeem presets: %w", err)
	}

	if value == "" {
		return []WorkbenchRedeemPreset{}, nil
	}

	var presets []WorkbenchRedeemPreset
	if err := json.Unmarshal([]byte(value), &presets); err != nil {
		return nil, fmt.Errorf("parse workbench redeem presets: %w", err)
	}

	sort.SliceStable(presets, func(i, j int) bool {
		if presets[i].SortOrder != presets[j].SortOrder {
			return presets[i].SortOrder < presets[j].SortOrder
		}
		if presets[i].Name != presets[j].Name {
			return presets[i].Name < presets[j].Name
		}
		return presets[i].ID < presets[j].ID
	})

	return presets, nil
}

// SetWorkbenchRedeemPresets 保存运营工具页的一键兑换码预设。
func (s *SettingService) SetWorkbenchRedeemPresets(ctx context.Context, presets []WorkbenchRedeemPreset) error {
	if presets == nil {
		presets = []WorkbenchRedeemPreset{}
	}

	raw, err := json.Marshal(presets)
	if err != nil {
		return fmt.Errorf("marshal workbench redeem presets: %w", err)
	}

	if err := s.settingRepo.Set(ctx, SettingKeyWorkbenchRedeemPresets, string(raw)); err != nil {
		return fmt.Errorf("set workbench redeem presets: %w", err)
	}

	return nil
}
