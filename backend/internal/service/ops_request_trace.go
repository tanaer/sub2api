package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *OpsService) GetRequestTrace(ctx context.Context, query *OpsRequestTraceQuery) (*OpsRequestTraceLookupResult, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	if s.opsRepo == nil {
		return &OpsRequestTraceLookupResult{}, nil
	}
	if query == nil {
		return nil, fmt.Errorf("nil trace query")
	}

	key := strings.TrimSpace(query.Key)
	if key == "" {
		return nil, fmt.Errorf("trace key is required")
	}
	keyType, err := normalizeOpsRequestTraceKeyType(query.KeyType)
	if err != nil {
		return nil, err
	}

	searchOrder := []string{keyType}
	if keyType == OpsRequestTraceKeyTypeAuto {
		searchOrder = opsRequestTraceAutoPriority
	}

	for _, currentKeyType := range searchOrder {
		traces, err := s.opsRepo.ListRequestTracesByKey(ctx, key, currentKeyType, opsRequestTraceCandidateLimit)
		if err != nil {
			return nil, err
		}
		if len(traces) == 0 {
			continue
		}

		sort.SliceStable(traces, func(i, j int) bool {
			return traces[i].CreatedAt.After(traces[j].CreatedAt)
		})
		if len(traces) == 1 {
			return &OpsRequestTraceLookupResult{
				Trace: traces[0],
			}, nil
		}
		return &OpsRequestTraceLookupResult{
			Ambiguous:  true,
			Candidates: buildOpsRequestTraceCandidates(traces),
		}, nil
	}

	return &OpsRequestTraceLookupResult{}, nil
}

func (s *OpsService) EnqueueRequestTrace(ctx context.Context, trace *OpsRequestTrace) {
	if trace == nil || !s.IsMonitoringEnabled(ctx) || s.opsRepo == nil {
		return
	}
	writer := s.ensureRequestTraceWriter()
	if writer == nil {
		return
	}
	writer.Enqueue(trace)
}

func (s *OpsService) FinalizeAndEnqueueRequestTrace(ctx context.Context, trace *OpsRequestTrace) bool {
	if trace == nil {
		return false
	}
	clientRequestID := strings.TrimSpace(trace.ClientRequestID)
	if clientRequestID == "" {
		return false
	}
	if _, loaded := s.finalizedRequestTraces.LoadOrStore(clientRequestID, struct{}{}); loaded {
		return false
	}
	time.AfterFunc(10*time.Minute, func() {
		s.finalizedRequestTraces.Delete(clientRequestID)
	})

	if !hasOpsRequestTraceEvent(trace, "request_finished") {
		trace.TraceIncomplete = true
	}
	if trace.CreatedAt.IsZero() {
		trace.CreatedAt = time.Now().UTC()
	}
	s.EnqueueRequestTrace(ctx, trace)
	return true
}

func (s *OpsService) ensureRequestTraceWriter() *OpsRequestTraceWriter {
	if s == nil || s.opsRepo == nil {
		return nil
	}
	s.requestTraceWriterOnce.Do(func() {
		s.requestTraceWriter = NewOpsRequestTraceWriter(s.opsRepo, 1024)
		s.requestTraceWriter.Start()
	})
	return s.requestTraceWriter
}

func hasOpsRequestTraceEvent(trace *OpsRequestTrace, eventType string) bool {
	for _, event := range trace.TraceEvents {
		if event == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(event.Type), eventType) {
			return true
		}
	}
	return false
}
