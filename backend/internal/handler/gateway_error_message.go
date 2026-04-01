package handler

import "strings"

const noAvailableAccountsMappingHint = "，模型映射错误，使用了错误的模型请求，请查看发货教程！"

func normalizeNoAvailableAccountsErrorMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if !strings.Contains(trimmed, "No available accounts: no available accounts") {
		return message
	}
	if strings.Contains(trimmed, noAvailableAccountsMappingHint) {
		return message
	}
	return trimmed + noAvailableAccountsMappingHint
}
