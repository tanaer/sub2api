package service

import (
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func applyOpenAIModelCandidateToBody(body []byte, candidate string) ([]byte, string, error) {
	model := candidate
	if normalized := normalizeCodexModel(candidate); normalized != "" {
		model = normalized
	}

	nextBody, err := sjson.SetBytes(body, "model", model)
	if err != nil {
		return nil, "", fmt.Errorf("set openai fallback model: %w", err)
	}

	if !SupportsVerbosity(model) && gjson.GetBytes(nextBody, "text.verbosity").Exists() {
		nextBody, err = sjson.DeleteBytes(nextBody, "text.verbosity")
		if err != nil {
			return nil, "", fmt.Errorf("remove openai fallback verbosity: %w", err)
		}
	}

	return nextBody, model, nil
}

func applyOpenAIModelCandidateToRequestMap(reqBody map[string]any, candidate string) (map[string]any, string, error) {
	if reqBody == nil {
		return nil, "", fmt.Errorf("openai request body is nil")
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("marshal openai request body: %w", err)
	}
	nextBody, model, err := applyOpenAIModelCandidateToBody(body, candidate)
	if err != nil {
		return nil, "", err
	}

	var nextReqBody map[string]any
	if err := json.Unmarshal(nextBody, &nextReqBody); err != nil {
		return nil, "", fmt.Errorf("unmarshal openai fallback request body: %w", err)
	}
	return nextReqBody, model, nil
}
