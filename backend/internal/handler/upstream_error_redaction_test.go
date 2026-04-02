package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const leakingHandlerUpstreamModelMessage = "Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'"

func assertHandlerClientFacingModelLeakRedacted(t *testing.T, body []byte, messagePath string) {
	t.Helper()
	text := string(body)
	require.NotContains(t, text, "GLM-Z1-AirX")
	require.NotContains(t, text, "qwen2-coder-next")
	require.Equal(t, "Requested model is unavailable", strings.TrimSpace(gjson.GetBytes(body, messagePath).String()))
}

func TestGatewayHandleStreamingAwareError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &GatewayHandler{}
	h.handleStreamingAwareError(c, http.StatusBadRequest, "upstream_error", leakingHandlerUpstreamModelMessage, true)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "data:")
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestOpenAIHandleStreamingAwareError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &OpenAIGatewayHandler{}
	h.handleStreamingAwareError(c, http.StatusBadRequest, "upstream_error", leakingHandlerUpstreamModelMessage, false)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestOpenAIAnthropicStreamingAwareError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &OpenAIGatewayHandler{}
	h.anthropicStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", leakingHandlerUpstreamModelMessage, true)

	require.Contains(t, rec.Body.String(), "event: error")
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestGoogleError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	googleError(c, http.StatusBadRequest, leakingHandlerUpstreamModelMessage)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestWriteUpstreamResponse_ErrorRedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeUpstreamResponse(c, &service.UpstreamHTTPResult{
		StatusCode: http.StatusBadRequest,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"error":{"code":400,"message":"` + leakingHandlerUpstreamModelMessage + `","status":"INVALID_ARGUMENT"}}`),
	})

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestSoraHandleStreamingAwareError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &SoraGatewayHandler{}
	h.handleStreamingAwareError(c, http.StatusBadRequest, "upstream_error", leakingHandlerUpstreamModelMessage, false)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertHandlerClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestGatewayHandleStreamingAwareError_SSERemainsValidJSONAfterRedaction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	h := &GatewayHandler{}
	h.handleStreamingAwareError(c, http.StatusBadRequest, "upstream_error", leakingHandlerUpstreamModelMessage, true)

	lines := strings.Split(strings.TrimSpace(rec.Body.String()), "\n")
	require.Len(t, lines, 1)
	require.True(t, strings.HasPrefix(lines[0], "data: "))

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(lines[0], "data: ")), &parsed))
}
