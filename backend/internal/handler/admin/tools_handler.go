package admin

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type workbenchAPIKeyReader interface {
	GetByKey(ctx context.Context, key string) (*service.APIKey, error)
}

type workbenchAPIKeyUsageReader interface {
	GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*usagestats.UserDashboardStats, error)
}

type workbenchUserRedeemReader interface {
	GetByCode(ctx context.Context, code string) (*service.RedeemCode, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error)
}

type workbenchAdminService interface {
	GenerateRedeemCodes(ctx context.Context, input *service.GenerateRedeemCodesInput) ([]service.RedeemCode, error)
	GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int) ([]service.APIKey, int64, error)
}

type lookupAPIKeyRequest struct {
	RawText string `json:"raw_text" binding:"required"`
}

type lookupUserAPIKeyItem struct {
	ID               int64      `json:"id"`
	Key              string     `json:"key"`
	Status           string     `json:"status"`
	CreatedAt        *time.Time `json:"created_at,omitempty"`
	SuccessCallCount int64      `json:"success_call_count"`
}

type lookupAPIKeyItem struct {
	ExtractedKey     string                 `json:"extracted_key"`
	Matched          bool                   `json:"matched"`
	APIKey           string                 `json:"api_key,omitempty"`
	APIKeyID         int64                  `json:"api_key_id,omitempty"`
	UserID           int64                  `json:"user_id,omitempty"`
	UserEmail        string                 `json:"user_email,omitempty"`
	Username         string                 `json:"username,omitempty"`
	UserStatus       string                 `json:"user_status,omitempty"`
	GroupID          *int64                 `json:"group_id,omitempty"`
	LatestRedeemAt   *time.Time             `json:"latest_redeem_at,omitempty"`
	SuccessCallCount int64                  `json:"success_call_count"`
	APIKeys          []lookupUserAPIKeyItem `json:"api_keys"`
}

type lookupAPIKeyResponse struct {
	ExtractedKeys  []string           `json:"extracted_keys"`
	MatchedCount   int                `json:"matched_count"`
	UnmatchedCount int                `json:"unmatched_count"`
	Items          []lookupAPIKeyItem `json:"items"`
}

type updateWorkbenchRedeemPresetsRequest []service.WorkbenchRedeemPreset
type updateWorkbenchRedeemTemplatesRequest []service.WorkbenchRedeemTemplate

type generateWorkbenchRedeemPresetResponse struct {
	Code            string                        `json:"code"`
	RenderedMessage string                        `json:"rendered_message"`
	RedeemCode      *service.RedeemCode           `json:"redeem_code,omitempty"`
	Preset          service.WorkbenchRedeemPreset `json:"preset"`
}

// ToolsHandler handles admin workbench tools.
type ToolsHandler struct {
	apiKeys        workbenchAPIKeyReader
	usageStats     workbenchAPIKeyUsageReader
	redeemHistory  workbenchUserRedeemReader
	settingService *service.SettingService
	adminService   workbenchAdminService
}

var (
	workbenchLabeledKeyPattern  = regexp.MustCompile(`(?im)(?:api\s*key|apikey|密钥|key)\s*[：:]\s*([A-Za-z0-9_-]{16,128})`)
	workbenchSKKeyPattern       = regexp.MustCompile(`sk-[A-Za-z0-9_-]{16,128}`)
	workbenchHexKeyPattern      = regexp.MustCompile(`[a-fA-F0-9]{32,128}`)
	workbenchTemplateVarPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)
)

// NewToolsHandler creates a new admin tools handler.
func NewToolsHandler(
	apiKeys workbenchAPIKeyReader,
	usageStats workbenchAPIKeyUsageReader,
	redeemHistory workbenchUserRedeemReader,
	settingService *service.SettingService,
	adminService workbenchAdminService,
) *ToolsHandler {
	return &ToolsHandler{
		apiKeys:        apiKeys,
		usageStats:     usageStats,
		redeemHistory:  redeemHistory,
		settingService: settingService,
		adminService:   adminService,
	}
}

