package langchain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings" // Added for cleanLLMResponse
	"sync"

	"github.com/prashantgupta17/nlpromql/llm"
	"github.com/prashantgupta17/nlpromql/prompts"
	"github.com/tmc/langchaingo/llms"
	// Dependencies for specific llms.Model implementations are managed in main.go
)

// cleanLLMResponse trims whitespace and removes markdown code fences (```json ... ``` or ``` ... ```)
// from the LLM response string.
func cleanLLMResponse(response string) string {
	cleaned := strings.TrimSpace(response)
	if strings.HasPrefix(cleaned, "```json") && strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned) // Trim again after removing fences
	} else if strings.HasPrefix(cleaned, "```") && strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned) // Trim again after removing fences
	}
	return cleaned
}

// LangChainClient implements the llm.LLMClient interface using LangChainGo.
type LangChainClient struct {
	llmModel llms.Model // Generic LangChainGo LLM model
}

// NewLangChainClient creates a new LangChainClient.
// The specific model (e.g., OpenAI, Anthropic) should be initialized and passed here.
func NewLangChainClient(model llms.Model) *LangChainClient {
	return &LangChainClient{
		llmModel: model,
	}
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
			response, err := c.llmModel.Call(context.Background(), prompt)
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("LangChain LLM call failed: %w", err)}
				return
			}
			cleanedResponse := cleanLLMResponse(response)
			var synonymsBatch map[string][]string
			if err := json.Unmarshal([]byte(cleanedResponse), &synonymsBatch); err != nil {
				resultsChan <- result{nil, fmt.Errorf("error unmarshalling LLM response: %w. Raw response: %s", err, response)}
				return
			}
			resultsChan <- result{synonymsBatch, nil}
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
			// Continue processing other results to potentially gather partial data,
			// but the first error will be returned.
		} else if res.synonyms != nil {
			for key, value := range res.synonyms {
				consolidatedSynonyms[key] = append(consolidatedSynonyms[key], value...)
				// TODO: Consider if duplicate synonyms across batches should be handled (e.g., deduped).
				// For now, appending all.
			}
		}
	}

	if firstError != nil {
		return nil, firstError // Return the first error encountered
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
			response, err := c.llmModel.Call(context.Background(), prompt)
			if err != nil {
				resultsChan <- result{nil, fmt.Errorf("LangChain LLM call failed: %w", err)}
				return
			}
			cleanedResponse := cleanLLMResponse(response)
			var synonymsBatch map[string][]string
			if err := json.Unmarshal([]byte(cleanedResponse), &synonymsBatch); err != nil {
				resultsChan <- result{nil, fmt.Errorf("error unmarshalling LLM response: %w. Raw response: %s", err, response)}
				return
			}
			resultsChan <- result{synonymsBatch, nil}
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
				consolidatedSynonyms[key] = append(consolidatedSynonyms[key], value...)
				// TODO: Deduplication of synonyms if needed
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

	// Corrected: llms.Call is a method on the model instance: c.llmModel.Call
	response, err := c.llmModel.Call(context.Background(), prompt) // Removed c.llmModel from args
	if err != nil {
		return nil, fmt.Errorf("LangChain LLM call failed: %w", err)
	}
	cleanedResponse := cleanLLMResponse(response)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		return nil, fmt.Errorf("error unmarshalling LLM response: %w. Raw response: %s", err, response)
	}

	return result, nil
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

	// Construct the user prompt part for GetPromQLFromLLM
	// This follows the structure observed in openai/client.go's newFunction
	userPromptForPromQL := fmt.Sprintf("#Relevant Metrics:\n%s\n\n#Relevant Labels:\n%s\n\n#Relevant History:\n%s\n\n#User Query:\n%s",
		string(relevantMetricsJSON),
		string(relevantLabelsJSON),
		string(relevantHistoryJSON),
		userQuery,
	)

	// For LangChainGo, the system prompt is often handled as part of the model's initialization
	// or via specific options in the Call/GenerateContent methods.
	// Here, we'll pass it as part of the prompt itself if using a simple llms.Call.
	// If using a chat model, it would be a SystemChatMessage.
	// This assumes the model can take a combined system + user prompt.
	// A more sophisticated implementation would use llms.GenerateContent with specific message types.

	// According to LangchainGo docs, for models that support SystemPrompt
	// it should be passed as an option if the specific LLM wrapper supports it
	// or as the first message in a chat sequence.
	// For a generic llms.Call, we might prepend it to the user prompt.
	// However, the `prompts.SystemPrompt` is quite large and might be better handled
	// by specific model capabilities (e.g. `llms.WithSystemPrompt` if available or by using `llms.ChatMessage` types).

	// For now, we will use llms.GenerateContent which allows specifying a slice of llms.MessageContent.
	// We will create two messages: one for the system prompt and one for the user prompt.

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, prompts.SystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPromptForPromQL),
	}

	options := []llms.CallOption{} // Add temperature, max tokens etc. here if needed.

	// Using GenerateContent for better control over message types (system vs user)
	// Note: Not all models in LangchainGo might support the System message type in the same way.
	// This part might need adjustment based on the specific llms.Model being used.
	// For example, some models might expect the system prompt as a specific field during initialization or call.
	// Corrected: llms.GenerateContent is a method on the model instance: c.llmModel.GenerateContent
	contentResponse, err := c.llmModel.GenerateContent(context.Background(), messages, options...) // Removed c.llmModel from args
	if err != nil {
		return nil, fmt.Errorf("LangChain LLM GenerateContent call failed: %w", err)
	}

	if len(contentResponse.Choices) == 0 {
		return nil, errors.New("LLM returned no choices")
	}

	response := contentResponse.Choices[0].Content
	cleanedResponse := cleanLLMResponse(response)

	var promqlOptions []struct {
		PromQL string  `json:"promql"`
		Score  float64 `json:"score"`
		// metric_label_pairs is ignored for now as we only need PromQL strings
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &promqlOptions); err != nil {
		return nil, fmt.Errorf("error unmarshalling LLM response for PromQL: %w. Raw response: %s", err, response)
	}

	// Sort by score (descending) - already handled by prompt, but good practice
	// The prompt asks the LLM to sort, but we can re-sort if needed.
	// For now, we trust the LLM's sorting based on the prompt.

	var sortedPromqlStrings []string
	for _, option := range promqlOptions {
		sortedPromqlStrings = append(sortedPromqlStrings, option.PromQL)
	}

	return sortedPromqlStrings, nil
}

// Ensure LangChainClient implements the llm.LLMClient interface.
var _ llm.LLMClient = (*LangChainClient)(nil)
