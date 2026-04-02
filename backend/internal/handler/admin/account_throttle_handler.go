package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// AccountThrottleHandler 处理账户限流规则的 HTTP 请求
type AccountThrottleHandler struct {
	service *service.AccountThrottleService
}

// NewAccountThrottleHandler 创建账户限流规则处理器
func NewAccountThrottleHandler(service *service.AccountThrottleService) *AccountThrottleHandler {
	return &AccountThrottleHandler{service: service}
}

// CreateAccountThrottleRuleRequest 创建规则请求
type CreateAccountThrottleRuleRequest struct {
	Name              string   `json:"name" binding:"required"`
	Enabled           *bool    `json:"enabled"`
	Priority          int      `json:"priority"`
	Keywords          []string `json:"keywords"`
	MatchMode         string   `json:"match_mode"`
	TriggerMode       string   `json:"trigger_mode"`
	AccumulatedCount  *int     `json:"accumulated_count"`
	AccumulatedWindow *int     `json:"accumulated_window"`
	ActionType        string   `json:"action_type"`
	ActionDuration    *int     `json:"action_duration"`
	ActionRecoverHour *int     `json:"action_recover_hour"`
	Platforms         []string `json:"platforms"`
	Description       *string  `json:"description"`
}

// UpdateAccountThrottleRuleRequest 更新规则请求
type UpdateAccountThrottleRuleRequest struct {
	Name              *string  `json:"name"`
	Enabled           *bool    `json:"enabled"`
	Priority          *int     `json:"priority"`
	Keywords          []string `json:"keywords"`
	MatchMode         *string  `json:"match_mode"`
	TriggerMode       *string  `json:"trigger_mode"`
	AccumulatedCount  *int     `json:"accumulated_count"`
	AccumulatedWindow *int     `json:"accumulated_window"`
	ActionType        *string  `json:"action_type"`
	ActionDuration    *int     `json:"action_duration"`
	ActionRecoverHour *int     `json:"action_recover_hour"`
	Platforms         []string `json:"platforms"`
	Description       *string  `json:"description"`
}

// List 获取所有规则
// GET /api/v1/admin/account-throttle-rules
func (h *AccountThrottleHandler) List(c *gin.Context) {
	rules, err := h.service.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rules)
}

// GetByID 根据 ID 获取规则
// GET /api/v1/admin/account-throttle-rules/:id
func (h *AccountThrottleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid rule ID")
		return
	}

	rule, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if rule == nil {
		response.NotFound(c, "Rule not found")
		return
	}

	response.Success(c, rule)
}

// Create 创建规则
// POST /api/v1/admin/account-throttle-rules
func (h *AccountThrottleHandler) Create(c *gin.Context) {
	var req CreateAccountThrottleRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	rule := &model.AccountThrottleRule{
		Name:     req.Name,
		Priority: req.Priority,
		Keywords: req.Keywords,
	}

	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	} else {
		rule.Enabled = true
	}
	if req.MatchMode != "" {
		rule.MatchMode = req.MatchMode
	} else {
		rule.MatchMode = model.ThrottleMatchContains
	}
	if req.TriggerMode != "" {
		rule.TriggerMode = req.TriggerMode
	} else {
		rule.TriggerMode = model.ThrottleTriggerImmediate
	}
	if req.AccumulatedCount != nil {
		rule.AccumulatedCount = *req.AccumulatedCount
	} else {
		rule.AccumulatedCount = 3
	}
	if req.AccumulatedWindow != nil {
		rule.AccumulatedWindow = *req.AccumulatedWindow
	} else {
		rule.AccumulatedWindow = 60
	}
	if req.ActionType != "" {
		rule.ActionType = req.ActionType
	} else {
		rule.ActionType = model.ThrottleActionDuration
	}
	if req.ActionDuration != nil {
		rule.ActionDuration = *req.ActionDuration
	} else {
		rule.ActionDuration = 300
	}
	if req.ActionRecoverHour != nil {
		rule.ActionRecoverHour = *req.ActionRecoverHour
	}
	if req.Platforms != nil {
		rule.Platforms = req.Platforms
	}
	rule.Description = req.Description

	if rule.Keywords == nil {
		rule.Keywords = []string{}
	}
	if rule.Platforms == nil {
		rule.Platforms = []string{}
	}

	created, err := h.service.Create(c.Request.Context(), rule)
	if err != nil {
		if _, ok := err.(*model.ValidationError); ok {
			response.BadRequest(c, err.Error())
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, created)
}

// Update 更新规则（支持部分更新）
// PUT /api/v1/admin/account-throttle-rules/:id
func (h *AccountThrottleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid rule ID")
		return
	}

	var req UpdateAccountThrottleRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	existing, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "Rule not found")
		return
	}

	rule := &model.AccountThrottleRule{
		ID:                id,
		Name:              existing.Name,
		Enabled:           existing.Enabled,
		Priority:          existing.Priority,
		Keywords:          existing.Keywords,
		MatchMode:         existing.MatchMode,
		TriggerMode:       existing.TriggerMode,
		AccumulatedCount:  existing.AccumulatedCount,
		AccumulatedWindow: existing.AccumulatedWindow,
		ActionType:        existing.ActionType,
		ActionDuration:    existing.ActionDuration,
		ActionRecoverHour: existing.ActionRecoverHour,
		Platforms:         existing.Platforms,
		Description:       existing.Description,
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Keywords != nil {
		rule.Keywords = req.Keywords
	}
	if req.MatchMode != nil {
		rule.MatchMode = *req.MatchMode
	}
	if req.TriggerMode != nil {
		rule.TriggerMode = *req.TriggerMode
	}
	if req.AccumulatedCount != nil {
		rule.AccumulatedCount = *req.AccumulatedCount
	}
	if req.AccumulatedWindow != nil {
		rule.AccumulatedWindow = *req.AccumulatedWindow
	}
	if req.ActionType != nil {
		rule.ActionType = *req.ActionType
	}
	if req.ActionDuration != nil {
		rule.ActionDuration = *req.ActionDuration
	}
	if req.ActionRecoverHour != nil {
		rule.ActionRecoverHour = *req.ActionRecoverHour
	}
	if req.Platforms != nil {
		rule.Platforms = req.Platforms
	}
	if req.Description != nil {
		rule.Description = req.Description
	}

	if rule.Keywords == nil {
		rule.Keywords = []string{}
	}
	if rule.Platforms == nil {
		rule.Platforms = []string{}
	}

	updated, err := h.service.Update(c.Request.Context(), rule)
	if err != nil {
		if _, ok := err.(*model.ValidationError); ok {
			response.BadRequest(c, err.Error())
			return
		}
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, updated)
}

// Delete 删除规则
// DELETE /api/v1/admin/account-throttle-rules/:id
func (h *AccountThrottleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid rule ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{"message": "Rule deleted successfully"})
}
