package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	miniMaxToolLoopMaxIterations = 3
	miniMaxToolLoopTimeout       = 30 * time.Second
)

// miniMaxWebSearchToolJSON 是注入到请求中的 web_search 工具定义。
var miniMaxWebSearchToolJSON = map[string]any{
	"name":        "web_search",
	"description": "Search the web for real-time information. Use this when the question involves current events, recent data, or facts you are unsure about.",
	"input_schema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query keywords",
			},
		},
		"required": []string{"query"},
	},
}

// injectMiniMaxWebSearchTool 向请求中注入 web_search 工具定义。
// 仅当请求中没有已定义的 tools 时注入。
func injectMiniMaxWebSearchTool(body []byte) []byte {
	// 已有 tools 则不注入
	if gjson.GetBytes(body, "tools").Exists() {
		return body
	}

	tools := []any{miniMaxWebSearchToolJSON}
	result, err := sjson.SetBytes(body, "tools", tools)
	if err != nil {
		return body
	}
	return result
}

// shouldInjectMiniMaxTools 检查是否应注入 MiniMax 辅助工具。
func shouldInjectMiniMaxTools(account *Account, parsed *ParsedRequest) bool {
	if account == nil || !account.IsMiniMaxAnthropicAPIKey() {
		return false
	}
	// 仅当客户端未定义自己的 tools 时注入
	return !parsed.HasTools
}

// executeMiniMaxToolLoop 在获得 MiniMax 响应后，检测并执行 web_search 工具调用。
// 返回最终的 http.Response（可能是原始响应或工具循环后的最终响应）。
// 如果不需要工具循环，返回 (nil, false, nil)。
// reqStream 指示客户端的原始流式请求设置。
func (s *GatewayService) executeMiniMaxToolLoop(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	firstResp *http.Response,
	originalBody []byte,
	token, tokenType string,
	reqModel string,
	reqStream bool,
	proxyURL string,
	tlsProfile *tlsfingerprint.Profile,
	shouldMimicClaudeCode bool,
	parsed *ParsedRequest,
) (*http.Response, bool, error) {
	loopCtx, cancel := context.WithTimeout(ctx, miniMaxToolLoopTimeout)
	defer cancel()

	// 读取首次响应（需要完整读取才能检测 tool_use）
	respBody, err := io.ReadAll(io.LimitReader(firstResp.Body, 2<<20))
	_ = firstResp.Body.Close()
	if err != nil {
		return rebuildResponse(firstResp, respBody), false, nil
	}

	// 检测是否有 web_search tool_use
	toolCalls := extractWebSearchToolCalls(respBody)
	if len(toolCalls) == 0 {
		// 无工具调用，返回原始响应
		return rebuildResponse(firstResp, respBody), false, nil
	}

	baseURL := miniMaxBaseURL(account)
	apiKey := account.GetCredential("api_key")

	// 工具执行循环
	currentBody := originalBody
	currentRespBody := respBody
	for iteration := 0; iteration < miniMaxToolLoopMaxIterations; iteration++ {
		toolCalls = extractWebSearchToolCalls(currentRespBody)
		if len(toolCalls) == 0 {
			break
		}

		logger.LegacyPrintf("service.gateway",
			"Account %d: MiniMax web_search tool loop iteration %d, %d tool calls",
			account.ID, iteration+1, len(toolCalls))

		// 执行所有 web_search 调用
		toolResults := make([]miniMaxToolResult, len(toolCalls))
		for i, tc := range toolCalls {
			searchResult, searchErr := s.miniMaxAPI.CallWebSearch(loopCtx, baseURL, apiKey, tc.Query)
			if searchErr != nil {
				logger.LegacyPrintf("service.gateway",
					"Account %d: MiniMax web_search failed for query %q: %v",
					account.ID, tc.Query, searchErr)
				toolResults[i] = miniMaxToolResult{
					ToolUseID: tc.ID,
					Content:   fmt.Sprintf("Search failed: %v", searchErr),
					IsError:   true,
				}
			} else {
				toolResults[i] = miniMaxToolResult{
					ToolUseID: tc.ID,
					Content:   searchResult,
				}
			}
		}

		// 构建下一轮请求：追加 assistant 回复 + tool_result
		currentBody = buildToolLoopNextRequest(currentBody, currentRespBody, toolResults)

		// 判断是否是最后一轮（如果是，使用客户端原始的 stream 设置）
		isLastIteration := (iteration == miniMaxToolLoopMaxIterations-1)
		useStream := isLastIteration && reqStream

		if useStream {
			currentBody, _ = sjson.SetBytes(currentBody, "stream", true)
		} else {
			currentBody, _ = sjson.SetBytes(currentBody, "stream", false)
		}

		// 发送下一轮请求
		upstreamCtx, releaseCtx := context.WithCancel(loopCtx)
		upstreamReq, buildErr := s.buildUpstreamRequest(upstreamCtx, c, account, currentBody, token, tokenType, reqModel, useStream, shouldMimicClaudeCode)
		if buildErr != nil {
			releaseCtx()
			return rebuildResponse(firstResp, currentRespBody), false, nil
		}

		nextResp, doErr := s.doMiniMaxToolLoopRequest(upstreamReq, proxyURL, account, tlsProfile)
		releaseCtx()
		if doErr != nil {
			logger.LegacyPrintf("service.gateway",
				"Account %d: MiniMax tool loop request failed: %v", account.ID, doErr)
			return rebuildResponse(firstResp, currentRespBody), false, nil
		}

		if nextResp.StatusCode >= 400 {
			// 上游错误，返回该错误响应
			return nextResp, useStream, nil
		}

		if useStream {
			// 最后一轮使用流式，直接返回
			return nextResp, true, nil
		}

		// 读取非流式响应，继续循环
		nextRespBody, readErr := io.ReadAll(io.LimitReader(nextResp.Body, 2<<20))
		_ = nextResp.Body.Close()
		if readErr != nil {
			return rebuildResponse(nextResp, nextRespBody), false, nil
		}
		currentRespBody = nextRespBody

		// 检查是否还有 tool_use
		if len(extractWebSearchToolCalls(currentRespBody)) == 0 {
			// 循环结束，返回最终响应
			return rebuildResponse(nextResp, currentRespBody), false, nil
		}
	}

	// 循环次数用尽，返回最后的响应
	return rebuildResponseFromBody(firstResp, currentRespBody), false, nil
}

