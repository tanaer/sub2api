package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
