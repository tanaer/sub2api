//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsRequestTraceRepository_LookupByEachKeyType(t *testing.T) {
	ctx := context.Background()
	_, _ = integrationDB.ExecContext(ctx, "TRUNCATE ops_request_traces RESTART IDENTITY")

	repo := NewOpsRepository(integrationDB).(*opsRepository)
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	trace := buildOpsRequestTraceFixture("trace-client-1", base)
	trace.LocalRequestID = "local-lookup-1"
	trace.UsageRequestID = "usage-lookup-1"
	trace.UpstreamRequestIDs = []string{"upstream-lookup-1", "upstream-lookup-2"}

	require.NoError(t, repo.UpsertRequestTrace(ctx, trace))

	cases := []struct {
		name    string
		key     string
		keyType string
	}{
		{name: "client", key: "trace-client-1", keyType: service.OpsRequestTraceKeyTypeClient},
		{name: "local", key: "local-lookup-1", keyType: service.OpsRequestTraceKeyTypeLocal},
		{name: "usage", key: "usage-lookup-1", keyType: service.OpsRequestTraceKeyTypeUsage},
		{name: "upstream", key: "upstream-lookup-2", keyType: service.OpsRequestTraceKeyTypeUpstream},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			traces, err := repo.ListRequestTracesByKey(ctx, tc.key, tc.keyType, 20)
			require.NoError(t, err)
			require.Len(t, traces, 1)
			require.Equal(t, "trace-client-1", traces[0].ClientRequestID)
		})
	}
}

func TestOpsRequestTraceRepository_LookupReturnsNewestFirstAndCapsCandidates(t *testing.T) {
	ctx := context.Background()
	_, _ = integrationDB.ExecContext(ctx, "TRUNCATE ops_request_traces RESTART IDENTITY")

	repo := NewOpsRepository(integrationDB).(*opsRepository)
	base := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 25; i++ {
		trace := buildOpsRequestTraceFixture(fmt.Sprintf("trace-client-%02d", i), base.Add(time.Duration(i)*time.Minute))
		trace.LocalRequestID = "shared-local-key"
		trace.Status = "success"
		accountID := int64(i + 1)
		trace.FinalAccountID = &accountID
		require.NoError(t, repo.UpsertRequestTrace(ctx, trace))
	}

	traces, err := repo.ListRequestTracesByKey(ctx, "shared-local-key", service.OpsRequestTraceKeyTypeLocal, 20)
	require.NoError(t, err)
	require.Len(t, traces, 20)
	require.Equal(t, "trace-client-24", traces[0].ClientRequestID)
	require.Equal(t, "trace-client-05", traces[19].ClientRequestID)
}

func buildOpsRequestTraceFixture(clientRequestID string, createdAt time.Time) *service.OpsRequestTrace {
	return &service.OpsRequestTrace{
		ClientRequestID:           clientRequestID,
		Status:                    "processing",
		Platform:                  "openai",
		RequestPath:               "/v1/chat/completions",
		InboundEndpoint:           "/v1/chat/completions",
		UpstreamEndpoint:          "/v1/chat/completions",
		OriginalRequestedModel:    "sonnet",
		GroupResolvedModel:        "glm-4.5-air",
		AccountSupportLookupModel: "glm-4.5-air",
		FinalUpstreamModel:        "glm-4.5-air",
		CreatedAt:                 createdAt,
		TraceEvents: []*service.OpsRequestTraceEvent{
			{
				Type:       "request_received",
				OccurredAt: createdAt,
			},
		},
	}
}
