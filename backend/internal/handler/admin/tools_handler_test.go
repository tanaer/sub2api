package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type toolsAPIKeyLookupStub struct {
	byKey map[string]*service.APIKey
}

func (s *toolsAPIKeyLookupStub) GetByKey(ctx context.Context, key string) (*service.APIKey, error) {
	item, ok := s.byKey[key]
	if !ok {
		return nil, service.ErrAPIKeyNotFound
	}
	cloned := *item
	return &cloned, nil
}

type toolsUsageStatsStub struct {
	byAPIKeyID map[int64]int64
}

func (s *toolsUsageStatsStub) GetAPIKeyDashboardStats(ctx context.Context, apiKeyID int64) (*usagestats.UserDashboardStats, error) {
	return &usagestats.UserDashboardStats{
		TotalRequests: s.byAPIKeyID[apiKeyID],
	}, nil
}

type toolsRedeemHistoryStub struct {
	byCode map[string]*service.RedeemCode
	byUser map[int64][]service.RedeemCode
}

func (s *toolsRedeemHistoryStub) GetByCode(ctx context.Context, code string) (*service.RedeemCode, error) {
	item, ok := s.byCode[code]
	if !ok {
		return nil, service.ErrRedeemCodeNotFound
	}
	cloned := *item
	return &cloned, nil
}

func (s *toolsRedeemHistoryStub) ListByUser(ctx context.Context, userID int64, limit int) ([]service.RedeemCode, error) {
	items := s.byUser[userID]
	if len(items) > limit {
		items = items[:limit]
	}
	cloned := make([]service.RedeemCode, len(items))
	copy(cloned, items)
	return cloned, nil
}

type toolsRedeemGeneratorStub struct {
	lastInput    *service.GenerateRedeemCodesInput
	codes        []service.RedeemCode
	apiKeysByUser map[int64][]service.APIKey
}

func (s *toolsRedeemGeneratorStub) GenerateRedeemCodes(ctx context.Context, input *service.GenerateRedeemCodesInput) ([]service.RedeemCode, error) {
	if input != nil {
		copied := *input
		s.lastInput = &copied
	}
	cloned := make([]service.RedeemCode, len(s.codes))
	copy(cloned, s.codes)
	return cloned, nil
}

func (s *toolsRedeemGeneratorStub) GetUserAPIKeys(ctx context.Context, userID int64, page, pageSize int) ([]service.APIKey, int64, error) {
	items := s.apiKeysByUser[userID]
	cloned := make([]service.APIKey, len(items))
	copy(cloned, items)
	return cloned, int64(len(cloned)), nil
}

type toolsSettingRepoStub struct {
	values map[string]string
}

func (s *toolsSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	panic("unexpected Get call")
}

func (s *toolsSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *toolsSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *toolsSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	panic("unexpected GetMultiple call")
}

func (s *toolsSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	panic("unexpected SetMultiple call")
}

func (s *toolsSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	panic("unexpected GetAll call")
}

func (s *toolsSettingRepoStub) Delete(ctx context.Context, key string) error {
	panic("unexpected Delete call")
}

func setupToolsHandlerRouter(handler *ToolsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tools := router.Group("/api/v1/admin/tools")
	{
		tools.POST("/api-key-lookup", handler.LookupAPIKeys)
		tools.GET("/redeem-presets", handler.GetRedeemPresets)
		tools.PUT("/redeem-presets", handler.UpdateRedeemPresets)
		tools.GET("/redeem-templates", handler.GetRedeemTemplates)
		tools.PUT("/redeem-templates", handler.UpdateRedeemTemplates)
		tools.POST("/redeem-presets/:id/generate", handler.GenerateRedeemPreset)
	}
	return router
}

