package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpsRequestTraceLookup_AutoPriorityStopsAtFirstMatch(t *testing.T) {
	ctx := context.Background()
	callOrder := make([]string, 0, 4)

	repo := &opsRepoMock{
		ListRequestTracesByKeyFn: func(ctx context.Context, key string, keyType string, limit int) ([]*OpsRequestTrace, error) {
			callOrder = append(callOrder, keyType)
			if keyType == OpsRequestTraceKeyTypeLocal {
				return []*OpsRequestTrace{
					{
						ClientRequestID: "client-1",
						LocalRequestID:  "local-1",
						Status:          "success",
						CreatedAt:       time.Unix(1712188800, 0).UTC(),
					},
				}, nil
			}
			return nil, nil
		},
	}

	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	result, err := svc.GetRequestTrace(ctx, &OpsRequestTraceQuery{
		Key:     "trace-key",
		KeyType: OpsRequestTraceKeyTypeAuto,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Trace)
	require.Equal(t, []string{OpsRequestTraceKeyTypeClient, OpsRequestTraceKeyTypeLocal}, callOrder)
	require.Equal(t, "local-1", result.Trace.LocalRequestID)
	require.False(t, result.Ambiguous)
}

func TestOpsRequestTraceLookup_AmbiguousCandidatesSorted(t *testing.T) {
	ctx := context.Background()

	repo := &opsRepoMock{
		ListRequestTracesByKeyFn: func(ctx context.Context, key string, keyType string, limit int) ([]*OpsRequestTrace, error) {
			return []*OpsRequestTrace{
				{
					ClientRequestID: "client-older",
					Status:          "failed",
					FinalAccountID:  int64PtrTrace(11),
					CreatedAt:       time.Unix(1712188800, 0).UTC(),
				},
				{
					ClientRequestID: "client-newer",
					Status:          "success",
					FinalAccountID:  int64PtrTrace(22),
					CreatedAt:       time.Unix(1712189900, 0).UTC(),
				},
			}, nil
		},
	}

	svc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	result, err := svc.GetRequestTrace(ctx, &OpsRequestTraceQuery{
		Key:     "local-dup",
		KeyType: OpsRequestTraceKeyTypeLocal,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Ambiguous)
	require.Nil(t, result.Trace)
	require.Len(t, result.Candidates, 2)
	require.Equal(t, "client-newer", result.Candidates[0].ClientRequestID)
	require.Equal(t, "success", result.Candidates[0].Status)
	require.Equal(t, int64(22), *result.Candidates[0].FinalAccountID)
	require.Equal(t, "client-older", result.Candidates[1].ClientRequestID)
}

func int64PtrTrace(v int64) *int64 {
	return &v
}
