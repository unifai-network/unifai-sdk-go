package unifai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/openai/openai-go"
	"github.com/unifai-network/unifai-sdk-go/common"
)

const (
	SEARCH_TOOLS = "search_services"
	CALL_TOOL    = "invoke_service"
)

// ToolsConfig holds the configuration for the Tools wrapper.
type ToolsConfig struct {
	APIKey               string
	CallToolsConcurrency int
}

// Tools provides a higher-level abstraction around the ToolsAPI.
type Tools struct {
	api         *ToolsAPI
	concurrency int
	openaiTools []openai.ChatCompletionToolParam
}

// NewTools creates a new Tools instance with the provided configuration.
// It defines the tools directly using OpenAI types.
func NewTools(config ToolsConfig) *Tools {
	if config.CallToolsConcurrency < 1 {
		config.CallToolsConcurrency = 1
	}

	toolsList := []openai.ChatCompletionToolParam{
		{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(SEARCH_TOOLS),
				Description: openai.String(fmt.Sprintf("Search for tools. The tools cover a wide range of domains including data sources, APIs, SDKs, etc. Actions returned should be used in %s.", CALL_TOOL)),
				Parameters: openai.F(openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "The query to search for tools. Describe what you want to do or what tools to use.",
						},
						"limit": map[string]interface{}{
							"type":        "number",
							"description": "The maximum number of tools to return (must be between 1 and 100, default is 10).",
						},
					},
					"required": []string{"query"},
				}),
			}),
		},
		{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(CALL_TOOL),
				Description: openai.String(fmt.Sprintf("Call a tool returned by %s.", SEARCH_TOOLS)),
				Parameters: openai.F(openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"action": map[string]interface{}{
							"type":        "string",
							"description": fmt.Sprintf("The exact action to be called from the %s result.", SEARCH_TOOLS),
						},
						"payload": map[string]interface{}{
							"type":        "string",
							"description": "The action payload (can be a JSON object or JSON-encoded string).",
						},
						"payment": map[string]interface{}{
							"type":        "number",
							"description": "Amount to authorize in USD. A positive number indicates a charge cap, while a negative number requests a minimum payout.",
						},
					},
					"required": []string{"action", "payload"},
				}),
			}),
		},
	}

	return &Tools{
		api:         NewToolsAPI(common.APIConfig{APIKey: config.APIKey}),
		concurrency: config.CallToolsConcurrency,
		openaiTools: toolsList,
	}
}

// SetAPIEndpoint sets the API endpoint for the ToolsAPI.
func (t *Tools) SetAPIEndpoint(endpoint string) {
	t.api.SetEndpoint(endpoint)
}

// GetTools returns the list of available tools as OpenAI types.
func (t *Tools) GetTools() []openai.ChatCompletionToolParam {
	return t.openaiTools
}

// CallTool calls a single tool by name with the provided arguments.
// It accepts args as a string (JSON) or as a map, and correctly handles both
// map[string]interface{} and map[string]string for tools.
func (t *Tools) CallTool(ctx context.Context, name string, args interface{}) (interface{}, error) {
	var params interface{}
	switch v := args.(type) {
	case string:
		// Attempt to unmarshal the string as JSON.
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, fmt.Errorf("failed to unmarshal args: %w", err)
		}
		params = m
	default:
		params = v
	}

	switch name {
	case SEARCH_TOOLS:
		// Handle both map[string]interface{} and map[string]string.
		if m, ok := params.(map[string]interface{}); ok {
			paramsMap := make(map[string]string)
			for key, value := range m {
				paramsMap[key] = fmt.Sprintf("%v", value)
			}
			return t.api.SearchTools(paramsMap)
		} else if m, ok := params.(map[string]string); ok {
			return t.api.SearchTools(m)
		}
		return nil, fmt.Errorf("invalid parameter type for %s", SEARCH_TOOLS)
	case CALL_TOOL:
		return t.api.CallTool(params)
	default:
		return nil, fmt.Errorf("unknown tool name: %s", name)
	}
}

// CallTools concurrently calls multiple tools while limiting the concurrency.
// toolCalls is a slice of openai.ChatCompletionMessageToolCallParam and
// this function returns a slice of openai.ToolMessage, which can be used directly
// in the OpenAI SDK.
func (t *Tools) CallTools(ctx context.Context, toolCalls []openai.ChatCompletionMessageToolCall) ([]openai.ChatCompletionToolMessageParam, error) {
	if len(toolCalls) == 0 {
		return []openai.ChatCompletionToolMessageParam{}, nil
	}

	sem := make(chan struct{}, t.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []openai.ChatCompletionToolMessageParam

	for _, tc := range toolCalls {
		wg.Add(1)
		go func(toolCall openai.ChatCompletionMessageToolCall) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Extract the tool's name and arguments.
			toolName := toolCall.Function.Name
			args := toolCall.Function.Arguments

			// Call the tool.
			res, err := t.CallTool(ctx, toolName, args)
			if err != nil {
				res = map[string]interface{}{"error": err.Error()}
			}

			contentBytes, jsonErr := json.Marshal(res)
			if jsonErr != nil {
				contentBytes = []byte(fmt.Sprintf(`{"error": "%s"}`, jsonErr.Error()))
			}

			message := openai.ToolMessage(toolCall.ID, string(contentBytes))

			mu.Lock()
			results = append(results, message)
			mu.Unlock()
		}(tc)
	}

	wg.Wait()
	return results, nil
}
