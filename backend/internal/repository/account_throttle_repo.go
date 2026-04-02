package repository

import (
	"context"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/accountthrottlerule"
	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type accountThrottleRepository struct {
	client *ent.Client
}

func NewAccountThrottleRepository(client *ent.Client) service.AccountThrottleRepository {
	return &accountThrottleRepository{client: client}
}

func (r *accountThrottleRepository) List(ctx context.Context) ([]*model.AccountThrottleRule, error) {
	rules, err := r.client.AccountThrottleRule.Query().
		Order(ent.Asc(accountthrottlerule.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*model.AccountThrottleRule, len(rules))
	for i, rule := range rules {
		result[i] = r.toModel(rule)
	}
	return result, nil
}

func (r *accountThrottleRepository) GetByID(ctx context.Context, id int64) (*model.AccountThrottleRule, error) {
	rule, err := r.client.AccountThrottleRule.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(rule), nil
}

func (r *accountThrottleRepository) Create(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error) {
	builder := r.client.AccountThrottleRule.Create().
		SetName(rule.Name).
		SetEnabled(rule.Enabled).
		SetPriority(rule.Priority).
		SetMatchMode(rule.MatchMode).
		SetTriggerMode(rule.TriggerMode).
		SetAccumulatedCount(rule.AccumulatedCount).
		SetAccumulatedWindow(rule.AccumulatedWindow).
		SetActionType(rule.ActionType).
		SetActionDuration(rule.ActionDuration).
		SetActionRecoverHour(rule.ActionRecoverHour)

	if len(rule.Keywords) > 0 {
		builder.SetKeywords(rule.Keywords)
	}
	if len(rule.Platforms) > 0 {
		builder.SetPlatforms(rule.Platforms)
	}
	if rule.Description != nil {
		builder.SetDescription(*rule.Description)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toModel(created), nil
}

func (r *accountThrottleRepository) Update(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error) {
	builder := r.client.AccountThrottleRule.UpdateOneID(rule.ID).
		SetName(rule.Name).
		SetEnabled(rule.Enabled).
		SetPriority(rule.Priority).
		SetMatchMode(rule.MatchMode).
		SetTriggerMode(rule.TriggerMode).
		SetAccumulatedCount(rule.AccumulatedCount).
		SetAccumulatedWindow(rule.AccumulatedWindow).
		SetActionType(rule.ActionType).
		SetActionDuration(rule.ActionDuration).
		SetActionRecoverHour(rule.ActionRecoverHour)

	if len(rule.Keywords) > 0 {
		builder.SetKeywords(rule.Keywords)
	} else {
		builder.ClearKeywords()
	}
	if len(rule.Platforms) > 0 {
		builder.SetPlatforms(rule.Platforms)
	} else {
		builder.ClearPlatforms()
	}
	if rule.Description != nil {
		builder.SetDescription(*rule.Description)
	} else {
		builder.ClearDescription()
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return nil, err
	}
	return r.toModel(updated), nil
}

func (r *accountThrottleRepository) Delete(ctx context.Context, id int64) error {
	return r.client.AccountThrottleRule.DeleteOneID(id).Exec(ctx)
}

func (r *accountThrottleRepository) toModel(e *ent.AccountThrottleRule) *model.AccountThrottleRule {
	rule := &model.AccountThrottleRule{
		ID:                int64(e.ID),
		Name:              e.Name,
		Enabled:           e.Enabled,
		Priority:          e.Priority,
		Keywords:          e.Keywords,
		MatchMode:         e.MatchMode,
		TriggerMode:       e.TriggerMode,
		AccumulatedCount:  e.AccumulatedCount,
		AccumulatedWindow: e.AccumulatedWindow,
		ActionType:        e.ActionType,
		ActionDuration:    e.ActionDuration,
		ActionRecoverHour: e.ActionRecoverHour,
		Platforms:         e.Platforms,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}

	if e.Description != nil {
		rule.Description = e.Description
	}

	if rule.Keywords == nil {
		rule.Keywords = []string{}
	}
	if rule.Platforms == nil {
		rule.Platforms = []string{}
	}

	return rule
}