// LookupAPIKeys extracts API keys from raw text and returns aggregated lookup results.
func (h *ToolsHandler) LookupAPIKeys(c *gin.Context) {
	var req lookupAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	extractedKeys := extractWorkbenchAPIKeys(req.RawText)
	items := make([]lookupAPIKeyItem, 0, len(extractedKeys))
	matchedCount := 0

	for _, extractedKey := range extractedKeys {
		item, matched, err := h.lookupSingleAPIKey(c, extractedKey)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		if matched {
			matchedCount++
		}
		items = append(items, item)
	}

	response.Success(c, lookupAPIKeyResponse{
		ExtractedKeys:  extractedKeys,
		MatchedCount:   matchedCount,
		UnmatchedCount: len(extractedKeys) - matchedCount,
		Items:          items,
	})
}

// GetRedeemPresets returns shared workbench redeem presets.
func (h *ToolsHandler) GetRedeemPresets(c *gin.Context) {
	presets, err := h.settingService.GetWorkbenchRedeemPresets(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, presets)
}

// UpdateRedeemPresets updates shared workbench redeem presets.
func (h *ToolsHandler) UpdateRedeemPresets(c *gin.Context) {
	var req updateWorkbenchRedeemPresetsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	presets, err := normalizeWorkbenchRedeemPresets([]service.WorkbenchRedeemPreset(req))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.settingService.SetWorkbenchRedeemPresets(c.Request.Context(), presets); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, presets)
}

// GetRedeemTemplates returns shared workbench message templates.
func (h *ToolsHandler) GetRedeemTemplates(c *gin.Context) {
	templates, err := h.settingService.GetWorkbenchRedeemTemplates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, templates)
}

// UpdateRedeemTemplates updates shared workbench message templates.
func (h *ToolsHandler) UpdateRedeemTemplates(c *gin.Context) {
	var req updateWorkbenchRedeemTemplatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	templates, err := normalizeWorkbenchRedeemTemplates([]service.WorkbenchRedeemTemplate(req))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.settingService.SetWorkbenchRedeemTemplates(c.Request.Context(), templates); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, templates)
}

