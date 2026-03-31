//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type userRepoStubForGroupRequestQuota struct {
	user    *User
	updated *User
}

func (s *userRepoStubForGroupRequestQuota) Create(context.Context, *User) error {
	panic("unexpected Create call")
}

func (s *userRepoStubForGroupRequestQuota) GetByID(context.Context, int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	clone := *s.user
	return &clone, nil
}

func (s *userRepoStubForGroupRequestQuota) GetByEmail(context.Context, string) (*User, error) {
	panic("unexpected GetByEmail call")
}

func (s *userRepoStubForGroupRequestQuota) GetFirstAdmin(context.Context) (*User, error) {
	panic("unexpected GetFirstAdmin call")
}

func (s *userRepoStubForGroupRequestQuota) Update(_ context.Context, user *User) error {
	clone := *user
	s.updated = &clone
	return nil
}

func (s *userRepoStubForGroupRequestQuota) Delete(context.Context, int64) error {
	panic("unexpected Delete call")
}

func (s *userRepoStubForGroupRequestQuota) List(context.Context, pagination.PaginationParams) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *userRepoStubForGroupRequestQuota) ListWithFilters(context.Context, pagination.PaginationParams, UserListFilters) ([]User, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *userRepoStubForGroupRequestQuota) UpdateBalance(context.Context, int64, float64) error {
	panic("unexpected UpdateBalance call")
}

func (s *userRepoStubForGroupRequestQuota) DeductBalance(context.Context, int64, float64) error {
	panic("unexpected DeductBalance call")
}

func (s *userRepoStubForGroupRequestQuota) UpdateConcurrency(context.Context, int64, int) error {
	panic("unexpected UpdateConcurrency call")
}

func (s *userRepoStubForGroupRequestQuota) ExistsByEmail(context.Context, string) (bool, error) {
	panic("unexpected ExistsByEmail call")
}

func (s *userRepoStubForGroupRequestQuota) RemoveGroupFromAllowedGroups(context.Context, int64) (int64, error) {
	panic("unexpected RemoveGroupFromAllowedGroups call")
}

func (s *userRepoStubForGroupRequestQuota) RemoveGroupFromUserAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected RemoveGroupFromUserAllowedGroups call")
}

func (s *userRepoStubForGroupRequestQuota) AddGroupToAllowedGroups(context.Context, int64, int64) error {
	panic("unexpected AddGroupToAllowedGroups call")
}

func (s *userRepoStubForGroupRequestQuota) UpdateTotpSecret(context.Context, int64, *string) error {
	panic("unexpected UpdateTotpSecret call")
}

func (s *userRepoStubForGroupRequestQuota) EnableTotp(context.Context, int64) error {
	panic("unexpected EnableTotp call")
}

func (s *userRepoStubForGroupRequestQuota) DisableTotp(context.Context, int64) error {
	panic("unexpected DisableTotp call")
}

type userGroupRateRepoStubForRequestQuota struct {
	requestQuotas map[int64]int64

	gotUserID  int64
	syncUserID int64
	synced     map[int64]*int64
}

func (s *userGroupRateRepoStubForRequestQuota) GetByUserID(context.Context, int64) (map[int64]float64, error) {
	return map[int64]float64{}, nil
}

func (s *userGroupRateRepoStubForRequestQuota) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	return nil, nil
}

func (s *userGroupRateRepoStubForRequestQuota) GetByGroupID(context.Context, int64) ([]UserGroupRateEntry, error) {
	return nil, nil
}

func (s *userGroupRateRepoStubForRequestQuota) SyncUserGroupRates(context.Context, int64, map[int64]*float64) error {
	return nil
}

func (s *userGroupRateRepoStubForRequestQuota) SyncGroupRateMultipliers(context.Context, int64, []GroupRateMultiplierInput) error {
	return nil
}

func (s *userGroupRateRepoStubForRequestQuota) DeleteByGroupID(context.Context, int64) error {
	return nil
}

