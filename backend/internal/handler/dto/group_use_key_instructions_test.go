package dto

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupFromService_MapsUseKeyInstructionsForUserAndAdmin(t *testing.T) {
	group := &service.Group{
		ID:                 7,
		Name:               "vip-openai",
		Description:        "VIP",
		Platform:           service.PlatformOpenAI,
		Status:             service.StatusActive,
		UseKeyInstructions: "VIP 分组请优先使用内部 Codex 配置模板。",
	}

	userDTO := GroupFromService(group)
	adminDTO := GroupFromServiceAdmin(group)

	require.NotNil(t, userDTO)
	require.NotNil(t, adminDTO)
	require.Equal(t, "VIP 分组请优先使用内部 Codex 配置模板。", userDTO.UseKeyInstructions)
	require.Equal(t, "VIP 分组请优先使用内部 Codex 配置模板。", adminDTO.UseKeyInstructions)
}
