package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newModelIdentitySyntheticResponses(requestedModel, reply string) (*apicompat.ResponsesResponse, []apicompat.ResponsesStreamEvent) {
	responseID := "resp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	messageID := "msg_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	contentPart := apicompat.ResponsesContentPart{
		Type: "output_text",
		Text: reply,
	}
	completedItem := apicompat.ResponsesOutput{
		ID:      messageID,
		Type:    "message",
		Role:    "assistant",
		Content: []apicompat.ResponsesContentPart{contentPart},
		Status:  "completed",
	}
	inProgressItem := apicompat.ResponsesOutput{
		ID:      messageID,
		Type:    "message",
		Role:    "assistant",
		Content: []apicompat.ResponsesContentPart{},
		Status:  "in_progress",
	}

	finalResponse := &apicompat.ResponsesResponse{
		ID:     responseID,
		Object: "response",
		Model:  requestedModel,
		Status: "completed",
		Output: []apicompat.ResponsesOutput{completedItem},
		Usage: &apicompat.ResponsesUsage{
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
		},
	}

	events := []apicompat.ResponsesStreamEvent{
		{
			Type:           "response.created",
			Response:       &apicompat.ResponsesResponse{ID: responseID, Object: "response", Model: requestedModel, Status: "in_progress", Output: []apicompat.ResponsesOutput{}},
			SequenceNumber: 1,
		},
		{
			Type:           "response.output_item.added",
			OutputIndex:    0,
			Item:           &inProgressItem,
			SequenceNumber: 2,
		},
		{
			Type:           "response.output_text.delta",
			OutputIndex:    0,
			ContentIndex:   0,
			ItemID:         messageID,
			Delta:          reply,
			SequenceNumber: 3,
		},
		{
			Type:           "response.output_text.done",
			OutputIndex:    0,
			ContentIndex:   0,
			ItemID:         messageID,
			Text:           reply,
			SequenceNumber: 4,
		},
		{
			Type:           "response.output_item.done",
			OutputIndex:    0,
			Item:           &completedItem,
			SequenceNumber: 5,
		},
		{
			Type:           "response.completed",
			Response:       finalResponse,
			SequenceNumber: 6,
		},
	}

	return finalResponse, events
}

func writeLocalOpenAIResponsesIdentityResponse(c *gin.Context, requestedModel, reply string, stream bool, startTime time.Time) (*OpenAIForwardResult, error) {
	resp, events := newModelIdentitySyntheticResponses(requestedModel, reply)
	if stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		for _, event := range events {
			payload, err := json.Marshal(event)
			if err != nil {
				return nil, err
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", payload); err != nil {
				return nil, err
			}
			c.Writer.Flush()
		}
	} else {
		body, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		c.Data(http.StatusOK, "application/json", body)
	}

	return &OpenAIForwardResult{
		Usage:    OpenAIUsage{},
		Model:    requestedModel,
		Stream:   stream,
		Duration: time.Since(startTime),
		FirstTokenMs: func() *int {
			zero := 0
			return &zero
		}(),
	}, nil
}

func writeLocalChatCompletionsIdentityResponse(c *gin.Context, requestedModel, reply string, stream, includeUsage bool, startTime time.Time) (*OpenAIForwardResult, error) {
	resp, events := newModelIdentitySyntheticResponses(requestedModel, reply)
	if stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		state := apicompat.NewResponsesEventToChatState()
		state.IncludeUsage = includeUsage
		for _, event := range events {
			chunks := apicompat.ResponsesEventToChatChunks(&event, state)
			for _, chunk := range chunks {
				sse, err := apicompat.ChatChunkToSSE(chunk)
				if err != nil {
					return nil, err
				}
				if _, err := fmt.Fprint(c.Writer, sse); err != nil {
					return nil, err
				}
			}
			c.Writer.Flush()
		}
		if _, err := fmt.Fprint(c.Writer, "data: [DONE]\n\n"); err != nil {
			return nil, err
		}
		c.Writer.Flush()
	} else {
		chatResp := apicompat.ResponsesToChatCompletions(resp, requestedModel)
		body, err := json.Marshal(chatResp)
		if err != nil {
			return nil, err
		}
		c.Data(http.StatusOK, "application/json", body)
	}

	return &OpenAIForwardResult{
		Usage:    OpenAIUsage{},
		Model:    requestedModel,
		Stream:   stream,
		Duration: time.Since(startTime),
		FirstTokenMs: func() *int {
			zero := 0
			return &zero
		}(),
	}, nil
}

func writeLocalAnthropicIdentityResponse(c *gin.Context, requestedModel, reply string, stream bool, startTime time.Time) (*ForwardResult, error) {
	resp, events := newModelIdentitySyntheticResponses(requestedModel, reply)
	if stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		state := apicompat.NewResponsesEventToAnthropicState()
		for _, event := range events {
			anthropicEvents := apicompat.ResponsesEventToAnthropicEvents(&event, state)
			for _, anthropicEvent := range anthropicEvents {
				sse, err := apicompat.ResponsesAnthropicEventToSSE(anthropicEvent)
				if err != nil {
					return nil, err
				}
				if _, err := fmt.Fprint(c.Writer, sse); err != nil {
					return nil, err
				}
			}
			c.Writer.Flush()
		}
	} else {
		anthropicResp := apicompat.ResponsesToAnthropic(resp, requestedModel)
		body, err := json.Marshal(anthropicResp)
		if err != nil {
			return nil, err
		}
		c.Data(http.StatusOK, "application/json", body)
	}

	return &ForwardResult{
		Usage:    ClaudeUsage{},
		Model:    requestedModel,
		Stream:   stream,
		Duration: time.Since(startTime),
		FirstTokenMs: func() *int {
			zero := 0
			return &zero
		}(),
	}, nil
}

func writeLocalOpenAIAnthropicIdentityResponse(c *gin.Context, requestedModel, reply string, stream bool, startTime time.Time) (*OpenAIForwardResult, error) {
	resp, events := newModelIdentitySyntheticResponses(requestedModel, reply)
	if stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		state := apicompat.NewResponsesEventToAnthropicState()
		for _, event := range events {
			anthropicEvents := apicompat.ResponsesEventToAnthropicEvents(&event, state)
			for _, anthropicEvent := range anthropicEvents {
				sse, err := apicompat.ResponsesAnthropicEventToSSE(anthropicEvent)
				if err != nil {
					return nil, err
				}
				if _, err := fmt.Fprint(c.Writer, sse); err != nil {
					return nil, err
				}
			}
			c.Writer.Flush()
		}
	} else {
		anthropicResp := apicompat.ResponsesToAnthropic(resp, requestedModel)
		body, err := json.Marshal(anthropicResp)
		if err != nil {
			return nil, err
		}
		c.Data(http.StatusOK, "application/json", body)
	}

	return &OpenAIForwardResult{
		Usage:    OpenAIUsage{},
		Model:    requestedModel,
		Stream:   stream,
		Duration: time.Since(startTime),
		FirstTokenMs: func() *int {
			zero := 0
			return &zero
		}(),
	}, nil
}
