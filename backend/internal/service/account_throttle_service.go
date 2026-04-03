package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/tidwall/gjson"
)

// AccountThrottleRepository 账户限流规则数据访问接口
type AccountThrottleRepository interface {
	List(ctx context.Context) ([]*model.AccountThrottleRule, error)
	GetByID(ctx context.Context, id int64) (*model.AccountThrottleRule, error)
	Create(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error)
	Update(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error)
	Delete(ctx context.Context, id int64) error
}

// AccountThrottleCache 账户限流规则缓存接口
type AccountThrottleCache interface {
	Get(ctx context.Context) ([]*model.AccountThrottleRule, bool)
	Set(ctx context.Context, rules []*model.AccountThrottleRule) error
	Invalidate(ctx context.Context) error
	NotifyUpdate(ctx context.Context) error
	SubscribeUpdates(ctx context.Context, handler func())
}

// AccountThrottleCounterCache 累计限流计数器缓存接口
type AccountThrottleCounterCache interface {
	// IncrementThrottleCount 增加指定账户+规则的计数，返回当前值
	IncrementThrottleCount(ctx context.Context, accountID int64, ruleID int64, windowSeconds int) (int64, error)
}

// AccountThrottleService 账户智能限流服务
type AccountThrottleService struct {
	repo         AccountThrottleRepository
	cache        AccountThrottleCache
	counterCache AccountThrottleCounterCache

	localCache   []*cachedThrottleRule
	localCacheMu sync.RWMutex
}

// cachedThrottleRule 预计算的规则缓存
type cachedThrottleRule struct {
	*model.AccountThrottleRule
	lowerKeywords  []string
	lowerPlatforms []string
}

// NewAccountThrottleService 创建账户限流服务
func NewAccountThrottleService(
	repo AccountThrottleRepository,
	cache AccountThrottleCache,
	counterCache AccountThrottleCounterCache,
) *AccountThrottleService {
	svc := &AccountThrottleService{
		repo:         repo,
		cache:        cache,
		counterCache: counterCache,
	}

	ctx := context.Background()
	if err := svc.reloadRulesFromDB(ctx); err != nil {
		logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to load rules from DB on startup: %v", err)
		if fallbackErr := svc.refreshLocalCache(ctx); fallbackErr != nil {
			logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to load rules from cache fallback on startup: %v", fallbackErr)
		}
	}

	if cache != nil {
		cache.SubscribeUpdates(ctx, func() {
			if err := svc.refreshLocalCache(context.Background()); err != nil {
				logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to refresh cache on notification: %v", err)
			}
		})
	}

	return svc
}

// List 获取所有规则
func (s *AccountThrottleService) List(ctx context.Context) ([]*model.AccountThrottleRule, error) {
	return s.repo.List(ctx)
}

// GetByID 根据 ID 获取规则
func (s *AccountThrottleService) GetByID(ctx context.Context, id int64) (*model.AccountThrottleRule, error) {
	return s.repo.GetByID(ctx, id)
}

// Create 创建规则
func (s *AccountThrottleService) Create(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, rule)
	if err != nil {
		return nil, err
	}

	refreshCtx, cancel := s.newCacheRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)

	return created, nil
}

// Update 更新规则
func (s *AccountThrottleService) Update(ctx context.Context, rule *model.AccountThrottleRule) (*model.AccountThrottleRule, error) {
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, rule)
	if err != nil {
		return nil, err
	}

	refreshCtx, cancel := s.newCacheRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)

	return updated, nil
}

// Delete 删除规则
func (s *AccountThrottleService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	refreshCtx, cancel := s.newCacheRefreshContext()
	defer cancel()
	s.invalidateAndNotify(refreshCtx)

	return nil
}

// ThrottleMatchResult 限流匹配结果
type ThrottleMatchResult struct {
	Matched        bool
	Rule           *model.AccountThrottleRule
	MatchedKeyword string
	UntilTime      time.Time
}

