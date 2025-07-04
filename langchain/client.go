package langchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/prashantgupta17/nlpromql/llm"
	"github.com/prashantgupta17/nlpromql/prompts" // Keep for prompts, as they are still used to formulate the user message to the LLM
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema" // Commonly used for message types
	"github.com/tmc/langchaingo/tools"  // Import the tools package
)

// LangChainClient implements the llm.LLMClient interface using LangChainGo.
type LangChainClient struct {
	llmModel llms.Model // Generic LangChainGo LLM model
}

// NewLangChainClient creates a new LangChainClient.
func NewLangChainClient(model llms.Model) *LangChainClient {
	return &LangChainClient{
		llmModel: model,
	}
}

// Helper function to make LLM calls with tools and parse the response.
// This assumes the LLM will return a tool call and we need to parse the arguments of that call.
func (c *LangChainClient) callLLMWithTool(ctx context.Context, prompt string, tool tools.Tool, outputStruct interface{}) error {
	if c.llmModel == nil {
		return errors.New("LangChain LLM model is not initialized")
	}

	// Some models might require the system prompt to be passed differently,
	// but for now, we assume it's part of the main prompt or handled by the model.
	messages := []schema.ChatMessage{
		// schema.SystemChatMessage{Content: prompts.SystemPrompt}, // System prompt might be too general for all tools here.
		// The specific prompt for each function (e.g., MetricSynonymPrompt) acts as the main instruction.
		schema.HumanChatMessage{Content: prompt},
	}

	// GenerateContent is often used for chat models and tool calling
	// We pass the tool definition to the LLM.
	// The exact way to pass tools (WithToolChoice, WithTools) can vary.
	// We'll assume a generic `llms.WithTools` or similar option.
	// If `GenerateContent` is not the right method for a specific non-chat model,
	// this might need adjustment to use `Call` with appropriate options.
	resp, err := c.llmModel.GenerateContent(ctx, messagesToMessageContent(messages), llms.WithTools([]tools.Tool{tool}))
	if err != nil {
		return fmt.Errorf("LangChain LLM GenerateContent call failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Content == "" && len(resp.Choices[0].ToolCalls) == 0 {
		return errors.New("LLM returned no content or tool calls")
	}

	// Expecting the LLM to make a tool call
	if len(resp.Choices[0].ToolCalls) > 0 {
		toolCall := resp.Choices[0].ToolCalls[0] // Assuming one tool call
		if toolCall.FunctionCall.Name != tool.GetName() {
			return fmt.Errorf("LLM called unexpected tool: %s, expected %s", toolCall.FunctionCall.Name, tool.GetName())
		}
		// The arguments of the tool call should be the JSON string we need
		return json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), outputStruct)
	}

	// Fallback or error if no tool call was made but was expected
	// Sometimes, the response might be in Content if the LLM doesn't make a tool call but just returns JSON.
	// However, the goal of tool calling is to enforce the structure via the tool mechanism.
	// If the LLM is not consistently making tool calls, the prompt might need adjustment or
	// the model might not fully support forced tool calling in the way langchaingo implements it.
	// For now, we'll strictly expect a tool call.
	// If the content contains the JSON, it means the LLM didn't use the tool but still provided a response.
	// This was the original problem. By specifying a tool, we expect the LLM to fill that tool's arguments.
	// If `resp.Choices[0].Content` is not empty, it could be the raw JSON, or the markdown error.
	// The whole point of using tools is to get the arguments from `toolCall.FunctionCall.Arguments`.

	return fmt.Errorf("LLM did not make the expected tool call. Response content: %s", resp.Choices[0].Content)
}

// Helper to convert schema.ChatMessage to llms.MessageContent
func messagesToMessageContent(messages []schema.ChatMessage) []llms.MessageContent {
	content := make([]llms.MessageContent, len(messages))
	for i, msg := range messages {
		var role llms.ChatMessageType
		switch msg.GetType() {
		case schema.ChatMessageTypeSystem:
			role = llms.ChatMessageTypeSystem
		case schema.ChatMessageTypeAI:
			role = llms.ChatMessageTypeAI
		case schema.ChatMessageTypeHuman:
			role = llms.ChatMessageTypeHuman
		case schema.ChatMessageTypeTool:
			role = llms.ChatMessageTypeTool
		case schema.ChatMessageTypeGeneric:
			role = llms.ChatMessageTypeGeneric
		default:
			role = llms.ChatMessageTypeGeneric // Or handle error
		}
		content[i] = llms.TextParts(role, msg.GetContent())
		// If there are tool calls associated with the message, they should be added here.
		// For now, assuming simple text content for input messages.
	}
	return content
}