func TestToolsHandler_LookupAPIKeys_ExtractsAllMatchedAndUnmatchedKeys(t *testing.T) {
	createdAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	redeemedAt := time.Date(2026, 3, 25, 9, 30, 0, 0, time.UTC)
	groupID := int64(22)

	apiKey := &service.APIKey{
		ID:        101,
		UserID:    7,
		Key:       "58e06d0d1c63a738489e49ce4f2a425a",
		Status:    service.StatusActive,
		GroupID:   &groupID,
		CreatedAt: createdAt,
		User: &service.User{
			ID:       7,
			Email:    "lookup@example.com",
			Username: "lookup-user",
			Status:   service.StatusActive,
		},
		Group: &service.Group{
			ID:       groupID,
			Name:     "智谱",
			Platform: service.PlatformAnthropic,
			Status:   service.StatusActive,
		},
	}

	settingService := service.NewSettingService(&toolsSettingRepoStub{}, &config.Config{})
	handler := NewToolsHandler(
		&toolsAPIKeyLookupStub{byKey: map[string]*service.APIKey{
			apiKey.Key: apiKey,
		}},
		&toolsUsageStatsStub{byAPIKeyID: map[int64]int64{
			101: 12,
			102: 4,
		}},
		&toolsRedeemHistoryStub{byUser: map[int64][]service.RedeemCode{
			7: {
				{ID: 88, Code: "R-001", Status: service.StatusUsed, UsedAt: &redeemedAt},
			},
		}},
		settingService,
		&toolsRedeemGeneratorStub{
			apiKeysByUser: map[int64][]service.APIKey{
				7: {
					{
						ID:        101,
						UserID:    7,
						Key:       apiKey.Key,
						Status:    service.StatusActive,
						GroupID:   &groupID,
						CreatedAt: createdAt,
					},
					{
						ID:        102,
						UserID:    7,
						Key:       "sk-second-abcdef1234567890",
						Status:    service.StatusDisabled,
						GroupID:   &groupID,
						CreatedAt: createdAt.Add(5 * time.Minute),
					},
				},
			},
		},
	)
	router := setupToolsHandlerRouter(handler)

	body := map[string]string{
		"raw_text": "密钥：58e06d0d1c63a738489e49ce4f2a425a\n备用：sk-unmatched-1234567890abcdef",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools/api-key-lookup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ExtractedKeys  []string `json:"extracted_keys"`
			MatchedCount   int      `json:"matched_count"`
			UnmatchedCount int      `json:"unmatched_count"`
			Items          []struct {
				ExtractedKey     string `json:"extracted_key"`
				Matched          bool   `json:"matched"`
				UserEmail        string `json:"user_email"`
				Username         string `json:"username"`
				UserStatus       string `json:"user_status"`
				SuccessCallCount int64  `json:"success_call_count"`
				LatestRedeemAt   string `json:"latest_redeem_at"`
				APIKeys          []struct {
					ID               int64  `json:"id"`
					Key              string `json:"key"`
					CreatedAt        string `json:"created_at"`
					SuccessCallCount int64  `json:"success_call_count"`
					Status string `json:"status"`
				} `json:"api_keys"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotContains(t, rec.Body.String(), "\"group_name\"")
	require.NotContains(t, rec.Body.String(), "\"user_account\"")
	require.NotContains(t, rec.Body.String(), "\"api_key_created_at\"")
	require.Equal(t, []string{
		"58e06d0d1c63a738489e49ce4f2a425a",
		"sk-unmatched-1234567890abcdef",
	}, resp.Data.ExtractedKeys)
	require.Equal(t, 1, resp.Data.MatchedCount)
	require.Equal(t, 1, resp.Data.UnmatchedCount)
	require.Len(t, resp.Data.Items, 2)

	require.True(t, resp.Data.Items[0].Matched)
	require.Equal(t, "lookup@example.com", resp.Data.Items[0].UserEmail)
	require.Equal(t, "lookup-user", resp.Data.Items[0].Username)
	require.Equal(t, service.StatusActive, resp.Data.Items[0].UserStatus)
	require.Equal(t, int64(16), resp.Data.Items[0].SuccessCallCount)
	require.Equal(t, redeemedAt.Format(time.RFC3339), resp.Data.Items[0].LatestRedeemAt)
	require.Len(t, resp.Data.Items[0].APIKeys, 2)
	require.Equal(t, apiKey.Key, resp.Data.Items[0].APIKeys[0].Key)
	require.Equal(t, createdAt.Format(time.RFC3339), resp.Data.Items[0].APIKeys[0].CreatedAt)
	require.Equal(t, int64(12), resp.Data.Items[0].APIKeys[0].SuccessCallCount)
	require.Equal(t, service.StatusActive, resp.Data.Items[0].APIKeys[0].Status)
	require.Equal(t, "sk-second-abcdef1234567890", resp.Data.Items[0].APIKeys[1].Key)
	require.Equal(t, int64(4), resp.Data.Items[0].APIKeys[1].SuccessCallCount)

	require.False(t, resp.Data.Items[1].Matched)
	require.Equal(t, "sk-unmatched-1234567890abcdef", resp.Data.Items[1].ExtractedKey)
}

func TestToolsHandler_LookupAPIKeys_UnboundGroupStillReturnsUserAPIKeys(t *testing.T) {
	apiKey := &service.APIKey{
		ID:        201,
		UserID:    8,
		Key:       "sk-demo-unbound-1234567890abcdef",
		Status:    service.StatusActive,
		CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		User: &service.User{
			ID:       8,
			Email:    "nogroup@example.com",
			Username: "nogroup-user",
			Status:   service.StatusActive,
		},
	}

	settingService := service.NewSettingService(&toolsSettingRepoStub{}, &config.Config{})
	handler := NewToolsHandler(
		&toolsAPIKeyLookupStub{byKey: map[string]*service.APIKey{apiKey.Key: apiKey}},
		&toolsUsageStatsStub{},
		&toolsRedeemHistoryStub{},
		settingService,
		&toolsRedeemGeneratorStub{
			apiKeysByUser: map[int64][]service.APIKey{
				8: {
					{
						ID:        201,
						UserID:    8,
						Key:       apiKey.Key,
						Status:    service.StatusActive,
						CreatedAt: apiKey.CreatedAt,
					},
				},
			},
		},
	)
	router := setupToolsHandlerRouter(handler)

	payload := []byte(`{"raw_text":"sk-demo-unbound-1234567890abcdef"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools/api-key-lookup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			Items []struct {
				UserStatus string `json:"user_status"`
				APIKeys    []struct {
					ID int64 `json:"id"`
				} `json:"api_keys"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, service.StatusActive, resp.Data.Items[0].UserStatus)
	require.Len(t, resp.Data.Items[0].APIKeys, 1)
	require.Equal(t, int64(201), resp.Data.Items[0].APIKeys[0].ID)
}

func TestToolsHandler_LookupAPIKeys_FallsBackToRedeemCodeLookup(t *testing.T) {
	redeemedAt := time.Date(2026, 3, 31, 6, 3, 34, 0, time.UTC)
	apiKeyCreatedAt := redeemedAt.Add(2 * time.Minute)
	oldAPIKeyCreatedAt := redeemedAt.Add(-10 * time.Minute)
	groupID := int64(2)
	userID := int64(18)

	redeemCode := &service.RedeemCode{
		ID:        33,
		Code:      "ba44ab182fc862fa76e42590bbc11d97",
		Type:      service.RedeemTypeGroupRequestQuota,
		Value:     5000,
		Status:    service.StatusUsed,
		UsedBy:    &userID,
		UsedAt:    &redeemedAt,
		CreatedAt: redeemedAt.Add(-time.Hour),
		GroupID:   &groupID,
		User: &service.User{
			ID:       userID,
			Email:    "328867132@qq.com",
			Username: "",
			Status:   service.StatusActive,
		},
		Group: &service.Group{
			ID:       groupID,
			Name:     "智谱",
			Platform: service.PlatformAnthropic,
			Status:   service.StatusActive,
		},
	}

	settingService := service.NewSettingService(&toolsSettingRepoStub{}, &config.Config{})
	handler := NewToolsHandler(
		&toolsAPIKeyLookupStub{},
		&toolsUsageStatsStub{byAPIKeyID: map[int64]int64{
			23: 99,
			24: 1,
		}},
		&toolsRedeemHistoryStub{
			byCode: map[string]*service.RedeemCode{
				redeemCode.Code: redeemCode,
			},
			byUser: map[int64][]service.RedeemCode{
				userID: {*redeemCode},
			},
		},
		settingService,
		&toolsRedeemGeneratorStub{
			apiKeysByUser: map[int64][]service.APIKey{
				userID: {
					{ID: 23, UserID: userID, Key: "sk-old", GroupID: &groupID, CreatedAt: oldAPIKeyCreatedAt, Status: service.StatusActive},
					{ID: 24, UserID: userID, Key: "sk-new", GroupID: &groupID, CreatedAt: apiKeyCreatedAt, Status: service.StatusActive},
				},
			},
		},
	)
	router := setupToolsHandlerRouter(handler)

	payload := []byte(`{"raw_text":"ba44ab182fc862fa76e42590bbc11d97"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools/api-key-lookup", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotContains(t, rec.Body.String(), "\"group_name\"")
	require.NotContains(t, rec.Body.String(), "\"user_account\"")
	require.NotContains(t, rec.Body.String(), "\"api_key_created_at\"")

	var resp struct {
		Code int `json:"code"`
		Data struct {
			MatchedCount   int `json:"matched_count"`
			UnmatchedCount int `json:"unmatched_count"`
			Items          []struct {
				Matched          bool   `json:"matched"`
				UserEmail        string `json:"user_email"`
				UserStatus       string `json:"user_status"`
				LatestRedeemAt   string `json:"latest_redeem_at"`
				SuccessCallCount int64  `json:"success_call_count"`
				APIKeys          []struct {
					ID               int64  `json:"id"`
					Key              string `json:"key"`
					CreatedAt        string `json:"created_at"`
					SuccessCallCount int64  `json:"success_call_count"`
				} `json:"api_keys"`
			} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, resp.Data.MatchedCount)
	require.Equal(t, 0, resp.Data.UnmatchedCount)
	require.Len(t, resp.Data.Items, 1)
	require.True(t, resp.Data.Items[0].Matched)
	require.Equal(t, "328867132@qq.com", resp.Data.Items[0].UserEmail)
	require.Equal(t, service.StatusActive, resp.Data.Items[0].UserStatus)
	require.Equal(t, redeemedAt.Format(time.RFC3339), resp.Data.Items[0].LatestRedeemAt)
	require.Equal(t, int64(100), resp.Data.Items[0].SuccessCallCount)
	require.Len(t, resp.Data.Items[0].APIKeys, 2)
	require.Equal(t, int64(23), resp.Data.Items[0].APIKeys[0].ID)
	require.Equal(t, oldAPIKeyCreatedAt.Format(time.RFC3339), resp.Data.Items[0].APIKeys[0].CreatedAt)
	require.Equal(t, int64(99), resp.Data.Items[0].APIKeys[0].SuccessCallCount)
	require.Equal(t, int64(24), resp.Data.Items[0].APIKeys[1].ID)
	require.Equal(t, apiKeyCreatedAt.Format(time.RFC3339), resp.Data.Items[0].APIKeys[1].CreatedAt)
	require.Equal(t, int64(1), resp.Data.Items[0].APIKeys[1].SuccessCallCount)
}

func TestToolsHandler_GenerateRedeemPreset_RendersTemplateAndUsesPresetRules(t *testing.T) {
	settingRepo := &toolsSettingRepoStub{}
	settingService := service.NewSettingService(settingRepo, &config.Config{})
	require.NoError(t, settingService.SetWorkbenchRedeemPresets(context.Background(), []service.WorkbenchRedeemPreset{
		{
			ID:           "balance-50",
			Name:         "50额度",
			Enabled:      true,
			SortOrder:    1,
			Type:         service.RedeemTypeBalance,
			Value:        50,
			TemplateID:   "template-guide",
			ValidityDays: 30,
		},
	}))
	require.NoError(t, settingService.SetWorkbenchRedeemTemplates(context.Background(), []service.WorkbenchRedeemTemplate{
		{
			ID:        "template-guide",
			Name:      "默认指引",
			Enabled:   true,
			SortOrder: 1,
			Content:   "密钥：{{code}}\n请尽快兑换",
		},
	}))

	generator := &toolsRedeemGeneratorStub{
		codes: []service.RedeemCode{
			{
				ID:        301,
				Code:      "R-TEST-001",
				Type:      service.RedeemTypeBalance,
				Value:     50,
				Status:    service.StatusUnused,
				CreatedAt: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
			},
		},
	}
	handler := NewToolsHandler(
		&toolsAPIKeyLookupStub{},
		&toolsUsageStatsStub{},
		&toolsRedeemHistoryStub{},
		settingService,
		generator,
	)
	router := setupToolsHandlerRouter(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/tools/redeem-presets/balance-50/generate", http.NoBody)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, generator.lastInput)
	require.Equal(t, 1, generator.lastInput.Count)
	require.Equal(t, service.RedeemTypeBalance, generator.lastInput.Type)
	require.Equal(t, 50.0, generator.lastInput.Value)

	var resp struct {
		Data struct {
			Code            string `json:"code"`
			RenderedMessage string `json:"rendered_message"`
			Preset          struct {
				ID string `json:"id"`
			} `json:"preset"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "R-TEST-001", resp.Data.Code)
	require.Equal(t, "密钥：R-TEST-001\n请尽快兑换", resp.Data.RenderedMessage)
	require.Equal(t, "balance-50", resp.Data.Preset.ID)
}
