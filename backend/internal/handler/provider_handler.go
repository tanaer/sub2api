package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	providerChannel      = "aiapi.muskpay.top"
	providerClaudeOpus   = "claude-opus-4.6"
	providerClaudeSonnet = "claude-sonnet-4.6"
	providerClaudeHaiku  = "claude-haiku-4-5-20251001"
)

// ProviderHandler 处理 CLI provider 解析请求
type ProviderHandler struct {
	apiKeyService  *service.APIKeyService
	settingService *service.SettingService
}

// NewProviderHandler 创建 ProviderHandler
func NewProviderHandler(apiKeyService *service.APIKeyService, settingService *service.SettingService) *ProviderHandler {
	return &ProviderHandler{
		apiKeyService:  apiKeyService,
		settingService: settingService,
	}
}

type providerResolveRequest struct {
	APIKey  string `json:"apiKey"`
	Channel string `json:"channel"`
}

type providerModelTarget struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

type providerGroupResponse struct {
	ID               string                         `json:"id"`
	Name             string                         `json:"name"`
	Description      string                         `json:"description,omitempty"`
	AnthropicBaseURL string                         `json:"anthropic_base_url"`
	OpenCodeBaseURL  string                         `json:"opencode_base_url,omitempty"`
	ModelMapping     map[string]providerModelTarget `json:"model_mapping"`
}

type providerErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type providerResolveResponse struct {
	Version int                    `json:"version"`
	Channel string                 `json:"channel"`
	Matched bool                   `json:"matched"`
	Group   *providerGroupResponse `json:"group,omitempty"`
	Error   *providerErrorDetail   `json:"error,omitempty"`
}

// Resolve 根据 API Key 返回 CLI provider 分组配置
// POST /api/provider
func (h *ProviderHandler) Resolve(c *gin.Context) {
	apiKeyString := h.extractAPIKey(c)
	if apiKeyString == "" {
		c.JSON(http.StatusOK, providerResolveResponse{
			Version: 1,
			Channel: providerChannel,
			Matched: false,
			Error:   &providerErrorDetail{Code: "API_KEY_REQUIRED", Message: "API key is required"},
		})
		return
	}

	apiKey, err := h.apiKeyService.GetByKey(c.Request.Context(), apiKeyString)
	if err != nil {
		if errors.Is(err, service.ErrAPIKeyNotFound) {
			c.JSON(http.StatusOK, providerResolveResponse{
				Version: 1,
				Channel: providerChannel,
				Matched: false,
				Error:   &providerErrorDetail{Code: "GROUP_NOT_FOUND", Message: "No provider group matched for this API key"},
			})
			return
		}
		c.JSON(http.StatusOK, providerResolveResponse{
			Version: 1,
			Channel: providerChannel,
			Matched: false,
			Error:   &providerErrorDetail{Code: "UNAVAILABLE", Message: "Provider service temporarily unavailable"},
		})
		return
	}

	if apiKey.Group == nil {
		c.JSON(http.StatusOK, providerResolveResponse{
			Version: 1,
			Channel: providerChannel,
			Matched: false,
			Error:   &providerErrorDetail{Code: "GROUP_NOT_FOUND", Message: "No provider group matched for this API key"},
		})
		return
	}

	baseURL := h.resolveBaseURL(c)
	group := apiKey.Group

	c.JSON(http.StatusOK, providerResolveResponse{
		Version: 1,
		Channel: providerChannel,
		Matched: true,
		Group: &providerGroupResponse{
			ID:               strconv.FormatInt(group.ID, 10),
			Name:             group.Name,
			Description:      group.Description,
			AnthropicBaseURL: baseURL,
			OpenCodeBaseURL:  baseURL + "/v1",
			ModelMapping: map[string]providerModelTarget{
				providerClaudeOpus:   providerResolveModel(group, providerClaudeOpus),
				providerClaudeSonnet: providerResolveModel(group, providerClaudeSonnet),
				providerClaudeHaiku:  providerResolveModel(group, providerClaudeHaiku),
			},
		},
	})
}

// extractAPIKey 从请求体或 Authorization 头提取 API Key
func (h *ProviderHandler) extractAPIKey(c *gin.Context) string {
	// 优先从 JSON 请求体取
	var req providerResolveRequest
	if c.Request.Body != nil {
		_ = json.NewDecoder(c.Request.Body).Decode(&req)
	}
	if k := strings.TrimSpace(req.APIKey); k != "" {
		return k
	}
	// 从 Authorization: Bearer 取
	auth := c.GetHeader("Authorization")
	if parts := strings.SplitN(auth, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

// resolveBaseURL 从设置中获取 API 基础地址（不含 /v1 后缀）
func (h *ProviderHandler) resolveBaseURL(c *gin.Context) string {
	if settings, err := h.settingService.GetPublicSettings(c.Request.Context()); err == nil && settings.APIBaseURL != "" {
		base := strings.TrimRight(settings.APIBaseURL, "/")
		base = strings.TrimSuffix(base, "/v1")
		return base
	}
	// 回退：从请求 Host 推断
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + c.Request.Host
}

// providerResolveModel 将 Claude 模型 ID 通过分组别名映射解析为目标模型
func providerResolveModel(group *service.Group, claudeModelID string) providerModelTarget {
	resolved := group.ResolveModelAlias(claudeModelID)
	return providerModelTarget{ID: resolved, DisplayName: resolved}
}