// MatchAndThrottle 匹配限流规则并执行限流
// statusCode 用于 error_codes 过滤（与 per-account temp_unschedulable_rules 的 error_code 对齐）
// 返回是否匹配到规则以及限流截止时间
func (s *AccountThrottleService) MatchAndThrottle(ctx context.Context, accountID int64, platform string, statusCode int, responseBody []byte) *ThrottleMatchResult {
	rules := s.getCachedRules()
	if len(rules) == 0 {
		return nil
	}

	lowerPlatform := strings.ToLower(platform)
	var bodyLower string
	var bodyLowerDone bool

	// 提取响应体中的业务错误码（如讯飞 11200），用于 error_codes 匹配
	bodyErrorCodes := extractBodyErrorCodes(responseBody)

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if !s.platformMatches(rule, lowerPlatform) {
			continue
		}
		if !s.errorCodeMatches(rule, statusCode, bodyErrorCodes) {
			continue
		}

		matchedKeyword := s.matchKeywords(rule, responseBody, &bodyLower, &bodyLowerDone)
		if matchedKeyword == "" {
			continue
		}

		// 关键词匹配成功，检查触发条件
		if rule.TriggerMode == model.ThrottleTriggerAccumulated {
			// 累计模式：递增计数器，检查是否达到阈值
			if s.counterCache != nil {
				count, err := s.counterCache.IncrementThrottleCount(ctx, accountID, rule.ID, rule.AccumulatedWindow)
				if err != nil {
					logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to increment counter: account=%d rule=%d err=%v", accountID, rule.ID, err)
					continue
				}
				if count < int64(rule.AccumulatedCount) {
					continue // 未达到阈值
				}
			}
		}

		// 触发限流，计算截止时间
		until := s.calculateUntilTime(rule.AccountThrottleRule)

		return &ThrottleMatchResult{
			Matched:        true,
			Rule:           rule.AccountThrottleRule,
			MatchedKeyword: matchedKeyword,
			UntilTime:      until,
		}
	}

	return nil
}

// calculateUntilTime 根据规则计算限流截止时间
func (s *AccountThrottleService) calculateUntilTime(rule *model.AccountThrottleRule) time.Time {
	now := time.Now()
	if rule.ActionType == model.ThrottleActionScheduledRecovery {
		// 定期恢复：计算下一个恢复时刻
		targetHour := rule.ActionRecoverHour
		next := time.Date(now.Year(), now.Month(), now.Day(), targetHour, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next
	}
	// 时长限流
	return now.Add(time.Duration(rule.ActionDuration) * time.Second)
}

// matchKeywords 匹配关键词
func (s *AccountThrottleService) matchKeywords(rule *cachedThrottleRule, body []byte, bodyLower *string, bodyLowerDone *bool) string {
	if len(rule.lowerKeywords) == 0 {
		return ""
	}

	bl := ensureBodyLower(body, bodyLower, bodyLowerDone)
	if bl == "" {
		return ""
	}

	if rule.MatchMode == model.ThrottleMatchExact {
		for i, kw := range rule.lowerKeywords {
			if bl == kw {
				return rule.Keywords[i]
			}
		}
		return ""
	}

	// contains 模式
	for i, kw := range rule.lowerKeywords {
		if strings.Contains(bl, kw) {
			return rule.Keywords[i]
		}
	}
	return ""
}

// errorCodeMatches 检查错误码是否匹配（空列表=匹配所有）
// 同时检查 HTTP 状态码和响应体中的业务错误码
func (s *AccountThrottleService) errorCodeMatches(rule *cachedThrottleRule, statusCode int, bodyErrorCodes []int) bool {
	if len(rule.ErrorCodes) == 0 {
		return true
	}
	for _, code := range rule.ErrorCodes {
		if code == statusCode {
			return true
		}
		for _, bodyCode := range bodyErrorCodes {
			if code == bodyCode {
				return true
			}
		}
	}
	return false
}

// extractBodyErrorCodes 从响应体 JSON 中提取业务错误码
// 支持常见格式：{"code":11200}, {"error":{"code":1302}}, {"header":{"code":11200}}
func extractBodyErrorCodes(body []byte) []int {
	if len(body) == 0 {
		return nil
	}

	var codes []int
	seen := make(map[int]bool)

	paths := []string{
		"code",
		"error.code",
		"header.code",
	}

	for _, path := range paths {
		result := gjson.GetBytes(body, path)
		if !result.Exists() {
			continue
		}
		if c, err := strconv.Atoi(strings.TrimSpace(result.Raw)); err == nil && c > 599 && !seen[c] {
			// 只收集非标准 HTTP 状态码（>599），避免与 HTTP 状态码重复
			codes = append(codes, c)
			seen[c] = true
		}
		// 有些上游把 code 作为字符串返回
		if s := strings.TrimSpace(result.String()); s != "" {
			if c, err := strconv.Atoi(s); err == nil && c > 599 && !seen[c] {
				codes = append(codes, c)
				seen[c] = true
			}
		}
	}

	return codes
}

// platformMatches 检查平台是否匹配
func (s *AccountThrottleService) platformMatches(rule *cachedThrottleRule, lowerPlatform string) bool {
	if len(rule.lowerPlatforms) == 0 {
		return true
	}
	for _, p := range rule.lowerPlatforms {
		if p == lowerPlatform {
			return true
		}
	}
	return false
}

// getCachedRules 获取缓存的规则
func (s *AccountThrottleService) getCachedRules() []*cachedThrottleRule {
	s.localCacheMu.RLock()
	rules := s.localCache
	s.localCacheMu.RUnlock()

	if rules != nil {
		return rules
	}

	ctx := context.Background()
	if err := s.refreshLocalCache(ctx); err != nil {
		logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to refresh cache: %v", err)
		return nil
	}

	s.localCacheMu.RLock()
	defer s.localCacheMu.RUnlock()
	return s.localCache
}

func (s *AccountThrottleService) refreshLocalCache(ctx context.Context) error {
	if s.cache != nil {
		if rules, ok := s.cache.Get(ctx); ok {
			s.setLocalCache(rules)
			return nil
		}
	}
	return s.reloadRulesFromDB(ctx)
}

func (s *AccountThrottleService) reloadRulesFromDB(ctx context.Context) error {
	rules, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, rules); err != nil {
			logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to set cache: %v", err)
		}
	}

	s.setLocalCache(rules)
	return nil
}

