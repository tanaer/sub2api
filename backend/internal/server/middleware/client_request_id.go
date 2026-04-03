package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ClientRequestID ensures every request has a unique client_request_id in request.Context().
//
// This is used by the Ops monitoring module for end-to-end request correlation.
func ClientRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		if v := c.Request.Context().Value(ctxkey.ClientRequestID); v != nil {
			c.Next()
			return
		}

		id := uuid.New().String()
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, id)
		requestLogger := logger.FromContext(ctx).With(zap.String("client_request_id", strings.TrimSpace(id)))
		traceState := service.OpsRequestTraceStateFromContext(ctx)
		if traceState == nil {
			traceState = service.NewOpsRequestTraceState(time.Now().UTC())
		}
		traceState.Update(func(trace *service.OpsRequestTrace) {
			trace.ClientRequestID = id
			if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
				trace.LocalRequestID = strings.TrimSpace(requestID)
			}
			if c.Request != nil && c.Request.URL != nil {
				trace.RequestPath = c.Request.URL.Path
			}
		})
		ctx = service.WithOpsRequestTraceState(ctx, traceState)
		ctx = logger.IntoContext(ctx, requestLogger)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
