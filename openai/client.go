package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/prashantgupta17/nlpromql/prompts"

	openai "github.com/sashabaranov/go-openai"
)

// MetricInfo holds information about a metric, including its labels.
type RelevantMetricInfo struct {
	MatchScore int                          `json:"match_score"`
	Labels     map[string]RelevantLabelInfo `json:"labels"`
}

// LabelInfo holds information about a label, including its values.
type RelevantLabelInfo struct {
	MatchScore int                    `json:"match_score"`
	Values     map[string]interface{} `json:"values"`
}

// MetricLabelMap represents a map of metric names to their labels and values (sets).
type RelevantMetricsMap map[string]RelevantMetricInfo // Nested map: metric -> label -> value set

// LabelValueMap represents a map of label names to their values (sets).
type RelevantLabelsMap map[string]RelevantLabelInfo // Nested map: label -> value set

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
		fmt.Println(batch)
		batchJson, err := json.MarshalIndent(batch, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling metric batch: %v", err)
		}
		// Use CreateCompletion instead of CreateChatCompletion
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
		fmt.Println(resp.Choices)
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
		fmt.Println(batch)
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
	fmt.Println(resp.Choices[0].Message.Content)
	// Parse and return the response (adjust based on your desired structure)
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
	}
	return result, nil
}

// getPromQLFromLLM generates PromQL queries based on user input and context.
func (c *OpenAIClient) GetPromQLFromLLM(userQuery string, relevantMetrics RelevantMetricsMap,
	relevantLabels RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error) {
	// Prepare input data for LLM

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
	fmt.Println("Prompt:", prompt)
	// Send data to LLM for PromQL generation
	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: c.llmSystemPrompt}, // Use your system prompt
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
			Temperature: 0.3,
			MaxTokens:   2000,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %v", err)
	}

	// Parse the response into PromQL options
	var promqlOptions []map[string]interface{}
	fmt.Println(resp.Choices[0].Message.Content)
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &promqlOptions); err != nil {
		return nil, fmt.Errorf("error parsing OpenAI response: %v", err)
	}

	// Sort promqlOptions by score
	sort.Slice(promqlOptions, func(i, j int) bool {
		scoreI := promqlOptions[i]["score"].(float64)
		scoreJ := promqlOptions[j]["score"].(float64)
		return scoreI > scoreJ
	})

	// Extract promql values into a new string array
	var sortedPromqlOptions []string
	for _, option := range promqlOptions {
		promql := option["promql"].(string)
		sortedPromqlOptions = append(sortedPromqlOptions, promql)
	}

	return sortedPromqlOptions, nil
}
