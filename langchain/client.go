package langchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log" // Added for more detailed logging
	"sync"

	"github.com/prashantgupta17/nlpromql/llm"
	"github.com/prashantgupta17/nlpromql/prompts"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

// LangChainClient implements the llm.LLMClient interface using LangChainGo.
type LangChainClient struct {
	llmModel llms.Model
}

// NewLangChainClient creates a new LangChainClient.
func NewLangChainClient(model llms.Model) *LangChainClient {
	return &LangChainClient{
		llmModel: model,
	}
}

// messagesToMessageContent converts []schema.ChatMessage to []llms.MessageContent.
// (This helper function remains unchanged from the previous version)
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
			// Fallback to generic, or handle as an error if strict type matching is required.
			log.Printf("Warning: Unknown chat message type '%s', defaulting to generic.", msg.GetType())
			role = llms.ChatMessageTypeGeneric
		}
		content[i] = llms.TextParts(role, msg.GetContent())
	}
	return content
}

// callLLMWithTool makes an LLM call expecting a specific tool to be used.
func (c *LangChainClient) callLLMWithTool(ctx context.Context, prompt string, toolToUse tools.Tool, outputStruct interface{}) error {
	if c.llmModel == nil {
		return errors.New("LangChain LLM model is not initialized")
	}

	messages := []schema.ChatMessage{
		schema.HumanChatMessage{Content: prompt},
	}

	// Using llms.WithTools to provide the LLM with the available tool.
	// If langchaingo offers a more forceful way like WithToolChoice("tool_name"), that could be an alternative.
	// For now, we rely on the prompt and the availability of the tool.
	llmOptions := []llms.CallOption{
		llms.WithTools([]tools.Tool{toolToUse}),
		// If a specific tool *must* be called, some LLM providers support a "tool_choice" parameter.
		// For OpenAI, this might look like:
		// llms.WithToolChoice(schema.ToolChoice{Type: schema.ToolTypeFunction, Function: schema.ToolFunction{Name: toolToUse.GetName()}})
		// This depends on the specific llms.Model implementation and langchaingo's support for it.
		// We will proceed assuming WithTools makes the tool available and the prompt guides its use.
	}

	resp, err := c.llmModel.GenerateContent(ctx, messagesToMessageContent(messages), llmOptions...)
	if err != nil {
		return fmt.Errorf("LangChain LLM GenerateContent call failed: %w. Prompt: %s", err, prompt)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("LLM returned no choices. Prompt: %s", prompt)
	}

	choice := resp.Choices[0]
	if len(choice.ToolCalls) > 0 {
		toolCall := choice.ToolCalls[0] // Assuming one tool call, as we specified one tool.
		if toolCall.FunctionCall.Name != toolToUse.GetName() {
			return fmt.Errorf("LLM called unexpected tool: got '%s', expected '%s'. Prompt: %s. Arguments received: %s",
				toolCall.FunctionCall.Name, toolToUse.GetName(), prompt, toolCall.FunctionCall.Arguments)
		}

		// Unmarshal the JSON arguments from the tool call.
		err = json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), outputStruct)
		if err != nil {
			return fmt.Errorf("error unmarshalling tool call arguments for tool '%s': %w. Raw arguments: '%s'. Prompt: %s",
				toolToUse.GetName(), err, toolCall.FunctionCall.Arguments, prompt)
		}
		return nil
	}

	// If no tool call was made, this is an issue.
	log.Printf("Warning: LLM did not make the expected tool call for tool '%s'. Prompt: '%s'. Response content: '%s'",
		toolToUse.GetName(), prompt, choice.Content)
	return fmt.Errorf("LLM did not make the expected tool call for '%s'. Check logs for more details. Prompt: %s", toolToUse.GetName(), prompt)
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
	metricSynonymTool := GetMetricSynonymsTool()

	for _, batch := range metricBatches {
		wg.Add(1)
		go func(metricMap map[string]string) {
			defer wg.Done()
			metricMapJSON, err := json.MarshalIndent(metricMap, "", "  ")
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("error marshalling metricMap for GetMetricSynonyms: %w", err)}
				return
			}
			prompt := fmt.Sprintf(prompts.MetricSynonymPrompt, string(metricMapJSON))
			var output MetricSynonymsToolOutput
			if err := c.callLLMWithTool(context.Background(), prompt, metricSynonymTool, &output); err != nil {
				resultsChan <- result{nil, fmt.Errorf("error calling LLM for metric synonyms (batch): %w", err)}
				return
			}
			resultsChan <- result{output.Synonyms, nil}
		}(batch)
	}

	wg.Wait()
	close(resultsChan)

	consolidatedSynonyms := make(map[string][]string)
	var firstError error
	for res := range resultsChan {
		if res.err != nil {
			if firstError == nil {
				firstError = res.err
			}
			// Continue collecting other results even if one batch fails
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

// GetLabelSynonyms gets synonyms for the given labels from the LLM in batches.
func (c *LangChainClient) GetLabelSynonyms(labelBatches [][]string) (map[string][]string, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}
	// Similar structure to GetMetricSynonyms
	type result struct {
		synonyms map[string][]string
		err      error
	}

	numBatches := len(labelBatches)
	resultsChan := make(chan result, numBatches)
	var wg sync.WaitGroup
	labelSynonymTool := GetLabelSynonymsTool()

	for _, batch := range labelBatches {
		wg.Add(1)
		go func(labelNames []string) {
			defer wg.Done()
			labelNamesJSON, err := json.MarshalIndent(labelNames, "", "  ")
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("error marshalling labelNames for GetLabelSynonyms: %w", err)}
				return
			}
			prompt := fmt.Sprintf(prompts.LabelSynonymPrompt, string(labelNamesJSON))
			var output LabelSynonymsToolOutput
			if err := c.callLLMWithTool(context.Background(), prompt, labelSynonymTool, &output); err != nil {
				resultsChan <- result{nil, fmt.Errorf("error calling LLM for label synonyms (batch): %w", err)}
				return
			}
			resultsChan <- result{output.Synonyms, nil}
		}(batch)
	}

	wg.Wait()
	close(resultsChan)

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
	processQueryTool := ProcessUserQueryTool()
	var output ProcessQueryToolOutput

	if err := c.callLLMWithTool(context.Background(), prompt, processQueryTool, &output); err != nil {
		return nil, fmt.Errorf("error calling LLM for ProcessUserQuery: %w", err)
	}

	resultMap := map[string]interface{}{
		"possible_metric_names": output.PossibleMetricNames,
		"possible_label_names":  output.PossibleLabelNames,
		"possible_label_values": output.PossibleLabelValues,
	}
	return resultMap, nil
}

