package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const leakingUpstreamModelMessage = "Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'"

func assertClientFacingModelLeakRedacted(t *testing.T, body []byte, messagePath string) {
	t.Helper()
	text := string(body)
	require.NotContains(t, text, "GLM-Z1-AirX")
	require.NotContains(t, text, "qwen2-coder-next")
	require.Equal(t, clientFacingUnavailableModelMessage, strings.TrimSpace(gjson.GetBytes(body, messagePath).String()))
}

func TestGatewayHandleErrorResponse_BillingErrorRedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GatewayService{failoverPolicy: noFailoverPolicy()}
	respBody := []byte(`{"error":{"code":"1113","message":"Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'. Please recharge."}}`)
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 31, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "billing_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestOpenAIHandleErrorResponse_BillingErrorRedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &OpenAIGatewayService{failoverPolicy: noFailoverPolicy()}
	respBody := []byte(`{"error":{"code":"1113","message":"Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'. Please recharge."}}`)
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     http.Header{},
	}
	account := &Account{ID: 32, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}

	_, err := svc.handleErrorResponse(context.Background(), resp, c, account, nil)
	require.Error(t, err)
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Equal(t, "billing_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestWriteAnthropicError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeAnthropicError(c, http.StatusBadRequest, "invalid_request_error", leakingUpstreamModelMessage)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestWriteChatCompletionsError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeChatCompletionsError(c, http.StatusBadRequest, "invalid_request_error", leakingUpstreamModelMessage)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestWriteGatewayCCError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeGatewayCCError(c, http.StatusBadRequest, "server_error", leakingUpstreamModelMessage)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestGeminiMessagesCompatWriteErrors_RedactUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &GeminiMessagesCompatService{}

	t.Run("claude_format", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		err := svc.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", leakingUpstreamModelMessage)
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
	})

	t.Run("google_format", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		err := svc.writeGoogleError(c, http.StatusBadRequest, leakingUpstreamModelMessage)
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
	})
}

func TestGeminiWriteGeminiMappedError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &GeminiMessagesCompatService{}
	account := &Account{ID: 33, Platform: PlatformGemini, Type: AccountTypeAPIKey}
	respBody := []byte(`{"error":{"code":400,"message":"Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'","status":"INVALID_ARGUMENT"}}`)

	err := svc.writeGeminiMappedError(c, account, http.StatusBadRequest, "req-redact", respBody)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestAntigravityWriteErrors_RedactUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &AntigravityGatewayService{}

	t.Run("claude_format", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		err := svc.writeClaudeError(c, http.StatusBadRequest, "invalid_request_error", leakingUpstreamModelMessage)
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
	})

	t.Run("google_format", func(t *testing.T) {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)

		err := svc.writeGoogleError(c, http.StatusBadRequest, leakingUpstreamModelMessage)
		require.Error(t, err)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
	})
}

func TestAntigravityWriteMappedClaudeError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &AntigravityGatewayService{}
	account := &Account{ID: 34, Platform: PlatformAntigravity}
	body := []byte(`{"error":{"message":"Model GLM-Z1-AirX not supported. Available: 'qwen2-coder-next', 'glm-5'"}}`)

	err := svc.writeMappedClaudeError(c, account, http.StatusBadRequest, "req-antigravity", body)
	require.Error(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "invalid_request_error", gjson.GetBytes(rec.Body.Bytes(), "error.type").String())
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestOpenAIWSFallbackErrorResponse_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &OpenAIGatewayService{}
	ok := svc.writeOpenAIWSFallbackErrorResponse(c, &Account{ID: 35, Platform: PlatformOpenAI}, &openAIWSFallbackError{
		Reason: "auth_failed",
		Err: &openAIWSDialError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New(leakingUpstreamModelMessage),
		},
	})

	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestSoraWriteSoraError_RedactsUpstreamModelExposure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	svc := &SoraGatewayService{}
	svc.writeSoraError(c, http.StatusBadRequest, "upstream_error", leakingUpstreamModelMessage, false)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assertClientFacingModelLeakRedacted(t, rec.Body.Bytes(), "error.message")
}

func TestSoraBuildErrorPayload_RedactsUpstreamModelExposure(t *testing.T) {
	svc := &SoraGatewayService{}
	payload := svc.buildErrorPayload(
		[]byte(`{"error":{"type":"upstream_error","message":"`+leakingUpstreamModelMessage+`"}}`),
		leakingUpstreamModelMessage,
	)

	body, err := json.Marshal(payload)
	require.NoError(t, err)
	assertClientFacingModelLeakRedacted(t, body, "error.message")
}