type miniMaxToolCall struct {
	ID    string
	Name  string
	Query string
}

type miniMaxToolResult struct {
	ToolUseID string
	Content   string
	IsError   bool
}

// extractWebSearchToolCalls 从 MiniMax 响应中提取 web_search 工具调用。
func extractWebSearchToolCalls(respBody []byte) []miniMaxToolCall {
	content := gjson.GetBytes(respBody, "content")
	if !content.IsArray() {
		return nil
	}

	var calls []miniMaxToolCall
	content.ForEach(func(_, item gjson.Result) bool {
		if item.Get("type").String() != "tool_use" {
			return true
		}
		if item.Get("name").String() != "web_search" {
			return true
		}
		query := item.Get("input.query").String()
		id := item.Get("id").String()
		if query != "" && id != "" {
			calls = append(calls, miniMaxToolCall{
				ID:    id,
				Name:  "web_search",
				Query: query,
			})
		}
		return true
	})
	return calls
}

// buildToolLoopNextRequest 构建工具循环的下一轮请求体。
// 在原始 messages 后追加 assistant 响应和 tool_result。
func buildToolLoopNextRequest(currentBody, assistantRespBody []byte, results []miniMaxToolResult) []byte {
	var req map[string]any
	if err := json.Unmarshal(currentBody, &req); err != nil {
		return currentBody
	}

	messages, ok := req["messages"].([]any)
	if !ok {
		return currentBody
	}

	// 解析 assistant 响应的 content
	var assistantResp struct {
		Content []any `json:"content"`
	}
	if err := json.Unmarshal(assistantRespBody, &assistantResp); err != nil {
		return currentBody
	}

	// 追加 assistant 消息
	messages = append(messages, map[string]any{
		"role":    "assistant",
		"content": assistantResp.Content,
	})

	// 构建 tool_result blocks
	var toolResultBlocks []any
	for _, r := range results {
		block := map[string]any{
			"type":        "tool_result",
			"tool_use_id": r.ToolUseID,
			"content":     r.Content,
		}
		if r.IsError {
			block["is_error"] = true
		}
		toolResultBlocks = append(toolResultBlocks, block)
	}

	// 追加 user 消息（包含 tool_result）
	messages = append(messages, map[string]any{
		"role":    "user",
		"content": toolResultBlocks,
	})

	req["messages"] = messages

	newBody, err := json.Marshal(req)
	if err != nil {
		return currentBody
	}
	return newBody
}

// doMiniMaxToolLoopRequest 执行工具循环中的上游请求。
// 使用 GatewayService 的 httpUpstream 以复用连接池和 TLS 配置。
func (s *GatewayService) doMiniMaxToolLoopRequest(
	req *http.Request,
	proxyURL string,
	account *Account,
	tlsProfile *tlsfingerprint.Profile,
) (*http.Response, error) {
	return s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, tlsProfile)
}

// rebuildResponse 用新的 body 重建 http.Response。
func rebuildResponse(orig *http.Response, body []byte) *http.Response {
	return &http.Response{
		StatusCode: orig.StatusCode,
		Header:     orig.Header.Clone(),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

// rebuildResponseFromBody 从 body 重建响应（用于循环结束时）。
func rebuildResponseFromBody(orig *http.Response, body []byte) *http.Response {
	return rebuildResponse(orig, body)
}
