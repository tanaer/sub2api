package handler

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const traceUpstreamRequestIDLimit = 10

func startRequestTraceFromGin(c *gin.Context, endpoint string, requestedModel string, stream bool) {
	if c == nil || c.Request == nil {
		return
	}
	requestedModel = strings.TrimSpace(requestedModel)
	state := service.OpsRequestTraceStateFromContext(c.Request.Context())
	if state == nil {
		return
	}
	state.Update(func(trace *service.OpsRequestTrace) {
		if requestedModel != "" && trace.OriginalRequestedModel == "" {
			trace.OriginalRequestedModel = requestedModel
		}
		trace.RequestPath = c.Request.URL.Path
		trace.InboundEndpoint = strings.TrimSpace(endpoint)
	})
	appendRequestTraceEvent(c.Request.Context(), "request_received", "ingress", map[string]any{
		"endpoint":        strings.TrimSpace(endpoint),
		"requested_model": requestedModel,
		"stream":          stream,
	})
}

func recordGroupResolved(c *gin.Context, fromModel, toModel, reason string) {
	if c == nil || c.Request == nil {
		return
	}
	fromModel = strings.TrimSpace(fromModel)
	toModel = strings.TrimSpace(toModel)
	if fromModel == "" || toModel == "" || fromModel == toModel {
		return
	}
	state := service.OpsRequestTraceStateFromContext(c.Request.Context())
	if state == nil {
		return
	}
	state.Update(func(trace *service.OpsRequestTrace) {
		if trace.OriginalRequestedModel == "" {
			trace.OriginalRequestedModel = fromModel
		}
		trace.GroupResolvedModel = toModel
		trace.AccountSupportLookupModel = toModel
	})
	appendRequestTraceEvent(c.Request.Context(), "group_model_resolved", "routing", map[string]any{
		"from_model": fromModel,
		"to_model":   toModel,
		"reason":     strings.TrimSpace(reason),
	})
}

func recordSelectionStarted(c *gin.Context, lookupModel string, excludedIDs map[int64]struct{}) {
	if c == nil || c.Request == nil {
		return
	}
	lookupModel = strings.TrimSpace(lookupModel)
	state := service.OpsRequestTraceStateFromContext(c.Request.Context())
	if state == nil {
		return
	}
	state.Update(func(trace *service.OpsRequestTrace) {
		if lookupModel != "" {
			trace.AccountSupportLookupModel = lookupModel
			if trace.GroupResolvedModel == "" {
				trace.GroupResolvedModel = lookupModel
			}
		}
	})
	appendRequestTraceEvent(c.Request.Context(), "selection_started", "selection", map[string]any{
		"lookup_model":         lookupModel,
		"excluded_account_ids": truncateInt64Slice(sortedInt64Keys(excludedIDs), 20),
		"excluded_total":       len(excludedIDs),
	})
}

func recordSelectionResult(c *gin.Context, selection *service.AccountSelectionResult) {
	if c == nil || c.Request == nil || selection == nil || selection.Account == nil {
		return
	}
	data := map[string]any{
		"account_id":   selection.Account.ID,
		"account_name": strings.TrimSpace(selection.Account.Name),
		"acquired":     selection.Acquired,
	}
	if selection.WaitPlan != nil {
		data["wait_account_id"] = selection.WaitPlan.AccountID
		data["wait_timeout_ms"] = selection.WaitPlan.Timeout.Milliseconds()
	}
	appendRequestTraceEvent(c.Request.Context(), "selection_result", "selection", data)
}

func appendRequestTraceEvent(ctx context.Context, eventType string, phase string, data map[string]any) {
	if strings.TrimSpace(phase) != "" {
		if data == nil {
			data = make(map[string]any, 1)
		}
		data["phase"] = strings.TrimSpace(phase)
	}
	service.AppendOpsRequestTraceEvent(ctx, eventType, time.Now().UTC(), data)
}