// GenerateRedeemPreset generates a redeem code from a saved preset and renders its message template.
func (h *ToolsHandler) GenerateRedeemPreset(c *gin.Context) {
	presetID := strings.TrimSpace(c.Param("id"))
	if presetID == "" {
		response.BadRequest(c, "Invalid preset ID")
		return
	}

	presets, err := h.settingService.GetWorkbenchRedeemPresets(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	preset, ok := findWorkbenchRedeemPreset(presets, presetID)
	if !ok {
		response.NotFound(c, "Redeem preset not found")
		return
	}
	if !preset.Enabled {
		response.BadRequest(c, "Redeem preset is disabled")
		return
	}

	codes, err := h.adminService.GenerateRedeemCodes(c.Request.Context(), &service.GenerateRedeemCodesInput{
		Count:        1,
		Type:         preset.Type,
		Value:        preset.Value,
		GroupID:      preset.GroupID,
		ValidityDays: preset.ValidityDays,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if len(codes) == 0 {
		response.InternalError(c, "Failed to generate redeem code")
		return
	}

	templateContent, err := h.resolveWorkbenchTemplateContent(c.Request.Context(), preset)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	renderedMessage := renderWorkbenchTemplate(templateContent, codes[0].Code)
	response.Success(c, generateWorkbenchRedeemPresetResponse{
		Code:            codes[0].Code,
		RenderedMessage: renderedMessage,
		RedeemCode:      &codes[0],
		Preset:          preset,
	})
}

func (h *ToolsHandler) lookupSingleAPIKey(c *gin.Context, extractedKey string) (lookupAPIKeyItem, bool, error) {
	item := lookupAPIKeyItem{
		ExtractedKey: extractedKey,
		APIKeys:      []lookupUserAPIKeyItem{},
	}

	apiKey, err := lookupWorkbenchAPIKey(c, h.apiKeys, extractedKey)
	if err != nil {
		if !errors.Is(err, service.ErrAPIKeyNotFound) {
			return item, false, err
		}
		redeemCode, redeemErr := lookupWorkbenchRedeemCode(c, h.redeemHistory, extractedKey)
		if redeemErr != nil {
			if errors.Is(redeemErr, service.ErrRedeemCodeNotFound) {
				return item, false, nil
			}
			return item, false, redeemErr
		}
		return h.fillLookupItemFromRedeemCode(c, item, redeemCode)
	}

	return h.fillLookupItemFromAPIKey(c, item, apiKey)
}

func (h *ToolsHandler) fillLookupItemFromAPIKey(c *gin.Context, item lookupAPIKeyItem, apiKey *service.APIKey) (lookupAPIKeyItem, bool, error) {
	item.Matched = true
	item.APIKey = apiKey.Key
	item.APIKeyID = apiKey.ID
	item.UserID = apiKey.UserID
	item.GroupID = apiKey.GroupID
	if apiKey.User != nil {
		item.UserEmail = apiKey.User.Email
		item.Username = apiKey.User.Username
		item.UserStatus = apiKey.User.Status
	}

	if apiKey.UserID > 0 {
		redeems, err := h.redeemHistory.ListByUser(c.Request.Context(), apiKey.UserID, 1)
		if err != nil {
			return item, false, err
		}
		if len(redeems) > 0 && redeems[0].UsedAt != nil {
			item.LatestRedeemAt = redeems[0].UsedAt
		}
	}
	return h.attachWorkbenchUserAPIKeys(c, item)
}

func (h *ToolsHandler) fillLookupItemFromRedeemCode(c *gin.Context, item lookupAPIKeyItem, redeemCode *service.RedeemCode) (lookupAPIKeyItem, bool, error) {
	item.Matched = true
	item.APIKey = redeemCode.Code
	if redeemCode.UsedBy != nil {
		item.UserID = *redeemCode.UsedBy
	}
	if redeemCode.User != nil {
		item.UserEmail = redeemCode.User.Email
		item.Username = redeemCode.User.Username
		item.UserStatus = redeemCode.User.Status
	}
	if redeemCode.GroupID != nil {
		item.GroupID = redeemCode.GroupID
	}
	if redeemCode.UsedAt != nil {
		item.LatestRedeemAt = redeemCode.UsedAt
	}

	return h.attachWorkbenchUserAPIKeys(c, item)
}

func (h *ToolsHandler) attachWorkbenchUserAPIKeys(c *gin.Context, item lookupAPIKeyItem) (lookupAPIKeyItem, bool, error) {
	userKeys, err := h.listWorkbenchUserAPIKeys(c, item.UserID)
	if err != nil {
		return item, false, err
	}

	sortWorkbenchAPIKeys(userKeys)
	item.APIKeys = make([]lookupUserAPIKeyItem, 0, len(userKeys))
	item.SuccessCallCount = 0
	for i := range userKeys {
		stats, err := h.usageStats.GetAPIKeyDashboardStats(c.Request.Context(), userKeys[i].ID)
		if err != nil {
			return item, false, err
		}
		successCallCount := int64(0)
		if stats != nil {
			successCallCount = stats.TotalRequests
		}
		item.SuccessCallCount += successCallCount
		item.APIKeys = append(item.APIKeys, lookupUserAPIKeyItem{
			ID:               userKeys[i].ID,
			Key:              userKeys[i].Key,
			Status:           userKeys[i].Status,
			CreatedAt:        &userKeys[i].CreatedAt,
			SuccessCallCount: successCallCount,
		})
	}

	return item, true, nil
}

func normalizeWorkbenchRedeemPresets(presets []service.WorkbenchRedeemPreset) ([]service.WorkbenchRedeemPreset, error) {
	normalized := make([]service.WorkbenchRedeemPreset, 0, len(presets))
	seenIDs := make(map[string]struct{}, len(presets))

	for i := range presets {
		item := presets[i]
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.TemplateID = strings.TrimSpace(item.TemplateID)
		item.Template = normalizeWorkbenchOptionalTemplate(item.Template)
		item.Type = strings.TrimSpace(item.Type)

		if item.ID == "" {
			return nil, errors.New("preset id is required")
		}
		if _, ok := seenIDs[item.ID]; ok {
			return nil, errors.New("preset id must be unique")
		}
		seenIDs[item.ID] = struct{}{}
		if item.Name == "" {
			return nil, errors.New("preset name is required")
		}
		if item.TemplateID == "" && item.Template == "" {
			return nil, errors.New("preset template_id is required")
		}
		if item.Template != "" {
			if err := validateWorkbenchTemplate(item.Template); err != nil {
				return nil, err
			}
		}

		switch item.Type {
		case service.RedeemTypeBalance, service.RedeemTypeConcurrency:
			if item.Value <= 0 {
				return nil, errors.New("preset value must be greater than 0")
			}
			item.GroupID = nil
		case service.RedeemTypeInvitation:
			item.Value = 0
			item.GroupID = nil
			item.ValidityDays = 0
		case service.RedeemTypeSubscription:
			if item.GroupID == nil || *item.GroupID <= 0 {
				return nil, errors.New("subscription preset requires group_id")
			}
			if item.ValidityDays <= 0 {
				item.ValidityDays = 30
			}
			item.Value = 0
		case service.RedeemTypeGroupRequestQuota:
			if item.GroupID == nil || *item.GroupID <= 0 {
				return nil, errors.New("group_request_quota preset requires group_id")
			}
			if item.Value <= 0 || math.Trunc(item.Value) != item.Value {
				return nil, errors.New("group_request_quota preset value must be a positive integer")
			}
			if item.ValidityDays <= 0 {
				item.ValidityDays = 30
			}
		default:
			return nil, errors.New("preset type is invalid")
		}

		normalized = append(normalized, item)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].SortOrder != normalized[j].SortOrder {
			return normalized[i].SortOrder < normalized[j].SortOrder
		}
		return normalized[i].ID < normalized[j].ID
	})

	return normalized, nil
}

func normalizeWorkbenchRedeemTemplates(templates []service.WorkbenchRedeemTemplate) ([]service.WorkbenchRedeemTemplate, error) {
	normalized := make([]service.WorkbenchRedeemTemplate, 0, len(templates))
	seenIDs := make(map[string]struct{}, len(templates))

	for i := range templates {
		item := templates[i]
		item.ID = strings.TrimSpace(item.ID)
		item.Name = strings.TrimSpace(item.Name)
		item.Content = normalizeWorkbenchTemplate(item.Content)

		if item.ID == "" {
			return nil, errors.New("template id is required")
		}
		if _, ok := seenIDs[item.ID]; ok {
			return nil, errors.New("template id must be unique")
		}
		seenIDs[item.ID] = struct{}{}
		if item.Name == "" {
			return nil, errors.New("template name is required")
		}
		if err := validateWorkbenchTemplate(item.Content); err != nil {
			return nil, err
		}

		normalized = append(normalized, item)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].SortOrder != normalized[j].SortOrder {
			return normalized[i].SortOrder < normalized[j].SortOrder
		}
		return normalized[i].ID < normalized[j].ID
	})

	return normalized, nil
}

