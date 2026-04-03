package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpsRequestTraceWriter_SingleWriteSuccess(t *testing.T) {
	done := make(chan struct{}, 1)

	repo := &opsRepoMock{
		UpsertRequestTraceFn: func(ctx context.Context, trace *OpsRequestTrace) error {
			select {
			case done <- struct{}{}:
			default:
			}
			return nil
		},
	}

	writer := NewOpsRequestTraceWriter(repo, 4)
	writer.batchFlushTimeout = 10 * time.Millisecond
	writer.Start()
	defer writer.Stop()

	ok := writer.Enqueue(&OpsRequestTrace{
		ClientRequestID: "trace-success-1",
		CreatedAt:       time.Now().UTC(),
	})
	require.True(t, ok)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for trace write")
	}
}

func TestOpsRequestTraceWriter_RetriesUpToThreeTimes(t *testing.T) {
	var attempts atomic.Int32
	repo := &opsRepoMock{
		UpsertRequestTraceFn: func(ctx context.Context, trace *OpsRequestTrace) error {
			attempts.Add(1)
			return errors.New("db unavailable")
		},
	}

	writer := NewOpsRequestTraceWriter(repo, 2)
	trace := &OpsRequestTrace{
		ClientRequestID: "trace-retry-1",
		CreatedAt:       time.Now().UTC(),
	}

	err := writer.writeWithRetry(context.Background(), trace)
	require.Error(t, err)
	require.Equal(t, int32(3), attempts.Load())
}

func TestOpsRequestTraceWriter_DropsWhenQueueFull(t *testing.T) {
	writer := NewOpsRequestTraceWriter(&opsRepoMock{}, 1)
	require.True(t, writer.Enqueue(&OpsRequestTrace{ClientRequestID: "trace-drop-1"}))
	require.False(t, writer.Enqueue(&OpsRequestTrace{ClientRequestID: "trace-drop-2"}))
	require.Equal(t, uint64(1), writer.DroppedCount())
}

func TestOpsServiceFinalizeAndEnqueueRequestTrace_EnqueuesOnceAndMarksIncomplete(t *testing.T) {
	done := make(chan *OpsRequestTrace, 1)
	repo := &opsRepoMock{
		UpsertRequestTraceFn: func(ctx context.Context, trace *OpsRequestTrace) error {
			select {
			case done <- trace:
			default:
			}
			return nil
		},
	}

	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	trace := &OpsRequestTrace{
		ClientRequestID: "trace-finalize-1",
		CreatedAt:       time.Now().UTC(),
		TraceEvents: []*OpsRequestTraceEvent{
			{Type: "request_received", OccurredAt: time.Now().UTC()},
		},
	}

	first := svc.FinalizeAndEnqueueRequestTrace(context.Background(), trace)
	second := svc.FinalizeAndEnqueueRequestTrace(context.Background(), trace)
	require.True(t, first)
	require.False(t, second)

	select {
	case written := <-done:
		require.True(t, written.TraceIncomplete)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for finalized trace write")
	}
}
