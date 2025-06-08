package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"github.com/prashantgupta17/nlpromql/prompts"

	openai "github.com/sashabaranov/go-openai"
	"github.com/prashantgupta17/nlpromql/llm"
)

// Compile-time check to ensure OpenAIClient implements llm.LLMClient interface
var _ llm.LLMClient = (*OpenAIClient)(nil)

type OpenAIClient struct {
	client              *openai.Client
	llmSystemPrompt     string
	processQueryPrompt  string
	metricSynonymPrompt string
	labelSynonymPrompt  string
}

func NewOpenAIClient() (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	return &OpenAIClient{
		client:              openai.NewClient(apiKey),
		llmSystemPrompt:     prompts.SystemPrompt,
		processQueryPrompt:  prompts.ProcessQueryPrompt,
		metricSynonymPrompt: prompts.MetricSynonymPrompt,
		labelSynonymPrompt:  prompts.LabelSynonymPrompt,
	}, nil
}

func NewOpenAIClientWithKey(apiKey string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("open AI api key is empty")
	}

	return &OpenAIClient{
		client:              openai.NewClient(apiKey),
		llmSystemPrompt:     prompts.SystemPrompt,
		processQueryPrompt:  prompts.ProcessQueryPrompt,
		metricSynonymPrompt: prompts.MetricSynonymPrompt,
		labelSynonymPrompt:  prompts.LabelSynonymPrompt,
	}, nil
}

// getMetricSynonyms fetches metric synonyms using the OpenAI API.
func (c *OpenAIClient) GetMetricSynonyms(metricMap map[string]string) (map[string][]string, error) {
	batchSize := 20
	allSynonyms := make(map[string][]string)
	keys := make([]string, 0, len(metricMap))
	for k := range metricMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := 0; i < len(keys); i += batchSize {
		batch := make(map[string]string)
		for j := i; j < i+batchSize && j < len(keys); j++ {
			batch[keys[j]] = metricMap[keys[j]]
		}
		batchJson, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling metric batch: %v", err)
		}

		resp, err := c.client.CreateCompletion(
			context.Background(),
			openai.CompletionRequest{
				Model:       openai.GPT3Dot5TurboInstruct,
				Prompt:      fmt.Sprintf(c.metricSynonymPrompt, string(batchJson)), // Notice the use of a pointer to the prompt string
				Temperature: 0.3,
				MaxTokens:   2000,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("OpenAI API error: %v", err)
		}

		// Parse the response to get the synonyms
		rawResponseText := resp.Choices[0].Text
		var batchSynonyms map[string][]string
		if err := json.Unmarshal([]byte(rawResponseText), &batchSynonyms); err != nil {
			return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
		}

		// Update allSynonyms with the synonyms from this batch
		for metric, synonyms := range batchSynonyms {
			allSynonyms[metric] = synonyms
		}
	}
	return allSynonyms, nil
}

// getLabelSynonyms fetches label synonyms using the OpenAI API.
func (c *OpenAIClient) GetLabelSynonyms(labelNames []string) (map[string][]string, error) {
	batchSize := 20 // Adjust batch size as needed
	allSynonyms := make(map[string][]string)

	for i := 0; i < len(labelNames); i += batchSize {
		batch := labelNames[i:int(math.Min(float64(i+batchSize), float64(len(labelNames))))]
		batchJson, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling label batch: %v", err)
		}
		resp, err := c.client.CreateCompletion(
			context.Background(),
			openai.CompletionRequest{
				Model:       openai.GPT3Dot5TurboInstruct,                         // Or the appropriate model
				Prompt:      fmt.Sprintf(c.labelSynonymPrompt, string(batchJson)), // Use your label synonym prompt
				Temperature: 0.5,                                                  // Adjust as needed
				MaxTokens:   2000,
			},
		)

		if err != nil {
			return nil, fmt.Errorf("OpenAI API error: %v", err)
		}

		// Parse the response to get the synonyms
		rawResponseText := resp.Choices[0].Text

		var batchSynonyms map[string][]string
		if err := json.Unmarshal([]byte(rawResponseText), &batchSynonyms); err != nil {
			return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
		}

		// Update allSynonyms with the synonyms from this batch
		for label, synonyms := range batchSynonyms {
			allSynonyms[label] = synonyms
		}
	}

	return allSynonyms, nil
}

