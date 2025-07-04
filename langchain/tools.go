package langchain

import (
	"encoding/json"

	"github.com/tmc/langchaingo/tools"
	// Assuming a structure like this for schema definition.
	// This might need adjustment based on actual langchaingo capabilities for defining tool schemas.
	// We'll use a generic map[string]interface{} for the schema if specific struct-to-schema is not straightforward,
	// or define structs that can be marshaled into a JSON schema format if the library supports that.
)

// Define structs for the expected output of each tool, matching the JSON structure.

// MetricSynonymsToolOutput is the expected output structure for the metric synonyms tool.
type MetricSynonymsToolOutput struct {
	Synonyms map[string][]string `json:"synonyms"` // e.g., {"metric1": ["syn1", "syn2"]}
}

// LabelSynonymsToolOutput is the expected output structure for the label synonyms tool.
type LabelSynonymsToolOutput struct {
	Synonyms map[string][]string `json:"synonyms"` // e.g., {"label1": ["syn1", "syn2"]}
}

// ProcessQueryToolOutput is the expected output structure for the process query tool.
type ProcessQueryToolOutput struct {
	PossibleMetricNames []string `json:"possible_metric_names"`
	PossibleLabelNames  []string `json:"possible_label_names"`
	PossibleLabelValues []string `json:"possible_label_values"`
}

// PromQLQuery represents a single PromQL query with its metadata.
type PromQLQuery struct {
	PromQL            string            `json:"promql"`
	Score             float64           `json:"score"`
	MetricLabelPairs map[string]map[string]string `json:"metric_label_pairs"`
}

// GeneratePromQLToolOutput is the expected output structure for the PromQL generation tool.
type GeneratePromQLToolOutput struct {
	Queries []PromQLQuery `json:"queries"`
}

// newToolDefinition creates a generic tool definition.
// langchaingo's actual tool definition might require a more structured schema (e.g., JSON schema).
// For simplicity, we'll assume parameters can be described by a struct that gets marshalled to JSON.
// The LLM is expected to return parameters matching this structure.

// GetMetricSynonymsTool defines the tool for getting metric synonyms.
func GetMetricSynonymsTool() tools.Tool {
	// The schema here should represent the *input* to the tool if the tool were a callable function.
	// However, in this case, we are telling the LLM to *produce* output matching a schema.
	// The `Parameters` field in `ToolDefinition` is often used by LLMs to know what arguments a tool expects.
	// For "output shaping", the schema describes the desired JSON structure.
	// We'll define a schema that expects the LLM to return the synonyms map.
	schema := `{
		"type": "object",
		"properties": {
			"synonyms": {
				"type": "object",
				"additionalProperties": {
					"type": "array",
					"items": {
						"type": "string"
					}
				},
				"description": "A map where keys are original metric names and values are arrays of their synonyms."
			}
		},
		"required": ["synonyms"]
	}`
	var schemaMap map[string]interface{}
	_ = json.Unmarshal([]byte(schema), &schemaMap) // Error handling omitted for brevity in subtask

	return &tools.FunctionDefinition{
		Name:        "GetMetricSynonyms",
		Description: "Generates synonyms for given Prometheus metric names. The output should be a JSON object mapping original metric names to an array of their synonyms.",
		Parameters:  schemaMap,
		// Function: func(input map[string]any) (map[string]any, error) { ... } // Not needed if LLM directly outputs JSON
	}
}

// GetLabelSynonymsTool defines the tool for getting label synonyms.
func GetLabelSynonymsTool() tools.Tool {
	schema := `{
		"type": "object",
		"properties": {
			"synonyms": {
				"type": "object",
				"additionalProperties": {
					"type": "array",
					"items": {
						"type": "string"
					}
				},
				"description": "A map where keys are original label names and values are arrays of their synonyms."
			}
		},
		"required": ["synonyms"]
	}`
	var schemaMap map[string]interface{}
	_ = json.Unmarshal([]byte(schema), &schemaMap)

	return &tools.FunctionDefinition{
		Name:        "GetLabelSynonyms",
		Description: "Generates synonyms for given Prometheus label names. The output should be a JSON object mapping original label names to an array of their synonyms.",
		Parameters:  schemaMap,
	}
}

// ProcessUserQueryTool defines the tool for processing a user query.
func ProcessUserQueryTool() tools.Tool {
	schema := `{
		"type": "object",
		"properties": {
			"possible_metric_names": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Array of potential metric names relevant to the user query."
			},
			"possible_label_names": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Array of potential label names relevant to the user query."
			},
			"possible_label_values": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Array of potential label values relevant to the user query."
			}
		},
		"required": ["possible_metric_names", "possible_label_names", "possible_label_values"]
	}`
	var schemaMap map[string]interface{}
	_ = json.Unmarshal([]byte(schema), &schemaMap)

	return &tools.FunctionDefinition{
		Name:        "ProcessUserQuery",
		Description: "Analyzes a user query and identifies possible Prometheus metric names, label names, and label values. The output should be a JSON object with these three fields, each an array of strings.",
		Parameters:  schemaMap,
	}
}

// GeneratePromQLTool defines the tool for generating PromQL queries.
func GeneratePromQLTool() tools.Tool {
	schema := `{
		"type": "object",
		"properties": {
			"queries": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"promql": {"type": "string", "description": "The generated PromQL query."},
						"score": {"type": "number", "description": "Relevance score for the query."},
						"metric_label_pairs": {
							"type": "object",
							"description": "Metric names used in the query and their corresponding label-value pairs.",
							"additionalProperties": {
								"type": "object",
								"additionalProperties": {"type": "string"}
							}
						}
					},
					"required": ["promql", "score", "metric_label_pairs"]
				},
				"description": "An array of potential PromQL queries."
			}
		},
		"required": ["queries"]
	}`
	var schemaMap map[string]interface{}
	_ = json.Unmarshal([]byte(schema), &schemaMap)

	return &tools.FunctionDefinition{
		Name:        "GeneratePromQLQueries",
		Description: "Generates a list of PromQL queries based on user input and context. The output should be a JSON object containing an array of query objects, each with 'promql', 'score', and 'metric_label_pairs'.",
		Parameters:  schemaMap,
	}
}

// Note: The actual implementation of tool calling in langchaingo might involve
// specifying these tools in the llms.CallOption or equivalent.
// The structs (MetricSynonymsToolOutput etc.) are useful for unmarshalling
// the structured JSON that the LLM returns as the "arguments" to the tool call.
