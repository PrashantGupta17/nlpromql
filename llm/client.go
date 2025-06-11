package llm

// LabelContextDetail holds match score and example values for a label.
type LabelContextDetail struct {
	MatchScore float64  `json:"match_score"`
	Values     []string `json:"values"`
}

// RelevantMetricsMap is a map of relevant metric names to a nested map of
// label names to their LabelContextDetail.
// Example: {"metric1": {"labelA": {"match_score": 0.8, "values": ["val1", "val2"]}}}
type RelevantMetricsMap map[string]map[string]LabelContextDetail

// RelevantLabelsMap is a map of relevant label names to their LabelContextDetail.
// Example: {"labelA": {"match_score": 0.9, "values": ["val1", "val2", "val3"]}}
type RelevantLabelsMap map[string]LabelContextDetail

// LLMClient defines the interface for interacting with an LLM.
// The GetPromQLFromLLM method will now use the new map types.
type LLMClient interface {
	GetMetricSynonyms(metricBatches []map[string]string) (map[string][]string, error)
	GetLabelSynonyms(labelBatches [][]string) (map[string][]string, error)
	ProcessUserQuery(userQuery string) (map[string]interface{}, error)
	GetPromQLFromLLM(userQuery string, relevantMetrics RelevantMetricsMap, relevantLabels RelevantLabelsMap, relevantHistory map[string]interface{}) ([]string, error)
}
