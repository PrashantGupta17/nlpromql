package langchain_test

import (
	"context"
	"errors"
	"testing"
	"strings" // Added for strings.Contains
	"fmt"     // Added for fmt.Sprintf in ProcessUserQuery test

	"github.com/prashantgupta17/nlpromql/langchain" // Package to be tested
	"github.com/tmc/langchaingo/llms"
	// "github.com/tmc/langchaingo/schema" // Removed unused import
	"encoding/json" // Added for GetPromQLFromLLM test
	"github.com/prashantgupta17/nlpromql/llm" // Added for GetPromQLFromLLM test (llm.RelevantMetricsMap etc.)
	"github.com/prashantgupta17/nlpromql/prompts" // Added for GetPromQLFromLLM test (prompts.SystemPrompt)
)

	"reflect" // Added for DeepEqual
	"sync"    // Added for mutex in mock
)

// mockLLM is a mock implementation of the llms.Model interface for testing.
type mockLLM struct {
	GenerateContentFunc func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error)

	// For Call based methods like GetMetricSynonyms and GetLabelSynonyms
	mu          sync.Mutex
	CallInputs  []string // Stores the prompts received by Call
	CallResponses map[string]string // Map prompt to a JSON response string
	CallErrors    map[string]error  // Map prompt to an error
	DefaultCallResponse string
	DefaultCallError error
}

// Call implements the llms.Model interface.
func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallInputs = append(m.CallInputs, prompt)

	if err, ok := m.CallErrors[prompt]; ok {
		return "", err
	}
	if resp, ok := m.CallResponses[prompt]; ok {
		return resp, nil
	}
	return m.DefaultCallResponse, m.DefaultCallError
}

// GenerateContent implements the llms.Model interface.
func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, messages, options...)
	}
	// To satisfy the interface if only Call is being tested
	return nil, errors.New("GenerateContentFunc not implemented in mockLLM")
}

// GetNumTokens implements the llms.Model interface - minimal implementation.
func (m *mockLLM) GetNumTokens(text string) int {
	return len(text)
}

// GetIdentifiers implements the llms.Model interface - minimal implementation
func (m *mockLLM) GetIdentifiers() []string {
	return []string{"mockLLM"}
}

var _ llms.Model = (*mockLLM)(nil)

func (m *mockLLM) ResetCallTracking() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallInputs = []string{}
	m.CallResponses = make(map[string]string)
	m.CallErrors = make(map[string]error)
	m.DefaultCallResponse = ""
	m.DefaultCallError = nil
}

// TestNewLangChainClient tests the constructor for LangChainClient.
func TestNewLangChainClient(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)
	if client == nil {
		t.Error("NewLangChainClient returned nil")
	}
}