func (s *AccountThrottleService) setLocalCache(rules []*model.AccountThrottleRule) {
	cached := make([]*cachedThrottleRule, len(rules))
	for i, r := range rules {
		cr := &cachedThrottleRule{AccountThrottleRule: r}
		if len(r.Keywords) > 0 {
			cr.lowerKeywords = make([]string, len(r.Keywords))
			for j, kw := range r.Keywords {
				cr.lowerKeywords[j] = strings.ToLower(kw)
			}
		}
		if len(r.Platforms) > 0 {
			cr.lowerPlatforms = make([]string, len(r.Platforms))
			for j, p := range r.Platforms {
				cr.lowerPlatforms[j] = strings.ToLower(p)
			}
		}
		cached[i] = cr
	}

	sort.Slice(cached, func(i, j int) bool {
		return cached[i].Priority < cached[j].Priority
	})

	s.localCacheMu.Lock()
	s.localCache = cached
	s.localCacheMu.Unlock()
}

func (s *AccountThrottleService) clearLocalCache() {
	s.localCacheMu.Lock()
	s.localCache = nil
	s.localCacheMu.Unlock()
}

func (s *AccountThrottleService) newCacheRefreshContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

func (s *AccountThrottleService) invalidateAndNotify(ctx context.Context) {
	if s.cache != nil {
		if err := s.cache.Invalidate(ctx); err != nil {
			logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to invalidate cache: %v", err)
		}
	}

	if err := s.reloadRulesFromDB(ctx); err != nil {
		logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to refresh local cache: %v", err)
		s.clearLocalCache()
	}

	if s.cache != nil {
		if err := s.cache.NotifyUpdate(ctx); err != nil {
			logger.LegacyPrintf("service.account_throttle", "[AccountThrottleService] Failed to notify cache update: %v", err)
		}
	}
}

// FormatThrottleReason 格式化限流原因（用于存储到 TempUnschedState）
func FormatThrottleReason(rule *model.AccountThrottleRule, matchedKeyword string) string {
	return fmt.Sprintf("throttle_rule[%s]: matched keyword '%s'", rule.Name, matchedKeyword)
}
