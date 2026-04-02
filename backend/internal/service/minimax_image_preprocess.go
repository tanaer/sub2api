package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// preProcessMiniMaxImages 将请求中的 image block 通过 MiniMax VLM API 转为文字描述。
// 如果 VLM 调用失败，降级为占位符文本。document block 已在 PreFilterMiniMaxRequest 中处理。
func (s *GatewayService) preProcessMiniMaxImages(ctx context.Context, body []byte, account *Account) []byte {
	if s.miniMaxAPI == nil {
		return replaceImageBlocksWithPlaceholder(body)
	}

	// 快速检查：是否包含 image block
	if !containsImageBlock(body) {
		return body
	}

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	messages, ok := req["messages"].([]any)
	if !ok {
		return body
	}

	baseURL := miniMaxBaseURL(account)
	apiKey := account.GetCredential("api_key")
	if apiKey == "" {
		return replaceImageBlocksWithPlaceholder(body)
	}

	// 收集所有需要处理的 image block 引用
	type imageRef struct {
		msgIdx, blockIdx       int
		innerIdx               int // -1 表示非嵌套
		source                 map[string]any
		prompt                 string
		replaceTarget          *[]any // 指向需要替换的 content 数组
		replaceIdx             int    // 在 replaceTarget 中的索引
	}

	var refs []imageRef

	for mi, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content, ok := msgMap["content"].([]any)
		if !ok {
			continue
		}
		for bi, block := range content {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			if blockType == "image" {
				source, _ := blockMap["source"].(map[string]any)
				if source != nil {
					refs = append(refs, imageRef{
						msgIdx:        mi,
						blockIdx:      bi,
						innerIdx:      -1,
						source:        source,
						prompt:        buildImagePromptFromContext(messages, mi),
						replaceTarget: &content,
						replaceIdx:    bi,
					})
				}
			}
			// 处理 tool_result 内嵌 image
			if blockType == "tool_result" {
				if innerContent, ok := blockMap["content"].([]any); ok {
					for ii, innerBlock := range innerContent {
						innerMap, ok := innerBlock.(map[string]any)
						if !ok {
							continue
						}
						if innerType, _ := innerMap["type"].(string); innerType == "image" {
							source, _ := innerMap["source"].(map[string]any)
							if source != nil {
								refs = append(refs, imageRef{
									msgIdx:        mi,
									blockIdx:      bi,
									innerIdx:      ii,
									source:        source,
									prompt:        "Describe this image in detail.",
									replaceTarget: &innerContent,
									replaceIdx:    ii,
								})
							}
						}
					}
				}
			}
		}
	}

	if len(refs) == 0 {
		return body
	}

	// 并发调用 VLM API
	type vlmResult struct {
		idx     int
		content string
		err     error
	}
	results := make([]vlmResult, len(refs))
	var wg sync.WaitGroup

	for i, ref := range refs {
		wg.Add(1)
		go func(idx int, ref imageRef) {
			defer wg.Done()
			dataURL := buildImageDataURL(ref.source)
			if dataURL == "" {
				results[idx] = vlmResult{idx: idx, err: fmt.Errorf("cannot extract image data")}
				return
			}
			content, err := s.miniMaxAPI.CallVLM(ctx, baseURL, apiKey, ref.prompt, dataURL)
			results[idx] = vlmResult{idx: idx, content: content, err: err}
		}(i, ref)
	}
	wg.Wait()

	// 替换 image block 为文字描述（从后往前避免索引偏移）
	modified := false
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		r := results[i]

		var replacement map[string]any
		if r.err != nil {
			logger.LegacyPrintf("service.gateway", "MiniMax VLM failed for image in msg[%d].block[%d]: %v", ref.msgIdx, ref.blockIdx, r.err)
			replacement = map[string]any{
				"type": "text",
				"text": "[image content not supported by this model]",
			}
		} else {
			replacement = map[string]any{
				"type": "text",
				"text": fmt.Sprintf("[Image description: %s]", r.content),
			}
		}

		target := *ref.replaceTarget
		target[ref.replaceIdx] = replacement
		modified = true
	}

	if !modified {
		return body
	}

	result, err := json.Marshal(req)
	if err != nil {
		return body
	}
	return result
}

