package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type anthropic429AccountRepo struct {
	rateLimitedID      int64
	rateLimitedAt      time.Time
	rateLimitedCalls   int
	sessionWindowCalls int
}

func (r *anthropic429AccountRepo) Create(context.Context, *Account) error { return nil }

func (r *anthropic429AccountRepo) GetByID(context.Context, int64) (*Account, error) { return nil, nil }

func (r *anthropic429AccountRepo) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ExistsByID(context.Context, int64) (bool, error) { return false, nil }

func (r *anthropic429AccountRepo) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) Update(context.Context, *Account) error { return nil }

func (r *anthropic429AccountRepo) Delete(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *anthropic429AccountRepo) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (r *anthropic429AccountRepo) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListActive(context.Context) ([]Account, error) { return nil, nil }

func (r *anthropic429AccountRepo) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) UpdateLastUsed(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}

func (r *anthropic429AccountRepo) SetError(context.Context, int64, string) error { return nil }

func (r *anthropic429AccountRepo) ClearError(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) SetSchedulable(context.Context, int64, bool) error { return nil }

func (r *anthropic429AccountRepo) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *anthropic429AccountRepo) BindGroups(context.Context, int64, []int64) error { return nil }

func (r *anthropic429AccountRepo) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}

func (r *anthropic429AccountRepo) SetRateLimited(_ context.Context, id int64, resetAt time.Time) error {
	r.rateLimitedID = id
	r.rateLimitedAt = resetAt
	r.rateLimitedCalls++
	return nil
}

func (r *anthropic429AccountRepo) UpdateSessionWindow(_ context.Context, _ int64, _, _ *time.Time, _ string) error {
	r.sessionWindowCalls++
	return nil
}

func (r *anthropic429AccountRepo) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}

func (r *anthropic429AccountRepo) SetOverloaded(context.Context, int64, time.Time) error { return nil }

func (r *anthropic429AccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}

func (r *anthropic429AccountRepo) ClearTempUnschedulable(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) ClearRateLimit(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}

func (r *anthropic429AccountRepo) ClearModelRateLimits(context.Context, int64) error { return nil }

func (r *anthropic429AccountRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}

func (r *anthropic429AccountRepo) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}

func (r *anthropic429AccountRepo) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}

func (r *anthropic429AccountRepo) ResetQuotaUsed(context.Context, int64) error { return nil }

var _ AccountRepository = (*anthropic429AccountRepo)(nil)

func TestCalculateAnthropic429ResetTime_Only5hExceeded(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "1.02")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "0.32")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1770998400)

	if result.fiveHourReset == nil || !result.fiveHourReset.Equal(time.Unix(1770998400, 0)) {
		t.Errorf("expected fiveHourReset=1770998400, got %v", result.fiveHourReset)
	}
}

func TestCalculateAnthropic429ResetTime_Only7dExceeded(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "0.50")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "1.05")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1771549200)

	// fiveHourReset should still be populated for session window calculation
	if result.fiveHourReset == nil || !result.fiveHourReset.Equal(time.Unix(1770998400, 0)) {
		t.Errorf("expected fiveHourReset=1770998400, got %v", result.fiveHourReset)
	}
}

func TestCalculateAnthropic429ResetTime_BothExceeded(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "1.10")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "1.02")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1771549200)
}

func TestCalculateAnthropic429ResetTime_NoPerWindowHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	if result != nil {
		t.Errorf("expected nil result when no per-window headers, got resetAt=%v", result.resetAt)
	}
}

func TestCalculateAnthropic429ResetTime_NoHeaders(t *testing.T) {
	result := calculateAnthropic429ResetTime(http.Header{})
	if result != nil {
		t.Errorf("expected nil result for empty headers, got resetAt=%v", result.resetAt)
	}
}

func TestCalculateAnthropic429ResetTime_SurpassedThreshold(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-surpassed-threshold", "true")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-surpassed-threshold", "false")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1770998400)
}

func TestCalculateAnthropic429ResetTime_UtilizationExactlyOne(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "1.0")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "0.5")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1770998400)
}

func TestCalculateAnthropic429ResetTime_NeitherExceeded_UsesShorter(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "0.95")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400") // sooner
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "0.80")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200") // later

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1770998400)
}

func TestCalculateAnthropic429ResetTime_Only5hResetHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-5h-utilization", "1.05")
	headers.Set("anthropic-ratelimit-unified-5h-reset", "1770998400")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1770998400)
}

func TestCalculateAnthropic429ResetTime_Only7dResetHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("anthropic-ratelimit-unified-7d-utilization", "1.03")
	headers.Set("anthropic-ratelimit-unified-7d-reset", "1771549200")

	result := calculateAnthropic429ResetTime(headers)
	assertAnthropicResult(t, result, 1771549200)

	if result.fiveHourReset != nil {
		t.Errorf("expected fiveHourReset=nil when no 5h headers, got %v", result.fiveHourReset)
	}
}

