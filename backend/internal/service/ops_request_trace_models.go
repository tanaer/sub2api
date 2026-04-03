package service

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	OpsRequestTraceKeyTypeAuto     = "auto"
	OpsRequestTraceKeyTypeClient   = "client"
	OpsRequestTraceKeyTypeLocal    = "local"
	OpsRequestTraceKeyTypeUsage    = "usage"
	OpsRequestTraceKeyTypeUpstream = "upstream"

	opsRequestTraceCandidateLimit = 20
)

var opsRequestTraceAutoPriority = []string{
	OpsRequestTraceKeyTypeClient,
	OpsRequestTraceKeyTypeLocal,
	OpsRequestTraceKeyTypeUsage,
	OpsRequestTraceKeyTypeUpstream,
}

type OpsRequestTrace struct {
	ClientRequestID    string   `json:"client_request_id"`
	LocalRequestID     string   `json:"local_request_id,omitempty"`
	UsageRequestID     string   `json:"usage_request_id,omitempty"`
	UpstreamRequestIDs []string `json:"upstream_request_ids,omitempty"`

	OriginalRequestedModel    string `json:"original_requested_model,omitempty"`
	GroupResolvedModel        string `json:"group_resolved_model,omitempty"`
	AccountSupportLookupModel string `json:"account_support_lookup_model,omitempty"`
	FinalUpstreamModel        string `json:"final_upstream_model,omitempty"`

	Status          string `json:"status,omitempty"`
	FinalStatusCode *int   `json:"final_status_code,omitempty"`
	TraceIncomplete bool   `json:"trace_incomplete,omitempty"`

	Platform         string `json:"platform,omitempty"`
	RequestPath      string `json:"request_path,omitempty"`
	InboundEndpoint  string `json:"inbound_endpoint,omitempty"`
	UpstreamEndpoint string `json:"upstream_endpoint,omitempty"`

	UserID         *int64 `json:"user_id,omitempty"`
	APIKeyID       *int64 `json:"api_key_id,omitempty"`
	GroupID        *int64 `json:"group_id,omitempty"`
	FinalAccountID *int64 `json:"final_account_id,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	DurationMs *int64     `json:"duration_ms,omitempty"`

	TraceEvents []*OpsRequestTraceEvent `json:"trace_events,omitempty"`
}

type OpsRequestTraceEvent struct {
	Type       string         `json:"type"`
	OccurredAt time.Time      `json:"occurred_at"`
	Data       map[string]any `json:"data,omitempty"`
}

type OpsRequestTraceCandidate struct {
	ClientRequestID string    `json:"client_request_id"`
	CreatedAt       time.Time `json:"created_at"`
	Status          string    `json:"status,omitempty"`
	FinalAccountID  *int64    `json:"final_account_id,omitempty"`
}

type OpsRequestTraceQuery struct {
	Key     string `json:"key"`
	KeyType string `json:"key_type"`
}

type OpsRequestTraceLookupResult struct {
	Trace      *OpsRequestTrace            `json:"trace,omitempty"`
	Candidates []*OpsRequestTraceCandidate `json:"candidates,omitempty"`
	Ambiguous  bool                        `json:"ambiguous"`
}

func normalizeOpsRequestTraceKeyType(raw string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return OpsRequestTraceKeyTypeAuto, nil
	}
	switch value {
	case OpsRequestTraceKeyTypeAuto,
		OpsRequestTraceKeyTypeClient,
		OpsRequestTraceKeyTypeLocal,
		OpsRequestTraceKeyTypeUsage,
		OpsRequestTraceKeyTypeUpstream:
		return value, nil
	default:
		return "", fmt.Errorf("invalid trace key_type")
	}
}

func buildOpsRequestTraceCandidates(traces []*OpsRequestTrace) []*OpsRequestTraceCandidate {
	if len(traces) == 0 {
		return nil
	}
	sorted := append([]*OpsRequestTrace{}, traces...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})
	if len(sorted) > opsRequestTraceCandidateLimit {
		sorted = sorted[:opsRequestTraceCandidateLimit]
	}
	candidates := make([]*OpsRequestTraceCandidate, 0, len(sorted))
	for _, trace := range sorted {
		if trace == nil {
			continue
		}
		candidates = append(candidates, &OpsRequestTraceCandidate{
			ClientRequestID: trace.ClientRequestID,
			CreatedAt:       trace.CreatedAt,
			Status:          trace.Status,
			FinalAccountID:  trace.FinalAccountID,
		})
	}
	return candidates
}
