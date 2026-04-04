package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupAPIKeyRequestQuotaHandler(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewAdminAPIKeyHandler(adminSvc)
	router.PUT("/api/v1/admin/api-keys/:id/request-quota", h.UpdateRequestQuota)
	return router
}

func TestAdminAPIKeyHandler_UpdateRequestQuota(t *testing.T) {
	router := setupAPIKeyRequestQuotaHandler(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10/request-quota", bytes.NewBufferString(`{"request_quota":15,"reset_request_quota_used":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			ID               int64 `json:"id"`
			RequestQuota     int64 `json:"request_quota"`
			RequestQuotaUsed int64 `json:"request_quota_used"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, int64(10), resp.Data.ID)
	require.Equal(t, int64(15), resp.Data.RequestQuota)
	require.Equal(t, int64(0), resp.Data.RequestQuotaUsed)
}

func TestAdminAPIKeyHandler_UpdateRequestQuota_InvalidJSON(t *testing.T) {
	router := setupAPIKeyRequestQuotaHandler(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/api-keys/10/request-quota", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid request")
}
