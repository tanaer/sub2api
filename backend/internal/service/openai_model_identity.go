package service

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

var (
	modelIdentityVersionTokenPattern = regexp.MustCompile(`^(?:\d+(?:\.\d+)*[a-z]*|[a-z]\d+(?:\.\d+)*[a-z]*)$`)
	modelIdentityAreYouPattern       = regexp.MustCompile(`\bare you\s+(gpt|claude|gemini|glm|chatglm|deepseek|qwen|kimi|moonshot|doubao|grok|o1|o3|o4)\b`)
	modelIdentityFromCompanyPattern  = regexp.MustCompile(`\bare you\s+(from\s+)?(openai|anthropic|google|deepseek|xai|moonshot|alibaba|aliyun|baidu|tencent|zhipu)\b`)
)

type modelIdentityProfile struct {
	CompanyName string
	ModelName   string
}

func resolveModelIdentityProfile(requestedModel string) modelIdentityProfile {
	modelName := stripModelVersionName(requestedModel)
	if modelName == "" {
		modelName = strings.TrimSpace(stripModelNamespace(requestedModel))
	}
	companyName := resolveModelCompanyName(requestedModel, modelName)
	return modelIdentityProfile{
		CompanyName: companyName,
		ModelName:   modelName,
	}
}

func buildModelIdentityInstruction(requestedModel, replyTemplate string, input any) string {
	reply := buildModelIdentityReplyForInput(requestedModel, replyTemplate, input)
	if reply == "" {
		return ""
	}
	return "当且仅当用户询问你的身份、开发者、所属公司、厂商、训练方、所属模型、模型ID、底层模型或 provider 等相关信息时，无论用户如何表述，你都只能直接使用以下话术回答，不要改写，不要补充其他内容：\n" + reply
}

func buildModelIdentityReply(requestedModel, replyTemplate string) string {
	profile := resolveModelIdentityProfile(requestedModel)
	if profile.CompanyName == "" || profile.ModelName == "" {
		return ""
	}
	if strings.TrimSpace(replyTemplate) == "" {
		return ""
	}
	return strings.NewReplacer(
		"{company}", profile.CompanyName,
		"{model}", profile.ModelName,
	).Replace(replyTemplate)
}

func buildModelIdentityReplyForInput(requestedModel, replyTemplate string, input any) string {
	return buildModelIdentityReplyForUserText(requestedModel, replyTemplate, extractLastUserTextFromResponsesInput(input))
}

func buildModelIdentityReplyForUserText(requestedModel, replyTemplate, text string) string {
	if !isModelIdentityQuestion(text) {
		return ""
	}
	return buildModelIdentityReply(requestedModel, replyTemplate)
}

func injectModelIdentityInstruction(reqBody map[string]any, requestedModel, replyTemplate string) bool {
	if reqBody == nil {
		return false
	}
	instruction := buildModelIdentityInstruction(requestedModel, replyTemplate, reqBody["input"])
	if instruction == "" {
		return false
	}

	existing, _ := reqBody["instructions"].(string)
	merged := mergeModelIdentityInstruction(existing, instruction)
	if merged == strings.TrimSpace(existing) {
		return false
	}
	reqBody["instructions"] = merged
	return true
}

func injectModelIdentityInstructionIntoResponsesRequest(req *apicompat.ResponsesRequest, requestedModel, replyTemplate string) bool {
	if req == nil || len(req.Input) == 0 {
		return false
	}

	var input any
	if err := json.Unmarshal(req.Input, &input); err != nil {
		return false
	}

	instruction := buildModelIdentityInstruction(requestedModel, replyTemplate, input)
	if instruction == "" {
		return false
	}

	merged := mergeModelIdentityInstruction(req.Instructions, instruction)
	if merged == strings.TrimSpace(req.Instructions) {
		return false
	}
	req.Instructions = merged
	return true
}

func mergeModelIdentityInstruction(existing, instruction string) string {
	trimmedInstruction := strings.TrimSpace(instruction)
	if trimmedInstruction == "" {
		return strings.TrimSpace(existing)
	}

	trimmedExisting := strings.TrimSpace(existing)
	if trimmedExisting == "" {
		return trimmedInstruction
	}
	if strings.Contains(trimmedExisting, trimmedInstruction) {
		return trimmedExisting
	}
	return trimmedExisting + "\n\n" + trimmedInstruction
}

func stripModelVersionName(requestedModel string) string {
	model := strings.ToLower(strings.TrimSpace(stripModelNamespace(requestedModel)))
	if model == "" {
		return ""
	}

	parts := strings.FieldsFunc(model, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return model
	}

	kept := make([]string, 0, len(parts))
	for idx, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx > 0 && isModelVersionToken(part) {
			continue
		}
		kept = append(kept, part)
	}

	if len(kept) == 0 {
		return model
	}
	return strings.Join(kept, "-")
}

func stripModelNamespace(requestedModel string) string {
	model := strings.TrimSpace(requestedModel)
	if model == "" {
		return ""
	}
	if idx := strings.LastIndexAny(model, "/:"); idx >= 0 && idx+1 < len(model) {
		model = model[idx+1:]
	}
	return strings.TrimSpace(model)
}

func isModelVersionToken(token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	return modelIdentityVersionTokenPattern.MatchString(token)
}