func findWorkbenchRedeemPreset(presets []service.WorkbenchRedeemPreset, presetID string) (service.WorkbenchRedeemPreset, bool) {
	for i := range presets {
		if presets[i].ID == presetID {
			return presets[i], true
		}
	}
	return service.WorkbenchRedeemPreset{}, false
}

func findWorkbenchRedeemTemplate(templates []service.WorkbenchRedeemTemplate, templateID string) (service.WorkbenchRedeemTemplate, bool) {
	for i := range templates {
		if templates[i].ID == templateID {
			return templates[i], true
		}
	}
	return service.WorkbenchRedeemTemplate{}, false
}

func renderWorkbenchTemplate(template string, code string) string {
	return strings.ReplaceAll(normalizeWorkbenchTemplate(template), "{{code}}", code)
}

func (h *ToolsHandler) resolveWorkbenchTemplateContent(ctx context.Context, preset service.WorkbenchRedeemPreset) (string, error) {
	if preset.TemplateID != "" {
		templates, err := h.settingService.GetWorkbenchRedeemTemplates(ctx)
		if err != nil {
			return "", err
		}
		template, ok := findWorkbenchRedeemTemplate(templates, preset.TemplateID)
		if !ok {
			return "", errors.New("redeem template not found")
		}
		return template.Content, nil
	}

	if preset.Template != "" {
		return preset.Template, nil
	}

	return "{{code}}", nil
}

func normalizeWorkbenchTemplate(template string) string {
	template = strings.ReplaceAll(template, "\r\n", "\n")
	template = strings.ReplaceAll(template, "\r", "\n")
	template = strings.TrimSpace(template)
	if template == "" {
		return "{{code}}"
	}
	return template
}

func normalizeWorkbenchOptionalTemplate(template string) string {
	template = strings.ReplaceAll(template, "\r\n", "\n")
	template = strings.ReplaceAll(template, "\r", "\n")
	return strings.TrimSpace(template)
}

