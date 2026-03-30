//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandlerUsageQuotaLimited_IncludesRequestQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := &GatewayHandler{}
	h.usageQuotaLimited(c, context.Background(), &service.APIKey{
		Status:           service.StatusActive,
		RequestQuota:     12,
		RequestQuotaUsed: 5,
	}, nil, nil)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "quota_limited", body["mode"])

	requestQuota, ok := body["request_quota"].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, 12, requestQuota["limit"], 0.000001)
	require.InDelta(t, 5, requestQuota["used"], 0.000001)
	require.InDelta(t, 7, requestQuota["remaining"], 0.000001)
	require.Equal(t, "requests", requestQuota["unit"])
}
