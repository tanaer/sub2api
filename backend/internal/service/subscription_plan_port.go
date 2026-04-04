package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrSubscriptionPlanNotFound = infraerrors.NotFound("SUBSCRIPTION_PLAN_NOT_FOUND", "subscription plan not found")

type SubscriptionPlanRepository interface {
	Create(ctx context.Context, plan *SubscriptionPlan) error
	GetByID(ctx context.Context, id int64) (*SubscriptionPlan, error)
	List(ctx context.Context, status string) ([]SubscriptionPlan, error)
	Update(ctx context.Context, plan *SubscriptionPlan) error
	Delete(ctx context.Context, id int64) error
}
