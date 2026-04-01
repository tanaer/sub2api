package service

// WorkbenchRedeemPreset 表示运营工具页的一键兑换码预设。
type WorkbenchRedeemPreset struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Enabled      bool    `json:"enabled"`
	SortOrder    int     `json:"sort_order"`
	Type         string  `json:"type"`
	Value        float64 `json:"value"`
	GroupID      *int64  `json:"group_id,omitempty"`
	ValidityDays int     `json:"validity_days"`
	Template     string  `json:"template"`
}