func validateWorkbenchTemplate(template string) error {
	for _, token := range workbenchTemplateVarPattern.FindAllString(template, -1) {
		if token != "{{code}}" {
			return errors.New("template only supports {{code}}")
		}
	}
	return nil
}

type workbenchExtractedKey struct {
	Start int
	Value string
}

func extractWorkbenchAPIKeys(rawText string) []string {
	matches := make([]workbenchExtractedKey, 0)

	for _, indexes := range workbenchLabeledKeyPattern.FindAllStringSubmatchIndex(rawText, -1) {
		if len(indexes) >= 4 {
			matches = append(matches, workbenchExtractedKey{
				Start: indexes[2],
				Value: rawText[indexes[2]:indexes[3]],
			})
		}
	}
	for _, indexes := range workbenchSKKeyPattern.FindAllStringIndex(rawText, -1) {
		matches = append(matches, workbenchExtractedKey{
			Start: indexes[0],
			Value: rawText[indexes[0]:indexes[1]],
		})
	}
	for _, indexes := range workbenchHexKeyPattern.FindAllStringIndex(rawText, -1) {
		matches = append(matches, workbenchExtractedKey{
			Start: indexes[0],
			Value: rawText[indexes[0]:indexes[1]],
		})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Start != matches[j].Start {
			return matches[i].Start < matches[j].Start
		}
		return matches[i].Value < matches[j].Value
	})

	keys := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		value := strings.TrimSpace(match.Value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		keys = append(keys, value)
	}
	return keys
}

func lookupWorkbenchAPIKey(c *gin.Context, reader workbenchAPIKeyReader, extractedKey string) (*service.APIKey, error) {
	candidates := []string{strings.TrimSpace(extractedKey)}
	lowered := normalizeWorkbenchLookupCandidate(extractedKey)
	if lowered != "" && lowered != candidates[0] {
		candidates = append(candidates, lowered)
	}

	var lastErr error
	for _, candidate := range candidates {
		apiKey, err := reader.GetByKey(c.Request.Context(), candidate)
		if err == nil {
			return apiKey, nil
		}
		lastErr = err
		if !errors.Is(err, service.ErrAPIKeyNotFound) {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = service.ErrAPIKeyNotFound
	}
	return nil, lastErr
}

func lookupWorkbenchRedeemCode(c *gin.Context, reader workbenchUserRedeemReader, extractedKey string) (*service.RedeemCode, error) {
	candidates := []string{strings.TrimSpace(extractedKey)}
	lowered := normalizeWorkbenchLookupCandidate(extractedKey)
	if lowered != "" && lowered != candidates[0] {
		candidates = append(candidates, lowered)
	}

	var lastErr error
	for _, candidate := range candidates {
		redeemCode, err := reader.GetByCode(c.Request.Context(), candidate)
		if err == nil {
			return redeemCode, nil
		}
		lastErr = err
		if !errors.Is(err, service.ErrRedeemCodeNotFound) {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = service.ErrRedeemCodeNotFound
	}
	return nil, lastErr
}

func (h *ToolsHandler) listWorkbenchUserAPIKeys(c *gin.Context, userID int64) ([]service.APIKey, error) {
	if userID <= 0 {
		return []service.APIKey{}, nil
	}
	keys, _, err := h.adminService.GetUserAPIKeys(c.Request.Context(), userID, 1, 1000)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func sortWorkbenchAPIKeys(keys []service.APIKey) {
	sort.SliceStable(keys, func(i, j int) bool {
		if !keys[i].CreatedAt.Equal(keys[j].CreatedAt) {
			return keys[i].CreatedAt.Before(keys[j].CreatedAt)
		}
		return keys[i].ID < keys[j].ID
	})
}

func normalizeWorkbenchLookupCandidate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if isWorkbenchHexToken(value) {
		return strings.ToLower(value)
	}
	if strings.HasPrefix(strings.ToLower(value), "sk-") {
		prefix := value[:3]
		suffix := value[3:]
		if isWorkbenchHexToken(suffix) {
			return strings.ToLower(prefix + suffix)
		}
	}
	return ""
}

func isWorkbenchHexToken(value string) bool {
	match := workbenchHexKeyPattern.FindString(value)
	return match == value
}