func TestLangChainClient_ProcessUserQuery(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	tests := []struct {
		name          string
		userQuery     string
		mockResponse  string
		mockError     error
		expectedMap   map[string]interface{}
		expectedError string
	}{
		{
			name:         "successful response",
			userQuery:    "show me cpu usage",
			mockResponse: `{"possible_metric_names": ["cpu_usage", "system_cpu_usage"], "possible_label_names": ["instance", "host"]}`,
			mockError:    nil,
			expectedMap:  map[string]interface{}{"possible_metric_names": []interface{}{"cpu_usage", "system_cpu_usage"}, "possible_label_names": []interface{}{"instance", "host"}},
		},
		{
			name:          "llm returns error",
			userQuery:     "show me ram usage",
			mockError:     errors.New("llm simulated error for process query"),
			expectedError: "LangChain LLM call failed: llm simulated error for process query",
		},
		{
			name:          "malformed json response",
			userQuery:     "show me disk io",
			mockResponse:  `{"possible_metric_names": ["disk_io"]`, // Missing closing brace
			mockError:     nil,
			expectedError: "error unmarshalling LLM response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.CallFunc = func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
				// TODO: Could add prompt validation here if needed
				return tt.mockResponse, tt.mockError
			}

			resultMap, err := client.ProcessUserQuery(tt.userQuery)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedError, err)
				}
				return // Don't check map if error is expected
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Basic check for map length and a few keys. Deep comparison for maps of interface{} can be complex.
			if len(resultMap) != len(tt.expectedMap) {
				t.Errorf("expected map length %d, got %d. Result: %v, Expected: %v", len(tt.expectedMap), len(resultMap), resultMap, tt.expectedMap)
			}
			for key, expectedValue := range tt.expectedMap {
				actualValue, ok := resultMap[key]
				if !ok {
					t.Errorf("expected key '%s' not found in result map", key)
					continue
				}
				// This is a simplified comparison. For robust comparison of []interface{}, reflect.DeepEqual is better.
				// For now, comparing string representations if they are slices.
				expectedValSlice, eok := expectedValue.([]interface{})
				actualValSlice, aok := actualValue.([]interface{})
				if eok && aok {
					if len(expectedValSlice) != len(actualValSlice) {
						t.Errorf("for key '%s', expected slice length %d, got %d", key, len(expectedValSlice), len(actualValSlice))
						continue
					}
					for i_val := range expectedValSlice {
						if expectedValSlice[i_val] != actualValSlice[i_val] {
							t.Errorf("for key '%s' at slice index %d, expected '%v', got '%v'", key, i_val, expectedValSlice[i_val], actualValSlice[i_val])
						}
					}
				} else if fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
					t.Errorf("for key '%s', expected value '%v', got '%v'", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestLangChainClient_GetLabelSynonyms_Batching(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	// Helper to create expected prompt string
	makePrompt := func(data interface{}) string {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return fmt.Sprintf(prompts.LabelSynonymPrompt, string(jsonData))
	}

	batch1 := []string{"label1", "label2"}
	batch2 := []string{"label3"}
	prompt1 := makePrompt(batch1)
	prompt2 := makePrompt(batch2)

	tests := []struct {
		name           string
		labelBatches   [][]string
		mockResponses  map[string]string // map prompt to response
		mockErrors     map[string]error  // map prompt to error
		expectedMap    map[string][]string
		expectedError  string
		expectedCalls  int
		expectedPrompts []string
	}{
		{
			name:         "successful response with multiple batches",
			labelBatches: [][]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"label1": ["syn_a"], "label2": ["syn_b"]}`,
				prompt2: `{"label3": ["syn_c", "syn_d"]}`,
			},
			expectedMap: map[string][]string{
				"label1": {"syn_a"},
				"label2": {"syn_b"},
				"label3": {"syn_c", "syn_d"},
			},
			expectedCalls: 2,
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:         "successful response with single batch",
			labelBatches: [][]string{batch1},
			mockResponses: map[string]string{
				prompt1: `{"label1": ["syn_a"], "label2": ["syn_b"]}`,
			},
			expectedMap: map[string][]string{
				"label1": {"syn_a"},
				"label2": {"syn_b"},
			},
			expectedCalls: 1,
			expectedPrompts: []string{prompt1},
		},
		{
			name:         "llm returns error for one batch",
			labelBatches: [][]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"label1": ["syn_a"]}`,
			},
			mockErrors: map[string]error{
				prompt2: errors.New("llm simulated error for batch2 labels"),
			},
			expectedError: "LangChain LLM call failed: llm simulated error for batch2 labels",
			expectedCalls: 2, // Both calls should still be attempted
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:         "malformed json response for one batch",
			labelBatches: [][]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"label1": ["syn_a"]`, // Malformed
				prompt2: `{"label2": ["syn_b"]}`,
			},
			expectedError: "error unmarshalling LLM response",
			expectedCalls: 2,
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:         "empty label batches",
			labelBatches: [][]string{},
			expectedMap:  map[string][]string{},
			expectedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ResetCallTracking()
			mock.CallResponses = tt.mockResponses
			mock.CallErrors = tt.mockErrors

			resultMap, err := client.GetLabelSynonyms(tt.labelBatches)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(resultMap, tt.expectedMap) {
					t.Errorf("expected map %v, got %v", tt.expectedMap, resultMap)
				}
			}

			mock.mu.Lock()
			if len(mock.CallInputs) != tt.expectedCalls {
				t.Errorf("expected %d LLM calls, got %d. Inputs: %v", tt.expectedCalls, len(mock.CallInputs), mock.CallInputs)
			}
			// Check if all expected prompts were called, order might vary due to goroutines
			if tt.expectedPrompts != nil {
				calledPrompts := make(map[string]bool)
				for _, p := range mock.CallInputs {
					calledPrompts[p] = true
				}
				for _, ep := range tt.expectedPrompts {
					if !calledPrompts[ep] {
						t.Errorf("expected prompt was not called: %s", ep)
					}
				}
			}
			mock.mu.Unlock()
		})
	}
}

func TestLangChainClient_GetPromQLFromLLM(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	sampleQuery := "show cpu usage"
	sampleMetrics := llm.RelevantMetricsMap{
		"cpu_usage_total": {
			"instance": llm.LabelContextDetail{MatchScore: 0.8, Values: []string{"host1", "host2"}},
			"mode":     llm.LabelContextDetail{MatchScore: 0.9, Values: []string{"idle", "user"}},
		},
	}
	sampleLabels := llm.RelevantLabelsMap{
		"region": llm.LabelContextDetail{MatchScore: 0.7, Values: []string{"us-west", "us-east"}},
	}
	sampleHistory := map[string]interface{}{
		"cpu_usage_total": map[string]interface{}{"score": 3, "labels": map[string]string{"mode": "idle"}},
	}

	tests := []struct {
		name              string
		userQuery         string
		relevantMetrics   llm.RelevantMetricsMap
		relevantLabels    llm.RelevantLabelsMap
		relevantHistory   map[string]interface{}
		mockResponse      *llms.ContentResponse
		mockError         error
		expectedPromQLs   []string
		expectedError     string
		checkPrompt       bool // Flag to enable prompt checking for specific test cases
	}{
		{
			name:            "successful response",
			userQuery:       sampleQuery,
			relevantMetrics: sampleMetrics,
			relevantLabels:  sampleLabels,
			relevantHistory: sampleHistory,
			mockResponse: &llms.ContentResponse{Choices: []*llms.ContentChoice{
				{Content: `[{"promql": "query1", "score": 1.0}, {"promql": "query2", "score": 0.5}]`},
			}},
			mockError:       nil,
			expectedPromQLs: []string{"query1", "query2"},
			checkPrompt:     true,
		},
		{
			name:            "llm returns error",
			userQuery:       sampleQuery,
			relevantMetrics: sampleMetrics,
			relevantLabels:  sampleLabels,
			relevantHistory: sampleHistory,
			mockError:       errors.New("llm simulated error for promql"),
			expectedError:   "LangChain LLM GenerateContent call failed: llm simulated error for promql",
		},
		{
			name:            "malformed json response",
			userQuery:       sampleQuery,
			relevantMetrics: sampleMetrics,
			relevantLabels:  sampleLabels,
			relevantHistory: sampleHistory,
			mockResponse: &llms.ContentResponse{Choices: []*llms.ContentChoice{
				{Content: `[{"promql": "query1", "score": 1.0},`}, // Malformed
			}},
			mockError:     nil,
			expectedError: "error unmarshalling LLM response for PromQL",
		},
		{
			name:            "empty choices in response",
			userQuery:       sampleQuery,
			relevantMetrics: sampleMetrics,
			relevantLabels:  sampleLabels,
			relevantHistory: sampleHistory,
			mockResponse:    &llms.ContentResponse{Choices: []*llms.ContentChoice{}},
			mockError:       nil,
			expectedError:   "LLM returned no choices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedMessages []llms.MessageContent
			mock.GenerateContentFunc = func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
				capturedMessages = messages // Capture messages
				return tt.mockResponse, tt.mockError
			}

			resultPromQLs, err := client.GetPromQLFromLLM(tt.userQuery, tt.relevantMetrics, tt.relevantLabels, tt.relevantHistory)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.checkPrompt {
				if len(capturedMessages) != 2 {
					t.Fatalf("expected 2 messages (system, user), got %d", len(capturedMessages))
				}

				// Check system prompt
				if len(capturedMessages[0].Parts) != 1 {
					t.Fatalf("expected 1 part in system message, got %d", len(capturedMessages[0].Parts))
				}
				sysTextPart, okSys := capturedMessages[0].Parts[0].(llms.TextContent)
				if !okSys {
					t.Fatalf("system message part is not TextContent")
				}
				if sysTextPart.Text != prompts.SystemPrompt {
					t.Errorf("system prompt mismatch. Expected:\n%s\nGot:\n%s", prompts.SystemPrompt, sysTextPart.Text)
				}

				// Check user prompt for key elements
				if len(capturedMessages[1].Parts) != 1 {
					t.Fatalf("expected 1 part in user message, got %d", len(capturedMessages[1].Parts))
				}
				userTextPart, okUser := capturedMessages[1].Parts[0].(llms.TextContent)
				if !okUser {
					t.Fatalf("user message part is not TextContent")
				}
				userPromptContent := userTextPart.Text
				if !strings.Contains(userPromptContent, tt.userQuery) {
					t.Errorf("user prompt does not contain userQuery. Got: %s", userPromptContent)
				}
				metricsJSON, _ := json.MarshalIndent(tt.relevantMetrics, "", "  ")
				if !strings.Contains(userPromptContent, string(metricsJSON)) {
					t.Errorf("user prompt does not contain relevantMetrics JSON. Expected to contain:\n%s\nGot:\n%s", string(metricsJSON), userPromptContent)
				}
				labelsJSON, _ := json.MarshalIndent(tt.relevantLabels, "", "  ")
				if !strings.Contains(userPromptContent, string(labelsJSON)) {
					t.Errorf("user prompt does not contain relevantLabels JSON. Expected to contain:\n%s\nGot:\n%s", string(labelsJSON), userPromptContent)
				}
				historyJSON, _ := json.MarshalIndent(tt.relevantHistory, "", "  ")
				if !strings.Contains(userPromptContent, string(historyJSON)) {
					t.Errorf("user prompt does not contain relevantHistory JSON. Expected to contain:\n%s\nGot:\n%s", string(historyJSON), userPromptContent)
				}
			}

			if len(resultPromQLs) != len(tt.expectedPromQLs) {
				t.Errorf("expected %d PromQL queries, got %d. Result: %v", len(tt.expectedPromQLs), len(resultPromQLs), resultPromQLs)
			}
			for i, expectedQL := range tt.expectedPromQLs {
				if resultPromQLs[i] != expectedQL {
					t.Errorf("expected PromQL query '%s' at index %d, got '%s'", expectedQL, i, resultPromQLs[i])
				}
			}
		})
	}
}


func TestLangChainClient_GetMetricSynonyms_Batching(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	// Helper to create expected prompt string
	makePrompt := func(data interface{}) string {
		jsonData, _ := json.MarshalIndent(data, "", "  ")
		return fmt.Sprintf(prompts.MetricSynonymPrompt, string(jsonData))
	}

	batch1 := map[string]string{"metric1": "desc1", "metric2": "desc2"}
	batch2 := map[string]string{"metric3": "desc3"}
	prompt1 := makePrompt(batch1)
	prompt2 := makePrompt(batch2)

	tests := []struct {
		name            string
		metricBatches   []map[string]string
		mockResponses   map[string]string // map prompt to response
		mockErrors      map[string]error  // map prompt to error
		expectedMap     map[string][]string
		expectedError   string
		expectedCalls   int
		expectedPrompts  []string
	}{
		{
			name:          "successful response with multiple batches",
			metricBatches: []map[string]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"metric1": ["syn1_a"], "metric2": ["syn2_a"]}`,
				prompt2: `{"metric3": ["syn3_a", "syn3_b"]}`,
			},
			expectedMap: map[string][]string{
				"metric1": {"syn1_a"},
				"metric2": {"syn2_a"},
				"metric3": {"syn3_a", "syn3_b"},
			},
			expectedCalls: 2,
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:          "successful response with single batch",
			metricBatches: []map[string]string{batch1},
			mockResponses: map[string]string{
				prompt1: `{"metric1": ["syn1_a"], "metric2": ["syn2_a"]}`,
			},
			expectedMap: map[string][]string{
				"metric1": {"syn1_a"},
				"metric2": {"syn2_a"},
			},
			expectedCalls: 1,
			expectedPrompts: []string{prompt1},
		},
		{
			name:          "llm returns error for one batch",
			metricBatches: []map[string]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"metric1": ["syn1_a"]}`,
			},
			mockErrors: map[string]error{
				prompt2: errors.New("llm simulated error for batch2 metrics"),
			},
			expectedError: "LangChain LLM call failed: llm simulated error for batch2 metrics",
			expectedCalls: 2,
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:          "malformed json response for one batch",
			metricBatches: []map[string]string{batch1, batch2},
			mockResponses: map[string]string{
				prompt1: `{"metric1": ["syn1_a"]`, // Malformed
				prompt2: `{"metric2": ["syn2_a"]}`,
			},
			expectedError: "error unmarshalling LLM response",
			expectedCalls: 2,
			expectedPrompts: []string{prompt1, prompt2},
		},
		{
			name:          "empty metric batches",
			metricBatches: []map[string]string{},
			expectedMap:   map[string][]string{},
			expectedCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ResetCallTracking()
			mock.CallResponses = tt.mockResponses
			mock.CallErrors = tt.mockErrors

			resultMap, err := client.GetMetricSynonyms(tt.metricBatches)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error containing '%s', got '%v'", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(resultMap, tt.expectedMap) {
					t.Errorf("expected map %v, got %v", tt.expectedMap, resultMap)
				}
			}

			mock.mu.Lock()
			if len(mock.CallInputs) != tt.expectedCalls {
				t.Errorf("expected %d LLM calls, got %d. Inputs: %v", tt.expectedCalls, len(mock.CallInputs), mock.CallInputs)
			}
			// Check if all expected prompts were called, order might vary due to goroutines
			if tt.expectedPrompts != nil {
				calledPrompts := make(map[string]bool)
				for _, p := range mock.CallInputs {
					calledPrompts[p] = true
				}
				for _, ep := range tt.expectedPrompts {
					if !calledPrompts[ep] {
						t.Errorf("expected prompt was not called: %s", ep)
					}
				}
			}
			mock.mu.Unlock()
		})
	}
}
