package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type requestTraceRepoStub struct {
	service.OpsRepository
	upsertFn func(ctx context.Context, trace *service.OpsRequestTrace) error
}

func (s *requestTraceRepoStub) UpsertRequestTrace(ctx context.Context, trace *service.OpsRequestTrace) error {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, trace)
	}
	return nil
}

func TestOpsErrorLoggerMiddleware_FinalizesRequestTrace(t *testing.T) {
	resetOpsErrorLoggerStateForTest(t)
	gin.SetMode(gin.TestMode)

	done := make(chan *service.OpsRequestTrace, 1)
	ops := service.NewOpsService(&requestTraceRepoStub{
		upsertFn: func(ctx context.Context, trace *service.OpsRequestTrace) error {
			select {
			case done <- trace:
			default:
			}
			return nil
		},
	}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	r := gin.New()
	r.Use(middleware2.RequestLogger())
	r.Use(middleware2.ClientRequestID())
	r.Use(OpsErrorLoggerMiddleware(ops))
	r.POST("/v1/messages", func(c *gin.Context) {
		startRequestTraceFromGin(c, "/v1/messages", "sonnet", false)
		recordGroupResolved(c, "sonnet", "glm-4.5-air", "group_alias")
		setOpsSelectedAccount(c, 9, service.PlatformAnthropic)
		c.Status(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	select {
	case trace := <-done:
		require.NotEmpty(t, trace.ClientRequestID)
		require.NotEmpty(t, trace.LocalRequestID)
		require.Equal(t, "sonnet", trace.OriginalRequestedModel)
		require.Equal(t, "glm-4.5-air", trace.GroupResolvedModel)
		require.Equal(t, "success", trace.Status)
		require.NotNil(t, trace.FinalStatusCode)
		require.Equal(t, http.StatusOK, *trace.FinalStatusCode)
		require.False(t, trace.TraceIncomplete)
		require.NotNil(t, trace.FinalAccountID)
		require.EqualValues(t, 9, *trace.FinalAccountID)
		require.GreaterOrEqual(t, len(trace.TraceEvents), 2)
		require.Equal(t, "request_received", trace.TraceEvents[0].Type)
		require.Equal(t, "request_finished", trace.TraceEvents[len(trace.TraceEvents)-1].Type)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for request trace")
	}
}
