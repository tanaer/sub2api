package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

const (
	recoveryServiceScanInterval  = 30 * time.Second
	recoveryServiceMaxConcurrent = 3
	recoveryProbeTimeout         = 30 * time.Second
)

// AccountThrottleRecoveryService 恢复探测后台服务
// 定期扫描被限流的账号，如果配置了 recovery_check_interval，
// 则发送探测请求，成功后自动解除限流。
type AccountThrottleRecoveryService struct {
	db               *sql.DB
	accountTestSvc   *AccountTestService
	rateLimitSvc     *RateLimitService
	tempUnschedCache TempUnschedCache

	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewAccountThrottleRecoveryService 创建恢复探测服务
func NewAccountThrottleRecoveryService(
	db *sql.DB,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	tempUnschedCache TempUnschedCache,
) *AccountThrottleRecoveryService {
	return &AccountThrottleRecoveryService{
		db:               db,
		accountTestSvc:   accountTestSvc,
		rateLimitSvc:     rateLimitSvc,
		tempUnschedCache: tempUnschedCache,
		stopCh:           make(chan struct{}),
	}
}

// throttledAccountInfo 从 DB 查询到的限流账号信息
type throttledAccountInfo struct {
	AccountID int64
	Reason    string
}

// Start 启动后台恢复探测循环
func (s *AccountThrottleRecoveryService) Start() {
	if s == nil || s.db == nil || s.accountTestSvc == nil {
		return
	}
	go s.loop()
	slog.Info("account_throttle_recovery_service_started", "scan_interval", recoveryServiceScanInterval)
}

// Stop 停止后台服务
func (s *AccountThrottleRecoveryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		slog.Info("account_throttle_recovery_service_stopped")
	})
}

func (s *AccountThrottleRecoveryService) loop() {
	ticker := time.NewTicker(recoveryServiceScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.scan()
		}
	}
}

func (s *AccountThrottleRecoveryService) scan() {
	ctx, cancel := context.WithTimeout(context.Background(), recoveryServiceScanInterval)
	defer cancel()

	// 查询所有当前被限流的账号（temp_unschedulable_until > NOW()）
	accounts, err := s.listThrottledAccounts(ctx)
	if err != nil {
		slog.Warn("throttle_recovery_scan_failed", "error", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	// 限制并发探测数
	sem := make(chan struct{}, recoveryServiceMaxConcurrent)
	var wg sync.WaitGroup

	for _, acc := range accounts {
		// 解析 reason JSON，提取 recovery_check_interval
		var state TempUnschedState
		if err := json.Unmarshal([]byte(acc.Reason), &state); err != nil {
			continue
		}
		if state.RecoveryCheckInterval <= 0 {
			continue
		}

		// 检查是否到了探测时间
		lastProbe := time.Unix(state.TriggeredAtUnix, 0)
		if elapsed := time.Since(lastProbe); elapsed < time.Duration(state.RecoveryCheckInterval)*time.Second {
			continue
		}

		accountID := acc.AccountID
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			s.probeAccount(accountID, state)
		}()
	}

	wg.Wait()
}

func (s *AccountThrottleRecoveryService) probeAccount(accountID int64, state TempUnschedState) {
	ctx, cancel := context.WithTimeout(context.Background(), recoveryProbeTimeout)
	defer cancel()

	slog.Info("throttle_recovery_probe_start", "account_id", accountID, "rule", state.ThrottleRuleName)

	result, err := s.accountTestSvc.RunTestBackground(ctx, accountID, "")
	if err != nil {
		slog.Info("throttle_recovery_probe_error", "account_id", accountID, "error", err)
		return
	}

	if result.Status == "success" {
		// 探测成功，解除限流
		slog.Info("throttle_recovery_probe_success", "account_id", accountID, "rule", state.ThrottleRuleName)
		if err := s.rateLimitSvc.ClearTempUnschedulable(ctx, accountID); err != nil {
			slog.Warn("throttle_recovery_clear_failed", "account_id", accountID, "error", err)
		}
	} else {
		slog.Info("throttle_recovery_probe_failed", "account_id", accountID, "error", result.ErrorMessage)
		// 更新 triggered_at 以推迟下次探测
		s.updateLastProbeTime(ctx, accountID, state)
	}
}

func (s *AccountThrottleRecoveryService) updateLastProbeTime(ctx context.Context, accountID int64, state TempUnschedState) {
	state.TriggeredAtUnix = time.Now().Unix()
	raw, err := json.Marshal(state)
	if err != nil {
		return
	}
	_, _ = s.db.ExecContext(ctx, `
		UPDATE accounts
		SET temp_unschedulable_reason = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND temp_unschedulable_until > NOW()
	`, string(raw), accountID)

	if s.tempUnschedCache != nil {
		_ = s.tempUnschedCache.SetTempUnsched(ctx, accountID, &state)
	}
}

func (s *AccountThrottleRecoveryService) listThrottledAccounts(ctx context.Context) ([]throttledAccountInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(temp_unschedulable_reason, '')
		FROM accounts
		WHERE deleted_at IS NULL
			AND status = 'active'
			AND temp_unschedulable_until IS NOT NULL
			AND temp_unschedulable_until > NOW()
			AND temp_unschedulable_reason LIKE '%recovery_check_interval%'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []throttledAccountInfo
	for rows.Next() {
		var info throttledAccountInfo
		if err := rows.Scan(&info.AccountID, &info.Reason); err != nil {
			continue
		}
		result = append(result, info)
	}
	return result, rows.Err()
}
