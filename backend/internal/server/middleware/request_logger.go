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

const requestIDHeader = "X-Request-ID"

// RequestLogger 在请求入口注入 request-scoped logger。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request == nil {
			c.Next()
			return
		}

		requestID := strings.TrimSpace(c.GetHeader(requestIDHeader))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Header(requestIDHeader, requestID)

		ctx := context.WithValue(c.Request.Context(), ctxkey.RequestID, requestID)
		clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string)

		requestLogger := logger.With(
			zap.String("component", "http"),
			zap.String("request_id", requestID),
			zap.String("client_request_id", strings.TrimSpace(clientRequestID)),
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
		)

		traceState := service.OpsRequestTraceStateFromContext(ctx)
		if traceState == nil {
			traceState = service.NewOpsRequestTraceState(time.Now().UTC())
		}
		traceState.Update(func(trace *service.OpsRequestTrace) {
			trace.LocalRequestID = requestID
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
