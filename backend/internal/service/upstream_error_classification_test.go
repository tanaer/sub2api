//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsUpstreamBillingIssue_MatchesZhipuBillingCode(t *testing.T) {
	require.True(t, IsUpstreamBillingIssue(
		http.StatusTooManyRequests,
		[]byte(`{"error":{"code":"1113","message":"余额不足或无可用资源包,请充值。"}}`),
	))
}

func TestIsUpstreamBillingIssue_MatchesEnglishBillingMessage(t *testing.T) {
	require.True(t, IsUpstreamBillingIssue(
		http.StatusTooManyRequests,
		[]byte(`{"error":{"message":"insufficient balance, please recharge"}}`),
	))
}