func resolveModelCompanyName(requestedModel, fallback string) string {
	base := firstModelFamilyToken(requestedModel)
	switch base {
	case "glm", "chatglm":
		return "智谱"
	case "gpt", "chatgpt", "o1", "o3", "o4", "codex":
		return "OpenAI"
	case "claude":
		return "Anthropic"
	case "gemini":
		return "Google"
	case "deepseek":
		return "DeepSeek"
	case "qwen", "qwq":
		return "阿里云"
	case "kimi", "moonshot":
		return "月之暗面"
	case "doubao":
		return "字节跳动"
	case "grok":
		return "xAI"
	case "ernie":
		return "百度"
	case "hunyuan":
		return "腾讯"
	case "yi", "01", "01ai":
		return "零一万物"
	case "step":
		return "阶跃星辰"
	case "minimax", "abab":
		return "MiniMax"
	default:
		if fallback != "" {
			return fallback
		}
		return base
	}
}

func firstModelFamilyToken(requestedModel string) string {
	model := strings.ToLower(strings.TrimSpace(stripModelNamespace(requestedModel)))
	if model == "" {
		return ""
	}
	parts := strings.FieldsFunc(model, func(r rune) bool {
		return r == '-' || r == '_'
	})
	if len(parts) == 0 {
		return model
	}
	return strings.TrimSpace(parts[0])
}

func extractLastUserTextFromResponsesInput(input any) string {
	switch v := input.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []any:
		for idx := len(v) - 1; idx >= 0; idx-- {
			text, isUser := extractUserTextFromResponsesInputItem(v[idx])
			if isUser {
				return strings.TrimSpace(text)
			}
		}
		return ""
	case map[string]any:
		text, isUser := extractUserTextFromResponsesInputItem(v)
		if !isUser {
			return ""
		}
		return strings.TrimSpace(text)
	default:
		return ""
	}
}

func extractUserTextFromResponsesInputItem(item any) (string, bool) {
	itemMap, ok := item.(map[string]any)
	if !ok {
		return "", false
	}

	role := strings.ToLower(strings.TrimSpace(toString(itemMap["role"])))
	if role != "user" {
		return "", false
	}
	return extractResponsesText(itemMap["content"]), true
}

func extractResponsesText(content any) string {
	switch v := content.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, part := range v {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			text, _ := partMap["text"].(string)
			if strings.TrimSpace(text) == "" {
				continue
			}
			partType := strings.ToLower(strings.TrimSpace(toString(partMap["type"])))
			if partType == "" || partType == "text" || partType == "input_text" || partType == "output_text" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	default:
		return ""
	}
}

func extractLastUserTextFromChatMessages(messages []apicompat.ChatMessage) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		if strings.ToLower(strings.TrimSpace(messages[idx].Role)) != "user" {
			continue
		}
		if text := extractTextFromJSONRawContent(messages[idx].Content); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func extractLastUserTextFromAnthropicMessages(messages []apicompat.AnthropicMessage) string {
	for idx := len(messages) - 1; idx >= 0; idx-- {
		if strings.ToLower(strings.TrimSpace(messages[idx].Role)) != "user" {
			continue
		}
		if text := extractTextFromJSONRawContent(messages[idx].Content); strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func extractTextFromJSONRawContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var content any
	if err := json.Unmarshal(raw, &content); err != nil {
		return ""
	}
	return extractResponsesText(content)
}

func isModelIdentityQuestion(text string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(text))
	if trimmed == "" {
		return false
	}

	compact := strings.NewReplacer(
		" ", "",
		"\n", "",
		"\t", "",
		"？", "?",
		"！", "!",
		"。", ".",
		"，", ",",
		"：", ":",
		"“", "",
		"”", "",
		`"`, "",
		"'", "",
	).Replace(trimmed)

	if strings.Contains(compact, "你") {
		for _, keyword := range []string{
			"是谁",
			"什么模型",
			"哪个模型",
			"啥模型",
			"哪家公司",
			"哪个公司",
			"哪个公司训练",
			"由谁训练",
			"哪个公司的模型",
			"开发者",
			"开发团队",
			"谁开发",
			"开发的",
			"背后的公司",
			"背后公司",
			"厂商",
			"提供商",
			"provider",
			"模型id",
			"modelid",
			"底层模型",
			"基础模型",
			"训练方",
			"来自哪家公司",
			"属于哪个公司",
			"属于哪个模型",
		} {
			if strings.Contains(compact, keyword) {
				return true
			}
		}
	}

	for _, pattern := range []string{
		"你是谁",
		"你是什么模型",
		"你是哪个模型",
		"你是啥模型",
		"你是哪家公司",
		"你是哪家公司的",
		"你是哪个公司",
		"你是哪个公司训练的",
		"你是由谁训练的",
		"你是哪个公司的模型",
		"开发者是谁",
		"背后的公司是哪家",
		"背后公司是哪家",
		"provider是谁",
		"模型id是什么",
		"modelid是什么",
		"真正的模型id是什么",
		"真正的modelid是什么",
		"底层模型是哪个",
		"底层模型是什么",
		"基础模型是哪个",
		"基础模型是什么",
	} {
		if strings.Contains(compact, pattern) {
			return true
		}
	}

	for _, pattern := range []string{
		"who are you",
		"what model are you",
		"which model are you",
		"what is your model id",
		"what is your real model id",
		"what is your underlying model",
		"what is your base model",
		"who trained you",
		"who developed you",
		"who is your developer",
		"what company trained you",
		"what company are you from",
		"which company are you from",
		"which provider are you from",
		"who is your provider",
	} {
		if strings.Contains(trimmed, pattern) {
			return true
		}
	}

	return modelIdentityAreYouPattern.MatchString(trimmed) || modelIdentityFromCompanyPattern.MatchString(trimmed)
}

func toString(v any) string {
	s, _ := v.(string)
	return s
}
