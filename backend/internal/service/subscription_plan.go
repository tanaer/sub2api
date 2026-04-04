package service

const (
	BillingModePerRequest = "per_request"
	BillingModePerUSD     = "per_usd"

	SubscriptionPlanStatusActive   = "active"
	SubscriptionPlanStatusArchived = "archived"
)

// SubscriptionPlan 订阅计划（套餐商品）
type SubscriptionPlan struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	GroupID         *int64   `json:"group_id,omitempty"`
	BillingMode     string   `json:"billing_mode"`
	RequestQuota    int64    `json:"request_quota"`
	DailyLimitUSD   float64  `json:"daily_limit_usd"`
	WeeklyLimitUSD  float64  `json:"weekly_limit_usd"`
	MonthlyLimitUSD float64  `json:"monthly_limit_usd"`
	ValidityDays    int      `json:"validity_days"`
	Status          string   `json:"status"`

	Group *SubscriptionPlanGroup `json:"group,omitempty"`
}

// SubscriptionPlanGroup 订阅计划中嵌入的分组摘要（带 JSON tag）
type SubscriptionPlanGroup struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
}

func (p *SubscriptionPlan) IsPerRequest() bool {
	return p != nil && p.BillingMode == BillingModePerRequest
}

func (p *SubscriptionPlan) IsPerUSD() bool {
	return p != nil && p.BillingMode == BillingModePerUSD
}