func buildFinalRequestTrace(c *gin.Context) *service.OpsRequestTrace {
	if c == nil || c.Request == nil {
		return nil
	}
	state := service.OpsRequestTraceStateFromContext(c.Request.Context())
	if state == nil {
		return nil
	}
	trace := state.Snapshot()
	if trace == nil {
		return nil
	}

	now := time.Now().UTC()
	trace.FinishedAt = &now
	if trace.CreatedAt.IsZero() {
		trace.CreatedAt = now
	}
	durationMs := now.Sub(trace.CreatedAt).Milliseconds()
	if durationMs < 0 {
		durationMs = 0
	}
	trace.DurationMs = &durationMs

	statusCode := c.Writer.Status()
	trace.FinalStatusCode = &statusCode
	if statusCode >= 400 {
		trace.Status = "error"
	} else {
		trace.Status = "success"
	}

	clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	if trace.ClientRequestID == "" {
		trace.ClientRequestID = strings.TrimSpace(clientRequestID)
	}
	localRequestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
	if trace.LocalRequestID == "" {
		trace.LocalRequestID = strings.TrimSpace(localRequestID)
	}
	if c.Request.URL != nil {
		trace.RequestPath = c.Request.URL.Path
	}

	apiKey, _ := middleware2.GetAPIKeyFromContext(c)
	if apiKey != nil {
		trace.APIKeyID = &apiKey.ID
		if apiKey.User != nil {
			trace.UserID = &apiKey.User.ID
		}
		if apiKey.GroupID != nil {
			trace.GroupID = apiKey.GroupID
		}
		if apiKey.Group != nil && strings.TrimSpace(apiKey.Group.Platform) != "" {
			trace.Platform = strings.TrimSpace(apiKey.Group.Platform)
		}
	}
	if trace.Platform == "" {
		if platform, _ := c.Request.Context().Value(ctxkey.Platform).(string); strings.TrimSpace(platform) != "" {
			trace.Platform = strings.TrimSpace(platform)
		}
	}
	if trace.InboundEndpoint == "" {
		trace.InboundEndpoint = GetInboundEndpoint(c)
	}
	trace.UpstreamEndpoint = GetUpstreamEndpoint(c, trace.Platform)

	if v, ok := c.Get(opsUpstreamModelKey); ok {
		if upstreamModel, ok := v.(string); ok && strings.TrimSpace(upstreamModel) != "" {
			trace.FinalUpstreamModel = strings.TrimSpace(upstreamModel)
		}
	}
	if trace.FinalUpstreamModel == "" && trace.GroupResolvedModel != "" {
		trace.FinalUpstreamModel = trace.GroupResolvedModel
	}
	if v, ok := c.Get(opsAccountIDKey); ok {
		if accountID, ok := v.(int64); ok && accountID > 0 {
			trace.FinalAccountID = &accountID
		}
	}

	if originalModel, ok := c.Get(gatewayUserOriginalModelKey); ok {
		if value, ok := originalModel.(string); ok && strings.TrimSpace(value) != "" {
			trace.OriginalRequestedModel = strings.TrimSpace(value)
		}
	}
	if requestedModel, ok := c.Get(gatewayRequestedModelContextKey); ok {
		if value, ok := requestedModel.(string); ok && strings.TrimSpace(value) != "" {
			if trace.GroupResolvedModel == "" {
				trace.GroupResolvedModel = strings.TrimSpace(value)
			}
			if trace.AccountSupportLookupModel == "" {
				trace.AccountSupportLookupModel = strings.TrimSpace(value)
			}
		}
	}

	appendUpstreamTraceEvents(c, trace)
	appendRequestFinishedEvent(trace, statusCode, now)
	return trace
}

func appendUpstreamTraceEvents(c *gin.Context, trace *service.OpsRequestTrace) {
	if c == nil || trace == nil {
		return
	}
	var upstreamEvents []*service.OpsUpstreamErrorEvent
	if v, exists := c.Get(service.OpsUpstreamErrorsKey); exists {
		if events, ok := v.([]*service.OpsUpstreamErrorEvent); ok {
			upstreamEvents = events
		}
	}
	if len(upstreamEvents) == 0 {
		return
	}

	seenRequestIDs := make(map[string]struct{}, len(trace.UpstreamRequestIDs))
	for _, requestID := range trace.UpstreamRequestIDs {
		if value := strings.TrimSpace(requestID); value != "" {
			seenRequestIDs[value] = struct{}{}
		}
	}
	for _, event := range upstreamEvents {
		if event == nil {
			continue
		}
		if requestID := strings.TrimSpace(event.UpstreamRequestID); requestID != "" {
			if _, exists := seenRequestIDs[requestID]; !exists {
				seenRequestIDs[requestID] = struct{}{}
				trace.UpstreamRequestIDs = append(trace.UpstreamRequestIDs, requestID)
			}
		}
		trace.TraceEvents = append(trace.TraceEvents, &service.OpsRequestTraceEvent{
			Type:       "upstream_attempt",
			OccurredAt: time.UnixMilli(event.AtUnixMs).UTC(),
			Data: map[string]any{
				"phase":               "upstream",
				"account_id":          event.AccountID,
				"account_name":        strings.TrimSpace(event.AccountName),
				"kind":                strings.TrimSpace(event.Kind),
				"upstream_status":     event.UpstreamStatusCode,
				"upstream_request_id": strings.TrimSpace(event.UpstreamRequestID),
				"upstream_url":        strings.TrimSpace(event.UpstreamURL),
			},
		})
	}
	if len(trace.UpstreamRequestIDs) > traceUpstreamRequestIDLimit {
		trace.UpstreamRequestIDs = trace.UpstreamRequestIDs[:traceUpstreamRequestIDLimit]
	}
}

func appendRequestFinishedEvent(trace *service.OpsRequestTrace, statusCode int, occurredAt time.Time) {
	if trace == nil {
		return
	}
	if traceHasEvent(trace, "request_finished") {
		return
	}
	trace.TraceEvents = append(trace.TraceEvents, &service.OpsRequestTraceEvent{
		Type:       "request_finished",
		OccurredAt: occurredAt.UTC(),
		Data: map[string]any{
			"phase":       "finish",
			"status":      trace.Status,
			"status_code": statusCode,
		},
	})
}

func sortedInt64Keys(values map[int64]struct{}) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func truncateInt64Slice(values []int64, limit int) []int64 {
	if len(values) == 0 {
		return nil
	}
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return append([]int64(nil), values[:limit]...)
}

func traceHasEvent(trace *service.OpsRequestTrace, eventType string) bool {
	if trace == nil {
		return false
	}
	eventType = strings.TrimSpace(strings.ToLower(eventType))
	for _, event := range trace.TraceEvents {
		if event == nil {
			continue
		}
		if strings.TrimSpace(strings.ToLower(event.Type)) == eventType {
			return true
		}
	}
	return false
}