func (s *userGroupRateRepoStubForRequestQuota) DeleteByUserID(context.Context, int64) error {
	return nil
}

func (s *userGroupRateRepoStubForRequestQuota) GetRequestQuotasByUserID(_ context.Context, userID int64) (map[int64]int64, error) {
	s.gotUserID = userID
	out := make(map[int64]int64, len(s.requestQuotas))
	for groupID, quota := range s.requestQuotas {
		out[groupID] = quota
	}
	return out, nil
}

func (s *userGroupRateRepoStubForRequestQuota) GetRequestQuotaByUserAndGroup(context.Context, int64, int64) (*UserGroupRequestQuota, error) {
	return nil, nil
}

func (s *userGroupRateRepoStubForRequestQuota) SyncUserGroupRequestQuotas(_ context.Context, userID int64, quotas map[int64]*int64) error {
	s.syncUserID = userID
	s.synced = quotas
	return nil
}

func (s *userGroupRateRepoStubForRequestQuota) IncrementRequestQuotaUsed(context.Context, int64, int64, int64) (bool, error) {
	return false, nil
}

func (s *userGroupRateRepoStubForRequestQuota) CreateRequestQuotaGrant(context.Context, *UserGroupRequestQuotaGrant) error {
	return nil
}

func TestAdminService_GetUser_LoadsGroupRequestQuotas(t *testing.T) {
	userRepo := &userRepoStubForGroupRequestQuota{
		user: &User{
			ID:     7,
			Email:  "quota@example.com",
			Status: StatusActive,
		},
	}
	quotaRepo := &userGroupRateRepoStubForRequestQuota{
		requestQuotas: map[int64]int64{
			11: 20,
			22: 5,
		},
	}
	svc := &adminServiceImpl{
		userRepo:          userRepo,
		userGroupRateRepo: quotaRepo,
	}

	user, err := svc.GetUser(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), quotaRepo.gotUserID)
	require.Equal(t, int64(20), user.GroupRequestQuotas[11])
	require.Equal(t, int64(5), user.GroupRequestQuotas[22])
}

func TestAdminService_UpdateUser_SyncsGroupRequestQuotas(t *testing.T) {
	userRepo := &userRepoStubForGroupRequestQuota{
		user: &User{
			ID:          9,
			Email:       "sync@example.com",
			Role:        RoleUser,
			Status:      StatusActive,
			Concurrency: 1,
		},
	}
	quotaRepo := &userGroupRateRepoStubForRequestQuota{}
	svc := &adminServiceImpl{
		userRepo:          userRepo,
		userGroupRateRepo: quotaRepo,
	}

	updated, err := svc.UpdateUser(context.Background(), 9, &UpdateUserInput{
		GroupRequestQuotas: map[int64]*int64{
			11: int64PtrForGroupRequestQuota(30),
			22: nil,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, int64(9), quotaRepo.syncUserID)
	require.Equal(t, int64(30), *quotaRepo.synced[11])
	require.Nil(t, quotaRepo.synced[22])
}

func TestAdminService_UpdateUser_InvalidatesAuthCacheWhenGroupRequestQuotasChange(t *testing.T) {
	userRepo := &userRepoStubForGroupRequestQuota{
		user: &User{
			ID:          9,
			Email:       "sync@example.com",
			Role:        RoleUser,
			Status:      StatusActive,
			Concurrency: 1,
		},
	}
	quotaRepo := &userGroupRateRepoStubForRequestQuota{}
	invalidator := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		userRepo:             userRepo,
		userGroupRateRepo:    quotaRepo,
		authCacheInvalidator: invalidator,
	}

	_, err := svc.UpdateUser(context.Background(), 9, &UpdateUserInput{
		GroupRequestQuotas: map[int64]*int64{
			11: int64PtrForGroupRequestQuota(30),
		},
	})
	require.NoError(t, err)
	require.Equal(t, []int64{9}, invalidator.userIDs)
}

func int64PtrForGroupRequestQuota(v int64) *int64 {
	return &v
}
