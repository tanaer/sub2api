package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
)

func TestExtractBodyErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []int
	}{
		{
			name:     "empty body",
			body:     "",
			expected: nil,
		},
		{
			name:     "xunfei style: top-level code",
			body:     `{"code":11200,"message":"Xunfei claude request failed with Sid xxx"}`,
			expected: []int{11200},
		},
		{
			name:     "openai style: error.code (numeric)",
			body:     `{"error":{"code":1302,"message":"rate limited"}}`,
			expected: []int{1302},
		},
		{
			name:     "header.code style",
			body:     `{"header":{"code":11200,"message":"error"},"payload":{}}`,
			expected: []int{11200},
		},
		{
			name:     "http-range code ignored (400)",
			body:     `{"code":400,"message":"bad request"}`,
			expected: nil,
		},
		{
			name:     "http-range code ignored (200)",
			body:     `{"code":200,"message":"ok"}`,
			expected: nil,
		},
		{
			name:     "string code",
			body:     `{"code":"11200","message":"error"}`,
			expected: []int{11200},
		},
		{
			name:     "multiple paths with same code deduped",
			body:     `{"code":11200,"error":{"code":11200}}`,
			expected: []int{11200},
		},
		{
			name:     "multiple different codes",
			body:     `{"code":11200,"error":{"code":10001}}`,
			expected: []int{11200, 10001},
		},
		{
			name:     "non-numeric code ignored",
			body:     `{"error":{"code":"invalid_api_key","message":"error"}}`,
			expected: nil,
		},
		{
			name:     "no code field",
			body:     `{"error":{"message":"something went wrong"}}`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractBodyErrorCodes([]byte(tt.body))
			if len(got) != len(tt.expected) {
				t.Fatalf("extractBodyErrorCodes(%q) = %v, want %v", tt.body, got, tt.expected)
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("extractBodyErrorCodes(%q)[%d] = %d, want %d", tt.body, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestErrorCodeMatches_WithBodyCodes(t *testing.T) {
	svc := &AccountThrottleService{}

	tests := []struct {
		name           string
		ruleErrorCodes []int
		httpStatus     int
		bodyErrorCodes []int
		expected       bool
	}{
		{
			name:           "empty rule codes matches all",
			ruleErrorCodes: nil,
			httpStatus:     200,
			bodyErrorCodes: nil,
			expected:       true,
		},
		{
			name:           "http status matches",
			ruleErrorCodes: []int{429},
			httpStatus:     429,
			bodyErrorCodes: nil,
			expected:       true,
		},
		{
			name:           "body code matches",
			ruleErrorCodes: []int{11200},
			httpStatus:     200,
			bodyErrorCodes: []int{11200},
			expected:       true,
		},
		{
			name:           "neither matches",
			ruleErrorCodes: []int{11200},
			httpStatus:     400,
			bodyErrorCodes: []int{10001},
			expected:       false,
		},
		{
			name:           "body code matches one of multiple rule codes",
			ruleErrorCodes: []int{429, 11200},
			httpStatus:     200,
			bodyErrorCodes: []int{11200},
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &cachedThrottleRule{}
			rule.AccountThrottleRule = &model.AccountThrottleRule{
				ErrorCodes: tt.ruleErrorCodes,
			}
			got := svc.errorCodeMatches(rule, tt.httpStatus, tt.bodyErrorCodes)
			if got != tt.expected {
				t.Errorf("errorCodeMatches() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestMatchAndThrottle_DefaultBillingCycle403Rule(t *testing.T) {
	svc := &AccountThrottleService{}
	resetAt := time.Now().Add(6 * time.Hour).UTC().Truncate(time.Second)
	account := &Account{
		ID:       88,
		Platform: PlatformOpenAI,
		Extra: map[string]any{
			"quota_daily_reset_at": resetAt.Format(time.RFC3339),
		},
	}

	result := svc.MatchAndThrottle(
		context.Background(),
		account,
		403,
		[]byte(`{"error":{"message":"Access forbidden (403): You've reached your usage limit for this billing cycle. Your quota will be refreshed in the next cycle. Upgrade to get more."}}`),
	)

	if result == nil {
		t.Fatal("expected builtin billing-cycle 403 rule to match")
	}
	if !result.Matched {
		t.Fatal("expected builtin billing-cycle 403 rule to be marked matched")
	}
	if result.Rule == nil {
		t.Fatal("expected matched builtin rule metadata")
	}
	if result.UntilTime.IsZero() {
		t.Fatal("expected builtin billing-cycle 403 rule to set recovery time")
	}
	if !result.UntilTime.Equal(resetAt) {
		t.Fatalf("until = %v, want %v", result.UntilTime, resetAt)
	}
}