// containsImageBlock 快速检查 body 中是否包含 image block。
func containsImageBlock(body []byte) bool {
	return strings.Contains(string(body), `"type":"image"`) ||
		strings.Contains(string(body), `"type": "image"`)
}

// buildImageDataURL 从 Anthropic image source 格式构建 data URL。
// 支持 base64 source 和 URL source。
func buildImageDataURL(source map[string]any) string {
	sourceType, _ := source["type"].(string)
	switch sourceType {
	case "base64":
		mediaType, _ := source["media_type"].(string)
		data, _ := source["data"].(string)
		if mediaType == "" || data == "" {
			return ""
		}
		return fmt.Sprintf("data:%s;base64,%s", mediaType, data)
	case "url":
		url, _ := source["url"].(string)
		return url
	}
	return ""
}

// buildImagePromptFromContext 根据消息上下文构建图片理解 prompt。
// 尝试从同消息中提取文本作为 prompt，否则使用通用 prompt。
func buildImagePromptFromContext(messages []any, msgIdx int) string {
	if msgIdx < len(messages) {
		msgMap, ok := messages[msgIdx].(map[string]any)
		if !ok {
			return "Describe this image in detail."
		}
		content, ok := msgMap["content"].([]any)
		if !ok {
			// content 可能是字符串
			if text, ok := msgMap["content"].(string); ok && text != "" {
				return text
			}
			return "Describe this image in detail."
		}
		// 找同消息中的文本块作为 prompt
		var texts []string
		for _, block := range content {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			if blockType, _ := blockMap["type"].(string); blockType == "text" {
				if text, _ := blockMap["text"].(string); text != "" {
					texts = append(texts, text)
				}
			}
		}
		if len(texts) > 0 {
			return strings.Join(texts, "\n")
		}
	}
	return "Describe this image in detail."
}

// replaceImageBlocksWithPlaceholder 在 VLM 不可用时，将所有 image block 替换为占位符。
func replaceImageBlocksWithPlaceholder(body []byte) []byte {
	if !containsImageBlock(body) {
		return body
	}

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return body
	}

	messages, ok := req["messages"].([]any)
	if !ok {
		return body
	}

	modified := false
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			continue
		}
		content, ok := msgMap["content"].([]any)
		if !ok {
			continue
		}
		var newContent []any
		contentModified := false
		for _, block := range content {
			blockMap, ok := block.(map[string]any)
			if !ok {
				newContent = append(newContent, block)
				continue
			}
			blockType, _ := blockMap["type"].(string)
			if blockType == "image" {
				contentModified = true
				modified = true
				newContent = append(newContent, map[string]any{
					"type": "text",
					"text": "[image content not supported by this model]",
				})
			} else {
				newContent = append(newContent, blockMap)
				// 处理 tool_result 嵌套
				if blockType == "tool_result" {
					if innerContent, ok := blockMap["content"].([]any); ok {
						var newInner []any
						innerMod := false
						for _, ib := range innerContent {
							im, ok := ib.(map[string]any)
							if !ok {
								newInner = append(newInner, ib)
								continue
							}
							if it, _ := im["type"].(string); it == "image" {
								innerMod = true
								modified = true
								newInner = append(newInner, map[string]any{
									"type": "text",
									"text": "[image content not supported by this model]",
								})
							} else {
								newInner = append(newInner, im)
							}
						}
						if innerMod {
							blockMap["content"] = newInner
						}
					}
				}
			}
		}
		if contentModified {
			msgMap["content"] = newContent
		}
	}

	if !modified {
		return body
	}
	result, err := json.Marshal(req)
	if err != nil {
		return body
	}
	return result
}
