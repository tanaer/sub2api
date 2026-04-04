package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type subscriptionPlanRepository struct {
	client *dbent.Client
}

func NewSubscriptionPlanRepository(client *dbent.Client) service.SubscriptionPlanRepository {
	return &subscriptionPlanRepository{client: client}
}

func (r *subscriptionPlanRepository) Create(ctx context.Context, plan *service.SubscriptionPlan) error {
	client := clientFromContext(ctx, r.client)
	builder := client.SubscriptionPlan.Create().
		SetName(plan.Name).
		SetBillingMode(plan.BillingMode).
		SetRequestQuota(plan.RequestQuota).
		SetDailyLimitUsd(plan.DailyLimitUSD).
		SetWeeklyLimitUsd(plan.WeeklyLimitUSD).
		SetMonthlyLimitUsd(plan.MonthlyLimitUSD).
		SetValidityDays(plan.ValidityDays).
		SetStatus(plan.Status)

	if plan.Description != "" {
		builder.SetDescription(plan.Description)
	}
	if plan.GroupID != nil {
		builder.SetGroupID(*plan.GroupID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}
	plan.ID = created.ID
	return nil
}

func (r *subscriptionPlanRepository) GetByID(ctx context.Context, id int64) (*service.SubscriptionPlan, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.SubscriptionPlan.Query().
		Where(subscriptionplan.IDEQ(id), subscriptionplan.DeletedAtIsNil()).
		WithGroup().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSubscriptionPlanNotFound, nil)
	}
	return subscriptionPlanEntityToService(m), nil
}

func (r *subscriptionPlanRepository) List(ctx context.Context, status string) ([]service.SubscriptionPlan, error) {
	client := clientFromContext(ctx, r.client)
	query := client.SubscriptionPlan.Query().
		Where(subscriptionplan.DeletedAtIsNil()).
		WithGroup(func(q *dbent.GroupQuery) {
			q.Select("id", "name", "platform", "status")
		}).
		Order(dbent.Desc(subscriptionplan.FieldCreatedAt))

	if status != "" {
		query = query.Where(subscriptionplan.StatusEQ(status))
	}

	models, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.SubscriptionPlan, 0, len(models))
	for _, m := range models {
		if p := subscriptionPlanEntityToService(m); p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

func (r *subscriptionPlanRepository) Update(ctx context.Context, plan *service.SubscriptionPlan) error {
	client := clientFromContext(ctx, r.client)
	builder := client.SubscriptionPlan.UpdateOneID(plan.ID).
		SetName(plan.Name).
		SetBillingMode(plan.BillingMode).
		SetRequestQuota(plan.RequestQuota).
		SetDailyLimitUsd(plan.DailyLimitUSD).
		SetWeeklyLimitUsd(plan.WeeklyLimitUSD).
		SetMonthlyLimitUsd(plan.MonthlyLimitUSD).
		SetValidityDays(plan.ValidityDays).
		SetStatus(plan.Status)

	if plan.Description != "" {
		builder.SetDescription(plan.Description)
	} else {
		builder.ClearDescription()
	}
	if plan.GroupID != nil {
		builder.SetGroupID(*plan.GroupID)
	} else {
		builder.ClearGroupID()
	}

	_, err := builder.Save(ctx)
	return translatePersistenceError(err, service.ErrSubscriptionPlanNotFound, nil)
}

func (r *subscriptionPlanRepository) Delete(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	return client.SubscriptionPlan.DeleteOneID(id).Exec(ctx)
}

func subscriptionPlanEntityToService(m *dbent.SubscriptionPlan) *service.SubscriptionPlan {
	if m == nil {
		return nil
	}
	out := &service.SubscriptionPlan{
		ID:              m.ID,
		Name:            m.Name,
		BillingMode:     m.BillingMode,
		RequestQuota:    m.RequestQuota,
		DailyLimitUSD:   m.DailyLimitUsd,
		WeeklyLimitUSD:  m.WeeklyLimitUsd,
		MonthlyLimitUSD:  m.MonthlyLimitUsd,
		ValidityDays:    m.ValidityDays,
		Status:          m.Status,
	}
	if m.Description != nil {
		out.Description = *m.Description
	}
	if m.GroupID != nil {
		out.GroupID = m.GroupID
	}
	if m.Edges.Group != nil {
		g := m.Edges.Group
		out.Group = &service.SubscriptionPlanGroup{
			ID:       g.ID,
			Name:     g.Name,
			Platform: g.Platform,
		}
	}
	return out
}
