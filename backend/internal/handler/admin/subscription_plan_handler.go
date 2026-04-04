package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionPlanHandler struct {
	repo service.SubscriptionPlanRepository
}

func NewSubscriptionPlanHandler(repo service.SubscriptionPlanRepository) *SubscriptionPlanHandler {
	return &SubscriptionPlanHandler{repo: repo}
}

type CreateSubscriptionPlanRequest struct {
	Name            string   `json:"name" binding:"required,max=100"`
	Description     string   `json:"description"`
	GroupID         *int64   `json:"group_id"`
	BillingMode     string   `json:"billing_mode" binding:"required,oneof=per_request per_usd"`
	RequestQuota    int64    `json:"request_quota"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	ValidityDays    int      `json:"validity_days" binding:"required,min=1,max=36500"`
}

type UpdateSubscriptionPlanRequest struct {
	Name            string   `json:"name" binding:"omitempty,max=100"`
	Description     *string  `json:"description"`
	GroupID         *int64   `json:"group_id"`
	BillingMode     string   `json:"billing_mode" binding:"omitempty,oneof=per_request per_usd"`
	RequestQuota    *int64   `json:"request_quota"`
	DailyLimitUSD   *float64 `json:"daily_limit_usd"`
	WeeklyLimitUSD  *float64 `json:"weekly_limit_usd"`
	MonthlyLimitUSD *float64 `json:"monthly_limit_usd"`
	ValidityDays    *int     `json:"validity_days" binding:"omitempty,min=1,max=36500"`
	Status          string   `json:"status" binding:"omitempty,oneof=active archived"`
}

// List — GET /api/v1/admin/subscription-plans
func (h *SubscriptionPlanHandler) List(c *gin.Context) {
	status := c.Query("status")
	plans, err := h.repo.List(c.Request.Context(), status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, plans)
}

// Create — POST /api/v1/admin/subscription-plans
func (h *SubscriptionPlanHandler) Create(c *gin.Context) {
	var req CreateSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	plan := &service.SubscriptionPlan{
		Name:         req.Name,
		Description:  req.Description,
		GroupID:      req.GroupID,
		BillingMode:  req.BillingMode,
		RequestQuota: req.RequestQuota,
		ValidityDays: req.ValidityDays,
		Status:       service.SubscriptionPlanStatusActive,
	}
	if req.DailyLimitUSD != nil {
		plan.DailyLimitUSD = *req.DailyLimitUSD
	}
	if req.WeeklyLimitUSD != nil {
		plan.WeeklyLimitUSD = *req.WeeklyLimitUSD
	}
	if req.MonthlyLimitUSD != nil {
		plan.MonthlyLimitUSD = *req.MonthlyLimitUSD
	}

	if err := h.repo.Create(c.Request.Context(), plan); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Re-fetch with relations
	created, err := h.repo.GetByID(c.Request.Context(), plan.ID)
	if err != nil {
		response.Success(c, plan)
		return
	}
	response.Success(c, created)
}

// Update — PUT /api/v1/admin/subscription-plans/:id
func (h *SubscriptionPlanHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	existing, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	var req UpdateSubscriptionPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.GroupID != nil {
		existing.GroupID = req.GroupID
	}
	if req.BillingMode != "" {
		existing.BillingMode = req.BillingMode
	}
	if req.RequestQuota != nil {
		existing.RequestQuota = *req.RequestQuota
	}
	if req.DailyLimitUSD != nil {
		existing.DailyLimitUSD = *req.DailyLimitUSD
	}
	if req.WeeklyLimitUSD != nil {
		existing.WeeklyLimitUSD = *req.WeeklyLimitUSD
	}
	if req.MonthlyLimitUSD != nil {
		existing.MonthlyLimitUSD = *req.MonthlyLimitUSD
	}
	if req.ValidityDays != nil {
		existing.ValidityDays = *req.ValidityDays
	}
	if req.Status != "" {
		existing.Status = req.Status
	}

	if err := h.repo.Update(c.Request.Context(), existing); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	updated, _ := h.repo.GetByID(c.Request.Context(), id)
	response.Success(c, updated)
}

// Delete — DELETE /api/v1/admin/subscription-plans/:id
func (h *SubscriptionPlanHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}
