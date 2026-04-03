//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountHealthTracker_NewReturns100(t *testing.T) {
	ht := NewAccountHealthTracker()
	require.Equal(t, 100, ht.HealthScore(999))
}

func TestAccountHealthTracker_AllSuccess(t *testing.T) {
	ht := NewAccountHealthTracker()
	for i := 0; i < 5; i++ {
		ht.RecordSuccess(1)
	}
	require.Equal(t, 100, ht.HealthScore(1))
}

func TestAccountHealthTracker_AllFailure(t *testing.T) {
	ht := NewAccountHealthTracker()
	for i := 0; i < 5; i++ {
		ht.RecordFailure(1)
	}
	require.Equal(t, 0, ht.HealthScore(1))
}

func TestAccountHealthTracker_MixedSuccessFailure(t *testing.T) {
	ht := NewAccountHealthTracker()
	// 3 success, 2 failures = 60%
	for i := 0; i < 3; i++ {
		ht.RecordSuccess(1)
	}
	for i := 0; i < 2; i++ {
		ht.RecordFailure(1)
	}
	require.Equal(t, 60, ht.HealthScore(1))
}

func TestAccountHealthTracker_BelowMinSamplesReturns100(t *testing.T) {
	ht := NewAccountHealthTracker()
	ht.RecordFailure(1)
	ht.RecordFailure(1)
	// Only 2 samples, below healthMinSamples (3)
	require.Equal(t, 100, ht.HealthScore(1))
}

func TestAccountHealthTracker_IndependentAccounts(t *testing.T) {
	ht := NewAccountHealthTracker()
	for i := 0; i < 5; i++ {
		ht.RecordSuccess(1)
	}
	for i := 0; i < 5; i++ {
		ht.RecordFailure(2)
	}
	require.Equal(t, 100, ht.HealthScore(1))
	require.Equal(t, 0, ht.HealthScore(2))
}

func TestSortAccountsWithHealthWeighting(t *testing.T) {
	// Account A: priority 1, healthy (score 100)
	// Account B: priority 1, unhealthy (score < 50, effective priority = 3)
	// Account C: priority 2, healthy (score 100)
	// Expected order: A, C, B (B is deprioritized below C despite lower base priority)
	ht := NewAccountHealthTracker()
	for i := 0; i < 10; i++ {
		ht.RecordFailure(2) // Make account 2 very unhealthy
	}
	for i := 0; i < 10; i++ {
		ht.RecordSuccess(1) // Account 1 healthy
		ht.RecordSuccess(3) // Account 3 healthy
	}

	accounts := []*Account{
		{ID: 2, Priority: 1, Name: "unhealthy-p1"},
		{ID: 1, Priority: 1, Name: "healthy-p1"},
		{ID: 3, Priority: 2, Name: "healthy-p2"},
	}

	sortAccountsWithHealthWeighting(accounts, false, func(id int64) int { return ht.HealthScore(id) })

	require.Equal(t, int64(1), accounts[0].ID, "healthy p1 should be first")
	require.Equal(t, int64(3), accounts[1].ID, "healthy p2 should be second")
	require.Equal(t, int64(2), accounts[2].ID, "unhealthy p1 should be last")
}
