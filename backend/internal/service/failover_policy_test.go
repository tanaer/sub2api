//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testFailoverPolicy creates a FailoverPolicy with default codes for use in tests.
func testFailoverPolicy() *FailoverPolicy {
	p := &FailoverPolicy{}
	p.applyConfig(defaultFailoverCodes)
	return p
}

func TestFailoverPolicy_DefaultCodes(t *testing.T) {
	p := &FailoverPolicy{}
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
	p := &FailoverPolicy{}
	p.applyConfig(nil)
	assert.True(t, p.ShouldFailover(400))
	assert.True(t, p.ShouldFailover(500))
}
