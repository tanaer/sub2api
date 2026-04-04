//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type tempUnschedTraceRepo struct {
	mockAccountRepoForGemini
	account *Account
	reason  string
	until   time.Time
}

func (r *tempUnschedTraceRepo) GetByID(context.Context, int64) (*Account, error) { return r.account, nil }
func (r *tempUnschedTraceRepo) SetTempUnschedulable(_ context.Context, _ int64, until time.Time, reason string) error {
	r.until = until
	r.reason = reason
	return nil
}

func TestRateLimitService_HandleTempUnschedulable_PersistsRequestTrace(t *testing.T) {
	repo := &tempUnschedTraceRepo{
		account: &Account{
			ID:       7,
			Platform: PlatformOpenAI,
			Credentials: map[string]any{
				"temp_unschedulable_enabled": true,
				"temp_unschedulable_rules": []any{
					map[string]any{
						"error_code":       float64(404),
						"keywords":         []any{"Model Not Found"},
						"duration_minutes": float64(5),
					},
				},
			},
		},
	}
	svc := &RateLimitService{accountRepo: repo}
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "req-temp-404")

	hit := svc.HandleTempUnschedulable(ctx, repo.account, 404, []byte(`{"error":{"message":"Model Not Found"}}`))
	require.True(t, hit)

	var state TempUnschedState
	require.NoError(t, json.Unmarshal([]byte(repo.reason), &state))
	require.Equal(t, "req-temp-404", state.RequestID)
	require.Equal(t, 404, state.UpstreamStatusCode)
	require.Equal(t, "Model Not Found", state.UpstreamErrorMessage)
	require.Contains(t, state.UpstreamErrorDetail, `"Model Not Found"`)
}

func TestRateLimitService_TryAccountThrottle_PersistsRequestTrace(t *testing.T) {
	repo := &tempUnschedTraceRepo{account: &Account{ID: 23, Platform: PlatformAnthropic}}
	throttleSvc := &AccountThrottleService{
		localCache: []*cachedThrottleRule{
			{
				AccountThrottleRule: &model.AccountThrottleRule{
					ID:          1,
					Name:        "xunfei",
					Enabled:     true,
					Platforms:   []string{"anthropic"},
					ErrorCodes:  []int{},
					Keywords:    []string{"Xunfei claude request failed with Sid"},
					MatchMode:   model.ThrottleMatchContains,
					TriggerMode: model.ThrottleTriggerImmediate,
					ActionType:  model.ThrottleActionScheduledRecovery,
					ActionRecoverHour: 0,
				},
				lowerKeywords:  []string{"xunfei claude request failed with sid"},
				lowerPlatforms: []string{"anthropic"},
			},
		},
	}
	svc := &RateLimitService{accountRepo: repo, accountThrottleService: throttleSvc}
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "req-throttle-1")

	hit := svc.tryAccountThrottle(ctx, repo.account, 404, []byte(`{"code":10404,"message":"Xunfei claude request failed with Sid xx: Model Not Found"}`))
	require.True(t, hit)

	var state TempUnschedState
	require.NoError(t, json.Unmarshal([]byte(repo.reason), &state))
	require.Equal(t, "req-throttle-1", state.RequestID)
	require.Equal(t, 404, state.UpstreamStatusCode)
	require.Contains(t, state.UpstreamErrorMessage, "Xunfei")
}

func TestRateLimitService_TriggerStreamTimeoutTempUnsched_PersistsRequestID(t *testing.T) {
	repo := &tempUnschedTraceRepo{account: &Account{ID: 88, Platform: PlatformOpenAI}}
	svc := &RateLimitService{accountRepo: repo}
	settings := &StreamTimeoutSettings{TempUnschedMinutes: 3}
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "req-stream-timeout")

	hit := svc.triggerStreamTimeoutTempUnsched(ctx, repo.account, settings, "gpt-4.1")
	require.True(t, hit)

	var state TempUnschedState
	require.NoError(t, json.Unmarshal([]byte(repo.reason), &state))
	require.Equal(t, "req-stream-timeout", state.RequestID)
	require.Zero(t, state.UpstreamStatusCode)
	require.Equal(t, "Stream data interval timeout for model: gpt-4.1", state.ErrorMessage)
}
