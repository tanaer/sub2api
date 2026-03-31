package service

import (
	"bytes"
	"net/http"
	"slices"
	"strings"
)

const maxModelRetryChainAttempts = 3

var modelFallbackDegradeKeywords = []string{
	"429",
	"529",
	"capacity",
	"concurrency",
	"overloaded",
	"rate limit",
	"rate_limit",
	"rate-limited",
	"rate limited",
	"resource exhausted",
	"resource_exhausted",
	"too many requests",
}

func buildModelRetryChain(account *Account, requestedModel string) []string {
	if account == nil {
		return compactUniqueModels([]string{requestedModel}, maxModelRetryChainAttempts)
	}
	return compactUniqueModels(account.ResolveModelRetryChain(requestedModel, maxModelRetryChainAttempts), maxModelRetryChainAttempts)
}

func compactUniqueModels(models []string, maxAttempts int) []string {
	if maxAttempts <= 0 {
		maxAttempts = maxModelRetryChainAttempts
	}

	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, min(len(models), maxAttempts))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, model)
		if len(out) >= maxAttempts {
			break
		}
	}
	return out
}

func shouldFallbackToNextModel(statusCode int, responseBody []byte) bool {
	switch statusCode {
	case http.StatusTooManyRequests, 529:
		return true
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return false
	}
	if statusCode < 500 {
		return false
	}
	body := strings.ToLower(string(bytes.ToLower(bytes.TrimSpace(responseBody))))
	if body == "" {
		return false
	}
	return slices.ContainsFunc(modelFallbackDegradeKeywords, func(keyword string) bool {
		return strings.Contains(body, keyword)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
