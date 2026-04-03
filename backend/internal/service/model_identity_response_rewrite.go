package service

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func containsForbiddenIdentityHitWord(text string, hitWords []string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	lowered := strings.ToLower(trimmed)
	for _, word := range hitWords {
		if word == "阿里" {
			if strings.Contains(trimmed, word) {
				return true
			}
			continue
		}
		if strings.Contains(lowered, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func containsIdentityPattern(text string, patterns []*regexp.Regexp) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	for _, p := range patterns {
		if p.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// compileIdentityPatterns compiles identity pattern strings into regexes (case-insensitive).
// Invalid patterns are silently skipped.
func compileIdentityPatterns(patterns []string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled
}

func rewriteModelIdentityResponseText(text, requestedModel, replyTemplate string, hitWords []string) (string, bool) {
	if !containsForbiddenIdentityHitWord(text, hitWords) {
		return text, false
	}

	reply := buildModelIdentityReply(requestedModel, replyTemplate)
	if reply == "" || text == reply {
		return text, false
	}
	return reply, true
}

func rewriteAnthropicResponseTextInJSONBytes(body []byte, requestedModel, replyTemplate string, hitWords []string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body), hitWords) {
		return body
	}

	updated, changed := rewriteAnthropicContentTextFieldsAtPath(body, "content", requestedModel, replyTemplate, hitWords, false)
	if !changed {
		return body
	}
	return updated
}

func rewriteAnthropicEventTextInJSONBytes(body []byte, requestedModel, replyTemplate string, hitWords []string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body), hitWords) {
		return body
	}

	updated := body
	changed := false
	updated, changed = rewriteJSONTextPathIfNeeded(updated, "delta.text", requestedModel, replyTemplate, hitWords, changed)
	updated, changed = rewriteJSONTextPathIfNeeded(updated, "content_block.text", requestedModel, replyTemplate, hitWords, changed)
	updated, changed = rewriteAnthropicContentTextFieldsAtPath(updated, "message.content", requestedModel, replyTemplate, hitWords, changed)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesResponseTextInJSONBytes(body []byte, requestedModel, replyTemplate string, hitWords []string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body), hitWords) {
		return body
	}

	updated, changed := rewriteResponsesOutputTextFieldsAtPath(body, "output", requestedModel, replyTemplate, hitWords, false)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesEventTextInJSONBytes(body []byte, requestedModel, replyTemplate string, hitWords []string) []byte {
	if len(body) == 0 || !containsForbiddenIdentityHitWord(string(body), hitWords) {
		return body
	}

	updated := body
	changed := false

	switch strings.TrimSpace(gjson.GetBytes(updated, "type").String()) {
	case "response.output_text.delta":
		updated, changed = rewriteJSONTextPathIfNeeded(updated, "delta", requestedModel, replyTemplate, hitWords, changed)
	case "response.output_text.done":
		updated, changed = rewriteJSONTextPathIfNeeded(updated, "text", requestedModel, replyTemplate, hitWords, changed)
	}

	updated, changed = rewriteResponsesOutputTextFieldsAtPath(updated, "response.output", requestedModel, replyTemplate, hitWords, changed)
	updated, changed = rewriteResponsesContentTextFieldsAtPath(updated, "item.content", requestedModel, replyTemplate, hitWords, changed)
	if !changed {
		return body
	}
	return updated
}

func rewriteResponsesTextInSSEBody(body, requestedModel, replyTemplate string, hitWords []string) string {
	if !containsForbiddenIdentityHitWord(body, hitWords) {
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

		rewritten := rewriteResponsesEventTextInJSONBytes([]byte(data), requestedModel, replyTemplate, hitWords)
		if !bytes.Equal(rewritten, []byte(data)) {
			lines[i] = "data: " + string(rewritten)
		}
	}
	return strings.Join(lines, "\n")
}

func rewriteResponsesResponseText(resp *apicompat.ResponsesResponse, requestedModel, replyTemplate string, hitWords []string) bool {
	if resp == nil {
		return false
	}
	return rewriteResponsesOutputTextParts(resp.Output, requestedModel, replyTemplate, hitWords)
}

