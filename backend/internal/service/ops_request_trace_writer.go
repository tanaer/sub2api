package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type OpsRequestTraceWriter struct {
	opsRepo OpsRepository
	queue   chan *OpsRequestTrace

	batchFlushTimeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	started atomic.Bool

	droppedCount uint64
	writeFailed  uint64
}

func NewOpsRequestTraceWriter(opsRepo OpsRepository, capacity int) *OpsRequestTraceWriter {
	if capacity <= 0 {
		capacity = 256
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &OpsRequestTraceWriter{
		opsRepo:           opsRepo,
		queue:             make(chan *OpsRequestTrace, capacity),
		batchFlushTimeout: 5 * time.Second,
		ctx:               ctx,
		cancel:            cancel,
	}
}

func (w *OpsRequestTraceWriter) Start() {
	if w == nil || w.opsRepo == nil {
		return
	}
	if !w.started.CompareAndSwap(false, true) {
		return
	}
	w.wg.Add(1)
	go w.run()
}

func (w *OpsRequestTraceWriter) Stop() {
	if w == nil {
		return
	}
	if w.started.Load() && w.cancel != nil {
		w.cancel()
	}
	w.wg.Wait()
}

func (w *OpsRequestTraceWriter) Enqueue(trace *OpsRequestTrace) bool {
	if w == nil || trace == nil {
		return false
	}
	select {
	case w.queue <- trace:
		return true
	default:
		atomic.AddUint64(&w.droppedCount, 1)
		log.Printf("ops_trace_dropped client_request_id=%s", trace.ClientRequestID)
		return false
	}
}

func (w *OpsRequestTraceWriter) DroppedCount() uint64 {
	if w == nil {
		return 0
	}
	return atomic.LoadUint64(&w.droppedCount)
}

func (w *OpsRequestTraceWriter) run() {
	defer w.wg.Done()
	drain := func() {
		for {
			select {
			case trace := <-w.queue:
				if trace == nil {
					continue
				}
				if err := w.writeWithRetry(context.Background(), trace); err != nil {
					log.Printf("ops_trace_flush_failed client_request_id=%s err=%v", trace.ClientRequestID, err)
				}
			default:
				return
			}
		}
	}

	for {
		select {
		case <-w.ctx.Done():
			drain()
			return
		case trace := <-w.queue:
			if trace == nil {
				continue
			}
			if err := w.writeWithRetry(w.ctx, trace); err != nil {
				log.Printf("ops_trace_flush_failed client_request_id=%s err=%v", trace.ClientRequestID, err)
			}
		}
	}
}

func (w *OpsRequestTraceWriter) writeWithRetry(baseCtx context.Context, trace *OpsRequestTrace) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		ctx := baseCtx
		if ctx == nil || ctx.Err() != nil {
			ctx = context.Background()
		}
		flushCtx, cancel := context.WithTimeout(ctx, w.batchFlushTimeout)
		err := w.opsRepo.UpsertRequestTrace(flushCtx, trace)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		atomic.AddUint64(&w.writeFailed, 1)
	}
	return lastErr
}
