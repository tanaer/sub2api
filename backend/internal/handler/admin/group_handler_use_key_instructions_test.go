package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupHandler_Create_ForwardsUseKeyInstructions(t *testing.T) {
	router, adminSvc := setupAdminRouter()

	body, err := json.Marshal(map[string]any{
		"name":                 "vip-openai",
		"platform":             "openai",
		"use_key_instructions": "创建的分组需要展示这段说明",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/groups", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.createdGroup)
	require.Equal(t, "创建的分组需要展示这段说明", adminSvc.createdGroup.UseKeyInstructions)
	require.Contains(t, rec.Body.String(), "use_key_instructions")
}

func TestGroupHandler_Update_ForwardsUseKeyInstructions(t *testing.T) {
	router, adminSvc := setupAdminRouter()

	body, err := json.Marshal(map[string]any{
		"name":                 "vip-openai-updated",
		"use_key_instructions": "更新后的分组说明",
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/2", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, adminSvc.updatedGroup)
	require.Equal(t, int64(2), adminSvc.updatedGroupID)
	require.NotNil(t, adminSvc.updatedGroup.UseKeyInstructions)
	require.Equal(t, "更新后的分组说明", *adminSvc.updatedGroup.UseKeyInstructions)
	require.Contains(t, rec.Body.String(), "use_key_instructions")
}
