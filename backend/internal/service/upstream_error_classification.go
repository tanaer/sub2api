package service

import (
	"net/http"
	"strings"
)

// IsUpstreamBillingIssue reports whether the upstream response represents
// a balance or billing failure rather than a transient rate limit.
func IsUpstreamBillingIssue(statusCode int, body []byte) bool {
	if statusCode == http.StatusPaymentRequired {
		return true
	}

	code := strings.TrimSpace(extractUpstreamErrorCode(body))
	switch strings.ToUpper(code) {
	case "1113", "INSUFFICIENT_BALANCE":
		return true
	}

	msg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if msg == "" {
		return false
	}

	lower := strings.ToLower(msg)
	if strings.Contains(lower, "insufficient balance") ||
		strings.Contains(lower, "billing issue") ||
		strings.Contains(lower, "payment required") ||
		strings.Contains(lower, "resource package") ||
		strings.Contains(lower, "please recharge") {
		return true
	}

	return strings.Contains(msg, "余额不足") ||
		strings.Contains(msg, "资源包") ||
		strings.Contains(msg, "请充值")
}