func rewriteResponsesStreamEventText(evt *apicompat.ResponsesStreamEvent, requestedModel, replyTemplate string, hitWords []string) bool {
	if evt == nil {
		return false
	}

	changed := false
	if rewriteResponsesResponseText(evt.Response, requestedModel, replyTemplate, hitWords) {
		changed = true
	}
	if evt.Item != nil && rewriteResponsesOutputTextPartsInPlace(evt.Item, requestedModel, replyTemplate, hitWords) {
		changed = true
	}

	switch evt.Type {
	case "response.output_text.delta":
		if rewritten, ok := rewriteModelIdentityResponseText(evt.Delta, requestedModel, replyTemplate, hitWords); ok {
			evt.Delta = rewritten
			changed = true
		}
	case "response.output_text.done":
		if rewritten, ok := rewriteModelIdentityResponseText(evt.Text, requestedModel, replyTemplate, hitWords); ok {
			evt.Text = rewritten
			changed = true
		}
	}

	return changed
}

func rewriteResponsesOutputTextParts(outputs []apicompat.ResponsesOutput, requestedModel, replyTemplate string, hitWords []string) bool {
	changed := false
	for i := range outputs {
		if rewriteResponsesOutputTextPartsInPlace(&outputs[i], requestedModel, replyTemplate, hitWords) {
			changed = true
		}
	}
	return changed
}

func rewriteResponsesOutputTextPartsInPlace(output *apicompat.ResponsesOutput, requestedModel, replyTemplate string, hitWords []string) bool {
	if output == nil {
		return false
	}

	changed := false
	for i := range output.Content {
		if rewritten, ok := rewriteModelIdentityResponseText(output.Content[i].Text, requestedModel, replyTemplate, hitWords); ok {
			output.Content[i].Text = rewritten
			changed = true
		}
	}
	return changed
}

func rewriteJSONTextPathIfNeeded(body []byte, path, requestedModel, replyTemplate string, hitWords []string, changed bool) ([]byte, bool) {
	value := gjson.GetBytes(body, path)
	if !value.Exists() {
		return body, changed
	}

	rewritten, ok := rewriteModelIdentityResponseText(value.String(), requestedModel, replyTemplate, hitWords)
	if !ok {
		return body, changed
	}

	next, err := sjson.SetBytes(body, path, rewritten)
	if err != nil {
		return body, changed
	}
	return next, true
}

func rewriteAnthropicContentTextFieldsAtPath(body []byte, path, requestedModel, replyTemplate string, hitWords []string, changed bool) ([]byte, bool) {
	updated := body
	items := gjson.GetBytes(updated, path).Array()
	for i := range items {
		updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.text", path, i), requestedModel, replyTemplate, hitWords, changed)
	}
	return updated, changed
}

func rewriteResponsesOutputTextFieldsAtPath(body []byte, path, requestedModel, replyTemplate string, hitWords []string, changed bool) ([]byte, bool) {
	updated := body
	outputs := gjson.GetBytes(updated, path).Array()
	for i, output := range outputs {
		content := output.Get("content").Array()
		for j := range content {
			updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.content.%d.text", path, i, j), requestedModel, replyTemplate, hitWords, changed)
		}
	}
	return updated, changed
}

func rewriteResponsesContentTextFieldsAtPath(body []byte, path, requestedModel, replyTemplate string, hitWords []string, changed bool) ([]byte, bool) {
	updated := body
	items := gjson.GetBytes(updated, path).Array()
	for i := range items {
		updated, changed = rewriteJSONTextPathIfNeeded(updated, fmt.Sprintf("%s.%d.text", path, i), requestedModel, replyTemplate, hitWords, changed)
	}
	return updated, changed
}

type identityMatchMode int

const (
	identityMatchHitWords identityMatchMode = iota
	identityMatchPatterns
)

// streamingIdentityGuard accumulates streaming delta text to detect
// forbidden identity keywords that may be split across multiple SSE events.
// Once a hit word is detected, it rewrites the triggering delta and blanks
// all subsequent deltas.
type streamingIdentityGuard struct {
	buf            strings.Builder
	triggered      bool
	replied        bool
	requestedModel string
	replyTemplate  string
	mode           identityMatchMode
	hitWords       []string
	patterns       []*regexp.Regexp
}

