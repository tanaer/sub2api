package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MiniMaxAPIClient 封装 MiniMax 辅助 API（图片理解、网络搜索）的调用。
// 与主 Anthropic 兼容 API（/anthropic/v1/messages）使用同一 API Key，
// 但走独立的端点和认证方式（Bearer token）。
type MiniMaxAPIClient struct {
	vlmClient    *http.Client
	searchClient *http.Client
}

// NewMiniMaxAPIClient 创建客户端实例。
func NewMiniMaxAPIClient() *MiniMaxAPIClient {
	return &MiniMaxAPIClient{
		vlmClient:    &http.Client{Timeout: 15 * time.Second},
		searchClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// miniMaxBaseURL 从 account 中提取 MiniMax API 的 base URL。
// account 的 base_url 通常是 "https://api.minimaxi.com/anthropic"，
// 辅助 API 需要去掉 /anthropic 后缀。
func miniMaxBaseURL(account *Account) string {
	raw := strings.TrimSpace(account.GetCredential("base_url"))
	if raw == "" {
		return "https://api.minimaxi.com"
	}
	// 去掉 /anthropic 或 /anthropic/ 后缀
	raw = strings.TrimRight(raw, "/")
	raw = strings.TrimSuffix(raw, "/anthropic")
	return raw
}

// CallVLM 调用 MiniMax 图片理解 API。
// imageDataURL 支持 data:image/...;base64,... 格式。
// 返回 VLM 对图片的文字描述。
func (c *MiniMaxAPIClient) CallVLM(ctx context.Context, baseURL, apiKey, prompt, imageDataURL string) (string, error) {
	payload := map[string]string{
		"prompt":    prompt,
		"image_url": imageDataURL,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal VLM request: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/v1/coding_plan/vlm"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build VLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("MM-API-Source", "sub2api")

	resp, err := c.vlmClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("VLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read VLM response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("VLM API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content  string `json:"content"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse VLM response: %w", err)
	}
	if result.BaseResp.StatusCode != 0 {
		return "", fmt.Errorf("VLM API error %d: %s", result.BaseResp.StatusCode, result.BaseResp.StatusMsg)
	}
	if result.Content == "" {
		return "", fmt.Errorf("VLM API returned empty content")
	}
	return result.Content, nil
}

// CallWebSearch 调用 MiniMax 网络搜索 API。
// 返回搜索结果的格式化文本（用于注入到 tool_result）。
func (c *MiniMaxAPIClient) CallWebSearch(ctx context.Context, baseURL, apiKey, query string) (string, error) {
	payload := map[string]string{"q": query}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal search request: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + "/v1/coding_plan/search"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("MM-API-Source", "sub2api")

	resp, err := c.searchClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return "", fmt.Errorf("read search response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search API returned %d: %s", resp.StatusCode, string(respBody))
	}

	// 解析并格式化搜索结果为可读文本
	var result struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
			Date    string `json:"date"`
		} `json:"organic"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse search response: %w", err)
	}
	if result.BaseResp.StatusCode != 0 {
		return "", fmt.Errorf("search API error %d: %s", result.BaseResp.StatusCode, result.BaseResp.StatusMsg)
	}

	// 格式化为文本，便于模型理解
	var sb strings.Builder
	sb.WriteString("Web search results:\n\n")
	for i, item := range result.Organic {
		if i >= 8 {
			break // 最多保留 8 条结果
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
		if item.Date != "" {
			sb.WriteString(fmt.Sprintf("   Date: %s\n", item.Date))
		}
		sb.WriteString(fmt.Sprintf("   URL: %s\n", item.Link))
		sb.WriteString(fmt.Sprintf("   %s\n\n", item.Snippet))
	}
	return sb.String(), nil
}