// processUserQuery processes user queries using the OpenAI API.
func (c *OpenAIClient) ProcessUserQuery(userQuery string) (map[string]interface{}, error) {
	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo, // or whichever model you're using
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
				{Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf(c.processQueryPrompt, userQuery)}, // Use your prompt
			},
			Temperature: 0.2,
			MaxTokens:   1000, // Or your preferred value
		},
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}
	// Parse and return the response (adjust based on your desired structure)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
	}
	return result, nil
}

// getPromQLFromLLM generates PromQL queries based on user input and context.
func (c *OpenAIClient) GetPromQLFromLLM(userQuery string, relevantMetrics llm.RelevantMetricsMap,
	relevantLabels llm.RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error) {
	var promQLs []map[string]interface{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Split relevantMetrics into batches
	metricKeys := make([]string, 0, len(relevantMetrics))
	for k := range relevantMetrics {
		metricKeys = append(metricKeys, k)
	}
	sort.Strings(metricKeys)
	batchSize := 5
	numBatches := int(math.Ceil(float64(len(metricKeys)) / float64(batchSize)))
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := int(math.Min(float64(start+batchSize), float64(len(metricKeys))))
		batchMetrics := make(llm.RelevantMetricsMap)
		for _, key := range metricKeys[start:end] {
			batchMetrics[key] = relevantMetrics[key]
		}
		wg.Add(1)
		go func(metrics llm.RelevantMetricsMap) {
			defer wg.Done()
			promQLBatch, err := newFunction(metrics, llm.RelevantLabelsMap{}, relevantHistory, userQuery, c)
			if err != nil {
				// Handle error
				return
			}
			mu.Lock()
			promQLs = append(promQLs, promQLBatch...)
			mu.Unlock()
		}(batchMetrics)
	}

	// Split relevantLabels into batches
	labelKeys := make([]string, 0, len(relevantLabels))
	for k := range relevantLabels {
		labelKeys = append(labelKeys, k)
	}
	sort.Strings(labelKeys)
	numBatches = int(math.Ceil(float64(len(labelKeys)) / float64(batchSize)))
	for i := 0; i < numBatches; i++ {
		start := i * batchSize
		end := int(math.Min(float64(start+batchSize), float64(len(labelKeys))))
		batchLabels := make(llm.RelevantLabelsMap)
		for _, key := range labelKeys[start:end] {
			batchLabels[key] = relevantLabels[key]
		}
		wg.Add(1)
		go func(labels llm.RelevantLabelsMap) {
			defer wg.Done()
			promQLBatch, err := newFunction(llm.RelevantMetricsMap{}, labels, relevantHistory, userQuery, c)
			if err != nil {
				// Handle error
				return
			}
			mu.Lock()
			promQLs = append(promQLs, promQLBatch...)
			mu.Unlock()
		}(batchLabels)
	}

	wg.Wait()
	sort.Slice(promQLs, func(i, j int) bool {
		scoreI := promQLs[i]["score"].(float64)
		scoreJ := promQLs[j]["score"].(float64)
		return scoreI > scoreJ
	})

	var sortedPromqlOptions []string
	for _, option := range promQLs {
		promql := option["promql"].(string)
		sortedPromqlOptions = append(sortedPromqlOptions, promql)
	}

	return sortedPromqlOptions, nil
}

func newFunction(relevantMetrics llm.RelevantMetricsMap, relevantLabels llm.RelevantLabelsMap,
	relevantHistory map[string]interface{}, userQuery string, c *OpenAIClient) ([]map[string]interface{}, error) {
	prompt := fmt.Sprintf("#Relevant Metrics:\n%s\n\n#Relevant Labels:\n%s\n\n#Relevant History:\n%s\n\n#User Query:\n%s",
		func() string {
			relevantMetricsJSON, _ := json.MarshalIndent(relevantMetrics, "", "  ")
			return string(relevantMetricsJSON)
		}(),
		func() string {
			relevantLabelsJSON, _ := json.MarshalIndent(relevantLabels, "", "  ")
			return string(relevantLabelsJSON)
		}(),
		func() string {
			relevantHistoryJSON, _ := json.MarshalIndent(relevantHistory, "", "  ")
			return string(relevantHistoryJSON)
		}(),
		userQuery,
	)

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: c.llmSystemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
			Temperature: 0.3,
			MaxTokens:   2000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	var promqlOptions []map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &promqlOptions); err != nil {
		return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
	}
	return promqlOptions, nil
}
