package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestStartRequestTraceFromGin_AppendsRoutingEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)

	startedAt := time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)
	state := service.NewOpsRequestTraceState(startedAt)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request = req.WithContext(service.WithOpsRequestTraceState(context.Background(), state))

	startRequestTraceFromGin(c, "/v1/messages", "sonnet", true)
	recordGroupResolved(c, "sonnet", "glm-4.5-air", "group_alias")
	recordSelectionStarted(c, "glm-4.5-air", map[int64]struct{}{
		11: {},
		22: {},
	})

	trace := state.Snapshot()
	require.NotNil(t, trace)
	require.Equal(t, "sonnet", trace.OriginalRequestedModel)
	require.Equal(t, "glm-4.5-air", trace.GroupResolvedModel)
	require.Equal(t, "glm-4.5-air", trace.AccountSupportLookupModel)
	require.Equal(t, "/v1/messages", trace.RequestPath)
	require.Len(t, trace.TraceEvents, 3)
	require.Equal(t, "request_received", trace.TraceEvents[0].Type)
	require.Equal(t, "group_model_resolved", trace.TraceEvents[1].Type)
	require.Equal(t, "selection_started", trace.TraceEvents[2].Type)
	require.Equal(t, "selection", trace.TraceEvents[2].Data["phase"])
}