// GetMetricSynonyms gets synonyms for the given metrics from the LLM in batches.
func (c *LangChainClient) GetMetricSynonyms(metricBatches []map[string]string) (map[string][]string, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}

	type result struct {
		synonyms map[string][]string
		err      error
	}

	numBatches := len(metricBatches)
	resultsChan := make(chan result, numBatches)
	var wg sync.WaitGroup

	metricTool := GetMetricSynonymsTool()

	for _, batch := range metricBatches {
		wg.Add(1)
		go func(metricMap map[string]string) {
			defer wg.Done()

			metricMapJSON, err := json.MarshalIndent(metricMap, "", "  ")
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("error marshalling metricMap: %w", err)}
				return
			}

			prompt := fmt.Sprintf(prompts.MetricSynonymPrompt, string(metricMapJSON))
			var output MetricSynonymsToolOutput
			// Use the new helper function for the LLM call
			if err := c.callLLMWithTool(context.Background(), prompt, metricTool, &output); err != nil {
				// Adding raw response for debugging, but it might be complex if it's a structured error from callLLMWithTool
				resultsChan <- result{nil, fmt.Errorf("error calling LLM with tool: %w", err)}
				return
			}
			resultsChan <- result{output.Synonyms, nil}
		}(batch)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	consolidatedSynonyms := make(map[string][]string)
	var firstError error

	for res := range resultsChan {
		if res.err != nil {
			if firstError == nil {
				firstError = res.err
			}
		} else if res.synonyms != nil {
			for key, value := range res.synonyms {
				// Ensure no nil assignment if a key somehow has nil synonyms
				if consolidatedSynonyms[key] == nil {
					consolidatedSynonyms[key] = []string{}
				}
				consolidatedSynonyms[key] = append(consolidatedSynonyms[key], value...)
			}
		}
	}

	if firstError != nil {
		return nil, firstError
	}

	return consolidatedSynonyms, nil
}

// GetLabelSynonyms gets synonyms for the given labels from the LLM in batches.
func (c *LangChainClient) GetLabelSynonyms(labelBatches [][]string) (map[string][]string, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}

	type result struct {
		synonyms map[string][]string
		err      error
	}

	numBatches := len(labelBatches)
	resultsChan := make(chan result, numBatches)
	var wg sync.WaitGroup

	labelTool := GetLabelSynonymsTool()

	for _, batch := range labelBatches {
		wg.Add(1)
		go func(labelNames []string) {
			defer wg.Done()

			labelNamesJSON, err := json.MarshalIndent(labelNames, "", "  ")
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("error marshalling labelNames: %w", err)}
				return
			}

			prompt := fmt.Sprintf(prompts.LabelSynonymPrompt, string(labelNamesJSON))
			var output LabelSynonymsToolOutput
			if err := c.callLLMWithTool(context.Background(), prompt, labelTool, &output); err != nil {
				resultsChan <- result{nil, fmt.Errorf("error calling LLM with tool: %w", err)}
				return
			}
			resultsChan <- result{output.Synonyms, nil}
		}(batch)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	consolidatedSynonyms := make(map[string][]string)
	var firstError error

	for res := range resultsChan {
		if res.err != nil {
			if firstError == nil {
				firstError = res.err
			}
		} else if res.synonyms != nil {
			for key, value := range res.synonyms {
				if consolidatedSynonyms[key] == nil {
					consolidatedSynonyms[key] = []string{}
				}
				consolidatedSynonyms[key] = append(consolidatedSynonyms[key], value...)
			}
		}
	}

	if firstError != nil {
		return nil, firstError
	}

	return consolidatedSynonyms, nil
}

