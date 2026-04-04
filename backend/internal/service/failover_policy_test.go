//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailoverPolicy_DefaultCodes(t *testing.T) {
	// Simulate what NewFailoverPolicy does: initialize with default codes.
	// Note: ensureLoaded() does NOT apply the package-level cache to the instance;
	// it only calls refresh() when the cache is missing/expired. So we must
	// call applyConfig() directly on the instance.
	failoverPolicySF.Forget("failover_policy")
	failoverPolicyCache.Store(&cachedFailoverPolicy{
		codeMap:    map[int]bool{400: true, 401: true, 402: true, 403: true, 404: true, 429: true, 529: true},
		include5xx: true,
		expiresAt:  time.Now().Add(time.Hour).UnixNano(),
	})
	p := &FailoverPolicy{}
	p.applyConfig(defaultFailoverCodes)
	// Default codes: 400, 401, 402, 403, 404, 429, 529 + all 5xx
	assert.True(t, p.ShouldFailover(400))
	assert.True(t, p.ShouldFailover(401))
	assert.True(t, p.ShouldFailover(402))
	assert.True(t, p.ShouldFailover(403))
	assert.True(t, p.ShouldFailover(404))
	assert.True(t, p.ShouldFailover(429))
	assert.True(t, p.ShouldFailover(529))
	assert.True(t, p.ShouldFailover(500))
	assert.True(t, p.ShouldFailover(502))
	assert.True(t, p.ShouldFailover(503))
	assert.False(t, p.ShouldFailover(200))
	assert.False(t, p.ShouldFailover(201))
	assert.False(t, p.ShouldFailover(204))
	assert.False(t, p.ShouldFailover(301))
	assert.False(t, p.ShouldFailover(408))
}

func TestFailoverPolicy_CustomCodes(t *testing.T) {
	p := &FailoverPolicy{}
	p.applyConfig(&FailoverStatusCodesConfig{
		Codes:      []int{401, 403},
		Include5xx: false,
	})
	assert.True(t, p.ShouldFailover(401))
	assert.True(t, p.ShouldFailover(403))
	assert.False(t, p.ShouldFailover(400))
	assert.False(t, p.ShouldFailover(500))
	assert.False(t, p.ShouldFailover(502))
}

func TestFailoverPolicy_Include5xx(t *testing.T) {
	p := &FailoverPolicy{}
	p.applyConfig(&FailoverStatusCodesConfig{
		Codes:      []int{401},
		Include5xx: true,
	})
	assert.True(t, p.ShouldFailover(401))
	assert.True(t, p.ShouldFailover(500))
	assert.True(t, p.ShouldFailover(599))
	assert.False(t, p.ShouldFailover(400))
}

func TestFailoverPolicy_EmptyConfig_FallsBackToDefaults(t *testing.T) {
	// Ensure default codes are cached so ensureLoaded() uses them (and doesn't trigger refresh).
	failoverPolicySF.Forget("failover_policy")
	failoverPolicyCache.Store(&cachedFailoverPolicy{
		codeMap:    map[int]bool{400: true, 401: true, 402: true, 403: true, 404: true, 429: true, 529: true},
		include5xx: true,
		expiresAt:  time.Now().Add(time.Hour).UnixNano(),
	})
	p := &FailoverPolicy{}
	p.applyConfig(nil) // nil -> defaults
	p.applyConfig(nil)
	assert.True(t, p.ShouldFailover(400))
	assert.True(t, p.ShouldFailover(500))
}
