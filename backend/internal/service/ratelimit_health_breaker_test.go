//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCircuitBreaker_TripsAfterThreeFailures(t *testing.T) {
	cb := NewAccountCircuitBreaker()
	ht := NewAccountHealthTracker()
	svc := &RateLimitService{
		accountRepo:    &rateLimitAccountRepoStub{},
		circuitBreaker: cb,
		healthTracker:  ht,
	}

	account := &Account{ID: 1001}

	for i := 0; i < 3; i++ {
		svc.HandleUpstreamError(context.Background(), account, 400, http.Header{}, []byte(`{"error":"bad request"}`))
	}
	assert.True(t, cb.IsTripped(1001), "account should be circuit-breaker tripped after 3 failures")
}

func TestHealthCircuitBreaker_NoTripWithInsufficientSamples(t *testing.T) {
	cb := NewAccountCircuitBreaker()
	ht := NewAccountHealthTracker()
	svc := &RateLimitService{
		accountRepo:    &rateLimitAccountRepoStub{},
		circuitBreaker: cb,
		healthTracker:  ht,
	}

	account := &Account{ID: 1002}

	for i := 0; i < 2; i++ {
		svc.HandleUpstreamError(context.Background(), account, 400, http.Header{}, []byte(`{"error":"bad"}`))
	}
	assert.False(t, cb.IsTripped(1002), "account should not be tripped with only 2 failures")
}

func TestHealthCircuitBreaker_NilTrackerNoPanic(t *testing.T) {
	cb := NewAccountCircuitBreaker()
	svc := &RateLimitService{
		accountRepo:    &rateLimitAccountRepoStub{},
		circuitBreaker: cb,
		healthTracker:  nil,
	}

	account := &Account{ID: 1004}

	require.NotPanics(t, func() {
		svc.HandleUpstreamError(context.Background(), account, 400, http.Header{}, []byte(`{"error":"bad"}`))
	})
}
