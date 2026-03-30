package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayHandleFailoverExhausted_429BillingIssueMapsToBillingError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	h := &GatewayHandler{}
	h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
		StatusCode: http.StatusTooManyRequests,
		ResponseBody: []byte(
			`{"error":{"code":"1113","message":"余额不足或无可用资源包,请充值。"},"request_id":"req_test"}`,
		),
	}, service.PlatformAnthropic, false)

	require.Equal(t, http.StatusForbidden, w.Code)

	var payload struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
	require.Equal(t, "error", payload.Type)
	require.Equal(t, "billing_error", payload.Error.Type)
	require.Contains(t, payload.Error.Message, "余额不足")
}
