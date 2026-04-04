package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestInjectMiniMaxWebSearchTool(t *testing.T) {
	t.Run("injects when no tools", func(t *testing.T) {
		body := []byte(`{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"hello"}]}`)
		out := injectMiniMaxWebSearchTool(body)
		tools := gjson.GetBytes(out, "tools")
		require.True(t, tools.IsArray())
		require.Equal(t, 1, len(tools.Array()))
		require.Equal(t, "web_search", tools.Array()[0].Get("name").String())
	})

	t.Run("does not inject when tools already exist", func(t *testing.T) {
		body := []byte(`{"model":"MiniMax-M2.5","tools":[{"name":"my_tool"}],"messages":[{"role":"user","content":"hello"}]}`)
		out := injectMiniMaxWebSearchTool(body)
		tools := gjson.GetBytes(out, "tools")
		require.Equal(t, 1, len(tools.Array()))
		require.Equal(t, "my_tool", tools.Array()[0].Get("name").String())
	})
}

func TestShouldInjectMiniMaxTools(t *testing.T) {
	minimax := &Account{
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.minimaxi.com/anthropic"},
	}

	t.Run("inject when no tools", func(t *testing.T) {
		parsed := &ParsedRequest{HasTools: false}
		require.True(t, shouldInjectMiniMaxTools(minimax, parsed))
	})

	t.Run("skip when has tools", func(t *testing.T) {
		parsed := &ParsedRequest{HasTools: true}
		require.False(t, shouldInjectMiniMaxTools(minimax, parsed))
	})

	t.Run("skip for non-minimax", func(t *testing.T) {
		other := &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}
		parsed := &ParsedRequest{HasTools: false}
		require.False(t, shouldInjectMiniMaxTools(other, parsed))
	})
}

func TestExtractWebSearchToolCalls(t *testing.T) {
	t.Run("extracts web_search calls", func(t *testing.T) {
		resp := []byte(`{
			"content": [
				{"type":"thinking","thinking":"...","signature":"abc"},
				{"type":"tool_use","id":"call_1","name":"web_search","input":{"query":"latest news"}},
				{"type":"text","text":"Let me search for that."}
			]
		}`)
		calls := extractWebSearchToolCalls(resp)
		require.Len(t, calls, 1)
		require.Equal(t, "call_1", calls[0].ID)
		require.Equal(t, "latest news", calls[0].Query)
	})

	t.Run("ignores non-web_search tools", func(t *testing.T) {
		resp := []byte(`{
			"content": [
				{"type":"tool_use","id":"call_1","name":"get_weather","input":{"city":"shanghai"}}
			]
		}`)
		calls := extractWebSearchToolCalls(resp)
		require.Len(t, calls, 0)
	})

	t.Run("empty content", func(t *testing.T) {
		resp := []byte(`{"content":[{"type":"text","text":"hello"}]}`)
		calls := extractWebSearchToolCalls(resp)
		require.Len(t, calls, 0)
	})
}

func TestBuildToolLoopNextRequest(t *testing.T) {
	currentBody := []byte(`{
		"model":"MiniMax-M2.5",
		"messages":[{"role":"user","content":"What's the latest news?"}],
		"tools":[{"name":"web_search"}]
	}`)

	assistantResp := []byte(`{
		"content":[
			{"type":"tool_use","id":"call_1","name":"web_search","input":{"query":"latest news 2026"}}
		]
	}`)

	results := []miniMaxToolResult{
		{ToolUseID: "call_1", Content: "1. Breaking news: AI advances..."},
	}

	out := buildToolLoopNextRequest(currentBody, assistantResp, results)

	var req map[string]any
	require.NoError(t, json.Unmarshal(out, &req))

	messages, ok := req["messages"].([]any)
	require.True(t, ok)
	require.Len(t, messages, 3) // original user + assistant + tool_result

	// Check assistant message
	assistantMsg, ok := messages[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "assistant", assistantMsg["role"])

	// Check user message with tool_result
	userMsg, ok := messages[2].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "user", userMsg["role"])
	toolResults, ok := userMsg["content"].([]any)
	require.True(t, ok)
	require.Len(t, toolResults, 1)
	tr, ok := toolResults[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "tool_result", tr["type"])
	require.Equal(t, "call_1", tr["tool_use_id"])
	require.Contains(t, tr["content"], "Breaking news")
}

func TestBuildToolLoopNextRequest_ErrorResult(t *testing.T) {
	currentBody := []byte(`{
		"model":"MiniMax-M2.5",
		"messages":[{"role":"user","content":"search something"}],
		"tools":[{"name":"web_search"}]
	}`)

	assistantResp := []byte(`{
		"content":[{"type":"tool_use","id":"call_1","name":"web_search","input":{"query":"test"}}]
	}`)

	results := []miniMaxToolResult{
		{ToolUseID: "call_1", Content: "Search failed: timeout", IsError: true},
	}

	out := buildToolLoopNextRequest(currentBody, assistantResp, results)

	var req map[string]any
	require.NoError(t, json.Unmarshal(out, &req))

	messages, ok := req["messages"].([]any)
	require.True(t, ok)
	userMsg, ok := messages[2].(map[string]any)
	require.True(t, ok)
	toolResults, ok := userMsg["content"].([]any)
	require.True(t, ok)
	tr, ok := toolResults[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, tr["is_error"])
}