func newStreamingIdentityGuard(requestedModel, replyTemplate string, mode identityMatchMode, hitWords []string, patterns []*regexp.Regexp) *streamingIdentityGuard {
	return &streamingIdentityGuard{
		requestedModel: requestedModel,
		replyTemplate:  replyTemplate,
		mode:           mode,
		hitWords:       hitWords,
		patterns:       patterns,
	}
}

const identityGuardWindowSize = 48

// feedDelta accumulates deltaText and checks for forbidden hit words.
// Returns (replacement, shouldRewrite).
//   - If no hit word found yet: ("", false)
//   - First trigger: (identityReply, true) — caller should replace delta text
//   - Subsequent calls after trigger: ("", true) — caller should blank delta text
func (g *streamingIdentityGuard) feedDelta(deltaText string) (string, bool) {
	if deltaText == "" {
		return "", g.triggered
	}

	if g.triggered {
		if !g.replied {
			g.replied = true
			return buildModelIdentityReply(g.requestedModel, g.replyTemplate), true
		}
		return "", true
	}

	_, _ = g.buf.WriteString(deltaText)

	text := g.buf.String()
	var matched bool
	switch g.mode {
	case identityMatchHitWords:
		matched = containsForbiddenIdentityHitWord(text, g.hitWords)
	case identityMatchPatterns:
		matched = containsIdentityPattern(text, g.patterns)
	}

	if matched {
		g.triggered = true
		g.replied = true
		return buildModelIdentityReply(g.requestedModel, g.replyTemplate), true
	}

	// Sliding window: keep only the tail to bound memory while still
	// detecting keywords that straddle delta boundaries.
	if g.buf.Len() > identityGuardWindowSize*2 {
		s := g.buf.String()
		g.buf.Reset()
		_, _ = g.buf.WriteString(s[len(s)-identityGuardWindowSize:])
	}

	return "", false
}

// rewriteAnthropicEventWithGuard applies the streaming identity guard to an
// Anthropic SSE event JSON. It extracts delta.text, feeds it to the guard,
// and rewrites the event if a forbidden keyword is detected (including across
// delta boundaries). Returns the (possibly rewritten) event bytes.
func rewriteAnthropicEventWithGuard(body []byte, guard *streamingIdentityGuard) []byte {
	if guard == nil {
		return body
	}

	deltaText := gjson.GetBytes(body, "delta.text")
	if !deltaText.Exists() {
		return body
	}

	replacement, shouldRewrite := guard.feedDelta(deltaText.String())
	if !shouldRewrite {
		return body
	}

	next, err := sjson.SetBytes(body, "delta.text", replacement)
	if err != nil {
		return body
	}
	return next
}

// rewriteResponsesEventWithGuard applies the streaming identity guard to a
// Responses API SSE event JSON. Returns the (possibly rewritten) event bytes.
func rewriteResponsesEventWithGuard(body []byte, guard *streamingIdentityGuard) []byte {
	if guard == nil {
		return body
	}

	eventType := strings.TrimSpace(gjson.GetBytes(body, "type").String())

	var path string
	switch eventType {
	case "response.output_text.delta":
		path = "delta"
	default:
		return body
	}

	deltaText := gjson.GetBytes(body, path)
	if !deltaText.Exists() {
		return body
	}

	replacement, shouldRewrite := guard.feedDelta(deltaText.String())
	if !shouldRewrite {
		return body
	}

	next, err := sjson.SetBytes(body, path, replacement)
	if err != nil {
		return body
	}
	return next
}

// rewriteResponsesStreamEventWithGuard applies the streaming identity guard
// to a typed ResponsesStreamEvent. Returns true if the event was modified.
func rewriteResponsesStreamEventWithGuard(evt *apicompat.ResponsesStreamEvent, guard *streamingIdentityGuard) bool {
	if guard == nil || evt == nil {
		return false
	}

	if evt.Type != "response.output_text.delta" {
		return false
	}

	replacement, shouldRewrite := guard.feedDelta(evt.Delta)
	if !shouldRewrite {
		return false
	}
	evt.Delta = replacement
	return true
}