// ProcessUserQuery processes the user query and returns relevant information.
func (c *LangChainClient) ProcessUserQuery(userQuery string) (map[string]interface{}, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}

	prompt := fmt.Sprintf(prompts.ProcessQueryPrompt, userQuery)
	queryTool := ProcessUserQueryTool()
	var output ProcessQueryToolOutput

	if err := c.callLLMWithTool(context.Background(), prompt, queryTool, &output); err != nil {
		return nil, fmt.Errorf("error calling LLM with tool for ProcessUserQuery: %w", err)
	}

	// Convert ProcessQueryToolOutput to map[string]interface{} for compatibility with the existing interface.
	// This could also be refactored to return the struct type directly if the interface can be changed.
	resultMap := map[string]interface{}{
		"possible_metric_names": output.PossibleMetricNames,
		"possible_label_names":  output.PossibleLabelNames,
		"possible_label_values": output.PossibleLabelValues,
	}

	return resultMap, nil
}

// GetPromQLFromLLM gets PromQL queries from the LLM based on the user query and relevant context.
func (c *LangChainClient) GetPromQLFromLLM(userQuery string, relevantMetrics llm.RelevantMetricsMap, relevantLabels llm.RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}

	relevantMetricsJSON, err := json.MarshalIndent(relevantMetrics, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantMetrics: %w", err)
	}
	relevantLabelsJSON, err := json.MarshalIndent(relevantLabels, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantLabels: %w", err)
	}
	relevantHistoryJSON, err := json.MarshalIndent(relevantHistory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantHistory: %w", err)
	}

	// The user prompt for GetPromQLFromLLM is complex and constructed from several parts.
	// The SystemPrompt is also crucial here.
	fullUserPrompt := fmt.Sprintf("#Relevant Metrics:
%s

#Relevant Labels:
%s

#Relevant History:
%s

#User Query:
%s",
		string(relevantMetricsJSON),
		string(relevantLabelsJSON),
		string(relevantHistoryJSON),
		userQuery,
	)

	// For this specific function, the SystemPrompt is highly relevant.
	// We will construct messages including the system prompt.
	messages := []schema.ChatMessage{
		schema.SystemChatMessage{Content: prompts.SystemPrompt},
		schema.HumanChatMessage{Content: fullUserPrompt},
	}

	promqlTool := GeneratePromQLTool()
	var output GeneratePromQLToolOutput

	// We need to use a slightly different helper or inline the logic if system prompt handling is different.
	// For now, let's adapt the callLLMWithTool logic here.
	// This is because `callLLMWithTool` was made generic and doesn't include specific system prompts.

	generateContentMessages := messagesToMessageContent(messages)

	resp, err := c.llmModel.GenerateContent(context.Background(), generateContentMessages, llms.WithTools([]tools.Tool{promqlTool}))
	if err != nil {
		return nil, fmt.Errorf("LangChain LLM GenerateContent call failed for GetPromQLFromLLM: %w", err)
	}

	if len(resp.Choices) == 0 || (resp.Choices[0].Content == "" && len(resp.Choices[0].ToolCalls) == 0) {
		return nil, errors.New("LLM returned no content or tool calls for GetPromQLFromLLM")
	}

	if len(resp.Choices[0].ToolCalls) > 0 {
		toolCall := resp.Choices[0].ToolCalls[0]
		if toolCall.FunctionCall.Name != promqlTool.GetName() {
			return nil, fmt.Errorf("LLM called unexpected tool: %s, expected %s for GetPromQLFromLLM", toolCall.FunctionCall.Name, promqlTool.GetName())
		}
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &output); err != nil {
			return nil, fmt.Errorf("error unmarshalling tool call arguments for GetPromQLFromLLM: %w. Raw args: %s", err, toolCall.FunctionCall.Arguments)
		}
	} else {
		// Fallback: if no tool call, try to parse content directly (though this is what we want to avoid)
		// This might indicate the prompt needs to be more forceful about using the tool.
		// Or the model doesn't support the tool use as expected with the given prompt.
		// Log this situation if it happens.
		// For now, error out if tool call is not made.
		return nil, fmt.Errorf("LLM did not make the expected tool call for GetPromQLFromLLM. Response content: %s", resp.Choices[0].Content)
	}

	var promqlStrings []string
	for _, q := range output.Queries {
		promqlStrings = append(promqlStrings, q.PromQL)
	}

	return promqlStrings, nil
}

// Ensure LangChainClient implements the llm.LLMClient interface.
var _ llm.LLMClient = (*LangChainClient)(nil)
