//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAdminService_AdminUpdateAPIKeyRequestQuota_SetQuotaAndResetUsage(t *testing.T) {
	repo := &apiKeyRepoStubForGroupUpdate{
		key: &APIKey{
			ID:               1,
			Key:              "sk-request-quota",
			RequestQuota:     8,
			RequestQuotaUsed: 3,
		},
	}
	cache := &authCacheInvalidatorStub{}
	svc := &adminServiceImpl{
		apiKeyRepo:           repo,
		authCacheInvalidator: cache,
	}

	updated, err := svc.AdminUpdateAPIKeyRequestQuota(context.Background(), 1, int64Ptr(20), true)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, int64(20), updated.RequestQuota)
	require.Equal(t, int64(0), updated.RequestQuotaUsed)
	require.NotNil(t, repo.updated)
	require.Equal(t, int64(20), repo.updated.RequestQuota)
	require.Equal(t, int64(0), repo.updated.RequestQuotaUsed)
	require.Equal(t, []string{"sk-request-quota"}, cache.keys)
}

func TestAdminService_AdminUpdateAPIKeyRequestQuota_RejectsNegativeQuota(t *testing.T) {
	repo := &apiKeyRepoStubForGroupUpdate{
		key: &APIKey{ID: 1, Key: "sk-request-quota"},
	}
	svc := &adminServiceImpl{apiKeyRepo: repo}

	_, err := svc.AdminUpdateAPIKeyRequestQuota(context.Background(), 1, int64Ptr(-1), false)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}
