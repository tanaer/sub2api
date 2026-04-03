package handler

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestHandleFailoverError_AppendsTraceEvents(t *testing.T) {
	t.Run("same account retry", func(t *testing.T) {
		state := service.NewOpsRequestTraceState(time.Now().UTC())
		ctx := service.WithOpsRequestTraceState(context.Background(), state)

		fs := NewFailoverState(3, false)
		action := fs.HandleFailoverError(ctx, &mockTempUnscheduler{}, 42, service.PlatformOpenAI, newTestFailoverErr(400, true, false))

		require.Equal(t, FailoverContinue, action)

		trace := state.Snapshot()
		require.Len(t, trace.TraceEvents, 1)
		require.Equal(t, "same_account_retry", trace.TraceEvents[0].Type)
		require.EqualValues(t, 42, trace.TraceEvents[0].Data["account_id"])
		require.EqualValues(t, 1, trace.TraceEvents[0].Data["retry_count"])
	})

	t.Run("account failover", func(t *testing.T) {
		state := service.NewOpsRequestTraceState(time.Now().UTC())
		ctx := service.WithOpsRequestTraceState(context.Background(), state)

		fs := NewFailoverState(3, false)
		action := fs.HandleFailoverError(ctx, &mockTempUnscheduler{}, 88, service.PlatformOpenAI, newTestFailoverErr(503, false, false))

		require.Equal(t, FailoverContinue, action)

		trace := state.Snapshot()
		require.Len(t, trace.TraceEvents, 1)
		require.Equal(t, "account_failover", trace.TraceEvents[0].Type)
		require.EqualValues(t, 88, trace.TraceEvents[0].Data["account_id"])
		require.EqualValues(t, 1, trace.TraceEvents[0].Data["switch_count"])
	})
}
