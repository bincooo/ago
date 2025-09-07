package model

import (
	"fmt"
)

func MakeSSEResponse(content string, created int64) *Response {
	return &Response{
		Model:   "LLM",
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion.chunk",
		Choices: []Choice{
			{
				Index: 0,
				Delta: &struct {
					Type             string `json:"type,omitempty"`
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"text", "assistant", content, "", nil},
			},
		},
	}
}

func MakeResponse(content string, created int64) *Response {
	stop := "stop"
	return &Response{
		Model:   "LLM",
		Created: created,
		Id:      fmt.Sprintf("chatcmpl-%d", created),
		Object:  "chat.completion",
		Choices: []Choice{
			{
				Index: 0,
				Message: &struct {
					Role             string `json:"role,omitempty"`
					Content          string `json:"content,omitempty"`
					ReasoningContent string `json:"reasoning_content,omitempty"`

					ToolCalls []ChoiceToolCall `json:"tool_calls,omitempty"`
				}{"assistant", content, "", nil},
				FinishReason: &stop,
			},
		},
		//Usage: usage,
	}
}
