package service

import (
	"context"
	"strings"
	"sync"
	"time"
)

type opsRequestTraceStateContextKey struct{}

type OpsRequestTraceState struct {
	mu    sync.Mutex
	trace *OpsRequestTrace
}

func NewOpsRequestTraceState(startedAt time.Time) *OpsRequestTraceState {
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	return &OpsRequestTraceState{
		trace: &OpsRequestTrace{
			CreatedAt: startedAt.UTC(),
		},
	}
}

func WithOpsRequestTraceState(ctx context.Context, state *OpsRequestTraceState) context.Context {
	if state == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, opsRequestTraceStateContextKey{}, state)
}

func OpsRequestTraceStateFromContext(ctx context.Context) *OpsRequestTraceState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(opsRequestTraceStateContextKey{}).(*OpsRequestTraceState)
	return state
}

func (s *OpsRequestTraceState) Update(fn func(trace *OpsRequestTrace)) {
	if s == nil || fn == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.trace == nil {
		s.trace = &OpsRequestTrace{CreatedAt: time.Now().UTC()}
	}
	fn(s.trace)
	if s.trace.CreatedAt.IsZero() {
		s.trace.CreatedAt = time.Now().UTC()
	}
}

func (s *OpsRequestTraceState) AppendEvent(eventType string, occurredAt time.Time, data map[string]any) {
	if s == nil {
		return
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	s.Update(func(trace *OpsRequestTrace) {
		trace.TraceEvents = append(trace.TraceEvents, &OpsRequestTraceEvent{
			Type:       eventType,
			OccurredAt: occurredAt.UTC(),
			Data:       cloneTraceEventData(data),
		})
	})
}

func AppendOpsRequestTraceEvent(ctx context.Context, eventType string, occurredAt time.Time, data map[string]any) bool {
	state := OpsRequestTraceStateFromContext(ctx)
	if state == nil {
		return false
	}
	state.AppendEvent(eventType, occurredAt, data)
	return true
}

func (s *OpsRequestTraceState) Snapshot() *OpsRequestTrace {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneOpsRequestTrace(s.trace)
}

func cloneOpsRequestTrace(trace *OpsRequestTrace) *OpsRequestTrace {
	if trace == nil {
		return nil
	}
	out := *trace
	if trace.FinalStatusCode != nil {
		value := *trace.FinalStatusCode
		out.FinalStatusCode = &value
	}
	if trace.UserID != nil {
		value := *trace.UserID
		out.UserID = &value
	}
	if trace.APIKeyID != nil {
		value := *trace.APIKeyID
		out.APIKeyID = &value
	}
	if trace.GroupID != nil {
		value := *trace.GroupID
		out.GroupID = &value
	}
	if trace.FinalAccountID != nil {
		value := *trace.FinalAccountID
		out.FinalAccountID = &value
	}
	if trace.FinishedAt != nil {
		value := trace.FinishedAt.UTC()
		out.FinishedAt = &value
	}
	if trace.DurationMs != nil {
		value := *trace.DurationMs
		out.DurationMs = &value
	}
	out.UpstreamRequestIDs = append([]string(nil), trace.UpstreamRequestIDs...)
	if len(trace.TraceEvents) > 0 {
		out.TraceEvents = make([]*OpsRequestTraceEvent, 0, len(trace.TraceEvents))
		for _, event := range trace.TraceEvents {
			out.TraceEvents = append(out.TraceEvents, cloneOpsRequestTraceEvent(event))
		}
	}
	return &out
}

func cloneOpsRequestTraceEvent(event *OpsRequestTraceEvent) *OpsRequestTraceEvent {
	if event == nil {
		return nil
	}
	out := *event
	out.Data = cloneTraceEventData(event.Data)
	return &out
}

func cloneTraceEventData(data map[string]any) map[string]any {
	if len(data) == 0 {
		return nil
	}
	out := make(map[string]any, len(data))
	for key, value := range data {
		out[key] = value
	}
	return out
}
