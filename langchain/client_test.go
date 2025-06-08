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

// mockLLM is a mock implementation of the llms.Model interface for testing.
type mockLLM struct {
	CallFunc            func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error)
	GenerateContentFunc func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error)
	// Add other methods if they are called by the client, otherwise, they can be minimal implementations.
}

// Call implements the llms.Model interface.
func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.CallFunc != nil {
		return m.CallFunc(ctx, prompt, options...)
	}
	return "", errors.New("CallFunc not implemented in mockLLM")
}

// GenerateContent implements the llms.Model interface.
func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, messages, options...)
	}
	return nil, errors.New("GenerateContentFunc not implemented in mockLLM")
}

// GetNumTokens implements the llms.Model interface - minimal implementation.
// This might be part of an llms.LanguageModel interface or similar, depending on LangchainGo version.
// For basic Model interface, it might not be strictly required if not used by the client.
// Let's assume it's part of a broader interface that might be checked.
func (m *mockLLM) GetNumTokens(text string) int {
	return len(text) // Dummy implementation
}

// Getआईdentifiers implements the llms.Model interface - minimal implementation
func (m *mockLLM) GetIdentifiers() []string {
	return []string{"mockLLM"}
}

// GetType implements the llms.Model interface - minimal implementation
// This is often part of schema.LLM or similar.
// Assuming llms.Model is a simpler interface for now.
// If the actual llms.Model includes this, we need it.
// Based on tmc/langchaingo/llms/llms.go, Model does not have GetType.
// However, specific model implementations might, or it might be part of a different interface.
// For now, we'll omit it unless a compile error shows it's needed for llms.Model.

// Ensure mockLLM implements llms.Model.
// This line might cause a compile error if llms.Model is more complex than just Call and GenerateContent.
// We will adjust based on that.
var _ llms.Model = (*mockLLM)(nil)

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

func TestLangChainClient_GetLabelSynonyms(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	tests := []struct {
		name          string
		labelNames    []string
		mockResponse  string
		mockError     error
		expectedMap   map[string][]string
		expectedError string
	}{
		{
			name:         "successful response",
			labelNames:   []string{"label1", "label2"},
			mockResponse: `{"label1": ["syn_a", "syn_b"], "label2": ["syn_c"]}`,
			mockError:    nil,
			expectedMap:  map[string][]string{"label1": {"syn_a", "syn_b"}, "label2": {"syn_c"}},
		},
		{
			name:          "llm returns error",
			labelNames:    []string{"label1"},
			mockError:     errors.New("llm simulated error for labels"),
			expectedError: "LangChain LLM call failed: llm simulated error for labels",
		},
		{
			name:          "malformed json response",
			labelNames:    []string{"label1"},
			mockResponse:  `{"label1": ["syn_a", "syn_b"]`, // Missing closing brace
			mockError:     nil,
			expectedError: "error unmarshalling LLM response",
		},
		{
			name:          "empty label names",
			labelNames:    []string{},
			mockResponse:  `{}`,
			mockError:     nil,
			expectedMap:   map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.CallFunc = func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
				// TODO: Could add prompt validation here if needed, e.g., check if labelNamesJSON is in prompt
				return tt.mockResponse, tt.mockError
			}

			resultMap, err := client.GetLabelSynonyms(tt.labelNames)

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

			if len(resultMap) != len(tt.expectedMap) {
				t.Errorf("expected map length %d, got %d. Result: %v", len(tt.expectedMap), len(resultMap), resultMap)
			}
			for key, expectedValues := range tt.expectedMap {
				actualValues, ok := resultMap[key]
				if !ok {
					t.Errorf("expected key '%s' not found in result map", key)
					continue
				}
				if len(actualValues) != len(expectedValues) {
					t.Errorf("for key '%s', expected %d values, got %d. Actual: %v", key, len(expectedValues), len(actualValues), actualValues)
					continue
				}
				for i, v := range expectedValues {
					if actualValues[i] != v {
						t.Errorf("for key '%s' at index %d, expected value '%s', got '%s'", key, i, v, actualValues[i])
					}
				}
			}
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


func TestLangChainClient_GetMetricSynonyms(t *testing.T) {
	mock := &mockLLM{}
	client := langchain.NewLangChainClient(mock)

	tests := []struct {
		name          string
		metricMap     map[string]string
		mockResponse  string
		mockError     error
		expectedMap   map[string][]string
		expectedError string
	}{
		{
			name:         "successful response",
			metricMap:    map[string]string{"metric1": "desc1", "metric2": "desc2"},
			mockResponse: `{"metric1": ["syn1_1", "syn1_2"], "metric2": ["syn2_1"]}`,
			mockError:    nil,
			expectedMap:  map[string][]string{"metric1": {"syn1_1", "syn1_2"}, "metric2": {"syn2_1"}},
		},
		{
			name:          "llm returns error",
			metricMap:     map[string]string{"metric1": "desc1"},
			mockError:     errors.New("llm simulated error"),
			expectedError: "LangChain LLM call failed: llm simulated error",
		},
		{
			name:          "malformed json response",
			metricMap:     map[string]string{"metric1": "desc1"},
			mockResponse:  `{"metric1": ["syn1_1", "syn1_2"]`, // Missing closing brace
			mockError:     nil,
			expectedError: "error unmarshalling LLM response", // Error message should contain this
		},
		{
			name:          "empty metric map",
			metricMap:     map[string]string{},
			mockResponse:  `{}`,
			mockError:     nil,
			expectedMap:   map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.CallFunc = func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
				return tt.mockResponse, tt.mockError
			}

			resultMap, err := client.GetMetricSynonyms(tt.metricMap)

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

			if len(resultMap) != len(tt.expectedMap) {
				t.Errorf("expected map length %d, got %d. Result: %v", len(tt.expectedMap), len(resultMap), resultMap)
			}
			for key, expectedValues := range tt.expectedMap {
				actualValues, ok := resultMap[key]
				if !ok {
					t.Errorf("expected key '%s' not found in result map", key)
					continue
				}
				if len(actualValues) != len(expectedValues) {
					t.Errorf("for key '%s', expected %d values, got %d. Actual: %v", key, len(expectedValues), len(actualValues), actualValues)
					continue
				}
				for i, v := range expectedValues {
					if actualValues[i] != v {
						t.Errorf("for key '%s' at index %d, expected value '%s', got '%s'", key, i, v, actualValues[i])
					}
				}
			}
		})
	}
}