func TestHandle429_AnthropicWithoutResetHeaderButRateLimitBodyUsesFallbackCooldown(t *testing.T) {
	repo := &anthropic429AccountRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 321, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"type":"1302","message":"您的账户已达到速率限制，请您控制请求频率，您可联系管理员提升账户等级。"}}`)

	before := time.Now()
	svc.handle429(context.Background(), account, http.Header{}, body)
	after := time.Now()

	if repo.rateLimitedCalls != 1 {
		t.Fatalf("rateLimitedCalls = %d, want 1", repo.rateLimitedCalls)
	}
	if repo.rateLimitedID != account.ID {
		t.Fatalf("rateLimitedID = %d, want %d", repo.rateLimitedID, account.ID)
	}

	minExpected := before.Add(25 * time.Second)
	maxExpected := after.Add(35 * time.Second)
	if repo.rateLimitedAt.Before(minExpected) || repo.rateLimitedAt.After(maxExpected) {
		t.Fatalf("rateLimitedAt = %v, want within [%v, %v]", repo.rateLimitedAt, minExpected, maxExpected)
	}
	if repo.sessionWindowCalls != 0 {
		t.Fatalf("sessionWindowCalls = %d, want 0", repo.sessionWindowCalls)
	}
}

func TestHandle429_AnthropicWithoutResetHeaderButQuotaBodyParsesResetTime(t *testing.T) {
	repo := &anthropic429AccountRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 777, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"code":"AccountQuotaExceeded","message":"You have exceeded the monthly usage quota. It will reset at 2026-04-14 23:59:59 +0800 CST. We recommend upgrading your plan for more quota, or waiting for the reset. Request id: 021775028605020933bcb8fff5a6e275f05e6439785ee7938e92d","param":"","type":"TooManyRequests"}}`)

	svc.handle429(context.Background(), account, http.Header{}, body)

	if repo.rateLimitedCalls != 1 {
		t.Fatalf("rateLimitedCalls = %d, want 1", repo.rateLimitedCalls)
	}
	want := time.Date(2026, 4, 14, 23, 59, 59, 0, time.FixedZone("CST", 8*60*60))
	if !repo.rateLimitedAt.Equal(want) {
		t.Fatalf("rateLimitedAt = %v, want %v", repo.rateLimitedAt, want)
	}
	if repo.sessionWindowCalls != 0 {
		t.Fatalf("sessionWindowCalls = %d, want 0", repo.sessionWindowCalls)
	}
}

func TestHandle429_AnthropicWithoutResetHeaderButHighTrafficBodyUsesFallbackCooldown(t *testing.T) {
	repo := &anthropic429AccountRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 888, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"type":"1305","message":"该模型当前访问量过大，请您稍后再试 (request id: 20260401073610846050136KnA11zRt)"},"type":"error"}`)

	before := time.Now()
	svc.handle429(context.Background(), account, http.Header{}, body)
	after := time.Now()

	if repo.rateLimitedCalls != 1 {
		t.Fatalf("rateLimitedCalls = %d, want 1", repo.rateLimitedCalls)
	}
	minExpected := before.Add(25 * time.Second)
	maxExpected := after.Add(35 * time.Second)
	if repo.rateLimitedAt.Before(minExpected) || repo.rateLimitedAt.After(maxExpected) {
		t.Fatalf("rateLimitedAt = %v, want within [%v, %v]", repo.rateLimitedAt, minExpected, maxExpected)
	}
	if repo.sessionWindowCalls != 0 {
		t.Fatalf("sessionWindowCalls = %d, want 0", repo.sessionWindowCalls)
	}
}

func TestHandle429_AnthropicWithoutResetHeaderAndNonRateLimitBodyStillSkips(t *testing.T) {
	repo := &anthropic429AccountRepo{}
	svc := NewRateLimitService(repo, nil, nil, nil, nil)
	account := &Account{ID: 654, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
	body := []byte(`{"error":{"message":"Extra usage required to access this model"}}`)

	svc.handle429(context.Background(), account, http.Header{}, body)

	if repo.rateLimitedCalls != 0 {
		t.Fatalf("rateLimitedCalls = %d, want 0", repo.rateLimitedCalls)
	}
	if repo.sessionWindowCalls != 0 {
		t.Fatalf("sessionWindowCalls = %d, want 0", repo.sessionWindowCalls)
	}
}

func TestIsAnthropicWindowExceeded(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		window   string
		expected bool
	}{
		{
			name:     "utilization above 1.0",
			headers:  makeHeader("anthropic-ratelimit-unified-5h-utilization", "1.02"),
			window:   "5h",
			expected: true,
		},
		{
			name:     "utilization exactly 1.0",
			headers:  makeHeader("anthropic-ratelimit-unified-5h-utilization", "1.0"),
			window:   "5h",
			expected: true,
		},
		{
			name:     "utilization below 1.0",
			headers:  makeHeader("anthropic-ratelimit-unified-5h-utilization", "0.99"),
			window:   "5h",
			expected: false,
		},
		{
			name:     "surpassed-threshold true",
			headers:  makeHeader("anthropic-ratelimit-unified-7d-surpassed-threshold", "true"),
			window:   "7d",
			expected: true,
		},
		{
			name:     "surpassed-threshold True (case insensitive)",
			headers:  makeHeader("anthropic-ratelimit-unified-7d-surpassed-threshold", "True"),
			window:   "7d",
			expected: true,
		},
		{
			name:     "surpassed-threshold false",
			headers:  makeHeader("anthropic-ratelimit-unified-7d-surpassed-threshold", "false"),
			window:   "7d",
			expected: false,
		},
		{
			name:     "no headers",
			headers:  http.Header{},
			window:   "5h",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isAnthropicWindowExceeded(tc.headers, tc.window)
			if got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

// assertAnthropicResult is a test helper that verifies the result is non-nil and
// has the expected resetAt unix timestamp.
func assertAnthropicResult(t *testing.T, result *anthropic429Result, wantUnix int64) {
	t.Helper()
	if result == nil {
		t.Fatal("expected non-nil result")
		return // unreachable, but satisfies staticcheck SA5011
	}
	want := time.Unix(wantUnix, 0)
	if !result.resetAt.Equal(want) {
		t.Errorf("expected resetAt=%v, got %v", want, result.resetAt)
	}
}

func makeHeader(key, value string) http.Header {
	h := http.Header{}
	h.Set(key, value)
	return h
}
