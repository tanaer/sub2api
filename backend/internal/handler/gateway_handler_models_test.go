//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakeSettingRepo implements service.SettingRepository for testing.
type fakeSettingRepo struct {
	values map[string]string
}

func (f *fakeSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	return nil, nil
}

func (f *fakeSettingRepo) GetValue(ctx context.Context, key string) (string, error) {
	if v, ok := f.values[key]; ok {
		return v, nil
	}
	return "", nil
}

func (f *fakeSettingRepo) Set(ctx context.Context, key, value string) error {
	return nil
}

func (f *fakeSettingRepo) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, k := range keys {
		if v, ok := f.values[k]; ok {
			result[k] = v
		}
	}
	return result, nil
}

func (f *fakeSettingRepo) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}

func (f *fakeSettingRepo) GetAll(ctx context.Context) (map[string]string, error) {
	return f.values, nil
}

func (f *fakeSettingRepo) Delete(ctx context.Context, key string) error {
	return nil
}

// TestModels_CustomModelList verifies that when custom models are configured in
// settings, the Models() handler returns them instead of querying accounts.
func TestModels_CustomModelList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build a real SettingService with our fake repo that returns custom models.
	repo := &fakeSettingRepo{values: map[string]string{
		service.SettingKeyCustomModelList: `["model-a","model-b"]`,
	}}
	settingSvc := service.NewSettingService(repo, &config.Config{})

	// Build a minimal GatewayHandler.  gatewayService is left nil; the custom
	// models branch returns before Models() reaches GetAvailableModels(), so no
	// panic occurs.  cfg is required because the Sora branch dereferences it.
	h := &GatewayHandler{
		settingService: settingSvc,
		cfg:            &config.Config{},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	h.Models(c)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "list", resp["object"])

	data, ok := resp["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 2)

	first, ok := data[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "model-a", first["id"])
	require.Equal(t, "model", first["type"])
	require.Equal(t, "model-a", first["display_name"])
	require.Equal(t, "2024-01-01T00:00:00Z", first["created_at"])

	second, ok := data[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "model-b", second["id"])
}
