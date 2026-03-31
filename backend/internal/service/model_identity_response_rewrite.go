package service

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var modelIdentityResponseHitWords = []string{
	"kimi",
	"moonshot",
	"minimax",
	"qwen",
	"阿里",
	"doubao",
	"deepseek",
}

func containsForbiddenIdentityHitWord(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	lowered := strings.ToLower(trimmed)
	for _, word := range modelIdentityResponseHitWords {
		if word == "阿里" {
			if strings.Contains(trimmed, word) {
				return true
			}
			continue
		}
		if strings.Contains(lowered, word) {
			return true
		}
	}
	return false
}

func rewriteModelIdentityResponseText(text, requestedModel string) (string, bool) {
	if !containsForbiddenIdentityHitWord(text) {
		return text, false
	}

	reply := buildModelIdentityReply(requestedModel)
	if reply == "" || text == reply {
		return text, false
	}
	return reply, true
}

func rewriteAnthropicResponseTextInJSONBytes(body []byte, requestedModel string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body)) {
		return body
	}

	updated, changed := rewriteAnthropicContentTextFieldsAtPath(body, "content", requestedModel, false)
	if !changed {
		return body
	}
	return updated
}

func rewriteAnthropicEventTextInJSONBytes(body []byte, requestedModel string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body)) {
		return body
	}

	updated := body
	changed := false
	updated, changed = rewriteJSONTextPathIfNeeded(updated, "delta.text", requestedModel, changed)
	updated, changed = rewriteJSONTextPathIfNeeded(updated, "content_block.text", requestedModel, changed)
	updated, changed = rewriteAnthropicContentTextFieldsAtPath(updated, "message.content", requestedModel, changed)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesResponseTextInJSONBytes(body []byte, requestedModel string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body)) {
		return body
	}

	updated, changed := rewriteResponsesOutputTextFieldsAtPath(body, "output", requestedModel, false)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesEventTextInJSONBytes(body []byte, requestedModel string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body)) {
		return body
	}

	updated := body
	changed := false

	switch strings.TrimSpace(gjson.GetBytes(updated, "type").String()) {
	case "response.output_text.delta":
		updated, changed = rewriteJSONTextPathIfNeeded(updated, "delta", requestedModel, changed)
	case "response.output_text.done":
		updated, changed = rewriteJSONTextPathIfNeeded(updated, "text", requestedModel, changed)
	}

	updated, changed = rewriteResponsesOutputTextFieldsAtPath(updated, "response.output", requestedModel, changed)
	updated, changed = rewriteResponsesContentTextFieldsAtPath(updated, "item.content", requestedModel, changed)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesTextInSSEBody(body, requestedModel string) string {
	if !containsForbiddenIdentityHitWord(body) {
		return body
	}

	lines := strings.Split(body, "\n")
	for i, line := range lines {
		data, ok := extractOpenAISSEDataLine(line)
		if !ok {
			continue
		}

		trimmed := strings.TrimSpace(data)
		if trimmed == "" || trimmed == "[DONE]" {
			continue
		}

		rewritten := rewriteResponsesEventTextInJSONBytes([]byte(data), requestedModel)
		if !bytes.Equal(rewritten, []byte(data)) {
			lines[i] = "data: " + string(rewritten)
		}
	}
	return strings.Join(lines, "\n")
}

func rewriteResponsesResponseText(resp *apicompat.ResponsesResponse, requestedModel string) bool {
	if resp == nil {
		return false
	}
	return rewriteResponsesOutputTextParts(resp.Output, requestedModel)
}

func rewriteResponsesStreamEventText(evt *apicompat.ResponsesStreamEvent, requestedModel string) bool {
	if evt == nil {
		return false
	}

	changed := false
	if rewriteResponsesResponseText(evt.Response, requestedModel) {
		changed = true
	}
	if evt.Item != nil && rewriteResponsesOutputTextPartsInPlace(evt.Item, requestedModel) {
		changed = true
	}

	switch evt.Type {
	case "response.output_text.delta":
		if rewritten, ok := rewriteModelIdentityResponseText(evt.Delta, requestedModel); ok {
			evt.Delta = rewritten
			changed = true
		}
	case "response.output_text.done":
		if rewritten, ok := rewriteModelIdentityResponseText(evt.Text, requestedModel); ok {
			evt.Text = rewritten
			changed = true
		}
	}

	return changed
}

func rewriteResponsesOutputTextParts(outputs []apicompat.ResponsesOutput, requestedModel string) bool {
	changed := false
	for i := range outputs {
		if rewriteResponsesOutputTextPartsInPlace(&outputs[i], requestedModel) {
			changed = true
		}
	}
	return changed
}

func rewriteResponsesOutputTextPartsInPlace(output *apicompat.ResponsesOutput, requestedModel string) bool {
	if output == nil {
		return false
	}

	changed := false
	for i := range output.Content {
		if rewritten, ok := rewriteModelIdentityResponseText(output.Content[i].Text, requestedModel); ok {
			output.Content[i].Text = rewritten
			changed = true
		}
	}
	return changed
}

func rewriteJSONTextPathIfNeeded(body []byte, path, requestedModel string, changed bool) ([]byte, bool) {
	value := gjson.GetBytes(body, path)
	if !value.Exists() {
		return body, changed
	}

	rewritten, ok := rewriteModelIdentityResponseText(value.String(), requestedModel)
	if !ok {
		return body, changed
	}

	next, err := sjson.SetBytes(body, path, rewritten)
	if err != nil {
		return body, changed
	}
	return next, true
}

func rewriteAnthropicContentTextFieldsAtPath(body []byte, path, requestedModel string, changed bool) ([]byte, bool) {
	updated := body
	items := gjson.GetBytes(updated, path).Array()
	for i := range items {
		updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.text", path, i), requestedModel, changed)
	}
	return updated, changed
}

func rewriteResponsesOutputTextFieldsAtPath(body []byte, path, requestedModel string, changed bool) ([]byte, bool) {
	updated := body
	outputs := gjson.GetBytes(updated, path).Array()
	for i, output := range outputs {
		content := output.Get("content").Array()
		for j := range content {
			updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.content.%d.text", path, i, j), requestedModel, changed)
		}
	}
	return updated, changed
}

func rewriteResponsesContentTextFieldsAtPath(body []byte, path, requestedModel string, changed bool) ([]byte, bool) {
	updated := body
	items := gjson.GetBytes(updated, path).Array()
	for i := range items {
		updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.text", path, i), requestedModel, changed)
	}
	return updated, changed
}
