//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type userGroupRateRepoStubForAuthQuota struct {
	quota *service.UserGroupRequestQuota
	err   error
}

func (s *userGroupRateRepoStubForAuthQuota) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

func (s *userGroupRateRepoStubForAuthQuota) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	return nil, nil
}

func (s *userGroupRateRepoStubForAuthQuota) GetByGroupID(context.Context, int64) ([]service.UserGroupRateEntry, error) {
	return nil, nil
}

func (s *userGroupRateRepoStubForAuthQuota) SyncUserGroupRates(context.Context, int64, map[int64]*float64) error {
	return nil
}

func (s *userGroupRateRepoStubForAuthQuota) SyncGroupRateMultipliers(context.Context, int64, []service.GroupRateMultiplierInput) error {
	return nil
}

func (s *userGroupRateRepoStubForAuthQuota) DeleteByGroupID(context.Context, int64) error {
	return nil
}

func (s *userGroupRateRepoStubForAuthQuota) DeleteByUserID(context.Context, int64) error {
	return nil
}

func (s *userGroupRateRepoStubForAuthQuota) GetRequestQuotasByUserID(context.Context, int64) (map[int64]int64, error) {
	return map[int64]int64{}, nil
}

func (s *userGroupRateRepoStubForAuthQuota) GetRequestQuotaByUserAndGroup(context.Context, int64, int64) (*service.UserGroupRequestQuota, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.quota == nil {
		return nil, nil
	}
	clone := *s.quota
	return &clone, nil
}

func (s *userGroupRateRepoStubForAuthQuota) SyncUserGroupRequestQuotas(context.Context, int64, map[int64]*int64) error {
	return nil
}

func (s *userGroupRateRepoStubForAuthQuota) IncrementRequestQuotaUsed(context.Context, int64, int64, int64) (bool, error) {
	return false, nil
}

func (s *userGroupRateRepoStubForAuthQuota) CreateRequestQuotaGrant(context.Context, *service.UserGroupRequestQuotaGrant) error {
	return nil
}

func TestAPIKeyAuth_AllowsUserGroupRequestQuotaWithoutBalance(t *testing.T) {
	user := &service.User{
		ID:          21,
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     0,
		Concurrency: 1,
	}
	groupID := int64(301)
	apiKey := &service.APIKey{
		ID:      501,
		UserID:  user.ID,
		Key:     "group-rq-allow",
		Status:  service.StatusActive,
		User:    user,
		GroupID: &groupID,
		Group: &service.Group{
			ID:               groupID,
			Name:             "glm-group",
			Status:           service.StatusActive,
			SubscriptionType: service.SubscriptionTypeStandard,
		},
	}

	apiKeyRepo := &stubApiKeyRepo{
		getByKey: func(ctx context.Context, key string) (*service.APIKey, error) {
			if key != apiKey.Key {
				return nil, service.ErrAPIKeyNotFound
			}
			clone := *apiKey
			return &clone, nil
		},
	}

	cfg := &config.Config{RunMode: config.RunModeStandard}
	apiKeyService := service.NewAPIKeyService(
		apiKeyRepo,
		nil,
		nil,
		nil,
		&userGroupRateRepoStubForAuthQuota{
			quota: &service.UserGroupRequestQuota{
				RequestQuota:     8,
				RequestQuotaUsed: 3,
			},
		},
		nil,
		cfg,
	)
	router := newAuthTestRouter(apiKeyService, nil, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("x-api-key", apiKey.Key)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuth_UserGroupRequestQuotaExhaustedFallsBackToBalanceCheck(t *testing.T) {
	user := &service.User{
		ID:          22,
		Role:        service.RoleUser,
		Status:      service.StatusActive,
		Balance:     0,
		Concurrency: 1,
	}
	groupID := int64(302)
	apiKey := &service.APIKey{
		ID:      502,
		UserID:  user.ID,
		Key:     "group-rq-exhausted",
		Status:  service.StatusActive,
		User:    user,
		GroupID: &groupID,
		Group: &service.Group{
			ID:               groupID,
			Name:             "glm-group",
			Status:           service.StatusActive,
			SubscriptionType: service.SubscriptionTypeStandard,
		},
	}

	apiKeyRepo := &stubApiKeyRepo{
		getByKey: func(ctx context.Context, key string) (*service.APIKey, error) {
			if key != apiKey.Key {
				return nil, service.ErrAPIKeyNotFound
			}
			clone := *apiKey
			return &clone, nil
		},
	}

	cfg := &config.Config{RunMode: config.RunModeStandard}
	apiKeyService := service.NewAPIKeyService(
		apiKeyRepo,
		nil,
		nil,
		nil,
		&userGroupRateRepoStubForAuthQuota{
			quota: &service.UserGroupRequestQuota{
				RequestQuota:     6,
				RequestQuotaUsed: 6,
			},
		},
		nil,
		cfg,
	)
	router := newAuthTestRouter(apiKeyService, nil, cfg)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/t", nil)
	req.Header.Set("x-api-key", apiKey.Key)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "INSUFFICIENT_BALANCE")
}
