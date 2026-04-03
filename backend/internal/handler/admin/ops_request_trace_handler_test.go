package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type requestTraceRepoCapture struct {
	service.OpsRepository
	listFn func(ctx context.Context, key string, keyType string, limit int) ([]*service.OpsRequestTrace, error)
}

func (r *requestTraceRepoCapture) ListRequestTracesByKey(ctx context.Context, key string, keyType string, limit int) ([]*service.OpsRequestTrace, error) {
	if r.listFn != nil {
		return r.listFn(ctx, key, keyType, limit)
	}
	return nil, nil
}

func newTraceHandler(t *testing.T, repo service.OpsRepository) *OpsHandler {
	t.Helper()
	return NewOpsHandler(service.NewOpsService(
		repo,
		newTestSettingRepo(),
		&config.Config{Ops: config.OpsConfig{Enabled: true}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	))
}

func TestOpsHandler_GetRequestTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing key returns 400", func(t *testing.T) {
		handler := newTraceHandler(t, &requestTraceRepoCapture{})
		router := gin.New()
		router.GET("/request-trace", handler.GetRequestTrace)

		req := httptest.NewRequest(http.MethodGet, "/request-trace", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid key type returns 400", func(t *testing.T) {
		handler := newTraceHandler(t, &requestTraceRepoCapture{})
		router := gin.New()
		router.GET("/request-trace", handler.GetRequestTrace)

		req := httptest.NewRequest(http.MethodGet, "/request-trace?key=req-1&key_type=bad", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found returns 404 contract", func(t *testing.T) {
		handler := newTraceHandler(t, &requestTraceRepoCapture{})
		router := gin.New()
		router.GET("/request-trace", handler.GetRequestTrace)

		req := httptest.NewRequest(http.MethodGet, "/request-trace?key=req-missing", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		errorBody, ok := body["error"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "trace_not_found", errorBody["code"])
	})

	t.Run("ambiguous returns 409 contract", func(t *testing.T) {
		repo := &requestTraceRepoCapture{
			listFn: func(ctx context.Context, key string, keyType string, limit int) ([]*service.OpsRequestTrace, error) {
				base := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
				return []*service.OpsRequestTrace{
					{ClientRequestID: "client-2", LocalRequestID: "shared-local", Status: "success", CreatedAt: base.Add(2 * time.Minute)},
					{ClientRequestID: "client-1", LocalRequestID: "shared-local", Status: "error", CreatedAt: base},
				}, nil
			},
		}
		handler := newTraceHandler(t, repo)
		router := gin.New()
		router.GET("/request-trace", handler.GetRequestTrace)

		req := httptest.NewRequest(http.MethodGet, "/request-trace?key=shared-local&key_type=local", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		errorBody, ok := body["error"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "ambiguous_request_identifier", errorBody["code"])
		candidates, ok := errorBody["candidates"].([]any)
		require.True(t, ok)
		require.Len(t, candidates, 2)
	})

	t.Run("success returns trace payload", func(t *testing.T) {
		finishedAt := time.Date(2026, 4, 4, 12, 1, 1, 0, time.UTC)
		statusCode := 200
		accountID := int64(123)
		trace := &service.OpsRequestTrace{
			ClientRequestID:           "client-123",
			LocalRequestID:            "local-123",
			UsageRequestID:            "usage-123",
			UpstreamRequestIDs:        []string{"up-1"},
			OriginalRequestedModel:    "sonnet",
			GroupResolvedModel:        "glm-4.5-air",
			AccountSupportLookupModel: "glm-4.5-air",
			FinalUpstreamModel:        "glm-4.5-air",
			Status:                    "success",
			FinalStatusCode:           &statusCode,
			Platform:                  "anthropic",
			RequestPath:               "/v1/messages",
			InboundEndpoint:           "messages",
			UpstreamEndpoint:          "/v1/chat/completions",
			FinalAccountID:            &accountID,
			CreatedAt:                 time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
			FinishedAt:                &finishedAt,
			TraceEvents: []*service.OpsRequestTraceEvent{
				{Type: "request_received", OccurredAt: time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC), Data: map[string]any{"phase": "ingress", "stream": false}},
				{Type: "request_finished", OccurredAt: finishedAt, Data: map[string]any{"phase": "finish", "status_code": 200}},
			},
		}
		repo := &requestTraceRepoCapture{
			listFn: func(ctx context.Context, key string, keyType string, limit int) ([]*service.OpsRequestTrace, error) {
				return []*service.OpsRequestTrace{trace}, nil
			},
		}
		handler := newTraceHandler(t, repo)
		router := gin.New()
		router.GET("/request-trace", handler.GetRequestTrace)

		req := httptest.NewRequest(http.MethodGet, "/request-trace?key=client-123", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		identity, ok := body["identity"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "client-123", identity["client_request_id"])
		require.Equal(t, "client_request_id", identity["matched_by"])
		models, ok := body["models"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "glm-4.5-air", models["group_resolved_model"])
		result, ok := body["result"].(map[string]any)
		require.True(t, ok)
		require.EqualValues(t, 123, result["final_account_id"])
		timeline, ok := body["timeline"].([]any)
		require.True(t, ok)
		require.Len(t, timeline, 2)
	})
}
