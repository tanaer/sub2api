package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/googleapi"
)

const clientFacingUnavailableModelMessage = "Requested model is unavailable"

type clientFacingUpstreamErrorFormat string

const (
	clientFacingUpstreamErrorFormatAnthropic clientFacingUpstreamErrorFormat = "anthropic"
	clientFacingUpstreamErrorFormatOpenAI    clientFacingUpstreamErrorFormat = "openai"
	clientFacingUpstreamErrorFormatGoogle    clientFacingUpstreamErrorFormat = "google"
)

func sanitizeClientFacingUpstreamErrorMessage(message string) (string, bool) {
	message = sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
	if message == "" {
		return "", false
	}
	if exposesUpstreamModelDetails(message) {
		return clientFacingUnavailableModelMessage, true
	}
	return message, false
}

func NormalizeClientFacingUpstreamErrorMessage(message string) string {
	if sanitized, redacted := sanitizeClientFacingUpstreamErrorMessage(message); redacted {
		return sanitized
	}
	return sanitizeUpstreamErrorMessage(strings.TrimSpace(message))
}

func normalizeClientFacingUpstreamErrorMessageWithSource(message string, sources ...string) string {
	for _, source := range append([]string{message}, sources...) {
		if sanitized, redacted := sanitizeClientFacingUpstreamErrorMessage(source); redacted {
			return sanitized
		}
	}
	return NormalizeClientFacingUpstreamErrorMessage(message)
}

func redactClientFacingUpstreamErrorBody(
	status int,
	body []byte,
	format clientFacingUpstreamErrorFormat,
) ([]byte, bool) {
	if !exposesUpstreamModelDetails(string(body)) {
		return body, false
	}

	switch format {
	case clientFacingUpstreamErrorFormatAnthropic:
		payload, _ := json.Marshal(map[string]any{
			"type": "error",
			"error": map[string]any{
				"type":    anthropicLikeErrorTypeForStatus(status),
				"message": clientFacingUnavailableModelMessage,
			},
		})
		return payload, true
	case clientFacingUpstreamErrorFormatGoogle:
		payload, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    status,
				"message": clientFacingUnavailableModelMessage,
				"status":  googleapi.HTTPStatusToGoogleStatus(status),
			},
		})
		return payload, true
	default:
		payload, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"type":    anthropicLikeErrorTypeForStatus(status),
				"message": clientFacingUnavailableModelMessage,
			},
		})
		return payload, true
	}
}

func exposesUpstreamModelDetails(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}

	if strings.Contains(normalized, "unsupported model") ||
		strings.Contains(normalized, "model unsupported") ||
		strings.Contains(normalized, "available models") ||
		strings.Contains(normalized, "supported models") {
		return true
	}

	if (strings.Contains(normalized, "model") || strings.Contains(normalized, "models")) &&
		(strings.Contains(normalized, "not supported") ||
			strings.Contains(normalized, "not found") ||
			strings.Contains(normalized, "not in whitelist") ||
			strings.Contains(normalized, "unavailable") ||
			strings.Contains(normalized, "available:")) {
		return true
	}

	return false
}

func anthropicLikeErrorTypeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	default:
		return "upstream_error"
	}
}