// GetPromQLFromLLM gets PromQL queries from the LLM.
func (c *LangChainClient) GetPromQLFromLLM(userQuery string, relevantMetrics llm.RelevantMetricsMap, relevantLabels llm.RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error) {
	if c.llmModel == nil {
		return nil, errors.New("LangChain LLM model is not initialized")
	}

	relevantMetricsJSON, err := json.MarshalIndent(relevantMetrics, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantMetrics for GetPromQLFromLLM: %w", err)
	}
	relevantLabelsJSON, err := json.MarshalIndent(relevantLabels, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantLabels for GetPromQLFromLLM: %w", err)
	}
	relevantHistoryJSON, err := json.MarshalIndent(relevantHistory, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshalling relevantHistory for GetPromQLFromLLM: %w", err)
	}

	fullUserPrompt := fmt.Sprintf("#Relevant Metrics:
%s

#Relevant Labels:
%s

#Relevant History:
%s

#User Query:
%s",
		string(relevantMetricsJSON), string(relevantLabelsJSON), string(relevantHistoryJSON), userQuery)

	messages := []schema.ChatMessage{
		schema.SystemChatMessage{Content: prompts.SystemPrompt},
		schema.HumanChatMessage{Content: fullUserPrompt},
	}

	promqlGenTool := GeneratePromQLTool()
	llmOptions := []llms.CallOption{
		llms.WithTools([]tools.Tool{promqlGenTool}),
		// As with callLLMWithTool, a more forceful tool_choice could be used if available/necessary.
		// e.g., llms.WithToolChoice(schema.ToolChoice{Type: schema.ToolTypeFunction, Function: schema.ToolFunction{Name: promqlGenTool.GetName()}})
	}

	resp, err := c.llmModel.GenerateContent(context.Background(), messagesToMessageContent(messages), llmOptions...)
	if err != nil {
		return nil, fmt.Errorf("LangChain LLM GenerateContent call failed for GetPromQLFromLLM: %w. Prompt: %s", err, fullUserPrompt)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices for GetPromQLFromLLM. Prompt: %s", fullUserPrompt)
	}

	choice := resp.Choices[0]
	var output GeneratePromQLToolOutput

	if len(choice.ToolCalls) > 0 {
		toolCall := choice.ToolCalls[0]
		if toolCall.FunctionCall.Name != promqlGenTool.GetName() {
			return nil, fmt.Errorf("LLM called unexpected tool for GetPromQLFromLLM: got '%s', expected '%s'. Prompt: %s. Args: %s",
				toolCall.FunctionCall.Name, promqlGenTool.GetName(), fullUserPrompt, toolCall.FunctionCall.Arguments)
		}
		err = json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &output)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling tool call arguments for GetPromQLFromLLM (tool '%s'): %w. Raw arguments: '%s'. Prompt: %s",
				promqlGenTool.GetName(), err, toolCall.FunctionCall.Arguments, fullUserPrompt)
		}
	} else {
		log.Printf("Warning: LLM did not make the expected tool call for GetPromQLFromLLM (tool '%s'). Prompt: '%s'. Response content: '%s'",
			promqlGenTool.GetName(), fullUserPrompt, choice.Content)
		return nil, fmt.Errorf("LLM did not make the expected tool call for GetPromQLFromLLM. Check logs. Prompt: %s", fullUserPrompt)
	}

	var promqlStrings []string
	for _, q := range output.Queries {
		promqlStrings = append(promqlStrings, q.PromQL)
	}
	return promqlStrings, nil
}

// Ensure LangChainClient implements the llm.LLMClient interface.
var _ llm.LLMClient = (*LangChainClient)(nil)
